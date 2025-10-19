package kafka

import (
	"context"
	"time"

	"github.com/darthlynx/kafka-exporter/internal/config"
	"github.com/segmentio/kafka-go"
)

// Message is a kafka-agnostic message used by the exporter.
// CommitFunc() will commit the underlying offset when supported by the client.
type Message struct {
	Key   []byte
	Value []byte
	Time  time.Time

	CommitFunc func(context.Context) error
}

// Commit commits the message offset (does nothing if not supported).
func (m Message) Commit(ctx context.Context) error {
	if m.CommitFunc == nil {
		return nil
	}
	return m.CommitFunc(ctx)
}

// Consumer is the Kafka Consumer interface used by the exporter.
type Consumer interface {
	FetchMessage(ctx context.Context) (Message, error)
	Close() error
}

// Producer is the Kafka Producer interface used by the exporter.
type Producer interface {
	WriteMessages(ctx context.Context, msgs ...Message) error
	Close() error
}

type kafkaConsumer struct {
	r *kafka.Reader
}

func (c *kafkaConsumer) FetchMessage(ctx context.Context) (Message, error) {
	kmsg, err := c.r.FetchMessage(ctx)
	if err != nil {
		return Message{}, err
	}

	msg := Message{
		Key:   kmsg.Key,
		Value: kmsg.Value,
		Time:  kmsg.Time,
		CommitFunc: func(ctx context.Context) error {
			// commit using the original kafka message
			return c.r.CommitMessages(ctx, kmsg)
		},
	}
	return msg, nil
}

func (c *kafkaConsumer) Close() error { return c.r.Close() }

type kafkaProducer struct {
	w *kafka.Writer
}

func (p *kafkaProducer) WriteMessages(ctx context.Context, msgs ...Message) error {
	kmsgs := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		kmsgs = append(kmsgs, kafka.Message{
			Key:   m.Key,
			Value: m.Value,
			Time:  m.Time,
		})
	}
	return p.w.WriteMessages(ctx, kmsgs...)
}

func (p *kafkaProducer) Close() error { return p.w.Close() }

// NewClients constructs a Consumer and Producer from the provided config.
// Returned clients must be Closed by the caller.
func NewClients(cfg config.Config) (Consumer, Producer) {
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

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.DestinationTopic,
		Balancer:     &kafka.Hash{},
		Transport:    &kafka.Transport{TLS: cfg.TLSConfig},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	return &kafkaConsumer{r: reader}, &kafkaProducer{w: writer}
}
