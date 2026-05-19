package service

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type DataCenter struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	Location     GeoCoord       `json:"location"`
	Priority     int               `json:"priority"`
	mu           sync.RWMutex
	Healthy      atomic.Bool       `json:"healthy"`
	LoadFactor   float64           `json:"load_factor"`
	LatencyMs    atomic.Int64      `json:"latency_ms"`
	Capacity     int               `json:"capacity"`
	CurrentLoad  atomic.Int64      `json:"current_load"`
	Metadata     map[string]string `json:"metadata"`
}

type GeoCoord struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
}

type DNSRecord struct {
	ID        string       `json:"id"`
	Domain    string       `json:"domain"`
	Type      string       `json:"type"`
	Values    []string     `json:"values"`
	TTL       int          `json:"ttl"`
	Priority  int          `json:"priority"`
	Weight    int          `json:"weight"`
	DataCenter string      `json:"data_center"`
}

type TrafficPolicy struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Strategy        string            `json:"strategy"`
	Enabled         bool              `json:"enabled"`
	HealthCheckPath string            `json:"health_check_path"`
	HealthCheckIntvl int               `json:"health_check_interval_secs"`
	FailoverEnabled bool              `json:"failover_enabled"`
	Weights         map[string]int    `json:"weights"`
}

type DistributedArchitectureV2 struct {
	mu          sync.RWMutex
	dataCenters map[string]*DataCenter
	dnsRecords  map[string]*DNSRecord
	policies    map[string]*TrafficPolicy
	trafficLogs []TrafficLog
	muLogs      sync.RWMutex
}

type TrafficLog struct {
	Timestamp   time.Time `json:"timestamp"`
	SourceIP    string    `json:"source_ip"`
	TargetDC    string    `json:"target_dc"`
	LatencyMs   int64     `json:"latency_ms"`
	Success     bool      `json:"success"`
	Reason      string    `json:"reason"`
}

type DNSResolveResult struct {
	IP         string      `json:"ip"`
	DataCenter string      `json:"data_center"`
	LatencyMs  int64       `json:"latency_ms"`
	Region     string      `json:"region"`
	FromCache  bool        `json:"from_cache"`
}

type TrafficAllocation struct {
	DataCenterID string  `json:"data_center_id"`
	Percentage   float64 `json:"percentage"`
	CurrentLoad  int64   `json:"current_load"`
	AvailableCap int     `json:"available_capacity"`
}

type FailoverEvent struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	SourceDC    string    `json:"source_dc"`
	TargetDC    string    `json:"target_dc"`
	Reason      string    `json:"reason"`
	DurationMs  int64     `json:"duration_ms"`
	Recovered   bool      `json:"recovered"`
}

func NewDistributedArchitectureV2() *DistributedArchitectureV2 {
	return &DistributedArchitectureV2{
		dataCenters: make(map[string]*DataCenter),
		dnsRecords:  make(map[string]*DNSRecord),
		policies:    make(map[string]*TrafficPolicy),
		trafficLogs: make([]TrafficLog, 0),
	}
}

func (da *DistributedArchitectureV2) RegisterDataCenter(dc *DataCenter) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	dc.Healthy.Store(true)
	da.dataCenters[dc.ID] = dc

	return nil
}

func (da *DistributedArchitectureV2) GetDataCenter(id string) (*DataCenter, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	dc, exists := da.dataCenters[id]
	return dc, exists
}

func (da *DistributedArchitectureV2) GetAllDataCenters() []*DataCenter {
	da.mu.RLock()
	defer da.mu.RUnlock()

	dcs := make([]*DataCenter, 0, len(da.dataCenters))
	for _, dc := range da.dataCenters {
		dcs = append(dcs, dc)
	}

	return dcs
}

func (da *DistributedArchitectureV2) GetHealthyDataCenters() []*DataCenter {
	da.mu.RLock()
	defer da.mu.RUnlock()

	var healthyDCs []*DataCenter
	for _, dc := range da.dataCenters {
		if dc.Healthy.Load() {
			healthyDCs = append(healthyDCs, dc)
		}
	}

	return healthyDCs
}

