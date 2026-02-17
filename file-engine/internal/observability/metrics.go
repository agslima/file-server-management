package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type counterVec struct {
	values sync.Map
}

func (v *counterVec) inc(key string) {
	val, _ := v.values.LoadOrStore(key, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}

func (v *counterVec) snapshot() map[string]int64 {
	out := map[string]int64{}
	v.values.Range(func(k, val any) bool {
		key, _ := k.(string)
		ctr, _ := val.(*atomic.Int64)
		out[key] = ctr.Load()
		return true
	})
	return out
}

type durationSummary struct {
	count atomic.Int64
	sumMs atomic.Int64
	maxMs atomic.Int64
}

func (s *durationSummary) observe(ms int64) {
	s.count.Add(1)
	s.sumMs.Add(ms)
	for {
		cur := s.maxMs.Load()
		if ms <= cur {
			return
		}
		if s.maxMs.CompareAndSwap(cur, ms) {
			return
		}
	}
}

func (s *durationSummary) snapshot() (count, sumMs, maxMs int64) {
	return s.count.Load(), s.sumMs.Load(), s.maxMs.Load()
}

type Metrics struct {
	queueDepth          atomic.Int64
	tasksEnqueuedTotal  atomic.Int64
	tasksRunningTotal   atomic.Int64
	tasksSucceededTotal atomic.Int64
	tasksFailedTotal    atomic.Int64
	auditEventsTotal    atomic.Int64
	auditSinkFailures   atomic.Int64
	auditDeadLetters    atomic.Int64
	queueLagMs          durationSummary
	auditSinkLagMs      atomic.Int64
	uploadDurationMs    durationSummary

	statusTransitions sync.Map
	httpRequests      counterVec
	grpcRequests      counterVec
	authzDecisions    counterVec
	httpDurations     sync.Map
	grpcDurations     sync.Map
}

func NewMetrics() *Metrics { return &Metrics{} }

var DefaultMetrics = NewMetrics()

func (m *Metrics) SetQueueDepth(depth int64) { m.queueDepth.Store(depth) }
func (m *Metrics) IncEnqueued()              { m.tasksEnqueuedTotal.Add(1) }

func (m *Metrics) ObserveStatus(status string) {
	n := strings.ToLower(status)
	val, _ := m.statusTransitions.LoadOrStore(n, &atomic.Int64{})
	ctr := val.(*atomic.Int64)
	ctr.Add(1)
	switch n {
	case "running":
		m.tasksRunningTotal.Add(1)
	case "success":
		m.tasksSucceededTotal.Add(1)
	case "failed":
		m.tasksFailedTotal.Add(1)
	}
}

func (m *Metrics) FailedTasksTotal() int64 { return m.tasksFailedTotal.Load() }
func (m *Metrics) IncAuditEventEmitted()   { m.auditEventsTotal.Add(1) }
func (m *Metrics) IncAuditSinkFailure()    { m.auditSinkFailures.Add(1) }
func (m *Metrics) IncAuditDeadLetter()     { m.auditDeadLetters.Add(1) }

func (m *Metrics) ObserveHTTPRequest(method, route string, status int, durationMs int64) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "UNKNOWN"
	}
	route = sanitizeLabel(route)
	key := fmt.Sprintf("%s|%s|%d", method, route, status)
	m.httpRequests.inc(key)
	durKey := fmt.Sprintf("%s|%s", method, route)
	val, _ := m.httpDurations.LoadOrStore(durKey, &durationSummary{})
	val.(*durationSummary).observe(durationMs)
}

func (m *Metrics) ObserveGRPCRequest(method, code string, durationMs int64) {
	method = sanitizeLabel(method)
	code = sanitizeLabel(strings.ToLower(code))
	key := fmt.Sprintf("%s|%s", method, code)
	m.grpcRequests.inc(key)
	val, _ := m.grpcDurations.LoadOrStore(method, &durationSummary{})
	val.(*durationSummary).observe(durationMs)
}

func (m *Metrics) ObserveAuthzDecision(allowed bool, reason string) {
	decision := "deny"
	if allowed {
		decision = "allow"
	}
	reason = sanitizeLabel(strings.ToLower(reason))
	m.authzDecisions.inc(fmt.Sprintf("%s|%s", decision, reason))
}

func (m *Metrics) ObserveQueueLagMs(lagMs int64) {
	if lagMs < 0 {
		lagMs = 0
	}
	m.queueLagMs.observe(lagMs)
}

func (m *Metrics) SetAuditSinkLagMs(lagMs int64) {
	if lagMs < 0 {
		lagMs = 0
	}
	m.auditSinkLagMs.Store(lagMs)
}

func (m *Metrics) ObserveUploadDurationMs(d int64) {
	if d < 0 {
		d = 0
	}
	m.uploadDurationMs.observe(d)
}

