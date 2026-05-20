package privacy

import (
	"testing"
)

func TestDifferentialPrivacy_AddNoise(t *testing.T) {
	dp := NewDifferentialPrivacy(1.0, 1e-5, 1.0, GaussianNoise, CountingMechanism)

	originalValue := 100.0
	noisyValue := dp.AddNoise(originalValue)

	if noisyValue == originalValue {
		t.Error("Noise should have been added to the value")
	}

	epsilon, delta, sensitivity := dp.GetPrivacyParameters()
	if epsilon != 1.0 {
		t.Errorf("Expected epsilon 1.0, got %f", epsilon)
	}
	if delta != 1e-5 {
		t.Errorf("Expected delta 1e-5, got %e", delta)
	}
	if sensitivity != 1.0 {
		t.Errorf("Expected sensitivity 1.0, got %f", sensitivity)
	}
}

func TestDifferentialPrivacy_LaplaceMechanism(t *testing.T) {
	dp := NewDifferentialPrivacy(1.0, 0, 1.0, LaplaceNoise, SumMechanism)

	originalValue := 50.0
	noisyValue := dp.AddNoise(originalValue)

	if noisyValue == originalValue {
		t.Error("Laplace noise should have been added")
	}
}

func TestDifferentialPrivacy_Compose(t *testing.T) {
	dp1 := NewDifferentialPrivacy(1.0, 1e-5, 1.0, GaussianNoise, CountingMechanism)
	dp2 := NewDifferentialPrivacy(0.5, 5e-6, 1.0, GaussianNoise, CountingMechanism)

	composed := dp1.Compose(dp2)

	epsilon, delta, _ := composed.GetPrivacyParameters()
	if epsilon != 1.5 {
		t.Errorf("Expected composed epsilon 1.5, got %f", epsilon)
	}
	if delta < 1.49e-5 || delta > 1.51e-5 {
		t.Errorf("Expected composed delta around 1.5e-5, got %e", delta)
	}
}

func TestDifferentialPrivacy_PostProcess(t *testing.T) {
	dp := NewDifferentialPrivacy(1.0, 1e-5, 1.0, GaussianNoise, CountingMechanism)

	testCases := []struct {
		input    float64
		min      float64
		max      float64
		expected float64
	}{
		{50.0, 0.0, 100.0, 50.0},
		{-10.0, 0.0, 100.0, 0.0},
		{150.0, 0.0, 100.0, 100.0},
	}

	for _, tc := range testCases {
		result := dp.PostProcess(tc.input, tc.min, tc.max)
		if result != tc.expected {
			t.Errorf("PostProcess(%f, %f, %f) = %f; expected %f",
				tc.input, tc.min, tc.max, result, tc.expected)
		}
	}
}

func TestPrivacyBudgetManager(t *testing.T) {
	pbm := NewPrivacyBudgetManager(10.0)

	if !pbm.Spend(5.0) {
		t.Error("Should be able to spend 5.0 from budget of 10.0")
	}

	remaining := pbm.GetRemainingBudget()
	if remaining != 5.0 {
		t.Errorf("Expected remaining budget 5.0, got %f", remaining)
	}

	if pbm.Spend(6.0) {
		t.Error("Should not be able to spend 6.0 from remaining budget of 5.0")
	}

	pbm.Reset()
	if pbm.GetRemainingBudget() != 10.0 {
		t.Error("Budget should be reset to 10.0")
	}
}

func TestPrivacyAccountant(t *testing.T) {
	pa := NewPrivacyAccountant(10.0, 1e-5)

	err := pa.Account(3.0, 3e-6)
	if err != nil {
		t.Error("First account should succeed")
	}

	spentEpsilon, _ := pa.GetSpentBudget()
	if spentEpsilon != 3.0 {
		t.Errorf("Expected spent epsilon 3.0, got %f", spentEpsilon)
	}

	err = pa.Account(8.0, 8e-6)
	if err != ErrBudgetExceeded {
		t.Error("Second account should fail due to budget exceeded")
	}
}

func TestPrivateQueryExecutor(t *testing.T) {
	executor := NewPrivateQueryExecutor(10.0, 1e-5)

	executor.RegisterQuery("count", &PrivateQuery{
		QueryType:   "count",
		Sensitivity: 1.0,
		Bounds:      [2]float64{0, 1000},
		NoiseType:   LaplaceNoise,
		Epsilon:     1.0,
		Delta:       1e-6,
	})

	result, err := executor.ExecuteQuery("count", 100)
	if err != nil {
		t.Errorf("ExecuteQuery failed: %v", err)
	}

	if result < 0 || result > 1100 {
		t.Errorf("Result %f is outside expected range", result)
	}

	_, err = executor.ExecuteQuery("nonexistent", 100)
	if err != ErrQueryNotFound {
		t.Error("Should return ErrQueryNotFound for nonexistent query")
	}
}

