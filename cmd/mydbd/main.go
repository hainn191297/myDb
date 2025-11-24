package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hainn191297/myDb/internal/config"
	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/server"
)

func main() {
	cfg := config.Load()

	logging.Infof("myDb booting with data dir %s", cfg.DataDir)
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
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
			logging.Warnf("received shutdown signal")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}
