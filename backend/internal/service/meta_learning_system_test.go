package service

import (
	"context"
	"testing"
	"time"
)

func TestMetaLearningVerificationSystem(t *testing.T) {
	system := NewMetaLearningVerificationSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestMetaFewShotLearner(t *testing.T) {
	learner := NewMetaFewShotLearner()
	ctx := context.Background()

	if err := learner.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize learner: %v", err)
	}

	if !learner.initialized {
		t.Error("Learner should be initialized")
	}
}

func TestMetaFewShotLearner_FewShotLearn(t *testing.T) {
	learner := NewMetaFewShotLearner()
	ctx := context.Background()

	if err := learner.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize learner: %v", err)
	}

	task := &MetaFewShotTask{
		TaskID:   "test-task-1",
		Classes:  []string{"class1", "class2", "class3"},
		NWay:     3,
		KShot:    5,
		Complexity: 0.5,
	}

	supportSamples := make([]*MetaFewShotSample, 0, 15)
	for i := 0; i < 15; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i % 10)
		}
		supportSamples = append(supportSamples, &MetaFewShotSample{
			SampleID:  "sample_" + string(rune('0'+i)),
			Features:  features,
			Label:     task.Classes[i%3],
			IsSupport: true,
		})
	}
	task.SupportSamples = supportSamples

	querySamples := make([]*MetaFewShotSample, 0, 5)
	for i := 0; i < 5; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i % 10)
		}
		querySamples = append(querySamples, &MetaFewShotSample{
			SampleID: "query_" + string(rune('0'+i)),
			Features: features,
			Label:    task.Classes[i%3],
		})
	}
	task.QuerySamples = querySamples

	result, err := learner.FewShotLearn(ctx, task)
	if err != nil {
		t.Fatalf("Failed to learn from few-shot task: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.TaskID != task.TaskID {
		t.Errorf("Expected task ID %s, got %s", task.TaskID, result.TaskID)
	}

	if len(result.Predictions) != len(task.QuerySamples) {
		t.Errorf("Expected %d predictions, got %d", len(task.QuerySamples), len(result.Predictions))
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}

	if result.AdaptationSteps != len(task.SupportSamples) {
		t.Errorf("Expected %d adaptation steps, got %d", len(task.SupportSamples), result.AdaptationSteps)
	}

	if len(result.LearningCurve) != len(task.SupportSamples) {
		t.Errorf("Learning curve length should match support samples, expected %d, got %d",
			len(task.SupportSamples), len(result.LearningCurve))
	}
}

func TestMetaRapidAttackAdapter(t *testing.T) {
	adapter := NewMetaRapidAttackAdapter()
	ctx := context.Background()

	if err := adapter.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	if !adapter.initialized {
		t.Error("Adapter should be initialized")
	}

	if len(adapter.attackTypes) == 0 {
		t.Error("Adapter should have attack types")
	}

	if len(adapter.detectors) == 0 {
		t.Error("Adapter should have detectors")
	}
}

func TestMetaRapidAttackAdapter_RapidAdapt(t *testing.T) {
	adapter := NewMetaRapidAttackAdapter()
	ctx := context.Background()

	if err := adapter.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	attackType := "adversarial"

	samples := make([]*MetaAttackSample, 5)
	for i := 0; i < 5; i++ {
		samples[i] = &MetaAttackSample{
			SampleID: "attack_" + string(rune('0'+i)),
			Pattern:  "adversarial_pattern_" + string(rune('0'+i)),
			IsAttack: true,
			Features: map[string]float64{"magnitude": 0.8, "frequency": 0.5},
		}
	}

	result, err := adapter.RapidAdapt(ctx, attackType, samples)
	if err != nil {
		t.Fatalf("Failed to adapt to attack: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.AttackType != attackType {
		t.Errorf("Expected attack type %s, got %s", attackType, result.AttackType)
	}

	if result.AdaptationSteps <= 0 {
		t.Error("Should have at least one adaptation step")
	}

	if result.DetectionRate < 0 || result.DetectionRate > 1 {
		t.Errorf("Detection rate should be between 0 and 1, got %f", result.DetectionRate)
	}

	if result.ThreatLevel == "" {
		t.Error("Threat level should be set")
	}
}

func TestMetaKnowledgeTransferEngine(t *testing.T) {
	engine := NewMetaKnowledgeTransferEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized")
	}

	if len(engine.transferRules) == 0 {
		t.Error("Engine should have transfer rules")
	}
}

