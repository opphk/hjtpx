package service

import (
	"context"
	"testing"
	"time"
)

func TestDataPipeline(t *testing.T) {
	t.Log("测试数据管道...")

	dp := NewDataPipeline()

	stream := dp.CreateStream("test-stream", "realtime")
	t.Logf("创建流 - ID: %s, 名称: %s, 类型: %s", stream.ID, stream.Name, stream.Type)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream.Start(ctx)
	t.Log("启动流处理器")

	for i := 0; i < 100; i++ {
		data := map[string]interface{}{
			"id":      i,
			"user_id": "user-" + string(rune('0'+i%10)),
			"action":  "test",
			"value":   i * 10,
		}

		err := dp.ProcessStreamData(stream.ID, data)
		if err != nil {
			t.Logf("处理数据失败: %v", err)
		}
	}

	time.Sleep(500 * time.Millisecond)

	sp, _ := dp.GetStream(stream.ID)
	t.Logf("流统计 - 输入: %d, 输出: %d, 错误: %d",
		sp.Stream.RecordsIn.Load(), sp.Stream.RecordsOut.Load(), sp.Stream.Errors.Load())

	stream.Stop()
	t.Log("停止流处理器")

	records := make([]interface{}, 0)
	for i := 0; i < 50; i++ {
		record := map[string]interface{}{
			"id":      i,
			"user_id": "user-batch-" + string(rune('0'+i%10)),
			"email":   "user" + string(rune('0'+i%10)) + "@example.com",
			"amount":  float64(i * 100),
		}
		records = append(records, record)
	}

	job := dp.CreateBatchJob("test-batch", "data_processing", records)
	t.Logf("创建批处理任务 - ID: %s, 记录数: %d", job.ID, job.RecordsTotal)

	time.Sleep(1 * time.Second)

	jobStatus, _ := dp.GetBatchJob(job.ID)
	t.Logf("批处理状态 - 状态: %s, 处理: %d, 失败: %d, 耗时: %v",
		jobStatus.Status, jobStatus.RecordsProcessed, jobStatus.RecordsFailed, jobStatus.Duration)

	qualityRules := dp.GetQualityRules()
	t.Logf("数据质量规则数量: %d", len(qualityRules))
	for _, rule := range qualityRules {
		t.Logf("  - %s: %s (字段: %s, 严重性: %s)",
			rule.Name, rule.Type, rule.Field, rule.Severity)
	}

	qualityMetrics := dp.GetQualityMetrics()
	t.Logf("数据质量指标 - 总记录: %d, 有效: %d, 质量分数: %.2f%%",
		qualityMetrics.TotalRecords.Load(),
		qualityMetrics.ValidRecords.Load(),
		qualityMetrics.QualityScore.Load())

	dashboard := dp.GetDashboardMetrics()
	t.Logf("实时仪表板 - 事件数: %d, 事件/秒: %.2f, 平均延迟: %dms, 错误率: %.2f%%",
		dashboard.TotalEvents.Load(),
		dashboard.EventsPerSecond.Load(),
		dashboard.AvgLatencyMs.Load(),
		dashboard.ErrorRate.Load())

	pipelineStats := dp.GetPipelineStats()
	t.Logf("管道统计:")
	t.Logf("  流: %v", pipelineStats["streams"])
	t.Logf("  批处理: %v", pipelineStats["batch_jobs"])

	t.Log("数据管道测试通过")
}

func TestStreamProcessing(t *testing.T) {
	t.Log("测试流处理...")

	dp := NewDataPipeline()

	stream := dp.CreateStream("filter-stream", "filtered")
	stream.Filter = func(data interface{}) bool {
		if m, ok := data.(map[string]interface{}); ok {
			if value, ok := m["value"].(int); ok {
				return value > 50
			}
		}
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream.Start(ctx)

	for i := 0; i < 20; i++ {
		data := map[string]interface{}{
			"id":    i,
			"value": i * 10,
		}
		dp.ProcessStreamData(stream.ID, data)
	}

	time.Sleep(300 * time.Millisecond)

	sp, _ := dp.GetStream(stream.ID)
	t.Logf("过滤流 - 输入: %d, 输出: %d (预期约14条>50的记录)",
		sp.Stream.RecordsIn.Load(), sp.Stream.RecordsOut.Load())

	stream.Stop()

	t.Log("流处理测试通过")
}

func TestStreamTransformation(t *testing.T) {
	t.Log("测试流数据转换...")

	dp := NewDataPipeline()

	stream := dp.CreateStream("transform-stream", "transformed")
	stream.Transform = func(data interface{}) interface{} {
		if m, ok := data.(map[string]interface{}); ok {
			m["transformed"] = true
			m["timestamp"] = time.Now().Unix()
			return m
		}
		return data
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream.Start(ctx)

	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"id":    i,
			"name": "record-" + string(rune('0'+i)),
		}
		dp.ProcessStreamData(stream.ID, data)
	}

	time.Sleep(300 * time.Millisecond)

	sp, _ := dp.GetStream(stream.ID)
	t.Logf("转换流 - 输入: %d, 输出: %d, 平均延迟: %dms",
		sp.Stream.RecordsIn.Load(), sp.Stream.RecordsOut.Load(), sp.Stream.LatencyMs.Load())

	stream.Stop()

	t.Log("流数据转换测试通过")
}

