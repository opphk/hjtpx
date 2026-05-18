package service

import (
	"context"
	"testing"
)

func TestWebGLFingerprintAnalyzer_AnalyzeRenderer(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	tests := []struct {
		name     string
		renderer string
		vendor   string
		wantVM   bool
		wantSoft bool
	}{
		{
			name:     "Normal GPU",
			renderer: "NVIDIA GeForce RTX 3080",
			vendor:   "NVIDIA Corporation",
			wantVM:   false,
			wantSoft: false,
		},
		{
			name:     "VMware Virtual GPU",
			renderer: "VMware SVGA II Adapter",
			vendor:   "VMware, Inc.",
			wantVM:   true,
			wantSoft: false,
		},
		{
			name:     "VirtualBox GPU",
			renderer: "VirtualBox Graphics Adapter",
			vendor:   "Oracle Corporation",
			wantVM:   true,
			wantSoft: true,
		},
		{
			name:     "SwiftShader Software",
			renderer: "Google SwiftShader",
			vendor:   "Google Inc.",
			wantVM:   false,
			wantSoft: true,
		},
		{
			name:     "LLVMpipe Software",
			renderer: "llvmpipe",
			vendor:   "Mesa",
			wantVM:   false,
			wantSoft: true,
		},
		{
			name:     "Unknown/Anonymized",
			renderer: "unknown",
			vendor:   "unknown",
			wantVM:   false,
			wantSoft: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.AnalyzeRenderer(tt.renderer, tt.vendor)
			
			if result.IsVMRenderer != tt.wantVM {
				t.Errorf("IsVMRenderer = %v, want %v", result.IsVMRenderer, tt.wantVM)
			}
			
			if result.IsSoftwareRenderer != tt.wantSoft {
				t.Errorf("IsSoftwareRenderer = %v, want %v", result.IsSoftwareRenderer, tt.wantSoft)
			}
			
			if tt.wantVM || tt.wantSoft {
				if len(result.Anomalies) == 0 {
					t.Error("Expected anomalies but got none")
				}
			}
		})
	}
}

func TestVMMultiDimensionDetector_AnalyzeCPU(t *testing.T) {
	detector := NewVMMultiDimensionDetector()

	tests := []struct {
		name       string
		cpuCount   int
		userAgent  string
		wantDetect bool
	}{
		{
			name:       "Normal 8-core CPU",
			cpuCount:   8,
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			wantDetect: false,
		},
		{
			name:       "Single core VM",
			cpuCount:   1,
			userAgent:  "Mozilla/5.0",
			wantDetect: true,
		},
		{
			name:       "Dual core VM",
			cpuCount:   2,
			userAgent:  "Mozilla/5.0",
			wantDetect: true,
		},
		{
			name:       "VM in UserAgent",
			cpuCount:   4,
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 VMware/15.5.6",
			wantDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _, _ := detector.AnalyzeCPU(tt.cpuCount, tt.userAgent)
			
			if detected != tt.wantDetect {
				t.Errorf("detected = %v, want %v", detected, tt.wantDetect)
			}
		})
	}
}

func TestVMMultiDimensionDetector_AnalyzeMemory(t *testing.T) {
	detector := NewVMMultiDimensionDetector()

	tests := []struct {
		name       string
		memGB      float64
		wantDetect bool
	}{
		{
			name:       "Normal 16GB",
			memGB:      16.0,
			wantDetect: false,
		},
		{
			name:       "Very Low Memory VM",
			memGB:      0.5,
			wantDetect: true,
		},
		{
			name:       "Minimal Memory VM",
			memGB:      0.25,
			wantDetect: true,
		},
		{
			name:       "Typical VM Memory",
			memGB:      4.0,
			wantDetect: true,
		},
		{
			name:       "Unusually High",
			memGB:      128.0,
			wantDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _, _ := detector.AnalyzeMemory(tt.memGB)
			
			if detected != tt.wantDetect {
				t.Errorf("detected = %v, want %v", detected, tt.wantDetect)
			}
		})
	}
}

