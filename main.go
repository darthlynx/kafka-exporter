package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func createTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert and key: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Skip hostname verification. For local development only
	}, nil
}

type Config struct {
	Brokers          []string
	SourceTopic      string
	DestinationTopic string
	GroupID          string
	TLSConfig        *tls.Config
}

func RunExporter(ctx context.Context, config Config) error {
	// Kafka dialer with TLS
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		TLS:       config.TLSConfig,
		DualStack: true,
	}

	// Consumer setup
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           config.Brokers,
		GroupID:           config.GroupID,
		Topic:             config.SourceTopic,
		Dialer:            dialer,
		MinBytes:          1,
		MaxBytes:          10e6,              // 10MB
		StartOffset:       kafka.FirstOffset, // earliest
		CommitInterval:    0,                 // manual commit
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
	})
	defer reader.Close()

	// Producer setup
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.DestinationTopic,
		Balancer:     &kafka.Hash{}, // partition by key
		Transport:    &kafka.Transport{TLS: config.TLSConfig},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer writer.Close()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Consumer loop stopped")
				return nil
			}
			log.Printf("Failed to fetch message: %v", err)
			continue
		}

		log.Printf("Received message: key=%s value=%s", string(msg.Key), string(msg.Value))

		// Write to destination topic
		err = writer.WriteMessages(ctx, kafka.Message{
			Key:   msg.Key,
			Value: msg.Value,
			Time:  msg.Time, // preserve timestamp (could be excluded)
		})
		if err != nil {
			log.Printf("Failed to write message: %v", err)
			continue
		}

		// Manually commit offset
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit offset: %v", err)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	// Wait for signal
	go func() {
		sig := <-sigchan
		log.Printf("Received signal: %v. Shutting down...", sig)
		cancel()
	}()

	// TLS setup
	tlsConfig, err := createTLSConfig("certs/ca.crt", "certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatalf("TLS config error: %v", err)
	}

	conf := Config{
		Brokers:          []string{"localhost:9093"}, // TLS is listened on the 9093 port in this setup
		SourceTopic:      "source-topic",
		DestinationTopic: "destination-topic",
		GroupID:          "my-group",
		TLSConfig:        tlsConfig,
	}

	if err := RunExporter(ctx, conf); err != nil {
		log.Fatalf("Exporter error: %v", err)
	}

	log.Println("Shutdown complete")
}
