package observability

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

type Metrics struct {
	queueDepth          atomic.Int64
	tasksEnqueuedTotal  atomic.Int64
	tasksRunningTotal   atomic.Int64
	tasksSucceededTotal atomic.Int64
	tasksFailedTotal    atomic.Int64

	statusTransitions sync.Map
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

var DefaultMetrics = NewMetrics()

func (m *Metrics) SetQueueDepth(depth int64) {
	m.queueDepth.Store(depth)
}

func (m *Metrics) IncEnqueued() {
	m.tasksEnqueuedTotal.Add(1)
}

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

func (m *Metrics) FailedTasksTotal() int64 {
	return m.tasksFailedTotal.Load()
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

	m.statusTransitions.Range(func(key, value any) bool {
		name, _ := key.(string)
		counter, _ := value.(*atomic.Int64)
		fmt.Fprintf(&b, "fileengine_task_status_transitions_total{status=\"%s\"} %d\n", name, counter.Load())
		return true
	})

	return b.String()
}
