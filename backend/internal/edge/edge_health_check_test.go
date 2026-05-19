package edge

import (
	"testing"
)

func TestHealthCheckManager_NewHealthCheckManager(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)

	if manager == nil {
		t.Fatal("Expected manager to not be nil")
	}

	if len(manager.checks) != 0 {
		t.Errorf("Expected 0 checks initially, got %d", len(manager.checks))
	}

	if len(manager.nodeHealth) != 0 {
		t.Errorf("Expected 0 node health entries initially, got %d", len(manager.nodeHealth))
	}
}

func TestHealthCheckManager_AddCheck(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)
	defer manager.Stop()

	check := &HealthCheck{
		NodeID:   "test-node",
		Target:   "10.0.0.1",
		Port:     8080,
		Type:     CheckTypeTCP,
		Interval: 30,
		Timeout:  5,
	}

	err := manager.AddCheck(check)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.ID == "" {
		t.Error("Expected check ID to be set")
	}

	retrieved, err := manager.GetCheck(check.ID)
	if err != nil {
		t.Fatalf("GetCheck failed: %v", err)
	}

	if retrieved.Target != check.Target {
		t.Errorf("Expected target %s, got %s", check.Target, retrieved.Target)
	}
}

func TestHealthCheckManager_DeleteCheck(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)
	defer manager.Stop()

	check := &HealthCheck{
		NodeID:   "test-node",
		Target:   "10.0.0.1",
		Port:     8080,
		Type:     CheckTypeTCP,
		Interval: 30,
		Timeout:  5,
	}

	manager.AddCheck(check)

	err := manager.DeleteCheck(check.ID)
	if err != nil {
		t.Fatalf("DeleteCheck failed: %v", err)
	}

	_, err = manager.GetCheck(check.ID)
	if err == nil {
		t.Error("Expected error for deleted check")
	}
}

func TestHealthCheckManager_ListChecks(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)
	defer manager.Stop()

	checks := []*HealthCheck{
		{
			NodeID:   "node-1",
			Target:   "10.0.0.1",
			Port:     8080,
			Type:     CheckTypeTCP,
			Interval: 30,
			Timeout:  5,
		},
		{
			NodeID:   "node-2",
			Target:   "10.0.0.2",
			Port:     8080,
			Type:     CheckTypeHTTP,
			Interval: 30,
			Timeout:  5,
		},
	}

	for _, check := range checks {
		manager.AddCheck(check)
	}

	allChecks := manager.ListChecks("")
	if len(allChecks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(allChecks))
	}

	node1Checks := manager.ListChecks("node-1")
	if len(node1Checks) != 1 {
		t.Errorf("Expected 1 check for node-1, got %d", len(node1Checks))
	}
}

func TestHealthCheckManager_GetMetrics(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)
	defer manager.Stop()

	metrics := manager.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	if metrics.TotalChecks != 0 {
		t.Errorf("Expected 0 total checks, got %d", metrics.TotalChecks)
	}
}

func TestHealthCheckManager_UpdateCheck(t *testing.T) {
	manager := NewHealthCheckManager(nil, nil)
	defer manager.Stop()

	check := &HealthCheck{
		NodeID:   "test-node",
		Target:   "10.0.0.1",
		Port:     8080,
		Type:     CheckTypeTCP,
		Interval: 30,
		Timeout:  5,
	}

	manager.AddCheck(check)

	check.Target = "10.0.0.2"
	check.Timeout = 10

	err := manager.UpdateCheck(check)
	if err != nil {
		t.Fatalf("UpdateCheck failed: %v", err)
	}

	retrieved, _ := manager.GetCheck(check.ID)
	if retrieved.Target != "10.0.0.2" {
		t.Errorf("Expected target 10.0.0.2, got %s", retrieved.Target)
	}
	if retrieved.Timeout != 10 {
		t.Errorf("Expected timeout 10, got %d", retrieved.Timeout)
	}
}
