package main

import (
	"fmt"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"time"
)

func main() {
	fmt.Println("测试优化后的行为分析引擎...")

	svc := service.NewBehaviorAnalysisService()

	testPoints := []service.BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "mousemove"},
		{X: 110, Y: 110, Timestamp: 1100, Event: "mousemove"},
		{X: 120, Y: 120, Timestamp: 1200, Event: "mousemove"},
		{X: 130, Y: 130, Timestamp: 1300, Event: "mousemove"},
		{X: 140, Y: 140, Timestamp: 1400, Event: "mousemove"},
		{X: 150, Y: 150, Timestamp: 1500, Event: "mousemove"},
		{X: 160, Y: 160, Timestamp: 1600, Event: "mousemove"},
	}

	fmt.Println("\n1. 测试自适应平滑轨迹算法...")
	adaptiveSmoothed := svc.AdaptiveSmoothTrajectory(testPoints)
	fmt.Printf("自适应平滑后点数: %d (原始: %d)\n", len(adaptiveSmoothed), len(testPoints))

	fmt.Println("\n2. 测试Savitzky-Golay平滑算法...")
	sgSmoothed := svc.SavitzkyGolaySmooth(testPoints, 3, 2)
	fmt.Printf("SG平滑后点数: %d\n", len(sgSmoothed))

	fmt.Println("\n3. 测试速度分析...")
	speedAnalysis, _ := svc.AnalyzeSpeed(createTestBehaviorDataArray(testPoints))
	fmt.Printf("平均速度: %.4f\n", speedAnalysis.AverageSpeed)
	fmt.Printf("最大速度: %.4f\n", speedAnalysis.MaxSpeed)
	fmt.Printf("速度方差: %.4f\n", speedAnalysis.SpeedVariance)
	fmt.Printf("速度标准差: %.4f\n", speedAnalysis.SpeedStdDev)

	fmt.Println("\n4. 测试曲率统计...")
	curvMean, curvStdDev, curvMax := svc.ComputeCurvatureStatistics(testPoints)
	fmt.Printf("平均曲率: %.6f\n", curvMean)
	fmt.Printf("曲率标准差: %.6f\n", curvStdDev)
	fmt.Printf("最大曲率: %.6f\n", curvMax)

	fmt.Println("\n5. 测试轨迹平滑度指标...")
	smoothnessMetrics := svc.ComputeTrajectorySmoothnessMetrics(testPoints)
	fmt.Printf("平均角度变化: %.6f\n", smoothnessMetrics["avg_angle_change"])
	fmt.Printf("平滑度分数: %.6f\n", smoothnessMetrics["smoothness_score"])
	fmt.Printf("锐利转折比例: %.6f\n", smoothnessMetrics["sharp_turn_ratio"])

	fmt.Println("\n6. 测试加速度异常检测...")
	accelerations := []float64{0.1, 0.2, 0.15, 0.25, 0.2, 0.18, 0.22}
	accelAnomalies := svc.DetectAccelerationAnomalies(testPoints, accelerations)
	fmt.Printf("异常数量: %d\n", accelAnomalies["anomaly_count"])
	fmt.Printf("是否有异常: %v\n", accelAnomalies["has_anomaly"])

	fmt.Println("\n7. 测试加速度模式分析...")
	accelPattern := svc.AnalyzeAccelerationPattern(testPoints)
	fmt.Printf("平均加速度: %.6f\n", accelPattern["mean_acceleration"])
	fmt.Printf("加速度方差: %.6f\n", accelPattern["acceleration_variance"])
	fmt.Printf("振荡次数: %.0f\n", accelPattern["acceleration_oscillation_count"])

	fmt.Println("\n8. 测试速度熵...")
	speeds := []float64{0.5, 0.6, 0.55, 0.65, 0.58, 0.62, 0.59}
	speedEntropy := svc.CalculateSpeedEntropy(speeds)
	fmt.Printf("速度熵: %.6f\n", speedEntropy)

	fmt.Println("\n9. 测试速度突发性...")
	speedBurstiness := svc.CalculateSpeedBurstiness(speeds)
	fmt.Printf("速度突发性: %.6f\n", speedBurstiness)

	fmt.Println("\n10. 测试点击节奏分析...")
	clicks := []service.BehaviorDataPoint{
		{X: 100, Y: 100, Timestamp: 1000, Event: "click"},
		{X: 150, Y: 150, Timestamp: 1200, Event: "click"},
		{X: 200, Y: 200, Timestamp: 1400, Event: "click"},
		{X: 250, Y: 250, Timestamp: 1600, Event: "click"},
	}
	clickRhythm := svc.AnalyzeClickRhythmAdvanced(clicks, testPoints)
	fmt.Printf("点击间隔变异系数: %.6f\n", clickRhythm.ClickIntervalCV)
	fmt.Printf("点击突发性: %.6f\n", clickRhythm.ClickBurstiness)
	fmt.Printf("点击节奏一致性: %.6f\n", clickRhythm.ClickRhythmConsistency)
	fmt.Printf("点击时间模式: %s\n", clickRhythm.ClickTimingPattern)

	fmt.Println("\n所有测试完成!")
}

func createTestBehaviorDataArray(points []service.BehaviorDataPoint) []models.BehaviorData {
	result := make([]models.BehaviorData, len(points))
	for i, p := range points {
		result[i] = models.BehaviorData{
			Data:      fmt.Sprintf(`{"x":%d,"y":%d,"timestamp":%d,"event":"%s"}`, p.X, p.Y, p.Timestamp, p.Event),
			DataType:  p.Event,
			Timestamp: time.Now(),
		}
	}
	return result
}
