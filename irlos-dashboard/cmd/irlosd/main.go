// SPDX-License-Identifier: GPL-3.0-or-later
// Command irlosd is the Irlos IRL streaming dashboard — an SSH-accessible
// control room TUI for the Irlos streaming OS.
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ethanmanners/irlos-dashboard/internal/model"
)

func main() {
	// Log to stderr (captured by the parent shell, not the TUI).
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lshortfile)

	p := tea.NewProgram(
		model.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "irlosd: %v\n", err)
		os.Exit(1)
	}
}
