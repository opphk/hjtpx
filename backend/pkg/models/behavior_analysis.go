package models

import (
	"time"
)

type BehaviorFeatures struct {
	MouseMoveCount    int       `json:"mouse_move_count"`
	MouseClickCount   int       `json:"mouse_click_count"`
	KeyboardCount     int       `json:"keyboard_count"`
	ScrollCount       int       `json:"scroll_count"`
	TouchCount        int       `json:"touch_count"`
	AverageSpeed      float64   `json:"average_speed"`
	TotalDistance     float64   `json:"total_distance"`
	MaxSpeed          float64   `json:"max_speed"`
	MinSpeed          float64   `json:"min_speed"`
	SessionDuration   int64     `json:"session_duration"`
	IdleTime          int64     `json:"idle_time"`
	ActivityRate      float64   `json:"activity_rate"`
	DirectionChanges  int       `json:"direction_changes"`
	ClickPositions    []string  `json:"click_positions"`
	FocusCount        int       `json:"focus_count"`
	CopyPasteCount    int       `json:"copy_paste_count"`
	RightClickCount   int       `json:"right_click_count"`
	WheelDelta        int       `json:"wheel_delta"`
	TouchPoints       int       `json:"touch_points"`
	GestureComplexity float64   `json:"gesture_complexity"`
	Timestamp         time.Time `json:"timestamp"`
}

type ScoreCard struct {
	SessionID       string             `json:"session_id"`
	UserID          uint               `json:"user_id,omitempty"`
	IPAddress       string             `json:"ip_address"`
	UserAgent       string             `json:"user_agent"`
	Features        BehaviorFeatures   `json:"features"`
	RiskScore       float64            `json:"risk_score"`
	IsAnomaly       bool               `json:"is_anomaly"`
	AnomalyReasons  []string           `json:"anomaly_reasons"`
	Recommendations []string           `json:"recommendations"`
	ModelVersion    string             `json:"model_version"`
	PredictionTime  time.Time          `json:"prediction_time"`
	Labels          map[string]float64 `json:"labels"`
}

type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type RuleAction struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}

type MLRule struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Name        string           `gorm:"size:100;not null" json:"name"`
	Description string           `gorm:"size:255" json:"description"`
	Conditions  []RuleCondition  `gorm:"-" json:"conditions"`
	Condition   string           `gorm:"type:text" json:"condition"`
	Action      RuleAction       `gorm:"-" json:"action"`
	ActionJSON  string           `gorm:"type:text" json:"-"`
	Priority    int              `gorm:"default:0" json:"priority"`
	IsEnabled   bool             `gorm:"default:true" json:"is_enabled"`
	IsBuiltIn   bool             `gorm:"default:false" json:"is_builtin"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

const (
	OperatorEqual       = "="
	OperatorNotEqual    = "!="
	OperatorGreaterThan = ">"
	OperatorLessThan    = "<"
	OperatorContains    = "contains"
	OperatorIn          = "in"
)

const (
	ActionAllow      = "allow"
	ActionBlock       = "block"
	ActionChallenge   = "challenge"
	ActionMonitor     = "monitor"
	ActionAlert       = "alert"
	ActionLog         = "log"
)
