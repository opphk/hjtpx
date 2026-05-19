package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type RootCauseAnalyzer struct {
	mu               sync.RWMutex
	knowledgeBase    *KnowledgeBase
	dependencyGraph  *ServiceDependencyGraph
	correlationRules []CorrelationRule
	analysisHistory  []RootCauseAnalysis
}

type KnowledgeBase struct {
	Symptoms    map[string]*Symptom    `json:"symptoms"`
	Causes      map[string]*Cause      `json:"causes"`
	Rules       []*DiagnosticRule      `json:"rules"`
	Services    map[string]*ServiceInfo `json:"services"`
}

type Symptom struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Severity      string   `json:"severity"`
	Indicators    []string `json:"indicators"`
	RelatedCauses []string `json:"related_causes"`
}

type Cause struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Symptoms     []string `json:"symptoms"`
	Services     []string `json:"services"`
	Occurrences  int      `json:"occurrences"`
	Solutions    []string `json:"solutions"`
}

type DiagnosticRule struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RootCauseConditions    []RootCauseCondition `json:"conditions"`
	Conclusions  []Conclusion `json:"conclusions"`
	Priority     int      `json:"priority"`
	Confidence   float64  `json:"confidence"`
}

type RootCauseCondition struct {
	Type       string      `json:"type"`
	Metric     string      `json:"metric"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value"`
	Weight     float64     `json:"weight"`
}

type Conclusion struct {
	Type     string  `json:"type"`
	CauseID  string  `json:"cause_id"`
	Weight   float64 `json:"weight"`
	Action   string  `json:"action"`
}

type ServiceInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Dependencies []string `json:"dependencies"`
	Metrics     []string `json:"metrics"`
	Status      string   `json:"status"`
}

type ServiceDependencyGraph struct {
	Nodes map[string]*ServiceGraphNode `json:"nodes"`
	Edges []*ServiceGraphEdge          `json:"edges"`
}

type ServiceGraphNode struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type ServiceGraphEdge struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Type        string  `json:"type"`
	Weight      float64 `json:"weight"`
	Latency     float64 `json:"latency"`
}

type CorrelationRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SourceEvent string   `json:"source_event"`
	TargetEvent string   `json:"target_event"`
	TimeWindow  int      `json:"time_window"`
	Correlation float64  `json:"correlation"`
	Confidence  float64  `json:"confidence"`
}

type RootCauseAnalysis struct {
	ID             string          `json:"id"`
	Timestamp      time.Time       `json:"timestamp"`
	Symptoms       []SymptomMatch  `json:"symptoms"`
	IdentifiedCause *RootCause     `json:"identified_cause"`
	ContributingFactors []Factor   `json:"contributing_factors"`
	Evidence       []Evidence      `json:"evidence"`
	Confidence     float64         `json:"confidence"`
	Duration       time.Duration   `json:"duration"`
	Actions        []Action        `json:"actions"`
}

type SymptomMatch struct {
	Symptom    Symptom    `json:"symptom"`
	MatchScore float64    `json:"match_score"`
	Evidence   []Evidence `json:"evidence"`
}

type Factor struct {
	Name       string  `json:"name"`
	Impact     float64 `json:"impact"`
	Weight     float64 `json:"weight"`
	Evidence   []Evidence `json:"evidence"`
}

type Evidence struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	Timestamp   time.Time   `json:"timestamp"`
	Source      string      `json:"source"`
	Weight      float64     `json:"weight"`
}

type Action struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Automated   bool   `json:"automated"`
}

type ServiceDependency struct {
	SourceService string  `json:"source_service"`
	TargetService string  `json:"target_service"`
	DependencyType string `json:"dependency_type"`
	HealthImpact  float64 `json:"health_impact"`
}

type IncidentTimeline struct {
	Events      []TimelineEvent `json:"events"`
	RootCause   *RootCause      `json:"root_cause"`
	Conclusion  string          `json:"conclusion"`
}

type TimelineEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	Service     string    `json:"service"`
	Severity    string    `json:"severity"`
}

