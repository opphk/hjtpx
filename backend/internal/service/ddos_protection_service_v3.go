package service

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type DDoSProtectionV3Config struct {
	EnableSmartTrafficAnalysis bool
	EnableMLAttackDetection    bool
	EnableAutoTrafficCleaning   bool
	EnableGlobalNodeSupport     bool
	GlobalNodes                []GlobalNode
	MLModelThreshold           float64
	TrafficCleaningThreshold   float64
}

type GlobalNode struct {
	ID         string  `json:"id"`
	Region     string  `json:"region"`
	IPAddress  string  `json:"ip_address"`
	Weight     float64 `json:"weight"`
	IsActive   bool    `json:"is_active"`
	LastHealth time.Time `json:"last_health"`
}

type DDoSProtectionV3Service struct {
	config            DDoSProtectionV3Config
	ipStats           map[string]*DDoSIPStatsV3
	trafficData       map[string]*DDoSTrafficDataV3
	blacklist         map[string]time.Time
	whitelist         map[string]bool
	globalStats       *GlobalDDoSStats
	mlDetector        *MLAttackDetector
	trafficCleaner    *AutoTrafficCleaner
	nodeHealthMonitor *NodeHealthMonitor
	anomalyEngine     *AnomalyDetectionEngine
	mu                sync.RWMutex
}

type DDoSIPStatsV3 struct {
	IP                  string
	RequestCount        int64
	RequestCountMinute  int64
	RequestCountHour    int64
	BlockedCount        int64
	ConnectionCount     int
	FirstSeen           time.Time
	LastSeen            time.Time
	RequestRate         float64
	AvgRequestInterval  time.Duration
	IsAnomaly           bool
	IsBlacklisted       bool
	ThreatScore         float64
	Country             string
	UserAgents          []string
	UniquePaths         int
	UniqueMethods       int
	ErrorRate           float64
	TotalErrors         int
	TotalRequests       int64
	TrafficSignature    *TrafficSignature
	MLPrediction        float64
	AttackType          string
	Confidence          float64
}

type TrafficSignature struct {
	PatternType     string
	PatternHash     string
	FirstSeen       time.Time
	LastSeen        time.Time
	RequestCount    int64
	SourceCount     int
	IsMalicious     bool
	SignatureVector []float64
}

type DDoSTrafficDataV3 struct {
	RequestTimes      []time.Time
	RequestSizes      []int64
	Methods           []string
	Paths             []string
	UserAgents        []string
	StatusCodes       []int
	Countries         []string
	RequestIntervals  []time.Duration
	mu                sync.RWMutex
}

type MLAttackDetector struct {
	model           *DDoSMLModel
	trainingData    []TrainingSample
	isTrained       bool
	featureWeights  map[string]float64
}

type TrainingSample struct {
	Features   []float64
	Label      float64
	Timestamp  time.Time
}

type DDoSMLModel struct {
	weights    [][]float64
	bias       []float64
	inputSize  int
	outputSize int
}

type AutoTrafficCleaner struct {
	enabled       bool
	cleaningRules []CleaningRule
	cleaningQueue chan *CleaningTask
	lastCleanup   time.Time
}

type CleaningRule struct {
	Name        string
	Condition   func(*DDoSIPStatsV3) bool
	Action      CleaningAction
	Priority    int
}

type CleaningAction string

const (
	ActionBlockIP       CleaningAction = "block_ip"
	ActionRateLimit    CleaningAction = "rate_limit"
	ActionChallenge    CleaningAction = "challenge"
	ActionRedirect     CleaningAction = "redirect"
	ActionLog          CleaningAction = "log"
)

type CleaningTask struct {
	IP       string
	Rule     string
	Action   CleaningAction
	Duration time.Duration
}

type NodeHealthMonitor struct {
	nodes     map[string]*GlobalNode
	healthLog map[string][]HealthStatus
	mu        sync.RWMutex
}

type HealthStatus struct {
	NodeID    string
	Status    string
	Latency   time.Duration
	Timestamp time.Time
}

type AnomalyDetectionEngine struct {
	baselineMetrics map[string]*AnomalyBaseline
	stdDevThresholds map[string]float64
	detectionMethods []AnomalyMethod
}

type AnomalyBaseline struct {
	Mean           float64
	StdDev         float64
	SampleCount    int
	LastUpdated    time.Time
	MetricType     string
}

