package service

import (
	"testing"
)

func TestNewFingerprintService(t *testing.T) {
	fingerprintService := NewFingerprintService()
	if fingerprintService == nil {
		t.Error("NewFingerprintService 返回了 nil")
	}
}

func TestGenerateFingerprintFingerprint(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint, err := fingerprintService.GenerateFingerprint("test-user-agent", map[string]string{
		"Accept-Language": "en-US",
	})
	if err != nil {
		t.Errorf("生成指纹失败: %v", err)
	}
	if fingerprint == "" {
		t.Error("指纹不应为空")
	}
}

func TestValidateFingerprintFingerprint(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint, err := fingerprintService.GenerateFingerprint("test-user-agent", nil)
	if err != nil {
		t.Skipf("无法生成指纹，跳过测试: %v", err)
	}
	
	valid := fingerprintService.ValidateFingerprint(fingerprint)
	if !valid {
		t.Error("有效的指纹应该通过验证")
	}
}

func TestValidateFingerprint_Invalid(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	valid := fingerprintService.ValidateFingerprint("invalid-fingerprint-12345")
	if valid {
		t.Error("无效的指纹不应该通过验证")
	}
}

func TestCompareFingerprintsFingerprint(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fp1, _ := fingerprintService.GenerateFingerprint("user-agent-1", nil)
	fp2, _ := fingerprintService.GenerateFingerprint("user-agent-2", nil)
	
	if fp1 == fp2 {
		t.Error("不同的用户代理应该生成不同的指纹")
	}
}

func TestGetFingerprintComponents(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	components, err := fingerprintService.GetFingerprintComponents("test-user-agent", map[string]string{
		"Accept-Language": "en-US",
	})
	if err != nil {
		t.Errorf("获取指纹组件失败: %v", err)
	}
	if components == nil {
		t.Error("指纹组件不应为 nil")
	}
}

func TestAnalyzeFingerprintFingerprint(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint, _ := fingerprintService.GenerateFingerprint("test-user-agent", nil)
	
	analysis, err := fingerprintService.AnalyzeFingerprint(fingerprint)
	if err != nil {
		t.Errorf("分析指纹失败: %v", err)
	}
	if analysis == nil {
		t.Error("指纹分析结果不应为 nil")
	}
}

func TestDetectFingerprintAnomaly(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint, _ := fingerprintService.GenerateFingerprint("test-user-agent", nil)
	
	isAnomaly, err := fingerprintService.DetectFingerprintAnomaly(fingerprint)
	if err != nil {
		t.Errorf("检测指纹异常失败: %v", err)
	}
	if isAnomaly {
		t.Log("检测到指纹异常")
	}
}

func TestUpdateFingerprintCache(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint := fingerprintService.GenerateFingerprintString("cache-test")
	err := fingerprintService.UpdateFingerprintCache(fingerprint, "user-id-123")
	if err != nil {
		t.Errorf("更新指纹缓存失败: %v", err)
	}
}

func TestGetFingerprintFromCache(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint := fingerprintService.GenerateFingerprintString("cache-get-test")
	err := fingerprintService.UpdateFingerprintCache(fingerprint, "user-id-456")
	if err != nil {
		t.Skipf("无法更新缓存，跳过测试: %v", err)
	}
	
	cachedFingerprint, err := fingerprintService.GetFingerprintFromCache("user-id-456")
	if err != nil {
		t.Errorf("从缓存获取指纹失败: %v", err)
	}
	if cachedFingerprint == "" {
		t.Error("缓存的指纹不应为空")
	}
}

func TestDeleteFingerprintCache(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprint := fingerprintService.GenerateFingerprintString("cache-delete-test")
	err := fingerprintService.UpdateFingerprintCache(fingerprint, "user-id-delete")
	if err != nil {
		t.Skipf("无法更新缓存，跳过测试: %v", err)
	}
	
	err = fingerprintService.DeleteFingerprintCache("user-id-delete")
	if err != nil {
		t.Errorf("删除指纹缓存失败: %v", err)
	}
}

func TestClearAllFingerprints(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	err := fingerprintService.ClearAllFingerprints()
	if err != nil {
		t.Errorf("清空所有指纹失败: %v", err)
	}
}

func TestGetFingerprintStats(t *testing.T) {
	fingerprintService := NewFingerprintService()
	
	fingerprintService.GenerateFingerprint("test-user-agent", nil)
	
	stats, err := fingerprintService.GetFingerprintStats()
	if err != nil {
		t.Errorf("获取指纹统计失败: %v", err)
	}
	if stats == nil {
		t.Error("指纹统计不应为 nil")
	}
}
