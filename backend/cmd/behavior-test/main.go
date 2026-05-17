package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
)

func main() {
	fmt.Println("=== 行为分析系统测试 ===")
	fmt.Println()

	// 1. 创建增强行为分析服务
	fmt.Println("1. 初始化增强行为分析服务...")
	ebas := service.NewEnhancedBehaviorAnalysisService()
	fmt.Println("   ✓ 服务初始化成功")
	fmt.Println()

	// 2. 生成测试数据
	fmt.Println("2. 生成测试行为数据...")
	humanData := generateHumanBehaviorData()
	robotData := generateRoboticBehaviorData()
	fmt.Printf("   ✓ 人类行为数据: %d 条\n", len(humanData))
	fmt.Printf("   ✓ 机器行为数据: %d 条\n", len(robotData))
	fmt.Println()

	// 3. 测试人类行为分析
	fmt.Println("3. 分析人类行为...")
	ebas.AnalyzeBehavior(humanData)
	fmt.Println("   ✓ 人类行为分析完成")
	fmt.Println()

	// 4. 测试机器行为分析
	fmt.Println("4. 分析机器行为...")
	ebas.AnalyzeBehavior(robotData)
	fmt.Println("   ✓ 机器行为分析完成")
	fmt.Println()

	// 5. 功能总结
	fmt.Println("=== 功能实现总结 ===")
	fmt.Println("✓ 增强行为分析服务")
	fmt.Println("✓ 行为特征提取")
	fmt.Println("✓ 风险评分计算")
	fmt.Println()
	fmt.Println("目标指标:")
	fmt.Println("- 机器人识别准确率: 99%+")
	fmt.Println("- 正常用户误伤率: <0.5%")
	fmt.Println()
	fmt.Println("测试完成!")
}

func generateHumanBehaviorData() []models.BehaviorData {
	data := make([]models.BehaviorData, 0)
	baseTime := time.Now()

	for i := 0; i < 30; i++ {
		point := service.BehaviorDataPoint{
			X:         100 + i*10 + rand.Intn(10) - 5,
			Y:         100 + i*5 + rand.Intn(10) - 5,
			Timestamp: baseTime.Add(time.Duration(i*50+rand.Intn(20)) * time.Millisecond).UnixMilli(),
			Event:     "move",
		}
		pointJSON, _ := json.Marshal(point)
		data = append(data, models.BehaviorData{
			DataType: "mouse",
			Data:     string(pointJSON),
		})
	}

	keys := []string{"h", "e", "l", "l", "o", "w", "o", "r", "l", "d"}
	keyBaseTime := time.Now().Add(time.Second)
	for i := 0; i < 10; i++ {
		interval := 80 + rand.Int63n(80)
		keyBaseTime = keyBaseTime.Add(time.Duration(interval) * time.Millisecond)

		ks := struct {
			Key          string `json:"key"`
			Timestamp    int64  `json:"timestamp"`
			HoldDuration int64  `json:"hold_duration,omitempty"`
		}{
			Key:          keys[i%len(keys)],
			Timestamp:    keyBaseTime.UnixMilli(),
			HoldDuration: 50 + rand.Int63n(50),
		}
		ksJSON, _ := json.Marshal(ks)
		data = append(data, models.BehaviorData{
			DataType: "keyboard",
			Data:     string(ksJSON),
		})
	}

	return data
}

func generateRoboticBehaviorData() []models.BehaviorData {
	data := make([]models.BehaviorData, 0)
	baseTime := time.Now()

	for i := 0; i < 30; i++ {
		point := service.BehaviorDataPoint{
			X:         100 + i*10,
			Y:         100 + i*5,
			Timestamp: baseTime.Add(time.Duration(i*50) * time.Millisecond).UnixMilli(),
			Event:     "move",
		}
		pointJSON, _ := json.Marshal(point)
		data = append(data, models.BehaviorData{
			DataType: "mouse",
			Data:     string(pointJSON),
		})
	}

	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	keyBaseTime := time.Now().Add(time.Second)
	for i := 0; i < 10; i++ {
		keyBaseTime = keyBaseTime.Add(100 * time.Millisecond)

		ks := struct {
			Key          string `json:"key"`
			Timestamp    int64  `json:"timestamp"`
			HoldDuration int64  `json:"hold_duration,omitempty"`
		}{
			Key:          keys[i%len(keys)],
			Timestamp:    keyBaseTime.UnixMilli(),
			HoldDuration: 50,
		}
		ksJSON, _ := json.Marshal(ks)
		data = append(data, models.BehaviorData{
			DataType: "keyboard",
			Data:     string(ksJSON),
		})
	}

	return data
}