type AnomalyMethod struct {
	Name         string
	DetectFunc   func(*AnomalyDetectionEngine, *DDoSTrafficDataV3, string) (bool, float64)
	Weight       float64
}

type GlobalDDoSStats struct {
	TotalRequests    int64
	TotalBlocked     int64
	ActiveAttacks    int
	PeakQPS          float64
	AvgLatency       time.Duration
	CleanedTraffic   int64
	GlobalThreats    map[string]*GlobalThreat
	mu               sync.RWMutex
}

type GlobalThreat struct {
	ThreatID      string
	ThreatType    string
	Severity      float64
	SourceIPs     []string
	TargetRegion  string
	StartTime     time.Time
	Status        string
}

func NewDDoSProtectionV3Service(config DDoSProtectionV3Config) *DDoSProtectionV3Service {
	service := &DDoSProtectionV3Service{
		config:            config,
		ipStats:           make(map[string]*DDoSIPStatsV3),
		trafficData:       make(map[string]*DDoSTrafficDataV3),
		blacklist:         make(map[string]time.Time),
		whitelist:         make(map[string]bool),
		globalStats:       &GlobalDDoSStats{GlobalThreats: make(map[string]*GlobalThreat)},
	}

	if config.EnableMLAttackDetection {
		service.mlDetector = NewMLAttackDetector()
	}

	if config.EnableAutoTrafficCleaning {
		service.trafficCleaner = NewAutoTrafficCleaner()
		go service.trafficCleaner.Start()
	}

	if config.EnableGlobalNodeSupport {
		service.nodeHealthMonitor = NewNodeHealthMonitor(config.GlobalNodes)
		go service.nodeHealthMonitor.StartMonitoring()
	}

	service.anomalyEngine = NewAnomalyDetectionEngine()
	service.anomalyEngine.initializeDetectionMethods()

	go service.cleanupRoutine()
	go service.statsUpdateRoutine()

	return service
}

func NewMLAttackDetector() *MLAttackDetector {
	return &MLAttackDetector{
		model:          NewDDoSMLModel(),
		trainingData:   make([]TrainingSample, 0),
		isTrained:      false,
		featureWeights: map[string]float64{
			"request_rate":        0.3,
			"error_rate":          0.2,
			"unique_paths":         0.15,
			"avg_interval":         0.25,
			"connection_count":     0.1,
		},
	}
}

func NewDDoSMLModel() *DDoSMLModel {
	return &DDoSMLModel{
		weights:    make([][]float64, 5),
		inputSize:  5,
		outputSize: 1,
	}
}

func (m *DDoSMLModel) Predict(features []float64) float64 {
	if len(features) != m.inputSize {
		return 0.5
	}

	var sum float64
	for i := 0; i < m.inputSize; i++ {
		if i < len(m.weights) && len(m.weights[i]) > 0 {
			sum += features[i] * m.weights[i][0]
		}
	}

	prediction := 1.0 / (1.0 + math.Exp(-sum))
	return math.Max(0, math.Min(1, prediction))
}

func (d *MLAttackDetector) DetectAttack(stats *DDoSIPStatsV3) (bool, float64, string) {
	features := d.extractFeatures(stats)
	prediction := d.model.Predict(features)

	attackTypes := d.classifyAttack(features)
	
	threshold := 0.7
	isAttack := prediction > threshold

	return isAttack, prediction, attackTypes
}

func (d *MLAttackDetector) extractFeatures(stats *DDoSIPStatsV3) []float64 {
	features := make([]float64, 5)

	features[0] = math.Min(1.0, float64(stats.RequestCountMinute)/1000.0)
	features[1] = math.Min(1.0, stats.ErrorRate)
	features[2] = math.Min(1.0, float64(stats.UniquePaths)/100.0)

	if stats.AvgRequestInterval > 0 {
		features[3] = math.Min(1.0, 1.0/(stats.AvgRequestInterval.Seconds()+0.001))
	}

	features[4] = math.Min(1.0, float64(stats.ConnectionCount)/100.0)

	return features
}

func (d *MLAttackDetector) classifyAttack(features []float64) string {
	if features[0] > 0.8 && features[1] < 0.2 {
		return "volume_based"
	}
	if features[1] > 0.7 {
		return "application_layer"
	}
	if features[2] > 0.9 {
		return "scanning"
	}
	if features[4] > 0.8 {
		return "connection_exhaustion"
	}
	return "unknown"
}

