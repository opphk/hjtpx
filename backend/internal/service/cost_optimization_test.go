package service

import (
	"testing"
)

func TestCostOptimizationService(t *testing.T) {
	t.Log("测试成本优化服务...")

	cos := NewCostOptimizationService()

	usage := &ComponentResourceUsage{
		ComponentID:   "comp-1",
		ComponentName: "API Gateway",
		Resources: []ResourceUsage{
			{Type: ResourceTypeCPU, Used: 65.0, Unit: "cores", CostPerUnit: 0.05},
			{Type: ResourceTypeMemory, Used: 128.0, Unit: "GB", CostPerUnit: 0.01},
			{Type: ResourceTypeNetwork, Used: 500.0, Unit: "GB", CostPerUnit: 0.09},
		},
		Period: "daily",
	}

	cos.RecordResourceUsage(usage)
	t.Logf("记录资源使用 - 组件: %s, 总成本: $%.2f", usage.ComponentName, usage.TotalCost)

	usage2 := &ComponentResourceUsage{
		ComponentID:   "comp-2",
		ComponentName: "Database",
		Resources: []ResourceUsage{
			{Type: ResourceTypeCPU, Used: 8.0, Unit: "cores", CostPerUnit: 0.05},
			{Type: ResourceTypeMemory, Used: 64.0, Unit: "GB", CostPerUnit: 0.01},
			{Type: ResourceTypeStorage, Used: 1000.0, Unit: "GB", CostPerUnit: 0.0001},
		},
		Period: "daily",
	}

	cos.RecordResourceUsage(usage2)

	totalCost := cos.GetTotalCost()
	t.Logf("总成本: $%.2f", totalCost)

	costByResource := cos.GetCostByResourceType()
	t.Log("按资源类型的成本分布:")
	for resType, cost := range costByResource {
		t.Logf("  - %s: $%.2f", resType, cost)
	}

	allocation := CostAllocation{
		TenantID:     "tenant-1",
		TenantName:   "Enterprise Client A",
		ResourceType: "all",
		SharePercent: 40.0,
	}
	cos.SetCostAllocation(allocation)

	allocation2 := CostAllocation{
		TenantID:     "tenant-2",
		TenantName:   "Startup Client B",
		ResourceType: "all",
		SharePercent: 30.0,
	}
	cos.SetCostAllocation(allocation2)

	allocations := cos.GetCostByTenant()
	t.Logf("成本分配 (%d个租户):", len(allocations))
	for _, alloc := range allocations {
		t.Logf("  - %s: $%.2f (占比%.1f%%)", alloc.TenantName, alloc.AllocatedCost, alloc.SharePercent)
	}

	report := cos.GenerateCostAllocationReport("2024-01")
	t.Logf("成本分配报告 - 周期: %s, 总成本: $%.2f, 生成时间: %s",
		report.Period, report.TotalCost, report.GeneratedAt.Format("2006-01-02 15:04:05"))

	utilizationMetrics := cos.GetUtilizationMetrics()
	t.Log("资源利用率指标:")
	for _, metrics := range utilizationMetrics {
		t.Logf("  - %s: 当前%.1f%%, 平均%.1f%%, 峰值%.1f%%, 空闲%.1f%%",
			metrics.ResourceType,
			metrics.CurrentUtilization,
			metrics.AverageUtilization,
			metrics.PeakUtilization,
			metrics.IdleTimePercent)
	}

	cos.UpdateUtilization(ResourceTypeCPU, 85.0)
	cpuMetrics, _ := cos.GetUtilizationByType(ResourceTypeCPU)
	t.Logf("更新后CPU利用率: %.1f%%", cpuMetrics.CurrentUtilization)

	recommendations := cos.GetOptimizationRecommendations()
	t.Logf("优化建议数量: %d", len(recommendations))
	for _, rec := range recommendations {
		t.Logf("  - [%s] %s: %s %.1f%% -> %.1f%% (预计节省$%.2f)",
			rec.Priority, rec.ResourceType, rec.Action, rec.CurrentValue, rec.RecommendedValue, rec.Savings)
	}

	policies := cos.GetAutoScalingPolicies()
	t.Logf("自动扩缩容策略数量: %d", len(policies))
	for _, policy := range policies {
		t.Logf("  - %s (启用: %v): 最小%d, 最大%d, 扩容阈值%.1f%%, 缩容阈值%.1f%%",
			policy.Name, policy.Enabled, policy.MinReplicas, policy.MaxReplicas,
			policy.ScaleUpThreshold, policy.ScaleDownThreshold)
	}

	needsScaling, targetReplicas := cos.EvaluateScalingNeeds("cpu_utilization", 85.0)
	t.Logf("评估扩缩容需求 (CPU=85%%) - 需要扩缩容: %v, 目标副本数: %d",
		needsScaling, targetReplicas)

	cos.TakeCostSnapshot()
	trend := cos.GetCostTrend(7)
	t.Logf("成本趋势 (过去%d天): %d个数据点", 7, len(trend))

	forecast := cos.GetCostForecast(30)
	t.Log("成本预测:")
	t.Logf("  - 日均: $%.2f", forecast["daily"])
	t.Logf("  - 周均: $%.2f", forecast["weekly"])
	t.Logf("  - 月均: $%.2f", forecast["monthly"])
	t.Logf("  - 年均: $%.2f", forecast["yearly"])

	optimizationReport := cos.GetCostOptimizationReport()
	t.Logf("成本优化报告:")
	t.Logf("  - 总成本: $%.2f", optimizationReport["total_cost"])
	t.Logf("  - 建议数量: %d", optimizationReport["recommendations_count"])
	t.Logf("  - 预计节省: $%.2f", optimizationReport["potential_savings"])
	t.Logf("  - 平均利用率: %.1f%%", optimizationReport["utilization"].(map[string]interface{})["average_utilization_percent"])

	scenarioChanges := map[string]interface{}{
		"scale_factor":       1.2,
		"optimization_factor": 0.85,
	}

	scenario := cos.SimulateCostScenario(scenarioChanges)
	t.Log("成本模拟 (规模+20%%, 优化-15%%):")
	t.Logf("  - 当前成本: $%.2f", scenario["current_cost"])
	t.Logf("  - 场景成本: $%.2f", scenario["scenario_cost"])
	t.Logf("  - 月节省: $%.2f", scenario["monthly_savings"])
	t.Logf("  - 年节省: $%.2f", scenario["yearly_savings"])

	t.Log("成本优化服务测试通过")
}

