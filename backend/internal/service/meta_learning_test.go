package service

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMetaLearningSystem_Initialize(t *testing.T) {
	m := NewMetaLearningSystem()

	if m == nil {
		t.Fatal("Failed to create MetaLearningSystem instance")
	}

	if m.initialized {
		t.Error("Instance should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := m.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !m.initialized {
		t.Error("Instance should be initialized after Initialize() call")
	}
}

func TestMetaLearningSystem_RunMetaLearningPipeline(t *testing.T) {
	m := NewMetaLearningSystem()
	ctx := context.Background()

	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize MetaLearningSystem: %v", err)
	}

	fewShotTask := &FewShotTask{
		TaskID: "test_task",
		SupportSamples: []*FewShotSample{
			{SampleID: "s1", Features: make([]float64, 10), Label: "class1"},
			{SampleID: "s2", Features: make([]float64, 10), Label: "class2"},
		},
		QuerySamples: []*FewShotSample{
			{SampleID: "q1", Features: make([]float64, 10)},
		},
		Classes: []string{"class1", "class2"},
		NWay:    2,
		KShot:   1,
	}

	result, err := m.RunMetaLearningPipeline(ctx, fewShotTask, "adversarial", "image", "text", "continual_task")

	if err != nil {
		t.Errorf("RunMetaLearningPipeline() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("RunMetaLearningPipeline() returned nil result")
	}

	if result.Timestamp.IsZero() {
		t.Error("Result timestamp should not be zero")
	}
}

func TestMetaLearningSystem_GetAdaptationHistory(t *testing.T) {
	m := NewMetaLearningSystem()
	ctx := context.Background()

	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize MetaLearningSystem: %v", err)
	}

	fewShotTask := &FewShotTask{
		TaskID: "history_test",
		SupportSamples: []*FewShotSample{
			{SampleID: "s1", Features: make([]float64, 10), Label: "class1"},
		},
		QuerySamples:  []*FewShotSample{},
		Classes:       []string{"class1"},
	}

	m.RunMetaLearningPipeline(ctx, fewShotTask, "injection", "audio", "text", "task1")

	if len(m.adaptationHistory) != 1 {
		t.Errorf("Expected 1 adaptation record, got %d", len(m.adaptationHistory))
	}

	for id := range m.adaptationHistory {
		record, err := m.GetAdaptationHistory(ctx, id)
		if err != nil {
			t.Errorf("GetAdaptationHistory() returned error: %v", err)
		}
		if record == nil {
			t.Error("GetAdaptationHistory() returned nil")
		}
	}
}

func TestMetaLearningSystem_GetAdaptationHistory_NotFound(t *testing.T) {
	m := NewMetaLearningSystem()
	ctx := context.Background()

	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize MetaLearningSystem: %v", err)
	}

	_, err := m.GetAdaptationHistory(ctx, "non_existent_id")

	if err == nil {
		t.Error("GetAdaptationHistory() should return error for non-existent ID")
	}
}

func TestMetaLearningSystem_GetSystemMetrics(t *testing.T) {
	m := NewMetaLearningSystem()
	ctx := context.Background()

	if err := m.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize MetaLearningSystem: %v", err)
	}

	metrics := m.GetSystemMetrics(ctx)

	if metrics == nil {
		t.Fatal("GetSystemMetrics() returned nil")
	}

	if metrics.FewShotEpisodes != 0 {
		t.Errorf("Expected initial FewShotEpisodes to be 0, got %d", metrics.FewShotEpisodes)
	}

	if metrics.ActiveDetectors <= 0 {
		t.Error("Expected at least one active detector")
	}

	if metrics.TransferRules <= 0 {
		t.Error("Expected at least one transfer rule")
	}

	if metrics.Plasticity < 0 || metrics.Plasticity > 1 {
		t.Errorf("Plasticity %f out of range [0, 1]", metrics.Plasticity)
	}

	if metrics.Stability < 0 || metrics.Stability > 1 {
		t.Errorf("Stability %f out of range [0, 1]", metrics.Stability)
	}
}

func TestMetaLearningSystem_NotInitialized(t *testing.T) {
	m := NewMetaLearningSystem()
	ctx := context.Background()

	fewShotTask := &FewShotTask{
		TaskID:        "test",
		SupportSamples: []*FewShotSample{},
		QuerySamples:  []*FewShotSample{},
		Classes:       []string{},
	}

	_, err := m.RunMetaLearningPipeline(ctx, fewShotTask, "adversarial", "image", "text", "task1")

	if err == nil {
		t.Error("RunMetaLearningPipeline() should return error when not initialized")
	}
}