func (da *DistributedArchitectureV2) UpdateDataCenterHealth(id string, healthy bool) {
	da.mu.Lock()
	defer da.mu.Unlock()

	if dc, exists := da.dataCenters[id]; exists {
		wasHealthy := dc.Healthy.Load()
		dc.Healthy.Store(healthy)

		if wasHealthy && !healthy {
			da.initiateFailover(id)
		} else if !wasHealthy && healthy {
			da.initiateRecovery(id)
		}
	}
}

func (da *DistributedArchitectureV2) initiateFailover(dcID string) {
}

func (da *DistributedArchitectureV2) initiateRecovery(dcID string) {
}

func (da *DistributedArchitectureV2) UpdateDataCenterLoad(id string, load int64) {
	da.mu.Lock()
	defer da.mu.Unlock()

	if dc, exists := da.dataCenters[id]; exists {
		dc.CurrentLoad.Store(load)
		loadFactor := math.Min(1.0, float64(load)/float64(dc.Capacity))
		dc.LoadFactor = loadFactor
	}
}

func (da *DistributedArchitectureV2) UpdateDataCenterLatency(id string, latencyMs int64) {
	da.mu.Lock()
	defer da.mu.Unlock()

	if dc, exists := da.dataCenters[id]; exists {
		dc.LatencyMs.Store(latencyMs)
	}
}

func (da *DistributedArchitectureV2) CreateDNSRecord(record *DNSRecord) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	da.dnsRecords[record.Domain] = record
	return nil
}

func (da *DistributedArchitectureV2) GetDNSRecord(domain string) (*DNSRecord, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	record, exists := da.dnsRecords[domain]
	return record, exists
}

func (da *DistributedArchitectureV2) ResolveDNS(domain string, clientIP string) (*DNSResolveResult, error) {
	da.mu.RLock()
	record, exists := da.dnsRecords[domain]
	da.mu.RUnlock()

	if !exists {
		return nil, ErrNotFound
	}

	healthyDCs := da.GetHealthyDataCenters()
	if len(healthyDCs) == 0 {
		return nil, ErrNoHealthyNodes
	}

	targetDC := da.selectBestDataCenter(healthyDCs, clientIP)
	if targetDC == nil {
		return nil, ErrNoHealthyNodes
	}

	if len(record.Values) == 0 {
		return nil, ErrNoHealthyNodes
	}

	da.muLogs.Lock()
	da.trafficLogs = append(da.trafficLogs, TrafficLog{
		Timestamp: time.Now(),
		SourceIP:  clientIP,
		TargetDC:  targetDC.ID,
		LatencyMs: targetDC.LatencyMs.Load(),
		Success:   true,
	})
	da.muLogs.Unlock()

	return &DNSResolveResult{
		IP:         record.Values[0],
		DataCenter: targetDC.ID,
		LatencyMs:  targetDC.LatencyMs.Load(),
		Region:     targetDC.Region,
		FromCache:  false,
	}, nil
}

func (da *DistributedArchitectureV2) selectBestDataCenter(dcs []*DataCenter, clientIP string) *DataCenter {
	if len(dcs) == 0 {
		return nil
	}

	var bestDC *DataCenter
	bestScore := float64(math.MaxFloat64)

	for _, dc := range dcs {
		dc.mu.RLock()
		loadScore := dc.LoadFactor
		dc.mu.RUnlock()
		latencyScore := float64(dc.LatencyMs.Load()) / 1000.0
		priorityScore := float64(dc.Priority) / 100.0

		score := loadScore*0.4 + latencyScore*0.4 + (1.0 - priorityScore)*0.2

		if score < bestScore {
			bestScore = score
			bestDC = dc
		}
	}

	return bestDC
}

func (da *DistributedArchitectureV2) CreateTrafficPolicy(policy *TrafficPolicy) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	da.policies[policy.ID] = policy
	return nil
}

func (da *DistributedArchitectureV2) GetTrafficPolicy(id string) (*TrafficPolicy, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	policy, exists := da.policies[id]
	return policy, exists
}

