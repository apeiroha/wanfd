package wanf

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestEncoder_Styles(t *testing.T) {
	type SubBlock struct {
		B_sub_kv string `wanf:"b_sub_kv"`
		A_sub_kv string `wanf:"a_sub_kv"`
	}
	type Config struct {
		C_kv    string            `wanf:"c_kv"`
		A_block SubBlock          `wanf:"a_block"`
		B_kv    int               `wanf:"b_kv"`
		D_map   map[string]string `wanf:"d_map"`
	}

	testData := Config{
		C_kv:    "c",
		A_block: SubBlock{B_sub_kv: "b", A_sub_kv: "a"},
		B_kv:    123,
		D_map:   map[string]string{"z_key": "z", "y_key": "y"},
	}

	tests := []struct {
		name    string
		options []EncoderOption
		want    string
	}{
		{
			"Default (StyleDefault)",
			[]EncoderOption{},
			strings.Join(
				[]string{
					`c_kv = "c"`,
					``,
					`a_block {`,
					`	a_sub_kv = "a"`,
					`	b_sub_kv = "b"`,
					`}`,
					``,
					`b_kv = 123`,
					`d_map = {[`,
					`	y_key = "y",`,
					`	z_key = "z",`,
					`]}`,
					``,
				},
				"\n",
			),
		},
		{
			"StyleStreaming",
			[]EncoderOption{WithStyle(StyleStreaming)},
			"c_kv = \"c\"\na_block {\n\tb_sub_kv = \"b\"\n\ta_sub_kv = \"a\"\n}\nb_kv = 123\nd_map = {[\n\ty_key = \"y\",\n\tz_key = \"z\",\n]}\n",
		},
		{
			"StyleSingleLine",
			[]EncoderOption{WithStyle(StyleSingleLine)},
			"c_kv=\"c\";a_block{b_sub_kv=\"b\";a_sub_kv=\"a\"};b_kv=123;d_map={[y_key=\"y\",z_key=\"z\",]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf, tt.options...)
			if err := encoder.Encode(testData); err != nil {
				t.Fatalf("Encode() with style %v failed: %v", tt.name, err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("Encode() with style %v got:\n%q\nWant:\n%q", tt.name, got, tt.want)
			}

			var decodedCfg Config
			if err := Decode(buf.Bytes(), &decodedCfg); err != nil {
				t.Fatalf("Decode round-trip failed for style %v: %v\nGot:\n%s", tt.name, err, buf.String())
			}
			if !reflect.DeepEqual(testData, decodedCfg) {
				t.Errorf("Round-trip failed for style %v. Got %+v, want %+v", tt.name, decodedCfg, testData)
			}
		})
	}
}

func TestEncoder_EmptyMap(t *testing.T) {
	type ConfigWithMap struct {
		Name  string            `wanf:"name"`
		Attrs map[string]string `wanf:"attrs"`
	}
	cfg := ConfigWithMap{Name: "test", Attrs: make(map[string]string)}
	b, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal with empty map: %v", err)
	}
	if strings.Contains(string(b), "attrs") {
		t.Errorf("Encoder should omit empty map field. Got: %s", string(b))
	}
}

func TestDecode_CompactFormat(t *testing.T) {
	compactData := `enabled=true;server{port=8080};ids=[1,2,3]`
	var cfg struct {
		Enabled bool `wanf:"enabled"`
		Server  struct {
			Port int `wanf:"port"`
		} `wanf:"server"`
		IDs []interface{} `wanf:"ids"`
	}
	err := Decode([]byte(compactData), &cfg)
	if err != nil {
		t.Fatalf("Decode failed for compact format: %v", err)
	}
	if !(cfg.Enabled && cfg.Server.Port == 8080 && len(cfg.IDs) == 3) {
		t.Errorf("Compact format decode mismatch. Got: %+v", cfg)
	}
}

func TestFieldMatching_Fallback(t *testing.T) {
	type Config struct {
		TaggedField   string `wanf:"tagged_field"`
		UntaggedField int
		LogLevel      string // Test case-insensitivity
	}

	// Test encoding
	cfg := Config{
		TaggedField:   "value1",
		UntaggedField: 123,
		LogLevel:      "INFO",
	}
	b, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	encoded := string(b)
	if !strings.Contains(encoded, "tagged_field = \"value1\"") {
		t.Errorf("Expected tagged field to be encoded")
	}
	if !strings.Contains(encoded, "UntaggedField = 123") {
		t.Errorf("Expected untagged field to be encoded with its field name")
	}
	if !strings.Contains(encoded, "LogLevel = \"INFO\"") {
		t.Errorf("Expected untagged field to be encoded with its field name")
	}

	// Test decoding
	wanfData := `
		tagged_field = "new_value"
		untaggedfield = 456
		loglevel = "DEBUG"
	`
	var decodedCfg Config
	err = Decode([]byte(wanfData), &decodedCfg)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	expected := Config{
		TaggedField:   "new_value",
		UntaggedField: 456,
		LogLevel:      "DEBUG",
	}
	if !reflect.DeepEqual(decodedCfg, expected) {
		t.Errorf("Fallback decoding failed. Got %+v, want %+v", decodedCfg, expected)
	}
}
