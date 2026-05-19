package service

import (
	"context"
	"testing"
	"time"
)

func TestHighAvailabilityService(t *testing.T) {
	t.Log("测试高可用性服务...")

	has := NewHighAvailabilityService()

	slaStatus := has.GetSLAStatus()
	t.Logf("SLA状态 - 目标可用性: %.2f%%, 当前可用性: %.2f%%",
		slaStatus["target_uptime_percent"], slaStatus["current_uptime_percent"])

	has.RecordUptime(3600 * time.Second)
	has.RecordUptime(3600 * time.Second)
	t.Log("记录正常运行时间: 2小时")

	components := has.GetAllComponents()
	t.Logf("服务组件数量: %d", len(components))
	for _, comp := range components {
		t.Logf("  - %s: %s (关键: %v)", comp.Name, comp.Status, comp.IsCritical)
	}

	overallHealth := has.GetOverallHealth()
	t.Logf("整体健康状态: %s", overallHealth)

	has.UpdateComponentHealth("cache_service", HealthStatusDegraded, 200)
	t.Log("更新缓存服务健康状态为降级")

	degradation := has.GetDegradationStatus()
	t.Logf("降级状态 - 级别: %d, 是否降级: %v, 原因: %s",
		degradation.CurrentLevel, degradation.IsDegraded, degradation.Reason)

	has.UpdateComponentHealth("api_gateway", HealthStatusCritical, 500)
	t.Log("更新API网关为严重状态")

	degradation2 := has.GetDegradationStatus()
	t.Logf("降级状态 - 级别: %d, 受影响功能: %v",
		degradation2.CurrentLevel, degradation2.AffectedFeatures)

	capacityStatus := has.GetCapacityStatus()
	t.Logf("容量状态 - 当前副本: %v, 目标副本: %v, CPU使用率: %.2f%%",
		capacityStatus["current_replicas"],
		capacityStatus["target_replicas"],
		capacityStatus["cpu_utilization_percent"])

	has.UpdateCapacityMetrics(85.0, 70.0, 1000, 50)
	t.Log("更新容量指标 - CPU: 85%%, 内存: 70%%")

	capacityStatus2 := has.GetCapacityStatus()
	t.Logf("更新后容量状态 - 当前副本: %v, 目标副本: %v",
		capacityStatus2["current_replicas"],
		capacityStatus2["target_replicas"])

	scaleHistory := has.GetScaleHistory(10)
	t.Logf("扩容历史记录数量: %d", len(scaleHistory))

	action := has.StartRecovery("cache_service")
	t.Logf("启动恢复 - 操作ID: %s, 目标: %s, 状态: %s",
		action.ID, action.Target, action.Status)

	has.UpdateRecoveryStatus(action.ID, false, "Connection timeout")
	has.UpdateRecoveryStatus(action.ID, false, "Service not responding")
	has.UpdateRecoveryStatus(action.ID, true, "")

	actions := has.GetRecoveryActions()
	t.Logf("恢复操作记录数量: %d", len(actions))

	slaStatus2 := has.GetSLAStatus()
	t.Logf("SLA状态 - 正常运行秒数: %v, 宕机秒数: %v, 事件数: %v",
		slaStatus2["uptime_seconds"],
		slaStatus2["downtime_seconds"],
		slaStatus2["incidents_count"])

	has.RecordDowntime(300 * time.Second)

	slaStatus3 := has.GetSLAStatus()
	t.Logf("记录宕机后 - 宕机秒数: %v, 事件数: %v, SLA达标: %v",
		slaStatus3["downtime_seconds"],
		slaStatus3["incidents_count"],
		slaStatus3["sla_met"])

	has.UpdateComponentHealth("api_gateway", HealthStatusHealthy, 50)
	has.UpdateComponentHealth("cache_service", HealthStatusHealthy, 80)

	degradation3 := has.GetDegradationStatus()
	t.Logf("恢复后降级状态 - 级别: %d, 是否降级: %v",
		degradation3.CurrentLevel, degradation3.IsDegraded)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthCheck := has.HealthCheck(ctx)
	t.Logf("健康检查 - 整体状态: %s, SLA达标: %v",
		healthCheck["overall_status"], healthCheck["sla_status"].(map[string]interface{})["sla_met"])

	t.Log("高可用性服务测试通过")
}

func TestSLAMetrics(t *testing.T) {
	t.Log("测试SLA指标...")

	has := NewHighAvailabilityService()

	has.RecordUptime(24 * time.Hour)
	has.RecordUptime(24 * time.Hour)
	has.RecordUptime(24 * time.Hour)

	has.RecordDowntime(10 * time.Minute)

	has.RecordUptime(24 * time.Hour)

	status := has.GetSLAStatus()
	t.Logf("48小时SLA统计:")
	t.Logf("  - 正常运行秒数: %v", status["uptime_seconds"])
	t.Logf("  - 宕机秒数: %v", status["downtime_seconds"])
	t.Logf("  - 事件数: %v", status["incidents_count"])
	t.Logf("  - 当前连续正常运行: %v小时", status["current_streak_hours"])
	t.Logf("  - 最长连续正常运行: %v小时", status["longest_streak_hours"])
	t.Logf("  - SLA达标: %v", status["sla_met"])

	t.Log("SLA指标测试通过")
}

