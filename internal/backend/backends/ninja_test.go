package backends_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

func TestNinjaBackendGoldenFiles(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
	configFiles, err := filepath.Glob(filepath.Join(repoRoot, "test", "*", "build.toml"))
	if err != nil {
		t.Fatalf("find build.toml files: %v", err)
	}
	sort.Strings(configFiles)
	if len(configFiles) == 0 {
		t.Fatal("no test build.toml files found")
	}

	for _, configFile := range configFiles {
		name := filepath.Base(filepath.Dir(configFile))
		t.Run(name, func(t *testing.T) {
			config, err := parser.Parse(configFile)
			if err != nil {
				t.Fatalf("parse %s: %v", configFile, err)
			}
			config.ApplyDefaults()

			goldenFile := filepath.Join(filepath.Dir(configFile), "build.ninja.golden")
			golden, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("read %s: %v", goldenFile, err)
			}

			actual := (&backends.NinjaBackend{}).Generate(*config)
			if !bytes.Equal(actual, golden) {
				t.Errorf("generated output differs from %s\n--- got ---\n%s\n--- want ---\n%s", goldenFile, actual, golden)
			}
		})
	}
}
