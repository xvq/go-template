package logger

import (
	"regexp"
	"strings"
)

var diagnosticRedactions = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:https?|redis|rediss)://[^\s"'<>]+`),
	regexp.MustCompile(`[^\s:]+:[^\s]*@(?:tcp|unix)\([^)]*\)[^\s]*`),
	regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[^\s,;]+`),
	regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'`),
	regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|token|access_token|refresh_token|api[_-]?key|secret|authorization)\s*[:=]\s*[^\s,;]+`),
	regexp.MustCompile(`\b[0-9a-fA-F]{64}\b`),
}

// SafeMessage redacts common credential formats and quoted values (including
// values embedded in database errors). It is not a general-purpose DLP filter:
// callers must never embed request bodies, SQL, or credentials in error text.
func SafeMessage(message string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[REDACTED]")
		}
	}
	for _, pattern := range diagnosticRedactions {
		message = pattern.ReplaceAllString(message, "[REDACTED]")
	}
	const limit = 4096
	runes := []rune(message)
	if len(runes) > limit {
		message = string(runes[:limit]) + "…[truncated]"
	}
	return message
}
