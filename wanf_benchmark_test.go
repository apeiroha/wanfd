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
		Format(program, FormatOptions{Style: StyleDefault, EmptyLines: true})
	}
}
