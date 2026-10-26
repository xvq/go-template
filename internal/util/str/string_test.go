package str

import "testing"

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		value string
		limit int
		want  string
	}{
		{"hello", 3, "hel"},
		{"你好世界", 2, "你好"},
		{"A😀中B", 3, "A😀中"},
		{"你好", 2, "你好"},
		{"你好", 10, "你好"},
		{"", 3, ""},
		{"hello", 0, ""},
		{"hello", -1, ""},
	} {
		if got := Truncate(tc.value, tc.limit); got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.value, tc.limit, got, tc.want)
		}
	}
}
