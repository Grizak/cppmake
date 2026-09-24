package backend

import (
	"fmt"

	"github.com/Grizak/cppmake/internal/backend/backends"
	"github.com/Grizak/cppmake/internal/plan"
)

type Backend interface {
	Emit(p *plan.Plan) ([]byte, error)
	Filename() string
}

func BackendFactory(name string) (Backend, error) {
	switch name {
	case "ninja":
		return &backends.NinjaBackend{}, nil
	}
	return nil, fmt.Errorf("Backend not found: %s", name)
}
