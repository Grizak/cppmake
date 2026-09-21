package parser

import "github.com/BurntSushi/toml"

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
			c.Targets[1].Outfile = c.Targets[1].Name
		}
	}
}

func Parse(file string) (*Config, error) {
	var conf Config
	_, err := toml.DecodeFile(file, &conf)
	return &conf, err
}
