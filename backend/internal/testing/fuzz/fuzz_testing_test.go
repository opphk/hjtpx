package fuzz

import (
	"testing"
)

func TestFuzzerCreation(t *testing.T) {
	config := DefaultFuzzingConfig()
	fuzzer := NewFuzzer(config)
	
	if fuzzer == nil {
		t.Fatal("Expected fuzzer to be created, got nil")
	}
}

func TestFuzzBytes(t *testing.T) {
	fuzzer := NewFuzzer(nil)
	
	for i := 0; i < 100; i++ {
		data := fuzzer.FuzzBytes(0)
		if data == nil {
			t.Error("Expected non-nil data")
		}
	}
}

func TestEdgeCases(t *testing.T) {
	cases := EdgeCases()
	
	if len(cases) == 0 {
		t.Fatal("Expected some edge cases, got none")
	}
	
	t.Logf("Got %d edge cases", len(cases))
}

func TestFuzzRun(t *testing.T) {
	fuzzer := NewFuzzer(&FuzzingConfig{
		Iterations: 10,
		Verbose:    false,
	})
	
	result := fuzzer.Run(t, "test", func(input []byte) error {
		// Just process the input without errors
		return nil
	})
	
	if result.TotalIterations != 10 {
		t.Errorf("Expected 10 iterations, got %d", result.TotalIterations)
	}
	
	if result.Panics != 0 {
		t.Errorf("Expected 0 panics, got %d", result.Panics)
	}
}
