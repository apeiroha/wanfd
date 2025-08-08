package wanf

import "strings"

// wanfTag holds the parsed information from a `wanf` struct tag.
type wanfTag struct {
	Name      string
	KeyField  string
	Omitempty bool
}

// parseWanfTag parses a raw struct tag string into a wanfTag struct.
// If the tag string is empty, it uses the provided fieldName as the default.
func parseWanfTag(tagStr, fieldName string) wanfTag {
	if tagStr == "" {
		return wanfTag{Name: fieldName}
	}
	parts := strings.Split(tagStr, ",")
	tag := wanfTag{Name: parts[0]}
	if tag.Name == "" {
		tag.Name = fieldName
	}
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "key=") {
			tag.KeyField = strings.TrimPrefix(part, "key=")
		} else if part == "omitempty" {
			tag.Omitempty = true
		}
	}
	return tag
}
