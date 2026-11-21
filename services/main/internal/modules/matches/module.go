package matches

import (
	"github.com/go-chi/chi/v5"

	"futhub/main/internal/app"
	"futhub/main/internal/platform/routing"
	"futhub/main/internal/platform/security"
)

const basePath = "/"

type Module struct {
	uc     UseCase
	ctrl   *controller
	tokens *security.TokenManager
}

func New(d app.Deps, championships ChampionshipService) *Module {
	repo := newPostgresRepository(d.DB)
	uc := newUseCase(repo, championships)
	return &Module{
		uc:     uc,
		ctrl:   newController(uc),
		tokens: d.Tokens,
	}
}

func (m *Module) Name() string { return "matches" }

func (m *Module) RegisterRoutes(r chi.Router) error {
	return routing.Register(r, basePath, m.tokens, m.ctrl.routes())
}

func (m *Module) Service() UseCase { return m.uc }
