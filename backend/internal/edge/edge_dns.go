package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type DNSRecordType string

const (
	DNSRecordA     DNSRecordType = "A"
	DNSRecordAAAA  DNSRecordType = "AAAA"
	DNSRecordCNAME DNSRecordType = "CNAME"
	DNSRecordTXT   DNSRecordType = "TXT"
)

type DNSRecord struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Type      DNSRecordType `json:"type"`
	Value     string        `json:"value"`
	TTL       int           `json:"ttl"`
	Priority  int           `json:"priority"`
	Weight    int           `json:"weight"`
	Region    Region        `json:"region,omitempty"`
	Country   string        `json:"country,omitempty"`
	LatencyMs float64       `json:"latency_ms"`
	Healthy   bool          `json:"healthy"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type DNSZone struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Domain      string       `json:"domain"`
	Records     []*DNSRecord `json:"records"`
	TTL         int           `json:"ttl"`
	PrimaryNode string       `json:"primary_node"`
	DNSSEC      bool          `json:"dnssec"`
	CreatedAt   time.Time    `json:"created_at"`
}

type GeoDNSRule struct {
	ID           string   `json:"id"`
	ZoneID       string   `json:"zone_id"`
	Pattern      string   `json:"pattern"`
	MatchType    string   `json:"match_type"`
	Regions      []Region `json:"regions"`
	Countries    []string `json:"countries"`
	ASNRanges    []string `json:"asn_ranges"`
	RecordValues []string `json:"record_values"`
	TTL          int      `json:"ttl"`
	Priority     int      `json:"priority"`
	Enabled      bool     `json:"enabled"`
}

type DNSResolver struct {
	zone            *DNSZone
	rules           map[string]*GeoDNSRule
	ruleIndex       map[int][]*GeoDNSRule
	healthCheckers  map[string]*HealthChecker
	nodeManager     *EdgeNodeManager
	redisClient     *redis.Client
	cache           *DNSCache
	mu              sync.RWMutex
	resolverTimeout time.Duration
	maxRetries      int
	version         int64
}

type DNSCache struct {
	entries  map[string]*DNSCacheEntry
	mu       sync.RWMutex
	maxSize  int
	hits     int64
	misses   int64
}

type DNSCacheEntry struct {
	Records    []*DNSRecord
	ExpiresAt  time.Time
	GeoMatch   bool
	ClientIP  string
}

type DNSQuery struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	ClientIP    string   `json:"client_ip"`
	GeoLocation *GeoLocation `json:"geo_location,omitempty"`
	EDNSClientSubnet net.IP `json:"edns_client_subnet,omitempty"`
	DoH         bool     `json:"doh"`
	Recursion   bool     `json:"recursion"`
}

type DNSResponse struct {
	Records   []*DNSRecord  `json:"records"`
	TTL       int           `json:"ttl"`
	FromCache bool          `json:"from_cache"`
	LatencyMs float64       `json:"latency_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

type HealthChecker struct {
	Target     string
	Interval   time.Duration
	Timeout    time.Duration
	Threshold  int
	Failures   int
	Healthy    bool
	LastCheck  time.Time
	CheckFunc  func(string) (bool, float64)
}

func NewDNSResolver(zone *DNSZone, redisClient *redis.Client, nodeManager *EdgeNodeManager) *DNSResolver {
	resolver := &DNSResolver{
		zone:            zone,
		rules:           make(map[string]*GeoDNSRule),
		ruleIndex:       make(map[int][]*GeoDNSRule),
		healthCheckers:  make(map[string]*HealthChecker),
		nodeManager:     nodeManager,
		redisClient:     redisClient,
		cache:           NewDNSCache(10000),
		resolverTimeout: 5 * time.Second,
		maxRetries:      3,
		version:         1,
	}

	if zone != nil {
		for _, record := range zone.Records {
			resolver.healthCheckers[record.Value] = &HealthChecker{
				Target:    record.Value,
				Interval:  10 * time.Second,
				Timeout:   3 * time.Second,
				Threshold: 3,
				Healthy:   true,
				CheckFunc: resolver.checkHealth,
			}
		}
	}

	go resolver.startHealthChecks()

	return resolver
}

func NewDNSCache(maxSize int) *DNSCache {
	return &DNSCache{
		entries: make(map[string]*DNSCacheEntry),
		maxSize: maxSize,
	}
}

func (c *DNSCache) Get(key string) (*DNSCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&c.hits, 1)
	return entry, true
}