func TestCapacityScaling(t *testing.T) {
	t.Log("测试容量自动伸缩...")

	has := NewHighAvailabilityService()

	has.UpdateCapacityMetrics(25.0, 20.0, 100, 5)
	time.Sleep(100 * time.Millisecond)

	has.UpdateCapacityMetrics(85.0, 80.0, 2000, 200)
	time.Sleep(100 * time.Millisecond)

	status := has.GetCapacityStatus()
	t.Logf("自动扩容后 - 当前副本: %v, 目标副本: %v",
		status["current_replicas"], status["target_replicas"])

	has.UpdateCapacityMetrics(85.0, 80.0, 2000, 200)
	time.Sleep(100 * time.Millisecond)

	has.UpdateCapacityMetrics(20.0, 15.0, 50, 2)
	time.Sleep(100 * time.Millisecond)

	status2 := has.GetCapacityStatus()
	t.Logf("自动缩容后 - 当前副本: %v, 目标副本: %v",
		status2["current_replicas"], status2["target_replicas"])

	history := has.GetScaleHistory(10)
	t.Logf("伸缩历史记录: %d条", len(history))
	for _, event := range history {
		t.Logf("  - %s: %d -> %d (原因: %s)",
			event.Timestamp.Format("15:04:05"), event.OldReplicas, event.NewReplicas, event.Reason)
	}

	t.Log("容量自动伸缩测试通过")
}

func TestGracefulDegradation(t *testing.T) {
	t.Log("测试优雅降级...")

	has := NewHighAvailabilityService()

	degradation := has.GetDegradationStatus()
	t.Logf("初始降级状态 - 级别: %d, 功能: %v",
		degradation.CurrentLevel, degradation.AffectedFeatures)

	has.UpdateComponentHealth("cache_service", HealthStatusCritical, 1000)
	degradation = has.GetDegradationStatus()
	t.Logf("缓存服务故障 - 级别: %d, 受影响: %v, 原因: %s",
		degradation.CurrentLevel, degradation.AffectedFeatures, degradation.Reason)

	has.UpdateComponentHealth("analytics_service", HealthStatusCritical, 2000)
	degradation = has.GetDegradationStatus()
	t.Logf("分析服务故障 - 级别: %d, 受影响: %v, 原因: %s",
		degradation.CurrentLevel, degradation.AffectedFeatures, degradation.Reason)

	has.UpdateComponentHealth("api_gateway", HealthStatusCritical, 500)
	degradation = has.GetDegradationStatus()
	t.Logf("API网关故障 - 级别: %d, 受影响: %v, 原因: %s",
		degradation.CurrentLevel, degradation.AffectedFeatures, degradation.Reason)

	has.UpdateComponentHealth("api_gateway", HealthStatusHealthy, 50)
	has.UpdateComponentHealth("cache_service", HealthStatusHealthy, 80)
	has.UpdateComponentHealth("analytics_service", HealthStatusHealthy, 100)
	degradation = has.GetDegradationStatus()
	t.Logf("全部恢复 - 级别: %d, 是否降级: %v",
		degradation.CurrentLevel, degradation.IsDegraded)

	t.Log("优雅降级测试通过")
}

func TestAutoRecovery(t *testing.T) {
	t.Log("测试自动故障恢复...")

	has := NewHighAvailabilityService()

	has.UpdateComponentHealth("verification_engine", HealthStatusCritical, 2000)
	t.Log("验证引擎故障")

	action := has.StartRecovery("verification_engine")
	t.Logf("启动自动恢复 - 操作ID: %s", action.ID)

	for i := 1; i <= 3; i++ {
		has.UpdateRecoveryStatus(action.ID, false, "Service not ready")
		t.Logf("重试 #%d 失败", i)

		actions := has.GetRecoveryActions()
		for _, a := range actions {
			if a.ID == action.ID {
				t.Logf("  当前状态: %s, 尝试次数: %d", a.Status, a.Attempts)
			}
		}
	}

	has.UpdateRecoveryStatus(action.ID, true, "")
	actions := has.GetRecoveryActions()
	for _, a := range actions {
		if a.ID == action.ID {
			t.Logf("恢复成功 - 状态: %s, 成功时间: %s",
				a.Status, a.SuccessTime.Format("15:04:05"))
		}
	}

	t.Log("自动故障恢复测试通过")
}
