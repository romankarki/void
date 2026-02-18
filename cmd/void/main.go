package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/void-shell/void/internal/config"
	"github.com/void-shell/void/internal/shell"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg, configFile, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to load config: %v\n", err)
		os.Exit(1)
	}

	app, err := shell.New(cfg, configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to initialize shell: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "void: %v\n", err)
		os.Exit(1)
	}
}
