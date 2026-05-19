package edge

import (
	"context"
	"testing"
	"time"
)

func TestDNSResolver_NewDNSResolver(t *testing.T) {
	zone := &DNSZone{
		ID:     "test-zone",
		Name:   "Test Zone",
		Domain: "example.com",
		TTL:    300,
		Records: []*DNSRecord{
			{
				ID:     "record-1",
				Name:   "api",
				Type:   DNSRecordA,
				Value:  "203.0.113.10",
				TTL:    300,
				Weight: 100,
				Healthy: true,
			},
		},
	}

	resolver := NewDNSResolver(zone, nil, nil)

	if resolver == nil {
		t.Fatal("Expected resolver to not be nil")
	}

	if len(resolver.healthCheckers) != 1 {
		t.Errorf("Expected 1 health checker, got %d", len(resolver.healthCheckers))
	}
}

func TestDNSResolver_AddRule(t *testing.T) {
	resolver := NewDNSResolver(nil, nil, nil)

	rule := &GeoDNSRule{
		ZoneID:       "test-zone",
		Pattern:      "api.example.com",
		MatchType:    "region",
		Regions:      []Region{RegionAP},
		RecordValues: []string{"203.0.113.10"},
		Priority:     50,
		Enabled:      true,
	}

	err := resolver.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if rule.ID == "" {
		t.Error("Expected rule ID to be set")
	}

	rules := resolver.GetRules("test-zone")
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
}

func TestDNSResolver_DeleteRule(t *testing.T) {
	resolver := NewDNSResolver(nil, nil, nil)

	rule := &GeoDNSRule{
		ZoneID:       "test-zone",
		Pattern:      "api.example.com",
		MatchType:    "region",
		Regions:      []Region{RegionAP},
		RecordValues: []string{"203.0.113.10"},
		Priority:     50,
		Enabled:      true,
	}

	resolver.AddRule(rule)

	err := resolver.DeleteRule(rule.ID)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	rules := resolver.GetRules("test-zone")
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after deletion, got %d", len(rules))
	}
}

func TestDNSResolver_AddRecord(t *testing.T) {
	zone := &DNSZone{
		ID:     "test-zone",
		Name:   "Test Zone",
		Domain: "example.com",
		TTL:    300,
		Records: []*DNSRecord{},
	}

	resolver := NewDNSResolver(zone, nil, nil)

	record := &DNSRecord{
		Name:   "api",
		Type:   DNSRecordA,
		Value:  "203.0.113.10",
		TTL:    300,
		Weight: 100,
		Region: RegionAP,
	}

	err := resolver.AddRecord(record)
	if err != nil {
		t.Fatalf("AddRecord failed: %v", err)
	}

	if record.ID == "" {
		t.Error("Expected record ID to be set")
	}
}

func TestDNSResolver_Resolve(t *testing.T) {
	zone := &DNSZone{
		ID:     "test-zone",
		Name:   "Test Zone",
		Domain: "example.com",
		TTL:    300,
		Records: []*DNSRecord{
			{
				ID:      "record-1",
				Name:    "api",
				Type:    DNSRecordA,
				Value:   "203.0.113.10",
				TTL:     300,
				Weight:  100,
				Healthy: true,
			},
		},
	}

	resolver := NewDNSResolver(zone, nil, nil)

	query := &DNSQuery{
		Name:     "api.example.com",
		Type:     "A",
		ClientIP: "8.8.8.8",
	}

	response, err := resolver.Resolve(context.Background(), query)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(response.Records) == 0 {
		t.Error("Expected at least one record in response")
	}
}

func TestDNSResolver_FlushCache(t *testing.T) {
	resolver := NewDNSResolver(nil, nil, nil)

	resolver.FlushCache()

	hits, misses, size := resolver.GetCacheStats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after flush, got %d", size)
	}

	_ = hits
	_ = misses
}

func TestDNSCache_GetSet(t *testing.T) {
	cache := NewDNSCache(100)

	records := []*DNSRecord{
		{
			ID:    "record-1",
			Name:  "test",
			Type:  DNSRecordA,
			Value: "192.168.1.1",
			TTL:   300,
		},
	}

	cache.Set("test-key", records, 5*time.Minute, true, "127.0.0.1")

	entry, exists := cache.Get("test-key")
	if !exists {
		t.Fatal("Expected entry to exist")
	}

	if len(entry.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(entry.Records))
	}

	if !entry.GeoMatch {
		t.Error("Expected GeoMatch to be true")
	}
}

func TestDNSCache_GetMiss(t *testing.T) {
	cache := NewDNSCache(100)

	_, exists := cache.Get("nonexistent-key")
	if exists {
		t.Error("Expected entry to not exist")
	}
}

func TestDNSCache_GetStats(t *testing.T) {
	cache := NewDNSCache(100)

	records := []*DNSRecord{
		{ID: "record-1", Name: "test", Type: DNSRecordA, Value: "192.168.1.1"},
	}

	cache.Set("test-key", records, 5*time.Minute, false, "")
	cache.Get("test-key")
	cache.Get("nonexistent")

	hits, misses, size := cache.GetStats()
	if hits != 1 {
		t.Errorf("Expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}
}

func TestRoundRobinStrategy_Select(t *testing.T) {
	strategy := NewRoundRobinStrategy()

	records := []*DNSRecord{
		{ID: "1", Value: "10.0.0.1"},
		{ID: "2", Value: "10.0.0.2"},
		{ID: "3", Value: "10.0.0.3"},
	}

	selected := make(map[string]int)
	for i := 0; i < 6; i++ {
		record := strategy.Select(records)
		selected[record.ID]++
	}

	for id, count := range selected {
		if count != 2 {
			t.Errorf("Expected record %s selected 2 times, got %d", id, count)
		}
	}
}

func TestWeightedRoundRobinStrategy_Select(t *testing.T) {
	strategy := NewWeightedRoundRobinStrategy()

	records := []*DNSRecord{
		{ID: "1", Value: "10.0.0.1", Weight: 3},
		{ID: "2", Value: "10.0.0.2", Weight: 1},
	}

	selected := make(map[string]int)
	for i := 0; i < 8; i++ {
		record := strategy.Select(records)
		selected[record.ID]++
	}

	if selected["1"] < selected["2"] {
		t.Error("Expected record with higher weight to be selected more frequently")
	}
}

func TestLatencyBasedStrategy_Select(t *testing.T) {
	strategy := NewLatencyBasedStrategy()

	records := []*DNSRecord{
		{ID: "1", Value: "10.0.0.1", LatencyMs: 100},
		{ID: "2", Value: "10.0.0.2", LatencyMs: 50},
		{ID: "3", Value: "10.0.0.3", LatencyMs: 75},
	}

	selected := strategy.Select(records)

	if selected.ID != "2" {
		t.Errorf("Expected record with lowest latency (ID: 2), got ID: %s", selected.ID)
	}
}

func TestGeolocationBasedStrategy_Select(t *testing.T) {
	strategy := NewGeolocationBasedStrategy()

	records := []*DNSRecord{
		{ID: "1", Value: "10.0.0.1", Region: RegionAP},
		{ID: "2", Value: "10.0.0.2", Region: RegionEU},
		{ID: "3", Value: "10.0.0.3", Region: ""},
	}

	selected := strategy.Select(records)

	if selected.Region != RegionAP {
		t.Errorf("Expected record with region, got region: %s", selected.Region)
	}
}
