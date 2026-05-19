package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type TrajectoryPoint struct {
	X        float64            `json:"x"`
	Y        float64            `json:"y"`
	Timestamp int64             `json:"timestamp"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Trajectory struct {
	ID          uint             `json:"id" gorm:"primaryKey"`
	SessionID   string           `json:"session_id" binding:"required"`
	UserID      uint             `json:"user_id"`
	AppID       uint             `json:"app_id"`
	EventType   string           `json:"event_type"`
	Points      string           `json:"points"`
	Duration    int64            `json:"duration"`
	PointCount  int              `json:"point_count"`
	IsComplete  bool             `json:"is_complete" gorm:"default:false"`
	Metadata    string           `json:"metadata,omitempty"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TrajectoryPlayback struct {
	ID        uint             `json:"id"`
	Points    []TrajectoryPoint `json:"points"`
	Duration  int64            `json:"duration"`
	Speed     float64          `json:"speed"`
	CurrentIndex int           `json:"current_index"`
	IsPlaying bool             `json:"is_playing"`
}

type TrajectoryPlaybackSession struct {
	ID            uint                `json:"id"`
	TrajectoryID  uint                `json:"trajectory_id"`
	Points        []TrajectoryPoint   `json:"points"`
	Duration      int64               `json:"duration"`
	Speed         float64             `json:"speed"`
	CurrentIndex  int                 `json:"current_index"`
	IsPlaying     bool                `json:"is_playing"`
	StartedAt     time.Time           `json:"started_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

var trajectoryPlaybackSessions = make(map[uint]*TrajectoryPlaybackSession)

type TrajectoryService struct{}

func NewTrajectoryService() *TrajectoryService {
	return &TrajectoryService{}
}

func (s *TrajectoryService) CreateTrajectory(sessionID string, userID, appID uint, eventType string, points []TrajectoryPoint, metadata map[string]interface{}) (*Trajectory, error) {
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal points: %w", err)
	}

	metadataJSON, _ := json.Marshal(metadata)

	var duration int64
	if len(points) > 1 {
		duration = points[len(points)-1].Timestamp - points[0].Timestamp
	}

	trajectory := &Trajectory{
		SessionID:  sessionID,
		UserID:    userID,
		AppID:     appID,
		EventType: eventType,
		Points:    string(pointsJSON),
		Duration:  duration,
		PointCount: len(points),
		Metadata:  string(metadataJSON),
		IsComplete: true,
	}

	if err := database.DB.Create(trajectory).Error; err != nil {
		return nil, fmt.Errorf("failed to save trajectory: %w", err)
	}

	return trajectory, nil
}

func (s *TrajectoryService) GetTrajectory(id uint) (*Trajectory, error) {
	var trajectory Trajectory
	if err := database.DB.First(&trajectory, id).Error; err != nil {
		return nil, err
	}
	return &trajectory, nil
}

func (s *TrajectoryService) GetTrajectoryPoints(trajectory *Trajectory) ([]TrajectoryPoint, error) {
	var points []TrajectoryPoint
	if err := json.Unmarshal([]byte(trajectory.Points), &points); err != nil {
		return nil, fmt.Errorf("failed to unmarshal points: %w", err)
	}
	return points, nil
}

func (s *TrajectoryService) GetTrajectoryBySession(sessionID string) ([]Trajectory, error) {
	var trajectories []Trajectory
	if err := database.DB.Where("session_id = ?", sessionID).Order("created_at DESC").Find(&trajectories).Error; err != nil {
		return nil, err
	}
	return trajectories, nil
}

func (s *TrajectoryService) ListTrajectories(userID uint, limit, offset int) ([]Trajectory, int64, error) {
	var trajectories []Trajectory
	var total int64

	query := database.DB.Model(&Trajectory{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&trajectories).Error; err != nil {
		return nil, 0, err
	}

	return trajectories, total, nil
}

func (s *TrajectoryService) DeleteTrajectory(id uint) error {
	return database.DB.Delete(&Trajectory{}, id).Error
}

func (s *TrajectoryService) UpdateTrajectory(id uint, points []TrajectoryPoint, isComplete bool) error {
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("failed to marshal points: %w", err)
	}

	var duration int64
	if len(points) > 1 {
		duration = points[len(points)-1].Timestamp - points[0].Timestamp
	}

	return database.DB.Model(&Trajectory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"points":      string(pointsJSON),
		"point_count": len(points),
		"duration":    duration,
		"is_complete": isComplete,
	}).Error
}

func CreateTrajectoryHandler(c *gin.Context) {
	var req struct {
		SessionID string               `json:"session_id" binding:"required"`
		UserID    uint                 `json:"user_id"`
		AppID     uint                 `json:"app_id"`
		EventType string               `json:"event_type" binding:"required"`
		Points    []TrajectoryPoint    `json:"points" binding:"required"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if len(req.Points) == 0 {
		response.BadRequest(c, "points cannot be empty")
		return
	}

	service := NewTrajectoryService()
	trajectory, err := service.CreateTrajectory(req.SessionID, req.UserID, req.AppID, req.EventType, req.Points, req.Metadata)
	if err != nil {
		response.Fail(c, response.CodeInternalError, fmt.Sprintf("failed to create trajectory: %v", err))
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "trajectory created successfully",
		"data": gin.H{
			"id":         trajectory.ID,
			"point_count": trajectory.PointCount,
			"duration":   trajectory.Duration,
		},
	})
}

func GetTrajectoryHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	service := NewTrajectoryService()
	trajectory, err := service.GetTrajectory(idNum)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "trajectory not found")
		return
	}

	points, err := service.GetTrajectoryPoints(trajectory)
	if err != nil {
		response.Fail(c, response.CodeInternalError, "failed to parse trajectory points")
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          trajectory.ID,
			"session_id":  trajectory.SessionID,
			"user_id":     trajectory.UserID,
			"app_id":      trajectory.AppID,
			"event_type":  trajectory.EventType,
			"points":      points,
			"duration":    trajectory.Duration,
			"point_count": trajectory.PointCount,
			"is_complete": trajectory.IsComplete,
			"created_at":  trajectory.CreatedAt,
		},
	})
}

func ListTrajectoriesHandler(c *gin.Context) {
	userID := c.Query("user_id")
	sessionID := c.Query("session_id")
	limit := 20
	offset := 0

	fmt.Sscanf(c.DefaultQuery("limit", "20"), "%d", &limit)
	fmt.Sscanf(c.DefaultQuery("offset", "0"), "%d", &offset)

	var uid uint
	fmt.Sscanf(userID, "%d", &uid)

	service := NewTrajectoryService()

	if sessionID != "" {
		trajectories, err := service.GetTrajectoryBySession(sessionID)
		if err != nil {
			response.Fail(c, response.CodeInternalError, "failed to get trajectories")
			return
		}

		c.JSON(200, gin.H{
			"code": 0,
			"data": gin.H{
				"trajectories": trajectories,
				"total":        len(trajectories),
			},
		})
		return
	}

	trajectories, total, err := service.ListTrajectories(uid, limit, offset)
	if err != nil {
		response.Fail(c, response.CodeInternalError, "failed to list trajectories")
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"trajectories": trajectories,
			"total":        total,
			"limit":        limit,
			"offset":       offset,
		},
	})
}

func DeleteTrajectoryHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	service := NewTrajectoryService()
	if err := service.DeleteTrajectory(idNum); err != nil {
		response.Fail(c, response.CodeInternalError, "failed to delete trajectory")
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "trajectory deleted successfully",
	})
}

func StartPlaybackHandler(c *gin.Context) {
	var req struct {
		TrajectoryID uint    `json:"trajectory_id" binding:"required"`
		Speed       float64 `json:"speed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	if req.Speed <= 0 {
		req.Speed = 1.0
	}

	service := NewTrajectoryService()
	trajectory, err := service.GetTrajectory(req.TrajectoryID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "trajectory not found")
		return
	}

	points, err := service.GetTrajectoryPoints(trajectory)
	if err != nil {
		response.Fail(c, response.CodeInternalError, "failed to parse trajectory")
		return
	}

	session := &TrajectoryPlaybackSession{
		ID:           req.TrajectoryID,
		TrajectoryID: req.TrajectoryID,
		Points:       points,
		Duration:     trajectory.Duration,
		Speed:        req.Speed,
		CurrentIndex: 0,
		IsPlaying:    true,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	trajectoryPlaybackSessions[req.TrajectoryID] = session

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"playback_id":    session.ID,
			"total_points":   len(points),
			"duration":       trajectory.Duration,
			"speed":          session.Speed,
			"current_index":  0,
		},
	})
}

func UpdatePlaybackHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	session, exists := trajectoryPlaybackSessions[idNum]
	if !exists {
		response.Fail(c, response.CodeNotFound, "playback session not found")
		return
	}

	var req struct {
		Action       string  `json:"action" binding:"required"`
		Speed        float64 `json:"speed"`
		CurrentIndex int     `json:"current_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	switch req.Action {
	case "play":
		session.IsPlaying = true
	case "pause":
		session.IsPlaying = false
	case "stop":
		session.IsPlaying = false
		session.CurrentIndex = 0
	case "seek":
		if req.CurrentIndex >= 0 && req.CurrentIndex < len(session.Points) {
			session.CurrentIndex = req.CurrentIndex
		}
	case "speed":
		if req.Speed > 0 {
			session.Speed = req.Speed
		}
	}

	session.UpdatedAt = time.Now()

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"playback_id":    session.ID,
			"is_playing":     session.IsPlaying,
			"current_index":  session.CurrentIndex,
			"speed":          session.Speed,
			"progress":       float64(session.CurrentIndex) / float64(len(session.Points)) * 100,
		},
	})
}

func GetPlaybackStatusHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	session, exists := trajectoryPlaybackSessions[idNum]
	if !exists {
		response.Fail(c, response.CodeNotFound, "playback session not found")
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"playback_id":    session.ID,
			"trajectory_id":  session.TrajectoryID,
			"is_playing":    session.IsPlaying,
			"current_index": session.CurrentIndex,
			"total_points":  len(session.Points),
			"speed":         session.Speed,
			"progress":      float64(session.CurrentIndex) / float64(len(session.Points)) * 100,
		},
	})
}

func GetPlaybackPointsHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	session, exists := trajectoryPlaybackSessions[idNum]
	if !exists {
		response.Fail(c, response.CodeNotFound, "playback session not found")
		return
	}

	startIndex := 0
	endIndex := len(session.Points)

	startStr := c.Query("start")
	endStr := c.Query("end")

	fmt.Sscanf(startStr, "%d", &startIndex)
	fmt.Sscanf(endStr, "%d", &endIndex)

	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(session.Points) {
		endIndex = len(session.Points)
	}

	points := session.Points[startIndex:endIndex]

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"points":       points,
			"start_index":  startIndex,
			"end_index":    endIndex,
			"current_index": session.CurrentIndex,
		},
	})
}

func StopPlaybackHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	if _, exists := trajectoryPlaybackSessions[idNum]; exists {
		delete(trajectoryPlaybackSessions, idNum)
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "playback stopped successfully",
	})
}

func RecordTrajectoryHandler(c *gin.Context) {
	var req struct {
		SessionID string               `json:"session_id" binding:"required"`
		UserID    uint                 `json:"user_id"`
		AppID     uint                 `json:"app_id"`
		EventType string               `json:"event_type" binding:"required"`
		Points    []TrajectoryPoint    `json:"points"`
		IsComplete bool                `json:"is_complete"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	var trajectory Trajectory
	var isNew bool

	err := database.DB.Where("session_id = ? AND event_type = ? AND is_complete = ?", req.SessionID, req.EventType, false).First(&trajectory).Error
	if err != nil {
		isNew = true
		trajectory = Trajectory{
			SessionID: req.SessionID,
			UserID:    req.UserID,
			AppID:     req.AppID,
			EventType: req.EventType,
			IsComplete: false,
		}
	}

	if len(req.Points) > 0 {
		pointsJSON, _ := json.Marshal(req.Points)
		trajectory.Points = string(pointsJSON)
		trajectory.PointCount = len(req.Points)
	}

	trajectory.IsComplete = req.IsComplete

	if trajectory.PointCount > 1 {
		var points []TrajectoryPoint
		json.Unmarshal([]byte(trajectory.Points), &points)
		trajectory.Duration = points[len(points)-1].Timestamp - points[0].Timestamp
	}

	if isNew {
		if err := database.DB.Create(&trajectory).Error; err != nil {
			response.Fail(c, response.CodeInternalError, "failed to create trajectory")
			return
		}
	} else {
		if err := database.DB.Save(&trajectory).Error; err != nil {
			response.Fail(c, response.CodeInternalError, "failed to update trajectory")
			return
		}
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           trajectory.ID,
			"point_count":  trajectory.PointCount,
			"is_complete":  trajectory.IsComplete,
		},
	})
}

func ExportTrajectoryHandler(c *gin.Context) {
	id := c.Param("id")
	var idNum uint
	fmt.Sscanf(id, "%d", &idNum)

	service := NewTrajectoryService()
	trajectory, err := service.GetTrajectory(idNum)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "trajectory not found")
		return
	}

	points, _ := service.GetTrajectoryPoints(trajectory)

	exportData := map[string]interface{}{
		"id":         trajectory.ID,
		"session_id": trajectory.SessionID,
		"user_id":    trajectory.UserID,
		"app_id":     trajectory.AppID,
		"event_type": trajectory.EventType,
		"points":     points,
		"duration":   trajectory.Duration,
		"created_at": trajectory.CreatedAt,
	}

	format := c.DefaultQuery("format", "json")

	if format == "csv" {
		csv := "x,y,timestamp\n"
		for _, p := range points {
			csv += fmt.Sprintf("%f,%f,%d\n", p.X, p.Y, p.Timestamp)
		}
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=trajectory_%d.csv", idNum))
		c.String(200, csv)
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": exportData,
	})
}
