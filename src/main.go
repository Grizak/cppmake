package main

import (
	"fmt"
	"os"

	"cppmake/src/backend"
	"cppmake/src/parser"

	"github.com/spf13/cobra"
)

var backendName string
var rootDir string

var rootCmd = &cobra.Command{
	Use:   "cppmake",
	Short: "A simple build file generator",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return os.Chdir(rootDir)
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate a build file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat("build.toml"); err != nil {
			return fmt.Errorf("could not find build.toml: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := parser.Parse("build.toml")
		if err != nil {
			return fmt.Errorf("parse build.toml: %w", err)
		}

		cfg.ApplyDefaults()

		b, err := backend.BackendFactory(backendName)
		if err != nil {
			return fmt.Errorf("create backend %q: %w", backendName, err)
		}

		content := b.Generate(*cfg)

		if err := os.WriteFile(
			b.Filename(),
			content, 0644,
		); err != nil {
			return fmt.Errorf("write %s: %w", b.Filename(), err)
		}

		fmt.Printf("Generated %s using the %s backend.\n", b.Filename(), backendName)
		return nil
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove generated build files",
	RunE: func(cmd *cobra.Command, args []string) error {
		files := []string{
			"build.ninja",
			".ninja_log",
			".ninja_deps",
		}

		for _, filename := range files {
			err := os.Remove(filename)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", filename, err)
			}
		}

		fmt.Println("Cleaned.")
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a cppmake project",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Add project initialization logic here.
		fmt.Println("Initialized.")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&rootDir,
		"context",
		"C",
		".",
		"Directory to execute from",
	)

	buildCmd.Flags().StringVarP(
		&backendName,
		"backend",
		"b",
		"ninja",
		"build backend to use",
	)

	rootCmd.AddCommand(buildCmd, initCmd, cleanCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
