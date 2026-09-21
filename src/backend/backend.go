package backend

import (
	"cppmake/src/backend/backends"
	"cppmake/src/parser"
	"fmt"
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
	return &backends.NinjaBackend{}, fmt.Errorf("Backend not found")
}
