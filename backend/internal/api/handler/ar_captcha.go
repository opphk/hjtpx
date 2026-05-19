package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// ARCaptchaGenerateRequest AR验证码生成请求
type ARCaptchaGenerateRequest struct {
	SceneType string `json:"scene_type" binding:"omitempty,oneof=object gesture placement"`
	Width  int `json:"width" binding:"omitempty,min=320,max=1920"`
	Height int `json:"height" binding:"omitempty,min=240,max=1080"`
	Difficulty int `json:"difficulty" binding:"omitempty,min=1,max=3"`
}

// ARCaptchaVerifyRequest AR验证码验证请求
type ARCaptchaVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	SceneData map[string]interface{} `json:"scene_data" binding:"required"`
	ActionData map[string]interface{} `json:"action_data"`
	BehaviorData map[string]interface{} `json:"behavior_data"`
}

// ARCaptchaGenerate 生成AR验证码
// @Summary 生成AR验证码
// @Description 生成一个新的AR验证码
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body ARCaptchaGenerateRequest true "生成请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/ar/generate [post]
func ARCaptchaGenerate(c *gin.Context) {
	var req ARCaptchaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 设置默认值
	if req.SceneType == "" {
		req.SceneType = "object"
	}
	if req.Width == 0 {
		req.Width = 640
	}
	if req.Height == 0 {
		req.Height = 480
	}
	if req.Difficulty == 0 {
		req.Difficulty = 2
	}

	captchaID := uuid.New().String()
	sceneConfig, err := generateARScene(req.SceneType, req.Width, req.Height, req.Difficulty)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成失败")
		return
	}

	response.Success(c, gin.H{
		"captcha_id": captchaID,
		"scene_type": req.SceneType,
		"scene_config": sceneConfig,
		"width": req.Width,
		"height": req.Height,
		"difficulty": req.Difficulty,
		"instructions": getInstructionsBySceneType(req.SceneType, req.Difficulty),
	})
}

// ARCaptchaVerify 验证AR验证码
// @Summary 验证AR验证码
// @Description 验证用户的AR交互操作
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body ARCaptchaVerifyRequest true "验证请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/ar/verify [post]
func ARCaptchaVerify(c *gin.Context) {
	var req ARCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	isValid, score := verifyARInteraction(req.CaptchaID, req.SceneData, req.ActionData, req.BehaviorData)

	if isValid {
		response.Success(c, gin.H{
			"success": true,
			"score": score,
			"message": "验证成功",
		})
	} else {
		response.Fail(c, 400, "验证失败")
	}
}

// generateARScene 生成AR场景（模拟实现）
func generateARScene(sceneType string, width, height, difficulty int) (map[string]interface{}, error) {
	sceneConfig := map[string]interface{}{
		"objects": []map[string]interface{}{},
		"environment": "room",
		"lighting": "natural",
	}

	switch sceneType {
	case "object":
		// 物体放置场景
		sceneConfig["objects"] = []map[string]interface{}{
			{"type": "cube", "position": []float64{0, 0, 0}, "color": "red", "target_position": []float64{1, 0, 0}},
			{"type": "sphere", "position": []float64{2, 1, 0}, "color": "blue", "target_position": []float64{-1, 0, 0}},
		}
	case "gesture":
		// 手势识别场景
		sceneConfig["gestures"] = []string{"wave", "point", "circle"}
		sceneConfig["target_gesture"] = "point"
	case "placement":
		// 物体定位场景
		sceneConfig["target_zone"] = map[string]interface{}{
			"position": []float64{0, 0.5, 0},
			"size": []float64{0.5, 0.5, 0.5},
		}
		sceneConfig["objects"] = []map[string]interface{}{
			{"type": "pyramid", "position": []float64{2, 0, 0}, "color": "gold"},
		}
	}

	return sceneConfig, nil
}

// verifyARInteraction 验证AR交互（模拟实现）
func verifyARInteraction(captchaID string, sceneData, actionData, behaviorData map[string]interface{}) (bool, float64) {
	score := 0.0
	
	// 验证场景数据存在
	if sceneData == nil {
		return false, 0.0
	}

	// 简单检查验证数据完整性
	if _, ok := sceneData["object_position"].([]interface{}); ok {
		score += 0.3
	}
	if _, ok := sceneData["object_rotation"].([]interface{}); ok {
		score += 0.2
	}
	
	// 检查行为数据
	if behaviorData != nil {
		if moveCount, ok := behaviorData["move_count"].(float64); ok && moveCount > 5 {
			score += 0.2
		}
		if timeSpent, ok := behaviorData["time_spent"].(float64); ok && timeSpent > 2.0 {
			score += 0.2
		}
	}
	
	// 检查动作数据
	if actionData != nil {
		score += 0.1
	}

	// 综合判定
	isValid := score >= 0.5
	if score > 1.0 {
		score = 1.0
	}

	return isValid, score
}

// getInstructionsBySceneType 获取场景类型对应的指令
func getInstructionsBySceneType(sceneType string, difficulty int) string {
	switch sceneType {
	case "object":
		if difficulty <= 1 {
			return "请将红色方块拖动到指定位置"
		} else if difficulty == 2 {
			return "请将多个物体拖动到各自的目标位置"
		}
		return "请按顺序将物体放置到正确位置"
	case "gesture":
		return "请做出指定的手势动作"
	case "placement":
		return "请将物体放置到高亮区域内"
	default:
		return "请按照提示完成验证"
	}
}

// ARCaptchaOptions 获取AR验证码选项
// @Summary 获取AR验证码选项
// @Description 获取AR验证码的配置选项
// @Tags captcha
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/ar/options [get]
func ARCaptchaOptions(c *gin.Context) {
	response.Success(c, gin.H{
		"scene_types": []gin.H{
			{"value": "object", "label": "物体操作"},
			{"value": "gesture", "label": "手势识别"},
			{"value": "placement", "label": "物体定位"},
		},
		"difficulty_options": []gin.H{
			{"value": 1, "label": "简单"},
			{"value": 2, "label": "中等"},
			{"value": 3, "label": "困难"},
		},
		"features": []string{
			"3D渲染",
			"WebXR支持",
			"行为分析",
			"AI检测",
		},
	})
}
