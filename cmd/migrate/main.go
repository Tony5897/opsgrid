// Command migrate applies the embedded database migrations as the schema
// owner role.
//
//	migrate [up|status]   (default: up)
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
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if err := app.RunMigrate(ctx, command); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		return 1
	}
	return 0
}
