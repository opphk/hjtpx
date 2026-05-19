package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// ComboCaptchaGenerateRequest 组合验证码生成请求
type ComboCaptchaGenerateRequest struct {
	UserID     string   `json:"user_id"`
	RiskLevel  int      `json:"risk_level" binding:"omitempty,min=1,max=5"`
	StepCount  int      `json:"step_count" binding:"omitempty,min=1,max=5"`
	CaptchaTypes []string `json:"captcha_types"` // 指定验证码类型顺序
}

// ComboCaptchaVerifyRequest 组合验证码验证请求
type ComboCaptchaVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	StepResults []StepVerifyResult `json:"step_results" binding:"required"`
	BehaviorData map[string]interface{} `json:"behavior_data"`
}

// StepVerifyResult 单个验证步骤的结果
type StepVerifyResult struct {
	StepIndex int                    `json:"step_index" binding:"required"`
	Result    map[string]interface{} `json:"result" binding:"required"`
}

// ComboCaptchaGenerate 生成组合验证码
// @Summary 生成组合验证码
// @Description 根据风险评估生成多步骤组合验证码
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body ComboCaptchaGenerateRequest true "生成请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/combo/generate [post]
func ComboCaptchaGenerate(c *gin.Context) {
	var req ComboCaptchaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 设置默认值
	if req.RiskLevel == 0 {
		req.RiskLevel = 2
	}
	if req.StepCount == 0 {
		req.StepCount = 2 // 默认两步验证
	}
	if req.StepCount > 5 {
		req.StepCount = 5
	}

	captchaID := uuid.New().String()
	steps, err := generateComboSteps(req.RiskLevel, req.StepCount, req.CaptchaTypes)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成失败")
		return
	}

	response.Success(c, gin.H{
		"captcha_id": captchaID,
		"steps": steps,
		"risk_level": req.RiskLevel,
		"total_steps": len(steps),
		"message": "请完成所有验证步骤",
	})
}

// ComboCaptchaVerify 验证组合验证码
// @Summary 验证组合验证码
// @Description 验证组合验证码的所有步骤
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body ComboCaptchaVerifyRequest true "验证请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/combo/verify [post]
func ComboCaptchaVerify(c *gin.Context) {
	var req ComboCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	isValid, score, stepResults := verifyComboSteps(req.CaptchaID, req.StepResults, req.BehaviorData)

	response.Success(c, gin.H{
		"success": isValid,
		"score": score,
		"step_results": stepResults,
		"message": func() string {
			if isValid {
				return "所有验证步骤通过"
			}
			return "验证失败"
		}(),
	})
}

// generateComboSteps 生成组合验证步骤
func generateComboSteps(riskLevel, stepCount int, preferredTypes []string) ([]map[string]interface{}, error) {
	var steps []map[string]interface{}

	// 根据风险等级选择合适的验证码组合
	var selectedTypes []string
	if len(preferredTypes) > 0 {
		selectedTypes = preferredTypes
		// 确保不超过stepCount
		if len(selectedTypes) > stepCount {
			selectedTypes = selectedTypes[:stepCount]
		}
	} else {
		selectedTypes = selectComboByRiskLevel(riskLevel, stepCount)
	}

	// 生成每个步骤的配置
	for i, captchaType := range selectedTypes {
		difficulty := calculateStepDifficulty(riskLevel, i, len(selectedTypes))
		steps = append(steps, map[string]interface{}{
			"index": i,
			"type": captchaType,
			"difficulty": difficulty,
			"name": getCaptchaTypeName(captchaType),
			"hint": getCaptchaHint(captchaType, difficulty),
		})
	}

	return steps, nil
}

// getAvailableCaptchaTypes 获取所有可用的验证码类型
func getAvailableCaptchaTypes() []string {
	return []string{
		"slider", "click", "image", "rotate", "gesture", 
		"puzzle", "voice", "lianliankan", "3d", "emoji", "semantic",
		"video", "ar",
	}
}