func (d *MLAttackDetector) Train(samples []TrainingSample) error {
	d.trainingData = append(d.trainingData, samples...)

	if len(d.trainingData) < 100 {
		return fmt.Errorf("insufficient training data")
	}

	d.isTrained = true
	return nil
}

func NewAutoTrafficCleaner() *AutoTrafficCleaner {
	cleaner := &AutoTrafficCleaner{
		enabled:       true,
		cleaningRules: make([]CleaningRule, 0),
		cleaningQueue: make(chan *CleaningTask, 1000),
		lastCleanup:   time.Now(),
	}

	cleaner.initializeRules()
	return cleaner
}

func (c *AutoTrafficCleaner) initializeRules() {
	c.cleaningRules = append(c.cleaningRules, CleaningRule{
		Name:      "high_volume_block",
		Condition: func(stats *DDoSIPStatsV3) bool { return stats.RequestCountMinute > 5000 },
		Action:    ActionBlockIP,
		Priority:  1,
	})

	c.cleaningRules = append(c.cleaningRules, CleaningRule{
		Name:      "malicious_signature",
		Condition: func(stats *DDoSIPStatsV3) bool { return stats.ThreatScore > 0.8 },
		Action:    ActionBlockIP,
		Priority:  2,
	})

	c.cleaningRules = append(c.cleaningRules, CleaningRule{
		Name:      "ml_detected_attack",
		Condition: func(stats *DDoSIPStatsV3) bool { return stats.MLPrediction > 0.85 },
		Action:    ActionChallenge,
		Priority:  3,
	})

	c.cleaningRules = append(c.cleaningRules, CleaningRule{
		Name:      "anomaly_detected",
		Condition: func(stats *DDoSIPStatsV3) bool { return stats.IsAnomaly },
		Action:    ActionRateLimit,
		Priority:  4,
	})
}

func (c *AutoTrafficCleaner) Start() {
	go func() {
		for task := range c.cleaningQueue {
			c.executeTask(task)
		}
	}()
}

func (c *AutoTrafficCleaner) executeTask(task *CleaningTask) {
	switch task.Action {
	case ActionBlockIP:
		_ = task
	case ActionRateLimit:
		_ = task
	case ActionChallenge:
		_ = task
	case ActionRedirect:
		_ = task
	case ActionLog:
		_ = task
	}
}

func (c *AutoTrafficCleaner) AddTask(task *CleaningTask) {
	select {
	case c.cleaningQueue <- task:
	default:
	}
}

func NewNodeHealthMonitor(nodes []GlobalNode) *NodeHealthMonitor {
	nodeMap := make(map[string]*GlobalNode)
	for _, node := range nodes {
		nodeMap[node.ID] = &node
	}

	return &NodeHealthMonitor{
		nodes:     nodeMap,
		healthLog: make(map[string][]HealthStatus),
	}
}

func (n *NodeHealthMonitor) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			n.checkAllNodes()
		}
	}()
}

func (n *NodeHealthMonitor) checkAllNodes() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for id, node := range n.nodes {
		health := HealthStatus{
			NodeID:    id,
			Status:    "healthy",
			Latency:   time.Duration(100) * time.Millisecond,
			Timestamp: time.Now(),
		}

		if node.Weight == 0 {
			health.Status = "degraded"
		}

		n.healthLog[id] = append(n.healthLog[id], health)
		if len(n.healthLog[id]) > 100 {
			n.healthLog[id] = n.healthLog[id][len(n.healthLog[id])-100:]
		}
	}
}

func (n *NodeHealthMonitor) GetHealthyNodes() []*GlobalNode {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var healthy []*GlobalNode
	for _, node := range n.nodes {
		if node.IsActive && time.Since(node.LastHealth) < 5*time.Minute {
			healthy = append(healthy, node)
		}
	}
	return healthy
}

func NewAnomalyDetectionEngine() *AnomalyDetectionEngine {
	return &AnomalyDetectionEngine{
		baselineMetrics:    make(map[string]*AnomalyBaseline),
		stdDevThresholds: map[string]float64{
			"request_rate": 2.5,
			"error_rate":   3.0,
			"interval":     2.0,
		},
		detectionMethods: make([]AnomalyMethod, 0),
	}
}

