package wanf

import (
	"os"
	"testing"
)

// Benchmark data - a reasonably complex wanf file content.
var benchmarkWanfData, _ = os.ReadFile("example.wanf")

// BenchmarkLexer measures the performance of tokenizing a wanf file.
func BenchmarkLexer(b *testing.B) {
	if benchmarkWanfData == nil {
		b.Skip("Cannot read benchmark data file")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := NewLexer(benchmarkWanfData)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkParser measures the performance of parsing a wanf file (lexing + parsing).
func BenchmarkParser(b *testing.B) {
	if benchmarkWanfData == nil {
		b.Skip("Cannot read benchmark data file")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := NewLexer(benchmarkWanfData)
		p := NewParser(l)
		p.ParseProgram()
	}
}

// BenchmarkFormat measures the end-to-end performance of linting and formatting.
func BenchmarkFormat(b *testing.B) {
	if benchmarkWanfData == nil {
		b.Skip("Cannot read benchmark data file")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program, _ := Lint(benchmarkWanfData)
		Format(program, FormatOptions{Style: StyleBlockSorted, EmptyLines: true})
	}
}

// BenchmarkDecode measures the performance of decoding a wanf file into a Go struct.
func BenchmarkDecode(b *testing.B) {
	if benchmarkWanfData == nil {
		b.Skip("Cannot read benchmark data file")
	}

	// Define a struct that matches the structure of the benchmark data file.
	type benchmarkConfig struct {
		Application struct {
			Name    string   `wanf:"name"`
			Version float64  `wanf:"version"`
			Debug   bool     `wanf:"debug_mode"`
			MaxJobs int      `wanf:"max_concurrent_jobs"`
			Allowed []string `wanf:"allowed_origins"`
		} `wanf:"application"`
		Logging struct {
			Level    string `wanf:"level"`
			Template string `wanf:"format_template"`
		} `wanf:"logging"`
		Server map[string]interface{} `wanf:"server"`
	}

	// Pre-populate the cache once before the benchmark loop, as it's a one-time cost.
	// This ensures we are benchmarking the cached performance.
	var cfg benchmarkConfig
	_ = Decode(benchmarkWanfData, &cfg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var cfg benchmarkConfig
		// The Decode function is what we are testing.
		// With the cache, this should be much faster on subsequent runs.
		_ = Decode(benchmarkWanfData, &cfg)
	}
}