// selectComboByRiskLevel 根据风险等级选择验证码组合
func selectComboByRiskLevel(riskLevel, stepCount int) []string {
	var selected []string
	
	switch riskLevel {
	case 1: // 低风险：简单组合
		selected = []string{"image", "slider"}
	case 2: // 中低风险
		selected = []string{"slider", "click"}
	case 3: // 中等风险
		selected = []string{"slider", "gesture", "puzzle"}
	case 4: // 中高风险
		selected = []string{"click", "puzzle", "rotate", "lianliankan"}
	case 5: // 高风险
		selected = []string{"semantic", "emoji", "3d", "video", "ar"}
	}

	// 调整数量
	if len(selected) > stepCount {
		selected = selected[:stepCount]
	} else if len(selected) < stepCount {
		// 如果不够，补充重复
		for len(selected) < stepCount {
			selected = append(selected, selected[len(selected)%len(selected)])
		}
	}
	
	return selected[:stepCount]
}

// calculateStepDifficulty 计算步骤难度
func calculateStepDifficulty(riskLevel, stepIndex, totalSteps int) int {
	// 难度递进：后续步骤难度更高
	baseDifficulty := 1
	if riskLevel >= 3 {
		baseDifficulty = 2
	}
	if riskLevel >= 5 {
		baseDifficulty = 3
	}
	
	stepBonus := stepIndex / 2 // 每两步增加一点难度
	difficulty := baseDifficulty + stepBonus
	
	if difficulty > 3 {
		difficulty = 3
	}
	return difficulty
}

// getCaptchaTypeName 获取验证码类型名称
func getCaptchaTypeName(captchaType string) string {
	nameMap := map[string]string{
		"slider": "滑块验证",
		"click": "点选验证",
		"image": "图形验证",
		"rotate": "旋转验证",
		"gesture": "手势验证",
		"puzzle": "拼图验证",
		"voice": "语音验证",
		"lianliankan": "连连看验证",
		"3d": "3D验证",
		"emoji": "表情验证",
		"semantic": "语义验证",
		"video": "视频验证",
		"ar": "AR验证",
	}
	if name, ok := nameMap[captchaType]; ok {
		return name
	}
	return captchaType
}

// getCaptchaHint 获取验证码提示
func getCaptchaHint(captchaType string, difficulty int) string {
	hintMap := map[string][]string{
		"slider": {"拖动滑块完成拼图", "拖动滑块到正确位置", "精确拖动滑块到目标位置"},
		"click": {"点击指定的目标", "按顺序点击目标", "识别并点击特定目标"},
		"image": {"输入图形验证码", "识别并输入验证码", "准确输入验证码"},
		"rotate": {"旋转图片到正确方向", "转动图片到正确角度", "精确调整图片角度"},
		"gesture": {"绘制指定手势", "画出正确的手势图案", "精准绘制手势"},
		"puzzle": {"完成拼图游戏", "拼凑完整图片", "快速完成拼图"},
		"voice": {"听语音输入验证码", "输入听到的内容", "准确识别语音内容"},
		"lianliankan": {"完成连连看游戏", "连接相同的图案", "快速配对所有图案"},
		"3d": {"旋转3D物体到指定视角", "调整物体到正确角度", "精确定位3D物体"},
		"emoji": {"选择指定的表情", "按要求选择表情", "识别并选择正确的表情"},
		"semantic": {"理解并选择正确答案", "根据问题选择答案", "准确理解语义"},
		"video": {"观察视频回答问题", "注意视频细节并回答", "分析视频内容回答问题"},
		"ar": {"完成AR交互操作", "进行指定的AR动作", "精准完成AR任务"},
	}
	
	if hints, ok := hintMap[captchaType]; ok {
		index := difficulty - 1
		if index >= len(hints) {
			index = len(hints) - 1
		}
		return hints[index]
	}
	return "请完成验证"
}

