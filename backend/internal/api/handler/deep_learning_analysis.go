package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/model"
	github.com/hjtpx/hjtpx/internal/service"
	github.com/hjtpx/hjtpx/internal/service/trace"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	traceServiceInstance   *trace.TraceService
	sliderAnalyzerInstance *service.SliderAnalyzer
	multiFactorVerifier    *service.MultiFactorVerifier
	difficultyAdjuster     *service.SlidingDifficultyAdjuster
	securityAssessor       *service.SliderSecurityAssessor
)

func initDeepLearningServices() {
	if traceServiceInstance == nil {
		traceServiceInstance = trace.NewTraceService()
	}
	if sliderAnalyzerInstance == nil {
		sliderAnalyzerInstance = service.NewSliderAnalyzer()
	}
	if multiFactorVerifier == nil {
		multiFactorVerifier = service.NewMultiFactorVerifier()
	}
	if difficultyAdjuster == nil {
		difficultyAdjuster = service.NewSlidingDifficultyAdjuster()
	}
	if securityAssessor == nil {
		securityAssessor = service.NewSliderSecurityAssessor()
	}
}

// GetModelPerformanceReport - 获取模型性能报告
func GetModelPerformanceReport(c *gin.Context) {
	initDeepLearningServices()
	report := traceServiceInstance.GetModelPerformanceReport()
	response.Success(c, report)
}

// QueueModelUpdate - 排队更新模型
func QueueModelUpdate(c *gin.Context) {
	initDeepLearningServices()

	var req struct {
		IsBot     bool              `json:"isBot"`
		TraceData model.TraceData   `json:"traceData"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求格式")
		return
	}

	traceServiceInstance.QueueTrainingSample(&req.TraceData, req.IsBot, 0.8)
	response.Success(c, gin.H{"message": "训练样本已加入队列"})
}

// ToggleOnlineUpdate - 开启/关闭在线更新
func ToggleOnlineUpdate(c *gin.Context) {
	initDeepLearningServices()

	action := c.Param("action")
	if action == "start" {
		traceServiceInstance.StartOnlineUpdate()
		response.Success(c, gin.H{"message": "在线更新已启动"})
	} else if action == "stop" {
		traceServiceInstance.StopOnlineUpdate()
		response.Success(c, gin.H{"message": "在线更新已停止"})
	} else {
		response.BadRequest(c, "无效的操作")
	}
}

// GetTrajectoryVisualization - 获取轨迹可视化数据
func GetTrajectoryVisualization(c *gin.Context) {
	initDeepLearningServices()

	var traceData model.TraceData
	if err := c.ShouldBindJSON(&traceData); err != nil {
		traceDataJSON := c.Query("traceData")
		if traceDataJSON != "" {
			if err := json.Unmarshal([]byte(traceDataJSON), &traceData); err != nil {
				response.BadRequest(c, "无效的轨迹数据格式")
				return
			}
		} else {
			response.BadRequest(c, "轨迹数据为必需项")
			return
		}
	}

	if len(traceData.Points) < 2 {
		response.BadRequest(c, "轨迹数据点不足")
		return
	}

	vizData, err := traceServiceInstance.PrepareVisualizationData(&traceData)
	if err != nil {
		response.InternalServerError(c, "准备可视化数据失败")
		return
	}

	response.Success(c, vizData)
}

// VerifySliderWithMultiFactor - 多因素验证滑块
func VerifySliderWithMultiFactor(c *gin.Context) {
	initDeepLearningServices()

	var req struct {
		Trajectory []service.SliderPoint `json:"trajectory"`
		TargetPos  int                 `json:"targetPos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求格式")
		return
	}

	if len(req.Trajectory) < 3 {
		response.BadRequest(c, "轨迹数据点不足")
		return
	}

	analysisResult, err := sliderAnalyzerInstance.AnalyzeSliderTrajectory(req.Trajectory, req.TargetPos)
	if err != nil {
		response.InternalServerError(c, "分析轨迹失败")
		return
	}

	overallScore, factorScores, err := multiFactorVerifier.VerifyMultiFactor(
		req.Trajectory,
		analysisResult.Features,
		analysisResult,
	)
	if err != nil {
		response.InternalServerError(c, "多因素验证失败")
		return
	}

	isHuman := overallScore > 0.5

	response.Success(c, gin.H{
		"success":        isHuman,
		"overallScore":   overallScore,
		"factorScores":   factorScores,
		"analysisResult": analysisResult,
	})
}

// GetCurrentDifficulty - 获取当前难度
func GetCurrentDifficulty(c *gin.Context) {
	initDeepLearningServices()
	config := difficultyAdjuster.GetDifficultyConfig()
	response.Success(c, config)
}

// AdjustDifficulty - 调整难度
func AdjustDifficulty(c *gin.Context) {
	initDeepLearningServices()

	var req struct {
		Success     bool  `json:"success"`
		TimeSpentMs int64 `json:"timeSpentMs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求格式")
		return
	}

	newDifficulty := difficultyAdjuster.AdjustDifficulty(req.Success, req.TimeSpentMs)
	response.Success(c, gin.H{
		"newDifficulty": newDifficulty,
		"config":        difficultyAdjuster.GetDifficultyConfig(),
	})
}

// GetSecurityReport - 获取安全报告
func GetSecurityReport(c *gin.Context) {
	initDeepLearningServices()
	report := securityAssessor.GetSecurityReport()
	response.Success(c, report)
}

// PerformSecurityAssessment - 执行安全评估
func PerformSecurityAssessment(c *gin.Context) {
	initDeepLearningServices()

	var req struct {
		Trajectory []service.SliderPoint `json:"trajectory"`
		TargetPos  int                 `json:"targetPos"`
		Stats      map[string]interface{} `json:"stats,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求格式")
		return
	}

	var analysisResult *service.SliderAnalysisResult
	if len(req.Trajectory) >= 3 {
		var err error
		analysisResult, err = sliderAnalyzerInstance.AnalyzeSliderTrajectory(req.Trajectory, req.TargetPos)
		if err != nil {
			response.InternalServerError(c, "分析轨迹失败")
			return
		}
	}

	assessment, err := securityAssessor.AssessSecurity(analysisResult, req.Stats)
	if err != nil {
		response.InternalServerError(c, "执行安全评估失败")
		return
	}

	response.Success(c, assessment)
}

// RecordModelPrediction - 记录模型预测
func RecordModelPrediction(c *gin.Context) {
	initDeepLearningServices()

	var req struct {
		ModelType   string        `json:"modelType"`
		Prediction  bool          `json:"prediction"`
		Actual      bool          `json:"actual"`
		ResponseTime time.Duration `json:"responseTimeMs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求格式")
		return
	}

	traceServiceInstance.RecordPrediction(
		req.ModelType,
		req.Prediction,
		req.Actual,
		req.ResponseTime*time.Millisecond,
	)

	response.Success(c, gin.H{"message": "预测记录已保存"})
}
