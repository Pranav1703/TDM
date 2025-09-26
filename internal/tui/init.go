package tui

import (

	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func InitTui() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}