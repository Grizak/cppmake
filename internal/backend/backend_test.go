package backend_test

import (
	"testing"

	"github.com/Grizak/cppmake/internal/backend"
	"github.com/Grizak/cppmake/internal/backend/backends"
)

func TestBackendFactoryNinja(t *testing.T) {
	result, err := backend.BackendFactory("ninja")
	if err != nil {
		t.Fatalf("BackendFactory() error = %v", err)
	}
	if _, ok := result.(*backends.NinjaBackend); !ok {
		t.Fatalf("BackendFactory() type = %T, want *backends.NinjaBackend", result)
	}
}

func TestBackendFactoryUnknown(t *testing.T) {
	result, err := backend.BackendFactory("unknown")
	if err == nil {
		t.Fatal("BackendFactory() error = nil, want an error")
	}
	if result != nil {
		t.Fatalf("BackendFactory() result = %T, want nil", result)
	}
}