func TestGaussianMechanism_AddNoise(t *testing.T) {
	config := GaussianConfig{
		Epsilon:     1.0,
		Delta:       1e-5,
		Sensitivity: 1.0,
	}

	gm := NewGaussianMechanism(config)

	original := 100.0
	noisy := gm.AddNoise(original)

	if noisy == original {
		t.Error("Gaussian noise should have been added")
	}

	sigma := gm.GetSigma()
	if sigma <= 0 {
		t.Error("Sigma should be positive")
	}
}

func TestGaussianMechanism_Compose(t *testing.T) {
	config1 := GaussianConfig{
		Epsilon:     1.0,
		Delta:       1e-5,
		Sensitivity: 1.0,
	}

	config2 := GaussianConfig{
		Epsilon:     0.5,
		Delta:       5e-6,
		Sensitivity: 1.0,
	}

	gm1 := NewGaussianMechanism(config1)
	gm2 := NewGaussianMechanism(config2)

	composed := gm1.Compose(gm2)

	epsilon, delta := composed.PrivacyUsage()
	if epsilon != 1.5 {
		t.Errorf("Expected composed epsilon 1.5, got %f", epsilon)
	}
	if delta < 1.49e-5 || delta > 1.51e-5 {
		t.Errorf("Expected composed delta around 1.5e-5, got %e", delta)
	}
}

func TestGaussianMechanism_Vector(t *testing.T) {
	config := GaussianConfig{
		Epsilon:     1.0,
		Delta:       1e-5,
		Sensitivity: 1.0,
	}

	gm := NewGaussianMechanism(config)

	vector := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	noisyVector := gm.AddNoiseToVector(vector)

	if len(noisyVector) != len(vector) {
		t.Errorf("Expected length %d, got %d", len(vector), len(noisyVector))
	}

	for i := range vector {
		if noisyVector[i] == vector[i] {
			t.Errorf("Noise should have been added to element %d", i)
		}
	}
}

func TestLaplaceMechanism(t *testing.T) {
	config := LaplaceConfig{
		Epsilon:     1.0,
		Sensitivity: 1.0,
	}

	lm := NewLaplaceMechanism(config)

	original := 50.0
	noisy := lm.AddNoise(original)

	if noisy == original {
		t.Error("Laplace noise should have been added")
	}

	epsilon := lm.PrivacyUsage()
	if epsilon != 1.0 {
		t.Errorf("Expected epsilon 1.0, got %f", epsilon)
	}

	scale := lm.GetScale()
	if scale != 1.0 {
		t.Errorf("Expected scale 1.0, got %f", scale)
	}
}

func TestLaplaceMechanism_Bounded(t *testing.T) {
	config := LaplaceConfig{
		Epsilon:     1.0,
		Sensitivity: 1.0,
		Bounded:     true,
		LowerBound:  0.0,
		UpperBound:  100.0,
	}

	lm := NewLaplaceMechanism(config)

	for i := 0; i < 100; i++ {
		noisy := lm.AddNoise(50.0)
		if noisy < 0 || noisy > 100 {
			t.Errorf("Bounded value %f is outside bounds [0, 100]", noisy)
		}
	}
}

func TestLaplaceAccountant(t *testing.T) {
	la := NewLaplaceAccountant(10.0)

	err := la.Account(3.0)
	if err != nil {
		t.Error("First account should succeed")
	}

	remaining := la.GetRemainingBudget()
	if remaining != 7.0 {
		t.Errorf("Expected remaining budget 7.0, got %f", remaining)
	}

	err = la.Account(8.0)
	if err != ErrBudgetExceeded {
		t.Error("Second account should fail due to budget exceeded")
	}
}

func TestNoiseCalibrator(t *testing.T) {
	config := CalibrationConfig{
		MechanismType: GaussianCalibration,
		TargetEpsilon: 1.0,
		TargetDelta:   1e-5,
		Confidence:    0.99,
	}

	nc := NewNoiseCalibrator(config)

	sigma := nc.CalibrateSigma(1.0)
	if sigma <= 0 {
		t.Error("Sigma should be positive")
	}

	nc.SetTargetEpsilon(2.0)
	epsilon, _ := nc.GetPrivacyParameters()
	if epsilon != 2.0 {
		t.Errorf("Expected epsilon 2.0, got %f", epsilon)
	}
}

func TestRDPCalibrator(t *testing.T) {
	rc := NewRDPCalibrator(1.0, 10.0, 1.0)

	delta := 1e-5
	epsilon := rc.ComputeEpsilonFromRDP(delta)

	if epsilon <= 0 {
		t.Error("Epsilon should be positive")
	}

	computedDelta := rc.ComputeDeltaFromRDP(epsilon)
	if computedDelta < 0 {
		t.Error("Delta should be non-negative")
	}
}

func TestAdaptiveCalibrator(t *testing.T) {
	ac := NewAdaptiveCalibrator(1.0, 1e-5)

	epsilon1 := ac.GetCurrentEpsilon()
	ac.IncrementRound()
	epsilon2 := ac.GetCurrentEpsilon()

	if epsilon2 <= epsilon1 {
		t.Error("Epsilon should increase with rounds in default adaptation")
	}
}