func NewRootCauseAnalyzer() *RootCauseAnalyzer {
	analyzer := &RootCauseAnalyzer{
		knowledgeBase:    NewKnowledgeBase(),
		dependencyGraph:  NewServiceDependencyGraph(),
		correlationRules: make([]CorrelationRule, 0),
		analysisHistory:  make([]RootCauseAnalysis, 0),
	}
	analyzer.initializeRules()
	return analyzer
}

func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		Symptoms:    make(map[string]*Symptom),
		Causes:      make(map[string]*Cause),
		Rules:       make([]*DiagnosticRule, 0),
		Services:    make(map[string]*ServiceInfo),
	}
}

func NewServiceDependencyGraph() *ServiceDependencyGraph {
	return &ServiceDependencyGraph{
		Nodes: make(map[string]*ServiceGraphNode),
		Edges: make([]*ServiceGraphEdge, 0),
	}
}

func (a *RootCauseAnalyzer) initializeRules() {
	a.knowledgeBase.Symptoms = map[string]*Symptom{
		"high_latency": {
			ID:            "symptom-001",
			Name:          "高延迟",
			Description:   "系统响应时间超过正常阈值",
			Category:      "performance",
			Severity:      "warning",
			Indicators:    []string{"response_time > 200ms", "p99 > 500ms"},
			RelatedCauses: []string{"cause-001", "cause-002", "cause-003"},
		},
		"high_error_rate": {
			ID:            "symptom-002",
			Name:          "高错误率",
			Description:   "错误请求比例超过正常范围",
			Category:      "reliability",
			Severity:      "critical",
			Indicators:    []string{"error_rate > 5%", "5xx_count > 10"},
			RelatedCauses: []string{"cause-004", "cause-005", "cause-006"},
		},
		"cpu_spike": {
			ID:            "symptom-003",
			Name:          "CPU突增",
			Description:   "CPU使用率异常升高",
			Category:      "resource",
			Severity:      "warning",
			Indicators:    []string{"cpu_usage > 80%", "cpu_usage持续5分钟"},
			RelatedCauses: []string{"cause-001", "cause-007"},
		},
		"memory_leak": {
			ID:            "symptom-004",
			Name:          "内存泄漏",
			Description:   "内存使用持续增长未释放",
			Category:      "resource",
			Severity:      "critical",
			Indicators:    []string{"memory_usage > 85%", "内存持续增长"},
			RelatedCauses: []string{"cause-007", "cause-008"},
		},
		"connection_timeout": {
			ID:            "symptom-005",
			Name:          "连接超时",
			Description:   "数据库或服务连接超时",
			Category:      "connectivity",
			Severity:      "critical",
			Indicators:    []string{"connection_timeout > 0", "pool_exhausted"},
			RelatedCauses: []string{"cause-009", "cause-010"},
		},
	}

	a.knowledgeBase.Causes = map[string]*Cause{
		"cause-001": {
			ID:          "cause-001",
			Name:        "数据库查询慢",
			Description: "数据库查询性能下降",
			Category:    "database",
			Symptoms:    []string{"symptom-001", "symptom-002"},
			Services:    []string{"database", "api"},
			Occurrences: 150,
			Solutions:   []string{"添加索引", "优化查询", "使用缓存"},
		},
		"cause-002": {
			ID:          "cause-002",
			Name:        "网络延迟",
			Description: "网络通信延迟增加",
			Category:    "network",
			Symptoms:    []string{"symptom-001"},
			Services:    []string{"gateway", "api"},
			Occurrences: 80,
			Solutions:   []string{"检查网络配置", "使用CDN", "优化路由"},
		},
		"cause-003": {
			ID:          "cause-003",
			Name:        "应用代码问题",
			Description: "应用程序代码效率低",
			Category:    "application",
			Symptoms:    []string{"symptom-001", "symptom-003"},
			Services:    []string{"api", "worker"},
			Occurrences: 120,
			Solutions:   []string{"代码审查", "性能优化", "重构"},
		},
		"cause-004": {
			ID:          "cause-004",
			Name:        "服务宕机",
			Description: "依赖服务不可用",
			Category:    "service",
			Symptoms:    []string{"symptom-002", "symptom-005"},
			Services:    []string{"api", "gateway"},
			Occurrences: 45,
			Solutions:   []string{"检查服务状态", "启动备用服务", "告警通知"},
		},
		"cause-005": {
			ID:          "cause-005",
			Name:        "配置错误",
			Description: "系统配置参数错误",
			Category:    "configuration",
			Symptoms:    []string{"symptom-002"},
			Services:    []string{"api", "worker"},
			Occurrences: 30,
			Solutions:   []string{"检查配置", "回滚配置", "对比环境"},
		},
		"cause-006": {
			ID:          "cause-006",
			Name:        "资源耗尽",
			Description: "系统资源耗尽",
			Category:    "resource",
			Symptoms:    []string{"symptom-002", "symptom-003", "symptom-004"},
			Services:    []string{"api", "worker", "database"},
			Occurrences: 65,
			Solutions:   []string{"扩容", "释放资源", "优化资源使用"},
		},
		"cause-007": {
			ID:          "cause-007",
			Name:        "内存泄漏",
			Description: "应用程序内存泄漏",
			Category:    "application",
			Symptoms:    []string{"symptom-003", "symptom-004"},
			Services:    []string{"api", "worker"},
			Occurrences: 55,
			Solutions:   []string{"代码审查", "重启服务", "内存分析"},
		},
		"cause-008": {
			ID:          "cause-008",
			Name:        "缓存问题",
			Description: "缓存配置或容量问题",
			Category:    "cache",
			Symptoms:    []string{"symptom-001", "symptom-004"},
			Services:    []string{"cache", "api"},
			Occurrences: 40,
			Solutions:   []string{"检查缓存配置", "增加缓存容量", "清理缓存"},
		},
		"cause-009": {
			ID:          "cause-009",
			Name:        "数据库连接池满",
			Description: "数据库连接池资源耗尽",
			Category:    "database",
			Symptoms:    []string{"symptom-005"},
			Services:    []string{"database", "api"},
			Occurrences: 75,
			Solutions:   []string{"增加连接池大小", "优化连接使用", "增加数据库实例"},
		},
		"cause-010": {
			ID:          "cause-010",
			Name:        "网络分区",
			Description: "网络分区或中断",
			Category:    "network",
			Symptoms:    []string{"symptom-005"},
			Services:    []string{"gateway", "api"},
			Occurrences: 20,
			Solutions:   []string{"检查网络", "使用备用网络", "联系网络团队"},
		},
	}

	a.knowledgeBase.Services = map[string]*ServiceInfo{
		"api": {
			ID:          "api",
			Name:        "API服务",
			Type:        "backend",
			Dependencies: []string{"database", "cache"},
			Metrics:     []string{"response_time", "error_rate", "cpu_usage"},
			Status:      "healthy",
		},
		"database": {
			ID:          "database",
			Name:        "数据库服务",
			Type:        "storage",
			Dependencies: []string{},
			Metrics:     []string{"query_time", "connections", "cpu_usage"},
			Status:      "healthy",
		},
		"cache": {
			ID:          "cache",
			Name:        "缓存服务",
			Type:        "storage",
			Dependencies: []string{},
			Metrics:     []string{"hit_rate", "memory_usage", "connections"},
			Status:      "healthy",
		},
		"gateway": {
			ID:          "gateway",
			Name:        "网关服务",
			Type:        "gateway",
			Dependencies: []string{"api"},
			Metrics:     []string{"latency", "throughput", "error_rate"},
			Status:      "healthy",
		},
		"worker": {
			ID:          "worker",
			Name:        "后台任务服务",
			Type:        "worker",
			Dependencies: []string{"database", "cache"},
			Metrics:     []string{"queue_depth", "cpu_usage", "memory_usage"},
			Status:      "healthy",
		},
	}

	a.dependencyGraph.Nodes = map[string]*ServiceGraphNode{
		"gateway": {ID: "gateway", Name: "网关服务", Type: "gateway"},
		"api":     {ID: "api", Name: "API服务", Type: "backend"},
		"database": {ID: "database", Name: "数据库", Type: "storage"},
		"cache":   {ID: "cache", Name: "缓存", Type: "storage"},
		"worker":  {ID: "worker", Name: "后台任务", Type: "worker"},
	}

	a.dependencyGraph.Edges = []*ServiceGraphEdge{
		{Source: "gateway", Target: "api", Type: "http", Weight: 1.0, Latency: 10},
		{Source: "api", Target: "database", Type: "sql", Weight: 0.8, Latency: 50},
		{Source: "api", Target: "cache", Type: "redis", Weight: 0.9, Latency: 5},
		{Source: "worker", Target: "database", Type: "sql", Weight: 0.7, Latency: 50},
		{Source: "worker", Target: "cache", Type: "redis", Weight: 0.6, Latency: 5},
	}

	a.knowledgeBase.Rules = []*DiagnosticRule{
		{
			ID:   "rule-001",
			Name: "高延迟诊断",
			RootCauseConditions: []RootCauseCondition{
				{Type: "metric", Metric: "response_time", Operator: ">", Value: 200, Weight: 0.8},
				{Type: "metric", Metric: "cpu_usage", Operator: ">", Value: 70, Weight: 0.5},
			},
			Conclusions: []Conclusion{
				{Type: "cause", CauseID: "cause-003", Weight: 0.7, Action: "优化应用代码"},
			},
			Priority:   1,
			Confidence: 0.85,
		},
		{
			ID:   "rule-002",
			Name: "错误率高诊断",
			RootCauseConditions: []RootCauseCondition{
				{Type: "metric", Metric: "error_rate", Operator: ">", Value: 5, Weight: 0.9},
				{Type: "metric", Metric: "database_connections", Operator: ">", Value: 80, Weight: 0.6},
			},
			Conclusions: []Conclusion{
				{Type: "cause", CauseID: "cause-009", Weight: 0.8, Action: "增加连接池"},
			},
			Priority:   1,
			Confidence: 0.90,
		},
		{
			ID:   "rule-003",
			Name: "CPU突增诊断",
			RootCauseConditions: []RootCauseCondition{
				{Type: "metric", Metric: "cpu_usage", Operator: ">", Value: 80, Weight: 1.0},
				{Type: "metric", Metric: "memory_usage", Operator: ">", Value: 85, Weight: 0.6},
			},
			Conclusions: []Conclusion{
				{Type: "cause", CauseID: "cause-007", Weight: 0.7, Action: "检查内存泄漏"},
			},
			Priority:   2,
			Confidence: 0.75,
		},
	}

	a.correlationRules = []CorrelationRule{
		{
			ID:           "corr-001",
			Name:         "CPU与延迟相关性",
			SourceEvent:  "cpu_spike",
			TargetEvent:  "high_latency",
			TimeWindow:   300,
			Correlation:  0.85,
			Confidence:   0.90,
		},
		{
			ID:           "corr-002",
			Name:         "数据库与错误率相关性",
			SourceEvent:  "db_slow_query",
			TargetEvent:  "high_error_rate",
			TimeWindow:   600,
			Correlation:  0.75,
			Confidence:   0.85,
		},
	}
}

