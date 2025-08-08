package wanf

// OutputStyle defines the different formatting styles for the output.
type OutputStyle int

const (
	// StyleBlockSorted is the default style. It sorts fields within nested blocks
	// but preserves the top-level order from the struct definition.
	// It uses indentation and empty lines for readability.
	StyleBlockSorted OutputStyle = iota

	// StyleAllSorted sorts all fields at all levels alphabetically,
	// ensuring a canonical, deterministic output. KVs are placed before blocks.
	StyleAllSorted

	// StyleStreaming outputs fields in the exact order they are defined in the struct,
	// without any sorting. It's fast but not deterministic if map keys change order.
	StyleStreaming

	// StyleSingleLine outputs the entire configuration on a single line,
	// using semicolons as separators. This is the most compact format.
	StyleSingleLine
)

const (
	// StyleDefault is an alias for StyleBlockSorted for backward compatibility.
	StyleDefault = StyleBlockSorted
)

// FormatOptions provides options for controlling the formatter's output.
type FormatOptions struct {
	Style      OutputStyle
	EmptyLines bool // If true, adds empty lines between blocks in supported styles.
}
