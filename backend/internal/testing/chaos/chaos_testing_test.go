package chaos

import (
	"context"
	"testing"
	"time"
)

func TestChaosEngineCreation(t *testing.T) {
	engine := NewChaosEngine()
	
	if engine == nil {
		t.Fatal("Expected chaos engine to be created, got nil")
	}
}

func TestAddExperiment(t *testing.T) {
	engine := NewChaosEngine()
	
	exp := &ChaosExperiment{
		Name:        "test-experiment",
		Description: "Test experiment",
	}
	
	engine.AddExperiment(exp)
}

func TestLatencyInjector(t *testing.T) {
	injector := NewLatencyInjector(100*time.Millisecond, 20*time.Millisecond)
	
	start := time.Now()
	injector.Inject()
	elapsed := time.Since(start)
	
	if elapsed < 50*time.Millisecond {
		t.Errorf("Expected at least 50ms latency, got %v", elapsed)
	}
}

func TestFaultInjector(t *testing.T) {
	testErr := ErrNetDown
	injector := NewFaultInjector(1.0, testErr) // Always inject error
	
	err := injector.Inject()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestRunChaosExperiment(t *testing.T) {
	engine := NewChaosEngine()
	
	ran := false
	engine.AddExperiment(&ChaosExperiment{
		Name: "test-experiment",
		Run: func(ctx context.Context) error {
			ran = true
			return nil
		},
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := engine.RunExperiment(ctx, "test-experiment")
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if !result.Success {
		t.Error("Expected experiment to succeed")
	}
	
	if !ran {
		t.Error("Expected experiment to run")
	}
}

func TestNetworkPartition(t *testing.T) {
	partition := NewNetworkPartition([]string{"192.168.1.1", "10.0.0.1"})
	
	partition.Block()
	
	if !partition.IsBlocked("192.168.1.1") {
		t.Error("Expected 192.168.1.1 to be blocked")
	}
	
	partition.Unblock()
	
	if partition.IsBlocked("192.168.1.1") {
		t.Error("Expected 192.168.1.1 to be unblocked")
	}
}

func TestResourceExhaustion(t *testing.T) {
	res := NewResourceExhaustion(100)
	
	err := res.ConsumeMemory(50)
	if err != nil {
		t.Error("Expected to consume memory")
	}
	
	err = res.ConsumeMemory(100)
	if err == nil {
		t.Error("Expected to error when exceeding memory")
	}
	
	res.ReleaseMemory(50)
	
	err = res.ConsumeMemory(50)
	if err != nil {
		t.Error("Expected to consume memory after releasing")
	}
}

func TestDiskFailure(t *testing.T) {
	disk := NewDiskFailure()
	
	err := disk.Read()
	if err != nil {
		t.Error("Expected no error when not failing")
	}
	
	disk.EnableIOErrors()
	
	err = disk.Read()
	if err == nil {
		t.Error("Expected error when failing")
	}
	
	disk.DisableIOErrors()
	
	err = disk.Read()
	if err != nil {
		t.Error("Expected no error after disabling failure")
	}
}
