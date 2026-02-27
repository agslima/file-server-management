package observability

import (
	"fmt"
	"maps"
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
	queueDepth             atomic.Int64
	tasksEnqueuedTotal     atomic.Int64
	tasksRunningTotal      atomic.Int64
	tasksSucceededTotal    atomic.Int64
	tasksFailedTotal       atomic.Int64
	auditEventsTotal       atomic.Int64
	auditSinkFailures      atomic.Int64
	auditDeadLetters       atomic.Int64
	queueLagMs             durationSummary
	auditSinkLagMs         atomic.Int64
	scanBacklog            atomic.Int64
	scanDLQSize            atomic.Int64
	uploadDurationMs       durationSummary
	malwareScanDuration    durationSummary
	malwareScanVerdicts    counterVec
	quarantineTimeMs       durationSummary
	governanceDrift        counterVec
	governanceDriftOn      atomic.Int64
	archiveTransitions     atomic.Int64
	rateLimitBlocks        counterVec
	queueRejects           atomic.Int64
	tenantUploadBytes      sync.Map
	tenantDownloadBytes    sync.Map
	integrityChecksTotal   atomic.Int64
	integrityFailuresTotal atomic.Int64

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

func (m *Metrics) ObserveMalwareScanDurationMs(d int64) {
	if d < 0 {
		d = 0
	}
	m.malwareScanDuration.observe(d)
}

func (m *Metrics) SetScanBacklog(n int64) {
	if n < 0 {
		n = 0
	}
	m.scanBacklog.Store(n)
}

func (m *Metrics) SetScanDLQSize(n int64) {
	if n < 0 {
		n = 0
	}
	m.scanDLQSize.Store(n)
}

func (m *Metrics) ObserveQuarantineTimeMs(d int64) {
	if d < 0 {
		d = 0
	}
	m.quarantineTimeMs.observe(d)
}

func (m *Metrics) IncMalwareScanVerdict(verdict string) {
	v := sanitizeLabel(strings.ToLower(strings.TrimSpace(verdict)))
	if v == "" {
		v = "unknown"
	}
	m.malwareScanVerdicts.inc(v)
}

func (m *Metrics) ObserveGovernanceDrift(drifted bool, reason string) {
	state := "in_sync"
	if drifted {
		state = "drift"
		m.governanceDriftOn.Store(1)
	} else {
		m.governanceDriftOn.Store(0)
	}
	m.governanceDrift.inc(fmt.Sprintf("%s|%s", state, sanitizeLabel(strings.ToLower(reason))))
}

func (m *Metrics) IncArchiveTransition() {
	m.archiveTransitions.Add(1)
}

func (m *Metrics) IncRateLimitBlock(scope, reason string) {
	m.rateLimitBlocks.inc(fmt.Sprintf("%s|%s", sanitizeLabel(strings.ToLower(scope)), sanitizeLabel(strings.ToLower(reason))))
}

func (m *Metrics) IncQueueBackpressureReject() { m.queueRejects.Add(1) }

func (m *Metrics) addTenantBytes(mp *sync.Map, tenantID string, bytes int64) {
	tenantID = sanitizeLabel(tenantID)
	if tenantID == "unknown" || bytes <= 0 {
		return
	}
	val, _ := mp.LoadOrStore(tenantID, &atomic.Int64{})
	val.(*atomic.Int64).Add(bytes)
}

// snapshotTenantBytes returns a new map[string]int64 containing the current counter value for each tenant stored in mp.
// It treats map keys as strings and map values as *atomic.Int64, loading each counter's current int64 value.
func snapshotTenantBytes(mp *sync.Map) map[string]int64 {
	out := map[string]int64{}
	mp.Range(func(k, v any) bool {
		tenant, _ := k.(string)
		ctr, _ := v.(*atomic.Int64)
		out[tenant] = ctr.Load()
		return true
	})
	return out
}

func (m *Metrics) AddTenantUploadedBytes(tenantID string, bytes int64) {
	m.addTenantBytes(&m.tenantUploadBytes, tenantID, bytes)
}

func (m *Metrics) AddTenantDownloadedBytes(tenantID string, bytes int64) {
	m.addTenantBytes(&m.tenantDownloadBytes, tenantID, bytes)
}

func (m *Metrics) IncIntegrityCheck(ok bool) {
	m.integrityChecksTotal.Add(1)
	if !ok {
		m.integrityFailuresTotal.Add(1)
	}
}

func (m *Metrics) TenantUsageSnapshot() map[string]map[string]int64 {
	out := map[string]map[string]int64{}
	for tenant, uploaded := range snapshotTenantBytes(&m.tenantUploadBytes) {
		out[tenant] = map[string]int64{"uploaded_bytes": uploaded, "downloaded_bytes": 0}
	}
	for tenant, downloaded := range snapshotTenantBytes(&m.tenantDownloadBytes) {
		if _, ok := out[tenant]; !ok {
			out[tenant] = map[string]int64{"uploaded_bytes": 0, "downloaded_bytes": 0}
		}
		out[tenant]["downloaded_bytes"] = downloaded
	}
	return out
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

	count, sum, maxValue := m.queueLagMs.snapshot()
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_count %d\n", count)
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_sum %d\n", sum)
	fmt.Fprintf(&b, "fileengine_queue_lag_ms_max %d\n", maxValue)
	fmt.Fprintf(&b, "fileengine_audit_sink_lag_ms %d\n", m.auditSinkLagMs.Load())
	fmt.Fprintf(&b, "fileengine_scan_backlog %d\n", m.scanBacklog.Load())
	fmt.Fprintf(&b, "fileengine_scan_dlq_size %d\n", m.scanDLQSize.Load())
	uCount, uSum, uMax := m.uploadDurationMs.snapshot()
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_count %d\n", uCount)
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_sum %d\n", uSum)
	fmt.Fprintf(&b, "fileengine_upload_duration_ms_max %d\n", uMax)
	sCount, sSum, sMax := m.malwareScanDuration.snapshot()
	fmt.Fprintf(&b, "fileengine_malware_scan_duration_ms_count %d\n", sCount)
	fmt.Fprintf(&b, "fileengine_malware_scan_duration_ms_sum %d\n", sSum)
	fmt.Fprintf(&b, "fileengine_malware_scan_duration_ms_max %d\n", sMax)
	emitCounterVec(&b, "fileengine_malware_scan_verdict_total", []string{"verdict"}, m.malwareScanVerdicts.snapshot())
	qCount, qSum, qMax := m.quarantineTimeMs.snapshot()
	fmt.Fprintf(&b, "fileengine_quarantine_time_ms_count %d\n", qCount)
	fmt.Fprintf(&b, "fileengine_quarantine_time_ms_sum %d\n", qSum)
	fmt.Fprintf(&b, "fileengine_quarantine_time_ms_max %d\n", qMax)
	fmt.Fprintf(&b, "fileengine_governance_drift_active %d\n", m.governanceDriftOn.Load())
	fmt.Fprintf(&b, "fileengine_archive_transitions_total %d\n", m.archiveTransitions.Load())
	fmt.Fprintf(&b, "fileengine_queue_backpressure_rejections_total %d\n", m.queueRejects.Load())
	fmt.Fprintf(&b, "fileengine_integrity_checks_total %d\n", m.integrityChecksTotal.Load())
	fmt.Fprintf(&b, "fileengine_integrity_failures_total %d\n", m.integrityFailuresTotal.Load())
	emitCounterVec(&b, "fileengine_governance_drift_checks_total", []string{"state", "reason"}, m.governanceDrift.snapshot())
	emitCounterVec(&b, "fileengine_rate_limit_blocks_total", []string{"scope", "reason"}, m.rateLimitBlocks.snapshot())
	emitCounterVec(&b, "fileengine_tenant_uploaded_bytes_total", []string{"tenant"}, stringifyTenantBytes(snapshotTenantBytes(&m.tenantUploadBytes)))
	emitCounterVec(&b, "fileengine_tenant_downloaded_bytes_total", []string{"tenant"}, stringifyTenantBytes(snapshotTenantBytes(&m.tenantDownloadBytes)))

	emitCounterVec(&b, "fileengine_http_requests_total", []string{"method", "route", "status"}, m.httpRequests.snapshot())
	emitCounterVec(&b, "fileengine_grpc_requests_total", []string{"method", "code"}, m.grpcRequests.snapshot())
	emitCounterVec(&b, "fileengine_authz_decisions_total", []string{"decision", "reason"}, m.authzDecisions.snapshot())
	emitDurationSummaries(&b, "fileengine_http_request_duration_ms", []string{"method", "route"}, &m.httpDurations)
	emitDurationSummaries(&b, "fileengine_grpc_request_duration_ms", []string{"method"}, &m.grpcDurations)
	return b.String()
}

// stringifyTenantBytes returns a shallow copy of the provided tenant bytes map.
func stringifyTenantBytes(values map[string]int64) map[string]int64 {
	out := map[string]int64{}
	maps.Copy(out, values)
	return out
}

// emitCounterVec writes Prometheus-formatted counter lines for a metric with label names to the provided builder.
// It deterministically iterates over the sorted keys of values, where each key is a '|'-separated list of label values
// matching the provided labels slice. Keys with a different number of parts are skipped and each label value is sanitized.
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
		count, sum, maxValue := s.snapshot()
		pairs[key] = [3]int64{count, sum, maxValue}
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
		var labelSetSb249 strings.Builder
		for i, label := range labels {
			if i > 0 {
				labelSetSb249.WriteString(",")
			}
			fmt.Fprintf(&labelSetSb249, "%s=\"%s\"", label, sanitizeLabel(parts[i]))
		}
		labelSet += labelSetSb249.String()
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
