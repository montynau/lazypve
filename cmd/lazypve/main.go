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

	clusters, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "lazypve: "+err.Error())
		fmt.Fprintln(os.Stderr, "set LAZYPVE_HOST, LAZYPVE_TOKEN_ID, LAZYPVE_TOKEN_SECRET (and optionally LAZYPVE_INSECURE_SKIP_VERIFY=true),")
		fmt.Fprintln(os.Stderr, "or set LAZYPVE_CLUSTERS_FILE to a JSON file listing multiple clusters")
		os.Exit(1)
	}

	clients := make(map[string]*pve.Client, len(clusters))
	for _, c := range clusters {
		clients[c.Name] = pve.NewClient(c)
	}

	p := tea.NewProgram(ui.New(clients), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazypve: "+err.Error())
		os.Exit(1)
	}
}
