package monitors

import (
	"context"
	"errors"
	"testing"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

type fakePinger struct {
	stats       *probing.Statistics
	runErr      error
	fallbackErr error
	privileged  bool
	runCount    int
	count       int
	timeout     time.Duration
}

func (f *fakePinger) Run() error {
	f.runCount++
	if f.runCount == 1 {
		return f.runErr
	}
	return f.fallbackErr
}

func (f *fakePinger) Stop() {}

func (f *fakePinger) SetPrivileged(privileged bool) {
	f.privileged = privileged
}

func (f *fakePinger) Privileged() bool {
	return f.privileged
}

func (f *fakePinger) SetCount(count int) {
	f.count = count
}

func (f *fakePinger) SetTimeout(timeout time.Duration) {
	f.timeout = timeout
}

func (f *fakePinger) Statistics() *probing.Statistics {
	return f.stats
}

func newTestLogger(t *testing.T) *logging.Logger {
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

func TestPingMonitorCheckSuccess(t *testing.T) {
	logger := newTestLogger(t)
	monitorCfg := &models.Monitor{
		Name:   "loopback",
		Type:   models.MonitorTypePing,
		Target: "127.0.0.1",
		Count:  4,
	}

	pm, err := NewPingMonitor(monitorCfg, "core", logger, nil)
	if err != nil {
		t.Fatalf("failed to create ping monitor: %v", err)
	}

	fake := &fakePinger{
		stats: &probing.Statistics{
			PacketsSent: 4,
			PacketsRecv: 4,
			PacketLoss:  0,
			MinRtt:      2 * time.Millisecond,
			MaxRtt:      8 * time.Millisecond,
			AvgRtt:      4 * time.Millisecond,
		},
		privileged: true,
	}

	pm.newPinger = func(target string) (pinger, error) {
		return fake, nil
	}

	result, err := pm.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error from ping check: %v", err)
	}

	if result.Status != models.StatusUp {
		t.Fatalf("expected status up, got %s", result.Status)
	}

	if result.PingResult == nil || result.PingResult.PacketLoss != 0 {
		t.Fatalf("expected ping result with zero packet loss, got %+v", result.PingResult)
	}

	if fake.count != pm.count {
		t.Fatalf("expected pinger count %d, got %d", pm.count, fake.count)
	}

	if fake.timeout <= 0 {
		t.Fatalf("expected timeout to be set, got %s", fake.timeout)
	}
}

func TestPingMonitorFallbackToUnprivileged(t *testing.T) {
	logger := newTestLogger(t)
	monitorCfg := &models.Monitor{
		Name:   "loopback",
		Type:   models.MonitorTypePing,
		Target: "127.0.0.1",
	}

	pm, err := NewPingMonitor(monitorCfg, "core", logger, nil)
	if err != nil {
		t.Fatalf("failed to create ping monitor: %v", err)
	}

	fake := &fakePinger{
		stats: &probing.Statistics{
			PacketsSent: 3,
			PacketsRecv: 3,
			PacketLoss:  0,
		},
		runErr:      errors.New("icmp not permitted"),
		fallbackErr: nil,
		privileged:  true,
	}

	pm.newPinger = func(target string) (pinger, error) {
		return fake, nil
	}

	result, err := pm.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error from ping check: %v", err)
	}

	if fake.runCount != 2 {
		t.Fatalf("expected two run attempts, got %d", fake.runCount)
	}

	if fake.privileged {
		t.Fatalf("expected pinger to switch to unprivileged mode")
	}

	if result.Status != models.StatusUp {
		t.Fatalf("expected status up after fallback, got %s", result.Status)
	}
}

func TestPingMonitorHighPacketLoss(t *testing.T) {
	logger := newTestLogger(t)
	monitorCfg := &models.Monitor{
		Name:   "loopback",
		Type:   models.MonitorTypePing,
		Target: "127.0.0.1",
	}

	pm, err := NewPingMonitor(monitorCfg, "core", logger, nil)
	if err != nil {
		t.Fatalf("failed to create ping monitor: %v", err)
	}

	fake := &fakePinger{
		stats: &probing.Statistics{
			PacketsSent: 4,
			PacketsRecv: 1,
			PacketLoss:  75,
		},
		privileged: true,
	}

	pm.newPinger = func(target string) (pinger, error) {
		return fake, nil
	}

	result, err := pm.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error from ping check: %v", err)
	}

	if result.Status != models.StatusDown {
		t.Fatalf("expected status down for high packet loss, got %s", result.Status)
	}

	if result.Error == "" {
		t.Fatalf("expected error message for high packet loss")
	}
}
