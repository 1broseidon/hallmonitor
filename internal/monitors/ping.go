package monitors

import (
	"context"
	"fmt"
	"net"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

type pinger interface {
	Run() error
	Stop()
	SetPrivileged(bool)
	Privileged() bool
	SetCount(int)
	SetTimeout(time.Duration)
	Statistics() *probing.Statistics
}

type probingPinger struct {
	*probing.Pinger
}

func (p *probingPinger) SetCount(count int) {
	p.Pinger.Count = count
}

func (p *probingPinger) SetTimeout(timeout time.Duration) {
	p.Pinger.Timeout = timeout
}

func defaultPingerFactory(target string) (pinger, error) {
	p, err := probing.NewPinger(target)
	if err != nil {
		return nil, err
	}
	return &probingPinger{Pinger: p}, nil
}

// PingMonitor implements ICMP ping monitoring
type PingMonitor struct {
	*BaseMonitor
	target    net.IP
	isIPv6    bool
	count     int
	newPinger func(string) (pinger, error)
}

// NewPingMonitor creates a new ping monitor
func NewPingMonitor(config *models.Monitor, group string, logger *logging.Logger, metrics *metrics.Metrics) (*PingMonitor, error) {
	// Resolve target to IP address
	ip, err := net.ResolveIPAddr("ip", config.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target '%s': %w", config.Target, err)
	}

	// Determine if IPv6
	isIPv6 := ip.IP.To4() == nil

	// Set default count
	count := config.Count
	if count == 0 {
		count = 3
	}

	return &PingMonitor{
		BaseMonitor: NewBaseMonitor(config, group, logger, metrics),
		target:      ip.IP,
		isIPv6:      isIPv6,
		count:       count,
		newPinger:   defaultPingerFactory,
	}, nil
}

// Check performs the ping check
func (p *PingMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	startTime := time.Now()

	// Perform ping
	result, err := p.performPing(ctx)
	if err != nil {
		duration := time.Since(startTime)
		monitorResult := p.CreateResult(models.StatusDown, duration, err)
		p.RecordMetrics(monitorResult)
		p.LogResult(monitorResult)
		return monitorResult, nil
	}

	// Determine status based on packet loss
	var status models.MonitorStatus
	var checkError error

	if result.PacketLoss >= 100.0 {
		status = models.StatusDown
		checkError = fmt.Errorf("100%% packet loss")
	} else if result.PacketLoss >= 50.0 {
		status = models.StatusDown
		checkError = fmt.Errorf("high packet loss: %.1f%%", result.PacketLoss)
	} else {
		status = models.StatusUp
	}

	duration := time.Since(startTime)
	monitorResult := p.CreateResult(status, duration, checkError)
	monitorResult.PingResult = result

	// Record ping-specific metrics
	if p.Metrics != nil {
		p.Metrics.RecordPingCheck(
			p.Config.Name,
			p.Group,
			result.AvgRTT,
			result.PacketLoss,
		)
	}

	p.RecordMetrics(monitorResult)
	p.LogResult(monitorResult)

	return monitorResult, nil
}

// performPing executes the actual ICMP ping operation
func (p *PingMonitor) performPing(ctx context.Context) (*models.PingResult, error) {
	timeout := p.Config.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	// Create pinger
	pinger, err := p.newPinger(p.Config.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create pinger: %w", err)
	}

	// Configure pinger
	pinger.SetCount(p.count)
	pinger.SetTimeout(timeout)

	// Try privileged mode first (ICMP), fall back to unprivileged if needed
	pinger.SetPrivileged(true)

	// Handle context cancellation
	done := make(chan error, 1)
	go func() {
		done <- pinger.Run()
	}()

	// Wait for completion or context cancellation
	select {
	case <-ctx.Done():
		pinger.Stop()
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			// If privileged mode failed, try unprivileged (UDP-based)
			p.Logger.WithComponent(logging.ComponentMonitor).
				WithFields(map[string]interface{}{
					"monitor": p.Config.Name,
					"target":  p.Config.Target,
				}).
				Debug("Privileged ICMP failed, trying unprivileged mode")

			// Retry with unprivileged mode
			pinger.SetPrivileged(false)

			done2 := make(chan error, 1)
			go func() {
				done2 <- pinger.Run()
			}()

			select {
			case <-ctx.Done():
				pinger.Stop()
				return nil, ctx.Err()
			case err2 := <-done2:
				if err2 != nil {
					return nil, fmt.Errorf("ping failed in both privileged and unprivileged mode: %w", err2)
				}
			}
		}
	}

	// Get statistics
	stats := pinger.Statistics()

	// Log mode used
	mode := "ICMP"
	if !pinger.Privileged() {
		mode = "UDP (unprivileged)"
	}
	p.Logger.WithComponent(logging.ComponentMonitor).
		WithFields(map[string]interface{}{
			"monitor":      p.Config.Name,
			"target":       p.Config.Target,
			"mode":         mode,
			"packets_sent": stats.PacketsSent,
			"packets_recv": stats.PacketsRecv,
			"packet_loss":  stats.PacketLoss,
		}).
		Debug("Ping completed successfully")

	return &models.PingResult{
		PacketsSent:     stats.PacketsSent,
		PacketsReceived: stats.PacketsRecv,
		PacketLoss:      stats.PacketLoss,
		MinRTT:          stats.MinRtt,
		MaxRTT:          stats.MaxRtt,
		AvgRTT:          stats.AvgRtt,
	}, nil
}

// Validate validates the ping monitor configuration
func (p *PingMonitor) Validate() error {
	if p.Config.Target == "" {
		return fmt.Errorf("ping monitor requires target")
	}

	// Try to resolve the target
	_, err := net.ResolveIPAddr("ip", p.Config.Target)
	if err != nil {
		return fmt.Errorf("cannot resolve target '%s': %w", p.Config.Target, err)
	}

	// Validate count if specified
	if p.Config.Count < 0 || p.Config.Count > 100 {
		return fmt.Errorf("ping count must be between 0 and 100, got %d", p.Config.Count)
	}

	return nil
}
