// Command tuti-tui is an interactive terminal client for testing a running
// tuti-server instance: chat, capture uploads, and health checks.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tuti-tui/internal/tui"
)

func main() {
	var baseURL string
	flag.StringVar(&baseURL, "server", envOr("TUTI_SERVER_URL", "http://localhost:8080"), "tuti-server base URL")
	flag.Parse()

	p := tea.NewProgram(tui.New(baseURL), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuti-tui:", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
