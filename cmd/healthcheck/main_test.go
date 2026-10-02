package main

import "testing"

func TestLoopbackURL(t *testing.T) {
	t.Parallel()
	for raw, ok := range map[string]bool{
		"http://127.0.0.1:9090/readyz":   true,
		"http://localhost:9090/livez":    true,
		"http://[::1]:9090/readyz":       true,
		"http://10.0.0.5:9090/readyz":    false,
		"http://example.com/":            false,
		"http://169.254.169.254/latest/": false,
		"file:///etc/passwd":             false,
		"gopher://127.0.0.1/":            false,
	} {
		if _, err := loopbackURL(raw); (err == nil) != ok {
			t.Errorf("loopbackURL(%q) ok=%v, want %v", raw, err == nil, ok)
		}
	}
}