func TestFewShotLearner_Initialize(t *testing.T) {
	f := NewFewShotLearner()

	if f.initialized {
		t.Error("Learner should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := f.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !f.initialized {
		t.Error("Learner should be initialized after Initialize() call")
	}
}

func TestFewShotLearner_FewShotLearn(t *testing.T) {
	f := NewFewShotLearner()
	ctx := context.Background()

	if err := f.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize few-shot learner: %v", err)
	}

	task := &FewShotTask{
		TaskID: "fewshot_test",
		SupportSamples: []*FewShotSample{
			{SampleID: "s1", Features: make([]float64, 10), Label: "cat"},
			{SampleID: "s2", Features: make([]float64, 10), Label: "dog"},
		},
		QuerySamples: []*FewShotSample{
			{SampleID: "q1", Features: make([]float64, 10)},
			{SampleID: "q2", Features: make([]float64, 10)},
		},
		Classes: []string{"cat", "dog"},
		NWay:    2,
		KShot:   1,
	}

	result, err := f.FewShotLearn(ctx, task)

	if err != nil {
		t.Errorf("FewShotLearn() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("FewShotLearn() returned nil result")
	}

	if len(result.Predictions) != len(task.QuerySamples) {
		t.Errorf("Expected %d predictions, got %d", len(task.QuerySamples), len(result.Predictions))
	}

	if len(result.LearningCurve) == 0 {
		t.Error("Learning curve should not be empty")
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence %f out of range [0, 1]", result.Confidence)
	}
}

func TestFewShotLearner_GetEpisodeCount(t *testing.T) {
	f := NewFewShotLearner()
	ctx := context.Background()

	if err := f.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize few-shot learner: %v", err)
	}

	if f.GetEpisodeCount() != 0 {
		t.Error("Initial episode count should be 0")
	}

	for i := 0; i < 5; i++ {
		task := &FewShotTask{
			TaskID:        "ep_test",
			SupportSamples: []*FewShotSample{{SampleID: "s1", Features: make([]float64, 10), Label: "a"}},
			QuerySamples:  []*FewShotSample{},
			Classes:       []string{"a"},
		}
		f.FewShotLearn(ctx, task)
	}

	if f.GetEpisodeCount() != 5 {
		t.Errorf("Expected episode count 5, got %d", f.GetEpisodeCount())
	}
}

func TestRapidAttackAdapter_Initialize(t *testing.T) {
	ra := NewRapidAttackAdapter()

	if ra.initialized {
		t.Error("Adapter should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := ra.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !ra.initialized {
		t.Error("Adapter should be initialized after Initialize() call")
	}
}

func TestRapidAttackAdapter_RapidAdapt(t *testing.T) {
	ra := NewRapidAttackAdapter()
	ctx := context.Background()

	if err := ra.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize rapid attack adapter: %v", err)
	}

	samples := []*AttackSample{
		{SampleID: "a1", Pattern: "attack_1", IsAttack: true},
		{SampleID: "a2", Pattern: "attack_2", IsAttack: true},
	}

	result, err := ra.RapidAdapt(ctx, "adversarial", samples)

	if err != nil {
		t.Errorf("RapidAdapt() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("RapidAdapt() returned nil result")
	}

	if result.AttackType != "adversarial" {
		t.Errorf("Expected attack type 'adversarial', got '%s'", result.AttackType)
	}

	if result.DetectionRate < 0 || result.DetectionRate > 1 {
		t.Errorf("Detection rate %f out of range [0, 1]", result.DetectionRate)
	}

	if result.SuccessRate < 0 || result.SuccessRate > 1 {
		t.Errorf("Success rate %f out of range [0, 1]", result.SuccessRate)
	}

	if result.AdaptationSteps < 0 {
		t.Error("Adaptation steps should not be negative")
	}
}

func TestRapidAttackAdapter_GetDetector(t *testing.T) {
	ra := NewRapidAttackAdapter()
	ctx := context.Background()

	if err := ra.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize rapid attack adapter: %v", err)
	}

	detector, exists := ra.GetDetector("injection")

	if !exists {
		t.Error("Detector for 'injection' should exist")
	}

	if detector == nil {
		t.Error("GetDetector() returned nil")
	}

	_, exists = ra.GetDetector("non_existent_attack")
	if exists {
		t.Error("GetDetector() should return false for non-existent attack type")
	}
}

