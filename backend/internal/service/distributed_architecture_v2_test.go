package service

import (
	"context"
	"testing"
	"time"
)

func TestDistributedArchitectureV2(t *testing.T) {
	t.Log("测试全球分布式架构v2...")

	da := NewDistributedArchitectureV2()

	dc := &DataCenter{
		ID:   "dc-1",
		Name: "US East",
		Region: "us-east-1",
		Location: GeoCoord{
			Latitude:  40.7128,
			Longitude: -74.0060,
			Country:   "USA",
			City:      "New York",
		},
		Priority: 1,
		Capacity: 1000,
		Metadata: map[string]string{
			"provider": "aws",
		},
	}

	err := da.RegisterDataCenter(dc)
	if err != nil {
		t.Errorf("注册数据中心失败: %v", err)
		return
	}

	dc2 := &DataCenter{
		ID:   "dc-2",
		Name: "EU West",
		Region: "eu-west-1",
		Location: GeoCoord{
			Latitude:  51.5074,
			Longitude: -0.1278,
			Country:   "UK",
			City:      "London",
		},
		Priority: 2,
		Capacity: 800,
		Metadata: map[string]string{
			"provider": "gcp",
		},
	}

	err = da.RegisterDataCenter(dc2)
	if err != nil {
		t.Errorf("注册数据中心2失败: %v", err)
		return
	}

	t.Log("数据中心注册成功")

	retrievedDC, exists := da.GetDataCenter("dc-1")
	if !exists {
		t.Error("获取数据中心失败")
		return
	}
	t.Logf("获取数据中心: %s, 区域: %s", retrievedDC.Name, retrievedDC.Region)

	da.UpdateDataCenterLoad("dc-1", 500)
	da.UpdateDataCenterLatency("dc-1", 45)

	dcUpdated, _ := da.GetDataCenter("dc-1")
	t.Logf("数据中心1负载: %d, 延迟: %dms", dcUpdated.CurrentLoad.Load(), dcUpdated.LatencyMs.Load())

	healthyDCs := da.GetHealthyDataCenters()
	t.Logf("健康数据中心数量: %d", len(healthyDCs))

	dnsRecord := &DNSRecord{
		ID:        "dns-1",
		Domain:    "api.example.com",
		Type:      "A",
		Values:    []string{"1.2.3.4"},
		TTL:       300,
		Priority:  1,
		Weight:    100,
		DataCenter: "dc-1",
	}

	err = da.CreateDNSRecord(dnsRecord)
	if err != nil {
		t.Errorf("创建DNS记录失败: %v", err)
		return
	}

	resolveResult, err := da.ResolveDNS("api.example.com", "192.168.1.1")
	if err != nil {
		t.Errorf("DNS解析失败: %v", err)
		return
	}

	t.Logf("DNS解析结果 - IP: %s, 数据中心: %s, 延迟: %dms",
		resolveResult.IP, resolveResult.DataCenter, resolveResult.LatencyMs)

	policy := &TrafficPolicy{
		ID:              "policy-1",
		Name:            "Default Policy",
		Strategy:        "weighted",
		Enabled:         true,
		HealthCheckPath: "/health",
		HealthCheckIntvl: 30,
		FailoverEnabled: true,
		Weights: map[string]int{
			"dc-1": 70,
			"dc-2": 30,
		},
	}

	err = da.CreateTrafficPolicy(policy)
	if err != nil {
		t.Errorf("创建流量策略失败: %v", err)
		return
	}

	allocation, err := da.ApplyTrafficPolicy("policy-1")
	if err != nil {
		t.Errorf("应用流量策略失败: %v", err)
		return
	}

	t.Logf("流量分配 - 数据中心: %s, 百分比: %.2f%%, 当前负载: %d",
		allocation.DataCenterID, allocation.Percentage, allocation.CurrentLoad)

	clientLocation := GeoCoord{
		Latitude:  40.7128,
		Longitude: -74.0060,
		Country:   "USA",
		City:      "New York",
	}

	nearestDC := da.FindNearestDataCenter(clientLocation)
	if nearestDC != nil {
		t.Logf("最近数据中心: %s, 区域: %s", nearestDC.Name, nearestDC.Region)
	}

	da.UpdateDataCenterHealth("dc-1", false)
	t.Log("数据中心1健康检查失败，触发故障转移")

	stats := da.GetArchitectureStats()
	t.Logf("架构统计 - 总数据中心: %v, 健康数据中心: %v, 容量利用率: %.2f%%",
		stats["total_data_centers"],
		stats["healthy_data_centers"],
		stats["capacity_utilization"])

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthResult := da.HealthCheck(ctx)
	t.Logf("健康检查结果 - 状态: %s, 健康数量: %v, 平均延迟: %vms",
		healthResult["status"],
		healthResult["healthy_count"],
		healthResult["average_latency_ms"])

	da.UpdateDataCenterHealth("dc-1", true)
	t.Log("数据中心1恢复")

	da.UpdateDataCenterHealth("dc-2", false)
	da.UpdateDataCenterHealth("dc-1", false)
	healthResult2 := da.HealthCheck(ctx)
	t.Logf("全部数据中心故障 - 状态: %s", healthResult2["status"])

	da.UpdateDataCenterHealth("dc-1", true)
	da.UpdateDataCenterHealth("dc-2", true)

	t.Log("全球分布式架构v2测试通过")
}

func TestGeoDistanceCalculation(t *testing.T) {
	t.Log("测试地理距离计算...")

	da := NewDistributedArchitectureV2()

	loc1 := GeoCoord{
		Latitude:  40.7128,
		Longitude: -74.0060,
		Country:   "USA",
		City:      "New York",
	}

	loc2 := GeoCoord{
		Latitude:  51.5074,
		Longitude: -0.1278,
		Country:   "UK",
		City:      "London",
	}

	distance := da.CalculateDistance(loc1, loc2)
	t.Logf("纽约到伦敦的距离: %.2f km", distance)

	if distance < 5500 || distance > 5600 {
		t.Errorf("距离计算异常: %.2f km (预期约5570 km)", distance)
	}
}

func TestTrafficPolicyWeights(t *testing.T) {
	t.Log("测试流量策略权重分配...")

	da := NewDistributedArchitectureV2()

	dc1 := &DataCenter{
		ID:       "dc-heavy",
		Name:     "Heavy Load DC",
		Region:   "us-east",
		Priority: 1,
		Capacity: 100,
	}
	dc1.Healthy.Store(true)
	dc1.CurrentLoad.Store(90)

	dc2 := &DataCenter{
		ID:       "dc-light",
		Name:     "Light Load DC",
		Region:   "us-west",
		Priority: 2,
		Capacity: 100,
	}
	dc2.Healthy.Store(true)
	dc2.CurrentLoad.Store(10)

	da.RegisterDataCenter(dc1)
	da.RegisterDataCenter(dc2)

	policy := &TrafficPolicy{
		ID:       "test-policy",
		Name:     "Test Policy",
		Strategy: "weighted",
		Enabled:  true,
		Weights: map[string]int{
			"dc-heavy": 50,
			"dc-light": 50,
		},
	}

	da.CreateTrafficPolicy(policy)

	for i := 0; i < 10; i++ {
		allocation, _ := da.ApplyTrafficPolicy("test-policy")
		t.Logf("分配 #%d - 数据中心: %s, 负载: %d",
			i+1, allocation.DataCenterID, allocation.CurrentLoad)
	}

	t.Log("流量策略权重分配测试通过")
}
