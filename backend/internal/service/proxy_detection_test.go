package service

import (
	"testing"
)

func TestNewProxyDetectionService(t *testing.T) {
	proxyService := NewProxyDetectionService()
	if proxyService == nil {
		t.Error("NewProxyDetectionService 返回了 nil")
	}
}

func TestDetectProxy(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	result, err := proxyService.DetectProxy("192.168.1.1")
	if err != nil {
		t.Errorf("检测代理失败: %v", err)
	}
	if result == nil {
		t.Error("检测结果不应为 nil")
	}
}

func TestDetectProxy_WithHeaders(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	headers := map[string]string{
		"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
		"Via":             "1.1 proxy-server",
	}
	
	result, err := proxyService.DetectProxyWithHeaders("192.168.1.1", headers)
	if err != nil {
		t.Errorf("检测代理失败: %v", err)
	}
	if result == nil {
		t.Error("检测结果不应为 nil")
	}
}

func TestIsProxy(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	result, err := proxyService.DetectProxy("192.168.1.1")
	if err != nil {
		t.Skipf("检测失败: %v", err)
	}
	
	isProxy, err := proxyService.IsProxy(result.IP)
	if err != nil {
		t.Errorf("检查代理失败: %v", err)
	}
	if isProxy && !result.IsProxy {
		t.Error("代理检测结果不一致")
	}
}

func TestGetProxyInfo(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	info, err := proxyService.GetProxyInfo("8.8.8.8")
	if err != nil {
		t.Errorf("获取代理信息失败: %v", err)
	}
	if info == nil {
		t.Error("代理信息不应为 nil")
	}
}

func TestCheckVPN(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	isVPN, err := proxyService.CheckVPN("192.168.1.1")
	if err != nil {
		t.Errorf("检查 VPN 失败: %v", err)
	}
	if isVPN {
		t.Log("检测到 VPN 连接")
	}
}

func TestCheckTor(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	isTor, err := proxyService.CheckTor("192.168.1.1")
	if err != nil {
		t.Errorf("检查 Tor 失败: %v", err)
	}
	if isTor {
		t.Log("检测到 Tor 出口节点")
	}
}

func TestCheckHosting(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	isHosting, err := proxyService.CheckHosting("192.168.1.1")
	if err != nil {
		t.Errorf("检查托管服务失败: %v", err)
	}
	if isHosting {
		t.Log("检测到托管服务/数据中心")
	}
}

func TestGetIPReputation(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	reputation, err := proxyService.GetIPReputation("192.168.1.1")
	if err != nil {
		t.Errorf("获取 IP 信誉失败: %v", err)
	}
	if reputation == nil {
		t.Error("IP 信誉不应为 nil")
	}
}

func TestUpdateIPCache(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	info := &ProxyInfo{
		IP:       "192.168.1.1",
		IsProxy:  true,
		ProxyType: "HTTP",
	}
	
	err := proxyService.UpdateIPCache(info)
	if err != nil {
		t.Errorf("更新 IP 缓存失败: %v", err)
	}
}

func TestClearIPCache(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	err := proxyService.ClearIPCache()
	if err != nil {
		t.Errorf("清空 IP 缓存失败: %v", err)
	}
}

func TestGetDetectionStats(t *testing.T) {
	proxyService := NewProxyDetectionService()
	
	proxyService.DetectProxy("192.168.1.1")
	
	stats, err := proxyService.GetDetectionStats()
	if err != nil {
		t.Errorf("获取检测统计失败: %v", err)
	}
	if stats == nil {
		t.Error("检测统计不应为 nil")
	}
}
