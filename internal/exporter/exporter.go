package exporter

import (
	"context"

	"log/slog"

	"github.com/darthlynx/kafka-exporter/internal/config"
	"github.com/darthlynx/kafka-exporter/internal/kafka"
)

// Exporter coordinates consuming and producing messages.
type Exporter struct {
	consumer kafka.Consumer
	producer kafka.Producer
	logger   *slog.Logger
}

// New creates an Exporter. If logger is nil, slog.Default() is used.
func New(consumer kafka.Consumer, producer kafka.Producer, logger *slog.Logger) *Exporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Exporter{
		consumer: consumer,
		producer: producer,
		logger:   logger,
	}
}

// Run executes the main processing loop until ctx is done.
func (e *Exporter) Run(ctx context.Context) error {
	for {
		msg, err := e.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				e.logger.Info("consumer loop stopped")
				return nil
			}
			e.logger.Error("failed to fetch message", "err", err)
			continue
		}

		e.logger.Info("received message", "key", string(msg.Key), "value", string(msg.Value))

		if err := e.producer.WriteMessages(ctx, msg); err != nil {
			e.logger.Error("failed to write message", "err", err)
			continue
		}

		if err := msg.Commit(ctx); err != nil {
			e.logger.Error("failed to commit offset", "err", err)
		}
	}
}

// RunWithConfig is a convenience helper: build kafka clients from config,
// run the exporter and close clients when finished.
func RunWithConfig(ctx context.Context, cfg config.Config) error {
	consumer, producer := kafka.NewClients(cfg)
	defer func() {
		if err := consumer.Close(); err != nil {
			slog.Error("Cannot close the consumer", "err", err)
		}
		if err := producer.Close(); err != nil {
			slog.Error("Cannot close the producer", "err", err)
		}
	}()

	exp := New(consumer, producer, nil)
	return exp.Run(ctx)
}
