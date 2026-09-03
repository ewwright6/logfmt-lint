package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses flags, then processes each source in turn and returns the
// process exit code: 0 if every line parsed and validated cleanly, 1
// otherwise, 2 if the arguments themselves couldn't be parsed.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logfmt-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	requireFlag := fs.String("require", "", "comma-separated list of keys that must be present in every line, e.g. level,msg")
	jsonFlag := fs.Bool("json", false, "print each well-formed line as a JSON object instead of the aligned text form")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	required := splitRequired(*requireFlag)

	sources := fs.Args()
	if len(sources) == 0 {
		sources = []string{"-"}
	}

	color := !*jsonFlag && os.Getenv("NO_COLOR") == "" && isTerminal(stdout)

	hadError := false
	for _, src := range sources {
		if processSource(src, required, *jsonFlag, color, stdout, stderr) {
			hadError = true
		}
	}

	if hadError {
		return 1
	}
	return 0
}

// splitRequired turns a comma-separated flag value into a list of trimmed,
// non-empty key names.
func splitRequired(s string) []string {
	if s == "" {
		return nil
	}
	var keys []string
	for _, part := range strings.Split(s, ",") {
		key := strings.TrimSpace(part)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// isTerminal reports whether w is a character device such as a terminal,
// which is the case PrintRecord's ANSI colors are meant for. Piping or
// redirecting stdout swaps in a plain file or pipe, so this also serves
// as the check that keeps escape codes out of redirected output.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// processSource reads one file (or stdin, for "-") a line at a time and
// returns true if any line failed to parse or validate.
func processSource(src string, required []string, jsonOut, color bool, stdout, stderr io.Writer) bool {
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

		if errs := Validate(rec, required); len(errs) > 0 {
			for _, verr := range errs {
				fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, verr)
			}
			hadError = true
			continue
		}

		if jsonOut {
			if err := PrintRecordJSON(stdout, name, lineNo, rec); err != nil {
				fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, err)
				hadError = true
			}
			continue
		}
		PrintRecord(stdout, name, lineNo, rec, color)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", name, err)
		hadError = true
	}

	return hadError
}
