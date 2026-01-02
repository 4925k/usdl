package chatapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/4925k/usdl/chat/app/sdk/errs"
	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/4925k/usdl/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	log *logger.Logger
	WS  websocket.Upgrader
}

func newApp(log *logger.Logger) *app {
	return &app{
		log: log,
	}
}

func (a *app) test(_ context.Context, _ *http.Request) web.Encoder {
	return status{
		Status: "ok",
	}
}

func (a *app) connect(ctx context.Context, r *http.Request) web.Encoder {
	c, err := a.WS.Upgrade(web.GetWriter(ctx), r, nil)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket: %s", err)
	}

	defer c.Close()

	usr, err := a.handshake(c)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "handshake failed: %s", err)
	}

	a.log.Info(ctx, "user connected: %s", usr.Name)

	return web.NewNoResponse()
}

func (a *app) handshake(c *websocket.Conn) (user, error) {
	if err := c.WriteMessage(websocket.TextMessage, []byte("HELLO")); err != nil {
		return user{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msg, err := a.readMessage(ctx, c)
	if err != nil {
		return user{}, fmt.Errorf("read message: %w", err)
	}

	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return user{}, fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.WriteMessage(websocket.TextMessage, []byte("WELCOME "+usr.Name)); err != nil {
		return user{}, err
	}

	return usr, nil
}

func (a *app) readMessage(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)

	go func() {
		a.log.Info(ctx, "starting handshake read")
		defer a.log.Info(ctx, "finished handshake read")

		_, msg, err := c.ReadMessage()
		if err != nil {
			ch <- response{nil, err}
		}

		ch <- response{msg, nil}
	}()

	var resp response

	select {
	case <-ctx.Done():
		c.Close()
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.msg == nil {
			return nil, fmt.Errorf("empty message")
		}
	}

	return resp.msg, nil
}