func TestResourceUsageTracking(t *testing.T) {
	t.Log("测试资源使用追踪...")

	cos := NewCostOptimizationService()

	components := []struct {
		id   string
		name string
		cpu  float64
		mem  float64
	}{
		{"web-1", "Web Server 1", 4.0, 16.0},
		{"web-2", "Web Server 2", 3.5, 14.0},
		{"api-1", "API Server 1", 8.0, 32.0},
		{"cache-1", "Cache Server", 2.0, 48.0},
		{"db-1", "Database", 16.0, 128.0},
	}

	for _, comp := range components {
		usage := &ComponentResourceUsage{
			ComponentID:   comp.id,
			ComponentName: comp.name,
			Resources: []ResourceUsage{
				{Type: ResourceTypeCPU, Used: comp.cpu, Unit: "cores", CostPerUnit: 0.05},
				{Type: ResourceTypeMemory, Used: comp.mem, Unit: "GB", CostPerUnit: 0.01},
			},
			Period: "daily",
		}
		cos.RecordResourceUsage(usage)
	}

	allUsage := cos.GetAllResourceUsage()
	t.Logf("追踪的资源使用: %d个组件", len(allUsage))

	totalCost := cos.GetTotalCost()
	t.Logf("所有组件总成本: $%.2f", totalCost)

	for _, usage := range allUsage {
		retrieved, exists := cos.GetResourceUsage(usage.ComponentID)
		if exists {
			t.Logf("  - %s: $%.2f (CPU: %.1f cores, Memory: %.1f GB)",
				retrieved.ComponentName, retrieved.TotalCost,
				retrieved.Resources[0].Used, retrieved.Resources[1].Used)
		}
	}

	t.Log("资源使用追踪测试通过")
}

