package exporter

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/darthlynx/kafka-exporter/internal/tlsconfig"
	"github.com/kelseyhightower/envconfig"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers          []string
	SourceTopic      string
	DestinationTopic string
	GroupID          string
	TLSConfig        *tls.Config
}

// LoadConfigFromEnv loads exporter configuration from environment variables.
func LoadConfigFromEnv() (Config, error) {
	var spec struct {
		Brokers            []string `envconfig:"KAFKA_BROKERS" default:"localhost:9093"`
		SourceTopic        string   `envconfig:"SOURCE_TOPIC" default:"source-topic"`
		DestinationTopic   string   `envconfig:"DESTINATION_TOPIC" default:"destination-topic"`
		GroupID            string   `envconfig:"GROUP_ID" default:"exporter-group"`
		CAFile             string   `envconfig:"TLS_CA_FILE" default:"certs/ca.crt"`
		CertFile           string   `envconfig:"TLS_CERT_FILE" default:"certs/client.crt"`
		KeyFile            string   `envconfig:"TLS_KEY_FILE" default:"certs/client.key"`
		InsecureSkipVerify bool     `envconfig:"TLS_INSECURE_SKIP_VERIFY" default:"true"`
	}

	if err := envconfig.Process("", &spec); err != nil {
		return Config{}, err
	}

	tlsCfg, err := tlsconfig.New(spec.CAFile, spec.CertFile, spec.KeyFile, spec.InsecureSkipVerify)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Brokers:          spec.Brokers,
		SourceTopic:      spec.SourceTopic,
		DestinationTopic: spec.DestinationTopic,
		GroupID:          spec.GroupID,
		TLSConfig:        tlsCfg,
	}, nil
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