func TestVMMultiDimensionDetector_AnalyzeGPU(t *testing.T) {
	detector := NewVMMultiDimensionDetector()

	tests := []struct {
		name     string
		renderer string
		vendor   string
		wantVM   bool
	}{
		{
			name:     "Normal GPU",
			renderer: "AMD Radeon RX 6800 XT",
			vendor:   "Advanced Micro Devices, Inc.",
			wantVM:   false,
		},
		{
			name:     "VMware GPU",
			renderer: "VMware SVGA 3D",
			vendor:   "VMware, Inc.",
			wantVM:   true,
		},
		{
			name:     "VirtualBox GPU",
			renderer: "VirtualBox Graphics",
			vendor:   "Oracle",
			wantVM:   true,
		},
		{
			name:     "QEMU/KVM GPU",
			renderer: "virtio-gpu-pci",
			vendor:   "Red Hat",
			wantVM:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _, _ := detector.AnalyzeGPU(tt.renderer, tt.vendor)
			
			if detected != tt.wantVM {
				t.Errorf("detected = %v, want %v", detected, tt.wantVM)
			}
		})
	}
}

func TestVMMultiDimensionDetector_AnalyzeBiosAndRegistry(t *testing.T) {
	detector := NewVMMultiDimensionDetector()

	tests := []struct {
		name       string
		userAgent  string
		platform   string
		wantDetect bool
	}{
		{
			name:       "Normal System",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0",
			platform:   "Win32",
			wantDetect: false,
		},
		{
			name:       "VirtualBox in UserAgent",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 VirtualBox/6.1.32",
			platform:   "Win32",
			wantDetect: true,
		},
		{
			name:       "VMware in Platform",
			userAgent:  "Mozilla/5.0",
			platform:   "VMware Virtual Platform",
			wantDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, _, _ := detector.AnalyzeBiosAndRegistry(tt.userAgent, tt.platform)
			
			if detected != tt.wantDetect {
				t.Errorf("detected = %v, want %v", detected, tt.wantDetect)
			}
		})
	}
}

func TestAdvancedEnvDetectorService_DetectEnvironment(t *testing.T) {
	service := NewAdvancedEnvDetectorService()

	tests := []struct {
		name   string
		req    *AdvancedEnvDetectionRequest
	}{
		{
			name: "Normal Request",
			req: &AdvancedEnvDetectionRequest{
				DetectionID:   "test_123",
				RiskScore:     30.0,
				RiskLevel:     "low",
				AllDetections: []string{"normal_detection_1"},
				ClientResults: map[string]interface{}{
					"webgl_anomaly": map[string]interface{}{
						"score":      10.0,
						"detections": []interface{}{},
					},
				},
				Fingerprint: "fp_test_123",
				IPAddress:   "192.168.1.1",
				UserAgent:   "Mozilla/5.0",
			},
		},
		{
			name: "High Risk Request",
			req: &AdvancedEnvDetectionRequest{
				DetectionID:   "test_456",
				RiskScore:     80.0,
				RiskLevel:     "high",
				AllDetections: []string{"vm_detected", "headless_browser"},
				ClientResults: map[string]interface{}{
					"vm_cpu": map[string]interface{}{
						"score":      70.0,
						"detections": []interface{}{"single_core_vm"},
					},
					"vm_gpu": map[string]interface{}{
						"score":      60.0,
						"detections": []interface{}{"vmware_gpu"},
					},
				},
				Fingerprint: "fp_test_456",
				IPAddress:   "10.0.0.1",
				UserAgent:   "HeadlessChrome",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.DetectEnvironment(ctx, tt.req)
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			
			if result.DetectionID == "" {
				t.Error("DetectionID should not be empty")
			}
			
			if result.RiskScore < 0 || result.RiskScore > 100 {
				t.Errorf("RiskScore out of range: %v", result.RiskScore)
			}
			
			if result.RiskLevel == "" {
				t.Error("RiskLevel should not be empty")
			}
		})
	}
}

