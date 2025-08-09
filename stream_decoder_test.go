package wanf

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestStreamDecoder_RoundTrip 测试 StreamDecoder 是否能正确解码由标准 Encoder 编码的数据.
// 这确保了流式解码器和标准解码器在行为上的一致性.
func TestStreamDecoder_RoundTrip(t *testing.T) {
	// This struct and test data are borrowed from TestEncoder_Styles in wanf_test.go
	// to provide a consistent and complex test case.
	type SubBlock struct {
		B_sub_kv  string            `wanf:"b_sub_kv"`
		A_sub_kv  string            `wanf:"a_sub_kv"`
		C_sub_map map[string]string `wanf:"c_sub_map"`
	}
	type Config struct {
		C_kv    string   `wanf:"c_kv"`
		A_block SubBlock `wanf:"a_block"`
		B_kv    int      `wanf:"b_kv"`
	}

	testData := Config{
		C_kv: "c",
		A_block: SubBlock{
			B_sub_kv: "b",
			A_sub_kv: "a",
			C_sub_map: map[string]string{
				"sub_key": "sub_val",
			},
		},
		B_kv: 123,
	}

	// 1. Encode the test data into a WANF byte buffer.
	var buf bytes.Buffer
	encoder := NewEncoder(&buf) // Use default style
	if err := encoder.Encode(testData); err != nil {
		t.Fatalf("Setup failed: could not encode test data: %v", err)
	}

	// 2. Use a strings.Reader to simulate a stream from the encoded data.
	streamReader := strings.NewReader(buf.String())

	// 3. Use the new StreamDecoder to decode from the stream.
	var decodedCfg Config
	streamDecoder, err := NewStreamDecoder(streamReader)
	if err != nil {
		t.Fatalf("NewStreamDecoder() failed: %v", err)
	}

	if err := streamDecoder.Decode(&decodedCfg); err != nil {
		t.Fatalf("streamDecoder.Decode() failed: %v", err)
	}

	// 4. Verify that the decoded data is identical to the original data.
	if !reflect.DeepEqual(testData, decodedCfg) {
		t.Errorf("StreamDecoder round-trip failed.\nGot:  %+v\nWant: %+v", decodedCfg, testData)
	}
}

// TestStreamDecoderFile 确保 StreamDecoder 可以正确处理文件流.
func TestStreamDecoderFile(t *testing.T) {
	// Define a simple struct for this test.
	type FileConfig struct {
		Host string `wanf:"host"`
		Port int    `wanf:"port"`
	}

	wanfContent := `
		host = "localhost"
		port = 8080
	`
	// Create a temporary test file.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "config.wanf")
	if err := os.WriteFile(filePath, []byte(wanfContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// Open the file to get an io.Reader stream.
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open temp config file: %v", err)
	}
	defer f.Close()

	// Decode from the file stream.
	var cfg FileConfig
	decoder, err := NewStreamDecoder(f)
	if err != nil {
		t.Fatalf("NewStreamDecoder() with file failed: %v", err)
	}
	if err := decoder.Decode(&cfg); err != nil {
		t.Fatalf("decoder.Decode() with file failed: %v", err)
	}

	// Verify the contents.
	expected := FileConfig{Host: "localhost", Port: 8080}
	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("File stream decode failed.\nGot:  %+v\nWant: %+v", cfg, expected)
	}
}
