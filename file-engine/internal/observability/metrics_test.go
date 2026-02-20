package observability

import (
	"strings"
	"testing"
)

func TestSnapshotPrometheusIncludesQueueTaskAndOperabilityMetrics(t *testing.T) {
	m := NewMetrics()
	m.SetQueueDepth(9)
	m.IncEnqueued()
	m.ObserveStatus("running")
	m.ObserveStatus("success")
	m.ObserveStatus("failed")
	m.IncAuditEventEmitted()
	m.IncAuditSinkFailure()
	m.IncAuditDeadLetter()
	m.ObserveAuthzDecision(true, "ok")
	m.ObserveAuthzDecision(false, "tenant_membership")
	m.ObserveHTTPRequest("post", "/v1/objects:upload", 201, 42)
	m.ObserveGRPCRequest("/fileengine.FileEngine/ListObjects", "permission_denied", 7)
	m.ObserveQueueLagMs(55)
	m.SetAuditSinkLagMs(3)
	m.ObserveUploadDurationMs(77)
	m.SetScanBacklog(4)
	m.SetScanDLQSize(2)
	m.ObserveQuarantineTimeMs(1500)

	snapshot := m.SnapshotPrometheus()
	assertContains := func(expected string) {
		t.Helper()
		if !strings.Contains(snapshot, expected) {
			t.Fatalf("expected snapshot to include %q; snapshot=%s", expected, snapshot)
		}
	}

	assertContains("fileengine_queue_depth 9")
	assertContains("fileengine_tasks_enqueued_total 1")
	assertContains("fileengine_tasks_running_total 1")
	assertContains("fileengine_tasks_succeeded_total 1")
	assertContains("fileengine_tasks_failed_total 1")
	assertContains("fileengine_task_status_transitions_total{status=\"failed\"} 1")
	assertContains("fileengine_audit_events_total 1")
	assertContains("fileengine_audit_sink_failures_total 1")
	assertContains("fileengine_audit_dead_letters_total 1")
	assertContains("fileengine_authz_decisions_total{decision=\"allow\",reason=\"ok\"} 1")
	assertContains("fileengine_authz_decisions_total{decision=\"deny\",reason=\"tenant_membership\"} 1")
	assertContains("fileengine_http_requests_total{method=\"POST\",route=\"/v1/objects:upload\",status=\"201\"} 1")
	assertContains("fileengine_grpc_requests_total{method=\"/fileengine.FileEngine/ListObjects\",code=\"permission_denied\"} 1")
	assertContains("fileengine_queue_lag_ms_sum 55")
	assertContains("fileengine_audit_sink_lag_ms 3")
	assertContains("fileengine_upload_duration_ms_max 77")
	assertContains("fileengine_scan_backlog 4")
	assertContains("fileengine_scan_dlq_size 2")
	assertContains("fileengine_quarantine_time_ms_max 1500")
}