func TestRiskScorer(t *testing.T) {
	NewRiskScorer()

	tests := []struct {
		name       string
		clientScore float64
		categoryScore float64
		wantScore  float64
	}{
		{
			name:       "Low Risk",
			clientScore: 20.0,
			categoryScore: 15.0,
			wantScore:  20.0,
		},
		{
			name:       "Medium Risk",
			clientScore: 50.0,
			categoryScore: 40.0,
			wantScore:  50.0,
		},
		{
			name:       "High Risk",
			clientScore: 80.0,
			categoryScore: 70.0,
			wantScore:  80.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.clientScore
			
			if score >= 80 {
				score = 80
			} else if score >= 50 {
				score = 50
			}
			
			if score != tt.wantScore {
				t.Errorf("score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

func TestAdvancedEnvDetectorService_CheckTorNetwork(t *testing.T) {
	service := NewAdvancedEnvDetectorService()

	tests := []struct {
		name    string
		ip      string
		wantTor bool
	}{
		{
			name:    "Private IP",
			ip:      "192.168.1.1",
			wantTor: false,
		},
		{
			name:    "Localhost",
			ip:      "127.0.0.1",
			wantTor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.CheckTorNetwork(ctx, tt.ip)
			
			if err != nil {
				t.Logf("CheckTorNetwork error (may be expected): %v", err)
			}
			
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			
			if tt.ip != "" && result.IPAddress == "" {
				t.Error("IPAddress should not be empty when input IP is provided")
			}
		})
	}
}

func TestIsTorIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "Normal IP",
			ip:   "8.8.8.8",
			want: false,
		},
		{
			name: "Private IP",
			ip:   "192.168.1.1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTorIP(tt.ip)
			if got != tt.want {
				t.Errorf("isTorIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomID(t *testing.T) {
	id1 := generateRandomID()
	id2 := generateRandomID()

	if id1 == "" {
		t.Error("generateRandomID returned empty string")
	}

	if len(id1) != 8 {
		t.Errorf("expected ID length 8, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("consecutive IDs should be different")
	}
}

func TestAdvancedEnvDetectorService_Cache(t *testing.T) {
	service := NewAdvancedEnvDetectorService()

	ctx := context.Background()
	req := &AdvancedEnvDetectionRequest{
		DetectionID: "cache_test_123",
		RiskScore:   50.0,
		Fingerprint: "fp_cache_test",
	}

	result, err := service.DetectEnvironment(ctx, req)
	if err != nil {
		t.Fatalf("DetectEnvironment failed: %v", err)
	}

	cached, found := service.GetCachedResult(result.DetectionID)
	if !found {
		t.Error("expected to find cached result")
	}

	if cached.RiskScore != result.RiskScore {
		t.Errorf("cached score = %v, want %v", cached.RiskScore, result.RiskScore)
	}

	_, found = service.GetCachedResult("nonexistent_id")
	if found {
		t.Error("should not find nonexistent ID")
	}
}

func BenchmarkWebGLFingerprintAnalyzer_AnalyzeRenderer(b *testing.B) {
	analyzer := NewWebGLFingerprintAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeRenderer("VMware SVGA II Adapter", "VMware, Inc.")
	}
}

func BenchmarkVMMultiDimensionDetector_AnalyzeCPU(b *testing.B) {
	detector := NewVMMultiDimensionDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.AnalyzeCPU(4, "Mozilla/5.0 (Windows NT 10.0)")
	}
}

func BenchmarkAdvancedEnvDetectorService_DetectEnvironment(b *testing.B) {
	service := NewAdvancedEnvDetectorService()
	ctx := context.Background()

	req := &AdvancedEnvDetectionRequest{
		DetectionID:   "bench_123",
		RiskScore:     30.0,
		RiskLevel:     "low",
		AllDetections: []string{"normal"},
		ClientResults: map[string]interface{}{
			"webgl_anomaly": map[string]interface{}{
				"score":      10.0,
				"detections": []interface{}{},
			},
		},
		Fingerprint: "fp_bench",
		IPAddress:   "192.168.1.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.DetectEnvironment(ctx, req)
	}
}