func (a *RootCauseAnalyzer) Analyze(ctx context.Context, metrics OperationalMetrics, anomalies []LogAnomaly) (*RootCauseAnalysis, error) {
	startTime := time.Now()

	var symptomMatches []SymptomMatch
	for _, anomaly := range anomalies {
		matches := a.matchSymptoms(anomaly)
		symptomMatches = append(symptomMatches, matches...)
	}

	cause := a.identifyRootCause(symptomMatches, metrics)

	contributingFactors := a.identifyContributingFactors(cause, metrics)

	evidence := a.gatherEvidence(symptomMatches, metrics)

	confidence := a.calculateConfidence(cause, symptomMatches, evidence)

	actions := a.suggestActions(cause)

	analysis := &RootCauseAnalysis{
		ID:                  fmt.Sprintf("analysis-%d", time.Now().Unix()),
		Timestamp:           time.Now(),
		Symptoms:            symptomMatches,
		IdentifiedCause:     cause,
		ContributingFactors:  contributingFactors,
		Evidence:             evidence,
		Confidence:           confidence,
		Duration:             time.Since(startTime),
		Actions:              actions,
	}

	a.analysisHistory = append(a.analysisHistory, *analysis)
	if len(a.analysisHistory) > 100 {
		a.analysisHistory = a.analysisHistory[1:]
	}

	return analysis, nil
}