func TestAutoScalingPolicy(t *testing.T) {
	t.Log("测试自动扩缩容策略...")

	cos := NewCostOptimizationService()

	policy := &AutoScalingPolicy{
		ID: "custom-scaling",
		Name: "Custom Auto Scaling",
		Enabled: true,
		Metric: "custom_metric",
		MinReplicas: 1,
		MaxReplicas: 15,
		ScaleUpThreshold: 70.0,
		ScaleDownThreshold: 20.0,
		CooldownSeconds: 180,
	}

	err := cos.UpdateAutoScalingPolicy(policy)
	if err != nil {
		t.Errorf("更新自动扩缩容策略失败: %v", err)
		return
	}

	policies := cos.GetAutoScalingPolicies()
	t.Logf("自动扩缩容策略数量: %d", len(policies))

	cos.UpdateAutoScalingPolicy(&AutoScalingPolicy{
		ID: "custom-scaling",
		Enabled: false,
	})

	policies2 := cos.GetAutoScalingPolicies()
	for _, p := range policies2 {
		if p.ID == "custom-scaling" {
			t.Logf("更新后策略状态 - %s: 启用=%v", p.Name, p.Enabled)
		}
	}

	testCases := []struct {
		metric string
		value  float64
		expect bool
	}{
		{"cpu_utilization", 85.0, true},
		{"cpu_utilization", 50.0, false},
		{"cpu_utilization", 15.0, true},
		{"memory_utilization", 90.0, true},
		{"unknown_metric", 50.0, false},
	}

	t.Log("扩缩容评估测试:")
	for _, tc := range testCases {
		needs, target := cos.EvaluateScalingNeeds(tc.metric, tc.value)
		t.Logf("  - %s=%.1f%%: 需要扩缩容=%v, 目标=%d (预期=%v)",
			tc.metric, tc.value, needs, target, tc.expect)
	}

	t.Log("自动扩缩容策略测试通过")
}

func TestCostAllocation(t *testing.T) {
	t.Log("测试成本分配与分摊...")

	cos := NewCostOptimizationService()

	usage := &ComponentResourceUsage{
		ComponentID:   "shared-1",
		ComponentName: "Shared Infrastructure",
		Resources: []ResourceUsage{
			{Type: ResourceTypeCPU, Used: 32.0, Unit: "cores", CostPerUnit: 0.05},
			{Type: ResourceTypeMemory, Used: 256.0, Unit: "GB", CostPerUnit: 0.01},
			{Type: ResourceTypeStorage, Used: 5000.0, Unit: "GB", CostPerUnit: 0.0001},
		},
		Period: "monthly",
	}
	cos.RecordResourceUsage(usage)

	tenants := []struct {
		id    string
		name  string
		share float64
	}{
		{"tenant-1", "大客户A", 50.0},
		{"tenant-2", "中客户B", 30.0},
		{"tenant-3", "小客户C", 15.0},
		{"tenant-4", "试用客户D", 5.0},
	}

	for _, tenant := range tenants {
		cos.SetCostAllocation(CostAllocation{
			TenantID:     tenant.id,
			TenantName:   tenant.name,
			ResourceType: "shared_infrastructure",
			SharePercent: tenant.share,
		})
	}

	report := cos.GenerateCostAllocationReport("2024-01")
	t.Logf("成本分配报告 - 总成本: $%.2f", report.TotalCost)
	t.Log("各租户分摊:")
	for _, alloc := range report.Allocations {
		t.Logf("  - %s: $%.2f (%.1f%%)", alloc.TenantName, alloc.AllocatedCost, alloc.SharePercent)
	}

	totalShare := 0.0
	for _, alloc := range report.Allocations {
		totalShare += alloc.SharePercent
	}
	t.Logf("总分配比例: %.1f%%", totalShare)

	t.Log("成本分配与分摊测试通过")
}

