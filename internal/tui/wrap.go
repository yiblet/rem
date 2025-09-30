package tui

import (
	"strings"
	"unicode"
)

// WrapText wraps text to fit within a given width, breaking on word boundaries when possible.
// It handles newlines in the input and returns a slice of lines that fit within maxWidth.
// Height truncation is handled by the caller during rendering, not here.
func WrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{}
	}

	var result []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			result = append(result, "")
			continue
		}

		// If line fits, keep it as is
		if len(line) <= maxWidth {
			result = append(result, line)
			continue
		}

		// Line is too long, need to wrap
		wrapped := wrapLine(line, maxWidth)
		result = append(result, wrapped...)
	}

	return result
}

// wrapLine wraps a single line that is too long, breaking on word boundaries when possible
func wrapLine(line string, maxWidth int) []string {
	var result []string
	var currentLine strings.Builder
	currentWidth := 0

	words := splitWords(line)

	for i, word := range words {
		wordLen := len(word)

		// If word itself is longer than maxWidth, break it forcefully
		if wordLen > maxWidth {
			// Flush current line if it has content
			if currentWidth > 0 {
				result = append(result, currentLine.String())
				currentLine.Reset()
				currentWidth = 0
			}

			// Break the long word into chunks
			for len(word) > 0 {
				chunkSize := maxWidth
				if chunkSize > len(word) {
					chunkSize = len(word)
				}
				result = append(result, word[:chunkSize])
				word = word[chunkSize:]
			}
			continue
		}

		// Check if adding this word would exceed width
		spaceNeeded := wordLen
		if currentWidth > 0 {
			spaceNeeded++ // for the space before the word
		}

		if currentWidth+spaceNeeded > maxWidth {
			// Start new line
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
			currentWidth = wordLen
		} else {
			// Add to current line
			if currentWidth > 0 && i > 0 {
				currentLine.WriteString(" ")
				currentWidth++
			}
			currentLine.WriteString(word)
			currentWidth += wordLen
		}
	}

	// Add any remaining content
	if currentWidth > 0 {
		result = append(result, currentLine.String())
	}

	return result
}

// splitWords splits text into words, preserving spaces as part of word boundaries
func splitWords(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}
