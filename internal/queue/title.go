package queue

import (
	"strings"
	"unicode"
)

// GenerateTitle creates a title from a content sample (first few KB).
// For binary content, returns "[binary content]".
// For text content, uses the first non-empty line or sanitized content.
func GenerateTitle(sample []byte, isBinary bool) string {
	if isBinary {
		return "[binary content]"
	}

	if len(sample) == 0 {
		return "[empty]"
	}

	// Convert to string and get first non-empty line
	text := string(sample)
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		cleaned := strings.TrimSpace(line)
		if cleaned != "" {
			// Found a non-empty line, sanitize and use it as title
			return SanitizeTitle(cleaned)
		}
	}

	// Fallback: use sanitized content (collapse whitespace)
	sanitized := SanitizeTitle(text)
	if sanitized == "" {
		return "[empty]"
	}

	return sanitized
}

// TruncateTitle ensures title is at most maxLen characters.
// If truncation is needed, appends "..." to indicate truncation.
func TruncateTitle(title string, maxLen int) string {
	title = strings.TrimSpace(title)

	if len(title) <= maxLen {
		return title
	}

	// Reserve 3 characters for "..."
	if maxLen < 3 {
		return strings.Repeat(".", maxLen)
	}

	return title[:maxLen-3] + "..."
}

// SanitizeTitle removes control characters and collapses whitespace.
// This ensures titles are safe for display in terminals and UIs.
func SanitizeTitle(title string) string {
	// Replace control characters (except tab, newline, carriage return) with spaces
	// Then collapse all whitespace into single spaces
	title = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			// Convert control characters to space for later collapsing
			return ' '
		}
		return r
	}, title)

	// Collapse all whitespace (spaces, tabs, newlines, etc.) into single spaces
	fields := strings.Fields(title)
	return strings.Join(fields, " ")
}
