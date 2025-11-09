package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/hainn191297/myDb/internal/config"
    "github.com/hainn191297/myDb/internal/server"
)

func main() {
    cfg := config.Load()

    srv := server.New(cfg)
    ctx, cancel := signalContext()
    defer cancel()

    if err := srv.Start(ctx); err != nil {
        log.Fatalf("server exited with error: %v", err)
    }
}

func signalContext() (context.Context, context.CancelFunc) {
    ctx, cancel := context.WithCancel(context.Background())
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        select {
        case <-sigCh:
            cancel()
        case <-ctx.Done():
        }
    }()

    return ctx, cancel
}
