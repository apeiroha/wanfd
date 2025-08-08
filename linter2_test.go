package wanf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func TestLint2(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		wantOutput     string
		opts           FormatOptions
		wantErrors     []string
		expectNoChange bool
	}{
		{
			name: "redundant label",
			input: `
database "main" {
	host = "localhost"
}`,
			wantOutput: `
database {
	host = "localhost"
}`,
			wantErrors: []string{"the label \"main\" is redundant"},
		},
		{
			name: "redundant comma",
			input: `
server {
	port = 8080,
	host = "0.0.0.0"
}`,
			wantOutput: `
server {
	port = 8080
	host = "0.0.0.0"
}`,
			wantErrors: []string{"redundant comma"},
		},
		{
			name: "well-formatted file",
			input: `
server "api" {
	host = "localhost"
}
server "grpc" {
	host = "localhost"
}`,
			wantOutput: `server "api" {
	host = "localhost"
}

server "grpc" {
	host = "localhost"
}`,
		},
		{
			name: "streaming style",
			input: `server "b" {
	host = "2"
}
server "a" {
	host = "1"
}`,
			opts: FormatOptions{Style: StyleStreaming, EmptyLines: false},
			wantOutput: `server "b" {
	host = "2"
}
server "a" {
	host = "1"
}`,
		},
		{
			name:       "unclosed block comment",
			input:      `a = 1 /* comment`,
			wantOutput: `a = 1`,
			wantErrors: []string{"unclosed block comment"},
		},
		{
			name: "redundant trailing comma in list",
			input: `
list = [
	"a",
	"b",
]`,
			wantOutput: `
list = [
	"a",
	"b"
]`,
			wantErrors: []string{"redundant trailing comma in list literal"},
		},
		{
			name: "missing trailing comma in map",
			input: `
dashMap = {[
	key1 = "value1"
]}`,
			wantOutput: `
dashMap = {[
		key1 = "value1",
]}`,
			wantErrors: []string{"missing trailing comma"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.opts.Style == 0 && !tc.opts.EmptyLines {
				tc.opts.Style = StyleDefault
				tc.opts.EmptyLines = true
			}
			program, errs := Lint([]byte(tc.input))

			if len(tc.wantErrors) > 0 {
				if len(errs) != len(tc.wantErrors) {
					t.Fatalf("expected %d errors, got %d. Errors: %v", len(tc.wantErrors), len(errs), errs)
				}
				for i, wantErr := range tc.wantErrors {
					if !strings.Contains(errs[i].Error(), wantErr) {
						t.Errorf("expected error %q to contain %q", errs[i], wantErr)
					}
				}
			} else if len(errs) > 0 {
				t.Fatalf("expected no errors, got: %v", errs)
			}

			formattedBytes := Format(program, tc.opts)

			if tc.expectNoChange {
				if string(formattedBytes) != tc.input {
					t.Errorf("expected no change, but got diff:\n--- want\n%s\n--- got\n%s", tc.input, string(formattedBytes))
				}
			} else {
				// Normalize whitespace for comparison
				got := strings.TrimSpace(string(formattedBytes))
				want := strings.TrimSpace(tc.wantOutput)
				if got != want {
					t.Errorf("output mismatch:\n--- want\n%s\n--- got\n%s", want, got)
				}
			}
		})
	}
}

func TestLintJSONOutput(t *testing.T) {
	input := `
list = [
	"a",
]
`
	// Note: The redundant comma is at line 4, column 2. The ']' is at line 5, column 1.
	// The error is attached to the ']' token, so the line is 5, column is 1.
	// The "comma" is approximated as being just before it.
	wantJSON := `[{
  "line": 4,
  "column": 0,
  "endLine": 4,
  "endColumn": 1,
  "message": "redundant trailing comma in list literal",
  "level": 0,
  "type": 1
}]`

	_, errs := Lint([]byte(input))
	if len(errs) != 1 {
		t.Fatalf("expected 1 lint error, got %d", len(errs))
	}

	var buf bytes.Buffer
	err := json.MarshalWrite(&buf, errs, jsontext.Multiline(true), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("failed to marshal lint errors to json: %v", err)
	}

	// We can't do a direct string comparison due to the unpredictable order of JSON fields.
	// Instead, we'll unmarshal both and compare the structs.
	var gotErr []LintError
	if err := json.Unmarshal(buf.Bytes(), &gotErr); err != nil {
		t.Fatalf("failed to unmarshal generated json: %v", err)
	}

	var wantErr []LintError
	if err := json.Unmarshal([]byte(wantJSON), &wantErr); err != nil {
		t.Fatalf("failed to unmarshal want json: %v", err)
	}

	// For simplicity in this test, just compare the relevant fields.
	if gotErr[0].Line != wantErr[0].Line ||
		gotErr[0].Column != wantErr[0].Column ||
		gotErr[0].EndLine != wantErr[0].EndLine ||
		gotErr[0].EndColumn != wantErr[0].EndColumn {
		t.Errorf("JSON output mismatch for positions.\nGot:\n%+v\nWant:\n%+v", gotErr[0], wantErr[0])
	}

	if gotErr[0].Message != wantErr[0].Message {
		t.Errorf("JSON output mismatch for message.\nGot: %q\nWant: %q", gotErr[0].Message, wantErr[0].Message)
	}
}
