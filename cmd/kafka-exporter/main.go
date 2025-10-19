package main

import (
	"context"
	"log"
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
		log.Printf("Received signal: %v. Shutting down...", sig)
		cancel()
	}()

	conf, err := exporter.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if err := exporter.Run(ctx, conf); err != nil {
		log.Fatalf("Exporter error: %v", err)
	}

	log.Println("Shutdown complete")
}
