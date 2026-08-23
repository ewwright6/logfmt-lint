package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run processes each source in turn and returns the process exit code:
// 0 if every line parsed and validated cleanly, 1 otherwise.
func run(args []string, stdout, stderr io.Writer) int {
	sources := args
	if len(sources) == 0 {
		sources = []string{"-"}
	}

	hadError := false
	for _, src := range sources {
		if processSource(src, stdout, stderr) {
			hadError = true
		}
	}

	if hadError {
		return 1
	}
	return 0
}

// processSource reads one file (or stdin, for "-") a line at a time and
// returns true if any line failed to parse or validate.
func processSource(src string, stdout, stderr io.Writer) bool {
	var r io.Reader
	name := src

	if src == "-" {
		r = os.Stdin
		name = "stdin"
	} else {
		f, err := os.Open(src)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", src, err)
			return true
		}
		defer f.Close()
		r = f
	}

	hadError := false
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if line == "" {
			continue
		}

		rec, err := ParseLine(line)
		if err != nil {
			fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, err)
			hadError = true
			continue
		}

		if errs := Validate(rec); len(errs) > 0 {
			for _, verr := range errs {
				fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, verr)
			}
			hadError = true
			continue
		}

		PrintRecord(stdout, name, lineNo, rec)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", name, err)
		hadError = true
	}

	return hadError
}