func TestMetaKnowledgeTransferEngine_TransferKnowledge(t *testing.T) {
	engine := NewMetaKnowledgeTransferEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	tests := []struct {
		name         string
		sourceDomain string
		targetDomain string
	}{
		{
			name:         "Image to text transfer",
			sourceDomain: "image",
			targetDomain: "text",
		},
		{
			name:         "Text to audio transfer",
			sourceDomain: "text",
			targetDomain: "audio",
		},
		{
			name:         "Audio to tabular transfer",
			sourceDomain: "audio",
			targetDomain: "tabular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.TransferKnowledge(ctx, tt.sourceDomain, tt.targetDomain)
			if err != nil {
				t.Fatalf("Failed to transfer knowledge: %v", err)
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.SourceDomain != tt.sourceDomain {
				t.Errorf("Expected source domain %s, got %s", tt.sourceDomain, result.SourceDomain)
			}

			if result.TargetDomain != tt.targetDomain {
				t.Errorf("Expected target domain %s, got %s", tt.targetDomain, result.TargetDomain)
			}

			if result.TransferEfficiency < 0 || result.TransferEfficiency > 1 {
				t.Errorf("Transfer efficiency should be between 0 and 1, got %f", result.TransferEfficiency)
			}
		})
	}
}

func TestMetaContinualLearner(t *testing.T) {
	learner := NewMetaContinualLearner()
	ctx := context.Background()

	if err := learner.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize learner: %v", err)
	}

	if !learner.initialized {
		t.Error("Learner should be initialized")
	}
}

func TestMetaContinualLearner_LearnContinually(t *testing.T) {
	learner := NewMetaContinualLearner()
	ctx := context.Background()

	if err := learner.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize learner: %v", err)
	}

	task := &MetaLearningTask{
		TaskID:     "continual-task-1",
		TaskType:   "classification",
		Complexity: 0.6,
		DataSize:   1000,
		LabelSpace: []string{"class1", "class2", "class3"},
	}

	err := learner.AddTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	result, err := learner.LearnContinually(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("Failed to learn continually: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.TaskID != task.TaskID {
		t.Errorf("Expected task ID %s, got %s", task.TaskID, result.TaskID)
	}

	if result.LearningProgress < 0 || result.LearningProgress > 1 {
		t.Errorf("Learning progress should be between 0 and 1, got %f", result.LearningProgress)
	}

	if result.KnowledgeRetention < 0 || result.KnowledgeRetention > 1 {
		t.Errorf("Knowledge retention should be between 0 and 1, got %f", result.KnowledgeRetention)
	}
}

func TestMetaLearningVerificationSystem_RunPipeline(t *testing.T) {
	system := NewMetaLearningVerificationSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	fewShotTask := &MetaFewShotTask{
		TaskID:   "pipeline-test-task",
		Classes:  []string{"class1", "class2"},
		NWay:     2,
		KShot:    3,
		Complexity: 0.5,
	}

	supportSamples := make([]*MetaFewShotSample, 6)
	for i := 0; i < 6; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		supportSamples[i] = &MetaFewShotSample{
			SampleID:  "sample_" + string(rune('0'+i)),
			Features:  features,
			Label:     fewShotTask.Classes[i%2],
			IsSupport: true,
		}
	}
	fewShotTask.SupportSamples = supportSamples

	querySamples := make([]*MetaFewShotSample, 2)
	for i := 0; i < 2; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		querySamples[i] = &MetaFewShotSample{
			SampleID: "query_" + string(rune('0'+i)),
			Features: features,
			Label:    fewShotTask.Classes[i%2],
		}
	}
	fewShotTask.QuerySamples = querySamples

	attackType := "adversarial"
	sourceDomain := "image"
	targetDomain := "text"
	continualTaskID := "continual-task-1"

	continualTask := &MetaLearningTask{
		TaskID:     continualTaskID,
		TaskType:   "detection",
		Complexity: 0.4,
		DataSize:   500,
	}
	system.continualLearner.taskStream = append(system.continualLearner.taskStream, continualTask)

	result, err := system.RunMetaLearningPipeline(ctx, fewShotTask, attackType, sourceDomain, targetDomain, continualTaskID)
	if err != nil {
		t.Fatalf("Failed to run pipeline: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.OverallPerformance < 0 || result.OverallPerformance > 100 {
		t.Errorf("Overall performance should be between 0 and 100, got %f", result.OverallPerformance)
	}

	if result.FewShotLearning == nil {
		t.Error("Few-shot learning result should not be nil")
	}

	if result.RapidAdaptation == nil {
		t.Error("Rapid adaptation result should not be nil")
	}

	if result.KnowledgeTransfer == nil {
		t.Error("Knowledge transfer result should not be nil")
	}

	if result.ContinualLearning == nil {
		t.Error("Continual learning result should not be nil")
	}
}

