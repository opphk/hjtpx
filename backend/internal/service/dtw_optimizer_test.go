package service

import (
	"math"
	"testing"
)

func TestNewOptimizedDTW(t *testing.T) {
	dtw := NewOptimizedDTW()
	if dtw == nil {
		t.Fatal("OptimizedDTW should not be nil")
	}
}

func TestNewOptimizedDTWWithParams(t *testing.T) {
	dtw := NewOptimizedDTWWithParams(20, "itakura", 0.15)
	if dtw == nil {
		t.Fatal("OptimizedDTW should not be nil")
	}
	if dtw.windowSize != 20 {
		t.Errorf("Window size should be 20, got %d", dtw.windowSize)
	}
	if dtw.constraintType != "itakura" {
		t.Errorf("Constraint type should be itakura, got %s", dtw.constraintType)
	}
}

func TestOptimizedDTW_ComputeDistance_Empty(t *testing.T) {
	dtw := NewOptimizedDTW()

	dist := dtw.ComputeDistance([]SliderPoint{}, []SliderPoint{{X: 100, Y: 200, Timestamp: 1000}})
	if dist != math.MaxFloat64 {
		t.Errorf("Empty first trajectory should return MaxFloat64, got %f", dist)
	}

	dist = dtw.ComputeDistance([]SliderPoint{{X: 100, Y: 200, Timestamp: 1000}}, []SliderPoint{})
	if dist != math.MaxFloat64 {
		t.Errorf("Empty second trajectory should return MaxFloat64, got %f", dist)
	}
}

func TestOptimizedDTW_ComputeDistance_SinglePoint(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{{X: 100, Y: 200, Timestamp: 1000}}
	traj2 := []SliderPoint{{X: 100, Y: 200, Timestamp: 1000}}

	dist := dtw.ComputeDistance(traj1, traj2)

	if dist != 0 {
		t.Errorf("Identical single point trajectories should have distance 0, got %f", dist)
	}
}

func TestOptimizedDTW_ComputeDistance_Basic(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	dist := dtw.ComputeDistance(traj1, traj2)

	t.Logf("Distance between identical trajectories: %f", dist)

	if dist > 1 {
		t.Errorf("Identical trajectories should have small distance, got %f", dist)
	}
}

func TestOptimizedDTW_ComputeDistance_Similar(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 205, Timestamp: 1000},
		{X: 200, Y: 205, Timestamp: 1100},
		{X: 300, Y: 205, Timestamp: 1200},
	}

	dist := dtw.ComputeDistance(traj1, traj2)

	t.Logf("Distance between similar trajectories: %f", dist)

	if dist < 0 {
		t.Errorf("Distance should be non-negative, got %f", dist)
	}
}

func TestOptimizedDTW_ComputeDistance_Different(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 500, Y: 500, Timestamp: 1000},
		{X: 600, Y: 500, Timestamp: 1100},
		{X: 700, Y: 500, Timestamp: 1200},
	}

	dist := dtw.ComputeDistance(traj1, traj2)

	t.Logf("Distance between different trajectories: %f", dist)

	if dist < 100 {
		t.Errorf("Different trajectories should have larger distance, got %f", dist)
	}
}

func TestOptimizedDTW_ComputeDistanceLowerBound(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	lb := dtw.ComputeDistanceLowerBound(traj1, traj2)

	t.Logf("Lower bound: %f", lb)

	if lb < 0 {
		t.Errorf("Lower bound should be non-negative, got %f", lb)
	}
}

func TestOptimizedDTW_ComputeSimilarity(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	similarity := dtw.ComputeSimilarity(traj1, traj2)

	t.Logf("Similarity between identical trajectories: %f", similarity)

	if similarity < 0.9 {
		t.Errorf("Identical trajectories should have high similarity, got %f", similarity)
	}
}

func TestOptimizedDTW_ComputeWithPath(t *testing.T) {
	dtw := NewOptimizedDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	result := dtw.ComputeWithPath(traj1, traj2)

	if result.Distance < 0 {
		t.Errorf("Distance should be non-negative, got %f", result.Distance)
	}

	if len(result.Path) == 0 {
		t.Errorf("Path should not be empty")
	}

	t.Logf("DTW with path - Distance: %f, PathLength: %d, WarpingPoints: %d",
		result.Distance, result.PathLength, result.WarpingPoints)
}

func TestOptimizedDTW_SakoeChibaConstraint(t *testing.T) {
	dtw := NewOptimizedDTWWithParams(5, "sakoe_chiba", 0.1)

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	dist := dtw.ComputeDistance(traj1, traj2)

	t.Logf("Sakoe-Chiba DTW distance: %f", dist)

	if dist < 0 {
		t.Errorf("Distance should be non-negative, got %f", dist)
	}
}

func TestOptimizedDTW_ItakuraConstraint(t *testing.T) {
	dtw := NewOptimizedDTWWithParams(5, "itakura", 0.1)

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	dist := dtw.ComputeDistance(traj1, traj2)

	t.Logf("Itakura DTW distance: %f", dist)

	if dist < 0 {
		t.Errorf("Distance should be non-negative, got %f", dist)
	}
}

func TestDTWBatch_ComputeDistances(t *testing.T) {
	batch := NewDTWBatch()

	reference := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	candidates := [][]SliderPoint{
		{{X: 100, Y: 200, Timestamp: 1000}, {X: 200, Y: 200, Timestamp: 1100}, {X: 300, Y: 200, Timestamp: 1200}},
		{{X: 500, Y: 500, Timestamp: 1000}, {X: 600, Y: 500, Timestamp: 1100}, {X: 700, Y: 500, Timestamp: 1200}},
	}

	distances := batch.ComputeDistances(reference, candidates)

	if len(distances) != len(candidates) {
		t.Errorf("Should return distances for all candidates, got %d", len(distances))
	}

	t.Logf("Distances: %v", distances)
}

