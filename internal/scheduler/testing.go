package scheduler

import "github.com/1broseidon/hallmonitor/pkg/models"

// TestHelper provides test-only methods for scheduler testing.
// These methods should NEVER be used in production code.
type TestHelper struct {
	scheduler *Scheduler
}

// NewTestHelper creates a test helper for the given scheduler.
// This should only be used in test code.
func (s *Scheduler) NewTestHelper() *TestHelper {
	return &TestHelper{scheduler: s}
}

// InjectResult injects a test result directly into the result store.
// This bypasses the normal check execution flow and should only be used in tests.
func (th *TestHelper) InjectResult(monitorName string, result *models.MonitorResult) {
	th.scheduler.resultStore.StoreResult(monitorName, result)
}

// GetResultStore returns the underlying result store for advanced testing.
// Use with caution - prefer InjectResult for most test scenarios.
func (th *TestHelper) GetResultStore() *ResultStore {
	return th.scheduler.resultStore
}
