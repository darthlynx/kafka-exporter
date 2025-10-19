package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// New builds a TLS config using provided CA + client cert/key.
// InsecureSkipVerify is enabled for local development; make configurable if needed.
func New(caFile, certFile, keyFile string) (*tls.Config, error) {
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
		InsecureSkipVerify: true,
	}, nil
}
