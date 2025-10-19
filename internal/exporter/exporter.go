package exporter

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers          []string
	SourceTopic      string
	DestinationTopic string
	GroupID          string
	TLSConfig        *tls.Config
}

// Run runs the exporter: consumes from SourceTopic and writes to DestinationTopic.
func Run(ctx context.Context, config Config) error {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		TLS:       config.TLSConfig,
		DualStack: true,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           config.Brokers,
		GroupID:           config.GroupID,
		Topic:             config.SourceTopic,
		Dialer:            dialer,
		MinBytes:          1,
		MaxBytes:          10e6,
		StartOffset:       kafka.FirstOffset,
		CommitInterval:    0,
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("Failed to close reader: %v", err)
		}
	}()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.DestinationTopic,
		Balancer:     &kafka.Hash{},
		Transport:    &kafka.Transport{TLS: config.TLSConfig},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Failed to close writer: %v", err)
		}
	}()

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

		if err = writer.WriteMessages(ctx, kafka.Message{
			Key:   msg.Key,
			Value: msg.Value,
			Time:  msg.Time,
		}); err != nil {
			log.Printf("Failed to write message: %v", err)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Failed to commit offset: %v", err)
		}
	}
}