func TestMetaLearningVerificationSystem_GetSystemMetrics(t *testing.T) {
	system := NewMetaLearningVerificationSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	fewShotTask := &MetaFewShotTask{
		TaskID:   "metrics-test-task",
		Classes:  []string{"class1", "class2"},
		NWay:     2,
		KShot:    2,
		Complexity: 0.5,
	}

	supportSamples := make([]*MetaFewShotSample, 4)
	for i := 0; i < 4; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		supportSamples[i] = &MetaFewShotSample{
			SampleID:  "sample_" + string(rune('0'+i)),
			Features:  features,
			Label:     fewShotTask.Classes[i%2],
			IsSupport: true,
		}
	}
	fewShotTask.SupportSamples = supportSamples

	querySamples := make([]*MetaFewShotSample, 2)
	for i := 0; i < 2; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		querySamples[i] = &MetaFewShotSample{
			SampleID: "query_" + string(rune('0'+i)),
			Features: features,
			Label:    fewShotTask.Classes[i%2],
		}
	}
	fewShotTask.QuerySamples = querySamples

	_, err := system.RunMetaLearningPipeline(ctx, fewShotTask, "adversarial", "image", "text", "task-1")
	if err != nil {
		t.Fatalf("Failed to run pipeline: %v", err)
	}

	metrics := system.GetSystemMetrics(ctx)

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.TotalAdaptations < 1 {
		t.Errorf("Should have at least 1 adaptation, got %d", metrics.TotalAdaptations)
	}
}

func TestMetaLearningVerificationSystem_GetAdaptationHistory(t *testing.T) {
	system := NewMetaLearningVerificationSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	fewShotTask := &MetaFewShotTask{
		TaskID:   "history-test-task",
		Classes:  []string{"class1", "class2"},
		NWay:     2,
		KShot:    2,
		Complexity: 0.5,
	}

	supportSamples := make([]*MetaFewShotSample, 4)
	for i := 0; i < 4; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		supportSamples[i] = &MetaFewShotSample{
			SampleID:  "sample_" + string(rune('0'+i)),
			Features:  features,
			Label:     fewShotTask.Classes[i%2],
			IsSupport: true,
		}
	}
	fewShotTask.SupportSamples = supportSamples

	querySamples := make([]*MetaFewShotSample, 2)
	for i := 0; i < 2; i++ {
		features := make([]float64, 64)
		for j := range features {
			features[j] = float64(i)
		}
		querySamples[i] = &MetaFewShotSample{
			SampleID: "query_" + string(rune('0'+i)),
			Features: features,
			Label:    fewShotTask.Classes[i%2],
		}
	}
	fewShotTask.QuerySamples = querySamples

	result, err := system.RunMetaLearningPipeline(ctx, fewShotTask, "adversarial", "image", "text", "task-1")
	if err != nil {
		t.Fatalf("Failed to run pipeline: %v", err)
	}

	pipelineResult, ok := result.(*MetaLearningPipelineResult)
	if !ok {
		t.Fatal("Result should be of type *MetaLearningPipelineResult")
	}

	recordID := ""
	for id := range system.adaptationHistory {
		recordID = id
		break
	}

	if recordID == "" {
		t.Fatal("Should have at least one adaptation record")
	}

	record, err := system.GetAdaptationHistory(ctx, recordID)
	if err != nil {
		t.Fatalf("Failed to get adaptation history: %v", err)
	}

	if record == nil {
		t.Fatal("Record should not be nil")
	}

	if record.TaskType != pipelineResult.FewShotLearning.TaskID {
		t.Errorf("Expected task type %s, got %s", pipelineResult.FewShotLearning.TaskID, record.TaskType)
	}

	_, err = system.GetAdaptationHistory(ctx, "non-existent-id")
	if err == nil {
		t.Error("Should return error for non-existent record")
	}
}

func TestMetaLearningSystemCalculateOverallPerformance(t *testing.T) {
	system := NewMetaLearningVerificationSystem()

	result := &MetaLearningPipelineResult{
		Timestamp: time.Now(),
		FewShotLearning: &MetaFewShotResult{
			Confidence: 0.8,
		},
		RapidAdaptation: &MetaRapidAdaptationResult{
			SuccessRate: 0.9,
		},
		KnowledgeTransfer: &MetaKnowledgeTransferResult{
			TransferEfficiency: 0.75,
		},
		ContinualLearning: &MetaContinualLearningResult{
			PerformanceGain: 0.85,
		},
	}

	performance := system.calculateOverallPerformance(result)

	expectedPerformance := (0.8*100 + 0.9*100 + 0.75*100 + 0.85*100) / 4.0

	if performance != expectedPerformance {
		t.Errorf("Expected performance %f, got %f", expectedPerformance, performance)
	}
}

func TestMetaLearningSystemUpdateSystemMetrics(t *testing.T) {
	system := NewMetaLearningVerificationSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	record := &MetaAdaptationRecord{
		ID:             "test-record-1",
		Timestamp:      time.Now(),
		TaskType:       "test-task",
		AdaptationTime: time.Second * 5,
		Performance:    0.85,
		LearningMode:  "test",
	}

	system.updateSystemMetrics(record)

	if system.systemMetrics.TotalAdaptations != 1 {
		t.Errorf("Expected 1 total adaptation, got %d", system.systemMetrics.TotalAdaptations)
	}

	if system.systemMetrics.AveragePerformance != 0.85 {
		t.Errorf("Expected average performance 0.85, got %f", system.systemMetrics.AveragePerformance)
	}
}
