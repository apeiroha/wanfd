package wanf

import (
	"strings"
	"testing"
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
