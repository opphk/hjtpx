package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

// ============================================
// 实时行为学习系统
// 在线学习用户行为模式并动态更新模型
// ============================================

type RealTimeBehaviorLearningService struct {
	behaviorLearner     *OnlineBehaviorLearner
	patternDetector     *PatternChangeDetector
	modelAdaptor        *AdaptiveModelAdaptor
	anomalyUpdater      *AnomalyThresholdUpdater
	featureTracker      *FeatureDriftTracker
	learningEnabled     bool
	mu                  sync.RWMutex
}

type LearningResult struct {
	UserID           string                 `json:"user_id"`
	UpdatedProfile   *UserBehaviorSignature `json:"updated_profile"`
	DriftDetected    bool                   `json:"drift_detected"`
	DriftSeverity    float64                `json:"drift_severity"`
	AdaptationScore  float64                `json:"adaptation_score"`
	LearningStatus   string                 `json:"learning_status"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

func NewRealTimeBehaviorLearningService() *RealTimeBehaviorLearningService {
	return &RealTimeBehaviorLearningService{
		behaviorLearner:     NewOnlineBehaviorLearner(),
		patternDetector:     NewPatternChangeDetector(),
		modelAdaptor:        NewAdaptiveModelAdaptor(),
		anomalyUpdater:      NewAnomalyThresholdUpdater(),
		featureTracker:      NewFeatureDriftTracker(),
		learningEnabled:     true,
	}
}

func (s *RealTimeBehaviorLearningService) StartLearning(ctx context.Context) error {
	s.mu.Lock()
	s.learningEnabled = true
	s.mu.Unlock()
	
	go s.runContinuousLearning(ctx)
	
	return nil
}

func (s *RealTimeBehaviorLearningService) StopLearning() {
	s.mu.Lock()
	s.learningEnabled = false
	s.mu.Unlock()
}

func (s *RealTimeBehaviorLearningService) runContinuousLearning(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			s.performPeriodicLearning(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *RealTimeBehaviorLearningService) performPeriodicLearning(ctx context.Context) {
	s.mu.RLock()
	enabled := s.learningEnabled
	s.mu.RUnlock()
	
	if !enabled {
		return
	}
	
	s.modelAdaptor.AdaptModels(ctx)
	s.anomalyUpdater.UpdateThresholds(ctx)
	s.featureTracker.DetectDrift(ctx)
}

func (s *RealTimeBehaviorLearningService) LearnFromBehavior(ctx context.Context, userID string, traceData *model.TraceData) (*LearningResult, error) {
	s.mu.RLock()
	enabled := s.learningEnabled
	s.mu.RUnlock()
	
	if !enabled {
		return &LearningResult{
			UserID:         userID,
			LearningStatus: "learning_disabled",
			UpdatedAt:      time.Now(),
		}, nil
	}
	
	signature := s.behaviorLearner.Learn(ctx, userID, traceData)
	
	driftDetected, severity := s.patternDetector.DetectPatternChange(userID, traceData)
	
	adaptationScore := s.modelAdaptor.AdaptToUser(ctx, userID, signature)
	
	if driftDetected {
		s.anomalyUpdater.AdjustThresholds(userID, severity)
		s.featureTracker.TrackFeatureChange(userID, traceData)
	}
	
	return &LearningResult{
		UserID:           userID,
		UpdatedProfile:   signature,
		DriftDetected:    driftDetected,
		DriftSeverity:    severity,
		AdaptationScore:  adaptationScore,
		LearningStatus:   "completed",
		UpdatedAt:        time.Now(),
	}, nil
}

func (s *RealTimeBehaviorLearningService) GetUserSignature(ctx context.Context, userID string) (*UserBehaviorSignature, error) {
	return s.behaviorLearner.GetSignature(userID), nil
}

func (s *RealTimeBehaviorLearningService) IsLearningEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.learningEnabled
}

// ============================================
// 在线行为学习器
// ============================================

type OnlineBehaviorLearner struct {
	signatures    map[string]*UserBehaviorSignature
	learningRates map[string]float64
	mu            sync.RWMutex
}

type UserBehaviorSignature struct {
	UserID                   string
	CreationTime             time.Time
	LastUpdateTime           time.Time
	UpdateCount              int
	TotalSamples             int
	
	// 速度特征
	AverageSpeed             float64
	SpeedVariance            float64
	MaxSpeed                 float64
	MinSpeed                 float64
	
	// 轨迹特征
	AveragePathEfficiency    float64
	AverageCurvature         float64
	AverageJitter            float64
	DirectionChangeRate      float64
	
	// 点击特征
	ClickIntervalMean        float64
	ClickIntervalVariance    float64
	ClickAreaSize            float64
	PreClickHesitationMean   float64
	
	// 时间特征
	TypingSpeed              float64
	PauseFrequency           float64
	SessionDurationMean      float64
	
	// 行为模式
	BehaviorPatternHash      string
	TypicalActions           []string
	ActionTransitionMatrix   map[string]map[string]float64
}

func NewOnlineBehaviorLearner() *OnlineBehaviorLearner {
	return &OnlineBehaviorLearner{
		signatures:    make(map[string]*UserBehaviorSignature),
		learningRates: make(map[string]float64),
	}
}

func (l *OnlineBehaviorLearner) Learn(ctx context.Context, userID string, traceData *model.TraceData) *UserBehaviorSignature {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	signature, exists := l.signatures[userID]
	if !exists {
		signature = l.createNewSignature(userID)
		l.signatures[userID] = signature
		l.learningRates[userID] = 0.3
	}
	
	l.updateSignature(signature, traceData)
	
	return signature
}

func (l *OnlineBehaviorLearner) createNewSignature(userID string) *UserBehaviorSignature {
	return &UserBehaviorSignature{
		UserID:                 userID,
		CreationTime:           time.Now(),
		LastUpdateTime:         time.Now(),
		UpdateCount:            0,
		TotalSamples:           0,
		ActionTransitionMatrix: make(map[string]map[string]float64),
		TypicalActions:         []string{},
	}
}

func (l *OnlineBehaviorLearner) updateSignature(signature *UserBehaviorSignature, traceData *model.TraceData) {
	points := traceData.Points
	if len(points) < 2 {
		return
	}
	
	learningRate := l.getLearningRate(signature)
	
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	
	if len(speeds) > 0 {
		signature.AverageSpeed = l.updateExponential(signature.AverageSpeed, aiMeanFloatSlice(speeds), learningRate)
		signature.SpeedVariance = l.updateExponential(signature.SpeedVariance, aiVarianceFloatSlice(speeds), learningRate)
		signature.MaxSpeed = l.updateExponential(signature.MaxSpeed, aiMaxFloatSlice(speeds), learningRate)
		signature.MinSpeed = l.updateExponential(signature.MinSpeed, aiMinFloatSlice(speeds), learningRate)
	}
	
	totalDist := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	start, end := points[0], points[len(points)-1]
	straightDist := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
	efficiency := 0.0
	if totalDist > 0 {
		efficiency = straightDist / totalDist
	}
	signature.AveragePathEfficiency = l.updateExponential(signature.AveragePathEfficiency, efficiency, learningRate)
	
	curvatures := make([]float64, 0, len(points)-2)
	for i := 1; i < len(points)-1; i++ {
		curvatures = append(curvatures, aiComputeCurvature(points[i-1], points[i], points[i+1]))
	}
	if len(curvatures) > 0 {
		signature.AverageCurvature = l.updateExponential(signature.AverageCurvature, aiMeanFloatSlice(curvatures), learningRate)
	}
	
	clickCount := 0
	clickTimestamps := make([]int64, 0)
	for _, p := range points {
		if p.Event == "click" {
			clickCount++
			clickTimestamps = append(clickTimestamps, p.Timestamp)
		}
	}
	if len(clickTimestamps) >= 2 {
		intervals := make([]float64, len(clickTimestamps)-1)
		for i := 1; i < len(clickTimestamps); i++ {
			intervals[i-1] = float64(clickTimestamps[i] - clickTimestamps[i-1])
		}
		signature.ClickIntervalMean = l.updateExponential(signature.ClickIntervalMean, aiMeanFloatSlice(intervals), learningRate)
		signature.ClickIntervalVariance = l.updateExponential(signature.ClickIntervalVariance, aiVarianceFloatSlice(intervals), learningRate)
	}
	
	signature.UpdateCount++
	signature.TotalSamples += len(points)
	signature.LastUpdateTime = time.Now()
	signature.BehaviorPatternHash = l.computePatternHash(points)
	
	l.decayLearningRate(signature.UserID)
}

func (l *OnlineBehaviorLearner) getLearningRate(signature *UserBehaviorSignature) float64 {
	lr, exists := l.learningRates[signature.UserID]
	if !exists {
		return 0.3
	}
	return lr
}

func (l *OnlineBehaviorLearner) decayLearningRate(userID string) {
	lr, exists := l.learningRates[userID]
	if !exists {
		l.learningRates[userID] = 0.3
		return
	}
	l.learningRates[userID] = math.Max(0.01, lr*0.995)
}

func (l *OnlineBehaviorLearner) updateExponential(current, newValue, rate float64) float64 {
	if current == 0 {
		return newValue
	}
	return current*(1-rate) + newValue*rate
}

func (l *OnlineBehaviorLearner) computePatternHash(points []model.TracePoint) string {
	hash := 0
	for i, p := range points {
		if i%5 == 0 {
			hash = (hash * 31) ^ int(p.X)
			hash = (hash * 31) ^ int(p.Y)
		}
	}
	return fmt.Sprintf("%x", hash)
}

func (l *OnlineBehaviorLearner) GetSignature(userID string) *UserBehaviorSignature {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.signatures[userID]
}

// ============================================
// 模式变化检测器
// ============================================

type PatternChangeDetector struct {
	previousSignatures map[string]*UserBehaviorSignature
	changeHistory      map[string][]ChangeRecord
	mu                 sync.RWMutex
}

type ChangeRecord struct {
	Timestamp     time.Time
	DriftSeverity float64
	Feature       string
}

func NewPatternChangeDetector() *PatternChangeDetector {
	return &PatternChangeDetector{
		previousSignatures: make(map[string]*UserBehaviorSignature),
		changeHistory:      make(map[string][]ChangeRecord),
	}
}

func (d *PatternChangeDetector) DetectPatternChange(userID string, traceData *model.TraceData) (bool, float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	currentSig := extractCurrentSignature(traceData)
	prevSig, exists := d.previousSignatures[userID]
	
	if !exists {
		d.previousSignatures[userID] = currentSig
		return false, 0
	}
	
	driftScore := d.compareSignatures(prevSig, currentSig)
	
	if driftScore > 0.3 {
		d.changeHistory[userID] = append(d.changeHistory[userID], ChangeRecord{
			Timestamp:     time.Now(),
			DriftSeverity: driftScore,
			Feature:       d.identifyDriftFeature(prevSig, currentSig),
		})
		
		if len(d.changeHistory[userID]) > 100 {
			d.changeHistory[userID] = d.changeHistory[userID][1:]
		}
	}
	
	d.previousSignatures[userID] = currentSig
	
	return driftScore > 0.3, driftScore
}

func extractCurrentSignature(traceData *model.TraceData) *UserBehaviorSignature {
	points := traceData.Points
	if len(points) < 2 {
		return &UserBehaviorSignature{}
	}
	
	signature := &UserBehaviorSignature{}
	
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	
	if len(speeds) > 0 {
		signature.AverageSpeed = aiMeanFloatSlice(speeds)
		signature.SpeedVariance = aiVarianceFloatSlice(speeds)
	}
	
	return signature
}

func (d *PatternChangeDetector) compareSignatures(prev, curr *UserBehaviorSignature) float64 {
	if prev == nil || curr == nil {
		return 0
	}
	
	diff := 0.0
	count := 0
	
	if prev.AverageSpeed > 0 && curr.AverageSpeed > 0 {
		diff += math.Abs(prev.AverageSpeed-curr.AverageSpeed) / prev.AverageSpeed
		count++
	}
	if prev.SpeedVariance > 0 && curr.SpeedVariance > 0 {
		diff += math.Abs(prev.SpeedVariance-curr.SpeedVariance) / prev.SpeedVariance
		count++
	}
	if prev.AveragePathEfficiency > 0 && curr.AveragePathEfficiency > 0 {
		diff += math.Abs(prev.AveragePathEfficiency-curr.AveragePathEfficiency)
		count++
	}
	if prev.AverageCurvature > 0 && curr.AverageCurvature > 0 {
		diff += math.Abs(prev.AverageCurvature-curr.AverageCurvature)
		count++
	}
	
	if count == 0 {
		return 0
	}
	return diff / float64(count)
}

func (d *PatternChangeDetector) identifyDriftFeature(prev, curr *UserBehaviorSignature) string {
	if prev.AverageSpeed > 0 && curr.AverageSpeed > 0 {
		if math.Abs(prev.AverageSpeed-curr.AverageSpeed)/prev.AverageSpeed > 0.3 {
			return "speed"
		}
	}
	if prev.SpeedVariance > 0 && curr.SpeedVariance > 0 {
		if math.Abs(prev.SpeedVariance-curr.SpeedVariance)/prev.SpeedVariance > 0.3 {
			return "speed_variance"
		}
	}
	return "unknown"
}

// ============================================
// 自适应模型适配器
// ============================================

type AdaptiveModelAdaptor struct {
	userModels      map[string]*AdaptiveModel
	adaptationCache map[string]float64
	mu              sync.RWMutex
}

type AdaptiveModel struct {
	UserID           string
	ModelWeights     []float64
	AdaptationCount  int
	LastAdaptation   time.Time
}

func NewAdaptiveModelAdaptor() *AdaptiveModelAdaptor {
	return &AdaptiveModelAdaptor{
		userModels:      make(map[string]*AdaptiveModel),
		adaptationCache: make(map[string]float64),
	}
}

func (a *AdaptiveModelAdaptor) AdaptToUser(ctx context.Context, userID string, signature *UserBehaviorSignature) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	model, exists := a.userModels[userID]
	if !exists {
		model = a.createModel(userID)
		a.userModels[userID] = model
	}
	
	adaptationScore := a.adaptModel(model, signature)
	
	model.AdaptationCount++
	model.LastAdaptation = time.Now()
	
	a.adaptationCache[userID] = adaptationScore
	
	return adaptationScore
}

func (a *AdaptiveModelAdaptor) createModel(userID string) *AdaptiveModel {
	return &AdaptiveModel{
		UserID:           userID,
		ModelWeights:     make([]float64, 10),
		AdaptationCount:  0,
		LastAdaptation:   time.Now(),
	}
}

func (a *AdaptiveModelAdaptor) adaptModel(model *AdaptiveModel, signature *UserBehaviorSignature) float64 {
	targetWeights := []float64{
		signature.AverageSpeed / 10,
		signature.SpeedVariance / 5,
		signature.AveragePathEfficiency,
		signature.AverageCurvature,
		signature.ClickIntervalMean / 1000,
		signature.ClickIntervalVariance / 1000,
	}
	
	adaptationScore := 0.0
	learningRate := 0.1
	
	for i := 0; i < aiMin(len(model.ModelWeights), len(targetWeights)); i++ {
		delta := targetWeights[i] - model.ModelWeights[i]
		model.ModelWeights[i] += delta * learningRate
		adaptationScore += math.Abs(delta)
	}
	
	return math.Min(1, adaptationScore)
}

func (a *AdaptiveModelAdaptor) AdaptModels(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	now := time.Now()
	for userID, model := range a.userModels {
		if now.Sub(model.LastAdaptation) > 10*time.Minute {
			a.decayModelWeights(model)
		}
		
		if model.AdaptationCount > 100 {
			a.resetAdaptationCount(userID)
		}
	}
}

func (a *AdaptiveModelAdaptor) decayModelWeights(model *AdaptiveModel) {
	for i := range model.ModelWeights {
		model.ModelWeights[i] *= 0.99
	}
}

func (a *AdaptiveModelAdaptor) resetAdaptationCount(userID string) {
	if model, exists := a.userModels[userID]; exists {
		model.AdaptationCount = 0
	}
}

// ============================================
// 异常阈值更新器
// ============================================

type AnomalyThresholdUpdater struct {
	userThresholds map[string]*AnomalyThresholds
	mu             sync.RWMutex
}

type AnomalyThresholds struct {
	UserID            string
	SpeedThreshold    float64
	JitterThreshold   float64
	CurvatureThreshold float64
	ClickIntervalThreshold float64
	UpdatedAt        time.Time
}

func NewAnomalyThresholdUpdater() *AnomalyThresholdUpdater {
	return &AnomalyThresholdUpdater{
		userThresholds: make(map[string]*AnomalyThresholds),
	}
}

func (u *AnomalyThresholdUpdater) AdjustThresholds(userID string, driftSeverity float64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	thresholds, exists := u.userThresholds[userID]
	if !exists {
		thresholds = u.createDefaultThresholds(userID)
		u.userThresholds[userID] = thresholds
	}
	
	adjustmentFactor := 1.0 + driftSeverity*0.2
	
	thresholds.SpeedThreshold *= adjustmentFactor
	thresholds.JitterThreshold *= adjustmentFactor
	thresholds.CurvatureThreshold *= adjustmentFactor
	thresholds.ClickIntervalThreshold *= adjustmentFactor
	
	thresholds.SpeedThreshold = math.Max(5, math.Min(20, thresholds.SpeedThreshold))
	thresholds.JitterThreshold = math.Max(0.01, math.Min(1.0, thresholds.JitterThreshold))
	
	thresholds.UpdatedAt = time.Now()
}

func (u *AnomalyThresholdUpdater) createDefaultThresholds(userID string) *AnomalyThresholds {
	return &AnomalyThresholds{
		UserID:                  userID,
		SpeedThreshold:          10.0,
		JitterThreshold:         0.03,
		CurvatureThreshold:      0.05,
		ClickIntervalThreshold:  50.0,
		UpdatedAt:              time.Now(),
	}
}

func (u *AnomalyThresholdUpdater) UpdateThresholds(ctx context.Context) {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	now := time.Now()
	for _, thresholds := range u.userThresholds {
		if now.Sub(thresholds.UpdatedAt) > 1*time.Hour {
			u.revertToDefaults(thresholds)
		}
	}
}

func (u *AnomalyThresholdUpdater) revertToDefaults(thresholds *AnomalyThresholds) {
	thresholds.SpeedThreshold = 10.0
	thresholds.JitterThreshold = 0.03
	thresholds.CurvatureThreshold = 0.05
	thresholds.ClickIntervalThreshold = 50.0
}

// ============================================
// 特征漂移跟踪器
// ============================================

type FeatureDriftTracker struct {
	driftRecords map[string][]FeatureDriftRecord
	mu           sync.RWMutex
}

type FeatureDriftRecord struct {
	Timestamp     time.Time
	FeatureName   string
	DriftScore    float64
	OldValue      float64
	NewValue      float64
}

func NewFeatureDriftTracker() *FeatureDriftTracker {
	return &FeatureDriftTracker{
		driftRecords: make(map[string][]FeatureDriftRecord),
	}
}

func (t *FeatureDriftTracker) TrackFeatureChange(userID string, traceData *model.TraceData) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	points := traceData.Points
	if len(points) < 2 {
		return
	}
	
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	
	if len(speeds) > 0 {
		avgSpeed := aiMeanFloatSlice(speeds)
		
		records := t.driftRecords[userID]
		oldSpeed := 0.0
		if len(records) > 0 {
			for _, r := range records {
				if r.FeatureName == "average_speed" {
					oldSpeed = r.NewValue
					break
				}
			}
		}
		
		driftScore := 0.0
		if oldSpeed > 0 {
			driftScore = math.Abs(avgSpeed-oldSpeed) / oldSpeed
		}
		
		if driftScore > 0.1 {
			t.driftRecords[userID] = append(t.driftRecords[userID], FeatureDriftRecord{
				Timestamp:   time.Now(),
				FeatureName: "average_speed",
				DriftScore:  driftScore,
				OldValue:    oldSpeed,
				NewValue:    avgSpeed,
			})
		}
	}
	
	if len(t.driftRecords[userID]) > 50 {
		t.driftRecords[userID] = t.driftRecords[userID][1:]
	}
}

func (t *FeatureDriftTracker) DetectDrift(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	for userID, records := range t.driftRecords {
		if len(records) < 3 {
			continue
		}
		
		recentRecords := records[len(records)-3:]
		avgDrift := 0.0
		for _, r := range recentRecords {
			avgDrift += r.DriftScore
		}
		avgDrift /= float64(len(recentRecords))
		
		if avgDrift > 0.2 {
			t.triggerDriftAlert(userID, avgDrift)
		}
	}
}

func (t *FeatureDriftTracker) triggerDriftAlert(userID string, driftScore float64) {
	
}

func (t *FeatureDriftTracker) GetDriftHistory(userID string) []FeatureDriftRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.driftRecords[userID]
}