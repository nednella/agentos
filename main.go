package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nednella/agentos/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	if err := cli.Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "agentos:", err)
		os.Exit(1)
	}
}
