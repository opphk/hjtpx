package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BiometricsV15Handler struct {
	biometricsService *service.BiometricsV15Service
}

func NewBiometricsV15Handler() *BiometricsV15Handler {
	return &BiometricsV15Handler{
		biometricsService: service.NewBiometricsV15Service(),
	}
}

func GetBiometricsV15Handler() *BiometricsV15Handler {
	return NewBiometricsV15Handler()
}

type RegisterMultimodalRequest struct {
	UserID        string                           `json:"user_id" binding:"required"`
	BiometricData *service.MultimodalBiometricData `json:"biometric_data"`
}

type VerifyMultimodalRequest struct {
	UserID        string                           `json:"user_id" binding:"required"`
	BiometricData *service.MultimodalBiometricData `json:"biometric_data"`
}

type RegisterMousePressureRequest struct {
	UserID    string                     `json:"user_id" binding:"required"`
	MouseData *service.MousePressureData `json:"mouse_data"`
}

type RegisterTouchForceRequest struct {
	UserID    string                  `json:"user_id" binding:"required"`
	TouchData *service.TouchForceData `json:"touch_data"`
}

type RegisterEyeTrackingRequest struct {
	UserID          string                   `json:"user_id" binding:"required"`
	EyeTrackingData *service.EyeTrackingData `json:"eye_tracking_data"`
}

type FusionVerifyRequest struct {
	UserID        string                           `json:"user_id" binding:"required"`
	BiometricData *service.MultimodalBiometricData `json:"biometric_data"`
}

type RegisterMultimodalResponse struct {
	Profile      *service.MultimodalBiometricProfile `json:"profile"`
	FeatureCount int                                 `json:"feature_count"`
	Message      string                              `json:"message"`
}

type VerifyMultimodalResponse struct {
	Result     *service.FusionVerificationResult `json:"result"`
	IsVerified bool                              `json:"is_verified"`
	Confidence float64                           `json:"confidence"`
	Message    string                            `json:"message"`
}

func RegisterMultimodalBiometricProfile(c *gin.Context) {
	var req RegisterMultimodalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	if req.BiometricData == nil {
		response.BadRequest(c, "生物特征数据不能为空")
		return
	}

	profile, err := handler.biometricsService.RegisterMultimodalProfile(req.UserID, req.BiometricData)
	if err != nil {
		response.InternalServerError(c, "注册多模态生物特征档案失败")
		return
	}

	resp := RegisterMultimodalResponse{
		Profile:      profile,
		FeatureCount: len(profile.FeatureVector),
		Message:      "多模态生物特征档案注册成功",
	}

	response.Success(c, resp)
}

func VerifyMultimodalBiometrics(c *gin.Context) {
	var req VerifyMultimodalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	if req.BiometricData == nil {
		response.BadRequest(c, "生物特征数据不能为空")
		return
	}

	result, err := handler.biometricsService.VerifyMultimodal(req.UserID, req.BiometricData)
	if err != nil {
		response.InternalServerError(c, "多模态生物特征验证失败")
		return
	}

	resp := VerifyMultimodalResponse{
		Result:     result,
		IsVerified: result.IsVerified,
		Confidence: result.OverallConfidence,
		Message:    result.DecisionDetails,
	}

	response.Success(c, resp)
}

func RegisterMousePressureProfile(c *gin.Context) {
	var req RegisterMousePressureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	biometricData := &service.MultimodalBiometricData{
		UserID:        req.UserID,
		MousePressure: req.MouseData,
	}

	profile, err := handler.biometricsService.RegisterMultimodalProfile(req.UserID, biometricData)
	if err != nil {
		response.InternalServerError(c, "注册鼠标压力特征档案失败")
		return
	}

	response.Success(c, gin.H{
		"profile": profile.MousePressureProfile,
		"message": "鼠标压力特征档案注册成功",
	})
}

func RegisterTouchForceProfile(c *gin.Context) {
	var req RegisterTouchForceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	biometricData := &service.MultimodalBiometricData{
		UserID:     req.UserID,
		TouchForce: req.TouchData,
	}

	profile, err := handler.biometricsService.RegisterMultimodalProfile(req.UserID, biometricData)
	if err != nil {
		response.InternalServerError(c, "注册触摸力度特征档案失败")
		return
	}

	response.Success(c, gin.H{
		"profile": profile.TouchForceProfile,
		"message": "触摸力度特征档案注册成功",
	})
}

func RegisterEyeTrackingProfile(c *gin.Context) {
	var req RegisterEyeTrackingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	biometricData := &service.MultimodalBiometricData{
		UserID:      req.UserID,
		EyeTracking: req.EyeTrackingData,
	}

	profile, err := handler.biometricsService.RegisterMultimodalProfile(req.UserID, biometricData)
	if err != nil {
		response.InternalServerError(c, "注册眼动追踪特征档案失败")
		return
	}

	response.Success(c, gin.H{
		"profile": profile.EyeTrackingProfile,
		"message": "眼动追踪特征档案注册成功",
	})
}

func FusionVerifyBiometrics(c *gin.Context) {
	var req FusionVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	if req.BiometricData == nil {
		response.BadRequest(c, "生物特征数据不能为空")
		return
	}

	result, err := handler.biometricsService.VerifyMultimodal(req.UserID, req.BiometricData)
	if err != nil {
		response.InternalServerError(c, "融合验证失败")
		return
	}

	response.Success(c, gin.H{
		"result":       result,
		"is_verified":  result.IsVerified,
		"confidence":   result.OverallConfidence,
		"risk_level":   result.RiskLevel,
		"modal_scores": result.ModalScores,
		"message":      result.DecisionDetails,
	})
}