func TestMetaKnowledgeTransfer_Initialize(t *testing.T) {
	mkt := NewMetaKnowledgeTransfer()

	if mkt.initialized {
		t.Error("Transfer system should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := mkt.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !mkt.initialized {
		t.Error("Transfer system should be initialized after Initialize() call")
	}
}

func TestMetaKnowledgeTransfer_TransferKnowledge(t *testing.T) {
	mkt := NewMetaKnowledgeTransfer()
	ctx := context.Background()

	if err := mkt.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize meta knowledge transfer: %v", err)
	}

	result, err := mkt.TransferKnowledge(ctx, "image", "text")

	if err != nil {
		t.Errorf("TransferKnowledge() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("TransferKnowledge() returned nil result")
	}

	if result.SourceDomain != "image" {
		t.Errorf("Expected source domain 'image', got '%s'", result.SourceDomain)
	}

	if result.TargetDomain != "text" {
		t.Errorf("Expected target domain 'text', got '%s'", result.TargetDomain)
	}

	if result.TransferEfficiency < 0 || result.TransferEfficiency > 1 {
		t.Errorf("Transfer efficiency %f out of range [0, 1]", result.TransferEfficiency)
	}
}

func TestMetaKnowledgeTransfer_GetTransferRules(t *testing.T) {
	mkt := NewMetaKnowledgeTransfer()
	ctx := context.Background()

	if err := mkt.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize meta knowledge transfer: %v", err)
	}

	rules := mkt.GetTransferRules("image", "text")

	if rules == nil {
		t.Error("GetTransferRules() returned nil")
	}

	if len(rules) == 0 {
		t.Error("Should have at least one transfer rule for image->text")
	}
}

func TestMetaKnowledgeTransfer_UpdateKnowledgeGraph(t *testing.T) {
	mkt := NewMetaKnowledgeTransfer()
	ctx := context.Background()

	if err := mkt.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize meta knowledge transfer: %v", err)
	}

	err := mkt.UpdateKnowledgeGraph("image", "text", 0.5)

	if err != nil {
		t.Errorf("UpdateKnowledgeGraph() returned error: %v", err)
	}

	if len(mkt.knowledgeGraph.Edges) == 0 {
		t.Error("Should have at least one edge after update")
	}
}

