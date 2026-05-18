package service

import (
	"testing"
)

func TestSimpleEnvDetectorCPUIDFeatures(t *testing.T) {
	detector := &EnvDetector{}

	t.Run("detect_vmware_cpuid", func(t *testing.T) {
		info := &EnvInfo{
			CPUIDInfo:           "VMware Virtual CPU",
			HardwareConcurrency: 2,
		}

		detected, score, evidence := detector.DetectCPUIDFeatures(info, nil)

		if !detected {
			t.Error("Expected VMware CPUID to be detected")
		}
		if score < 40 {
			t.Errorf("Expected score >= 40, got %.2f", score)
		}
		if len(evidence) == 0 {
			t.Error("Expected evidence to be present")
		}
	})

	t.Run("normal_cpu_no_detection", func(t *testing.T) {
		info := &EnvInfo{
			CPUIDInfo:           "Intel Core i7",
			HardwareConcurrency: 8,
		}

		detected, _, _ := detector.DetectCPUIDFeatures(info, nil)

		if detected {
			t.Error("Expected normal CPU not to be detected as VM")
		}
	})
}

func TestSimpleEnvDetectorMemoryMapping(t *testing.T) {
	detector := &EnvDetector{}

	t.Run("virtual_gpu_detected", func(t *testing.T) {
		info := &EnvInfo{
			WebGLRenderer: "VMware SVGA II",
		}

		detected, score, evidence := detector.DetectMemoryMapping(info, nil)

		if !detected {
			t.Error("Expected virtual GPU to be detected")
		}
		if score < 35 {
			t.Errorf("Expected score >= 35, got %.2f", score)
		}
		t.Logf("Evidence: %v", evidence)
	})

	t.Run("normal_memory_no_detection", func(t *testing.T) {
		info := &EnvInfo{
			DeviceMemory:   16.0,
			MemorySize:     16384,
			WebGLRenderer: "NVIDIA GeForce RTX 3080",
		}

		detected, _, _ := detector.DetectMemoryMapping(info, nil)

		if detected {
			t.Error("Expected normal memory not to be detected")
		}
	})
}

func TestSimpleEnvDetectorTimingAttack(t *testing.T) {
	detector := &EnvDetector{}

	t.Run("high_timing_variance", func(t *testing.T) {
		info := &EnvInfo{
			TimingVariance: 0.95,
		}

		detected, score, evidence := detector.DetectTimingAttack(info, nil)

		if !detected {
			t.Error("Expected high timing variance to be detected")
		}
		if score < 30 {
			t.Errorf("Expected score >= 30, got %.2f", score)
		}
		t.Logf("Evidence: %v", evidence)
	})

	t.Run("normal_timing_no_detection", func(t *testing.T) {
		info := &EnvInfo{
			TimingVariance: 0.1,
			ExecutionTime:  0.1,
			FrameTimeDelta: 16.67,
		}

		detected, _, _ := detector.DetectTimingAttack(info, nil)

		if detected {
			t.Error("Expected normal timing not to be detected")
		}
	})
}

func TestSimpleEnvDetectorFingerprintBrowser(t *testing.T) {
	detector := &EnvDetector{}

	t.Run("detect_adspower", func(t *testing.T) {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/100.0.4896.127 AdsPower",
		}

		detected, score, evidence, browserType := detector.DetectFingerprintBrowser(info, nil)

		if !detected {
			t.Error("Expected AdsPower to be detected")
		}
		if score < 50 {
			t.Errorf("Expected score >= 50, got %.2f", score)
		}
		if browserType != "adspower" {
			t.Errorf("Expected browser type 'adspower', got '%s'", browserType)
		}
		t.Logf("Evidence: %v", evidence)
	})

	t.Run("detect_gologin", func(t *testing.T) {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/100.0.4896.127 GoLogin",
		}

		detected, _, _, browserType := detector.DetectFingerprintBrowser(info, nil)

		if !detected {
			t.Error("Expected GoLogin to be detected")
		}
		if browserType != "gologin" {
			t.Errorf("Expected browser type 'gologin', got '%s'", browserType)
		}
	})

	t.Run("normal_browser_no_detection", func(t *testing.T) {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36",
		}

		detected, _, _, _ := detector.DetectFingerprintBrowser(info, nil)

		if detected {
			t.Error("Expected normal browser not to be detected as fingerprint browser")
		}
	})
}

func TestSimpleEnvDetectorSandboxEscape(t *testing.T) {
	detector := &EnvDetector{}

	t.Run("detect_filesystem_escape", func(t *testing.T) {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) /proc/vz/test",
		}

		detected, score, evidence := detector.DetectFileSystemEscape(info, nil)

		if !detected {
			t.Error("Expected filesystem escape to be detected")
		}
		if score < 40 {
			t.Errorf("Expected score >= 40, got %.2f", score)
		}
		t.Logf("Evidence: %v", evidence)
	})

	t.Run("detect_network_escape", func(t *testing.T) {
		info := &EnvInfo{
			WebRTCIPs:    []string{"192.168.1.1", "10.0.0.1", "203.0.113.50"},
			NetworkType:  "vpn",
		}

		detected, score, evidence := detector.DetectNetworkEscape(info, nil)

		if !detected {
			t.Error("Expected network escape to be detected")
		}
		if score < 45 {
			t.Errorf("Expected score >= 45, got %.2f", score)
		}
		t.Logf("Evidence: %v", evidence)
	})

	t.Run("detect_process_escape", func(t *testing.T) {
		info := &EnvInfo{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) vboxservice",
		}

		detected, score, evidence := detector.DetectProcessEscape(info, nil)

		if !detected {
			t.Error("Expected process escape to be detected")
		}
		if score < 40 {
			t.Errorf("Expected score >= 40, got %.2f", score)
		}
		t.Logf("Evidence: %v", evidence)
	})
}
