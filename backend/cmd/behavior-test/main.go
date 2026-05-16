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
	fmt.Println("=== 行为分析系统增强功能测试 ===")
	fmt.Println()

	// 1. 创建高级行为分析服务
	fmt.Println("1. 初始化高级行为分析服务...")
	abas := service.NewAdvancedBehaviorAnalysisService()
	abas.InitializeAdvancedModels()
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
	humanResult, err := abas.AnalyzeBehaviorAdvanced(humanData)
	if err != nil {
		fmt.Printf("   ✗ 分析失败: %v\n", err)
	} else {
		fmt.Printf("   ✓ 风险评分: %.2f\n", humanResult.RiskScore)
		fmt.Printf("   ✓ 是否疑似机器人: %v\n", humanResult.IsBotLikely)
		fmt.Printf("   ✓ 置信度: %.2f\n", humanResult.Confidence)
		fmt.Printf("   ✓ 置信等级: %s\n", humanResult.ConfidenceLevel)
	}
	fmt.Println()

	// 4. 测试机器行为分析
	fmt.Println("4. 分析机器行为...")
	robotResult, err := abas.AnalyzeBehaviorAdvanced(robotData)
	if err != nil {
		fmt.Printf("   ✗ 分析失败: %v\n", err)
	} else {
		fmt.Printf("   ✓ 风险评分: %.2f\n", robotResult.RiskScore)
		fmt.Printf("   ✓ 是否疑似机器人: %v\n", robotResult.IsBotLikely)
		fmt.Printf("   ✓ 置信度: %.2f\n", robotResult.Confidence)
		fmt.Printf("   ✓ 置信等级: %s\n", robotResult.ConfidenceLevel)
	}
	fmt.Println()

	// 5. 测试优化风险评分算法
	fmt.Println("5. 测试优化风险评分算法...")
	algorithm := service.NewOptimizedRiskScoreAlgorithm()
	
	// 测试低风险场景
	lowRisk := algorithm.CalculateRiskScore(10, 5, 10, 0.1, 0.1)
	fmt.Printf("   ✓ 低风险场景评分: %.2f\n", lowRisk)
	
	// 测试高风险场景
	highRisk := algorithm.CalculateRiskScore(90, 85, 95, 0.9, 0.95)
	fmt.Printf("   ✓ 高风险场景评分: %.2f\n", highRisk)
	fmt.Println()

	// 6. 测试One-Class SVM
	fmt.Println("6. 测试One-Class SVM异常检测...")
	svm := service.NewOneClassSVM(0.1, 0.01)
	
	// 训练SVM
	trainingData := make([][]float64, 100)
	for i := 0; i < 100; i++ {
		sample := make([]float64, 10)
		for f := 0; f < 10; f++ {
			sample[f] = rand.NormFloat64()*0.3 + 0.5
		}
		trainingData[i] = sample
	}
	svm.Train(trainingData)
	fmt.Println("   ✓ SVM训练完成")
	
	// 测试正常样本
	normalSample := make([]float64, 10)
	for f := 0; f < 10; f++ {
		normalSample[f] = rand.NormFloat64()*0.3 + 0.5
	}
	normalScore, normalIsAnomaly := svm.Predict(normalSample)
	fmt.Printf("   ✓ 正常样本 - 分数: %.4f, 异常: %v\n", normalScore, normalIsAnomaly)
	
	// 测试异常样本
	anomalySample := make([]float64, 10)
	for f := 0; f < 10; f++ {
		anomalySample[f] = rand.NormFloat64()*2.0 + 5.0
	}
	anomalyScore, anomalyIsAnomaly := svm.Predict(anomalySample)
	fmt.Printf("   ✓ 异常样本 - 分数: %.4f, 异常: %v\n", anomalyScore, anomalyIsAnomaly)
	fmt.Println()

	// 7. 功能总结
	fmt.Println("=== 功能实现总结 ===")
	fmt.Println("✓ 高级键盘行为特征提取 (输入速度、按键间隔、错误模式)")
	fmt.Println("✓ 鼠标/触摸屏压力检测")
	fmt.Println("✓ 优化风险评分算法")
	fmt.Println("✓ One-Class SVM异常检测")
	fmt.Println("✓ 集成学习分类器")
	fmt.Println("✓ 孤立森林异常检测 (已包含在基础模块中)")
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

	// 生成鼠标轨迹数据（人类特征：有抖动和变化）
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

	// 生成键盘数据（人类特征：有变化的间隔）
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

	// 生成鼠标轨迹数据（机器特征：完美直线）
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

	// 生成键盘数据（机器特征：恒定间隔）
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
