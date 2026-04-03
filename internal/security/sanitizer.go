package security

import (
	"html"
	"regexp"
)

// MaxOutputLength is the maximum number of characters allowed in sanitized output.
const MaxOutputLength = 4096

// botCommandRe matches the known bot commands at any position in the string.
var botCommandRe = regexp.MustCompile(`/(start|quiz|resume|stats|packs|config|help)\b`)

// Sanitize cleans LLM output for safe display.
// It strips bot command patterns, escapes HTML special characters (for
// Telegram HTML mode), and truncates the result to MaxOutputLength.
func Sanitize(input string) string {
	// Strip bot commands
	result := botCommandRe.ReplaceAllString(input, "")

	// Escape HTML special characters
	result = html.EscapeString(result)

	// Truncate to MaxOutputLength
	runes := []rune(result)
	if len(runes) > MaxOutputLength {
		runes = runes[:MaxOutputLength]
	}

	return string(runes)
}
