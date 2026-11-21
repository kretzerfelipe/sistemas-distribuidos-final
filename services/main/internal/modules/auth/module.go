package auth

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"futhub/main/internal/app"
	"futhub/main/internal/platform/routing"
	"futhub/main/internal/platform/security"
)

const basePath = "/auth"

type Module struct {
	uc     UseCase
	ctrl   *controller
	tokens *security.TokenManager
}

func New(d app.Deps, userService UserService) (*Module, error) {
	uc, err := newUseCase(userService, d.Hasher, d.Tokens)
	if err != nil {
		return nil, fmt.Errorf("create auth use case: %w", err)
	}
	return &Module{
		uc:     uc,
		ctrl:   newController(uc),
		tokens: d.Tokens,
	}, nil
}

func (m *Module) Name() string { return "auth" }

func (m *Module) RegisterRoutes(r chi.Router) error {
	return routing.Register(r, basePath, m.tokens, m.ctrl.routes())
}

func (m *Module) SeedAdmin(ctx context.Context, name, email, password string) (bool, error) {
	return m.uc.SeedAdmin(ctx, name, email, password)
}
