package security

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/example/file-engine/internal/app/ports"
)

func TestBuildMalwareScannerFromEnvDefaultsToStub(t *testing.T) {
	t.Setenv(envScannerType, "")
	s := BuildMalwareScannerFromEnv()
	if _, ok := s.(*MalwareScannerStub); !ok {
		t.Fatalf("expected stub scanner by default")
	}
}

func TestClamAVScannerReturnsCleanAndInfected(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				req := string(buf[:n])
				if strings.Contains(req, "eicar") {
					_, _ = c.Write([]byte("/quarantine/acme/eicar.txt: Eicar-Test-Signature FOUND\x00"))
					return
				}
				_, _ = c.Write([]byte("/quarantine/acme/clean.txt: OK\x00"))
			}(conn)
		}
	}()

	scanner := NewClamAVScanner(ln.Addr().String(), 500*time.Millisecond)
	clean, err := scanner.Scan(context.Background(), "/quarantine/acme/clean.txt")
	if err != nil {
		t.Fatalf("clean scan error: %v", err)
	}
	if clean.Status != ports.MalwareStatusClean {
		t.Fatalf("expected clean status, got %+v", clean)
	}

	dirty, err := scanner.Scan(context.Background(), "/quarantine/acme/eicar.txt")
	if err != nil {
		t.Fatalf("dirty scan error: %v", err)
	}
	if dirty.Status != ports.MalwareStatusInfected {
		t.Fatalf("expected infected status, got %+v", dirty)
	}
}

func TestBuildMalwareScannerFromEnvCreatesClamAV(t *testing.T) {
	t.Setenv(envScannerType, "clamav")
	t.Setenv(envScannerAddress, "127.0.0.1:3310")
	t.Setenv(envScannerTimeoutMs, "2500")
	s := BuildMalwareScannerFromEnv()
	c, ok := s.(*ClamAVScanner)
	if !ok {
		t.Fatalf("expected clamav scanner")
	}
	if c.address != "127.0.0.1:3310" {
		t.Fatalf("unexpected address: %s", c.address)
	}
	if c.timeout != 2500*time.Millisecond {
		t.Fatalf("unexpected timeout: %v", c.timeout)
	}
}
