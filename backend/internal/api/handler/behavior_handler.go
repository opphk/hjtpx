package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/behavior"
)

type BehaviorHandler struct {
	trajectorySvc *behavior.TrajectoryService
	profileSvc    *behavior.ProfileService
	anomalySvc    *behavior.AnomalyService
}

func NewBehaviorHandler() *BehaviorHandler {
	return &BehaviorHandler{
		trajectorySvc: behavior.NewTrajectoryService(nil),
		profileSvc:    behavior.NewProfileService(nil),
		anomalySvc:    behavior.NewAnomalyService(nil),
	}
}

func (h *BehaviorHandler) SaveTrajectory(c *gin.Context) {
	var req struct {
		UserID        string                       `json:"user_id" binding:"required"`
		SessionID     string                      `json:"session_id" binding:"required"`
		ApplicationID uint                        `json:"application_id"`
		Points        []behavior.TrajectoryPoint   `json:"points" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	traj := &behavior.BehaviorTrajectory{
		UserID:        req.UserID,
		SessionID:     req.SessionID,
		ApplicationID: req.ApplicationID,
		Points:        req.Points,
	}

	if err := h.trajectorySvc.SaveTrajectory(c.Request.Context(), traj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": traj.ID}})
}

func (h *BehaviorHandler) GetTrajectory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	traj, err := h.trajectorySvc.GetTrajectory(c.Request.Context(), uint(id))
	if err != nil || traj == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": traj})
}

func (h *BehaviorHandler) ListTrajectories(c *gin.Context) {
	query := &behavior.TrajectoryQuery{
		UserID:    c.Query("user_id"),
		SessionID: c.Query("session_id"),
		Page:      1,
		PageSize:  20,
	}

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			query.Page = val
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			query.PageSize = val
		}
	}
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			query.StartDate = t
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse("2006-01-02", ed); err == nil {
			query.EndDate = t.Add(24*time.Hour - time.Second)
		}
	}

	trajs, total, err := h.trajectorySvc.ListTrajectories(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      trajs,
			"total":      total,
			"page":       query.Page,
			"page_size":  query.PageSize,
			"total_page": (total + int64(query.PageSize) - 1) / int64(query.PageSize),
		},
	})
}

func (h *BehaviorHandler) AnalyzeTrajectory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	traj, err := h.trajectorySvc.GetTrajectory(c.Request.Context(), uint(id))
	if err != nil || traj == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
		return
	}

	analysis, err := h.trajectorySvc.AnalyzeTrajectory(c.Request.Context(), traj)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": analysis})
}

func (h *BehaviorHandler) GetTrajectoryStatistics(c *gin.Context) {
	query := &behavior.TrajectoryQuery{UserID: c.Query("user_id")}
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse("2006-01-02", sd); err == nil {
			query.StartDate = t
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse("2006-01-02", ed); err == nil {
			query.EndDate = t.Add(24*time.Hour - time.Second)
		}
	}

	stats, err := h.trajectorySvc.GetStatistics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *BehaviorHandler) GetTrajectoryVisualization(c *gin.Context) {
	query := &behavior.TrajectoryQuery{
		UserID:   c.Query("user_id"),
		PageSize: 100,
	}
	trajs, _, err := h.trajectorySvc.ListTrajectories(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	data := make([]map[string]interface{}, 0)
	for _, t := range trajs {
		for _, p := range t.Points {
			data = append(data, map[string]interface{}{
				"x":         p.X,
				"y":         p.Y,
				"timestamp": p.Timestamp,
				"event":     p.Event,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"type":     "trajectory",
			"data":     data,
			"metadata": gin.H{"total_points": len(data)},
		},
	})
}

func (h *BehaviorHandler) GetUserProfile(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id required"})
		return
	}

	profile, err := h.profileSvc.GetProfile(c.Request.Context(), userID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *BehaviorHandler) CreateOrUpdateProfile(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		ApplicationID uint   `json:"application_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	analyses, _, _ := h.trajectorySvc.ListAnalyses(c.Request.Context(), &behavior.TrajectoryQuery{UserID: req.UserID, Page: 1, PageSize: 100})
	profile, err := h.profileSvc.GenerateProfile(c.Request.Context(), req.UserID, req.ApplicationID, analyses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *BehaviorHandler) ListProfiles(c *gin.Context) {
	query := &behavior.UserProfileQuery{
		UserID:   c.Query("user_id"),
		Page:     1,
		PageSize: 20,
	}

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			query.Page = val
		}
	}

	profiles, total, err := h.profileSvc.ListProfiles(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      profiles,
			"total":      total,
			"page":       query.Page,
			"page_size":  query.PageSize,
			"total_page": (total + int64(query.PageSize) - 1) / int64(query.PageSize),
		},
	})
}

