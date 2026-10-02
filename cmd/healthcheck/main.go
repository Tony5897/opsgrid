// Command healthcheck probes a local HTTP endpoint and exits 0 on 2xx.
// Distroless images have no shell or curl, so container health checks use
// this binary:
//
//	healthcheck http://127.0.0.1:9090/readyz
//
// Only loopback targets are accepted, so the binary cannot be repurposed to
// reach other hosts.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() { os.Exit(run()) }

func run() int {
	raw := "http://127.0.0.1:9090/livez"
	if len(os.Args) > 1 {
		raw = os.Args[1]
	}
	target, err := loopbackURL(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil) //nolint:gosec // G704: loopback-validated
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	client := &http.Client{
		// Never follow redirects off the loopback interface.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req) //nolint:gosec // G704: target validated as loopback by loopbackURL
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintln(os.Stderr, "unhealthy:", resp.Status)
		return 1
	}
	return 0
}

func loopbackURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("healthcheck: target must be an http(s) URL")
	}
	host := u.Hostname()
	if host == "localhost" {
		return u, nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return u, nil
	}
	return nil, errors.New("healthcheck: only loopback targets are allowed")
}
