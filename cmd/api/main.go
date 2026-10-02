// Command api serves the OpsGrid HTTP API, the realtime gateway and the embedded web UI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Tony5897/opsgrid/internal/app"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.RunAPI(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		return 1
	}
	return 0
}