func TestContinualLearningSystem_Initialize(t *testing.T) {
	cl := NewContinualLearningSystem()

	if cl.initialized {
		t.Error("System should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := cl.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !cl.initialized {
		t.Error("System should be initialized after Initialize() call")
	}
}

func TestContinualLearningSystem_AddTask(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	task := &LearningTaskMeta{
		TaskID:     "task_1",
		TaskType:   "classification",
		Complexity: 0.5,
		DataSize:   1000,
		LabelSpace: []string{"class1", "class2"},
	}

	err := cl.AddTask(ctx, task)

	if err != nil {
		t.Errorf("AddTask() returned error: %v", err)
	}

	if cl.GetTaskCount() != 1 {
		t.Errorf("Expected task count 1, got %d", cl.GetTaskCount())
	}
}

func TestContinualLearningSystem_LearnContinually(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	task := &LearningTaskMeta{
		TaskID:     "learning_task",
		TaskType:   "detection",
		Complexity: 0.6,
		DataSize:   2000,
		LabelSpace: []string{"attack", "normal"},
	}

	cl.AddTask(ctx, task)

	result, err := cl.LearnContinually(ctx, "learning_task")

	if err != nil {
		t.Errorf("LearnContinually() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("LearnContinually() returned nil result")
	}

	if result.LearningProgress < 0 || result.LearningProgress > 1 {
		t.Errorf("Learning progress %f out of range [0, 1]", result.LearningProgress)
	}

	if result.KnowledgeRetention < 0 || result.KnowledgeRetention > 1 {
		t.Errorf("Knowledge retention %f out of range [0, 1]", result.KnowledgeRetention)
	}

	if result.AdaptationSpeed < 0 || result.AdaptationSpeed > 1 {
		t.Errorf("Adaptation speed %f out of range [0, 1]", result.AdaptationSpeed)
	}
}

func TestContinualLearningSystem_UpdatePlasticityStability(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	initialPlasticity := cl.plasticity
	initialStability := cl.stability

	cl.UpdatePlasticityStability(0.1)

	if cl.stability <= initialStability {
		t.Error("Stability should increase with positive performance delta")
	}

	if cl.plasticity >= initialPlasticity {
		t.Error("Plasticity should decrease with positive performance delta")
	}
}

func TestContinualLearningSystem_GetForgettingMetrics(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	task := &LearningTaskMeta{
		TaskID:     "metrics_task",
		TaskType:   "classification",
		Complexity: 0.4,
		DataSize:   1500,
	}

	cl.AddTask(ctx, task)
	cl.LearnContinually(ctx, "metrics_task")

	metrics := cl.GetForgettingMetrics()

	if metrics == nil {
		t.Fatal("GetForgettingMetrics() returned nil")
	}

	if len(metrics.PerformanceHistory) == 0 {
		t.Error("Performance history should not be empty after learning")
	}
}

func TestMetaLearningResult_GetOverallPerformance(t *testing.T) {
	result := &MetaLearningResult{
		Timestamp: time.Now(),
		FewShotLearning: &FewShotLearningResult{
			Confidence: 0.8,
		},
		RapidAdaptation: &RapidAdaptationResult{
			SuccessRate: 0.9,
		},
		KnowledgeTransfer: &KnowledgeTransferResult{
			TransferEfficiency: 0.7,
		},
		ContinualLearning: &ContinualLearningResult{
			PerformanceGain: 0.85,
		},
	}

	performance := result.GetOverallPerformance()

	expected := (0.8*100 + 0.9*100 + 0.7*100 + 0.85*100) / 4

	if performance != expected {
		t.Errorf("Expected overall performance %f, got %f", expected, performance)
	}
}

func TestMetaLearningResult_GetOverallPerformance_Empty(t *testing.T) {
	result := &MetaLearningResult{
		Timestamp: time.Now(),
	}

	performance := result.GetOverallPerformance()

	if performance != 0.0 {
		t.Errorf("Expected 0.0 for empty result, got %f", performance)
	}
}

func TestFewShotLearner_SimulateLearningCurve(t *testing.T) {
	f := NewFewShotLearner()

	for _, supportSize := range []int{1, 5, 10, 20} {
		curve := f.simulateLearningCurve(supportSize)

		if len(curve) != supportSize {
			t.Errorf("Support size %d: expected curve length %d, got %d", supportSize, supportSize, len(curve))
		}

		for i, score := range curve {
			if score < 0 || score > 1 {
				t.Errorf("Support size %d, index %d: score %f out of range [0, 1]", supportSize, i, score)
			}
		}
	}
}

func TestRapidAttackAdapter_MultipleAdaptations(t *testing.T) {
	ra := NewRapidAttackAdapter()
	ctx := context.Background()

	if err := ra.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize rapid attack adapter: %v", err)
	}

	attackTypes := []string{"adversarial", "injection", "evasion"}

	for _, attackType := range attackTypes {
		samples := []*AttackSample{
			{SampleID: "s1", Pattern: fmt.Sprintf("pattern_%s", attackType), IsAttack: true},
		}

		result, err := ra.RapidAdapt(ctx, attackType, samples)

		if err != nil {
			t.Errorf("RapidAdapt(%s) returned error: %v", attackType, err)
		}

		if result == nil {
			t.Errorf("RapidAdapt(%s) returned nil", attackType)
		}
	}
}

func TestContinualLearningSystem_MultipleTasks(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	tasks := []*LearningTaskMeta{
		{TaskID: "task_1", TaskType: "classification", Complexity: 0.3, DataSize: 1000},
		{TaskID: "task_2", TaskType: "detection", Complexity: 0.5, DataSize: 1500},
		{TaskID: "task_3", TaskType: "generation", Complexity: 0.7, DataSize: 2000},
	}

	for _, task := range tasks {
		cl.AddTask(ctx, task)
		cl.LearnContinually(ctx, task.TaskID)
	}

	if cl.GetTaskCount() != 3 {
		t.Errorf("Expected task count 3, got %d", cl.GetTaskCount())
	}
}

func TestMetaKnowledgeTransfer_MultipleTransfers(t *testing.T) {
	mkt := NewMetaKnowledgeTransfer()
	ctx := context.Background()

	if err := mkt.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize meta knowledge transfer: %v", err)
	}

	transfers := []struct {
		source, target string
	}{
		{"image", "text"},
		{"audio", "text"},
		{"tabular", "image"},
	}

	for _, transfer := range transfers {
		result, err := mkt.TransferKnowledge(ctx, transfer.source, transfer.target)

		if err != nil {
			t.Errorf("TransferKnowledge(%s->%s) returned error: %v", transfer.source, transfer.target, err)
		}

		if result != nil && (result.TransferEfficiency < 0 || result.TransferEfficiency > 1) {
			t.Errorf("Transfer %s->%s: efficiency %f out of range", transfer.source, transfer.target, result.TransferEfficiency)
		}
	}
}

func TestFewShotTask_EmptyClasses(t *testing.T) {
	f := NewFewShotLearner()
	ctx := context.Background()

	if err := f.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize few-shot learner: %v", err)
	}

	task := &FewShotTask{
		TaskID:        "empty_test",
		SupportSamples: []*FewShotSample{},
		QuerySamples:  []*FewShotSample{},
		Classes:       []string{},
	}

	result, err := f.FewShotLearn(ctx, task)

	if err != nil {
		t.Errorf("FewShotLearn() returned error: %v", err)
	}

	if result != nil && len(result.Predictions) > 0 {
		for _, pred := range result.Predictions {
			if pred != "unknown" {
				t.Error("Prediction for empty classes should be 'unknown'")
			}
		}
	}
}

func TestContinualLearningSystem_NonExistentTask(t *testing.T) {
	cl := NewContinualLearningSystem()
	ctx := context.Background()

	if err := cl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize continual learning system: %v", err)
	}

	_, err := cl.LearnContinually(ctx, "non_existent_task")

	if err == nil {
		t.Error("LearnContinually() should return error for non-existent task")
	}
}
