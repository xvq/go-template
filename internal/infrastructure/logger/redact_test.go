package logger

import (
	"strings"
	"testing"
)

func TestSafeMessage(t *testing.T) {
	for _, input := range []string{
		"request https://user:secret@example.com/path?key=secret failed",
		"connect user:secret@tcp(localhost:3306)/db failed",
		"Authorization: Bearer secret",
		"password=secret api_key=secret",
		`Error 1062: Duplicate entry 'secret' for key 'users.name'`,
		`payload={"password":"secret"}`,
	} {
		if out := SafeMessage(input); strings.Contains(out, "secret") {
			t.Fatalf("leaked: %s", out)
		}
	}
	if got := SafeMessage("connection refused"); got != "connection refused" {
		t.Fatal(got)
	}
	if got := SafeMessage("custom private-value error", "private-value"); strings.Contains(got, "private-value") {
		t.Fatal(got)
	}
	if len([]rune(SafeMessage(strings.Repeat("字", 5000)))) > 4200 {
		t.Fatal("unbounded error text")
	}
}
