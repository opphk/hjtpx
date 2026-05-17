package behavior

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

type TrajectoryService struct {
	db *gorm.DB
}

type BehaviorTrajectory struct {
	ID            uint                    `json:"id" gorm:"primaryKey"`
	UserID        string                 `json:"user_id" gorm:"column:user_id;index"`
	SessionID     string                 `json:"session_id" gorm:"column:session_id;index"`
	ApplicationID uint                   `json:"application_id" gorm:"column:application_id;index"`
	PointsJSON    string                 `json:"-" gorm:"column:points;type:text"`
	Points        []TrajectoryPoint      `json:"points" gorm:"-"`
	TotalDistance float64                `json:"total_distance" gorm:"column:total_distance"`
	Duration      int64                  `json:"duration" gorm:"column:duration"`
	StartTime     time.Time              `json:"start_time" gorm:"column:start_time"`
	EndTime       time.Time              `json:"end_time" gorm:"column:end_time"`
	CreatedAt     time.Time              `json:"created_at" gorm:"column:created_at"`
}

type TrajectoryPoint struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
}

type BehaviorAnalysis struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	TrajectoryID       uint      `json:"trajectory_id" gorm:"column:trajectory_id;index"`
	UserID            string    `json:"user_id" gorm:"column:user_id;index"`
	TotalDistance     float64   `json:"total_distance" gorm:"column:total_distance"`
	AverageSpeed      float64   `json:"average_speed" gorm:"column:average_speed"`
	MaxSpeed          float64   `json:"max_speed" gorm:"column:max_speed"`
	PathEfficiency    float64   `json:"path_efficiency" gorm:"column:path_efficiency"`
	DirectionChanges  int       `json:"direction_changes" gorm:"column:direction_changes"`
	CurvatureAvg      float64   `json:"curvature_avg" gorm:"column:curvature_avg"`
	JitterScore       float64   `json:"jitter_score" gorm:"column:jitter_score"`
	PauseCount       int       `json:"pause_count" gorm:"column:pause_count"`
	MicroCorrections  int       `json:"micro_corrections" gorm:"column:micro_corrections"`
	ClickCount       int       `json:"click_count" gorm:"column:click_count"`
	ClickRegularity   float64   `json:"click_regularity" gorm:"column:click_regularity"`
	IsBotLikely      bool      `json:"is_bot_likely" gorm:"column:is_bot_likely"`
	RiskScore        float64   `json:"risk_score" gorm:"column:risk_score"`
	Confidence       float64   `json:"confidence" gorm:"column:confidence"`
	FeaturesJSON     string    `json:"-" gorm:"column:features;type:text"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at"`
}

type UserProfile struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        string         `json:"user_id" gorm:"column:user_id;uniqueIndex;not null"`
	ApplicationID uint           `json:"application_id" gorm:"column:application_id;index"`
	ProfileData   string         `json:"-" gorm:"column:profile_data;type:text"`
	FeaturesJSON  string         `json:"-" gorm:"column:features;type:text"`
	Features      UserFeatures   `json:"features" gorm:"-"`
	LabelsJSON    string         `json:"-" gorm:"column:labels;type:text"`
	Labels        map[string]int `json:"labels" gorm:"-"`
	Tags          string         `json:"tags" gorm:"column:tags;type:text"`
	RiskLevel     string         `json:"risk_level" gorm:"column:risk_level;size:20;default:low"`
	TrustScore    float64        `json:"trust_score" gorm:"column:trust_score;default:50"`
	IsActive      bool           `json:"is_active" gorm:"column:is_active;default:true"`
	Version       int            `json:"version" gorm:"column:version;default:1"`
	LastUpdatedAt time.Time      `json:"last_updated_at" gorm:"column:last_updated_at"`
	CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"column:updated_at"`
}

type UserFeatures struct {
	MouseSpeedAvg      float64 `json:"mouse_speed_avg"`
	MouseSpeedStd      float64 `json:"mouse_speed_std"`
	ClickFrequency     float64 `json:"click_frequency"`
	ClickRegularity    float64 `json:"click_regularity"`
	PathEfficiencyAvg float64 `json:"path_efficiency_avg"`
	RiskScoreAvg      float64 `json:"risk_score_avg"`
	BotRate           float64 `json:"bot_rate"`
	SessionFrequency   int     `json:"session_frequency"`
	SuccessRate       float64 `json:"success_rate"`
}

type AnomalyRecord struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          string    `json:"user_id" gorm:"column:user_id;index"`
	SessionID       string    `json:"session_id" gorm:"column:session_id;index"`
	Type            string    `json:"type" gorm:"column:type;size:50;index"`
	Severity        string    `json:"severity" gorm:"column:severity;size:20;index"`
	Score           float64   `json:"score" gorm:"column:score"`
	AnomalyData     string    `json:"-" gorm:"column:anomaly_data;type:text"`
	Data            string    `json:"data" gorm:"-"`
	IsProcessed     bool      `json:"is_processed" gorm:"column:is_processed;default=false"`
	IsFalsePositive bool      `json:"is_false_positive" gorm:"column:is_false_positive;default=false"`
	Description     string    `json:"description" gorm:"column:description;type:text"`
	Recommendation  string    `json:"recommendation" gorm:"column:recommendation;type:text"`
	DetectedAt      time.Time `json:"detected_at" gorm:"column:detected_at"`
	ProcessedAt     *time.Time `json:"processed_at" gorm:"column:processed_at"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
}

