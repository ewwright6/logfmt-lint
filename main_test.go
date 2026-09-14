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
