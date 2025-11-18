package scheduler

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/internal/monitors"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

func newSchedulerTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	prevLevel := zerolog.GlobalLevel()
	prevLogger := zerologlog.Logger
	t.Cleanup(func() {
		zerolog.SetGlobalLevel(prevLevel)
		zerologlog.Logger = prevLogger
	})

	logger, err := logging.InitLogger(logging.Config{Level: "debug", Format: "json", Output: "stdout"})
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	return logger
}

type stubMonitor struct {
	name        string
	group       string
	monitorType models.MonitorType
	interval    time.Duration
	timeout     time.Duration
	enabled     bool
	checks      int32
	result      models.MonitorResult
}

func (m *stubMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	atomic.AddInt32(&m.checks, 1)
	res := m.result
	res.Timestamp = time.Now()
	return &res, nil
}

func (m *stubMonitor) GetConfig() *models.Monitor {
	return &models.Monitor{Name: m.name, Type: m.monitorType, Interval: models.Duration(m.interval), Timeout: models.Duration(m.timeout)}
}

func (m *stubMonitor) GetName() string             { return m.name }
func (m *stubMonitor) GetType() models.MonitorType { return m.monitorType }
func (m *stubMonitor) GetGroup() string            { return m.group }
func (m *stubMonitor) IsEnabled() bool             { return m.enabled }
func (m *stubMonitor) Validate() error             { return nil }

func setMonitorManagerMonitors(t *testing.T, manager *monitors.MonitorManager, monitorList []monitors.Monitor) {
	t.Helper()
	value := reflect.ValueOf(manager).Elem()
	field := value.FieldByName("monitors")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(monitorList))
}

func TestSchedulerStartStopLifecycle(t *testing.T) {
	logger := newSchedulerTestLogger(t)
	metricRegistry := prometheus.NewRegistry()
	metricsInstance := metrics.NewMetrics(metricRegistry)
	manager := monitors.NewMonitorManager(logger, metricsInstance)

	sched := NewScheduler(logger, metricsInstance, manager)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("unexpected error starting scheduler: %v", err)
	}

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("expected second start call to be no-op, got error: %v", err)
	}

	cancel()

	if err := sched.Stop(); err != nil {
		t.Fatalf("unexpected error stopping scheduler: %v", err)
	}

	if err := sched.Stop(); err != nil {
		t.Fatalf("expected second stop call to be no-op, got error: %v", err)
	}
}

func TestSchedulerProcessesMonitorJob(t *testing.T) {
	logger := newSchedulerTestLogger(t)
	metricRegistry := prometheus.NewRegistry()
	metricsInstance := metrics.NewMetrics(metricRegistry)
	manager := monitors.NewMonitorManager(logger, metricsInstance)

	monitor := &stubMonitor{
		name:        "api",
		group:       "core",
		monitorType: models.MonitorTypeHTTP,
		interval:    5 * time.Second,
		timeout:     time.Second,
		enabled:     true,
		result: models.MonitorResult{
			Monitor:  "api",
			Type:     models.MonitorTypeHTTP,
			Group:    "core",
			Status:   models.StatusUp,
			Duration: 150 * time.Millisecond,
		},
	}

	setMonitorManagerMonitors(t, manager, []monitors.Monitor{monitor})

	sched := NewScheduler(logger, metricsInstance, manager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.workers = NewWorkerPool(1, logger, metricsInstance)
	sched.workers.Start(ctx)
	defer sched.workers.Stop()

	nextExecution := map[string]time.Time{
		monitor.GetName(): time.Now().Add(-time.Second),
	}

	sched.checkAndScheduleMonitors(ctx, time.Now(), nextExecution)

	deadline := time.Now().Add(500 * time.Millisecond)
	var latest *models.MonitorResult
	for time.Now().Before(deadline) {
		latest = sched.GetLatestResult(monitor.GetName())
		if latest != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if latest == nil {
		t.Fatalf("expected scheduler to store monitor result")
	}

	if latest.Status != models.StatusUp {
		t.Fatalf("expected stored result to be up, got %s", latest.Status)
	}

	if atomic.LoadInt32(&monitor.checks) == 0 {
		t.Fatalf("expected monitor check to be invoked")
	}

	if next := nextExecution[monitor.GetName()]; !next.After(time.Now()) {
		t.Fatalf("expected next execution to be scheduled in the future, got %s", next)
	}
}
