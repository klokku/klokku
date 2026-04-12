package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"golang.org/x/term"
)

// Format represents the output format.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Detect returns the appropriate format based on TTY detection,
// or the explicitly requested format if set.
func Detect(explicit string) Format {
	if explicit != "" {
		return Format(explicit)
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return FormatText
	}
	return FormatJSON
}

// PrintJSON outputs data as indented JSON.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintText outputs data as a formatted table.
// headers is a list of column headers, rows is a list of rows where each row
// is a list of column values.
func PrintText(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// Print outputs data in the specified format. For JSON, it prints the data directly.
// For text, it uses the provided table formatter function.
func Print(format Format, data any, textFn func()) error {
	if format == FormatJSON {
		return PrintJSON(data)
	}
	textFn()
	return nil
}
