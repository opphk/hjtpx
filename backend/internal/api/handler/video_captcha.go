package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// VideoCaptchaGenerateRequest 视频验证码生成请求
type VideoCaptchaGenerateRequest struct {
	Width  int `json:"width" binding:"omitempty,min=320,max=1920"`
	Height int `json:"height" binding:"omitempty,min=240,max=1080"`
	// 验证难度：1-简单，2-中等，3-困难
	Difficulty int `json:"difficulty" binding:"omitempty,min=1,max=3"`
}

// VideoCaptchaVerifyRequest 视频验证码验证请求
type VideoCaptchaVerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	Answer    string `json:"answer" binding:"required"`
	// 用户行为数据
	BehaviorData map[string]interface{} `json:"behavior_data"`
}

// VideoCaptchaGenerate 生成视频验证码
// @Summary 生成视频验证码
// @Description 生成一个新的视频验证码
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body VideoCaptchaGenerateRequest true "生成请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/video/generate [post]
func VideoCaptchaGenerate(c *gin.Context) {
	var req VideoCaptchaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 设置默认值
	if req.Width == 0 {
		req.Width = 640
	}
	if req.Height == 0 {
		req.Height = 360
	}
	if req.Difficulty == 0 {
		req.Difficulty = 2
	}

	captchaID := uuid.New().String()
	videoURL, _, err := generateVideoCaptcha(req.Width, req.Height, req.Difficulty)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成失败")
		return
	}

	// 存储验证码信息（实际项目应存入Redis）
	// 这里简化处理

	response.Success(c, gin.H{
		"captcha_id": captchaID,
		"video_url":  videoURL,
		"width":      req.Width,
		"height":     req.Height,
		"difficulty": req.Difficulty,
		"message":    "请观察视频并回答问题",
	})
}

// VideoCaptchaVerify 验证视频验证码
// @Summary 验证视频验证码
// @Description 验证用户对视频验证码的回答
// @Tags captcha
// @Accept json
// @Produce json
// @Param request body VideoCaptchaVerifyRequest true "验证请求参数"
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/video/verify [post]
func VideoCaptchaVerify(c *gin.Context) {
	var req VideoCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 验证逻辑（实际项目需要从Redis获取答案并验证）
	isValid, score := verifyVideoCaptchaAnswer(req.CaptchaID, req.Answer, req.BehaviorData)

	if isValid {
		response.Success(c, gin.H{
			"success": true,
			"score":   score,
			"message": "验证成功",
		})
	} else {
		response.Fail(c, 400, "验证失败")
	}
}

// generateVideoCaptcha 生成视频验证码（模拟实现）
func generateVideoCaptcha(width, height, difficulty int) (string, string, error) {
	// 实际项目中这里应该生成真实的视频
	// 这里返回一个模拟的视频URL和答案
	videoURL := "/static/video/captcha_demo.mp4"

	var answer string
	switch difficulty {
	case 1:
		answer = "3" // 简单：视频中有多少个圆形
	case 2:
		answer = "blue" // 中等：主要物体是什么颜色
	case 3:
		answer = "triangle" // 困难：最后出现的图形是什么形状
	}

	return videoURL, answer, nil
}

// verifyVideoCaptchaAnswer 验证视频验证码答案（模拟实现）
func verifyVideoCaptchaAnswer(captchaID, answer string, behaviorData map[string]interface{}) (bool, float64) {
	// 实际项目中需要从存储获取正确答案并验证
	// 这里简化处理，只验证非空
	if answer != "" {
		// 计算行为得分（如果有行为数据）
		score := 0.85
		if behaviorData != nil {
			// 根据行为数据调整分数
			if mouseMoveCount, ok := behaviorData["mouse_move_count"].(float64); ok {
				if mouseMoveCount > 10 {
					score += 0.05
				}
			}
		}
		return true, score
	}
	return false, 0.0
}

// VideoCaptchaOptions 获取视频验证码选项配置
// @Summary 获取视频验证码选项
// @Description 获取视频验证码的配置选项
// @Tags captcha
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}} "成功"
// @Router /api/v1/captcha/video/options [get]
func VideoCaptchaOptions(c *gin.Context) {
	response.Success(c, gin.H{
		"difficulty_options": []gin.H{
			{"value": 1, "label": "简单"},
			{"value": 2, "label": "中等"},
			{"value": 3, "label": "困难"},
		},
		"video_formats": []string{"mp4", "webm"},
		"features": []string{
			"行为分析",
			"AI检测",
			"多难度",
		},
	})
}
