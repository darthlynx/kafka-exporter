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
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

func main() {
	// Settings
	brokers := []string{"localhost:9092"}
	sourceTopic := "source-topic"
	destTopic := "destination-topic"
	groupID := "my-group"

	// TLS setup
	// tlsConfig, err := createTLSConfig("ca.pem", "client.pem", "client.key")
	// if err != nil {
	// 	log.Fatalf("TLS config error: %v", err)
	// }

	// Kafka dialer with TLS
	// dialer := &kafka.Dialer{
	// 	Timeout:   10 * time.Second,
	// 	TLS:       tlsConfig,
	// 	DualStack: true,
	// }

	// Reader setup
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   sourceTopic,
		// Dialer:            dialer,
		MinBytes:          1,
		MaxBytes:          10e6,              // 10MB
		StartOffset:       kafka.FirstOffset, // earliest
		CommitInterval:    0,                 // manual commit
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
	})
	defer reader.Close()

	// Writer setup
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    destTopic,
		Balancer: &kafka.Hash{}, // partition by key
		// Transport:    &kafka.Transport{TLS: tlsConfig},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					log.Println("Consumer loop stopped")
					return
				}
				log.Printf("Failed to fetch message: %v", err)
				continue
			}

			// Log message info
			log.Printf("Received message: key=%s value=%s", string(msg.Key), string(msg.Value))

			// Write to destination topic
			err = writer.WriteMessages(ctx, kafka.Message{
				Key:   msg.Key,
				Value: msg.Value,
				Time:  msg.Time, // preserve timestamp
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
	}()

	// Wait for signal
	sig := <-sigchan
	log.Printf("Received signal: %v. Shutting down...", sig)
	cancel()

	// Let goroutine finish
	time.Sleep(2 * time.Second)
	log.Println("Graceful shutdown complete.")
}
