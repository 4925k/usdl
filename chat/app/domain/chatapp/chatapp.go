package chatapp

import (
	"context"
	"net/http"

	"github.com/4925k/usdl/chat/app/sdk/chat"
	"github.com/4925k/usdl/chat/app/sdk/errs"
	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/4925k/usdl/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	log  *logger.Logger
	WS   websocket.Upgrader
	chat *chat.Chat
}

func newApp(log *logger.Logger) *app {
	return &app{
		log:  log,
		chat: chat.NewChat(log),
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

	err = a.chat.Handshake(ctx, c)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "handshake failed: %s", err)
	}

	a.chat.Listen(ctx, c)

	return web.NewNoResponse()
}
