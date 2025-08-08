package wanf

// OutputStyle defines the different formatting styles for the output.
type OutputStyle int

const (
	// StyleDefault is the default, block-sorted style.
	// It adds empty lines between blocks for readability.
	StyleDefault OutputStyle = iota
	// StyleStreaming mimics the input order as closely as possible.
	StyleStreaming
	// StyleSingleLine outputs the entire file on a single line.
	StyleSingleLine
)

// FormatOptions provides options for controlling the formatter's output.
type FormatOptions struct {
	Style      OutputStyle
	EmptyLines bool
}
