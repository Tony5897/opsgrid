package httpx_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Tony5897/opsgrid/internal/platform/httpx"
)

// Shutdown must let in-flight requests finish while refusing new ones.
func TestServeDrainsInFlightRequests(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		if r.Context().Err() != nil {
			t.Error("request context cancelled during graceful drain")
		}
		_, _ = io.WriteString(w, "done")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv := httpx.NewServer(ctx, httpx.ServerOptions{Handler: h, ReadHeaderTimeout: time.Second, Logger: discard})
	served := make(chan error, 1)
	go func() { served <- httpx.ServeListener(ctx, srv, ln, 5*time.Second, discard) }()

	url := "http://" + ln.Addr().String() + "/"
	resp := make(chan string, 1)
	go func() {
		r, err := http.Get(url)
		if err != nil {
			resp <- "error: " + err.Error()
			return
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		resp <- string(b)
	}()

	<-started
	cancel() // SIGTERM equivalent

	// New connections are refused once shutdown begins.
	deadline := time.Now().Add(2 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", ln.Addr().String(), 100*time.Millisecond)
		if err != nil {
			break
		}
		_ = c.Close()
		if time.Now().After(deadline) {
			t.Fatal("listener still accepting after shutdown began")
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(release)
	if got := <-resp; got != "done" {
		t.Fatalf("in-flight response = %q, want done", got)
	}
	if err := <-served; err != nil {
		t.Fatalf("serve returned %v", err)
	}
}
