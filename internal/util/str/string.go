// Package str provides string helpers without application dependencies.
package str

// Truncate returns at most n Unicode code points without adding an ellipsis.
// It returns an empty string when n <= 0. Code points are not grapheme clusters:
// combined characters and emoji sequences may contain several code points.
func Truncate(value string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > n {
		return string(runes[:n])
	}
	return value
}
