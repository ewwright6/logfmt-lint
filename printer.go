package main

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiCyan  = "\033[36m"
)

// PrintRecord writes rec in an aligned, human-readable form to w, labeled
// with its source and line number so the reader can find it in the
// original file (or stdin stream) it came from. When color is true, the
// header and field keys are wrapped in ANSI escape codes; callers should
// only set it when w is known to be a terminal.
func PrintRecord(w io.Writer, source string, lineNo int, rec Record, color bool) {
	if color {
		fmt.Fprintf(w, "%s%s:%d%s\n", ansiBold, source, lineNo, ansiReset)
	} else {
		fmt.Fprintf(w, "%s:%d\n", source, lineNo)
	}
	if len(rec.Fields) == 0 {
		fmt.Fprintln(w, "  (no fields)")
		return
	}

	width := 0
	for _, f := range rec.Fields {
		if len(f.Key) > width {
			width = len(f.Key)
		}
	}

	for _, f := range rec.Fields {
		if color {
			fmt.Fprintf(w, "  %s%-*s%s = %s\n", ansiCyan, width, f.Key, ansiReset, f.Value)
		} else {
			fmt.Fprintf(w, "  %-*s = %s\n", width, f.Key, f.Value)
		}
	}
}

// jsonRecord is the shape written by PrintRecordJSON. Fields is a map
// rather than the ordered []Field slice because a record only reaches
// here after Validate has rejected duplicate keys, so no ordering
// information is lost by collapsing to key/value pairs.
type jsonRecord struct {
	Source string            `json:"source"`
	Line   int               `json:"line"`
	Fields map[string]string `json:"fields"`
}

// PrintRecordJSON writes rec to w as a single line of JSON, suitable for
// piping into jq or another JSON-consuming tool. One record per line
// (JSON Lines), matching the one-record-per-input-line shape of the
// pretty printer.
func PrintRecordJSON(w io.Writer, source string, lineNo int, rec Record) error {
	fields := make(map[string]string, len(rec.Fields))
	for _, f := range rec.Fields {
		fields[f.Key] = f.Value
	}
	return json.NewEncoder(w).Encode(jsonRecord{
		Source: source,
		Line:   lineNo,
		Fields: fields,
	})
}
