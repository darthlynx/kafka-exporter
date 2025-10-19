package config

import (
	"testing"

	"github.com/darthlynx/kafka-exporter/internal/testutil"
)

func TestLoadFromEnv_Success(t *testing.T) {
	ca, cert, key := testutil.CreateTestCertsTemp(t)

	t.Setenv("TLS_CA_FILE", ca)
	t.Setenv("TLS_CERT_FILE", cert)
	t.Setenv("TLS_KEY_FILE", key)
	t.Setenv("KAFKA_BROKERS", "localhost:9093")
	t.Setenv("TLS_INSECURE_SKIP_VERIFY", "false")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if len(cfg.Brokers) == 0 {
		t.Fatalf("expected brokers to be set")
	}
	if cfg.SourceTopic == "" {
		t.Fatalf("expected source topic to be set")
	}
	if cfg.TLSConfig == nil {
		t.Fatalf("expected TLSConfig to be non-nil")
	}
	if cfg.TLSConfig.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=false")
	}
}

func TestLoadFromEnv_MissingTLSFiles(t *testing.T) {
	// Point to non-existent files to force tlsconfig failure
	t.Setenv("TLS_CA_FILE", "/does/not/exist/ca.crt")
	t.Setenv("TLS_CERT_FILE", "/does/not/exist/client.crt")
	t.Setenv("TLS_KEY_FILE", "/does/not/exist/client.key")
	t.Setenv("KAFKA_BROKERS", "localhost:9093")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatalf("expected LoadFromEnv to fail with invalid TLS files")
	}
}
