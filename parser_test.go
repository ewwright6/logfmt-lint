package main

import (
	"reflect"
	"testing"
)

func TestParseLineFields(t *testing.T) {
	cases := []struct {
		name string
		line string
		want []Field
	}{
		{
			name: "simple pairs",
			line: "level=error msg=boom",
			want: []Field{{Key: "level", Value: "error"}, {Key: "msg", Value: "boom"}},
		},
		{
			name: "bare key gets empty value",
			line: "level=error debug",
			want: []Field{{Key: "level", Value: "error"}, {Key: "debug", Value: "", Bare: true}},
		},
		{
			name: "quoted value with spaces",
			line: `msg="connection refused" retries=3`,
			want: []Field{{Key: "msg", Value: "connection refused"}, {Key: "retries", Value: "3"}},
		},
		{
			name: "quoted value with escaped quote and backslash",
			line: `msg="say \"hi\" then C:\\path"`,
			want: []Field{{Key: "msg", Value: `say "hi" then C:\path`}},
		},
		{
			name: "empty quoted value",
			line: `msg=""`,
			want: []Field{{Key: "msg", Value: ""}},
		},
		{
			name: "unquoted empty value before next key",
			line: "a= b=1",
			want: []Field{{Key: "a", Value: ""}, {Key: "b", Value: "1"}},
		},
		{
			name: "repeated and leading spaces are ignored",
			line: "   a=1    b=2   ",
			want: []Field{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
		},
		{
			name: "empty line has no fields",
			line: "",
			want: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec, err := ParseLine(c.line)
			if err != nil {
				t.Fatalf("ParseLine(%q) returned error: %v", c.line, err)
			}
			if !reflect.DeepEqual(rec.Fields, c.want) {
				t.Errorf("ParseLine(%q) = %#v, want %#v", c.line, rec.Fields, c.want)
			}
		})
	}
}

func TestParseLineErrors(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		wantColumn int
	}{
		{
			name:       "empty key from leading equals",
			line:       "=value",
			wantColumn: 1,
		},
		{
			name:       "empty key between two valid fields",
			line:       "a=1 =2 b=3",
			wantColumn: 5,
		},
		{
			name:       "unterminated quoted value",
			line:       `msg="never closed`,
			wantColumn: 5,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseLine(c.line)
			if err == nil {
				t.Fatalf("ParseLine(%q) returned no error, want one", c.line)
			}
			perr, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("ParseLine(%q) returned %T, want *ParseError", c.line, err)
			}
			if perr.Column != c.wantColumn {
				t.Errorf("ParseLine(%q) column = %d, want %d", c.line, perr.Column, c.wantColumn)
			}
		})
	}
}

func TestValidateDuplicateKeys(t *testing.T) {
	rec := Record{Fields: []Field{
		{Key: "level", Value: "error"},
		{Key: "msg", Value: "boom"},
		{Key: "level", Value: "warn"},
	}}

	errs := Validate(rec, nil, false)
	if len(errs) != 1 {
		t.Fatalf("Validate() returned %d errors, want 1: %v", len(errs), errs)
	}
	want := `duplicate key "level"`
	if errs[0].Error() != want {
		t.Errorf("Validate() error = %q, want %q", errs[0].Error(), want)
	}
}

func TestValidateRequiredKeys(t *testing.T) {
	rec := Record{Fields: []Field{
		{Key: "level", Value: "error"},
	}}

	errs := Validate(rec, [][]string{{"level", "msg"}}, false)
	if len(errs) != 1 {
		t.Fatalf("Validate() returned %d errors, want 1: %v", len(errs), errs)
	}
	want := `missing required key "msg"`
	if errs[0].Error() != want {
		t.Errorf("Validate() error = %q, want %q", errs[0].Error(), want)
	}
}

func TestValidateRequiredKeySetsSatisfiesEither(t *testing.T) {
	rec := Record{Fields: []Field{
		{Key: "method", Value: "GET"},
		{Key: "path", Value: "/health"},
		{Key: "status", Value: "200"},
	}}

	sets := [][]string{{"level", "msg"}, {"method", "path", "status"}}
	if errs := Validate(rec, sets, false); errs != nil {
		t.Errorf("Validate() = %v, want no errors (record satisfies the second set)", errs)
	}
}

func TestValidateRequiredKeySetsReportsClosestMatch(t *testing.T) {
	rec := Record{Fields: []Field{
		{Key: "level", Value: "error"},
	}}

	// "level" alone is one key short of the first set and two short of
	// the second, so the error should name only "msg".
	sets := [][]string{{"level", "msg"}, {"method", "path", "status"}}
	errs := Validate(rec, sets, false)
	if len(errs) != 1 {
		t.Fatalf("Validate() returned %d errors, want 1: %v", len(errs), errs)
	}
	want := `missing required key "msg"`
	if errs[0].Error() != want {
		t.Errorf("Validate() error = %q, want %q", errs[0].Error(), want)
	}
}

func TestValidateNoRequiredKeys(t *testing.T) {
	rec := Record{Fields: []Field{{Key: "level", Value: "error"}}}
	if errs := Validate(rec, nil, false); errs != nil {
		t.Errorf("Validate() = %v, want no errors", errs)
	}
}

func TestValidateStrictRejectsBareKeys(t *testing.T) {
	rec := Record{Fields: []Field{
		{Key: "level", Value: "error"},
		{Key: "debug", Value: "", Bare: true},
	}}

	errs := Validate(rec, nil, true)
	if len(errs) != 1 {
		t.Fatalf("Validate() returned %d errors, want 1: %v", len(errs), errs)
	}
	want := `bare key "debug" has no value`
	if errs[0].Error() != want {
		t.Errorf("Validate() error = %q, want %q", errs[0].Error(), want)
	}
}

func TestValidateStrictAllowsExplicitEmptyValue(t *testing.T) {
	rec := Record{Fields: []Field{{Key: "a", Value: "", Bare: false}}}
	if errs := Validate(rec, nil, true); errs != nil {
		t.Errorf("Validate() = %v, want no errors", errs)
	}
}

func TestValidateNonStrictAllowsBareKeys(t *testing.T) {
	rec := Record{Fields: []Field{{Key: "debug", Value: "", Bare: true}}}
	if errs := Validate(rec, nil, false); errs != nil {
		t.Errorf("Validate() = %v, want no errors", errs)
	}
}
