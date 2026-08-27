package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// PrintRecord writes rec in an aligned, human-readable form to w, labeled
// with its source and line number so the reader can find it in the
// original file (or stdin stream) it came from.
func PrintRecord(w io.Writer, source string, lineNo int, rec Record) {
	fmt.Fprintf(w, "%s:%d\n", source, lineNo)
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
		fmt.Fprintf(w, "  %-*s = %s\n", width, f.Key, f.Value)
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