func (da *DistributedArchitectureV2) ApplyTrafficPolicy(policyID string) (*TrafficAllocation, error) {
	policy, exists := da.GetTrafficPolicy(policyID)
	if !exists {
		return nil, ErrNotFound
	}

	healthyDCs := da.GetHealthyDataCenters()
	if len(healthyDCs) == 0 {
		return nil, ErrNoHealthyNodes
	}

	var totalWeight int
	for _, dc := range healthyDCs {
		if weight, ok := policy.Weights[dc.ID]; ok {
			totalWeight += weight
		} else {
			totalWeight += 1
		}
	}

	var bestDC *DataCenter
	bestScore := float64(math.MaxFloat64)

	for _, dc := range healthyDCs {
		weight := 1
		if w, ok := policy.Weights[dc.ID]; ok {
			weight = w
		}

		dc.mu.RLock()
		loadScore := dc.LoadFactor
		dc.mu.RUnlock()
		weightScore := float64(weight) / float64(totalWeight)
		score := loadScore - weightScore

		if score < bestScore {
			bestScore = score
			bestDC = dc
		}
	}

	if bestDC == nil {
		bestDC = healthyDCs[0]
	}

	return &TrafficAllocation{
		DataCenterID: bestDC.ID,
		Percentage:   float64(bestDC.CurrentLoad.Load()) / float64(bestDC.Capacity) * 100,
		CurrentLoad:  bestDC.CurrentLoad.Load(),
		AvailableCap: bestDC.Capacity - int(bestDC.CurrentLoad.Load()),
	}, nil
}

func (da *DistributedArchitectureV2) GetTrafficLogs(limit int) []TrafficLog {
	da.muLogs.RLock()
	defer da.muLogs.RUnlock()

	if limit <= 0 || limit > len(da.trafficLogs) {
		limit = len(da.trafficLogs)
	}

	logs := make([]TrafficLog, limit)
	copy(logs, da.trafficLogs[len(da.trafficLogs)-limit:])

	return logs
}

func (da *DistributedArchitectureV2) CalculateDistance(loc1, loc2 GeoCoord) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLat := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLon := (loc2.Longitude - loc1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func (da *DistributedArchitectureV2) FindNearestDataCenter(clientLocation GeoCoord) *DataCenter {
	healthyDCs := da.GetHealthyDataCenters()
	if len(healthyDCs) == 0 {
		return nil
	}

	var nearestDC *DataCenter
	minDistance := float64(math.MaxFloat64)

	for _, dc := range healthyDCs {
		dist := da.CalculateDistance(clientLocation, dc.Location)
		if dist < minDistance {
			minDistance = dist
			nearestDC = dc
		}
	}

	return nearestDC
}

func (da *DistributedArchitectureV2) GetArchitectureStats() map[string]interface{} {
	da.mu.RLock()
	defer da.mu.RUnlock()

	totalDCs := len(da.dataCenters)
	healthyDCs := 0
	var totalLoad, totalCapacity int64

	for _, dc := range da.dataCenters {
		if dc.Healthy.Load() {
			healthyDCs++
		}
		totalLoad += dc.CurrentLoad.Load()
		totalCapacity += int64(dc.Capacity)
	}

	da.muLogs.RLock()
	totalTraffic := len(da.trafficLogs)
	var totalLatency int64
	for _, log := range da.trafficLogs {
		totalLatency += log.LatencyMs
	}
	da.muLogs.RUnlock()

	avgLatency := int64(0)
	if totalTraffic > 0 {
		avgLatency = totalLatency / int64(totalTraffic)
	}

	return map[string]interface{}{
		"total_data_centers":      totalDCs,
		"healthy_data_centers":   healthyDCs,
		"total_load":              totalLoad,
		"total_capacity":          totalCapacity,
		"capacity_utilization":    float64(totalLoad) / float64(totalCapacity) * 100,
		"total_traffic_events":    totalTraffic,
		"average_latency_ms":      avgLatency,
		"total_dns_records":       len(da.dnsRecords),
		"total_traffic_policies":  len(da.policies),
	}
}

func (da *DistributedArchitectureV2) HealthCheck(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})

	healthyDCs := da.GetHealthyDataCenters()
	results["healthy_count"] = len(healthyDCs)
	results["unhealthy_count"] = len(da.dataCenters) - len(healthyDCs)

	var totalLatency int64
	for _, dc := range healthyDCs {
		totalLatency += dc.LatencyMs.Load()
	}

	if len(healthyDCs) > 0 {
		results["average_latency_ms"] = totalLatency / int64(len(healthyDCs))
	} else {
		results["average_latency_ms"] = int64(0)
	}

	results["status"] = "healthy"
	if len(healthyDCs) == 0 {
		results["status"] = "critical"
	} else if len(healthyDCs) < len(da.dataCenters) {
		results["status"] = "degraded"
	}

	return results
}
