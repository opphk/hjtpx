package behavior

import (
	"time"
)

type EventType string

const (
	EventMouseMove   EventType = "mouse_move"
	EventClick       EventType = "click"
	EventKeyPress    EventType = "key_press"
	EventScroll      EventType = "scroll"
	EventTouchStart  EventType = "touch_start"
	EventTouchMove   EventType = "touch_move"
	EventTouchEnd    EventType = "touch_end"
)

type BehaviorEvent struct {
	Type      EventType `json:"type"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Timestamp int64     `json:"timestamp"`
	Target    string    `json:"target,omitempty"`
	KeyCode   int       `json:"key_code,omitempty"`
}

type SessionMetrics struct {
	SessionID       string            `json:"session_id"`
	Fingerprint     string            `json:"fingerprint"`
	IPAddress       string            `json:"ip_address"`
	UserAgent       string            `json:"user_agent"`
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time,omitempty"`
	TotalDuration   int64             `json:"total_duration_ms"`
	EventCount      int               `json:"event_count"`
	MouseMoveCount  int               `json:"mouse_move_count"`
	ClickCount      int               `json:"click_count"`
	KeyPressCount   int               `json:"key_press_count"`
	ScrollCount     int               `json:"scroll_count"`
	TouchEventCount int               `json:"touch_event_count"`
	Events          []BehaviorEvent   `json:"events,omitempty"`
	Trajectory      []TrajectoryPoint `json:"trajectory"`
}

func NewSessionMetrics(sessionID, fingerprint, ipAddress, userAgent string) *SessionMetrics {
	return &SessionMetrics{
		SessionID:   sessionID,
		Fingerprint: fingerprint,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		StartTime:   time.Now(),
		Events:      make([]BehaviorEvent, 0),
		Trajectory:  make([]TrajectoryPoint, 0),
	}
}

func (m *SessionMetrics) AddEvent(event BehaviorEvent) {
	m.Events = append(m.Events, event)
	m.EventCount++

	switch event.Type {
	case EventMouseMove:
		m.MouseMoveCount++
		m.Trajectory = append(m.Trajectory, TrajectoryPoint{
			X: event.X,
			Y: event.Y,
			T: event.Timestamp,
		})
	case EventClick:
		m.ClickCount++
	case EventKeyPress:
		m.KeyPressCount++
	case EventScroll:
		m.ScrollCount++
	case EventTouchStart, EventTouchMove, EventTouchEnd:
		m.TouchEventCount++
	}
}

func (m *SessionMetrics) ToBehaviorData() *BehaviorData {
	return &BehaviorData{
		Trajectory:     m.Trajectory,
		TotalTime:      m.TotalDuration,
		ClickCount:     m.ClickCount,
		ScrollCount:    m.ScrollCount,
		KeyPressCount:  m.KeyPressCount,
		MouseMoveCount: m.MouseMoveCount,
	}
}

func (m *SessionMetrics) Finalize() {
	m.EndTime = time.Now()
	m.TotalDuration = m.EndTime.Sub(m.StartTime).Milliseconds()
}

type DeviceFingerprint struct {
	Fingerprint string    `json:"fingerprint"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	IPAddresses []string  `json:"ip_addresses"`
	IsTrusted   bool      `json:"is_trusted"`
	RiskLevel   string    `json:"risk_level"`
}

func NewDeviceFingerprint(fp string) *DeviceFingerprint {
	now := time.Now()
	return &DeviceFingerprint{
		Fingerprint: fp,
		FirstSeen:   now,
		LastSeen:    now,
		IPAddresses: make([]string, 0),
		IsTrusted:   false,
		RiskLevel:   "unknown",
	}
}

func (d *DeviceFingerprint) AddIP(ip string) {
	for _, existingIP := range d.IPAddresses {
		if existingIP == ip {
			return
		}
	}
	d.IPAddresses = append(d.IPAddresses, ip)
	d.LastSeen = time.Now()
}

func (d *DeviceFingerprint) IsKnownIP(ip string) bool {
	for _, knownIP := range d.IPAddresses {
		if knownIP == ip {
			return true
		}
	}
	return false
}
