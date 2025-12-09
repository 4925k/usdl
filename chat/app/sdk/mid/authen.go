package mid

import (
	"context"
	"net/http"

	"github.com/4925k/usdl/chat/app/sdk/auth"
	"github.com/4925k/usdl/chat/app/sdk/errs"
	"github.com/4925k/usdl/chat/foundation/web"
)

// Bearer processes JWT authentication logic.
func Bearer(ath *auth.Auth) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {
			claims, err := ath.Authenticate(ctx, r.Header.Get("authorization"))
			if err != nil {
				return errs.New(errs.Unauthenticated, err)
			}

			if claims.Subject == "" {
				return errs.Newf(errs.Unauthenticated, "authorize: you are not authorized for that action, no claims")
			}

			ctx = setUserID(ctx, claims.Subject)
			ctx = setClaims(ctx, *claims)

			return next(ctx, r)
		}

		return h
	}

	return m
}
