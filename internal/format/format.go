// Package format converts moment.js date tokens used by
// Obsidian daily notes into formatted strings.
package format

import (
	"fmt"
	"strings"
	"time"
)

// FormatMoment formats date with a moment-style pattern.
// Supported tokens (longest match first): YYYY, YY, MMMM,
// MMM, MM, M, dddd, ddd, DD, D. Literal text is preserved.
func FormatMoment(pattern string, date time.Time) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("format is empty")
	}
	var out strings.Builder
	out.Grow(len(pattern) + 8)
	i := 0
	for i < len(pattern) {
		if token, length := matchToken(pattern[i:]); token != "" {
			s, err := renderToken(token, date)
			if err != nil {
				return "", err
			}
			out.WriteString(s)
			i += length
			continue
		}
		r := rune(pattern[i])
		// Handle multi-byte UTF-8.
		if r >= 0x80 {
			for _, c := range pattern[i:] {
				out.WriteRune(c)
				i += len(string(c))
				break
			}
		} else {
			out.WriteByte(pattern[i])
			i++
		}
	}
	return out.String(), nil
}

var tokens = []string{
	"YYYY", "MMMM", "dddd", "ddd",
	"MMM", "YY", "MM", "DD", "M", "D",
}

func matchToken(rest string) (string, int) {
	for _, tok := range tokens {
		if strings.HasPrefix(rest, tok) {
			return tok, len(tok)
		}
	}
	return "", 0
}

func renderToken(
	token string, date time.Time,
) (string, error) {
	switch token {
	case "YYYY":
		return fmt.Sprintf("%04d", date.Year()), nil
	case "YY":
		return fmt.Sprintf("%02d", date.Year()%100), nil
	case "MMMM":
		return date.Month().String(), nil
	case "MMM":
		return date.Month().String()[:3], nil
	case "MM":
		return fmt.Sprintf("%02d", date.Month()), nil
	case "M":
		return fmt.Sprintf("%d", date.Month()), nil
	case "DD":
		return fmt.Sprintf("%02d", date.Day()), nil
	case "D":
		return fmt.Sprintf("%d", date.Day()), nil
	case "dddd":
		return date.Weekday().String(), nil
	case "ddd":
		return date.Weekday().String()[:3], nil
	default:
		return "", fmt.Errorf("unsupported token %s", token)
	}
}
