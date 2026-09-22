package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Project    ProjectConfig                   `toml:"project"`
	Build      BuildConfig                     `toml:"build"`
	Toolchains map[string]map[string]Toolchain `toml:"toolchains"`
	Linkers    map[string]Linker               `toml:"linkers"`
	Targets    []Target                        `toml:"target"`
}

type ProjectConfig struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type BuildConfig struct {
	BuildDir         string `toml:"build_dir"`
	DefaultToolchain string `toml:"default_toolchain"`
	DefaultLinker    string `toml:"default_linker"`
}

type Toolchain struct {
	Run    string   `toml:"run"`
	Flags  []string `toml:"flags"`
	Direct bool     `toml:"direct"`
}

type Linker struct {
	Run   string   `toml:"run"`
	Flags []string `toml:"flags"`
}

type Target struct {
	Name      string   `toml:"name"`
	Type      string   `toml:"type"` // binary, static, shared
	Lang      string   `toml:"lang"`
	Toolchain string   `toml:"toolchain"`
	Linker    string   `toml:"linker"`
	Src       []string `toml:"src"`
	Libs      []string `toml:"libs"`
	Flags     []string `toml:"flags"`
	DependsOn []string `toml:"depends_on"`
	Outfile   string   `toml:"outfile"`
}

func (c *Config) ApplyDefaults() {
	if c.Build.DefaultToolchain == "" {
		c.Build.DefaultToolchain = "gcc"
	}
	if c.Build.DefaultLinker == "" {
		c.Build.DefaultLinker = "gcc"
	}

	if c.Build.BuildDir == "" {
		c.Build.BuildDir = "build"
	}

	for i := range c.Targets {
		if c.Targets[i].Toolchain == "" {
			c.Targets[i].Toolchain = c.Build.DefaultToolchain
		}
		if c.Targets[i].Linker == "" {
			c.Targets[i].Linker = c.Build.DefaultLinker
		}
		if c.Targets[i].Type == "" {
			c.Targets[i].Type = "binary"
		}
		if c.Targets[i].Lang == "" {
			c.Targets[i].Lang = "c"
		}
		if c.Targets[i].Outfile == "" {
			c.Targets[i].Outfile = c.Targets[i].Name
		}
	}
}

func (c *Config) Validate() error {
	var validationErrors []error
	knownTypes := map[string]bool{
		"binary": true,
		"static": true,
		"shared": true,
	}
	targetIndexes := make(map[string]int, len(c.Targets))

	for index, target := range c.Targets {
		if previous, exists := targetIndexes[target.Name]; exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %d: duplicate target name %q (already used by target %d)", index, target.Name, previous))
		} else {
			targetIndexes[target.Name] = index
		}

		if len(target.Src) == 0 {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: src must not be empty", target.Name))
		}
		if !knownTypes[target.Type] {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown type %q", target.Name, target.Type))
		}
		if _, exists := c.Toolchains[target.Lang]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown lang %q", target.Name, target.Lang))
		} else if _, exists := c.Toolchains[target.Lang][target.Toolchain]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown toolchain %q for lang %q", target.Name, target.Toolchain, target.Lang))
		}
		if _, exists := c.Linkers[target.Linker]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown linker %q", target.Name, target.Linker))
		}
		for _, dependency := range target.DependsOn {
			found := false
			for _, candidate := range c.Targets {
				if candidate.Name == dependency {
					found = true
					break
				}
			}
			if !found {
				validationErrors = append(validationErrors, fmt.Errorf("target %q: depends_on references unknown target %q", target.Name, dependency))
			}
		}
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	states := make([]int, len(c.Targets))
	var visit func(int, []int)
	visit = func(index int, path []int) {
		states[index] = visiting
		path = append(path, index)
		for _, dependency := range c.Targets[index].DependsOn {
			dependencyIndex, exists := targetIndexes[dependency]
			if !exists {
				continue
			}
			switch states[dependencyIndex] {
			case unvisited:
				visit(dependencyIndex, path)
			case visiting:
				cycleStart := 0
				for cycleStart < len(path) && path[cycleStart] != dependencyIndex {
					cycleStart++
				}
				cycle := make([]string, 0, len(path)-cycleStart+1)
				for _, cycleIndex := range path[cycleStart:] {
					cycle = append(cycle, c.Targets[cycleIndex].Name)
				}
				cycle = append(cycle, c.Targets[dependencyIndex].Name)
				validationErrors = append(validationErrors, fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " -> ")))
			}
		}
		states[index] = visited
	}
	for index := range c.Targets {
		if states[index] == unvisited {
			visit(index, nil)
		}
	}

	return errors.Join(validationErrors...)
}

func Parse(file string) (*Config, error) {
	var conf Config
	_, err := toml.DecodeFile(file, &conf)
	return &conf, err
}
