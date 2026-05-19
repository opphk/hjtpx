package model

import "time"

type CanvasFeatureType string

const (
	CanvasFeatureText         CanvasFeatureType = "text"
	CanvasFeatureGradient     CanvasFeatureType = "gradient"
	CanvasFeatureBezierCurve  CanvasFeatureType = "bezier_curve"
	CanvasFeatureArc          CanvasFeatureType = "arc"
	CanvasFeatureImage        CanvasFeatureType = "image"
	CanvasFeatureShadow       CanvasFeatureType = "shadow"
	CanvasFeatureFilter       CanvasFeatureType = "filter"
	CanvasFeatureComposite    CanvasFeatureType = "composite"
)

type CanvasFeature struct {
	Type       CanvasFeatureType `json:"type"`
	Name       string            `json:"name"`
	DataHash   string            `json:"data_hash"`
	Properties map[string]string `json:"properties,omitempty"`
}

type CanvasTextFingerprint struct {
	TextContent string   `json:"text_content"`
	TextHash    string   `json:"text_hash"`
	FontFamily  string   `json:"font_family"`
	FontSize    int      `json:"font_size"`
	FontWeight  string   `json:"font_weight"`
	FillStyle   string   `json:"fill_style"`
	StrokeStyle string   `json:"stroke_style"`
	Features    []string `json:"features"`
}

type CanvasImageData struct {
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	DataHash      string   `json:"data_hash"`
	PixelCount    int      `json:"pixel_count"`
	ColorDistribution []ColorInfo `json:"color_distribution"`
	Histogram     []int    `json:"histogram"`
	Entropy       float64  `json:"entropy"`
}

type ColorInfo struct {
	Color     string  `json:"color"`
	R         int     `json:"r"`
	G         int     `json:"g"`
	B         int     `json:"b"`
	A         int     `json:"a"`
	Count     int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CanvasFingerprintStability struct {
	FingerprintID     string    `json:"fingerprint_id"`
	SessionID         string    `json:"session_id"`
	FirstSeen         time.Time `json:"first_seen"`
	LastSeen          time.Time `json:"last_seen"`
	HitCount          int       `json:"hit_count"`
	StabilityScore    float64   `json:"stability_score"`
	Variations        []string  `json:"variations"`
	IsStable          bool      `json:"is_stable"`
	Confidence        float64   `json:"confidence"`
}

type CanvasRenderAnalysis struct {
	CanvasHash          string             `json:"canvas_hash"`
	Features            []CanvasFeature    `json:"features"`
	TextFingerprint     *CanvasTextFingerprint `json:"text_fingerprint,omitempty"`
	ImageData           *CanvasImageData   `json:"image_data,omitempty"`
	RenderTime          int64              `json:"render_time_ms"`
	ComplexityScore     float64            `json:"complexity_score"`
	UniquenessScore     float64            `json:"uniqueness_score"`
	Anomalies           []CanvasAnomaly    `json:"anomalies,omitempty"`
}

type CanvasAnomaly struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
}

type CanvasFingerprintComparison struct {
	Hash1            string  `json:"hash1"`
	Hash2            string  `json:"hash2"`
	Similarity       float64 `json:"similarity"`
	CommonFeatures   []string `json:"common_features"`
	DifferentFeatures []string `json:"different_features"`
	IsSameDevice     bool    `json:"is_same_device"`
	Confidence       float64 `json:"confidence"`
}

type CanvasEnhancementConfig struct {
	EnableTextFingerprint bool     `json:"enable_text_fingerprint"`
	EnableImageAnalysis  bool     `json:"enable_image_analysis"`
	EnableStabilityTrack bool     `json:"enable_stability_track"`
	EnableAnomalyDetect  bool     `json:"enable_anomaly_detect"`
	SampleTexts         []string `json:"sample_texts"`
	ImageWidth          int      `json:"image_width"`
	ImageHeight         int      `json:"image_height"`
	StabilityThreshold  float64  `json:"stability_threshold"`
	AnomalyThreshold    float64  `json:"anomaly_threshold"`
}

type CanvasFingerprintResult struct {
	Success            bool                   `json:"success"`
	Fingerprint        string                 `json:"fingerprint"`
	EnhancedFeatures   map[string]interface{} `json:"enhanced_features"`
	RenderAnalysis     *CanvasRenderAnalysis  `json:"render_analysis,omitempty"`
	StabilityAnalysis  *CanvasStabilityResult `json:"stability_analysis,omitempty"`
	Anomalies          []CanvasAnomaly        `json:"anomalies,omitempty"`
	RiskLevel          string                 `json:"risk_level"`
	RiskScore          float64                `json:"risk_score"`
	Error              string                 `json:"error,omitempty"`
}

type CanvasStabilityResult struct {
	DeviceID          string    `json:"device_id"`
	SessionCount      int       `json:"session_count"`
	UniqueFingerprints int      `json:"unique_fingerprints"`
	StabilityScore    float64   `json:"stability_score"`
	IsTrusted         bool      `json:"is_trusted"`
	FirstSeen         time.Time `json:"first_seen"`
	LastSeen          time.Time `json:"last_seen"`
}

type CanvasFeatureVector struct {
	HashComponents   []string  `json:"hash_components"`
	NumericFeatures  []float64 `json:"numeric_features"`
	TextFeatures     []string  `json:"text_features"`
	RenderFeatures   []string  `json:"render_features"`
	FinalHash        string    `json:"final_hash"`
}

type CanvasAntiFingerprintResult struct {
	IsSpoofed         bool     `json:"is_spoofed"`
	SpoofingIndicators []string `json:"spoofing_indicators"`
	ConsistencyScore  float64  `json:"consistency_score"`
	Recommendation   string   `json:"recommendation"`
}
