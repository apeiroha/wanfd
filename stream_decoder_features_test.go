package wanf

import (
	"strings"
	"testing"
)

// TestStreamDecoder_MisplacedVarError tests that the decoder correctly returns an error
// if a var declaration appears after a body statement, as per the new spec.
func TestStreamDecoder_MisplacedVarError(t *testing.T) {
	wanfData := `
		host = "localhost"
		var a = 1 // This should be an error
	`
	var cfg struct {
		Host string `wanf:"host"`
	}

	r := strings.NewReader(wanfData)
	decoder, err := NewStreamDecoder(r)
	if err != nil {
		t.Fatalf("NewStreamDecoder failed: %v", err)
	}

	err = decoder.Decode(&cfg)
	if err == nil {
		t.Fatal("Expected an error for misplaced var declaration, but got nil")
	}

	expectedError := "var statements are not supported in stream decoding mode"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, but got: %v", expectedError, err)
	}
}

// TestStreamDecoder_ImportError tests that the decoder correctly returns an error
// if an import statement is used in stream mode.
func TestStreamDecoder_ImportError(t *testing.T) {
	wanfData := `
		import "other.wanf"
		host = "localhost"
	`
	var cfg struct {
		Host string `wanf:"host"`
	}

	r := strings.NewReader(wanfData)
	decoder, err := NewStreamDecoder(r)
	if err != nil {
		t.Fatalf("NewStreamDecoder failed: %v", err)
	}

	err = decoder.Decode(&cfg)
	if err == nil {
		t.Fatal("Expected an error for import statement, but got nil")
	}

	expectedError := "import statements are not supported in stream decoding mode"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, but got: %v", expectedError, err)
	}
}