func (a *RootCauseAnalyzer) matchSymptoms(anomaly LogAnomaly) []SymptomMatch {
	var matches []SymptomMatch

	for _, symptom := range a.knowledgeBase.Symptoms {
		score := a.calculateSymptomMatchScore(anomaly, symptom)
		if score > 0.5 {
			evidence := []Evidence{
				{
					Type:        "anomaly",
					Description: fmt.Sprintf("检测到异常: %s", anomaly.Type),
					Value:       anomaly.Score,
					Timestamp:   anomaly.Timestamp,
					Source:      "log_anomaly_detector",
					Weight:      score,
				},
			}

			matches = append(matches, SymptomMatch{
				Symptom:    *symptom,
				MatchScore: score,
				Evidence:   evidence,
			})
		}
	}

	return matches
}

func (a *RootCauseAnalyzer) calculateSymptomMatchScore(anomaly LogAnomaly, symptom *Symptom) float64 {
	score := 0.0

	typeScores := map[string]map[string]float64{
		"error_spike":     {"symptom-002": 0.9, "symptom-001": 0.6},
		"latency_spike":   {"symptom-001": 0.9},
		"memory_leak":     {"symptom-004": 0.9, "symptom-003": 0.5},
		"cpu_spike":       {"symptom-003": 0.9, "symptom-001": 0.4},
		"connection_error": {"symptom-005": 0.9},
	}

	if typeScores, ok := typeScores[anomaly.Type]; ok {
		if s, ok := typeScores[symptom.ID]; ok {
			score += s
		}
	}

	if anomaly.Severity == "critical" {
		if symptom.Severity == "critical" {
			score += 0.2
		}
	}

	score = math.Min(score, 1.0)

	return score
}

