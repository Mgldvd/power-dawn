package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	if os.Getuid() != 0 {
		printRootRequired()
		os.Exit(1)
	}

	initLogger()
	logOrDiscard().Info("Power-Dawn started")

	program := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

func printRootRequired() {
	// Derive a clean relative path for the binary so the hint is copy-pasteable.
	bin := filepath.Base(os.Args[0])
	hint := "sudo ./" + bin

	r := lipgloss.NewRenderer(os.Stderr)

	titleStyle := r.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true)

	hintCodeStyle := r.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	labelStyle := r.NewStyle().
		Foreground(lipgloss.Color("244"))

	boxStyle := r.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(1, 3)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("⛔  Permission denied"),
		"",
		labelStyle.Render("Power-Dawn must be run as root."),
		"",
		labelStyle.Render("Try:  ")+hintCodeStyle.Render(hint),
	)

	fmt.Fprintln(os.Stderr, boxStyle.Render(body))
}
