package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/celson/lazyrclone/internal/config"
	"github.com/celson/lazyrclone/internal/rclone"
	"github.com/celson/lazyrclone/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	demoFlag := flag.Bool("demo", false, "Run in simulated demo mode without invoking rclone")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	vShortFlag := flag.Bool("v", false, "Print version and exit (shorthand)")
	rcloneBinFlag := flag.String("rclone-path", "rclone", "Path to the rclone executable")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "lazyrclone — A simple terminal UI for rclone\n\n")
		fmt.Fprintf(os.Stderr, "Usage: lazyrclone [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeybindings inside lazyrclone:\n")
		fmt.Fprintf(os.Stderr, "  1-4          Switch between Profiles, Explorer, Transfers, and Remotes\n")
		fmt.Fprintf(os.Stderr, "  d            Run honest dry-run diff preview\n")
		fmt.Fprintf(os.Stderr, "  r / Enter    Execute sync/copy operation\n")
		fmt.Fprintf(os.Stderr, "  ?            Open interactive help overlay\n")
		fmt.Fprintf(os.Stderr, "  q            Quit\n")
	}

	flag.Parse()

	if *versionFlag || *vShortFlag {
		fmt.Printf("lazyrclone %s (commit: %s, built: %s)\n", displayVersion(Version), Commit, Date)
		os.Exit(0)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config (%v), using defaults\n", err)
		cfg = config.DefaultConfig()
	}

	binPath := *rcloneBinFlag
	if binPath == "rclone" && cfg.Settings.RclonePath != "" {
		binPath = cfg.Settings.RclonePath
	}

	client := rclone.GetClient(binPath, *demoFlag)

	app := ui.NewAppModel(cfg, client)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazyrclone: %v\n", err)
		os.Exit(1)
	}
}

// displayVersion prefixes numeric versions (e.g. "0.1.1") with "v"; other
// values such as "dev" are shown as-is.
func displayVersion(v string) string {
	if v != "" && v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}