func (a *RootCauseAnalyzer) identifyRootCause(symptomMatches []SymptomMatch, metrics OperationalMetrics) *RootCause {
	if len(symptomMatches) == 0 {
		return &RootCause{
			Component:  "unknown",
			Issue:      "无法确定根因",
			Impact:     "未知",
			Confidence: 0.0,
			ContributingFactors: []string{"症状信息不足"},
		}
	}

	causeScores := make(map[string]float64)

	for _, match := range symptomMatches {
		for _, causeID := range match.Symptom.RelatedCauses {
			cause, exists := a.knowledgeBase.Causes[causeID]
			if !exists {
				continue
			}

			baseScore := float64(cause.Occurrences) / 200.0

			ruleScore := a.applyRules(match.Symptom, metrics)

			causeScores[causeID] += match.MatchScore * baseScore * ruleScore
		}
	}

	var maxCauseID string
	var maxScore float64
	for causeID, score := range causeScores {
		if score > maxScore {
			maxScore = score
			maxCauseID = causeID
		}
	}

	cause, exists := a.knowledgeBase.Causes[maxCauseID]
	if !exists {
		return &RootCause{
			Component:  "unknown",
			Issue:      "无法确定根因",
			Impact:     "未知",
			Confidence: 0.1,
		}
	}

	return &RootCause{
		Component:   cause.Services[0],
		Issue:       cause.Name,
		Impact:      cause.Description,
		Confidence:  maxScore,
		ContributingFactors: cause.Solutions,
	}
}

func (a *RootCauseAnalyzer) applyRules(symptom Symptom, metrics OperationalMetrics) float64 {
	for _, rule := range a.knowledgeBase.Rules {
		if a.matchesRootCauseConditions(rule, symptom, metrics) {
			return rule.Confidence
		}
	}
	return 0.5
}

func (a *RootCauseAnalyzer) matchesRootCauseConditions(rule *DiagnosticRule, symptom Symptom, metrics OperationalMetrics) bool {
	matchCount := 0
	totalWeight := 0.0

	for _, condition := range rule.RootCauseConditions {
		totalWeight += condition.Weight

		metricValue := a.getMetricValue(metrics, condition.Metric)
		conditionValue := a.convertToFloat64(condition.Value)

		matched := a.evaluateRootCauseCondition(metricValue, condition.Operator, conditionValue)

		if matched {
			matchCount++
		}
	}

	if totalWeight == 0 {
		return false
	}

	matchRatio := float64(matchCount) / float64(len(rule.RootCauseConditions))

	return matchRatio >= 0.5
}