func (m *Metrics) SnapshotPrometheus() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP fileengine_queue_depth Current queue depth\n")
	fmt.Fprintf(&b, "# TYPE fileengine_queue_depth gauge\n")
	fmt.Fprintf(&b, "fileengine_queue_depth %d\n", m.queueDepth.Load())
	fmt.Fprintf(&b, "# HELP fileengine_tasks_enqueued_total Total enqueued tasks\n")
	fmt.Fprintf(&b, "# TYPE fileengine_tasks_enqueued_total counter\n")
	fmt.Fprintf(&b, "fileengine_tasks_enqueued_total %d\n", m.tasksEnqueuedTotal.Load())
	fmt.Fprintf(&b, "# HELP fileengine_tasks_running_total Total running state transitions\n")
	fmt.Fprintf(&b, "# TYPE fileengine_tasks_running_total counter\n")
	fmt.Fprintf(&b, "fileengine_tasks_running_total %d\n", m.tasksRunningTotal.Load())
	fmt.Fprintf(&b, "# HELP fileengine_tasks_succeeded_total Total successful tasks\n")
	fmt.Fprintf(&b, "# TYPE fileengine_tasks_succeeded_total counter\n")
	fmt.Fprintf(&b, "fileengine_tasks_succeeded_total %d\n", m.tasksSucceededTotal.Load())
	fmt.Fprintf(&b, "# HELP fileengine_tasks_failed_total Total failed tasks\n")
	fmt.Fprintf(&b, "# TYPE fileengine_tasks_failed_total counter\n")
	fmt.Fprintf(&b, "fileengine_tasks_failed_total %d\n", m.tasksFailedTotal.Load())
	fmt.Fprintf(&b, "# HELP fileengine_audit_events_total Total emitted audit events\n")
	fmt.Fprintf(&b, "# TYPE fileengine_audit_events_total counter\n")
	fmt.Fprintf(&b, "fileengine_audit_events_total %d\n", m.auditEventsTotal.Load())
	fmt.Fprintf(&b, "# HELP fileengine_audit_sink_failures_total Total immutable sink export failures\n")
	fmt.Fprintf(&b, "# TYPE fileengine_audit_sink_failures_total counter\n")
	fmt.Fprintf(&b, "fileengine_audit_sink_failures_total %d\n", m.auditSinkFailures.Load())
	fmt.Fprintf(&b, "# HELP fileengine_audit_dead_letters_total Total dead-lettered audit events\n")
	fmt.Fprintf(&b, "# TYPE fileengine_audit_dead_letters_total counter\n")
	fmt.Fprintf(&b, "fileengine_audit_dead_letters_total %d\n", m.auditDeadLetters.Load())

	m.statusTransitions.Range(func(key, value any) bool {
		name, _ := key.(string)
		counter, _ := value.(*atomic.Int64)
		fmt.Fprintf(&b, "fileengine_task_status_transitions_total{status=\"%s\"} %d\n", name, counter.Load())
		return true
	})

	count, sum, max := m.queueLagMs.snapshot()
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_count %d\n", count)
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_sum %d\n", sum)
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_max %d\n", max)
	fmt.Fprintf(&b, "fileengine_audit_sink_lag_ms %d\n", m.auditSinkLagMs.Load())
	uCount, uSum, uMax := m.uploadDurationMs.snapshot()
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_count %d\n", uCount)
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_sum %d\n", uSum)
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_max %d\n", uMax)

	emitCounterVec(&b, "fileengine_http_requests_total", []string{"method", "route", "status"}, m.httpRequests.snapshot())
	emitCounterVec(&b, "fileengine_grpc_requests_total", []string{"method", "code"}, m.grpcRequests.snapshot())
	emitCounterVec(&b, "fileengine_authz_decisions_total", []string{"decision", "reason"}, m.authzDecisions.snapshot())
	emitDurationSummaries(&b, "fileengine_http_request_duration_ms", []string{"method", "route"}, &m.httpDurations)
	emitDurationSummaries(&b, "fileengine_grpc_request_duration_ms", []string{"method"}, &m.grpcDurations)
	return b.String()
}

func emitCounterVec(b *strings.Builder, metric string, labels []string, values map[string]int64) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) != len(labels) {
			continue
		}
		fmt.Fprintf(b, "%s{", metric)
		for i, label := range labels {
			if i > 0 {
				fmt.Fprintf(b, ",")
			}
			fmt.Fprintf(b, "%s=\"%s\"", label, sanitizeLabel(parts[i]))
		}
		fmt.Fprintf(b, "} %d\n", values[key])
	}
}

func emitDurationSummaries(b *strings.Builder, metric string, labels []string, mp *sync.Map) {
	pairs := map[string][3]int64{}
	mp.Range(func(k, v any) bool {
		key, _ := k.(string)
		s, _ := v.(*durationSummary)
		count, sum, max := s.snapshot()
		pairs[key] = [3]int64{count, sum, max}
		return true
	})
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) != len(labels) {
			continue
		}
		labelSet := ""
		for i, label := range labels {
			if i > 0 {
				labelSet += ","
			}
			labelSet += fmt.Sprintf("%s=\"%s\"", label, sanitizeLabel(parts[i]))
		}
		v := pairs[key]
		fmt.Fprintf(b, "%s_count{%s} %d\n", metric, labelSet, v[0])
		fmt.Fprintf(b, "%s_sum{%s} %d\n", metric, labelSet, v[1])
		fmt.Fprintf(b, "%s_max{%s} %d\n", metric, labelSet, v[2])
	}
}

func sanitizeLabel(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return strings.ReplaceAll(v, "\"", "")
}
