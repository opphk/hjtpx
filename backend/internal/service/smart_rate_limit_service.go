package service

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type RateLimitTier struct {
	Name           string
	RequestsPerMin int
	BurstLimit     int
}

var defaultTiers = []RateLimitTier{
	{Name: "normal", RequestsPerMin: 60, BurstLimit: 10},
	{Name: "medium", RequestsPerMin: 120, BurstLimit: 20},
	{Name: "high", RequestsPerMin: 300, BurstLimit: 50},
	{Name: "premium", RequestsPerMin: 1000, BurstLimit: 100},
}

type ClientRecord struct {
	ID               string
	Tier             string
	RequestHistory   []time.Time
	RiskScore        float64
	LastSeen         time.Time
	TotalRequests    int64
	RateLimitHits    int64
	SuccessRequests  int64
	HotspotScore     float64
	LastTierChange   time.Time
}

type SmartRateLimitConfig struct {
	DefaultRequestsPerMin  int
	DefaultBurstLimit      int
	EnableAdaptiveLimit    bool
	EnableRiskBasedLimit   bool
	EnableHotspotDetection bool
	EnablePredictiveLimit  bool
	HotspotThreshold       float64
	Tiers                 []RateLimitTier
	HistoryWindow         time.Duration
}

type SmartRateLimitResult struct {
	Allowed      bool
	CurrentCount int
	Limit        int
	ResetTime    time.Time
	RetryAfter   int
	Tier         string
	RiskScore    float64
	HotspotScore float64
	IsPredicted  bool
}

type HotspotInfo struct {
	Key          string
	RequestCount int64
	Score        float64
	LastAccess   time.Time
}

type SmartRateLimitService struct {
	clients        map[string]*ClientRecord
	hotspots       map[string]*HotspotInfo
	mu             sync.RWMutex
	config         SmartRateLimitConfig
	tierMap        map[string]RateLimitTier
	totalRequests  atomic.Int64
	hitsCount      atomic.Int64
}

func NewSmartRateLimitService(config ...SmartRateLimitConfig) *SmartRateLimitService {
	cfg := SmartRateLimitConfig{
		DefaultRequestsPerMin:  60,
		DefaultBurstLimit:      10,
		EnableAdaptiveLimit:    true,
		EnableRiskBasedLimit:   true,
		EnableHotspotDetection: true,
		EnablePredictiveLimit:  true,
		HotspotThreshold:       0.8,
		Tiers:                 defaultTiers,
		HistoryWindow:         24 * time.Hour,
	}
	if len(config) > 0 {
		cfg = config[0]
		if len(cfg.Tiers) == 0 {
			cfg.Tiers = defaultTiers
		}
	}

	tierMap := make(map[string]RateLimitTier)
	for _, tier := range cfg.Tiers {
		tierMap[tier.Name] = tier
	}

	return &SmartRateLimitService{
		clients:       make(map[string]*ClientRecord),
		hotspots:      make(map[string]*HotspotInfo),
		config:        cfg,
		tierMap:       tierMap,
	}
}

func (s *SmartRateLimitService) CheckRateLimit(clientID string, riskScore float64) *SmartRateLimitResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	client := s.getOrCreateClient(clientID)
	client.LastSeen = now
	client.TotalRequests++
	client.RiskScore = riskScore
	s.totalRequests.Add(1)

	s.cleanOldRequests(client, now)

	if s.config.EnableHotspotDetection {
		s.updateHotspot(clientID)
	}

	tier := s.determineTier(client)
	limit := s.calculateLimit(tier, riskScore)

	if s.config.EnableHotspotDetection {
		limit = s.applyHotspotLimit(clientID, limit)
	}

	if s.config.EnablePredictiveLimit {
		limit = s.applyPredictiveLimit(client, limit)
	}

	currentCount := len(client.RequestHistory)
	resetTime := now.Add(time.Minute)
	retryAfter := 60

	result := &SmartRateLimitResult{
		Allowed:      currentCount < limit,
		CurrentCount: currentCount,
		Limit:        limit,
		ResetTime:    resetTime,
		RetryAfter:   retryAfter,
		Tier:         tier.Name,
		RiskScore:    riskScore,
		HotspotScore: client.HotspotScore,
	}

	if result.Allowed {
		client.RequestHistory = append(client.RequestHistory, now)
		client.SuccessRequests++
	} else {
		client.RateLimitHits++
		s.hitsCount.Add(1)
	}

	return result
}

