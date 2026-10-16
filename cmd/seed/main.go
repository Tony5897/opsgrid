// Command seed upserts deterministic demo tenants (Cascade / Northstar).
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
	if err := app.RunSeed(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		return 1
	}
	return 0
}
