package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/darthlynx/kafka-exporter/internal/config"
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

	conf, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	slog.Info("Starting exporter", "source_topic", conf.SourceTopic, "destination_topic", conf.DestinationTopic)
	if err := exporter.Run(ctx, conf); err != nil {
		slog.Error("Exporter error", "err", err)
		os.Exit(1)
	}

	slog.Info("Shutdown complete")
}
