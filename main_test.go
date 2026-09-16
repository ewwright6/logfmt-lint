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
