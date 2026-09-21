package parser_test

import (
	"os"
	"path/filepath"
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
