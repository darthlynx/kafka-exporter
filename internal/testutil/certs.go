package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// CreateTestCerts creates a CA and a client cert/key under dir and returns the file paths.
// Caller may pass t.TempDir() or any writable directory.
func CreateTestCerts(t *testing.T, dir string) (caFile, certFile, keyFile string) {
	t.Helper()

	// CA key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	// CA cert template
	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	caPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caPath, caPEM, 0644); err != nil {
		t.Fatalf("failed to write CA cert: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ca.key"), caKeyPEM, 0600); err != nil {
		t.Fatalf("failed to write CA key: %v", err)
	}

	// Client key
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}

	clientTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "kafka-exporter",
		},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		SubjectKeyId: []byte{1, 2, 3, 4},
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTmpl, caTmpl, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client cert: %v", err)
	}

	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

	clientCertPath := filepath.Join(dir, "client.crt")
	clientKeyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(clientCertPath, clientCertPEM, 0644); err != nil {
		t.Fatalf("failed to write client cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, clientKeyPEM, 0600); err != nil {
		t.Fatalf("failed to write client key: %v", err)
	}

	return caPath, clientCertPath, clientKeyPath
}

// CreateTestCertsTemp creates a temp dir and writes test certs there, returning file paths.
func CreateTestCertsTemp(t *testing.T) (caFile, certFile, keyFile string) {
	t.Helper()
	dir := t.TempDir()
	return CreateTestCerts(t, dir)
}