func (h *BehaviorHandler) GetProfileStatistics(c *gin.Context) {
	query := &behavior.UserProfileQuery{}
	stats, err := h.profileSvc.GetStatistics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *BehaviorHandler) GetAnomalies(c *gin.Context) {
	query := &behavior.AnomalyQuery{
		UserID:   c.Query("user_id"),
		Type:     c.Query("type"),
		Severity: c.Query("severity"),
		Page:     1,
		PageSize: 20,
	}

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			query.Page = val
		}
	}

	if ip := c.Query("is_processed"); ip == "true" {
		t := true
		query.IsProcessed = &t
	} else if ip == "false" {
		f := false
		query.IsProcessed = &f
	}

	anomalies, total, err := h.anomalySvc.GetAnomalies(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      anomalies,
			"total":      total,
			"page":       query.Page,
			"page_size":  query.PageSize,
			"total_page": (total + int64(query.PageSize) - 1) / int64(query.PageSize),
		},
	})
}

func (h *BehaviorHandler) ProcessAnomaly(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	var req struct {
		IsFalsePositive bool `json:"is_false_positive"`
	}
	c.ShouldBindJSON(&req)

	if err := h.anomalySvc.ProcessAnomaly(c.Request.Context(), uint(id), req.IsFalsePositive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "processed"})
}

func (h *BehaviorHandler) GetAnomalyStatistics(c *gin.Context) {
	query := &behavior.AnomalyQuery{}
	stats, err := h.anomalySvc.GetStatistics(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *BehaviorHandler) ListRules(c *gin.Context) {
	rules, err := h.anomalySvc.ListRules(c.Request.Context(), c.Query("type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
}

func (h *BehaviorHandler) CreateRule(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Type        string                 `json:"type" binding:"required"`
		Severity   string                 `json:"severity" binding:"required"`
		Conditions  map[string]interface{} `json:"conditions"`
		Action     string                 `json:"action"`
		Threshold   float64                `json:"threshold"`
		IsEnabled   bool                   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	rule := &behavior.AnomalyRule{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Severity:    req.Severity,
		Action:      req.Action,
		Threshold:   req.Threshold,
		IsEnabled:   req.IsEnabled,
	}

	if err := h.anomalySvc.CreateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rule})
}

func (h *BehaviorHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	c.ShouldBindJSON(&req)

	if err := h.anomalySvc.ToggleRule(c.Request.Context(), uint(id), req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BehaviorHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	if err := h.anomalySvc.DeleteRule(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BehaviorHandler) GetRecentAnomalies(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	anomalies, total, err := h.anomalySvc.GetAnomalies(c.Request.Context(), &behavior.AnomalyQuery{Page: 1, PageSize: limit})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	_ = total

	c.JSON(http.StatusOK, gin.H{"success": true, "data": anomalies})
}

func (h *BehaviorHandler) GetUserTrajectorySummary(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id required"})
		return
	}

	analyses, _, err := h.trajectorySvc.ListAnalyses(c.Request.Context(), &behavior.TrajectoryQuery{UserID: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	summary := map[string]interface{}{
		"total_trajectories": len(analyses),
		"bot_count":          0,
		"human_count":        0,
		"avg_risk_score":     0.0,
	}

	var totalRisk float64
	for _, a := range analyses {
		if a.IsBotLikely {
			summary["bot_count"] = summary["bot_count"].(int) + 1
		} else {
			summary["human_count"] = summary["human_count"].(int) + 1
		}
		totalRisk += a.RiskScore
	}

	if len(analyses) > 0 {
		summary["avg_risk_score"] = totalRisk / float64(len(analyses))
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func (h *BehaviorHandler) ExportAnomalies(c *gin.Context) {
	query := &behavior.AnomalyQuery{
		UserID:   c.Query("user_id"),
		Type:     c.Query("type"),
		Severity: c.Query("severity"),
	}

	anomalies, _, err := h.anomalySvc.GetAnomalies(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": anomalies, "total": len(anomalies)}})
}

func (h *BehaviorHandler) AnalyzePatterns(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
}

func GetBehaviorHandler() *BehaviorHandler {
	return NewBehaviorHandler()
}
