package utils

import (
	"fmt"
	"time"
	tea "github.com/charmbracelet/bubbletea"
)

// progressUpdateThreshold controls how often the progress bar updates.
const progressUpdateThreshold = 150 * time.Millisecond


// NewProgressWriter creates a new ProgressWriter.
func NewProgressWriter(total int64, filename, direction string, p *tea.Program) *ProgressWriter {
	return &ProgressWriter{
		total:     total,
		startTime: time.Now(),
		filename:  filename,
		direction: direction,
		program:   p,
	}
}

// Write implements the io.Writer interface for ProgressWriter.
// It is called for each chunk of data that is transferred.
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	// Throttle updates to prevent the TUI from re-rendering too frequently.
	if time.Since(pw.lastUpdate) < progressUpdateThreshold && pw.written < pw.total {
		return n, nil
	}

	pw.lastUpdate = time.Now()

	// Calculate progress and transfer rate.
	percentage := float64(pw.written) * 100 / float64(pw.total)
	elapsed := time.Since(pw.startTime).Seconds()
	var rate float64
	if elapsed > 0 {
		rate = float64(pw.written) / elapsed / 1024 // Rate in KB/s
	}

	// Determine the unit (KB/s or MB/s).
	rateStr := fmt.Sprintf("%.2f KB/s", rate)
	if rate > 1024 {
		rateStr = fmt.Sprintf("%.2f MB/s", rate/1024)
	}

	// Send a message to the TUI to update the progress display.
	if pw.program != nil {
		pw.program.Send(FileTransferMsg{
			Filename:  pw.filename,
			Progress:  percentage,
			Rate:      rateStr,
			Direction: pw.direction,
		})
	}

	return n, nil
}