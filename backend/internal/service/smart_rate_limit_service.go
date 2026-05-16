package service

import (
	"math"
	"sync"
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
	ID             string
	Tier           string
	RequestHistory []time.Time
	RiskScore      float64
	LastSeen       time.Time
	TotalRequests  int64
	RateLimitHits  int64
}

type SmartRateLimitConfig struct {
	DefaultRequestsPerMin int
	DefaultBurstLimit     int
	EnableAdaptiveLimit   bool
	EnableRiskBasedLimit  bool
	Tiers                []RateLimitTier
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
}

type SmartRateLimitService struct {
	clients    map[string]*ClientRecord
	mu         sync.RWMutex
	config     SmartRateLimitConfig
	tierMap    map[string]RateLimitTier
}

func NewSmartRateLimitService(config ...SmartRateLimitConfig) *SmartRateLimitService {
	cfg := SmartRateLimitConfig{
		DefaultRequestsPerMin: 60,
		DefaultBurstLimit:     10,
		EnableAdaptiveLimit:   true,
		EnableRiskBasedLimit:  true,
		Tiers:                defaultTiers,
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
		clients: make(map[string]*ClientRecord),
		config:  cfg,
		tierMap: tierMap,
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

	s.cleanOldRequests(client, now)

	tier := s.determineTier(client)
	limit := s.calculateLimit(tier, riskScore)

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
	}

	if result.Allowed {
		client.RequestHistory = append(client.RequestHistory, now)
	} else {
		client.RateLimitHits++
	}

	return result
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
	successRate := float64(client.TotalRequests - client.RateLimitHits) / math.Max(1, float64(client.TotalRequests))

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
		"client_id":       client.ID,
		"tier":            client.Tier,
		"risk_score":      client.RiskScore,
		"total_requests":  client.TotalRequests,
		"rate_limit_hits": client.RateLimitHits,
		"current_requests": len(client.RequestHistory),
		"last_seen":       client.LastSeen,
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
}

func (s *SmartRateLimitService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalClients := len(s.clients)
	totalRequests := int64(0)
	totalHits := int64(0)

	for _, client := range s.clients {
		totalRequests += client.TotalRequests
		totalHits += client.RateLimitHits
	}

	return map[string]interface{}{
		"total_clients":     totalClients,
		"total_requests":    totalRequests,
		"total_limit_hits":  totalHits,
		"hit_rate":          float64(totalHits) / math.Max(1, float64(totalRequests)),
		"adaptive_enabled":  s.config.EnableAdaptiveLimit,
		"risk_based_enabled": s.config.EnableRiskBasedLimit,
	}
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
