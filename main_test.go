package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunQuietWithFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	if err := os.WriteFile(path, []byte("level=error msg=boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-quiet", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr: %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty output for -quiet", stdout.String())
	}
}

func TestRunQuietStillReportsErrors(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	if err := os.WriteFile(path, []byte(`msg="never closed`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-quiet", path}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty output for -quiet", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unterminated quoted value") {
		t.Errorf("stderr = %q, want it to mention the parse error", stderr.String())
	}
}

func TestRunCommaSeparator(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	if err := os.WriteFile(path, []byte("level=error,msg=boom,retries=3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-sep=,", "-json", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"retries":"3"`) {
		t.Errorf("stdout = %q, want it to contain the comma-separated fields", stdout.String())
	}
}

func TestRunInvalidSeparatorIsRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-sep=ab", "-"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-sep") {
		t.Errorf("stderr = %q, want it to mention -sep", stderr.String())
	}
}

func TestRunCustomQuoteChar(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	line := "level=error,msg=`boom, again`,retries=3\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-sep=,", "-quote=`", "-json", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"msg":"boom, again"`) {
		t.Errorf("stdout = %q, want it to contain the backtick-quoted field", stdout.String())
	}
}

func TestRunQuoteSameAsSeparatorIsRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-quote=,", "-sep=,", "-"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-quote and -sep") {
		t.Errorf("stderr = %q, want it to mention the conflict", stderr.String())
	}
}

func TestRunInvalidQuoteIsRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-quote=ab", "-"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-quote") {
		t.Errorf("stderr = %q, want it to mention -quote", stderr.String())
	}
}

func TestParseQuoteChar(t *testing.T) {
	cases := []struct {
		in      string
		want    byte
		wantErr bool
	}{
		{in: `"`, want: '"'},
		{in: "`", want: '`'},
		{in: "'", want: '\''},
		{in: "", wantErr: true},
		{in: "ab", wantErr: true},
		{in: "=", wantErr: true},
	}

	for _, c := range cases {
		got, err := parseQuoteChar(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseQuoteChar(%q) returned no error, want one", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseQuoteChar(%q) returned error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseQuoteChar(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseSeparator(t *testing.T) {
	cases := []struct {
		in      string
		want    byte
		wantErr bool
	}{
		{in: " ", want: ' '},
		{in: ",", want: ','},
		{in: `\t`, want: '\t'},
		{in: "", wantErr: true},
		{in: "ab", wantErr: true},
		{in: "=", wantErr: true},
	}

	for _, c := range cases {
		got, err := parseSeparator(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseSeparator(%q) returned no error, want one", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSeparator(%q) returned error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseSeparator(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRunSortOrdersFieldsAlphabetically(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	if err := os.WriteFile(path, []byte("retries=3 level=error msg=boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-sort", path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr: %s", code, stderr.String())
	}
	levelIdx := strings.Index(stdout.String(), "level")
	msgIdx := strings.Index(stdout.String(), "msg")
	retriesIdx := strings.Index(stdout.String(), "retries")
	if !(levelIdx < msgIdx && msgIdx < retriesIdx) {
		t.Errorf("stdout = %q, want fields in alphabetical order (level, msg, retries)", stdout.String())
	}
}

func TestRunWithoutQuietPrintsWellFormedLines(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	if err := os.WriteFile(path, []byte("level=error msg=boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr: %s", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty, want the pretty-printed record")
	}
}
