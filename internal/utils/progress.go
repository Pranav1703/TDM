package utils

import (
	"fmt"
	"os"
	"time"
)

// progressUpdateThreshold controls how often the progress bar updates.
const progressUpdateThreshold = 150 * time.Millisecond

// ProgressWriter is a custom writer that tracks the progress of a file transfer.
type ProgressWriter struct {
	total      int64
	written    int64
	startTime  time.Time
	lastUpdate time.Time
	message    string
}

// NewProgressWriter creates a new ProgressWriter.
func NewProgressWriter(total int64, message string) *ProgressWriter {
	return &ProgressWriter{
		total:     total,
		startTime: time.Now(),
		message:   message,
	}
}

// Write implements the io.Writer interface for ProgressWriter.
// It is called for each chunk of data that is transferred.
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	// Throttle updates to prevent flickering, but always show the final 100% update.
	if time.Since(pw.lastUpdate) < progressUpdateThreshold && pw.written < pw.total {
		return n, nil
	}

	pw.lastUpdate = time.Now()

	// Calculate progress and transfer rate
	percentage := float64(pw.written) * 100 / float64(pw.total)
	elapsed := time.Since(pw.startTime).Seconds()
	var rate float64
	if elapsed > 0 {
		rate = float64(pw.written) / elapsed / 1024 // Rate in KB/s
	}

	// Determine the unit (KB/s or MB/s)
	rateStr := fmt.Sprintf("%.2f KB/s", rate)
	if rate > 1024 {
		rateStr = fmt.Sprintf("%.2f MB/s", rate/1024)
	}

	// Print the progress to os.Stderr on a single line to avoid buffering.
	// Adding padding with spaces at the end to clear any previous characters.
	fmt.Fprintf(os.Stderr, "\r%s... %.2f%% complete (%s)  ", pw.message, percentage, rateStr)

	return n, nil
}