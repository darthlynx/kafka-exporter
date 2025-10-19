package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/darthlynx/kafka-exporter/internal/exporter"
	"github.com/darthlynx/kafka-exporter/internal/tlsconfig"
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

	tlsCfg, err := tlsconfig.New("certs/ca.crt", "certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatalf("TLS config error: %v", err)
	}

	conf := exporter.Config{
		Brokers:          []string{"localhost:9093"},
		SourceTopic:      "source-topic",
		DestinationTopic: "destination-topic",
		GroupID:          "my-group",
		TLSConfig:        tlsCfg,
	}

	if err := exporter.Run(ctx, conf); err != nil {
		log.Fatalf("Exporter error: %v", err)
	}

	log.Println("Shutdown complete")
}
