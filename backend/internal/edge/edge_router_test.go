package edge

import (
	"testing"
)

func TestGeoRouter_NewGeoRouter(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	if router == nil {
		t.Fatal("Expected router to not be nil")
	}

	if router.table == nil {
		t.Fatal("Expected routing table to not be nil")
	}

	if len(router.table.Rules) != 0 {
		t.Errorf("Expected 0 rules initially, got %d", len(router.table.Rules))
	}
}

func TestGeoRouter_AddRule(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	rule := &RouteRule{
		Name:           "Test Rule",
		Priority:       50,
		Strategy:       RoutingGeo,
		SourceRegions:  []Region{RegionAP},
		TargetRegions:  []Region{RegionAP},
		Weight:         100,
		Enabled:        true,
	}

	err := router.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if rule.ID == "" {
		t.Error("Expected rule ID to be set")
	}

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
}

func TestGeoRouter_DeleteRule(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	rule := &RouteRule{
		Name:           "Test Rule",
		Priority:       50,
		Strategy:       RoutingGeo,
		SourceRegions:  []Region{RegionAP},
		TargetRegions:  []Region{RegionAP},
		Weight:         100,
		Enabled:        true,
	}

	router.AddRule(rule)

	err := router.DeleteRule(rule.ID)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	rules := router.GetRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after deletion, got %d", len(rules))
	}
}

func TestGeoRouter_UpdateRule(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	rule := &RouteRule{
		Name:           "Test Rule",
		Priority:       50,
		Strategy:       RoutingGeo,
		SourceRegions:  []Region{RegionAP},
		TargetRegions:  []Region{RegionAP},
		Weight:         100,
		Enabled:        true,
	}

	router.AddRule(rule)

	rule.Name = "Updated Rule"
	err := router.UpdateRule(rule)
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].Name != "Updated Rule" {
		t.Errorf("Expected rule name 'Updated Rule', got '%s'", rules[0].Name)
	}
}

func TestGeoRouter_CreateDefaultRules(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	err := router.CreateDefaultRules()
	if err != nil {
		t.Fatalf("CreateDefaultRules failed: %v", err)
	}

	rules := router.GetRules()
	if len(rules) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(rules))
	}
}

func TestGeoRouter_GetMetrics(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	metrics := router.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", metrics.TotalRequests)
	}
}

func TestGeoRouter_GetVersion(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	version := router.GetVersion()

	router.AddRule(&RouteRule{
		Name:      "Test",
		Priority:  50,
		Strategy:  RoutingGeo,
		Weight:    100,
		Enabled:   true,
	})

	newVersion := router.GetVersion()
	if newVersion <= version {
		t.Error("Expected version to increase after adding rule")
	}
}

func TestGeoRouter_SubscribeUpdates(t *testing.T) {
	router := NewGeoRouter(nil, nil)

	ch := router.SubscribeUpdates()
	if ch == nil {
		t.Fatal("Expected channel to not be nil")
	}
}