func (s *SmartRateLimitService) updateHotspot(clientID string) {
	if info, exists := s.hotspots[clientID]; exists {
		info.RequestCount++
		info.LastAccess = time.Now()
		info.Score = math.Min(1.0, float64(info.RequestCount)/1000.0)
	} else {
		s.hotspots[clientID] = &HotspotInfo{
			Key:         clientID,
			RequestCount: 1,
			Score:       0.001,
			LastAccess:  time.Now(),
		}
	}
}

func (s *SmartRateLimitService) applyHotspotLimit(clientID string, baseLimit int) int {
	if info, exists := s.hotspots[clientID]; exists {
		if info.Score > s.config.HotspotThreshold {
			reductionFactor := 1.0 - (info.Score - s.config.HotspotThreshold)
			if reductionFactor < 0.3 {
				reductionFactor = 0.3
			}
			return int(float64(baseLimit) * reductionFactor)
		}
	}
	return baseLimit
}

func (s *SmartRateLimitService) applyPredictiveLimit(client *ClientRecord, baseLimit int) int {
	if client.TotalRequests < 10 {
		return baseLimit
	}

	successRate := float64(client.SuccessRequests) / float64(client.TotalRequests)
	requestRate := float64(len(client.RequestHistory))

	var predictionFactor float64 = 1.0
	if successRate > 0.95 && requestRate < 30 {
		predictionFactor = 1.2
	} else if successRate < 0.5 {
		predictionFactor = 0.5
	}

	return int(float64(baseLimit) * predictionFactor)
}

func (s *SmartRateLimitService) getOrCreateClient(clientID string) *ClientRecord {
	client, exists := s.clients[clientID]
	if !exists {
		client = &ClientRecord{
			ID:             clientID,
			Tier:           "normal",
			RequestHistory: make([]time.Time, 0),
			RiskScore:      0,
			LastSeen:       time.Now(),
		}
		s.clients[clientID] = client
	}
	return client
}

func (s *SmartRateLimitService) cleanOldRequests(client *ClientRecord, now time.Time) {
	cutoff := now.Add(-time.Minute)
	filtered := make([]time.Time, 0)
	for _, t := range client.RequestHistory {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	client.RequestHistory = filtered
}

func (s *SmartRateLimitService) determineTier(client *ClientRecord) RateLimitTier {
	if s.config.EnableAdaptiveLimit {
		return s.calculateAdaptiveTier(client)
	}

	if tier, exists := s.tierMap[client.Tier]; exists {
		return tier
	}
	return s.tierMap["normal"]
}

func (s *SmartRateLimitService) calculateAdaptiveTier(client *ClientRecord) RateLimitTier {
	requestFrequency := float64(len(client.RequestHistory))
	successRate := float64(client.TotalRequests-client.RateLimitHits) / math.Max(1, float64(client.TotalRequests))

	var selectedTier RateLimitTier
	if successRate > 0.99 && requestFrequency < 30 {
		selectedTier = s.tierMap["premium"]
	} else if successRate > 0.95 && requestFrequency < 60 {
		selectedTier = s.tierMap["high"]
	} else if successRate > 0.9 {
		selectedTier = s.tierMap["medium"]
	} else {
		selectedTier = s.tierMap["normal"]
	}

	if selectedTier.Name != client.Tier {
		client.LastTierChange = time.Now()
	}

	return selectedTier
}

func (s *SmartRateLimitService) calculateLimit(tier RateLimitTier, riskScore float64) int {
	baseLimit := tier.RequestsPerMin

	if s.config.EnableRiskBasedLimit && riskScore > 0 {
		riskMultiplier := 1.0 - (riskScore / 200.0)
		if riskMultiplier < 0.2 {
			riskMultiplier = 0.2
		}
		baseLimit = int(float64(baseLimit) * riskMultiplier)
	}

	if baseLimit < 5 {
		baseLimit = 5
	}

	return baseLimit
}

func (s *SmartRateLimitService) GetTopHotspots(limit int) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type hotspotEntry struct {
		key  string
		info *HotspotInfo
	}

	entries := make([]hotspotEntry, 0, len(s.hotspots))
	for key, info := range s.hotspots {
		entries = append(entries, hotspotEntry{key: key, info: info})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].info.Score > entries[j].info.Score
	})

	result := make([]map[string]interface{}, 0, limit)
	for i := 0; i < limit && i < len(entries); i++ {
		result = append(result, map[string]interface{}{
			"key":           entries[i].key,
			"request_count": entries[i].info.RequestCount,
			"score":         entries[i].info.Score,
			"last_access":   entries[i].info.LastAccess,
		})
	}

	return result
}

