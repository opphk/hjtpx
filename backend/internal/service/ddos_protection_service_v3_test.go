package service

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestDDoSProtectionV3Service_BasicProtection(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableSmartTrafficAnalysis:  true,
		EnableMLAttackDetection:     true,
		EnableAutoTrafficCleaning:   true,
		EnableGlobalNodeSupport:    false,
	}

	service := NewDDoSProtectionV3Service(config)

	t.Run("Normal Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		result := service.CheckRequestV3(req)

		if !result.Allowed {
			t.Errorf("Normal request should be allowed, reason: %s", result.Reason)
		}
	})

	t.Run("Whitelisted IP", func(t *testing.T) {
		service.AddToWhitelist("192.168.1.100")

		req, _ := http.NewRequest("GET", "/test", nil)

		result := service.CheckRequestV3(req)

		if !result.Allowed {
			t.Errorf("Whitelisted IP should be allowed")
		}
	})
}

func TestDDoSProtectionV3Service_Blacklist(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	t.Run("Add to Blacklist", func(t *testing.T) {
		ip := "192.168.1.50"
		service.AddToBlacklist(ip, "test_block", 1*time.Hour)

		req, _ := http.NewRequest("GET", "/test", nil)

		result := service.CheckRequestV3(req)

		if result.Allowed {
			t.Errorf("Blacklisted IP should not be allowed")
		}

		if result.Reason != "ip_blacklisted" {
			t.Errorf("Expected reason 'ip_blacklisted', got '%s'", result.Reason)
		}
	})

	t.Run("Remove from Blacklist", func(t *testing.T) {
		ip := "192.168.1.51"
		service.AddToBlacklist(ip, "test_block", 1*time.Hour)
		service.RemoveFromBlacklist(ip)

		req, _ := http.NewRequest("GET", "/test", nil)

		result := service.CheckRequestV3(req)

		if !result.Allowed {
			t.Errorf("IP should be allowed after removal from blacklist")
		}
	})
}

func TestDDoSProtectionV3Service_MLAttackDetection(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	if service.mlDetector == nil {
		t.Fatal("ML detector should be initialized")
	}

	stats := &DDoSIPStatsV3{
		IP:                  "192.168.1.100",
		RequestCount:        10000,
		RequestCountMinute:  8000,
		ErrorRate:           0.05,
		UniquePaths:         5,
		AvgRequestInterval:  10 * time.Millisecond,
		ConnectionCount:     500,
	}

	isAttack, prediction, attackType := service.mlDetector.DetectAttack(stats)

	t.Logf("Is Attack: %v, Prediction: %f, Attack Type: %s", isAttack, prediction, attackType)

	if attackType != "volume_based" {
		t.Errorf("Expected attack type 'volume_based', got '%s'", attackType)
	}
}

func TestDDoSProtectionV3Service_MLModelTraining(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	samples := []TrainingSample{
		{
			Features:  []float64{0.9, 0.1, 0.2, 0.1, 0.8},
			Label:     1.0,
			Timestamp: time.Now(),
		},
		{
			Features:  []float64{0.1, 0.9, 0.8, 0.9, 0.2},
			Label:     0.0,
			Timestamp: time.Now(),
		},
	}

	err := service.TrainMLModel(context.Background(), samples)
	if err != nil {
		t.Fatalf("Failed to train ML model: %v", err)
	}
}

func TestDDoSProtectionV3Service_GlobalStats(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		service.CheckRequestV3(req)
	}

	stats := service.GetGlobalStats()

	if stats.TotalRequests < 10 {
		t.Errorf("Expected at least 10 total requests, got %d", stats.TotalRequests)
	}
}

