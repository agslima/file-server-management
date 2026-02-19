package security

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/file-engine/internal/app/ports"
)

const (
	defaultScannerType   = "stub"
	defaultClamAVAddress = "127.0.0.1:3310"
	defaultClamAVTimeout = 3 * time.Second
	envScannerType       = "MALWARE_SCANNER_TYPE"
	envScannerAddress    = "MALWARE_SCANNER_ADDR"
	envScannerTimeoutMs  = "MALWARE_SCANNER_TIMEOUT_MS"
)

type ClamAVScanner struct {
	address string
	timeout time.Duration
}

func NewClamAVScanner(address string, timeout time.Duration) *ClamAVScanner {
	addr := strings.TrimSpace(address)
	if addr == "" {
		addr = defaultClamAVAddress
	}
	if timeout <= 0 {
		timeout = defaultClamAVTimeout
	}
	return &ClamAVScanner{address: addr, timeout: timeout}
}

func BuildMalwareScannerFromEnv() ports.MalwareScanner {
	scannerType := strings.ToLower(strings.TrimSpace(os.Getenv(envScannerType)))
	if scannerType == "" {
		scannerType = defaultScannerType
	}
	if scannerType == "clamav" {
		timeout := defaultClamAVTimeout
		if raw := strings.TrimSpace(os.Getenv(envScannerTimeoutMs)); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
				timeout = time.Duration(parsed) * time.Millisecond
			}
		}
		return NewClamAVScanner(os.Getenv(envScannerAddress), timeout)
	}
	return NewMalwareScannerStub()
}

func (s *ClamAVScanner) Scan(ctx context.Context, stagedPath string) (ports.MalwareScanResult, error) {
	d := net.Dialer{Timeout: s.timeout}
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return ports.MalwareScanResult{}, fmt.Errorf("clamav dial: %w", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))

	cmd := fmt.Sprintf("zSCAN %s\x00", stagedPath)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return ports.MalwareScanResult{}, fmt.Errorf("clamav write: %w", err)
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\x00')
	if err != nil && strings.TrimSpace(line) == "" {
		return ports.MalwareScanResult{}, fmt.Errorf("clamav read: %w", err)
	}
	response := strings.TrimSpace(strings.TrimSuffix(line, "\x00"))
	if strings.HasSuffix(response, "OK") {
		return ports.MalwareScanResult{Status: ports.MalwareStatusClean, Engine: "clamav", Detail: response}, nil
	}
	if strings.Contains(response, "FOUND") {
		return ports.MalwareScanResult{Status: ports.MalwareStatusInfected, Engine: "clamav", Detail: response}, nil
	}
	return ports.MalwareScanResult{Status: ports.MalwareStatusUnknown, Engine: "clamav", Detail: response}, nil
}
