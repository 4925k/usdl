package mux

import (
	"context"
	"net/http"

	"github.com/4925k/usdl/chat/app/domain/chatapp"
	"github.com/4925k/usdl/chat/app/sdk/mid"
	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/4925k/usdl/chat/foundation/web"
)

type Config struct {
	Log *logger.Logger
}

func WebAPI(cfg Config) http.Handler {
	logger := func(ctx context.Context, msg string, args ...any) {
		cfg.Log.Info(ctx, msg, args...)
	}

	app := web.NewApp(
		logger,
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Panics(),
	)

	chatapp.Routes(app)

	return app
}
