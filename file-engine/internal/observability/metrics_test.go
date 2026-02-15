package observability

import (
	"strings"
	"testing"
)

func TestSnapshotPrometheusIncludesQueueAndTaskCounters(t *testing.T) {
	m := NewMetrics()
	m.SetQueueDepth(9)
	m.IncEnqueued()
	m.ObserveStatus("running")
	m.ObserveStatus("success")
	m.ObserveStatus("failed")

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
}
