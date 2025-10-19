package exporter

import (
	"context"
	"time"

	"log/slog"

	"github.com/darthlynx/kafka-exporter/internal/config"
	"github.com/segmentio/kafka-go"
)

// Run runs the exporter: consumes from SourceTopic and writes to DestinationTopic.
func Run(ctx context.Context, cfg config.Config) error {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		TLS:       cfg.TLSConfig,
		DualStack: true,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		GroupID:           cfg.GroupID,
		Topic:             cfg.SourceTopic,
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
			slog.Error("Failed to close reader", "err", err)
		}
	}()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.DestinationTopic,
		Balancer:     &kafka.Hash{},
		Transport:    &kafka.Transport{TLS: cfg.TLSConfig},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			slog.Error("Failed to close writer", "err", err)
		}
	}()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("Consumer loop stopped")
				return nil
			}
			slog.Error("Failed to fetch message", "err", err)
			continue
		}

		slog.Info("Received message", "key", string(msg.Key), "value", string(msg.Value))

		if err = writer.WriteMessages(ctx, kafka.Message{
			Key:   msg.Key,
			Value: msg.Value,
			Time:  msg.Time,
		}); err != nil {
			slog.Error("Failed to write message", "err", err)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Failed to commit offset", "err", err)
		}
	}
}
