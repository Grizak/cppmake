package main

import (
	"cppmake/src/backend"
	"cppmake/src/parser"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: cppmake <build|init|clean> [--backend=ninja]")
		os.Exit(1)
	}

	cfg, err := parser.Parse("build.toml")
	if err != nil {
		panic(err)
	}
	cfg.ApplyDefaults()

	switch os.Args[1] {
	case "build":
		backendName := flag.String("backend", "ninja", "")
		flag.Parse()
		// factory
		b, err := backend.BackendFactory(*backendName)
		if err != nil {
			panic(err)
		}
		content := b.Generate(*cfg)

		err = os.WriteFile(b.Filename(), content, 0644)
		if err != nil {
			panic(err)
		}
	case "clean":
		os.Remove("build.ninja")
		os.Remove(".ninja_log")
		os.Remove(".ninja_deps")
		fmt.Println("Cleaned.")
	}
}