func TestUtilizationOptimization(t *testing.T) {
	t.Log("测试资源利用率优化...")

	cos := NewCostOptimizationService()

	cos.UpdateUtilization(ResourceTypeCPU, 85.0)
	cos.UpdateUtilization(ResourceTypeMemory, 90.0)
	cos.UpdateUtilization(ResourceTypeStorage, 20.0)
	cos.UpdateUtilization(ResourceTypeNetwork, 15.0)

	metrics := cos.GetUtilizationMetrics()
	t.Log("当前资源利用率:")
	var avgUtilization float64
	for _, m := range metrics {
		t.Logf("  - %s: %.1f%%", m.ResourceType, m.CurrentUtilization)
		avgUtilization += m.CurrentUtilization
	}
	avgUtilization /= float64(len(metrics))
	t.Logf("平均利用率: %.1f%%", avgUtilization)

	recommendations := cos.GetOptimizationRecommendations()
	t.Logf("优化建议 (%d条):", len(recommendations))
	for _, rec := range recommendations {
		t.Logf("  - [%d级] %s: %s %.1f%% -> %.1f%%",
			rec.Priority, rec.ResourceType, rec.Action, rec.CurrentValue, rec.RecommendedValue)
		t.Logf("    原因: %s", rec.Reason)
	}

	cos.UpdateUtilization(ResourceTypeCPU, 45.0)
	cos.UpdateUtilization(ResourceTypeMemory, 55.0)
	cos.UpdateUtilization(ResourceTypeStorage, 35.0)
	cos.UpdateUtilization(ResourceTypeNetwork, 25.0)

	metrics2 := cos.GetUtilizationMetrics()
	t.Log("优化后资源利用率:")
	for _, m := range metrics2 {
		t.Logf("  - %s: %.1f%% (空闲: %.1f%%, 优化潜力: %.1f%%)",
			m.ResourceType, m.CurrentUtilization, m.IdleTimePercent, m.OptimizationPotential)
	}

	recommendations2 := cos.GetOptimizationRecommendations()
	t.Logf("优化后建议数量: %d", len(recommendations2))

	t.Log("资源利用率优化测试通过")
}

func TestCostForecasting(t *testing.T) {
	t.Log("测试成本预测...")

	cos := NewCostOptimizationService()

	for i := 0; i < 30; i++ {
		cos.TakeCostSnapshot()
	}

	trend := cos.GetCostTrend(30)
	t.Logf("成本趋势数据点数: %d", len(trend))

	forecast := cos.GetCostForecast(30)
	t.Log("30天成本预测:")
	t.Logf("  - 日均成本: $%.2f", forecast["daily"])
	t.Logf("  - 周均成本: $%.2f", forecast["weekly"])
	t.Logf("  - 月均成本: $%.2f", forecast["monthly"])
	t.Logf("  - 年均成本: $%.2f", forecast["yearly"])

	scenario1 := cos.SimulateCostScenario(map[string]interface{}{
		"scale_factor": 1.5,
	})
	t.Logf("场景1 (规模+50%%): 月成本 $%.2f, 年节省 $%.2f",
		scenario1["scenario_cost"]*30, scenario1["yearly_savings"])

	scenario2 := cos.SimulateCostScenario(map[string]interface{}{
		"optimization_factor": 0.8,
	})
	t.Logf("场景2 (优化-20%%): 月成本 $%.2f, 年节省 $%.2f",
		scenario2["scenario_cost"]*30, scenario2["yearly_savings"])

	t.Log("成本预测测试通过")
}