func TestDDoSProtectionV3Service_AnomalyDetection(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableSmartTrafficAnalysis: true,
	}

	service := NewDDoSProtectionV3Service(config)

	stats := &DDoSIPStatsV3{
		IP:         "192.168.1.200",
		ThreatScore: 0.0,
	}

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		service.recordTrafficV3("192.168.1.200", req, time.Now())
	}

	result := service.CheckRequestV3(nil)
	if result != nil {
		t.Logf("Result: Allowed=%v, ThreatScore=%f", result.Allowed, result.ThreatScore)
	}

	_ = stats
}

func TestDDoSProtectionV3Service_TopThreats(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	ips := []string{
		"192.168.1.10",
		"192.168.1.20",
		"192.168.1.30",
		"192.168.1.40",
		"192.168.1.50",
	}

	for i, ip := range ips {
		req, _ := http.NewRequest("GET", "/test", nil)
		service.CheckRequestV3(req)

		service.mu.Lock()
		if stats, exists := service.ipStats[ip]; exists {
			stats.ThreatScore = float64(i+1) * 0.2
		}
		service.mu.Unlock()
	}

	topThreats := service.GetTopThreats(3)

	if len(topThreats) != 3 {
		t.Errorf("Expected 3 top threats, got %d", len(topThreats))
	}

	for i := 0; i < len(topThreats)-1; i++ {
		if topThreats[i].ThreatScore < topThreats[i+1].ThreatScore {
			t.Error("Top threats should be sorted by threat score descending")
		}
	}
}

func TestDDoSProtectionV3Service_TrafficRecording(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	ip := "192.168.1.100"
	now := time.Now()

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test/path"+string(rune('0'+i)), nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		service.recordTrafficV3(ip, req, now.Add(time.Duration(i)*time.Second))
	}

	service.mu.RLock()
	traffic, exists := service.trafficData[ip]
	service.mu.RUnlock()

	if !exists {
		t.Fatal("Traffic data should exist")
	}

	if len(traffic.Paths) != 5 {
		t.Errorf("Expected 5 paths, got %d", len(traffic.Paths))
	}
}

func TestDDoSProtectionV3Service_MLFeatureExtraction(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	stats := &DDoSIPStatsV3{
		RequestCountMinute:  500,
		ErrorRate:           0.3,
		UniquePaths:         50,
		AvgRequestInterval:  500 * time.Millisecond,
		ConnectionCount:     30,
	}

	features := service.mlDetector.extractFeatures(stats)

	if len(features) != 5 {
		t.Errorf("Expected 5 features, got %d", len(features))
	}

	for i, feature := range features {
		if feature < 0 || feature > 1 {
			t.Errorf("Feature %d should be between 0 and 1, got %f", i, feature)
		}
	}
}

func TestDDoSProtectionV3Service_AttackClassification(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	testCases := []struct {
		name           string
		features       []float64
		expectedType   string
	}{
		{
			name:         "Volume Based",
			features:     []float64{0.9, 0.1, 0.5, 0.5, 0.5},
			expectedType: "volume_based",
		},
		{
			name:         "Application Layer",
			features:     []float64{0.5, 0.8, 0.5, 0.5, 0.5},
			expectedType: "application_layer",
		},
		{
			name:         "Scanning",
			features:     []float64{0.5, 0.1, 0.95, 0.5, 0.5},
			expectedType: "scanning",
		},
		{
			name:         "Connection Exhaustion",
			features:     []float64{0.5, 0.1, 0.5, 0.5, 0.9},
			expectedType: "connection_exhaustion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attackType := service.mlDetector.classifyAttack(tc.features)
			if attackType != tc.expectedType {
				t.Errorf("Expected '%s', got '%s'", tc.expectedType, attackType)
			}
		})
	}
}

func TestDDoSProtectionV3Service_NodeHealth(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableGlobalNodeSupport: true,
		GlobalNodes: []GlobalNode{
			{
				ID:        "node1",
				Region:    "us-west",
				IPAddress: "10.0.0.1",
				Weight:    1.0,
				IsActive:  true,
			},
			{
				ID:        "node2",
				Region:    "us-east",
				IPAddress: "10.0.0.2",
				Weight:    0.8,
				IsActive:  true,
			},
		},
	}

	service := NewDDoSProtectionV3Service(config)

	health := service.GetNodeHealth()

	if len(health) == 0 {
		t.Log("No health data available yet")
	}
}

