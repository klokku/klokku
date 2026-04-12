package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration_GoFormat(t *testing.T) {
	secs, err := parseDuration("2h30m")
	require.NoError(t, err)
	assert.Equal(t, 9000, secs)
}

func TestParseDuration_RawSeconds(t *testing.T) {
	secs, err := parseDuration("9000")
	require.NoError(t, err)
	assert.Equal(t, 9000, secs)
}

func TestParseDuration_MinutesOnly(t *testing.T) {
	secs, err := parseDuration("45m")
	require.NoError(t, err)
	assert.Equal(t, 2700, secs)
}

func TestParseDuration_Invalid(t *testing.T) {
	_, err := parseDuration("invalid")
	assert.Error(t, err)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0m"},
		{60, "1m"},
		{3600, "1h"},
		{5400, "1h30m"},
		{28800, "8h"},
		{9000, "2h30m"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatDuration(tt.seconds), "for %d seconds", tt.seconds)
	}
}