type AnomalyRule struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Name           string     `json:"name" gorm:"column:name;size:100;not null"`
	Description    string     `json:"description" gorm:"column:description;type:text"`
	Type           string     `json:"type" gorm:"column:type;size:50;not null;index"`
	Severity       string     `json:"severity" gorm:"column:severity;size:20;not null"`
	ConditionsJSON string     `json:"-" gorm:"column:conditions;type:text"`
	Conditions     Conditions `json:"conditions" gorm:"-"`
	Action         string     `json:"action" gorm:"column:action;size:50"`
	IsEnabled      bool       `json:"is_enabled" gorm:"column:is_enabled;default=true"`
	Threshold      float64    `json:"threshold" gorm:"column:threshold"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

type Conditions struct {
	MinValue  float64 `json:"min_value,omitempty"`
	MaxValue  float64 `json:"max_value,omitempty"`
	Operator  string  `json:"operator,omitempty"`
	Field     string  `json:"field,omitempty"`
}

type TrajectoryQuery struct {
	UserID        string    `form:"user_id"`
	SessionID     string    `form:"session_id"`
	ApplicationID uint      `form:"application_id"`
	StartDate     time.Time `form:"start_date"`
	EndDate       time.Time `form:"end_date"`
	Page          int       `form:"page,default=1"`
	PageSize      int       `form:"page_size,default=20"`
}

type AnomalyQuery struct {
	UserID        string    `form:"user_id"`
	ApplicationID uint      `form:"application_id"`
	Type          string    `form:"type"`
	Severity      string    `form:"severity"`
	IsProcessed   *bool     `form:"is_processed"`
	StartDate     time.Time `form:"start_date"`
	EndDate       time.Time `form:"end_date"`
	Page          int       `form:"page,default=1"`
	PageSize      int       `form:"page_size,default=20"`
}

type UserProfileQuery struct {
	UserID        string `form:"user_id"`
	ApplicationID uint   `form:"application_id"`
	RiskLevel     string `form:"risk_level"`
	IsActive      *bool  `form:"is_active"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"page_size,default=20"`
}

func NewTrajectoryService(db *gorm.DB) *TrajectoryService {
	return &TrajectoryService{db: db}
}

func (s *TrajectoryService) SaveTrajectory(ctx context.Context, traj *BehaviorTrajectory) error {
	if len(traj.Points) > 0 {
		data, _ := json.Marshal(traj.Points)
		traj.PointsJSON = string(data)
		traj.StartTime = time.Unix(traj.Points[0].Timestamp/1000, 0)
		traj.EndTime = time.Unix(traj.Points[len(traj.Points)-1].Timestamp/1000, 0)
		traj.Duration = traj.Points[len(traj.Points)-1].Timestamp - traj.Points[0].Timestamp
	}
	return s.db.WithContext(ctx).Create(traj).Error
}

func (s *TrajectoryService) GetTrajectory(ctx context.Context, id uint) (*BehaviorTrajectory, error) {
	var traj BehaviorTrajectory
	if err := s.db.WithContext(ctx).First(&traj, id).Error; err != nil {
		return nil, err
	}
	if traj.PointsJSON != "" {
		json.Unmarshal([]byte(traj.PointsJSON), &traj.Points)
	}
	return &traj, nil
}

func (s *TrajectoryService) ListTrajectories(ctx context.Context, query *TrajectoryQuery) ([]BehaviorTrajectory, int64, error) {
	var trajs []BehaviorTrajectory
	var total int64

	db := s.db.WithContext(ctx).Model(&BehaviorTrajectory{})
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.SessionID != "" {
		db = db.Where("session_id = ?", query.SessionID)
	}
	if query.ApplicationID > 0 {
		db = db.Where("application_id = ?", query.ApplicationID)
	}
	if !query.StartDate.IsZero() {
		db = db.Where("start_time >= ?", query.StartDate)
	}
	if !query.EndDate.IsZero() {
		db = db.Where("end_time <= ?", query.EndDate)
	}

	db.Count(&total)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&trajs).Error; err != nil {
		return nil, 0, err
	}

	for i := range trajs {
		if trajs[i].PointsJSON != "" {
			json.Unmarshal([]byte(trajs[i].PointsJSON), &trajs[i].Points)
		}
	}
	return trajs, total, nil
}

func (s *TrajectoryService) AnalyzeTrajectory(ctx context.Context, traj *BehaviorTrajectory) (*BehaviorAnalysis, error) {
	if len(traj.Points) == 0 && traj.PointsJSON != "" {
		json.Unmarshal([]byte(traj.PointsJSON), &traj.Points)
	}

	analysis := &BehaviorAnalysis{
		TrajectoryID: traj.ID,
		UserID:       traj.UserID,
		CreatedAt:    time.Now(),
	}

	if len(traj.Points) < 2 {
		return analysis, nil
	}

	points := traj.Points
	analysis.TotalDistance = s.calculateDistance(points)
	speeds := s.calculateSpeeds(points)
	if len(speeds) > 0 {
		analysis.AverageSpeed = mean(speeds)
		analysis.MaxSpeed = maxFloat(speeds)
	}
	analysis.PathEfficiency = s.calculateEfficiency(points)
	analysis.DirectionChanges = s.countDirectionChanges(points)
	analysis.CurvatureAvg = s.calculateCurvature(points)
	analysis.JitterScore = s.calculateJitter(points)
	analysis.PauseCount = s.countPauses(points)
	analysis.MicroCorrections = s.countCorrections(points)

	clicks := s.extractClicks(points)
	analysis.ClickCount = len(clicks)
	if len(clicks) > 2 {
		analysis.ClickRegularity = s.calculateClickRegularity(clicks)
	}

	analysis.RiskScore = s.calculateRiskScore(analysis)
	analysis.IsBotLikely = analysis.RiskScore >= 50
	analysis.Confidence = s.calculateConfidence(analysis)

	features := map[string]float64{
		"total_distance":    analysis.TotalDistance,
		"average_speed":     analysis.AverageSpeed,
		"path_efficiency":   analysis.PathEfficiency,
		"direction_changes": float64(analysis.DirectionChanges),
		"risk_score":        analysis.RiskScore,
	}
	featuresJSON, _ := json.Marshal(features)
	analysis.FeaturesJSON = string(featuresJSON)

	if err := s.db.WithContext(ctx).Create(analysis).Error; err != nil {
		return nil, err
	}
	return analysis, nil
}

func (s *TrajectoryService) calculateDistance(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	dist := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		dist += math.Sqrt(dx*dx + dy*dy)
	}
	return dist
}

func (s *TrajectoryService) calculateSpeeds(points []TrajectoryPoint) []float64 {
	if len(points) < 2 {
		return nil
	}
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, dist/dt)
		}
	}
	return speeds
}

func (s *TrajectoryService) calculateEfficiency(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	first, last := points[0], points[len(points)-1]
	straight := math.Sqrt(math.Pow(float64(last.X-first.X), 2) + math.Pow(float64(last.Y-first.Y), 2))
	total := s.calculateDistance(points)
	if total == 0 {
		return 0
	}
	return straight / total
}

func (s *TrajectoryService) countDirectionChanges(points []TrajectoryPoint) int {
	if len(points) < 3 {
		return 0
	}
	changes := 0
	prevAngle := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)
		if i > 1 {
			diff := math.Abs(angle - prevAngle)
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			if diff > 0.5 {
				changes++
			}
		}
		prevAngle = angle
	}
	return changes
}

func (s *TrajectoryService) calculateCurvature(points []TrajectoryPoint) float64 {
	if len(points) < 3 {
		return 0
	}
	sum := 0.0
	count := 0
	for i := 1; i < len(points)-1; i++ {
		v1x := float64(points[i].X - points[i-1].X)
		v1y := float64(points[i].Y - points[i-1].Y)
		v2x := float64(points[i+1].X - points[i].X)
		v2y := float64(points[i+1].Y - points[i].Y)
		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)
		if mag1 > 0 && mag2 > 0 {
			cos := dot / (mag1 * mag2)
			if cos > 1 {
				cos = 1
			}
			sum += math.Acos(cos)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (s *TrajectoryService) calculateJitter(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	smoothed := s.smoothPoints(points, 5)
	origDist := s.calculateDistance(points)
	smoothDist := s.calculateDistance(smoothed)
	if origDist == 0 {
		return 0
	}
	return (origDist - smoothDist) / origDist
}

func (s *TrajectoryService) smoothPoints(points []TrajectoryPoint, window int) []TrajectoryPoint {
	if len(points) < window {
		return points
	}
	if window%2 == 0 {
		window++
	}
	half := window / 2
	result := make([]TrajectoryPoint, len(points))
	for i := range points {
		start, end := i-half, i+half
		if start < 0 {
			start = 0
		}
		if end >= len(points) {
			end = len(points) - 1
		}
		sumX, sumY, count := 0, 0, 0
		for j := start; j <= end; j++ {
			sumX += points[j].X
			sumY += points[j].Y
			count++
		}
		result[i] = points[i]
		result[i].X = sumX / count
		result[i].Y = sumY / count
	}
	return result
}

func (s *TrajectoryService) countPauses(points []TrajectoryPoint) int {
	if len(points) < 2 {
		return 0
	}
	count := 0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		dt := points[i].Timestamp - points[i-1].Timestamp
		if dist < 2 && dt > 100 {
			count++
		}
	}
	return count
}

func (s *TrajectoryService) countCorrections(points []TrajectoryPoint) int {
	if len(points) < 3 {
		return 0
	}
	count := 0
	prevAngle := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)
		if i > 1 {
			diff := math.Abs(angle - prevAngle)
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			dist := math.Sqrt(dx*dx + dy*dy)
			if diff > 2.0 && dist < 10 {
				count++
			}
		}
		prevAngle = angle
	}
	return count
}

func (s *TrajectoryService) extractClicks(points []TrajectoryPoint) []TrajectoryPoint {
	clicks := make([]TrajectoryPoint, 0)
	for _, p := range points {
		if p.Event == "click" {
			clicks = append(clicks, p)
		}
	}
	return clicks
}

func (s *TrajectoryService) calculateClickRegularity(clicks []TrajectoryPoint) float64 {
	if len(clicks) < 3 {
		return 0
	}
	intervals := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		intervals = append(intervals, float64(clicks[i].Timestamp-clicks[i-1].Timestamp))
	}
	m := mean(intervals)
	if m == 0 {
		return 0
	}
	v := variance(intervals)
	return 1 - math.Min(math.Sqrt(v)/m, 1)
}

func (s *TrajectoryService) calculateRiskScore(a *BehaviorAnalysis) float64 {
	score := 0.0
	if a.PathEfficiency > 0.92 && a.TotalDistance > 100 {
		score += 25
	}
	if a.JitterScore < 0.03 {
		score += 20
	}
	if a.CurvatureAvg < 0.05 && a.DirectionChanges < 5 {
		score += 20
	}
	if a.PauseCount == 0 && a.TotalDistance > 100 {
		score += 15
	}
	if a.MicroCorrections == 0 && a.TotalDistance > 100 {
		score += 15
	}
	if a.ClickRegularity > 0.9 && a.ClickCount > 2 {
		score += 15
	}
	return math.Min(score, 100)
}

func (s *TrajectoryService) calculateConfidence(a *BehaviorAnalysis) float64 {
	conf := 0.5
	if a.RiskScore > 70 || a.RiskScore < 30 {
		conf += 0.2
	}
	if a.DirectionChanges > 5 {
		conf += 0.1
	}
	if a.ClickCount > 0 {
		conf += 0.1
	}
	if a.TotalDistance > 200 {
		conf += 0.1
	}
	return math.Min(conf, 0.95)
}

func (s *TrajectoryService) ListAnalyses(ctx context.Context, query *TrajectoryQuery) ([]BehaviorAnalysis, int64, error) {
	var analyses []BehaviorAnalysis
	var total int64

	db := s.db.WithContext(ctx).Model(&BehaviorAnalysis{})
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	if !query.StartDate.IsZero() {
		db = db.Where("created_at >= ?", query.StartDate)
	}
	if !query.EndDate.IsZero() {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	db.Count(&total)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&analyses).Error; err != nil {
		return nil, 0, err
	}
	return analyses, total, nil
}

func (s *TrajectoryService) GetStatistics(ctx context.Context, query *TrajectoryQuery) (map[string]interface{}, error) {
	var total int64
	var totalDist, avgDist float64

	db := s.db.WithContext(ctx).Model(&BehaviorTrajectory{})
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	db.Count(&total)
	db.Select("COALESCE(SUM(total_distance), 0), COALESCE(AVG(total_distance), 0)").Row().Scan(&totalDist, &avgDist)

	var analyses []BehaviorAnalysis
	analysisQuery := s.db.WithContext(ctx).Model(&BehaviorAnalysis{})
	if query.UserID != "" {
		analysisQuery = analysisQuery.Where("user_id = ?", query.UserID)
	}

	var avgSpeed float64
	analysisQuery.Select("COALESCE(AVG(average_speed), 0)").Row().Scan(&avgSpeed)
	analysisQuery.Find(&analyses)

	botCount, humanCount := 0, 0
	var totalRisk float64
	for _, a := range analyses {
		if a.IsBotLikely {
			botCount++
		} else {
			humanCount++
		}
		totalRisk += a.RiskScore
	}
	avgRisk := 0.0
	if len(analyses) > 0 {
		avgRisk = totalRisk / float64(len(analyses))
	}

	return map[string]interface{}{
		"total_count":    total,
		"total_distance": totalDist,
		"avg_distance":   avgDist,
		"avg_speed":      avgSpeed,
		"bot_count":      botCount,
		"human_count":    humanCount,
		"bot_rate":       float64(botCount) / math.Max(float64(len(analyses)), 1) * 100,
		"avg_risk_score": avgRisk,
	}, nil
}

func (t *BehaviorTrajectory) TableName() string {
	return "behavior_trajectories"
}

func (a *BehaviorAnalysis) TableName() string {
	return "behavior_analyses"
}

type ProfileService struct {
	db *gorm.DB
}

func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{db: db}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	var profile UserProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if profile.FeaturesJSON != "" {
		json.Unmarshal([]byte(profile.FeaturesJSON), &profile.Features)
	}
	if profile.LabelsJSON != "" {
		json.Unmarshal([]byte(profile.LabelsJSON), &profile.Labels)
	}
	return &profile, nil
}

func (s *ProfileService) CreateProfile(ctx context.Context, profile *UserProfile) error {
	profile.Version = 1
	profile.LastUpdatedAt = time.Now()
	if profile.FeaturesJSON != "" {
		json.Unmarshal([]byte(profile.FeaturesJSON), &profile.Features)
	}
	if profile.LabelsJSON != "" {
		json.Unmarshal([]byte(profile.LabelsJSON), &profile.Labels)
	}
	if profile.FeaturesJSON == "" {
		data, _ := json.Marshal(profile.Features)
		profile.FeaturesJSON = string(data)
	}
	if profile.LabelsJSON == "" && profile.Labels != nil {
		data, _ := json.Marshal(profile.Labels)
		profile.LabelsJSON = string(data)
	}
	return s.db.WithContext(ctx).Create(profile).Error
}

func (s *ProfileService) UpdateProfile(ctx context.Context, profile *UserProfile) error {
	profile.Version++
	profile.LastUpdatedAt = time.Now()
	profile.UpdatedAt = time.Now()
	if data, err := json.Marshal(profile.Features); err == nil {
		profile.FeaturesJSON = string(data)
	}
	if data, err := json.Marshal(profile.Labels); err == nil {
		profile.LabelsJSON = string(data)
	}
	return s.db.WithContext(ctx).Save(profile).Error
}

func (s *ProfileService) ListProfiles(ctx context.Context, query *UserProfileQuery) ([]UserProfile, int64, error) {
	var profiles []UserProfile
	var total int64

	db := s.db.WithContext(ctx).Model(&UserProfile{})
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.ApplicationID > 0 {
		db = db.Where("application_id = ?", query.ApplicationID)
	}
	if query.RiskLevel != "" {
		db = db.Where("risk_level = ?", query.RiskLevel)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}

	db.Count(&total)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("updated_at DESC").Offset(offset).Limit(query.PageSize).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}

	for i := range profiles {
		if profiles[i].FeaturesJSON != "" {
			json.Unmarshal([]byte(profiles[i].FeaturesJSON), &profiles[i].Features)
		}
		if profiles[i].LabelsJSON != "" {
			json.Unmarshal([]byte(profiles[i].LabelsJSON), &profiles[i].Labels)
		}
	}
	return profiles, total, nil
}

func (s *ProfileService) GenerateProfile(ctx context.Context, userID string, applicationID uint, analyses []BehaviorAnalysis) (*UserProfile, error) {
	profile, _ := s.GetProfile(ctx, userID)
	if profile == nil {
		profile = &UserProfile{
			UserID:        userID,
			ApplicationID: applicationID,
			Features:      UserFeatures{},
			Labels:        make(map[string]int),
			RiskLevel:     "low",
			TrustScore:    50,
			IsActive:      true,
		}
	}

	if len(analyses) > 0 {
		var speeds, efficiencies, riskScores []float64
		botCount := 0
		for _, a := range analyses {
			speeds = append(speeds, a.AverageSpeed)
			efficiencies = append(efficiencies, a.PathEfficiency)
			riskScores = append(riskScores, a.RiskScore)
			if a.IsBotLikely {
				botCount++
			}
		}
		n := float64(len(analyses))
		profile.Features.MouseSpeedAvg = mean(speeds)
		profile.Features.MouseSpeedStd = stdFloat(speeds)
		profile.Features.ClickFrequency = mean(analysesToFloat64(analyses, func(a BehaviorAnalysis) float64 { return float64(a.ClickCount) }))
		profile.Features.ClickRegularity = mean(analysesToFloat64(analyses, func(a BehaviorAnalysis) float64 { return a.ClickRegularity }))
		profile.Features.PathEfficiencyAvg = mean(efficiencies)
		profile.Features.RiskScoreAvg = mean(riskScores)
		profile.Features.BotRate = float64(botCount) / n
		profile.Features.SessionFrequency = len(analyses)
		profile.Features.SuccessRate = 1 - profile.Features.BotRate

		avgRisk := profile.Features.RiskScoreAvg
		if avgRisk >= 70 {
			profile.RiskLevel = "critical"
			profile.TrustScore = 10
		} else if avgRisk >= 50 {
			profile.RiskLevel = "high"
			profile.TrustScore = 30
		} else if avgRisk >= 30 {
			profile.RiskLevel = "medium"
			profile.TrustScore = 60
		} else if avgRisk >= 10 {
			profile.RiskLevel = "low"
			profile.TrustScore = 80
		} else {
			profile.RiskLevel = "minimal"
			profile.TrustScore = 95
		}
	}

	if profile.ID == 0 {
		if err := s.CreateProfile(ctx, profile); err != nil {
			return nil, err
		}
	} else {
		if err := s.UpdateProfile(ctx, profile); err != nil {
			return nil, err
		}
	}
	return profile, nil
}

func (s *ProfileService) GetStatistics(ctx context.Context, query *UserProfileQuery) (map[string]interface{}, error) {
	var total, active, inactive int64
	s.db.WithContext(ctx).Model(&UserProfile{}).Count(&total)
	s.db.WithContext(ctx).Model(&UserProfile{}).Where("is_active = ?", true).Count(&active)
	s.db.WithContext(ctx).Model(&UserProfile{}).Where("is_active = ?", false).Count(&inactive)

	var avgTrust float64
	s.db.WithContext(ctx).Model(&UserProfile{}).Select("AVG(trust_score)").Row().Scan(&avgTrust)

	riskDist := make(map[string]int64)
	var riskCounts []struct {
		RiskLevel string
		Count     int64
	}
	s.db.WithContext(ctx).Model(&UserProfile{}).Select("risk_level, COUNT(*) as count").Group("risk_level").Scan(&riskCounts)
	for _, rc := range riskCounts {
		riskDist[rc.RiskLevel] = rc.Count
	}

	return map[string]interface{}{
		"total_profiles":    total,
		"active_profiles":   active,
		"inactive_profiles": inactive,
		"avg_trust_score":   avgTrust,
		"risk_distribution": riskDist,
	}, nil
}

func (p *UserProfile) TableName() string {
	return "behavior_user_profiles"
}

type AnomalyService struct {
	db *gorm.DB
}

func NewAnomalyService(db *gorm.DB) *AnomalyService {
	return &AnomalyService{db: db}
}

func (s *AnomalyService) DetectAnomalies(ctx context.Context, analyses []BehaviorAnalysis) ([]AnomalyRecord, error) {
	anomalies := make([]AnomalyRecord, 0)
	for _, a := range analyses {
		if a.AverageSpeed > 10 || a.MaxSpeed > 20 {
			anomalies = append(anomalies, s.createAnomaly(a, "speed_anomaly", "high", a.AverageSpeed, "异常速度模式"))
		}
		if a.PathEfficiency > 0.92 && a.TotalDistance > 100 {
			anomalies = append(anomalies, s.createAnomaly(a, "path_anomaly", "medium", a.PathEfficiency, "异常路径模式"))
		}
		if a.ClickRegularity > 0.9 && a.ClickCount > 2 {
			anomalies = append(anomalies, s.createAnomaly(a, "click_anomaly", "medium", a.ClickRegularity, "异常点击模式"))
		}
		if a.RiskScore >= 70 {
			anomalies = append(anomalies, s.createAnomaly(a, "bot_detection", "critical", a.RiskScore, "疑似机器人行为"))
		}
	}

	if len(anomalies) > 0 {
		for i := range anomalies {
			data, _ := json.Marshal(map[string]interface{}{
				"user_id": anomalies[i].UserID,
				"type":    anomalies[i].Type,
				"severity": anomalies[i].Severity,
			})
			anomalies[i].AnomalyData = string(data)
		}
		if err := s.db.WithContext(ctx).Create(&anomalies).Error; err != nil {
			return nil, err
		}
	}
	return anomalies, nil
}

func (s *AnomalyService) createAnomaly(a BehaviorAnalysis, anomalyType, severity string, score float64, desc string) AnomalyRecord {
	return AnomalyRecord{
		UserID:        a.UserID,
		SessionID:     fmt.Sprintf("%d", a.TrajectoryID),
		Type:          anomalyType,
		Severity:      severity,
		Score:         score,
		Description:   desc,
		Recommendation: getRecommendation(anomalyType, severity),
		DetectedAt:    time.Now(),
		IsProcessed:   false,
	}
}

func getRecommendation(anomalyType, severity string) string {
	recs := map[string]map[string]string{
		"speed_anomaly":   {"critical": "立即封禁并告警", "high": "阻止访问", "medium": "增加验证", "low": "记录日志"},
		"path_anomaly":    {"critical": "阻止访问并告警", "high": "限制访问频率", "medium": "增加验证挑战", "low": "记录日志"},
		"click_anomaly":   {"critical": "阻止操作并告警", "high": "触发图像验证码", "medium": "触发滑动验证", "low": "观察用户行为"},
		"bot_detection":   {"critical": "立即封禁并告警", "high": "阻止访问", "medium": "触发人机验证", "low": "标记为可疑"},
	}
	if m, ok := recs[anomalyType]; ok {
		if r, ok := m[severity]; ok {
			return r
		}
	}
	return "建议进行进一步分析"
}

func (s *AnomalyService) GetAnomalies(ctx context.Context, query *AnomalyQuery) ([]AnomalyRecord, int64, error) {
	var anomalies []AnomalyRecord
	var total int64

	db := s.db.WithContext(ctx).Model(&AnomalyRecord{})
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.IsProcessed != nil {
		db = db.Where("is_processed = ?", *query.IsProcessed)
	}

	db.Count(&total)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("detected_at DESC").Offset(offset).Limit(query.PageSize).Find(&anomalies).Error; err != nil {
		return nil, 0, err
	}

	for i := range anomalies {
		anomalies[i].Data = anomalies[i].AnomalyData
	}
	return anomalies, total, nil
}

func (s *AnomalyService) ProcessAnomaly(ctx context.Context, id uint, isFalsePositive bool) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&AnomalyRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_processed":      true,
		"is_false_positive": isFalsePositive,
		"processed_at":       now,
	}).Error
}

func (s *AnomalyService) GetStatistics(ctx context.Context, query *AnomalyQuery) (map[string]interface{}, error) {
	var total, processed, pending, falsePos int64
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Count(&total)
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Where("is_processed = ?", true).Count(&processed)
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Where("is_processed = ?", false).Count(&pending)
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Where("is_false_positive = ?", true).Count(&falsePos)

	typeDist := make(map[string]int64)
	var typeCounts []struct{ Type string; Count int64 }
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Select("type, COUNT(*) as count").Group("type").Scan(&typeCounts)
	for _, tc := range typeCounts {
		typeDist[tc.Type] = tc.Count
	}

	severityDist := make(map[string]int64)
	var sevCounts []struct{ Severity string; Count int64 }
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Select("severity, COUNT(*) as count").Group("severity").Scan(&sevCounts)
	for _, sc := range sevCounts {
		severityDist[sc.Severity] = sc.Count
	}

	var avgScore float64
	s.db.WithContext(ctx).Model(&AnomalyRecord{}).Select("AVG(score)").Row().Scan(&avgScore)

	return map[string]interface{}{
		"total_anomalies":      total,
		"processed_count":      processed,
		"pending_count":        pending,
		"false_positive_count": falsePos,
		"by_type":             typeDist,
		"by_severity":         severityDist,
		"avg_score":           avgScore,
	}, nil
}

func (s *AnomalyService) CreateRule(ctx context.Context, rule *AnomalyRule) error {
	if data, err := json.Marshal(rule.Conditions); err == nil {
		rule.ConditionsJSON = string(data)
	}
	return s.db.WithContext(ctx).Create(rule).Error
}

func (s *AnomalyService) ListRules(ctx context.Context, ruleType string) ([]AnomalyRule, error) {
	var rules []AnomalyRule
	query := s.db.WithContext(ctx).Model(&AnomalyRule{}).Where("is_enabled = ?", true)
	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}
	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ConditionsJSON != "" {
			json.Unmarshal([]byte(rules[i].ConditionsJSON), &rules[i].Conditions)
		}
	}
	return rules, nil
}

func (s *AnomalyService) ToggleRule(ctx context.Context, id uint, enabled bool) error {
	return s.db.WithContext(ctx).Model(&AnomalyRule{}).Where("id = ?", id).Update("is_enabled", enabled).Error
}

func (s *AnomalyService) DeleteRule(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&AnomalyRule{}, id).Error
}

func (r *AnomalyRecord) TableName() string {
	return "behavior_anomaly_records"
}

func (r *AnomalyRule) TableName() string {
	return "behavior_anomaly_rules"
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := mean(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-m, 2)
	}
	return sum / float64(len(values))
}

func stdFloat(values []float64) float64 {
	return math.Sqrt(variance(values))
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func analysesToFloat64(analyses []BehaviorAnalysis, fn func(BehaviorAnalysis) float64) []float64 {
	result := make([]float64, len(analyses))
	for i, a := range analyses {
		result[i] = fn(a)
	}
	return result
}
