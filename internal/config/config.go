package config

import (
	"crypto/tls"
	"fmt"

	"github.com/darthlynx/kafka-exporter/internal/tlsconfig"
	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for the exporter.
type Config struct {
	Brokers          []string
	SourceTopic      string
	DestinationTopic string
	GroupID          string
	TLSConfig        *tls.Config
}

// LoadFromEnv loads configuration from environment variables.
//
// Environment variables:
//
//	KAFKA_BROKERS               (default: "localhost:9093", comma-separated list)
//	SOURCE_TOPIC                (default: "source-topic")
//	DESTINATION_TOPIC           (default: "destination-topic")
//	GROUP_ID                    (default: "exporter-group")
//	TLS_CA_FILE                 (default: "certs/ca.crt")
//	TLS_CERT_FILE               (default: "certs/client.crt")
//	TLS_KEY_FILE                (default: "certs/client.key")
//	TLS_INSECURE_SKIP_VERIFY    (default: true)
func LoadFromEnv() (Config, error) {
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
		return Config{}, fmt.Errorf("envconfig: %w", err)
	}

	tlsCfg, err := tlsconfig.New(spec.CAFile, spec.CertFile, spec.KeyFile, spec.InsecureSkipVerify)
	if err != nil {
		return Config{}, fmt.Errorf("tlsconfig: %w", err)
	}

	return Config{
		Brokers:          spec.Brokers,
		SourceTopic:      spec.SourceTopic,
		DestinationTopic: spec.DestinationTopic,
		GroupID:          spec.GroupID,
		TLSConfig:        tlsCfg,
	}, nil
}
