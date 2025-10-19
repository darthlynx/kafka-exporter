package tlsconfig

import (
	"testing"

	"github.com/darthlynx/kafka-exporter/internal/testutil"
)

func TestNew_Success(t *testing.T) {
	ca, cert, key := testutil.CreateTestCertsTemp(t)

	// expect no error and valid tls.Config
	tlsCfg, err := New(ca, cert, key, false)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if tlsCfg == nil {
		t.Fatalf("expected non-nil tls.Config")
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Fatalf("expected loaded certificate in tls.Config")
	}
	if tlsCfg.RootCAs == nil {
		t.Fatalf("expected RootCAs to be set")
	}
	if tlsCfg.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=false")
	}
}

func TestNew_InsecureTrue(t *testing.T) {
	ca, cert, key := testutil.CreateTestCertsTemp(t)

	tlsCfg, err := New(ca, cert, key, true)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if tlsCfg == nil {
		t.Fatalf("expected non-nil tls.Config")
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=true")
	}
}

func TestNew_InvalidFiles(t *testing.T) {
	_, err := New("/does/not/exist/ca.crt", "/does/not/exist/client.crt", "/does/not/exist/client.key", false)
	if err == nil {
		t.Fatalf("expected error for invalid files")
	}
}
