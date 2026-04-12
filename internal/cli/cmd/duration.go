package cmd

import (
	"fmt"
	"strconv"
	"time"
)

// parseDuration parses a duration string like "2h30m" or raw seconds "9000".
// Returns duration in seconds.
func parseDuration(s string) (int, error) {
	// Try raw seconds first
	if secs, err := strconv.Atoi(s); err == nil {
		return secs, nil
	}
	// Try Go duration format
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: use format like 2h30m or raw seconds", s)
	}
	return int(d.Seconds()), nil
}

// formatDuration formats seconds into human-readable form like "2h30m".
func formatDuration(seconds int) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
