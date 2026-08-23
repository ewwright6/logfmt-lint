package main

import (
	"fmt"
	"strings"
)

// Field is a single key/value pair parsed from a logfmt line.
type Field struct {
	Key   string
	Value string
}

// Record is one parsed line: its raw text and the fields found in it.
type Record struct {
	Raw    string
	Fields []Field
}

// ParseError describes a syntax problem found while parsing a line.
type ParseError struct {
	Column  int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("column %d: %s", e.Column, e.Message)
}

// ParseLine parses a single logfmt-style line such as:
//
//	ts=2026-08-24T10:00:00Z level=error msg="connection refused" retries=3
//
// Bare tokens without "=" are kept as keys with an empty value, matching
// the convention used by logrus and Heroku's router output.
func ParseLine(line string) (Record, error) {
	rec := Record{Raw: line}
	i := 0
	n := len(line)

	for i < n {
		for i < n && line[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}

		keyStart := i
		for i < n && line[i] != '=' && line[i] != ' ' {
			i++
		}
		key := line[keyStart:i]
		if key == "" {
			return rec, &ParseError{Column: keyStart + 1, Message: "empty key"}
		}

		if i >= n || line[i] != '=' {
			rec.Fields = append(rec.Fields, Field{Key: key, Value: ""})
			continue
		}
		i++ // consume '='

		if i < n && line[i] == '"' {
			value, next, err := parseQuoted(line, i)
			if err != nil {
				return rec, err
			}
			rec.Fields = append(rec.Fields, Field{Key: key, Value: value})
			i = next
			continue
		}

		valStart := i
		for i < n && line[i] != ' ' {
			i++
		}
		rec.Fields = append(rec.Fields, Field{Key: key, Value: line[valStart:i]})
	}

	return rec, nil
}

// parseQuoted reads a double-quoted value starting at line[start] == '"'.
// It returns the unescaped value and the index just past the closing quote.
func parseQuoted(line string, start int) (string, int, error) {
	var b strings.Builder
	i := start + 1
	n := len(line)
	for i < n {
		c := line[i]
		if c == '\\' && i+1 < n && (line[i+1] == '"' || line[i+1] == '\\') {
			b.WriteByte(line[i+1])
			i += 2
			continue
		}
		if c == '"' {
			return b.String(), i + 1, nil
		}
		b.WriteByte(c)
		i++
	}
	return "", i, &ParseError{Column: start + 1, Message: "unterminated quoted value"}
}

// Validate checks a parsed record for problems that aren't syntax errors
// on their own, such as a key appearing more than once.
func Validate(rec Record) []error {
	seen := make(map[string]bool, len(rec.Fields))
	var errs []error
	for _, f := range rec.Fields {
		if seen[f.Key] {
			errs = append(errs, fmt.Errorf("duplicate key %q", f.Key))
			continue
		}
		seen[f.Key] = true
	}
	return errs
}