func (c *DNSCache) Set(key string, records []*DNSRecord, ttl time.Duration, geoMatch bool, clientIP string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictExpired()
	}

	c.entries[key] = &DNSCacheEntry{
		Records:   records,
		ExpiresAt: time.Now().Add(ttl),
		GeoMatch:  geoMatch,
		ClientIP:  clientIP,
	}
}

func (c *DNSCache) evictExpired() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *DNSCache) GetStats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses), len(c.entries)
}

func (r *DNSResolver) Resolve(ctx context.Context, query *DNSQuery) (*DNSResponse, error) {
	startTime := time.Now()

	cacheKey := r.buildCacheKey(query)
	if entry, ok := r.cache.Get(cacheKey); ok {
		return &DNSResponse{
			Records:   entry.Records,
			TTL:       r.calculateTTL(entry.Records),
			FromCache: true,
			LatencyMs: float64(time.Since(startTime).Milliseconds()),
			Timestamp: time.Now(),
		}, nil
	}

	var geoLoc *GeoLocation
	var err error

	if query.GeoLocation != nil {
		geoLoc = query.GeoLocation
	} else if query.ClientIP != "" {
		geoLoc, err = r.nodeManager.ResolveIPRegion(query.ClientIP)
		if err != nil {
			parsedIP := net.ParseIP(query.ClientIP)
			isPublic := parsedIP != nil && !parsedIP.IsLoopback() && !parsedIP.IsUnspecified() && !parsedIP.IsLinkLocalUnicast()
			geoLoc = &GeoLocation{IP: query.ClientIP, IsPublic: isPublic}
		}
	}

	records, err := r.resolveRecords(ctx, query, geoLoc)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		record.LatencyMs = r.measureLatency(record.Value)
	}

	ttl := r.calculateTTL(records)
	r.cache.Set(cacheKey, records, time.Duration(ttl)*time.Second, geoLoc != nil, query.ClientIP)

	return &DNSResponse{
		Records:   records,
		TTL:       ttl,
		FromCache: false,
		LatencyMs: float64(time.Since(startTime).Milliseconds()),
		Timestamp: time.Now(),
	}, nil
}

func (r *DNSResolver) resolveRecords(ctx context.Context, query *DNSQuery, geoLoc *GeoLocation) ([]*DNSRecord, error) {
	var matchedRules []*GeoDNSRule

	if len(r.ruleIndex) > 0 {
		for priority, rules := range r.ruleIndex {
			for _, rule := range rules {
				if !rule.Enabled {
					continue
				}
				if r.matchRule(rule, query, geoLoc) {
					matchedRules = append(matchedRules, rule)
					if priority >= 100 {
						break
					}
				}
			}
		}
	}

	if len(matchedRules) > 0 {
		var records []*DNSRecord
		for _, rule := range matchedRules {
			for _, value := range rule.RecordValues {
				records = append(records, &DNSRecord{
					Value: value,
					TTL:   rule.TTL,
				})
			}
		}
		return records, nil
	}

	if r.zone != nil {
		var records []*DNSRecord
		for _, record := range r.zone.Records {
			if r.matchRecord(record, geoLoc) {
				records = append(records, record)
			}
		}
		if len(records) > 0 {
			return records, nil
		}
	}

	return r.getDefaultRecords(), nil
}

func (r *DNSResolver) matchRule(rule *GeoDNSRule, query *DNSQuery, geoLoc *GeoLocation) bool {
	if !strings.Contains(strings.ToLower(query.Name), strings.ToLower(rule.Pattern)) {
		return false
	}

	switch rule.MatchType {
	case "region":
		if geoLoc == nil {
			return false
		}
		for _, region := range rule.Regions {
			if region == geoLoc.Region {
				return true
			}
		}
		return false
	case "country":
		if geoLoc == nil {
			return false
		}
		for _, country := range rule.Countries {
			if country == geoLoc.Country {
				return true
			}
		}
		return false
	case "exact":
		return strings.EqualFold(query.Name, rule.Pattern)
	case "wildcard":
		return true
	default:
		return false
	}
}

func (r *DNSResolver) matchRecord(record *DNSRecord, geoLoc *GeoLocation) bool {
	if record.Region != "" && geoLoc != nil {
		if record.Region != geoLoc.Region {
			return false
		}
	}
	if record.Country != "" && geoLoc != nil {
		if record.Country != geoLoc.Country {
			return false
		}
	}
	return record.Healthy
}

