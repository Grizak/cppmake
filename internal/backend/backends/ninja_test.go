package backends_test

import (
	"strings"
	"testing"

	"github.com/Grizak/cppmake/internal/backend/backends"
	"github.com/Grizak/cppmake/internal/parser"
)

func TestNinjaBackendSortsToolchainAndLinkerRules(t *testing.T) {
	cfg := parser.Config{
		Toolchains: map[string]map[string]parser.Toolchain{
			"cpp": {
				"clang": {},
				"gcc":   {},
			},
			"c": {
				"gcc": {},
			},
		},
		Linkers: map[string]parser.Linker{
			"lld": {},
			"gcc": {},
		},
	}

	output := string((&backends.NinjaBackend{}).Generate(cfg))
	orderedRules := []string{
		"rule c_gcc\n",
		"rule cpp_clang\n",
		"rule cpp_gcc\n",
		"rule link_gcc\n",
		"rule link_lld\n",
	}
	last := -1
	for _, rule := range orderedRules {
		index := strings.Index(output, rule)
		if index <= last {
			t.Fatalf("expected %q after previous rule, output was:\n%s", rule, output)
		}
		last = index
	}
}
