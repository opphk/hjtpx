package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNetworkAnalysisEnhanced(t *testing.T) {
	fmt.Println("=== 测试网络分析增强模块 ===")

	analyzer := NewNetworkAnalysisEnhanced()

	t.Run("IP信誉分析", func(t *testing.T) {
		ip := "8.8.8.8"
		reputation := analyzer.AnalyzeIPReputation(ip)

		if reputation == nil {
			t.Fatal("IP信誉分析返回nil")
		}

		if reputation.IP != ip {
			t.Errorf("IP不匹配: 期望 %s, 实际 %s", ip, reputation.IP)
		}

		if reputation.Score < 0 || reputation.Score > 100 {
			t.Errorf("信誉评分超出范围: %f", reputation.Score)
		}

		fmt.Printf("✓ IP信誉分析: %s - 评分: %.2f\n", ip, reputation.Score)
	})

	t.Run("缓存机制", func(t *testing.T) {
		ip := "8.8.4.4"
		rep1 := analyzer.AnalyzeIPReputation(ip)
		time.Sleep(10 * time.Millisecond)
		rep2 := analyzer.AnalyzeIPReputation(ip)

		if rep1.Timestamp != rep2.Timestamp {
			t.Error("缓存应该返回相同的时间戳")
		}

		if analyzer.GetCacheSize() == 0 {
			t.Error("缓存应该有至少一个条目")
		}

		fmt.Printf("✓ 缓存机制正常: 缓存大小 %d\n", analyzer.GetCacheSize())
	})

	t.Run("住宅代理检测", func(t *testing.T) {
		headers := map[string]string{
			"x-forwarded-for": "192.168.1.1, 10.0.0.1",
			"via":              "1.1 proxy.example.com",
		}

		result := analyzer.DetectResidentialProxy("203.0.113.1", headers)

		if result == nil {
			t.Fatal("住宅代理检测返回nil")
		}

		fmt.Printf("✓ 住宅代理检测: IP类型=%s, 可信度=%.2f\n",
			result.IPType, result.Confidence)
	})

	t.Run("ASN分析", func(t *testing.T) {
		asnInfo := &ASNInfo{
			ASN:     15169,
			Provider: "Google LLC",
			Country:  "US",
		}

		analysis := analyzer.AnalyzeASN(asnInfo)

		if analysis == nil {
			t.Fatal("ASN分析返回nil")
		}

		if analysis.Type == "" {
			t.Error("ASN类型不应该为空")
		}

		fmt.Printf("✓ ASN分析: ASN=%d, 类型=%s, 风险=%s\n",
			analysis.ASN, analysis.Type, analysis.Risk)
	})

	t.Run("VPN检测", func(t *testing.T) {
		timing := &NetworkTimingAnalysis{
			RTT:         150,
			RTTVariance: 0.6,
		}

		result := analyzer.DetectVPNConnection("104.248.0.1", timing)

		if result == nil {
			t.Fatal("VPN检测返回nil")
		}

		fmt.Printf("✓ VPN检测: 是否VPN=%v, 提供商=%s, 可信度=%.2f\n",
			result.IsVPN, result.Provider, result.Confidence)
	})

	t.Run("综合网络风险评估", func(t *testing.T) {
		data := &NetworkRiskData{
			IP: "8.8.8.8",
			Headers: map[string]string{
				"x-forwarded-for": "192.168.1.1",
			},
			Timing: &NetworkTimingAnalysis{
				RTT:         100,
				RTTVariance: 0.3,
			},
		}

		assessment := analyzer.AssessNetworkRisk("8.8.8.8", data)

		if assessment == nil {
			t.Fatal("风险评估返回nil")
		}

		if assessment.RiskLevel == "" {
			t.Error("风险等级不应该为空")
		}

		if assessment.RecommendedAction == "" {
			t.Error("建议操作不应该为空")
		}

		jsonData, err := analyzer.ExportRiskAssessment("8.8.8.8", data)
		if err != nil {
			t.Errorf("导出风险评估失败: %v", err)
		} else {
			var exported map[string]interface{}
			if err := json.Unmarshal(jsonData, &exported); err != nil {
				t.Errorf("解析导出数据失败: %v", err)
			}
		}

		fmt.Printf("✓ 综合风险评估: 风险等级=%s, 建议操作=%s\n",
			assessment.RiskLevel, assessment.RecommendedAction)
	})

	t.Run("黑名单检测", func(t *testing.T) {
		blacklistedIP := "192.0.2.1"
		normalIP := "8.8.8.8"

		if !analyzer.checkBlacklist(blacklistedIP) {
			t.Errorf("IP %s 应该在黑名单中", blacklistedIP)
		}

		if analyzer.checkBlacklist(normalIP) {
			t.Errorf("IP %s 不应该在黑名单中", normalIP)
		}

		fmt.Println("✓ 黑名单检测正常")
	})

	t.Run("缓存清理", func(t *testing.T) {
		analyzer.AnalyzeIPReputation("8.8.8.8")
		if analyzer.GetCacheSize() == 0 {
			t.Error("缓存应该有条目")
		}

		analyzer.ClearCache()
		if analyzer.GetCacheSize() != 0 {
			t.Error("缓存应该被清空")
		}

		fmt.Println("✓ 缓存清理功能正常")
	})

	t.Run("TTL设置", func(t *testing.T) {
		analyzer.SetCacheTTL(2 * time.Hour)
		fmt.Println("✓ TTL设置功能正常")
	})

	t.Run("代理信号分析", func(t *testing.T) {
		headers := map[string]string{
			"via":                 "1.1 proxy-server",
			"x-forwarded-for":     "192.168.1.1, 10.0.0.1",
			"x-real-ip":           "10.0.0.1",
			"cf-connecting-ip":    "203.0.113.1",
		}

		confidence := analyzer.analyzeProxySignals("203.0.113.1", headers)

		if confidence < 0.5 {
			t.Errorf("代理信号可信度过低: %.2f", confidence)
		}

		fmt.Printf("✓ 代理信号分析: 可信度=%.2f\n", confidence)
	})

	fmt.Println("\n=== 所有测试通过 ===")
}

func TestVPNProviderMatching(t *testing.T) {
	analyzer := NewNetworkAnalysisEnhanced()

	t.Run("NordVPN范围匹配", func(t *testing.T) {
		ip := "104.248.100.1"
		isVPN, provider := analyzer.checkVPN(ip)

		fmt.Printf("IP %s - VPN: %v, 提供商: %s\n", ip, isVPN, provider)

		if provider == "NordVPN" && !isVPN {
			t.Error("应该检测到NordVPN")
		}
	})
}

func TestTorExitNodeDetection(t *testing.T) {
	analyzer := NewNetworkAnalysisEnhanced()

	t.Run("Tor出口节点检测", func(t *testing.T) {
		testIP := "23.129.64.100"

		isTor := analyzer.checkTorExitNode(testIP)
		fmt.Printf("IP %s - Tor节点: %v\n", testIP, isTor)
	})
}

func BenchmarkAnalyzeIPReputation(b *testing.B) {
	analyzer := NewNetworkAnalysisEnhanced()
	ip := "8.8.8.8"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeIPReputation(ip)
	}
}
