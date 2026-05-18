package trace

import (
	"testing"
	"time"
)

func TestYOLOServiceStartStop(t *testing.T) {
	service := NewYOLOService()
	
	if service.IsRunning() {
		t.Error("Service should not be running initially")
	}
	
	err := service.Start()
	if err != nil {
		t.Fatalf("Start should not return error: %v", err)
	}
	
	if !service.IsRunning() {
		t.Error("Service should be running after Start()")
	}
	
	service.Stop()
	
	if service.IsRunning() {
		t.Error("Service should not be running after Stop()")
	}
}

func TestYOLOServiceDetectCaptcha(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	request := &YOLODetectionRequest{
		ImageData:     imageData,
		TargetObjects: []string{"person"},
		RequestID:     "test_req_001",
	}
	
	response, err := service.DetectCaptcha(request)
	if err != nil {
		t.Fatalf("DetectCaptcha should not return error: %v", err)
	}
	
	if response == nil {
		t.Error("Response should not be nil")
	}
	
	if !response.Success {
		t.Error("Response should be successful")
	}
	
	if response.RequestID != "test_req_001" {
		t.Errorf("Expected RequestID to be 'test_req_001', got %s", response.RequestID)
	}
}

func TestYOLOServiceDetectBatch(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	requests := make([]YOLODetectionRequest, 3)
	for i := 0; i < 3; i++ {
		imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
		for j := range imageData {
			imageData[j] = byte((i*100 + j) % 256)
		}
		requests[i] = YOLODetectionRequest{
			ImageData:     imageData,
			TargetObjects: []string{"car"},
		}
	}
	
	batchRequest := &YOLOBatchRequest{
		Requests: requests,
	}
	
	response, err := service.DetectBatch(batchRequest)
	if err != nil {
		t.Fatalf("DetectBatch should not return error: %v", err)
	}
	
	if len(response.Responses) != 3 {
		t.Errorf("Expected 3 responses, got %d", len(response.Responses))
	}
}

func TestYOLOServiceVerifyClick(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	request := &YOLODetectionRequest{
		ImageData:     imageData,
		TargetObjects: []string{},
	}
	
	result, err := service.VerifyClick(request, 320, 320)
	if err != nil {
		t.Fatalf("VerifyClick should not return error: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if !result.Success {
		t.Error("Result should be successful")
	}
}

func TestYOLOServiceGetHealthStatus(t *testing.T) {
	service := NewYOLOService()
	
	status := service.GetHealthStatus()
	if status != "stopped" {
		t.Errorf("Expected status 'stopped', got %s", status)
	}
	
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	status = service.GetHealthStatus()
	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", status)
	}
}

func TestYOLOServiceGetServiceStats(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	stats := service.GetServiceStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}
	
	if stats["is_running"] != true {
		t.Error("Expected is_running to be true")
	}
}

func TestYOLOServiceCache(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	request := &YOLODetectionRequest{
		ImageData:     imageData,
		TargetObjects: []string{},
		RequestID:     "cache_test",
	}
	
	_, _ = service.DetectCaptcha(request)
	
	cacheSize := service.GetCacheSize()
	if cacheSize != 1 {
		t.Errorf("Expected cache size 1, got %d", cacheSize)
	}
	
	service.ClearCache()
	cacheSize = service.GetCacheSize()
	if cacheSize != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cacheSize)
	}
}

func TestYOLOServiceSetCacheTTL(t *testing.T) {
	service := NewYOLOService()
	
	ttl := 5 * time.Second
	service.SetCacheTTL(ttl)
	
	service.SetCacheTTL(10 * time.Minute)
}

func TestYOLOServiceLoadModelWeights(t *testing.T) {
	service := NewYOLOService()
	
	err := service.LoadModelWeights("test_weights.path")
	if err != nil {
		t.Errorf("LoadModelWeights should not return error: %v", err)
	}
}

func TestYOLOServiceSetDetectionThresholds(t *testing.T) {
	service := NewYOLOService()
	
	err := service.SetDetectionThresholds(0.5, 0.4)
	if err != nil {
		t.Errorf("SetDetectionThresholds should not return error: %v", err)
	}
	
	err = service.SetDetectionThresholds(1.5, 0.5)
	if err == nil {
		t.Error("SetDetectionThresholds should return error for invalid confidence")
	}
	
	err = service.SetDetectionThresholds(0.5, 1.5)
	if err == nil {
		t.Error("SetDetectionThresholds should return error for invalid IoU")
	}
}

func TestYOLOServiceGetDetector(t *testing.T) {
	service := NewYOLOService()
	
	detector := service.GetDetector()
	if detector == nil {
		t.Error("Detector should not be nil")
	}
}

func TestYOLOServiceDuplicateRequest(t *testing.T) {
	service := NewYOLOService()
	err := service.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer service.Stop()
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	request1 := &YOLODetectionRequest{
		ImageData:     imageData,
		TargetObjects: []string{},
		RequestID:     "duplicate_test",
	}
	
	request2 := &YOLODetectionRequest{
		ImageData:     imageData,
		TargetObjects: []string{},
		RequestID:     "duplicate_test",
	}
	
	response1, err := service.DetectCaptcha(request1)
	if err != nil {
		t.Fatalf("First DetectCaptcha failed: %v", err)
	}
	
	response2, err := service.DetectCaptcha(request2)
	if err != nil {
		t.Fatalf("Second DetectCaptcha failed: %v", err)
	}
	
	if response1.DetectionTime != response2.DetectionTime {
		t.Error("Cached responses should have same detection time")
	}
}