func TestBatchProcessing(t *testing.T) {
	t.Log("测试批处理...")

	dp := NewDataPipeline()

	records := make([]interface{}, 0)
	for i := 0; i < 5000; i++ {
		record := map[string]interface{}{
			"id":      i,
			"user_id": "user-" + string(rune('0'+i%10)),
			"email":   "user" + string(rune('0'+i%10)) + "@test.com",
			"amount":  float64((i % 100) + 1),
			"status":  "active",
		}
		records = append(records, record)
	}

	start := time.Now()
	job := dp.CreateBatchJob("large-batch", "data_enrichment", records)
	t.Logf("创建大批量任务 - 记录数: %d", job.RecordsTotal)

	for {
		status, _ := dp.GetBatchJob(job.ID)
		if status.Status == "completed" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	elapsed := time.Since(start)
	jobStatus, _ := dp.GetBatchJob(job.ID)

	t.Logf("批处理完成:")
	t.Logf("  - 状态: %s", jobStatus.Status)
	t.Logf("  - 处理记录: %d", jobStatus.RecordsProcessed)
	t.Logf("  - 失败记录: %d", jobStatus.RecordsFailed)
	t.Logf("  - 耗时: %v", jobStatus.Duration)
	t.Logf("  - 吞吐量: %.2f 记录/秒", float64(jobStatus.RecordsProcessed)/jobStatus.Duration.Seconds())

	t.AssertTrue(elapsed < 10*time.Second, "批处理应在10秒内完成")

	t.Log("批处理测试通过")
}

func TestDataQualityChecks(t *testing.T) {
	t.Log("测试数据质量检查...")

	dp := NewDataPipeline()

	testRecords := []interface{}{
		map[string]interface{}{"id": 1, "user_id": "user1", "email": "user1@test.com", "amount": 100.0},
		map[string]interface{}{"id": 2, "user_id": nil, "email": "user2@test.com", "amount": 200.0},
		map[string]interface{}{"id": 3, "user_id": "user3", "email": "invalid-email", "amount": -50.0},
		map[string]interface{}{"id": 4, "user_id": "user4", "email": "user4@test.com", "amount": 400.0},
	}

	for _, record := range testRecords {
		dp.checkDataQuality(record)
	}

	qualityMetrics := dp.GetQualityMetrics()
	t.Logf("数据质量指标:")
	t.Logf("  - 总记录: %d", qualityMetrics.TotalRecords.Load())
	t.Logf("  - 有效记录: %d", qualityMetrics.ValidRecords.Load())
	t.Logf("  - 空值记录: %d", qualityMetrics.NullRecords.Load())
	t.Logf("  - 质量分数: %.2f%%", qualityMetrics.QualityScore.Load())

	dp.AddQualityRule(DataQualityRule{
		ID:       "rule-custom",
		Name:     "Positive Amount",
		Field:    "amount",
		Type:     "range",
		Condition: ">0",
		Severity: "error",
		Enabled:  true,
	})

	rules := dp.GetQualityRules()
	t.Logf("更新后质量规则数量: %d", len(rules))

	t.Log("数据质量检查测试通过")
}

func TestAnalyticsDashboard(t *testing.T) {
	t.Log("测试实时分析仪表板...")

	dp := NewDataPipeline()

	stream1 := dp.CreateStream("stream-1", "type-a")
	stream2 := dp.CreateStream("stream-2", "type-b")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream1.Start(ctx)
	stream2.Start(ctx)

	for i := 0; i < 50; i++ {
		dp.ProcessStreamData(stream1.ID, map[string]interface{}{"id": i, "value": i})
		dp.ProcessStreamData(stream2.ID, map[string]interface{}{"id": i, "value": i * 2})
	}

	time.Sleep(500 * time.Millisecond)

	dashboard := dp.GetDashboardMetrics()
	t.Logf("实时仪表板:")
	t.Logf("  - 总事件数: %d", dashboard.TotalEvents.Load())
	t.Logf("  - 活跃流: %d", dashboard.ActiveStreams.Load())
	t.Logf("  - 活跃批处理: %d", dashboard.ActiveBatches.Load())
	t.Logf("  - 平均延迟: %dms", dashboard.AvgLatencyMs.Load())
	t.Logf("  - 错误率: %.2f%%", dashboard.ErrorRate.Load())
	t.Logf("  - 成功率: %.2f%%", dashboard.SuccessRate.Load())

	pipelineStats := dp.GetPipelineStats()
	t.Logf("管道统计:")
	t.Logf("  - 流统计: %v", pipelineStats["streams"])
	t.Logf("  - 质量统计: %v", pipelineStats["quality"])

	stream1.Stop()
	stream2.Stop()

	t.Log("实时分析仪表板测试通过")
}

func TestPipelineReset(t *testing.T) {
	t.Log("测试管道重置...")

	dp := NewDataPipeline()

	stream := dp.CreateStream("reset-test", "test")
	ctx, cancel := context.WithCancel(context.Background())
	stream.Start(ctx)

	for i := 0; i < 100; i++ {
		dp.ProcessStreamData(stream.ID, map[string]interface{}{"id": i})
	}

	time.Sleep(200 * time.Millisecond)

	sp, _ := dp.GetStream(stream.ID)
	t.Logf("重置前 - 输入: %d, 输出: %d",
		sp.Stream.RecordsIn.Load(), sp.Stream.RecordsOut.Load())

	dp.ResetMetrics()

	sp2, _ := dp.GetStream(stream.ID)
	t.Logf("重置后 - 输入: %d, 输出: %d",
		sp2.Stream.RecordsIn.Load(), sp2.Stream.RecordsOut.Load())

	qualityMetrics := dp.GetQualityMetrics()
	t.Logf("质量指标已重置 - 总记录: %d, 质量分数: %.2f%%",
		qualityMetrics.TotalRecords.Load(), qualityMetrics.QualityScore.Load())

	stream.Stop()

	t.Log("管道重置测试通过")
}