func (e *AnomalyDetectionEngine) initializeDetectionMethods() {
	e.detectionMethods = append(e.detectionMethods, AnomalyMethod{
		Name:   "statistical",
		DetectFunc: e.detectStatisticalAnomaly,
		Weight:     0.4,
	})

	e.detectionMethods = append(e.detectionMethods, AnomalyMethod{
		Name:   "pattern_based",
		DetectFunc: e.detectPatternAnomaly,
		Weight:     0.3,
	})

	e.detectionMethods = append(e.detectionMethods, AnomalyMethod{
		Name:   "signature_based",
		DetectFunc: e.detectSignatureAnomaly,
		Weight:     0.3,
	})
}

func (e *AnomalyDetectionEngine) detectStatisticalAnomaly(engine *AnomalyDetectionEngine, traffic *DDoSTrafficDataV3, ip string) (bool, float64) {
	if len(traffic.RequestTimes) < 10 {
		return false, 0
	}

	var intervals []float64
	for i := 1; i < len(traffic.RequestTimes); i++ {
		interval := traffic.RequestTimes[i].Sub(traffic.RequestTimes[i-1]).Seconds() * 1000
		intervals = append(intervals, interval)
	}

	if len(intervals) < 2 {
		return false, 0
	}

	mean := 0.0
	for _, v := range intervals {
		mean += v
	}
	mean /= float64(len(intervals))

	variance := 0.0
	for _, v := range intervals {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	cv := stdDev / (mean + 1)

	if cv < 0.05 && mean < 500 {
		return true, 0.95
	}

	return false, 0
}

func (e *AnomalyDetectionEngine) detectPatternAnomaly(engine *AnomalyDetectionEngine, traffic *DDoSTrafficDataV3, ip string) (bool, float64) {
	uniquePaths := make(map[string]bool)
	for _, path := range traffic.Paths {
		uniquePaths[path] = true
	}

	pathRatio := float64(len(uniquePaths)) / float64(len(traffic.Paths))

	if pathRatio > 0.9 && len(traffic.Paths) > 50 {
		return true, 0.85
	}

	return false, 0
}

func (e *AnomalyDetectionEngine) detectSignatureAnomaly(engine *AnomalyDetectionEngine, traffic *DDoSTrafficDataV3, ip string) (bool, float64) {
	attackPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop)`),
		regexp.MustCompile(`(?i)(<script|javascript:|onerror|onclick)`),
		regexp.MustCompile(`(?i)(\.\./|\.\.\\|%2e%2e)`),
		regexp.MustCompile(`(?i)(;|\|\||&&|\$\()`),
	}

	for _, pattern := range attackPatterns {
		for _, path := range traffic.Paths {
			if pattern.MatchString(path) {
				return true, 0.9
			}
		}
	}

	return false, 0
}

func (s *DDoSProtectionV3Service) CheckRequestV3(r *http.Request) *DDoSCheckResultV3 {
	ip := s.getClientIP(r)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.whitelist[ip] {
		return &DDoSCheckResultV3{Allowed: true, IPStats: s.getOrCreateStatsV3(ip)}
	}

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
			return &DDoSCheckResultV3{
				Allowed:    false,
				Reason:     "ip_blacklisted",
				Severity:   "critical",
				RetryAfter: int(time.Until(expiry).Seconds()),
			}
		}
		delete(s.blacklist, ip)
	}

	stats := s.getOrCreateStatsV3(ip)
	s.recordTrafficV3(ip, r, now)

	if s.config.EnableMLAttackDetection && s.mlDetector != nil {
		isAttack, prediction, attackType := s.mlDetector.DetectAttack(stats)
		stats.MLPrediction = prediction
		stats.AttackType = attackType

		if isAttack {
			stats.ThreatScore += prediction * 0.4
			if prediction > 0.9 {
				s.blacklist[ip] = now.Add(1 * time.Hour)
				stats.IsBlacklisted = true
				return &DDoSCheckResultV3{
					Allowed:      false,
					Reason:       "ml_detected_attack",
					AttackType:   attackType,
					MLPrediction: prediction,
					Severity:     "critical",
				}
			}
		}
	}

	if s.config.EnableSmartTrafficAnalysis {
		traffic := s.trafficData[ip]
		if traffic != nil {
			anomalyScore := s.anomalyEngine.DetectAnomaly(traffic, ip)
			if anomalyScore > 0.7 {
				stats.IsAnomaly = true
				stats.ThreatScore += anomalyScore * 0.3
			}
		}
	}

	if stats.ThreatScore > 0.8 {
		s.blacklist[ip] = now.Add(30 * time.Minute)
		stats.IsBlacklisted = true
		return &DDoSCheckResultV3{
			Allowed:    false,
			Reason:     "high_threat_score",
			Severity:   "critical",
			ThreatScore: stats.ThreatScore,
		}
	}

	stats.RequestCount++
	stats.RequestCountMinute++
	stats.RequestCountHour++
	stats.LastSeen = now

	s.updateGlobalStats(stats)

	return &DDoSCheckResultV3{
		Allowed:      true,
		IPStats:      stats,
		ThreatScore:  stats.ThreatScore,
		MLPrediction: stats.MLPrediction,
		AttackType:   stats.AttackType,
	}
}

type DDoSCheckResultV3 struct {
	Allowed      bool
	Reason      string
	Severity    string
	IPStats     *DDoSIPStatsV3
	RetryAfter  int
	ThreatScore float64
	MLPrediction float64
	AttackType  string
}

func (e *AnomalyDetectionEngine) DetectAnomaly(traffic *DDoSTrafficDataV3, ip string) float64 {
	var totalScore float64
	var totalWeight float64

	for _, method := range e.detectionMethods {
		detected, score := method.DetectFunc(e, traffic, ip)
		if detected {
			totalScore += score * method.Weight
			totalWeight += method.Weight
		}
	}

	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0
}

func (s *DDoSProtectionV3Service) getOrCreateStatsV3(ip string) *DDoSIPStatsV3 {
	stats, exists := s.ipStats[ip]
	if !exists {
		stats = &DDoSIPStatsV3{
			IP:                  ip,
			FirstSeen:          time.Now(),
			LastSeen:           time.Now(),
			TrafficSignature:   &TrafficSignature{},
		}
		s.ipStats[ip] = stats
	}
	return stats
}

func (s *DDoSProtectionV3Service) recordTrafficV3(ip string, r *http.Request, now time.Time) {
	traffic, exists := s.trafficData[ip]
	if !exists {
		traffic = &DDoSTrafficDataV3{
			RequestTimes:     make([]time.Time, 0, 1000),
			RequestSizes:     make([]int64, 0, 1000),
			Methods:          make([]string, 0, 100),
			Paths:            make([]string, 0, 100),
			UserAgents:       make([]string, 0, 100),
			RequestIntervals: make([]time.Duration, 0, 100),
		}
		s.trafficData[ip] = traffic
	}

	traffic.mu.Lock()
	defer traffic.mu.Unlock()

	traffic.RequestTimes = append(traffic.RequestTimes, now)
	if len(traffic.RequestTimes) > 1000 {
		traffic.RequestTimes = traffic.RequestTimes[len(traffic.RequestTimes)-1000:]
	}

	if len(traffic.RequestTimes) >= 2 {
		interval := traffic.RequestTimes[len(traffic.RequestTimes)-1].Sub(traffic.RequestTimes[len(traffic.RequestTimes)-2])
		traffic.RequestIntervals = append(traffic.RequestIntervals, interval)
		if len(traffic.RequestIntervals) > 100 {
			traffic.RequestIntervals = traffic.RequestIntervals[len(traffic.RequestIntervals)-100:]
		}
	}

	traffic.Paths = append(traffic.Paths, r.URL.Path)
	if len(traffic.Paths) > 100 {
		traffic.Paths = traffic.Paths[len(traffic.Paths)-100:]
	}

	traffic.Methods = append(traffic.Methods, r.Method)
	if len(traffic.Methods) > 100 {
		traffic.Methods = traffic.Methods[len(traffic.Methods)-100:]
	}

	if ua := r.Header.Get("User-Agent"); ua != "" {
		traffic.UserAgents = append(traffic.UserAgents, ua)
		if len(traffic.UserAgents) > 100 {
			traffic.UserAgents = traffic.UserAgents[len(traffic.UserAgents)-100:]
		}
	}
}

func (s *DDoSProtectionV3Service) updateGlobalStats(stats *DDoSIPStatsV3) {
	s.globalStats.mu.Lock()
	defer s.globalStats.mu.Unlock()

	s.globalStats.TotalRequests++

	if stats.ThreatScore > 0.7 {
		s.globalStats.TotalBlocked++
	}

	if stats.MLPrediction > 0.8 {
		if _, exists := s.globalStats.GlobalThreats[stats.IP]; !exists {
			s.globalStats.ActiveAttacks++
			s.globalStats.GlobalThreats[stats.IP] = &GlobalThreat{
				ThreatID:   fmt.Sprintf("threat_%s", stats.IP),
				ThreatType: stats.AttackType,
				Severity:   stats.MLPrediction,
				SourceIPs:  []string{stats.IP},
				StartTime:  time.Now(),
				Status:     "active",
			}
		}
	}
}

func (s *DDoSProtectionV3Service) getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	return r.RemoteAddr
}

func (s *DDoSProtectionV3Service) cleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-30 * time.Minute)
		for ip, stats := range s.ipStats {
			if stats.LastSeen.Before(cutoff) {
				delete(s.ipStats, ip)
				delete(s.trafficData, ip)
			}
		}

		now := time.Now()
		for ip, expiry := range s.blacklist {
			if now.After(expiry) {
				delete(s.blacklist, ip)
			}
		}
		s.mu.Unlock()
	}
}

func (s *DDoSProtectionV3Service) statsUpdateRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		for ip, stats := range s.ipStats {
			if stats.RequestCountMinute > int64(s.config.MLModelThreshold*1000) {
				stats.ThreatScore += 0.1
			}

			_ = ip
		}

		s.globalStats.mu.Lock()
		s.globalStats.PeakQPS = math.Max(s.globalStats.PeakQPS, float64(len(s.ipStats))/60.0)
		s.globalStats.mu.Unlock()

		s.mu.Unlock()
	}
}

func (s *DDoSProtectionV3Service) AddToBlacklist(ip string, reason string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist[ip] = time.Now().Add(duration)

	if stats := s.ipStats[ip]; stats != nil {
		stats.IsBlacklisted = true
	}
}

func (s *DDoSProtectionV3Service) RemoveFromBlacklist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blacklist, ip)

	if stats := s.ipStats[ip]; stats != nil {
		stats.IsBlacklisted = false
	}
}

func (s *DDoSProtectionV3Service) GetGlobalStats() GlobalDDoSStats {
	s.globalStats.mu.Lock()
	defer s.globalStats.mu.Unlock()

	return GlobalDDoSStats{
		TotalRequests:  s.globalStats.TotalRequests,
		TotalBlocked:   s.globalStats.TotalBlocked,
		ActiveAttacks:  s.globalStats.ActiveAttacks,
		PeakQPS:       s.globalStats.PeakQPS,
		CleanedTraffic: s.globalStats.CleanedTraffic,
	}
}

func (s *DDoSProtectionV3Service) GetNodeHealth() []*HealthStatus {
	if s.nodeHealthMonitor == nil {
		return []*HealthStatus{}
	}

	s.nodeHealthMonitor.mu.RLock()
	defer s.nodeHealthMonitor.mu.RUnlock()

	var allHealth []*HealthStatus
	for _, log := range s.nodeHealthMonitor.healthLog {
		if len(log) > 0 {
			status := log[len(log)-1]
			allHealth = append(allHealth, &status)
		}
	}

	return allHealth
}

func (s *DDoSProtectionV3Service) TrainMLModel(ctx context.Context, samples []TrainingSample) error {
	if s.mlDetector == nil {
		return fmt.Errorf("ML detector not enabled")
	}
	return s.mlDetector.Train(samples)
}

func (s *DDoSProtectionV3Service) ExportMLModel(ctx context.Context) ([]byte, error) {
	if s.mlDetector == nil {
		return nil, fmt.Errorf("ML detector not enabled")
	}

	exportData := struct {
		TrainingData  []TrainingSample `json:"training_data"`
		FeatureWeights map[string]float64 `json:"feature_weights"`
	}{
		TrainingData:    s.mlDetector.trainingData,
		FeatureWeights: s.mlDetector.featureWeights,
	}

	return []byte(fmt.Sprintf("%v", exportData)), nil
}

func (s *DDoSProtectionV3Service) GetTopThreats(limit int) []*DDoSIPStatsV3 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type threatPair struct {
		ip    string
		stats *DDoSIPStatsV3
	}

	pairs := make([]threatPair, 0, len(s.ipStats))
	for ip, stats := range s.ipStats {
		pairs = append(pairs, threatPair{ip: ip, stats: stats})
	}

	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].stats.ThreatScore > pairs[i].stats.ThreatScore {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]*DDoSIPStatsV3, 0, limit)
	for i := 0; i < limit && i < len(pairs); i++ {
		result = append(result, pairs[i].stats)
	}

	return result
}