func (r *DNSResolver) getDefaultRecords() []*DNSRecord {
	if r.zone == nil {
		return []*DNSRecord{
			{Type: DNSRecordA, Value: "127.0.0.1", TTL: 300, Healthy: true},
		}
	}

	var healthyRecords []*DNSRecord
	for _, record := range r.zone.Records {
		if record.Healthy {
			healthyRecords = append(healthyRecords, record)
		}
	}

	if len(healthyRecords) == 0 {
		return r.zone.Records
	}

	return healthyRecords
}

func (r *DNSResolver) buildCacheKey(query *DNSQuery) string {
	key := fmt.Sprintf("%s:%s:%s", query.Name, query.Type, query.ClientIP)
	if query.EDNSClientSubnet != nil {
		key += ":" + query.EDNSClientSubnet.String()
	}
	return key
}

func (r *DNSResolver) calculateTTL(records []*DNSRecord) int {
	if len(records) == 0 {
		return 300
	}

	minTTL := records[0].TTL
	for _, record := range records {
		if record.TTL < minTTL {
			minTTL = record.TTL
		}
	}

	return minTTL
}

func (r *DNSResolver) measureLatency(target string) float64 {
	if r.nodeManager == nil {
		return 0
	}

	ip := net.ParseIP(target)
	if ip == nil {
		return 0
	}

	pingResult := r.nodeManager.CalculateDistance(0, 0, 0, 0)
	return pingResult * 0.1
}

func (r *DNSResolver) checkHealth(target string) (bool, float64) {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", target, 3*time.Second)
	if err != nil {
		return false, 0
	}
	defer conn.Close()

	latency := time.Since(start).Seconds() * 1000
	return true, latency
}

func (r *DNSResolver) startHealthChecks() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for target, checker := range r.healthCheckers {
			go r.performHealthCheck(target, checker)
		}
	}
}

func (r *DNSResolver) performHealthCheck(target string, checker *HealthChecker) {
	healthy, latency := checker.CheckFunc(target)

	checker.LastCheck = time.Now()

	if !healthy {
		checker.Failures++
		if checker.Failures >= checker.Threshold {
			checker.Healthy = false
			r.updateRecordHealth(target, false)
		}
	} else {
		if checker.Failures > 0 {
			checker.Failures--
		}
		if checker.Failures < checker.Threshold {
			checker.Healthy = true
			r.updateRecordHealth(target, true)
		}
	}

	_ = latency
}

func (r *DNSResolver) updateRecordHealth(target string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.zone != nil {
		for _, record := range r.zone.Records {
			if record.Value == target {
				record.Healthy = healthy
				record.UpdatedAt = time.Now()
				atomic.AddInt64(&r.version, 1)
			}
		}
	}
}

func (r *DNSResolver) AddRule(rule *GeoDNSRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	r.rules[rule.ID] = rule
	r.ruleIndex[rule.Priority] = append(r.ruleIndex[rule.Priority], rule)
	atomic.AddInt64(&r.version, 1)

	return nil
}

func (r *DNSResolver) DeleteRule(ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(r.rules, ruleID)
	for i, ruleItem := range r.ruleIndex[rule.Priority] {
		if ruleItem.ID == ruleID {
			r.ruleIndex[rule.Priority] = append(r.ruleIndex[rule.Priority][:i], r.ruleIndex[rule.Priority][i+1:]...)
			break
		}
	}
	atomic.AddInt64(&r.version, 1)

	return nil
}

func (r *DNSResolver) UpdateRule(rule *GeoDNSRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.rules[rule.ID]
	if !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	delete(r.ruleIndex, existing.Priority)
	existing.Pattern = rule.Pattern
	existing.MatchType = rule.MatchType
	existing.Regions = rule.Regions
	existing.Countries = rule.Countries
	existing.RecordValues = rule.RecordValues
	existing.TTL = rule.TTL
	existing.Priority = rule.Priority
	existing.Enabled = rule.Enabled
	r.ruleIndex[rule.Priority] = append(r.ruleIndex[rule.Priority], existing)
	atomic.AddInt64(&r.version, 1)

	return nil
}