func GetMultimodalProfile(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.BadRequest(c, "用户ID不能为空")
		return
	}

	handler := GetBiometricsV15Handler()
	profile, exists := handler.biometricsService.GetProfile(userID)

	if !exists {
		response.NotFound(c, "未找到该用户的生物特征档案")
		return
	}

	response.Success(c, gin.H{
		"profile":            profile,
		"feature_count":      len(profile.FeatureVector),
		"verification_count": profile.VerificationCount,
		"confidence_score":   profile.ConfidenceScore,
	})
}

func DeleteMultimodalProfile(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.BadRequest(c, "用户ID不能为空")
		return
	}

	handler := GetBiometricsV15Handler()
	deleted := handler.biometricsService.DeleteProfile(userID)

	if !deleted {
		response.NotFound(c, "未找到该用户的生物特征档案")
		return
	}

	response.Success(c, gin.H{
		"message": "生物特征档案删除成功",
	})
}

func GetBiometricsCapabilities(c *gin.Context) {
	response.Success(c, gin.H{
		"version": "15.0",
		"capabilities": gin.H{
			"mouse_pressure": gin.H{
				"enabled": true,
				"features": []string{
					"pressure_sensing",
					"click_pattern",
					"drag_pattern",
					"movement_analysis",
					"direction_preference",
					"force_calculation",
				},
			},
			"touch_force": gin.H{
				"enabled": true,
				"features": []string{
					"force_sensing",
					"swipe_analysis",
					"pinch_detection",
					"multi_touch",
					"velocity_profile",
					"direction_entropy",
				},
			},
			"eye_tracking": gin.H{
				"enabled": true,
				"features": []string{
					"gaze_tracking",
					"blink_detection",
					"fixation_analysis",
					"saccade_detection",
					"dwell_analysis",
					"focus_tracking",
					"scan_pattern",
					"attention_ratio",
				},
			},
			"fusion": gin.H{
				"enabled": true,
				"methods": []string{
					"weighted_averaging",
					"adaptive_weights",
					"cosine_similarity",
					"euclidean_distance",
					"feature_vector",
				},
			},
		},
		"modal_weights": gin.H{
			"mouse_pressure": 0.4,
			"touch_force":    0.3,
			"eye_tracking":   0.3,
		},
		"verification_threshold": 0.85,
	})
}

func AnalyzeBiometricData(c *gin.Context) {
	var req struct {
		BiometricData *service.MultimodalBiometricData `json:"biometric_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	analysis := gin.H{
		"timestamp":      req.BiometricData.Timestamp,
		"session_id":     req.BiometricData.SessionID,
		"device_info":    req.BiometricData.DeviceInfo,
		"modality_count": 0,
	}

	if req.BiometricData.MousePressure != nil {
		analysis["modality_count"] = analysis["modality_count"].(int) + 1
		analysis["mouse_pressure"] = req.BiometricData.MousePressure.PressureAnalysis
		analysis["mouse_movement"] = req.BiometricData.MousePressure.MovementAnalysis
		analysis["mouse_clicks"] = req.BiometricData.MousePressure.ClickAnalysis
	}

	if req.BiometricData.TouchForce != nil {
		analysis["modality_count"] = analysis["modality_count"].(int) + 1
		analysis["touch_force"] = req.BiometricData.TouchForce.ForceAnalysis
		analysis["touch_swipes"] = req.BiometricData.TouchForce.SwipeAnalysis
		analysis["touch_gestures"] = req.BiometricData.TouchForce.MultitouchAnalysis
	}

	if req.BiometricData.EyeTracking != nil {
		analysis["modality_count"] = analysis["modality_count"].(int) + 1
		analysis["eye_gaze"] = req.BiometricData.EyeTracking.GazeAnalysis
		analysis["eye_blinks"] = req.BiometricData.EyeTracking.BlinkAnalysis
		analysis["eye_fixations"] = req.BiometricData.EyeTracking.FixationAnalysis
		analysis["eye_saccades"] = req.BiometricData.EyeTracking.SaccadeAnalysis
	}

	_ = handler

	response.Success(c, analysis)
}

func CompareBiometricProfiles(c *gin.Context) {
	var req struct {
		UserID1 string `json:"user_id_1" binding:"required"`
		UserID2 string `json:"user_id_2" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	handler := GetBiometricsV15Handler()

	profile1, exists1 := handler.biometricsService.GetProfile(req.UserID1)
	profile2, exists2 := handler.biometricsService.GetProfile(req.UserID2)

	if !exists1 || !exists2 {
		response.NotFound(c, "未找到对应的生物特征档案")
		return
	}

	cosineSim := handler.biometricsService.CalculateCosineSimilarity(
		profile1.FeatureVector,
		profile2.FeatureVector,
	)

	euclideanDist := handler.biometricsService.CalculateEuclideanDistance(
		profile1.FeatureVector,
		profile2.FeatureVector,
	)

	similarity := 1.0 / (1.0 + euclideanDist)

	response.Success(c, gin.H{
		"user_id_1":             req.UserID1,
		"user_id_2":             req.UserID2,
		"cosine_similarity":     cosineSim,
		"euclidean_distance":    euclideanDist,
		"normalized_similarity": similarity,
		"feature_count":         len(profile1.FeatureVector),
		"is_same_user":          cosineSim > 0.9 && euclideanDist < 0.5,
	})
}
