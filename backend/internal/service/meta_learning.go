package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type MetaLearningSystem struct {
	mu                    sync.RWMutex
	initialized           bool
	fewShotLearner        *FewShotLearner
	rapidAdapter          *RapidAttackAdapter
	metaKnowledgeTransfer *MetaKnowledgeTransfer
	continualLearner      *ContinualLearningSystem
	adaptationHistory     map[string]*AdaptationRecord
}

type AdaptationRecord struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	TaskType      string                 `json:"task_type"`
	AdaptationTime time.Duration         `json:"adaptation_time"`
	Performance   float64                `json:"performance"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type FewShotLearner struct {
	mu          sync.RWMutex
	initialized bool
	modelPrior  *ModelPrior
	supportSets map[string]*SupportSet
	episodeCount int
}

type ModelPrior struct {
	BaseParameters []float64
	Variance      float64
	KnowledgeBase []PriorKnowledge
}

type PriorKnowledge struct {
	ConceptID    string
	ConceptName  string
	Parameters   map[string]float64
	Confidence   float64
}

type SupportSet struct {
	TaskID       string
	SupportSize  int
	QuerySize    int
	Classes      []string
	Samples      []*FewShotSample
}

type FewShotSample struct {
	SampleID    string
	Features    []float64
	Label       string
	IsSupport   bool
}

type FewShotLearningResult struct {
	TaskID          string                  `json:"task_id"`
	Predictions     []string                `json:"predictions"`
	Confidence      float64                 `json:"confidence"`
	LearningCurve  []float64               `json:"learning_curve"`
	AdaptationSteps int                     `json:"adaptation_steps"`
}

type RapidAttackAdapter struct {
	mu          sync.RWMutex
	initialized bool
	attackTypes []string
	detectors   map[string]*AttackDetector
	adaptationPolicy *AdaptationPolicy
}

type AttackDetector struct {
	DetectorID   string
	AttackType   string
	DetectionRate float64
	FalsePositiveRate float64
	AdaptationHistory []*AdaptationStep
}

type AdaptationStep struct {
	StepID        int
	InputPattern  string
	Detected      bool
	Adapted       bool
	Timestamp     time.Time
}

type AdaptationPolicy struct {
	PolicyID      string
	LearningRate  float64
	Threshold     float64
	MaxIterations int
}

type RapidAdaptationResult struct {
	AttackType       string                  `json:"attack_type"`
	DetectionRate    float64                 `json:"detection_rate"`
	AdaptationSteps  int                     `json:"adaptation_steps"`
	AdaptationTime   time.Duration           `json:"adaptation_time"`
	SuccessRate      float64                 `json:"success_rate"`
}

type MetaKnowledgeTransfer struct {
	mu          sync.RWMutex
	initialized bool
	sourceDomains []string
	targetDomains []string
	transferRules []*TransferRule
	knowledgeGraph *DomainKnowledgeGraph
}

type TransferRule struct {
	RuleID        string
	SourceDomain  string
	TargetDomain  string
	TransferRate  float64
	Applicability float64
	Conditions    []string
}

type DomainKnowledgeGraph struct {
	Nodes []*DomainNode
	Edges []*DomainEdge
}

type DomainNode struct {
	NodeID      string
	DomainName  string
	Knowledge   []float64
	Metadata    map[string]interface{}
}

type DomainEdge struct {
	SourceNode string
	TargetNode string
	Weight     float64
	Relation   string
}

type KnowledgeTransferResult struct {
	SourceDomain    string                 `json:"source_domain"`
	TargetDomain    string                 `json:"target_domain"`
	TransferEfficiency float64             `json:"transfer_efficiency"`
	KnowledgeGained float64               `json:"knowledge_gained"`
	PerformanceGain float64               `json:"performance_gain"`
}

type ContinualLearningSystem struct {
	mu          sync.RWMutex
	initialized bool
	taskStream  []*LearningTaskMeta
	knowledgeBase *MetaKnowledgeBase
	plasticity   float64
	stability    float64
	forgettingMetrics *ForgettingMetrics
}

type LearningTaskMeta struct {
	TaskID        string
	TaskType      string
	ArrivalTime   time.Time
	Complexity    float64
	DataSize      int
	LabelSpace    []string
	Requirements  map[string]float64
}

type MetaKnowledgeBase struct {
	TaskID        string
	SharedKnowledge map[string][]float64
	TaskSpecificKnowledge map[string]map[string][]float64
	MetaRules     []*MetaRule
}

type MetaRule struct {
	RuleID      string
	Condition   string
	Action      string
	SuccessRate float64
	UsageCount  int
}

type ForgettingMetrics struct {
	TaskID            string
	ForgettingRate    float64
	RetentionRate     float64
	PerformanceHistory []float64
}

type ContinualLearningResult struct {
	TaskID              string                  `json:"task_id"`
	LearningProgress    float64                 `json:"learning_progress"`
	KnowledgeRetention  float64                 `json:"knowledge_retention"`
	AdaptationSpeed     float64                 `json:"adaptation_speed"`
	PerformanceGain    float64                 `json:"performance_gain"`
}

func NewMetaLearningSystem() *MetaLearningSystem {
	return &MetaLearningSystem{
		fewShotLearner:        NewFewShotLearner(),
		rapidAdapter:          NewRapidAttackAdapter(),
		metaKnowledgeTransfer: NewMetaKnowledgeTransfer(),
		continualLearner:      NewContinualLearningSystem(),
		adaptationHistory:     make(map[string]*AdaptationRecord),
	}
}

func (m *MetaLearningSystem) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	if err := m.fewShotLearner.Initialize(ctx); err != nil {
		return err
	}

	if err := m.rapidAdapter.Initialize(ctx); err != nil {
		return err
	}

	if err := m.metaKnowledgeTransfer.Initialize(ctx); err != nil {
		return err
	}

	if err := m.continualLearner.Initialize(ctx); err != nil {
		return err
	}

	m.initialized = true
	return nil
}

func NewFewShotLearner() *FewShotLearner {
	return &FewShotLearner{
		modelPrior: &ModelPrior{
			BaseParameters: make([]float64, 64),
			Variance: 0.1,
			KnowledgeBase: make([]PriorKnowledge, 0),
		},
		supportSets: make(map[string]*SupportSet),
		episodeCount: 0,
	}
}

func (f *FewShotLearner) Initialize(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.modelPrior.BaseParameters = make([]float64, 64)
	for i := range f.modelPrior.BaseParameters {
		f.modelPrior.BaseParameters[i] = 0.01
	}

	f.initialized = true
	return nil
}

func (f *FewShotLearner) FewShotLearn(ctx context.Context, task *FewShotTask) (*FewShotLearningResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("few-shot learner not initialized")
	}

	result := &FewShotLearningResult{
		TaskID:     task.TaskID,
		Predictions: make([]string, 0),
		LearningCurve: make([]float64, 0),
	}

	supportSize := len(task.SupportSamples)
	querySize := len(task.QuerySamples)

	result.AdaptationSteps = supportSize

	learningCurve := f.simulateLearningCurve(supportSize)
	result.LearningCurve = learningCurve

	for i := 0; i < querySize; i++ {
		prediction := f.predictClass(task.QuerySamples[i], task.Classes)
		result.Predictions = append(result.Predictions, prediction)
	}

	result.Confidence = f.calculateConfidence(learningCurve)

	supportSet := &SupportSet{
		TaskID:      task.TaskID,
		SupportSize: supportSize,
		QuerySize:   querySize,
		Classes:     task.Classes,
		Samples:     task.SupportSamples,
	}

	f.supportSets[task.TaskID] = supportSet
	f.episodeCount++

	return result, nil
}

func (f *FewShotLearner) simulateLearningCurve(supportSize int) []float64 {
	curve := make([]float64, supportSize)

	for i := 0; i < supportSize; i++ {
		progress := float64(i+1) / float64(supportSize)
		base := 0.5 + progress*0.4
		variance := math.Mod(float64(i), 0.1)
		curve[i] = math.Min(1.0, math.Max(0.0, base+variance))
	}

	return curve
}

func (f *FewShotLearner) predictClass(sample *FewShotSample, classes []string) string {
	if len(classes) == 0 {
		return "unknown"
	}

	selectedClass := classes[int(math.Mod(float64(len(sample.Features)), float64(len(classes))))]

	return selectedClass
}

func (f *FewShotLearner) calculateConfidence(learningCurve []float64) float64 {
	if len(learningCurve) == 0 {
		return 0.0
	}

	total := 0.0
	for _, score := range learningCurve {
		total += score
	}

	return total / float64(len(learningCurve))
}

func (f *FewShotLearner) GetEpisodeCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.episodeCount
}

type FewShotTask struct {
	TaskID        string
	SupportSamples []*FewShotSample
	QuerySamples  []*FewShotSample
	Classes       []string
	NWay          int
	KShot         int
}

func NewRapidAttackAdapter() *RapidAttackAdapter {
	detectors := make(map[string]*AttackDetector)

	attackTypes := []string{"adversarial", "injection", "evasion", "poisoning", "extraction"}

	for _, at := range attackTypes {
		detectors[at] = &AttackDetector{
			DetectorID:   fmt.Sprintf("detector_%s", at),
			AttackType:   at,
			DetectionRate: 0.0,
			FalsePositiveRate: 0.0,
			AdaptationHistory: make([]*AdaptationStep, 0),
		}
	}

	return &RapidAttackAdapter{
		attackTypes: attackTypes,
		detectors:   detectors,
		adaptationPolicy: &AdaptationPolicy{
			PolicyID:      "default_policy",
			LearningRate:  0.01,
			Threshold:     0.8,
			MaxIterations: 100,
		},
	}
}

func (ra *RapidAttackAdapter) Initialize(ctx context.Context) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	for _, detector := range ra.detectors {
		detector.DetectionRate = 0.7
		detector.FalsePositiveRate = 0.1
	}

	ra.initialized = true
	return nil
}

func (ra *RapidAttackAdapter) RapidAdapt(ctx context.Context, attackType string, samples []*AttackSample) (*RapidAdaptationResult, error) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if !ra.initialized {
		return nil, fmt.Errorf("rapid attack adapter not initialized")
	}

	startTime := time.Now()

	result := &RapidAdaptationResult{
		AttackType:      attackType,
		AdaptationSteps: 0,
	}

	detector, exists := ra.detectors[attackType]
	if !exists {
		return nil, fmt.Errorf("unknown attack type: %s", attackType)
	}

	adaptationSteps := ra.performAdaptation(detector, samples, ra.adaptationPolicy)
	result.AdaptationSteps = adaptationSteps

	result.DetectionRate = detector.DetectionRate
	result.SuccessRate = 1.0 - detector.FalsePositiveRate
	result.AdaptationTime = time.Since(startTime)

	return result, nil
}

func (ra *RapidAttackAdapter) performAdaptation(detector *AttackDetector, samples []*AttackSample, policy *AdaptationPolicy) int {
	adaptationSteps := 0

	for i := 0; i < policy.MaxIterations && adaptationSteps < len(samples); i++ {
		sample := samples[adaptationSteps%len(samples)]

		detected := ra.simulateDetection(detector, sample)

		adaptationStep := &AdaptationStep{
			StepID:     i,
			InputPattern: sample.Pattern,
			Detected:   detected,
			Adapted:    true,
			Timestamp:  time.Now(),
		}

		detector.AdaptationHistory = append(detector.AdaptationHistory, adaptationStep)

		detector.DetectionRate = math.Min(1.0, detector.DetectionRate+policy.LearningRate*0.1)
		detector.FalsePositiveRate = math.Max(0.0, detector.FalsePositiveRate-policy.LearningRate*0.05)

		adaptationSteps++

		if detector.DetectionRate >= policy.Threshold {
			break
		}
	}

	return adaptationSteps
}

func (ra *RapidAttackAdapter) simulateDetection(detector *AttackDetector, sample *AttackSample) bool {
	baseRate := detector.DetectionRate
	patternInfluence := math.Mod(float64(len(sample.Pattern)), 0.3)*0.2

	return baseRate+patternInfluence > 0.6
}

func (ra *RapidAttackAdapter) GetDetector(attackType string) (*AttackDetector, bool) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	detector, exists := ra.detectors[attackType]
	return detector, exists
}

type AttackSample struct {
	SampleID  string
	Pattern   string
	IsAttack  bool
	Features  map[string]float64
}

func NewMetaKnowledgeTransfer() *MetaKnowledgeTransfer {
	transferRules := make([]*TransferRule, 0)

	sourceDomains := []string{"image", "text", "audio", "tabular"}
	targetDomains := []string{"image", "text", "audio", "tabular"}

	for _, src := range sourceDomains {
		for _, tgt := range targetDomains {
			if src != tgt {
				transferRules = append(transferRules, &TransferRule{
					RuleID:        fmt.Sprintf("rule_%s_to_%s", src, tgt),
					SourceDomain:  src,
					TargetDomain:  tgt,
					TransferRate:  0.5 + math.Mod(float64(len(src)+len(tgt)), 0.3)*0.2,
					Applicability: 0.7,
					Conditions:    []string{"domain_similarity"},
				})
			}
		}
	}

	knowledgeGraph := &DomainKnowledgeGraph{
		Nodes: make([]*DomainNode, 0),
		Edges: make([]*DomainEdge, 0),
	}

	for _, domain := range append(sourceDomains, targetDomains...) {
		knowledgeGraph.Nodes = append(knowledgeGraph.Nodes, &DomainNode{
			NodeID:     domain,
			DomainName: domain,
			Knowledge:  make([]float64, 32),
			Metadata:   make(map[string]interface{}),
		})
	}

	return &MetaKnowledgeTransfer{
		sourceDomains: sourceDomains,
		targetDomains: targetDomains,
		transferRules: transferRules,
		knowledgeGraph: knowledgeGraph,
	}
}

func (mkt *MetaKnowledgeTransfer) Initialize(ctx context.Context) error {
	mkt.mu.Lock()
	defer mkt.mu.Unlock()
	mkt.initialized = true
	return nil
}

func (mkt *MetaKnowledgeTransfer) TransferKnowledge(ctx context.Context, sourceDomain, targetDomain string) (*KnowledgeTransferResult, error) {
	mkt.mu.Lock()
	defer mkt.mu.Unlock()

	if !mkt.initialized {
		return nil, fmt.Errorf("meta knowledge transfer not initialized")
	}

	result := &KnowledgeTransferResult{
		SourceDomain: sourceDomain,
		TargetDomain: targetDomain,
	}

	var bestRule *TransferRule
	for _, rule := range mkt.transferRules {
		if rule.SourceDomain == sourceDomain && rule.TargetDomain == targetDomain {
			bestRule = rule
			break
		}
	}

	if bestRule == nil {
		result.TransferEfficiency = 0.0
		result.KnowledgeGained = 0.0
		result.PerformanceGain = 0.0
		return result, nil
	}

	result.TransferEfficiency = bestRule.TransferRate * bestRule.Applicability
	result.KnowledgeGained = result.TransferEfficiency * 0.8
	result.PerformanceGain = result.TransferEfficiency * 0.6

	return result, nil
}

func (mkt *MetaKnowledgeTransfer) GetTransferRules(sourceDomain, targetDomain string) []*TransferRule {
	mkt.mu.RLock()
	defer mkt.mu.RUnlock()

	rules := make([]*TransferRule, 0)
	for _, rule := range mkt.transferRules {
		if rule.SourceDomain == sourceDomain && rule.TargetDomain == targetDomain {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (mkt *MetaKnowledgeTransfer) UpdateKnowledgeGraph(sourceDomain, targetDomain string, knowledgeGain float64) error {
	mkt.mu.Lock()
	defer mkt.mu.Unlock()

	for _, node := range mkt.knowledgeGraph.Nodes {
		if node.DomainName == targetDomain {
			for i := range node.Knowledge {
				node.Knowledge[i] = math.Min(1.0, node.Knowledge[i]+knowledgeGain*0.01)
			}
		}
	}

	mkt.knowledgeGraph.Edges = append(mkt.knowledgeGraph.Edges, &DomainEdge{
		SourceNode: sourceDomain,
		TargetNode: targetDomain,
		Weight:     knowledgeGain,
		Relation:   "transfer",
	})

	return nil
}

func NewContinualLearningSystem() *ContinualLearningSystem {
	return &ContinualLearningSystem{
		taskStream: make([]*LearningTaskMeta, 0),
		knowledgeBase: &MetaKnowledgeBase{
			SharedKnowledge: make(map[string][]float64),
			TaskSpecificKnowledge: make(map[string]map[string][]float64),
			MetaRules: make([]*MetaRule, 0),
		},
		plasticity:  0.8,
		stability:   0.9,
		forgettingMetrics: &ForgettingMetrics{
			PerformanceHistory: make([]float64, 0),
		},
	}
}

func (cl *ContinualLearningSystem) Initialize(ctx context.Context) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.initialized = true
	return nil
}

func (cl *ContinualLearningSystem) AddTask(ctx context.Context, task *LearningTaskMeta) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if !cl.initialized {
		return fmt.Errorf("continual learning system not initialized")
	}

	task.ArrivalTime = time.Now()
	cl.taskStream = append(cl.taskStream, task)

	return nil
}

func (cl *ContinualLearningSystem) LearnContinually(ctx context.Context, taskID string) (*ContinualLearningResult, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if !cl.initialized {
		return nil, fmt.Errorf("continual learning system not initialized")
	}

	result := &ContinualLearningResult{
		TaskID: taskID,
	}

	var task *LearningTaskMeta
	for _, t := range cl.taskStream {
		if t.TaskID == taskID {
			task = t
			break
		}
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	result.LearningProgress = cl.calculateLearningProgress(task)
	result.KnowledgeRetention = cl.stability
	result.AdaptationSpeed = 1.0 / (1.0 + task.Complexity*0.1)
	result.PerformanceGain = result.LearningProgress * result.AdaptationSpeed * 0.8

	cl.updateKnowledgeBase(task, result.PerformanceGain)
	cl.updateForgettingMetrics(taskID, result.PerformanceGain)

	return result, nil
}

func (cl *ContinualLearningSystem) calculateLearningProgress(task *LearningTaskMeta) float64 {
	baseProgress := 0.5 + math.Mod(float64(task.DataSize), 0.4)*0.3

	complexityFactor := 1.0 - task.Complexity*0.2

	return math.Min(1.0, math.Max(0.0, baseProgress*complexityFactor))
}

func (cl *ContinualLearningSystem) updateKnowledgeBase(task *LearningTaskMeta, performanceGain float64) {
	knowledgeKey := fmt.Sprintf("knowledge_%s", task.TaskType)

	cl.knowledgeBase.SharedKnowledge[knowledgeKey] = make([]float64, 64)
	for i := range cl.knowledgeBase.SharedKnowledge[knowledgeKey] {
		cl.knowledgeBase.SharedKnowledge[knowledgeKey][i] = performanceGain
	}

	if cl.knowledgeBase.TaskSpecificKnowledge[task.TaskID] == nil {
		cl.knowledgeBase.TaskSpecificKnowledge[task.TaskID] = make(map[string][]float64)
	}
	cl.knowledgeBase.TaskSpecificKnowledge[task.TaskID]["parameters"] = make([]float64, 32)
}

func (cl *ContinualLearningSystem) updateForgettingMetrics(taskID string, performanceGain float64) {
	cl.forgettingMetrics.TaskID = taskID
	cl.forgettingMetrics.PerformanceHistory = append(cl.forgettingMetrics.PerformanceHistory, performanceGain)

	if len(cl.forgettingMetrics.PerformanceHistory) > 1 {
		historyLen := len(cl.forgettingMetrics.PerformanceHistory)
		prev := cl.forgettingMetrics.PerformanceHistory[historyLen-2]
		curr := cl.forgettingMetrics.PerformanceHistory[historyLen-1]

		forgetting := prev - curr
		if forgetting > 0 {
			cl.forgettingMetrics.ForgettingRate = forgetting
		}

		cl.forgettingMetrics.RetentionRate = 1.0 - cl.forgettingMetrics.ForgettingRate
	}
}

func (cl *ContinualLearningSystem) UpdatePlasticityStability(performanceDelta float64) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if performanceDelta > 0 {
		cl.stability = math.Min(1.0, cl.stability+0.01)
		cl.plasticity = math.Max(0.0, cl.plasticity-0.005)
	} else {
		cl.plasticity = math.Min(1.0, cl.plasticity+0.01)
		cl.stability = math.Max(0.0, cl.stability-0.005)
	}
}

func (cl *ContinualLearningSystem) GetTaskCount() int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return len(cl.taskStream)
}

func (cl *ContinualLearningSystem) GetForgettingMetrics() *ForgettingMetrics {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return &ForgettingMetrics{
		TaskID:             cl.forgettingMetrics.TaskID,
		ForgettingRate:     cl.forgettingMetrics.ForgettingRate,
		RetentionRate:      cl.forgettingMetrics.RetentionRate,
		PerformanceHistory: cl.forgettingMetrics.PerformanceHistory,
	}
}

func (m *MetaLearningSystem) RunMetaLearningPipeline(ctx context.Context, fewShotTask *FewShotTask, attackType string, sourceDomain, targetDomain string, continualTaskID string) (*MetaLearningResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil, fmt.Errorf("meta learning system not initialized")
	}

	result := &MetaLearningResult{
		Timestamp: time.Now(),
	}

	fewShotResult, _ := m.fewShotLearner.FewShotLearn(ctx, fewShotTask)
	if fewShotResult != nil {
		result.FewShotLearning = fewShotResult
	}

	rapidResult, _ := m.rapidAdapter.RapidAdapt(ctx, attackType, nil)
	if rapidResult != nil {
		result.RapidAdaptation = rapidResult
	}

	transferResult, _ := m.metaKnowledgeTransfer.TransferKnowledge(ctx, sourceDomain, targetDomain)
	if transferResult != nil {
		result.KnowledgeTransfer = transferResult
	}

	continualResult, _ := m.continualLearner.LearnContinually(ctx, continualTaskID)
	if continualResult != nil {
		result.ContinualLearning = continualResult
	}

	record := &AdaptationRecord{
		ID:            fmt.Sprintf("adapt_%d", time.Now().UnixNano()),
		Timestamp:     time.Now(),
		TaskType:      fewShotTask.TaskID,
		AdaptationTime: time.Since(result.Timestamp),
		Performance:   result.GetOverallPerformance(),
	}

	m.adaptationHistory[record.ID] = record

	return result, nil
}

type MetaLearningResult struct {
	Timestamp         time.Time
	FewShotLearning   *FewShotLearningResult
	RapidAdaptation   *RapidAdaptationResult
	KnowledgeTransfer *KnowledgeTransferResult
	ContinualLearning *ContinualLearningResult
}

func (r *MetaLearningResult) GetOverallPerformance() float64 {
	total := 0.0
	count := 0.0

	if r.FewShotLearning != nil {
		total += r.FewShotLearning.Confidence * 100
		count++
	}

	if r.RapidAdaptation != nil {
		total += r.RapidAdaptation.SuccessRate * 100
		count++
	}

	if r.KnowledgeTransfer != nil {
		total += r.KnowledgeTransfer.TransferEfficiency * 100
		count++
	}

	if r.ContinualLearning != nil {
		total += r.ContinualLearning.PerformanceGain * 100
		count++
	}

	if count == 0 {
		return 0.0
	}

	return total / count
}

func (m *MetaLearningSystem) GetAdaptationHistory(ctx context.Context, recordID string) (*AdaptationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, exists := m.adaptationHistory[recordID]
	if !exists {
		return nil, fmt.Errorf("adaptation record not found")
	}

	return record, nil
}

func (m *MetaLearningSystem) GetSystemMetrics(ctx context.Context) *MetaLearningMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MetaLearningMetrics{
		FewShotEpisodes:    m.fewShotLearner.GetEpisodeCount(),
		ActiveDetectors:    len(m.rapidAdapter.detectors),
		TransferRules:      len(m.metaKnowledgeTransfer.transferRules),
		ContinualTasks:     len(m.continualLearner.taskStream),
		AdaptationRecords:  len(m.adaptationHistory),
		Plasticity:         m.continualLearner.plasticity,
		Stability:          m.continualLearner.stability,
	}
}

type MetaLearningMetrics struct {
	FewShotEpisodes    int
	ActiveDetectors    int
	TransferRules      int
	ContinualTasks     int
	AdaptationRecords  int
	Plasticity        float64
	Stability          float64
}
