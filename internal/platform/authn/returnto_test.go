package authn

import "testing"

func TestSanitizeReturnTo(t *testing.T) {
	t.Parallel()
	const origin = "http://localhost:5173"
	cases := []struct {
		in, want string
	}{
		{"", "/"},
		{"/dashboard", "/dashboard"},
		{"/wo/1?tab=notes", "/wo/1?tab=notes"},
		{"http://localhost:5173/app", "/app"},
		{"http://localhost:5173/app?x=1", "/app?x=1"},
		{"https://evil.example/phish", "/"},
		{"http://evil.example/", "/"},
		{"//evil.example", "/"},
		{"/\\evil.example", "/"},
		{"/\tevil", "/"},
		{"javascript:alert(1)", "/"},
		{"/ok://still-bad", "/"},
	}
	for _, tc := range cases {
		if got := sanitizeReturnTo(tc.in, origin); got != tc.want {
			t.Errorf("sanitizeReturnTo(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