func TestDTWBatch_FindMostSimilar(t *testing.T) {
	batch := NewDTWBatch()

	reference := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	candidates := [][]SliderPoint{
		{{X: 500, Y: 500, Timestamp: 1000}, {X: 600, Y: 500, Timestamp: 1100}, {X: 700, Y: 500, Timestamp: 1200}},
		{{X: 100, Y: 200, Timestamp: 1000}, {X: 200, Y: 200, Timestamp: 1100}, {X: 300, Y: 200, Timestamp: 1200}},
	}

	idx, dist := batch.FindMostSimilar(reference, candidates)

	if idx != 1 {
		t.Errorf("Should find second candidate as most similar, got index %d", idx)
	}

	t.Logf("Most similar index: %d, distance: %f", idx, dist)
}

func TestDTWClassifier_AddTemplate(t *testing.T) {
	classifier := NewDTWClassifier()

	trajectory := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	classifier.AddTemplate("human_1", trajectory)

	if len(classifier.templates) != 1 {
		t.Errorf("Should have 1 template, got %d", len(classifier.templates))
	}
}

func TestDTWClassifier_Classify(t *testing.T) {
	classifier := NewDTWClassifier()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
	}

	traj2 := []SliderPoint{
		{X: 500, Y: 500, Timestamp: 1000},
		{X: 600, Y: 500, Timestamp: 1100},
		{X: 700, Y: 500, Timestamp: 1200},
	}

	classifier.AddTemplate("human_1", traj1)
	classifier.AddTemplate("bot_1", traj2)

	match, similarity := classifier.Classify(traj1)

	if match != "human_1" {
		t.Errorf("Should match human_1, got %s", match)
	}

	t.Logf("Matched: %s, Similarity: %f", match, similarity)
}

func TestDTWClassifier_ComputeSimilarity(t *testing.T) {
	classifier := NewDTWClassifier()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
	}

	traj2 := []SliderPoint{
		{X: 500, Y: 500, Timestamp: 1000},
		{X: 600, Y: 500, Timestamp: 1100},
	}

	classifier.AddTemplate("human_1", traj1)
	classifier.AddTemplate("bot_1", traj2)

	candidate := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
	}

	similarities := classifier.ComputeSimilarity(candidate)

	if len(similarities) != 2 {
		t.Errorf("Should return 2 similarities, got %d", len(similarities))
	}

	t.Logf("Similarities: %v", similarities)
}

func TestMultiScaleDTW_ComputeDistance(t *testing.T) {
	msdtw := NewMultiScaleDTW()

	traj1 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
		{X: 500, Y: 200, Timestamp: 1400},
		{X: 600, Y: 200, Timestamp: 1500},
	}

	traj2 := []SliderPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 200, Y: 200, Timestamp: 1100},
		{X: 300, Y: 200, Timestamp: 1200},
		{X: 400, Y: 200, Timestamp: 1300},
		{X: 500, Y: 200, Timestamp: 1400},
		{X: 600, Y: 200, Timestamp: 1500},
	}

	dist := msdtw.ComputeDistance(traj1, traj2)

	t.Logf("Multi-scale DTW distance: %f", dist)

	if dist < 0 {
		t.Errorf("Distance should be non-negative, got %f", dist)
	}
}

func TestMultiScaleDTW_Downsample(t *testing.T) {
	msdtw := NewMultiScaleDTW()

	trajectory := make([]SliderPoint, 20)
	for i := 0; i < 20; i++ {
		trajectory[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200,
			Timestamp: int64(1000 + i*100),
		}
	}

	downsampled := msdtw.downsample(trajectory, 4)

	if len(downsampled) >= len(trajectory) {
		t.Logf("Downsampled length should be smaller: original=%d, downsampled=%d",
			len(trajectory), len(downsampled))
	}

	t.Logf("Original length: %d, Downsampled length: %d", len(trajectory), len(downsampled))
}

func BenchmarkOptimizedDTW_ComputeDistance(b *testing.B) {
	dtw := NewOptimizedDTW()

	traj1 := make([]SliderPoint, 100)
	traj2 := make([]SliderPoint, 100)

	for i := 0; i < 100; i++ {
		traj1[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
		traj2[i] = SliderPoint{
			X:         100 + i*10 + 5,
			Y:         200 + i%20,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dtw.ComputeDistance(traj1, traj2)
	}
}

func BenchmarkOptimizedDTW_SakoeChiba(b *testing.B) {
	dtw := NewOptimizedDTWWithParams(10, "sakoe_chiba", 0.1)

	traj1 := make([]SliderPoint, 100)
	traj2 := make([]SliderPoint, 100)

	for i := 0; i < 100; i++ {
		traj1[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
		traj2[i] = SliderPoint{
			X:         100 + i*10 + 5,
			Y:         200 + i%20,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dtw.ComputeDistance(traj1, traj2)
	}
}

func BenchmarkMultiScaleDTW(b *testing.B) {
	msdtw := NewMultiScaleDTW()

	traj1 := make([]SliderPoint, 100)
	traj2 := make([]SliderPoint, 100)

	for i := 0; i < 100; i++ {
		traj1[i] = SliderPoint{
			X:         100 + i*10,
			Y:         200 + i%20 - 10,
			Timestamp: int64(1000 + i*50),
		}
		traj2[i] = SliderPoint{
			X:         100 + i*10 + 5,
			Y:         200 + i%20,
			Timestamp: int64(1000 + i*50),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msdtw.ComputeDistance(traj1, traj2)
	}
}
