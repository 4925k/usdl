package chatapp

import (
	"net/http"

	"github.com/4925k/usdl/chat/foundation/web"
)

func Routes(app *web.App) {
	api := newApp()

	app.HandlerFunc(http.MethodGet, "", "/test", api.test)
	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)

}
