package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifloresarg/gh-projects/internal/config"
	"github.com/ifloresarg/gh-projects/internal/github"
	"github.com/ifloresarg/gh-projects/internal/tui"
)

var version = "dev"

func main() {
	_ = version

	if err := github.CheckAuth(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	owner := flag.String("owner", "", "GitHub owner (user or org)")
	number := flag.Int("number", 0, "Project number")
	view := flag.String("view", "", "View name to load directly (case-insensitive)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *owner != "" {
		cfg.DefaultOwner = *owner
	}
	if *number != 0 {
		cfg.DefaultProject = *number
	}

	rawClient, err := github.NewGraphQLClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing GitHub client: %v\n", err)
		os.Exit(1)
	}

	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	client := github.NewCachedClient(rawClient, cacheTTL)

	app := tui.NewApp(cfg, client)
	if *owner != "" && *number != 0 {
		app = app.WithInitialState(tui.ViewLoading).WithInitialView(*view)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