// verifyComboSteps 验证组合验证码步骤
func verifyComboSteps(captchaID string, stepResults []StepVerifyResult, behaviorData map[string]interface{}) (bool, float64, []map[string]interface{}) {
	if len(stepResults) == 0 {
		return false, 0, nil
	}

	totalScore := 0.0
	stepDetailResults := []map[string]interface{}{}
	allPassed := true

	for _, stepResult := range stepResults {
		// 验证每个步骤的结果
		stepPassed, stepScore := verifySingleStep(stepResult)
		
		stepDetail := map[string]interface{}{
			"index": stepResult.StepIndex,
			"passed": stepPassed,
			"score": stepScore,
			"timestamp": time.Now().Unix(),
		}
		stepDetailResults = append(stepDetailResults, stepDetail)

		totalScore += stepScore
		if !stepPassed {
			allPassed = false
		}
	}

	// 计算最终得分
	finalScore := totalScore / float64(len(stepResults))
	
	// 加入行为数据得分
	if behaviorData != nil {
		behaviorScore := calculateBehaviorScore(behaviorData)
		finalScore = (finalScore * 0.7) + (behaviorScore * 0.3)
	}

	// 综合判定：所有步骤通过且最终得分>=0.7
	isValid := allPassed && finalScore >= 0.7

	return isValid, finalScore, stepDetailResults
}

// verifySingleStep 验证单个步骤（模拟实现）
func verifySingleStep(step StepVerifyResult) (bool, float64) {
	// 简单验证：检查是否有result
	if step.Result == nil {
		return false, 0
	}
	
	// 模拟步骤验证，实际项目应该调用对应的验证码验证函数
	passed := true
	score := 0.85
	
	// 检查是否有错误
	if hasError, ok := step.Result["has_error"].(bool); ok && hasError {
		passed = false
		score = 0.3
	}
	
	return passed, score
}

// calculateBehaviorScore 计算行为数据得分
func calculateBehaviorScore(behaviorData map[string]interface{}) float64 {
	score := 0.5 // 基础分
	
	// 检查鼠标移动数据
	if mouseMoves, ok := behaviorData["mouse_moves"].([]interface{}); ok && len(mouseMoves) > 5 {
		score += 0.15
	}
	
	// 检查时间间隔
	if timeSpent, ok := behaviorData["total_time_spent"].(float64); ok {
		if timeSpent >= 2.0 && timeSpent <= 30.0 {
			score += 0.2
		}
	}
	
	// 检查输入速度
	if keystrokes, ok := behaviorData["keystrokes"].([]interface{}); ok && len(keystrokes) > 2 {
		score += 0.15
	}
	
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// ComboCaptchaOptions 获取组合验证码选项
// @Summary 获取组合验证码选项
// @Description 获取组合验证码的配置选项
// @Tags captcha
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/combo/options [get]
func ComboCaptchaOptions(c *gin.Context) {
	response.Success(c, gin.H{
		"available_types": []gin.H{
			{"value": "slider", "label": "滑块验证"},
			{"value": "click", "label": "点选验证"},
			{"value": "image", "label": "图形验证"},
			{"value": "rotate", "label": "旋转验证"},
			{"value": "gesture", "label": "手势验证"},
			{"value": "puzzle", "label": "拼图验证"},
			{"value": "voice", "label": "语音验证"},
			{"value": "lianliankan", "label": "连连看验证"},
			{"value": "3d", "label": "3D验证"},
			{"value": "emoji", "label": "表情验证"},
			{"value": "semantic", "label": "语义验证"},
			{"value": "video", "label": "视频验证"},
			{"value": "ar", "label": "AR验证"},
		},
		"risk_levels": []gin.H{
			{"value": 1, "label": "极低风险", "steps": 1},
			{"value": 2, "label": "低风险", "steps": 2},
			{"value": 3, "label": "中等风险", "steps": 2},
			{"value": 4, "label": "高风险", "steps": 3},
			{"value": 5, "label": "极高风险", "steps": 4},
		},
		"max_steps": 5,
		"features": []string{
			"智能组合",
			"风险自适应",
			"多步骤验证",
			"行为分析",
		},
	})
}