func (r *DNSResolver) GetRules(zoneID string) []*GeoDNSRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var rules []*GeoDNSRule
	for _, rule := range r.rules {
		if rule.ZoneID == zoneID {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (r *DNSResolver) AddRecord(record *DNSRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.zone == nil {
		return fmt.Errorf("zone not configured")
	}

	record.ID = fmt.Sprintf("record-%d", time.Now().UnixNano())
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()
	record.Healthy = true

	r.zone.Records = append(r.zone.Records, record)
	r.healthCheckers[record.Value] = &HealthChecker{
		Target:    record.Value,
		Interval:  10 * time.Second,
		Timeout:   3 * time.Second,
		Threshold: 3,
		Healthy:   true,
		CheckFunc: r.checkHealth,
	}
	atomic.AddInt64(&r.version, 1)

	return nil
}

func (r *DNSResolver) DeleteRecord(recordID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.zone == nil {
		return fmt.Errorf("zone not configured")
	}

	var target string
	var index = -1
	for i, record := range r.zone.Records {
		if record.ID == recordID {
			target = record.Value
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("record not found: %s", recordID)
	}

	r.zone.Records = append(r.zone.Records[:index], r.zone.Records[index+1:]...)
	delete(r.healthCheckers, target)
	atomic.AddInt64(&r.version, 1)

	return nil
}

func (r *DNSResolver) GetZone() *DNSZone {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.zone == nil {
		return nil
	}

	zoneCopy := *r.zone
	zoneCopy.Records = make([]*DNSRecord, len(r.zone.Records))
	copy(zoneCopy.Records, r.zone.Records)
	return &zoneCopy
}

func (r *DNSResolver) FlushCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache = NewDNSCache(r.cache.maxSize)
	atomic.AddInt64(&r.version, 1)
}

func (r *DNSResolver) GetCacheStats() (hits, misses int64, size int) {
	return r.cache.GetStats()
}

func (r *DNSResolver) GetVersion() int64 {
	return atomic.LoadInt64(&r.version)
}

func (r *DNSResolver) SyncToRedis(ctx context.Context) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.Marshal(r.zone)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("edge:dns:zone:%s", r.zone.ID)
	return r.redisClient.Set(ctx, key, data, 24*time.Hour).Err()
}

func (r *DNSResolver) SyncFromRedis(ctx context.Context, zoneID string) error {
	if r.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("edge:dns:zone:%s", zoneID)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return json.Unmarshal(data, &r.zone)
}

func (r *DNSResolver) ResolveWithLoadBalancing(ctx context.Context, query *DNSQuery, strategy LoadBalanceStrategy) (*DNSResponse, error) {
	response, err := r.Resolve(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(response.Records) <= 1 {
		return response, nil
	}

	selectedRecord := strategy.Select(response.Records)

	return &DNSResponse{
		Records:   []*DNSRecord{selectedRecord},
		TTL:       response.TTL,
		FromCache: response.FromCache,
		LatencyMs: response.LatencyMs,
		Timestamp: response.Timestamp,
	}, nil
}

type LoadBalanceStrategy interface {
	Select(records []*DNSRecord) *DNSRecord
}

type RoundRobinStrategy struct {
	index int32
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{}
}

func (s *RoundRobinStrategy) Select(records []*DNSRecord) *DNSRecord {
	index := atomic.AddInt32(&s.index, 1) - 1
	return records[index%int32(len(records))]
}

type WeightedRoundRobinStrategy struct {
	index int32
}

func NewWeightedRoundRobinStrategy() *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{}
}

func (s *WeightedRoundRobinStrategy) Select(records []*DNSRecord) *DNSRecord {
	var totalWeight int
	for _, record := range records {
		if record.Weight <= 0 {
			totalWeight += 100
		} else {
			totalWeight += record.Weight
		}
	}

	index := atomic.AddInt32(&s.index, 1) - 1
	target := int(index) % totalWeight

	var cumulative int
	for _, record := range records {
		weight := record.Weight
		if weight <= 0 {
			weight = 100
		}
		cumulative += weight
		if target < cumulative {
			return record
		}
	}

	return records[0]
}

type LatencyBasedStrategy struct{}

func NewLatencyBasedStrategy() *LatencyBasedStrategy {
	return &LatencyBasedStrategy{}
}

func (s *LatencyBasedStrategy) Select(records []*DNSRecord) *DNSRecord {
	var best *DNSRecord
	bestLatency := math.MaxFloat64

	for _, record := range records {
		if record.LatencyMs < bestLatency {
			bestLatency = record.LatencyMs
			best = record
		}
	}

	if best == nil {
		return records[0]
	}
	return best
}

type GeolocationBasedStrategy struct{}

func NewGeolocationBasedStrategy() *GeolocationBasedStrategy {
	return &GeolocationBasedStrategy{}
}

func (s *GeolocationBasedStrategy) Select(records []*DNSRecord) *DNSRecord {
	var best *DNSRecord

	for _, record := range records {
		if record.Region != "" {
			best = record
			break
		}
	}

	if best == nil {
		return records[0]
	}
	return best
}
