package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/darthlynx/kafka-exporter/internal/exporter"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigchan
		slog.Info("Received signal. Shutting down...", "signal", sig)
		cancel()
	}()

	slog.Info("Starting the app")
	conf, err := exporter.LoadConfigFromEnv()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	if err := exporter.Run(ctx, conf); err != nil {
		slog.Error("Exporter error", "err", err)
		os.Exit(1)
	}

	slog.Info("Shutdown complete")
}
