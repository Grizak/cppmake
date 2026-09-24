package backends_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Grizak/cppmake/internal/backend/backends"
	"github.com/Grizak/cppmake/internal/parser"
	"github.com/Grizak/cppmake/internal/plan"
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
		Targets: []parser.Target{
			{Name: "a", Type: "binary", Lang: "c", Toolchain: "gcc", Linker: "gcc", Src: []string{"a.c"}, Outfile: "a"},
			{Name: "b", Type: "binary", Lang: "cpp", Toolchain: "clang", Linker: "lld", Src: []string{"b.cpp"}, Outfile: "b"},
			{Name: "c", Type: "binary", Lang: "cpp", Toolchain: "gcc", Linker: "gcc", Src: []string{"c.cpp"}, Outfile: "c"},
		},
	}

	resolved, err := plan.Resolve(&cfg)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	outputBytes, err := (&backends.NinjaBackend{}).Emit(resolved)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	output := string(outputBytes)

	var filteredLines []string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	output = strings.Join(filteredLines, "\n")

	orderedRules := []string{
		"rule c_gcc\n",
		"rule cpp_clang\n",
		"rule cpp_gcc\n",
		"rule link_gcc\n",
		"rule link_lld\n",
		"default",
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
	var errors map[string][]byte = make(map[string][]byte)

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
			buildPlan, err := plan.Resolve(config)
			if err != nil {
				t.Fatalf("resolve %s: %v", configFile, err)
			}

			goldenFile := filepath.Join(filepath.Dir(configFile), "build.ninja.golden")
			golden, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("read %s: %v", goldenFile, err)
			}

			actual, err := (&backends.NinjaBackend{}).Emit(buildPlan)
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			if !bytes.Equal(actual, golden) {
				t.Errorf("generated output differs from %s\n--- got ---\n%s\n--- want ---\n%s", goldenFile, actual, golden)
				errors[configFile] = actual
			}
		})
	}

	// Newline
	fmt.Println()
	if len(errors) > 0 {
		// Run "diff" command on the errors to show the differences
		for configFile, actual := range errors {
			goldenFile := filepath.Join(filepath.Dir(configFile), "build.ninja.golden")
			cmd := exec.Command("diff", "-u", "-", goldenFile)
			cmd.Stdin = bytes.NewReader(actual)

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err := cmd.Run()
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 1 {
					t.Errorf("golden file mismatch for %s:\n%s", configFile, out.String())
				} else {
					t.Fatalf("diff exited with code %d for %s: %v\n%s", exitErr.ExitCode(), configFile, err, out.String())
				}
			} else if err != nil {
				t.Fatalf("failed to run diff for %s: %v", configFile, err)
			}
		}
	}
}
