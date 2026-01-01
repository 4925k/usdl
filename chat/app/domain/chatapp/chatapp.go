package chatapp

import (
	"context"
	"net/http"
	"time"

	"github.com/4925k/usdl/chat/app/sdk/errs"
	"github.com/4925k/usdl/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	WS websocket.Upgrader
}

func newApp() *app {
	return &app{}
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

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case msg, wd := <-ch:
			if !wd {
				return web.NewNoResponse()
			}

			if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return err
			}
		case <-ticker.C:
			err := c.WriteMessage(websocket.PingMessage, []byte("ping"))
			if err != nil {
				return nil
			}
		}
	}

}
