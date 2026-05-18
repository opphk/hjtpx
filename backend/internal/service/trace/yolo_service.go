package trace

import (
	"errors"
	"sync"
	"time"
)

type YOLOService struct {
	mu              sync.RWMutex
	detector        *YOLODetector
	isRunning       bool
	lastHealthCheck time.Time
	healthStatus    string
	detectionCache  map[string]*CaptchaDetectionResult
	cacheTTL        time.Duration
}

type YOLODetectionRequest struct {
	ImageData      []byte
	TargetObjects  []string
	RequestID      string
	Timeout        time.Duration
}

type YOLODetectionResponse struct {
	Success       bool           `json:"success"`
	Objects       []CaptchaObject `json:"objects"`
	DetectionTime time.Duration  `json:"detection_time_ms"`
	RequestID     string         `json:"request_id"`
	Error         string         `json:"error,omitempty"`
}

type YOLOBatchRequest struct {
	Requests []YOLODetectionRequest
}

type YOLOBatchResponse struct {
	Responses []YOLODetectionResponse
	TotalTime time.Duration
}

type ClickVerificationResult struct {
	Success       bool      `json:"success"`
	ClickedObject *CaptchaObject `json:"clicked_object,omitempty"`
	IsValidClick  bool      `json:"is_valid_click"`
	Confidence    float64   `json:"confidence"`
	Error         string    `json:"error,omitempty"`
}

func NewYOLOService() *YOLOService {
	return &YOLOService{
		detector:       NewYOLODetector(),
		isRunning:      false,
		healthStatus:   "not_ready",
		detectionCache: make(map[string]*CaptchaDetectionResult),
		cacheTTL:       30 * time.Second,
	}
}

func (s *YOLOService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return errors.New("YOLO service is already running")
	}

	if err := s.detector.Initialize(); err != nil {
		return err
	}

	s.isRunning = true
	s.healthStatus = "healthy"
	s.lastHealthCheck = time.Now()

	return nil
}

func (s *YOLOService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isRunning = false
	s.healthStatus = "stopped"
}

func (s *YOLOService) DetectCaptcha(request *YOLODetectionRequest) (*YOLODetectionResponse, error) {
	if !s.isRunning {
		return nil, errors.New("YOLO service is not running")
	}

	if len(request.ImageData) == 0 {
		return nil, errors.New("empty image data")
	}

	if request.RequestID == "" {
		request.RequestID = generateRequestID()
	}

	s.mu.RLock()
	cached, exists := s.detectionCache[request.RequestID]
	s.mu.RUnlock()

	if exists {
		return &YOLODetectionResponse{
			Success:       cached.Success,
			Objects:       cached.Objects,
			DetectionTime: cached.DetectionTime,
			RequestID:     request.RequestID,
		}, nil
	}

	result, err := s.detector.DetectCaptcha(request.ImageData, request.TargetObjects)
	if err != nil {
		return &YOLODetectionResponse{
			Success:   false,
			RequestID: request.RequestID,
			Error:     err.Error(),
		}, err
	}

	s.mu.Lock()
	s.detectionCache[request.RequestID] = result
	s.mu.Unlock()

	go s.cleanupCache()

	return &YOLODetectionResponse{
		Success:       result.Success,
		Objects:       result.Objects,
		DetectionTime: result.DetectionTime,
		RequestID:     request.RequestID,
	}, nil
}

func (s *YOLOService) DetectBatch(request *YOLOBatchRequest) (*YOLOBatchResponse, error) {
	if !s.isRunning {
		return nil, errors.New("YOLO service is not running")
	}

	startTime := time.Now()

	responses := make([]YOLODetectionResponse, 0, len(request.Requests))

	for _, req := range request.Requests {
		resp, _ := s.DetectCaptcha(&req)
		responses = append(responses, *resp)
	}

	totalTime := time.Since(startTime)

	return &YOLOBatchResponse{
		Responses: responses,
		TotalTime: totalTime,
	}, nil
}

func (s *YOLOService) VerifyClick(request *YOLODetectionRequest, clickX, clickY float64) (*ClickVerificationResult, error) {
	if !s.isRunning {
		return nil, errors.New("YOLO service is not running")
	}

	if len(request.ImageData) == 0 {
		return nil, errors.New("empty image data")
	}

	object, err := s.detector.DetectPointClick(request.ImageData, clickX, clickY, request.TargetObjects)
	if err != nil {
		return &ClickVerificationResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	if object != nil {
		return &ClickVerificationResult{
			Success:       true,
			ClickedObject: object,
			IsValidClick:  true,
			Confidence:    object.Confidence,
		}, nil
	}

	return &ClickVerificationResult{
		Success:      true,
		IsValidClick: false,
		Confidence:   0.0,
	}, nil
}

func (s *YOLOService) GetHealthStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		return "stopped"
	}

	if time.Since(s.lastHealthCheck) > 5*time.Minute {
		s.performHealthCheck()
	}

	return s.healthStatus
}

func (s *YOLOService) performHealthCheck() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.detector.IsInitialized() {
		s.healthStatus = "healthy"
	} else {
		s.healthStatus = "degraded"
	}

	s.lastHealthCheck = time.Now()
}

func (s *YOLOService) GetServiceStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"is_running":        s.isRunning,
		"health_status":     s.healthStatus,
		"detection_count":   s.detector.GetDetectionCount(),
		"last_detection":    s.detector.GetLastDetectionTime(),
		"model_initialized": s.detector.IsInitialized(),
		"weights_loaded":    s.detector.IsWeightsLoaded(),
		"cache_size":        len(s.detectionCache),
	}
}

func (s *YOLOService) cleanupCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, result := range s.detectionCache {
		if result.DetectionTime > s.cacheTTL {
			delete(s.detectionCache, key)
		}
	}
}

func (s *YOLOService) SetCacheTTL(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
}

func (s *YOLOService) LoadModelWeights(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.detector.LoadWeights(path)
}

func (s *YOLOService) SetDetectionThresholds(confidence, iou float64) error {
	if err := s.detector.SetConfidenceThreshold(confidence); err != nil {
		return err
	}

	if err := s.detector.SetIoUThreshold(iou); err != nil {
		return err
	}

	return nil
}

func generateRequestID() string {
	return "req_" + time.Now().Format("20060102150405") + "_" + randString(8)
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[int(time.Now().UnixNano())%len(letters)]
	}
	return string(result)
}

func (s *YOLOService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

func (s *YOLOService) GetDetector() *YOLODetector {
	return s.detector
}

func (s *YOLOService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detectionCache = make(map[string]*CaptchaDetectionResult)
}

func (s *YOLOService) GetCacheSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.detectionCache)
}