func (s *SmartRateLimitService) GetTierDistribution() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	distribution := make(map[string]int)
	for _, client := range s.clients {
		distribution[client.Tier]++
	}
	return distribution
}

func (s *SmartRateLimitService) SetClientTier(clientID, tier string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		client.Tier = tier
	}
}

func (s *SmartRateLimitService) GetClientStats(clientID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return nil
	}

	return map[string]interface{}{
		"client_id":        client.ID,
		"tier":             client.Tier,
		"risk_score":       client.RiskScore,
		"total_requests":   client.TotalRequests,
		"success_requests":  client.SuccessRequests,
		"rate_limit_hits":  client.RateLimitHits,
		"current_requests": len(client.RequestHistory),
		"hotspot_score":    client.HotspotScore,
		"last_tier_change": client.LastTierChange,
		"last_seen":        client.LastSeen,
	}
}

func (s *SmartRateLimitService) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.HistoryWindow)
	for id, client := range s.clients {
		if client.LastSeen.Before(cutoff) {
			delete(s.clients, id)
		}
	}

	for key, info := range s.hotspots {
		if info.LastAccess.Before(cutoff) {
			delete(s.hotspots, key)
		}
	}
}

func (s *SmartRateLimitService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalClients := len(s.clients)
	tierDistribution := make(map[string]int)
	hotspotCount := len(s.hotspots)

	for _, client := range s.clients {
		tierDistribution[client.Tier]++
	}

	return map[string]interface{}{
		"total_clients":       totalClients,
		"total_requests":      s.totalRequests.Load(),
		"total_limit_hits":    s.hitsCount.Load(),
		"hit_rate":           float64(s.hitsCount.Load()) / math.Max(1, float64(s.totalRequests.Load())),
		"adaptive_enabled":    s.config.EnableAdaptiveLimit,
		"risk_based_enabled": s.config.EnableRiskBasedLimit,
		"hotspot_enabled":     s.config.EnableHotspotDetection,
		"predictive_enabled":  s.config.EnablePredictiveLimit,
		"hotspot_count":      hotspotCount,
		"tier_distribution":  tierDistribution,
	}
}

func (s *SmartRateLimitService) GetHotspots(limit int) []map[string]interface{} {
	return s.GetTopHotspots(limit)
}

func (s *SmartRateLimitService) GetClientList(page, pageSize int) ([]map[string]interface{}, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*ClientRecord, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].TotalRequests > clients[j].TotalRequests
	})

	total := len(clients)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []map[string]interface{}{}, total
	}
	if end > total {
		end = total
	}

	result := make([]map[string]interface{}, 0, end-start)
	for i := start; i < end; i++ {
		client := clients[i]
		result = append(result, map[string]interface{}{
			"client_id":        client.ID,
			"tier":             client.Tier,
			"total_requests":   client.TotalRequests,
			"success_requests": client.SuccessRequests,
			"rate_limit_hits":  client.RateLimitHits,
			"risk_score":       client.RiskScore,
			"hotspot_score":    client.HotspotScore,
			"last_seen":        client.LastSeen,
		})
	}

	return result, total
}

func (s *SmartRateLimitService) UpdateConfig(config SmartRateLimitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config

	s.tierMap = make(map[string]RateLimitTier)
	for _, tier := range config.Tiers {
		s.tierMap[tier.Name] = tier
	}
}