func TestDDoSProtectionV3Service_IPExtraction(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	testCases := []struct {
		name      string
		headers   map[string]string
		remoteAddr string
		expected  string
	}{
		{
			name:      "X-Forwarded-For Multiple",
			headers:   map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "127.0.0.1:8080",
			expected:  "203.0.113.195",
		},
		{
			name:      "X-Real-IP",
			headers:   map[string]string{"X-Real-IP": "198.51.100.178"},
			remoteAddr: "127.0.0.1:8080",
			expected:  "198.51.100.178",
		},
		{
			name:      "No Headers",
			headers:   map[string]string{},
			remoteAddr: "192.0.2.1:8080",
			expected:  "192.0.2.1:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remoteAddr

			ip := service.getClientIP(req)
			if ip != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, ip)
			}
		})
	}
}

func TestDDoSProtectionV3Service_TrafficDataCreation(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	ip := "192.168.1.100"
	stats := service.getOrCreateStatsV3(ip)

	if stats.IP != ip {
		t.Errorf("Expected IP '%s', got '%s'", ip, stats.IP)
	}

	if stats.FirstSeen.IsZero() {
		t.Error("FirstSeen should not be zero")
	}

	stats2 := service.getOrCreateStatsV3(ip)

	if stats != stats2 {
		t.Error("Should return same stats for same IP")
	}
}

func TestDDoSProtectionV3Service_MLModelExport(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	exportedData, err := service.ExportMLModel(context.Background())
	if err != nil {
		t.Fatalf("Failed to export ML model: %v", err)
	}

	if len(exportedData) == 0 {
		t.Error("Exported data should not be empty")
	}
}

func TestDDoSProtectionV3Service_GlobalThreatTracking(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableMLAttackDetection: true,
	}

	service := NewDDoSProtectionV3Service(config)

	ip := "192.168.1.100"
	req, _ := http.NewRequest("GET", "/test", nil)

	for i := 0; i < 10; i++ {
		service.CheckRequestV3(req)
	}

	globalStats := service.GetGlobalStats()
	t.Logf("Global stats: TotalRequests=%d, ActiveAttacks=%d", 
		globalStats.TotalRequests, globalStats.ActiveAttacks)
}

func TestDDoSProtectionV3Service_AutoTrafficCleaner(t *testing.T) {
	config := DDoSProtectionV3Config{
		EnableAutoTrafficCleaning: true,
	}

	service := NewDDoSProtectionV3Service(config)

	if service.trafficCleaner == nil {
		t.Fatal("Traffic cleaner should be initialized")
	}

	task := &CleaningTask{
		IP:      "192.168.1.100",
		Rule:    "test_rule",
		Action:  ActionBlockIP,
		Duration: 1 * time.Hour,
	}

	service.trafficCleaner.AddTask(task)
}

func TestDDoSProtectionV3Service_CleaningRules(t *testing.T) {
	cleaner := NewAutoTrafficCleaner()

	if len(cleaner.cleaningRules) == 0 {
		t.Error("Should have cleaning rules")
	}

	for _, rule := range cleaner.cleaningRules {
		t.Logf("Rule: %s, Priority: %d", rule.Name, rule.Priority)
	}
}

func TestDDoSProtectionV3Service_ConcurrentRequests(t *testing.T) {
	config := DDoSProtectionV3Config{}
	service := NewDDoSProtectionV3Service(config)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				req, _ := http.NewRequest("GET", "/test", nil)
				service.CheckRequestV3(req)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := service.GetGlobalStats()
	if stats.TotalRequests < 100 {
		t.Errorf("Expected at least 100 total requests, got %d", stats.TotalRequests)
	}
}
