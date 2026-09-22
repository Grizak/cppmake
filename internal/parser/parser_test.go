package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Grizak/cppmake/internal/parser"
)

func TestApplyDefaults(t *testing.T) {
	config := parser.Config{
		Targets: []parser.Target{
			{Name: "default-target"},
			{
				Name:      "configured-target",
				Type:      "static",
				Lang:      "cpp",
				Toolchain: "clang",
				Linker:    "lld",
				Outfile:   "custom.out",
			},
		},
	}

	config.ApplyDefaults()

	if config.Build.DefaultToolchain != "gcc" {
		t.Errorf("DefaultToolchain = %q, want %q", config.Build.DefaultToolchain, "gcc")
	}
	if config.Build.DefaultLinker != "gcc" {
		t.Errorf("DefaultLinker = %q, want %q", config.Build.DefaultLinker, "gcc")
	}
	if config.Build.BuildDir != "build" {
		t.Errorf("BuildDir = %q, want %q", config.Build.BuildDir, "build")
	}

	defaultTarget := config.Targets[0]
	if defaultTarget.Toolchain != "gcc" || defaultTarget.Linker != "gcc" || defaultTarget.Type != "binary" || defaultTarget.Lang != "c" || defaultTarget.Outfile != "default-target" {
		t.Errorf("default target was not fully populated: %+v", defaultTarget)
	}

	configuredTarget := config.Targets[1]
	if configuredTarget.Type != "static" || configuredTarget.Lang != "cpp" || configuredTarget.Toolchain != "clang" || configuredTarget.Linker != "lld" || configuredTarget.Outfile != "custom.out" {
		t.Errorf("configured target was changed: %+v", configuredTarget)
	}
}

func TestValidateCollectsErrors(t *testing.T) {
	config := parser.Config{
		Toolchains: map[string]map[string]parser.Toolchain{
			"c": {"gcc": {}},
		},
		Linkers: map[string]parser.Linker{"gcc": {}},
		Targets: []parser.Target{
			{Name: "app", Src: []string{"main.c"}, Type: "binary", Lang: "c", Toolchain: "gcc", Linker: "gcc", DependsOn: []string{"missing"}},
			{Name: "app"},
			{Name: "bad", Type: "unknown", Lang: "rust", Toolchain: "rustc", Linker: "ld", DependsOn: []string{"missing-bad"}},
			{Name: "tool", Src: []string{"tool.c"}, Type: "binary", Lang: "c", Toolchain: "clang", Linker: "gcc"},
			{Name: "first", Src: []string{"first.c"}, Type: "binary", Lang: "c", Toolchain: "gcc", Linker: "gcc", DependsOn: []string{"second"}},
			{Name: "second", Src: []string{"second.c"}, Type: "binary", Lang: "c", Toolchain: "gcc", Linker: "gcc", DependsOn: []string{"first"}},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation errors")
	}
	for _, expected := range []string{
		"duplicate target name \"app\"",
		"target \"app\": src must not be empty",
		"target \"app\": depends_on references unknown target \"missing\"",
		"target \"bad\": unknown type \"unknown\"",
		"target \"bad\": unknown lang \"rust\"",
		"target \"bad\": unknown linker \"ld\"",
		"target \"bad\": depends_on references unknown target \"missing-bad\"",
		"target \"tool\": unknown toolchain \"clang\" for lang \"c\"",
		"dependency cycle: first -> second -> first",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error = %q, want it to contain %q", err, expected)
		}
	}
}

func TestValidateSuccess(t *testing.T) {
	config := parser.Config{
		Toolchains: map[string]map[string]parser.Toolchain{"c": {"gcc": {}}},
		Linkers:    map[string]parser.Linker{"gcc": {}},
		Targets: []parser.Target{
			{Name: "app", Src: []string{"main.c"}, Type: "binary", Lang: "c", Toolchain: "gcc", Linker: "gcc"},
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestParse(t *testing.T) {
	file := filepath.Join(t.TempDir(), "build.toml")
	content := []byte(`[project]
name = "example"
version = "1.2.3"

[toolchains.c.gcc]
run = "gcc"
flags = ["-Wall"]

[[target]]
name = "app"
src = ["main.c"]
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	config, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if config.Project.Name != "example" || config.Project.Version != "1.2.3" {
		t.Errorf("Project = %+v, want example 1.2.3", config.Project)
	}
	if config.Toolchains["c"]["gcc"].Run != "gcc" {
		t.Errorf("parsed toolchain = %+v", config.Toolchains["c"]["gcc"])
	}
	if len(config.Targets) != 1 || config.Targets[0].Name != "app" {
		t.Errorf("Targets = %+v, want one app target", config.Targets)
	}
}

func TestParseMissingFile(t *testing.T) {
	_, err := parser.Parse(filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("Parse() error = nil, want an error")
	}
}
