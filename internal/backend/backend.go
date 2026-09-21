package backend

import (
	"fmt"

	"github.com/Grizak/cppmake/internal/backend/backends"
	"github.com/Grizak/cppmake/internal/parser"
)

type Backend interface {
	Generate(cfg parser.Config) []byte
	Filename() string
}

func BackendFactory(name string) (Backend, error) {
	switch name {
	case "ninja":
		return &backends.NinjaBackend{}, nil
	}
	return nil, fmt.Errorf("Backend not found")
}