func (a *RootCauseAnalyzer) getMetricValue(metrics OperationalMetrics, metricName string) float64 {
	switch metricName {
	case "response_time":
		return metrics.AvgResponseTime
	case "cpu_usage":
		return metrics.CPUUsage
	case "memory_usage":
		return metrics.MemoryUsage
	case "error_rate":
		return metrics.ErrorRate
	case "database_connections":
		return float64(metrics.ActiveConnections)
	default:
		return 0
	}
}

func (a *RootCauseAnalyzer) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func (a *RootCauseAnalyzer) evaluateRootCauseCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return math.Abs(value-threshold) < 0.001
	case "!=":
		return math.Abs(value-threshold) >= 0.001
	default:
		return false
	}
}

func (a *RootCauseAnalyzer) identifyContributingFactors(cause *RootCause, metrics OperationalMetrics) []Factor {
	var factors []Factor

	if metrics.CPUUsage > 70 {
		factors = append(factors, Factor{
			Name:    "CPU使用率过高",
			Impact:  0.7,
			Weight:  0.3,
			Evidence: []Evidence{{Type: "metric", Description: "CPU使用率", Value: metrics.CPUUsage}},
		})
	}

	if metrics.MemoryUsage > 80 {
		factors = append(factors, Factor{
			Name:    "内存使用率过高",
			Impact:  0.6,
			Weight:  0.25,
			Evidence: []Evidence{{Type: "metric", Description: "内存使用率", Value: metrics.MemoryUsage}},
		})
	}

	if metrics.ErrorRate > 5 {
		factors = append(factors, Factor{
			Name:    "错误率升高",
			Impact:  0.8,
			Weight:  0.35,
			Evidence: []Evidence{{Type: "metric", Description: "错误率", Value: metrics.ErrorRate}},
		})
	}

	if metrics.DBLatency > 50 {
		factors = append(factors, Factor{
			Name:    "数据库延迟",
			Impact:  0.5,
			Weight:  0.2,
			Evidence: []Evidence{{Type: "metric", Description: "DB延迟", Value: metrics.DBLatency}},
		})
	}

	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Impact > factors[j].Impact
	})

	return factors
}

func (a *RootCauseAnalyzer) gatherEvidence(symptomMatches []SymptomMatch, metrics OperationalMetrics) []Evidence {
	var evidence []Evidence

	for _, match := range symptomMatches {
		evidence = append(evidence, match.Evidence...)
	}

	evidence = append(evidence, Evidence{
		Type:        "metric",
		Description: "CPU使用率",
		Value:       metrics.CPUUsage,
		Timestamp:   time.Now(),
		Source:      "system",
		Weight:      0.7,
	})

	evidence = append(evidence, Evidence{
		Type:        "metric",
		Description: "内存使用率",
		Value:       metrics.MemoryUsage,
		Timestamp:   time.Now(),
		Source:      "system",
		Weight:      0.6,
	})

	evidence = append(evidence, Evidence{
		Type:        "metric",
		Description: "错误率",
		Value:       metrics.ErrorRate,
		Timestamp:   time.Now(),
		Source:      "system",
		Weight:      0.8,
	})

	return evidence
}

func (a *RootCauseAnalyzer) calculateConfidence(cause *RootCause, symptomMatches []SymptomMatch, evidence []Evidence) float64 {
	if len(symptomMatches) == 0 {
		return 0.1
	}

	baseConfidence := cause.Confidence

	symptomWeight := float64(len(symptomMatches)) / 10.0
	if symptomWeight > 1 {
		symptomWeight = 1
	}

	evidenceWeight := float64(len(evidence)) / 20.0
	if evidenceWeight > 1 {
		evidenceWeight = 1
	}

	confidence := baseConfidence * 0.4 + symptomWeight*0.3 + evidenceWeight*0.3

	return math.Min(math.Max(confidence, 0), 1)
}

