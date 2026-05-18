package trace

import (
	"image"
	"testing"
	"time"
)

func TestYOLODetectorInitialization(t *testing.T) {
	detector := NewYOLODetector()
	
	if detector == nil {
		t.Fatal("YOLODetector should not be nil")
	}
	
	if detector.IsInitialized() {
		t.Error("YOLODetector should not be initialized initially")
	}
	
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize should not return error: %v", err)
	}
	
	if !detector.IsInitialized() {
		t.Error("YOLODetector should be initialized after Initialize()")
	}
}

func TestYOLODetectorDetect(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	detections, err := detector.Detect(imageData)
	if err != nil {
		t.Fatalf("Detect should not return error: %v", err)
	}
	
	if detections == nil {
		t.Error("Detections should not be nil")
	}
}

func TestYOLODetectorDetectCaptcha(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	result, err := detector.DetectCaptcha(imageData, []string{"person", "car"})
	if err != nil {
		t.Fatalf("DetectCaptcha should not return error: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	if !result.Success {
		t.Error("Result should be successful")
	}
}

func TestYOLODetectorDetectPointClick(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	for i := range imageData {
		imageData[i] = byte(i % 256)
	}
	
	obj, err := detector.DetectPointClick(imageData, 320, 320, []string{})
	if err != nil {
		t.Fatalf("DetectPointClick should not return error: %v", err)
	}
	
	_ = obj
}

func TestYOLODetectorGetModelInfo(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	info := detector.GetModelInfo()
	if info == nil {
		t.Error("GetModelInfo should not return nil")
	}
	
	if info["model_type"] != "YOLOv5" {
		t.Errorf("Expected model_type to be 'YOLOv5', got %v", info["model_type"])
	}
	
	if info["input_size"] != YOLOInputSize {
		t.Errorf("Expected input_size to be %d, got %v", YOLOInputSize, info["input_size"])
	}
}

func TestYOLODetectorSetThresholds(t *testing.T) {
	detector := NewYOLODetector()
	
	err := detector.SetConfidenceThreshold(0.5)
	if err != nil {
		t.Errorf("SetConfidenceThreshold should not return error: %v", err)
	}
	
	err = detector.SetConfidenceThreshold(1.5)
	if err == nil {
		t.Error("SetConfidenceThreshold should return error for value > 1")
	}
	
	err = detector.SetIoUThreshold(0.5)
	if err != nil {
		t.Errorf("SetIoUThreshold should not return error: %v", err)
	}
	
	err = detector.SetIoUThreshold(-0.1)
	if err == nil {
		t.Error("SetIoUThreshold should return error for negative value")
	}
}

func TestYOLODetectorNonMaxSuppression(t *testing.T) {
	detector := NewYOLODetector()
	
	boxes := []DetectionResult{
		{Box: BoundingBox{Left: 0, Top: 0, Right: 100, Bottom: 100}, Confidence: 0.9},
		{Box: BoundingBox{Left: 10, Top: 10, Right: 110, Bottom: 110}, Confidence: 0.8},
		{Box: BoundingBox{Left: 200, Top: 200, Right: 300, Bottom: 300}, Confidence: 0.7},
	}
	
	result := detector.nonMaxSuppression(boxes)
	if len(result) != 2 {
		t.Errorf("Expected 2 results after NMS, got %d", len(result))
	}
}

func TestYOLODetectorCalculateIoU(t *testing.T) {
	detector := NewYOLODetector()
	
	box1 := BoundingBox{Left: 0, Top: 0, Right: 100, Bottom: 100}
	box2 := BoundingBox{Left: 50, Top: 50, Right: 150, Bottom: 150}
	
	iou := detector.calculateIoU(box1, box2)
	if iou <= 0 || iou >= 1 {
		t.Errorf("Expected IoU between 0 and 1, got %f", iou)
	}
}

func TestYOLODetectorDetectFromImage(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for i := 0; i < 200; i++ {
		for j := 0; j < 200; j++ {
			img.Set(i, j, image.White)
		}
	}
	
	detections, err := detector.DetectFromImage(img)
	if err != nil {
		t.Fatalf("DetectFromImage should not return error: %v", err)
	}
	
	if detections == nil {
		t.Error("Detections should not be nil")
	}
}

func TestYOLODetectorDetectionCount(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	
	for i := 0; i < 5; i++ {
		_, _ = detector.Detect(imageData)
	}
	
	count := detector.GetDetectionCount()
	if count != 5 {
		t.Errorf("Expected detection count to be 5, got %d", count)
	}
}

func TestYOLODetectorLastDetectionTime(t *testing.T) {
	detector := NewYOLODetector()
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	initialTime := detector.GetLastDetectionTime()
	time.Sleep(10 * time.Millisecond)
	
	imageData := make([]byte, YOLOInputSize*YOLOInputSize*3)
	_, _ = detector.Detect(imageData)
	
	lastTime := detector.GetLastDetectionTime()
	if lastTime.Before(initialTime) {
		t.Error("Last detection time should be after initial time")
	}
}

func TestYOLODetectorIsWeightsLoaded(t *testing.T) {
	detector := NewYOLODetector()
	
	if detector.IsWeightsLoaded() {
		t.Error("IsWeightsLoaded should be false initially")
	}
	
	err := detector.LoadWeights("test_path")
	if err != nil {
		t.Errorf("LoadWeights should not return error: %v", err)
	}
	
	if !detector.IsWeightsLoaded() {
		t.Error("IsWeightsLoaded should be true after LoadWeights")
	}
}

func TestYOLODetectorSigmoid(t *testing.T) {
	detector := NewYOLODetector()
	
	result := detector.sigmoid(0)
	if result != 0.5 {
		t.Errorf("Expected sigmoid(0) = 0.5, got %f", result)
	}
	
	result = detector.sigmoid(10)
	if result < 0.999 {
		t.Errorf("Expected sigmoid(10) > 0.999, got %f", result)
	}
	
	result = detector.sigmoid(-10)
	if result > 0.001 {
		t.Errorf("Expected sigmoid(-10) < 0.001, got %f", result)
	}
}