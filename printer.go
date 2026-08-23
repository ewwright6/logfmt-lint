package main

import (
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
