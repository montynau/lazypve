package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"

	"github.com/MontyNau/lazypve/internal/config"
	"github.com/MontyNau/lazypve/internal/pve"
	"github.com/MontyNau/lazypve/internal/ui"
)

func main() {
	// .env is optional — ignore if it doesn't exist, real env vars still work.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "lazypve: "+err.Error())
		fmt.Fprintln(os.Stderr, "set LAZYPVE_HOST, LAZYPVE_TOKEN_ID, LAZYPVE_TOKEN_SECRET (and optionally LAZYPVE_INSECURE_SKIP_VERIFY=true)")
		os.Exit(1)
	}

	client := pve.NewClient(cfg)

	p := tea.NewProgram(ui.New(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazypve: "+err.Error())
		os.Exit(1)
	}
}
