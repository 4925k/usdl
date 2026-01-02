package chatapp

import (
	"net/http"

	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/4925k/usdl/chat/foundation/web"
)

func Routes(app *web.App, log *logger.Logger) {
	api := newApp(log)

	app.HandlerFunc(http.MethodGet, "", "/test", api.test)
	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)

}