func (a *RootCauseAnalyzer) suggestActions(cause *RootCause) []Action {
	if cause == nil || len(cause.ContributingFactors) == 0 {
		return []Action{
			{Type: "investigate", Description: "进一步调查问题", Priority: 1, Automated: false},
		}
	}

	var actions []Action

	actions = append(actions, Action{
		Type:        "implement",
		Description: fmt.Sprintf("实施解决方案: %s", cause.ContributingFactors[0]),
		Priority:    1,
		Automated:   false,
	})

	if cause.Confidence < 0.5 {
		actions = append(actions, Action{
			Type:        "investigate",
			Description: "收集更多证据以提高置信度",
			Priority:    2,
			Automated:   false,
		})
	}

	actions = append(actions, Action{
		Type:        "monitor",
		Description: "持续监控相关指标",
		Priority:    3,
		Automated:   true,
	})

	return actions
}

func (a *RootCauseAnalyzer) GetServiceDependencies(ctx context.Context) ([]ServiceDependency, error) {
	var dependencies []ServiceDependency

	for _, edge := range a.dependencyGraph.Edges {
		dependencies = append(dependencies, ServiceDependency{
			SourceService:  edge.Source,
			TargetService: edge.Target,
			DependencyType: edge.Type,
			HealthImpact:  edge.Weight * 100,
		})
	}

	return dependencies, nil
}

func (a *RootCauseAnalyzer) GetIncidentTimeline(ctx context.Context, startTime, endTime time.Time) (*IncidentTimeline, error) {
	events := make([]TimelineEvent, 0)

	var analysisInRange []RootCauseAnalysis
	for _, analysis := range a.analysisHistory {
		if analysis.Timestamp.After(startTime) && analysis.Timestamp.Before(endTime) {
			analysisInRange = append(analysisInRange, analysis)
		}
	}

	if len(analysisInRange) == 0 {
		events = append(events, TimelineEvent{
			Timestamp:   time.Now(),
			EventType:   "normal",
			Description: "系统运行正常",
			Service:     "system",
			Severity:    "info",
		})
	}

	for _, analysis := range analysisInRange {
		events = append(events, TimelineEvent{
			Timestamp:   analysis.Timestamp,
			EventType:   "incident",
			Description: fmt.Sprintf("检测到根因: %s", analysis.IdentifiedCause.Issue),
			Service:     analysis.IdentifiedCause.Component,
			Severity:    "warning",
		})
	}

	var rootCause *RootCause
	if len(analysisInRange) > 0 {
		rootCause = analysisInRange[len(analysisInRange)-1].IdentifiedCause
	}

	return &IncidentTimeline{
		Events:     events,
		RootCause:   rootCause,
		Conclusion: "已完成事件分析",
	}, nil
}

func (a *RootCauseAnalyzer) GetKnowledgeBase(ctx context.Context) (*KnowledgeBase, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.knowledgeBase, nil
}

func (a *RootCauseAnalyzer) GetAnalysisHistory(ctx context.Context, limit int) ([]RootCauseAnalysis, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.analysisHistory) {
		limit = len(a.analysisHistory)
	}

	history := make([]RootCauseAnalysis, limit)
	copy(history, a.analysisHistory[len(a.analysisHistory)-limit:])

	return history, nil
}

func (a *RootCauseAnalyzer) ExportAnalysis(ctx context.Context, format string) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result string

	result += fmt.Sprintf("Root Cause Analysis Report - %s\n", time.Now().Format(time.RFC3339))
	result += fmt.Sprintf("Total Analyses: %d\n\n", len(a.analysisHistory))

	for i, analysis := range a.analysisHistory {
		result += fmt.Sprintf("Analysis #%d:\n", i+1)
		result += fmt.Sprintf("  Timestamp: %s\n", analysis.Timestamp.Format(time.RFC3339))
		result += fmt.Sprintf("  Root Cause: %s\n", analysis.IdentifiedCause.Issue)
		result += fmt.Sprintf("  Component: %s\n", analysis.IdentifiedCause.Component)
		result += fmt.Sprintf("  Confidence: %.2f%%\n", analysis.Confidence*100)
		result += fmt.Sprintf("  Duration: %s\n\n", analysis.Duration.String())
	}

	return []byte(result), nil
}

