package auth

import "context"

type Auth struct{}

type Claims struct {
	Subject string
}

func (a *Auth) Authenticate(ctx context.Context, authHeader string) (*Claims, error) {
	return &Claims{}, nil
}
