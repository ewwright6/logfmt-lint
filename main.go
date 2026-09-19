package main

import (
	"bufio"
	"compress/gzip"
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
	var required requireSets
	fs.Var(&required, "require", "comma-separated list of keys that must be present in every line, e.g. level,msg. May be given more than once to accept alternative sets of required keys, for sources that mix line shapes.")
	jsonFlag := fs.Bool("json", false, "print each well-formed line as a JSON object instead of the aligned text form")
	strictFlag := fs.Bool("strict", false, "fail on bare keys (a key with no \"=\" and no value)")
	quietFlag := fs.Bool("quiet", false, "don't print well-formed lines, only errors")
	sepFlag := fs.String("sep", " ", "character separating key=value pairs (default space); use \\t for tab")
	quoteFlag := fs.String("quote", `"`, "character used to quote a value containing the separator (default \")")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	sep, err := parseSeparator(*sepFlag)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	quote, err := parseQuoteChar(*quoteFlag)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if quote == sep {
		fmt.Fprintf(stderr, "-quote and -sep cannot be the same character (%q)\n", quote)
		return 2
	}

	sources := fs.Args()
	if len(sources) == 0 {
		sources = []string{"-"}
	}

	color := !*jsonFlag && os.Getenv("NO_COLOR") == "" && isTerminal(stdout)

	hadError := false
	for _, src := range sources {
		if processSource(src, required, sep, quote, *jsonFlag, *strictFlag, *quietFlag, color, stdout, stderr) {
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

// requireSets collects one or more comma-separated -require values, one
// per flag occurrence, into alternative sets of required keys. It
// implements flag.Value so that -require can be repeated on the command
// line instead of only accepting one comma-separated list.
type requireSets [][]string

func (r *requireSets) String() string {
	if r == nil || len(*r) == 0 {
		return ""
	}
	parts := make([]string, len(*r))
	for i, set := range *r {
		parts[i] = strings.Join(set, ",")
	}
	return strings.Join(parts, " ")
}

func (r *requireSets) Set(value string) error {
	if keys := splitRequired(value); len(keys) > 0 {
		*r = append(*r, keys)
	}
	return nil
}

// parseSeparator turns a -sep flag value into the single byte it names.
// "\t" is accepted literally since a shell can't easily pass a raw tab as
// a command-line argument; anything else must be exactly one character.
func parseSeparator(s string) (byte, error) {
	if s == `\t` {
		s = "\t"
	}
	if len(s) != 1 {
		return 0, fmt.Errorf("-sep must be a single character, got %q", s)
	}
	if s[0] == '=' {
		return 0, fmt.Errorf("-sep cannot be \"=\", it would collide with the key=value separator")
	}
	return s[0], nil
}

// parseQuoteChar turns a -quote flag value into the single byte it names.
func parseQuoteChar(s string) (byte, error) {
	if len(s) != 1 {
		return 0, fmt.Errorf("-quote must be a single character, got %q", s)
	}
	if s[0] == '=' {
		return 0, fmt.Errorf("-quote cannot be \"=\", it would collide with the key=value separator")
	}
	return s[0], nil
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

// isGzip reports whether r starts with the gzip magic number, without
// consuming any bytes, so files and stdin alike can be transparently
// decompressed regardless of their name.
func isGzip(r *bufio.Reader) (bool, error) {
	header, err := r.Peek(2)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}
	return header[0] == 0x1f && header[1] == 0x8b, nil
}

// processSource reads one file (or stdin, for "-") a line at a time and
// returns true if any line failed to parse or validate.
func processSource(src string, required [][]string, sep, quote byte, jsonOut, strict, quiet, color bool, stdout, stderr io.Writer) bool {
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

	br := bufio.NewReader(r)
	if gz, err := isGzip(br); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", name, err)
		return true
	} else if gz {
		zr, err := gzip.NewReader(br)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", name, err)
			return true
		}
		defer zr.Close()
		r = zr
	} else {
		r = br
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

		rec, err := ParseLineWithOptions(line, sep, quote)
		if err != nil {
			fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, err)
			hadError = true
			continue
		}

		if errs := Validate(rec, required, strict); len(errs) > 0 {
			for _, verr := range errs {
				fmt.Fprintf(stderr, "%s:%d: %v\n", name, lineNo, verr)
			}
			hadError = true
			continue
		}

		if quiet {
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
