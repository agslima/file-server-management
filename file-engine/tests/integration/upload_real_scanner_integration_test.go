package integration

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
	"github.com/example/file-engine/internal/services"
)

func TestUploadRealScannerIntegrationEmitsMetricsAndLogs(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen scanner: %v", err)
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
				if strings.Contains(strings.ToLower(req), "eicar") {
					_, _ = c.Write([]byte("/quarantine/acme/eicar.txt: Eicar-Test-Signature FOUND\x00"))
					return
				}
				_, _ = c.Write([]byte("/quarantine/acme/clean.txt: OK\x00"))
			}(conn)
		}
	}()

	origMetrics := observability.DefaultMetrics
	observability.DefaultMetrics = observability.NewMetrics()
	t.Cleanup(func() { observability.DefaultMetrics = origMetrics })

	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stdout) })

	st := localstorage.New(t.TempDir())
	svc := services.NewUploadServiceWithLogger(
		st,
		adaptersecurity.NewClamAVScanner(ln.Addr().String(), time.Second),
		services.UploadPolicy{MaxObjectSizeBytes: 1024, TenantQuotaBytes: 10 * 1024, RequestTimeout: 2 * time.Second, RequireCleanScan: true},
		logger.New("debug"),
	)

	ctx := context.Background()
	if _, err := svc.UploadStream(ctx, "/tenants/acme/docs/clean.txt", bytes.NewReader([]byte("ok")), "clean-real"); err != nil {
		t.Fatalf("clean upload: %v", err)
	}
	if _, err := svc.UploadStream(ctx, "/tenants/acme/docs/eicar.txt", bytes.NewReader([]byte("bad")), "dirty-real"); err == nil {
		t.Fatalf("expected dirty upload to be quarantined")
	}

	snapshot := observability.DefaultMetrics.SnapshotPrometheus()
	if !strings.Contains(snapshot, "fileengine_malware_scan_duration_ms_count 2") {
		t.Fatalf("expected scan duration count metric, got %s", snapshot)
	}
	if !strings.Contains(snapshot, `fileengine_malware_scan_verdict_total{verdict="clean"} 1`) {
		t.Fatalf("expected clean verdict metric, got %s", snapshot)
	}
	if !strings.Contains(snapshot, `fileengine_malware_scan_verdict_total{verdict="infected"} 1`) {
		t.Fatalf("expected infected verdict metric, got %s", snapshot)
	}

	out := logs.String()
	if !strings.Contains(out, "upload.scan.completed") || !strings.Contains(out, `"scan_verdict":"clean"`) || !strings.Contains(out, `"scan_verdict":"infected"`) {
		t.Fatalf("expected scan logs for clean and infected verdicts, got %s", out)
	}
}
