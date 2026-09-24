package plan_test

import (
	"strings"
	"testing"

	"github.com/Grizak/cppmake/internal/parser"
	"github.com/Grizak/cppmake/internal/plan"
)

func TestResolveRejectsInvalidConfig(t *testing.T) {
	_, err := plan.Resolve(&parser.Config{Targets: []parser.Target{{Name: "app"}}})
	if err == nil || !strings.Contains(err.Error(), "src must not be empty") {
		t.Fatalf("Resolve() error = %v, want validation error", err)
	}
}

func TestResolveAppliesDefaultsBeforeValidation(t *testing.T) {
	config := &parser.Config{
		Toolchains: map[string]map[string]parser.Toolchain{"c": {"gcc": {}}},
		Linkers:    map[string]parser.Linker{"gcc": {}},
		Targets:    []parser.Target{{Name: "app", Src: []string{"main.c"}}},
	}

	if _, err := plan.Resolve(config); err != nil {
		t.Fatalf("Resolve() error = %v, want nil for defaultable target", err)
	}
}
