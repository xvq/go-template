package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/xvq/go-template/internal/infrastructure/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cli.NewRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