func (a *RootCauseAnalyzer) AddCorrelationRule(ctx context.Context, rule CorrelationRule) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	rule.ID = fmt.Sprintf("corr-%d", len(a.correlationRules)+1)
	a.correlationRules = append(a.correlationRules, rule)
	return nil
}

func (a *RootCauseAnalyzer) GetCorrelations(ctx context.Context) ([]CorrelationRule, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.correlationRules, nil
}

func (a *RootCauseAnalyzer) AnalyzeWithServiceMap(ctx context.Context, serviceName string, metrics OperationalMetrics) (*ServiceAnalysis, error) {
	service, exists := a.knowledgeBase.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	var impactedServices []string
	for _, dep := range service.Dependencies {
		impactedServices = append(impactedServices, dep)
	}

	metricsImpact := a.calculateServiceMetricsImpact(serviceName, metrics)

	return &ServiceAnalysis{
		Service:         service,
		ImpactedServices: impactedServices,
		MetricsImpact:   metricsImpact,
		Recommendations: a.getServiceRecommendations(serviceName),
	}, nil
}

type ServiceAnalysis struct {
	Service           *ServiceInfo              `json:"service"`
	ImpactedServices  []string                 `json:"impacted_services"`
	MetricsImpact     map[string]float64       `json:"metrics_impact"`
	Recommendations   []string                 `json:"recommendations"`
}

func (a *RootCauseAnalyzer) calculateServiceMetricsImpact(serviceName string, metrics OperationalMetrics) map[string]float64 {
	impacts := make(map[string]float64)

	impacts["cpu_usage"] = metrics.CPUUsage / 100.0
	impacts["memory_usage"] = metrics.MemoryUsage / 100.0
	impacts["error_rate"] = metrics.ErrorRate / 100.0
	impacts["latency"] = metrics.AvgResponseTime / 1000.0

	return impacts
}

func (a *RootCauseAnalyzer) getServiceRecommendations(serviceName string) []string {
	var recommendations []string

	recommendations = append(recommendations, fmt.Sprintf("监控 %s 服务的关键指标", serviceName))
	recommendations = append(recommendations, "设置合理的告警阈值")
	recommendations = append(recommendations, "准备应急预案")

	return recommendations
}

func (a *RootCauseAnalyzer) GetServiceDependencyGraph(ctx context.Context) (*ServiceDependencyGraph, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.dependencyGraph, nil
}

func (a *RootCauseAnalyzer) UpdateServiceStatus(ctx context.Context, serviceName string, status string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	service, exists := a.knowledgeBase.Services[serviceName]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	service.Status = status
	return nil
}

func (a *RootCauseAnalyzer) AddSymptom(ctx context.Context, symptom *Symptom) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	symptom.ID = fmt.Sprintf("symptom-%d", len(a.knowledgeBase.Symptoms)+1)
	a.knowledgeBase.Symptoms[symptom.ID] = symptom
	return nil
}

func (a *RootCauseAnalyzer) AddCause(ctx context.Context, cause *Cause) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cause.ID = fmt.Sprintf("cause-%d", len(a.knowledgeBase.Causes)+1)
	a.knowledgeBase.Causes[cause.ID] = cause
	return nil
}

func (a *RootCauseAnalyzer) GetTopCauses(ctx context.Context, limit int) ([]*Cause, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var causes []*Cause
	for _, cause := range a.knowledgeBase.Causes {
		causes = append(causes, cause)
	}

	sort.Slice(causes, func(i, j int) bool {
		return causes[i].Occurrences > causes[j].Occurrences
	})

	if limit > len(causes) {
		limit = len(causes)
	}

	return causes[:limit], nil
}

func (a *RootCauseAnalyzer) SearchCauses(ctx context.Context, query string) ([]*Cause, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []*Cause
	queryLower := strings.ToLower(query)

	for _, cause := range a.knowledgeBase.Causes {
		if strings.Contains(strings.ToLower(cause.Name), queryLower) ||
			strings.Contains(strings.ToLower(cause.Description), queryLower) ||
			strings.Contains(strings.ToLower(cause.Category), queryLower) {
			results = append(results, cause)
		}
	}

	return results, nil
}
