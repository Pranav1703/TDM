package utils

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type PeersUpdatedMsg struct {
	Peers []string
}

// FileTransferMsg is sent by the progress writer during a file transfer.
type FileTransferMsg struct {
	Filename  string
	Progress  float64
	Rate      string
	Direction string // "Sending" or "Receiving"
}

// LogMsg is a generic message for logging information to the UI.
type LogMsg struct {
	Message string
}

type ProgressWriter struct {
	total      int64
	written    int64
	startTime  time.Time
	lastUpdate time.Time
	filename   string
	direction  string
	program    *tea.Program
}
