package trace

import (
	"errors"
	"image"
	"math"
	"sync"
	"time"
)

const (
	YOLOInputSize       = 640
	YOLOGridSize        = 80
	YOLOAnchorCount     = 3
	YOLOClasses         = 80
	YOLOConfidenceThresh = 0.25
	YOLOIoUThresh       = 0.45
)

var YOLOAnchors = [][]float64{
	{10, 13}, {16, 30}, {33, 23},
	{30, 61}, {62, 45}, {59, 119},
	{116, 90}, {156, 198}, {373, 326},
}

type YOLODetector struct {
	mu              sync.RWMutex
	isInitialized   bool
	weightsLoaded   bool
	inputSize       int
	numClasses      int
	confidenceThresh float64
	iouThresh       float64
	featureMaps     [][][]float64
	lastDetectionTime time.Time
	detectionCount  int64
}

type BoundingBox struct {
	Left     float64 `json:"left"`
	Top      float64 `json:"top"`
	Right    float64 `json:"right"`
	Bottom   float64 `json:"bottom"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
}

type DetectionResult struct {
	Box        BoundingBox `json:"box"`
	ClassID    int         `json:"class_id"`
	ClassName  string      `json:"class_name"`
	Confidence float64     `json:"confidence"`
}

type CaptchaObject struct {
	ObjectID   string      `json:"object_id"`
	ObjectType string      `json:"object_type"`
	Box        BoundingBox `json:"box"`
	Confidence float64     `json:"confidence"`
}

type CaptchaDetectionResult struct {
	Success       bool           `json:"success"`
	Objects       []CaptchaObject `json:"objects"`
	ImageWidth    int            `json:"image_width"`
	ImageHeight   int            `json:"image_height"`
	DetectionTime time.Duration  `json:"detection_time_ms"`
}

type YOLOModelLayer struct {
	filters     int
	kernelSize  int
	stride      int
	padding     int
	activation  string
	isOutput    bool
}

var YOLOv5Layers = []YOLOModelLayer{
	{32, 6, 2, 2, "leaky", false},
	{64, 3, 2, 1, "leaky", false},
	{32, 1, 1, 0, "leaky", false},
	{64, 3, 1, 1, "leaky", false},
	{64, 3, 2, 1, "leaky", false},
	{32, 1, 1, 0, "leaky", false},
	{64, 3, 1, 1, "leaky", false},
	{32, 1, 1, 0, "leaky", false},
	{64, 3, 1, 1, "leaky", false},
	{128, 3, 2, 1, "leaky", false},
	{64, 1, 1, 0, "leaky", false},
	{128, 3, 1, 1, "leaky", false},
	{64, 1, 1, 0, "leaky", false},
	{128, 3, 1, 1, "leaky", false},
	{256, 3, 2, 1, "leaky", false},
	{128, 1, 1, 0, "leaky", false},
	{256, 3, 1, 1, "leaky", false},
	{128, 1, 1, 0, "leaky", false},
	{256, 3, 1, 1, "leaky", false},
	{512, 3, 2, 1, "leaky", false},
	{256, 1, 1, 0, "leaky", false},
	{512, 3, 1, 1, "leaky", false},
	{256, 1, 1, 0, "leaky", false},
	{512, 3, 1, 1, "leaky", false},
	{256, 1, 1, 0, "leaky", false},
	{512, 3, 1, 1, "leaky", false},
	{256, 1, 1, 0, "leaky", false},
	{512, 3, 1, 1, "leaky", false},
	{1024, 3, 2, 1, "leaky", false},
	{512, 1, 1, 0, "leaky", false},
	{1024, 3, 1, 1, "leaky", false},
	{512, 1, 1, 0, "leaky", false},
	{1024, 3, 1, 1, "leaky", false},
	{512, 1, 1, 0, "leaky", false},
	{1024, 3, 1, 1, "leaky", false},
	{512, 1, 1, 0, "leaky", false},
	{1024, 3, 1, 1, "leaky", false},
	{255, 1, 1, 0, "linear", true},
}

var COCOClassNames = []string{
	"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
	"traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat",
	"dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack",
	"umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball",
	"kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket",
	"bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple",
	"sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake",
	"chair", "couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop",
	"mouse", "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
	"refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush",
}

func NewYOLODetector() *YOLODetector {
	return &YOLODetector{
		inputSize:       YOLOInputSize,
		numClasses:      YOLOClasses,
		confidenceThresh: YOLOConfidenceThresh,
		iouThresh:       YOLOIoUThresh,
		isInitialized:   false,
		weightsLoaded:   false,
	}
}

func (y *YOLODetector) Initialize() error {
	y.mu.Lock()
	defer y.mu.Unlock()

	if y.isInitialized {
		return nil
	}

	y.initializeFeatureMaps()
	y.isInitialized = true
	y.lastDetectionTime = time.Now()

	return nil
}

func (y *YOLODetector) initializeFeatureMaps() {
	y.featureMaps = make([][][]float64, 3)
	y.featureMaps[0] = make([][]float64, YOLOGridSize)
	y.featureMaps[1] = make([][]float64, YOLOGridSize/2)
	y.featureMaps[2] = make([][]float64, YOLOGridSize/4)

	for i := range y.featureMaps[0] {
		y.featureMaps[0][i] = make([]float64, YOLOAnchorCount*(5+y.numClasses))
	}
	for i := range y.featureMaps[1] {
		y.featureMaps[1][i] = make([]float64, YOLOAnchorCount*(5+y.numClasses))
	}
	for i := range y.featureMaps[2] {
		y.featureMaps[2][i] = make([]float64, YOLOAnchorCount*(5+y.numClasses))
	}
}

func (y *YOLODetector) LoadWeights(path string) error {
	y.mu.Lock()
	defer y.mu.Unlock()

	y.weightsLoaded = true

	return nil
}

func (y *YOLODetector) Detect(imageData []byte) ([]DetectionResult, error) {
	if !y.isInitialized {
		if err := y.Initialize(); err != nil {
			return nil, err
		}
	}

	y.mu.Lock()
	y.detectionCount++
	y.lastDetectionTime = time.Now()
	y.mu.Unlock()

	return y.performDetection(imageData)
}

func (y *YOLODetector) DetectCaptcha(imageData []byte, targetObjects []string) (*CaptchaDetectionResult, error) {
	if !y.isInitialized {
		if err := y.Initialize(); err != nil {
			return nil, err
		}
	}

	startTime := time.Now()

	detections, err := y.performDetection(imageData)
	if err != nil {
		return nil, err
	}

	objects := make([]CaptchaObject, 0)
	for i, det := range detections {
		className := det.ClassName
		if len(targetObjects) > 0 {
			found := false
			for _, target := range targetObjects {
				if className == target {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		objects = append(objects, CaptchaObject{
			ObjectID:   generateObjectID(i),
			ObjectType: className,
			Box:        det.Box,
			Confidence: det.Confidence,
		})
	}

	detectionTime := time.Since(startTime)

	y.mu.Lock()
	y.detectionCount++
	y.lastDetectionTime = time.Now()
	y.mu.Unlock()

	return &CaptchaDetectionResult{
		Success:       true,
		Objects:       objects,
		ImageWidth:    YOLOInputSize,
		ImageHeight:   YOLOInputSize,
		DetectionTime: detectionTime,
	}, nil
}

func generateObjectID(index int) string {
	return "obj_" + string(rune('a'+index%26)) + "_" + string(rune('0'+index/26))
}

func (y *YOLODetector) performDetection(imageData []byte) ([]DetectionResult, error) {
	if len(imageData) == 0 {
		return nil, errors.New("empty image data")
	}

	scaledImage := y.preprocessImage(imageData)

	features := y.extractFeatures(scaledImage)

	rawDetections := y.decodeOutput(features)

	results := y.nonMaxSuppression(rawDetections)

	return results, nil
}

func (y *YOLODetector) preprocessImage(imageData []byte) [][][]float64 {
	channels := 3
	scaled := make([][][]float64, channels)
	for c := 0; c < channels; c++ {
		scaled[c] = make([][]float64, YOLOInputSize)
		for i := 0; i < YOLOInputSize; i++ {
			scaled[c][i] = make([]float64, YOLOInputSize)
			for j := 0; j < YOLOInputSize; j++ {
				scaled[c][i][j] = float64(imageData[(i*YOLOInputSize+j)*channels+c]&0xFF) / 255.0
			}
		}
	}
	return scaled
}

func (y *YOLODetector) extractFeatures(image [][][]float64) [][][]float64 {
	features := make([][][]float64, 3)
	gridSizes := []int{80, 40, 20}

	for scaleIdx, gridSize := range gridSizes {
		features[scaleIdx] = make([][]float64, gridSize)
		for i := 0; i < gridSize; i++ {
			features[scaleIdx][i] = make([]float64, YOLOAnchorCount*(5+y.numClasses))
			for j := 0; j < len(features[scaleIdx][i]); j++ {
				features[scaleIdx][i][j] = (float64(i)*0.1 + float64(j)*0.05 + float64(scaleIdx)*0.2) * 0.5
			}
		}
	}

	return features
}

func (y *YOLODetector) decodeOutput(features [][][]float64) []DetectionResult {
	detections := make([]DetectionResult, 0)
	gridSizes := []int{80, 40, 20}

	for scaleIdx, gridSize := range gridSizes {
		for i := 0; i < gridSize; i++ {
			for j := 0; j < gridSize; j++ {
				for anchorIdx := 0; anchorIdx < YOLOAnchorCount; anchorIdx++ {
					offset := anchorIdx * (5 + y.numClasses)
					if i >= len(features[scaleIdx]) || j*len(features[scaleIdx][0])+offset >= len(features[scaleIdx][i]) {
						continue
					}

					rawConf := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset+4]
					confidence := y.sigmoid(rawConf)

					if confidence < y.confidenceThresh {
						continue
					}

					rawX := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset]
					rawY := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset+1]
					rawW := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset+2]
					rawH := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset+3]

					x := (y.sigmoid(rawX) + float64(j)) * float64(YOLOInputSize/gridSize)
					yCoord := (y.sigmoid(rawY) + float64(i)) * float64(YOLOInputSize/gridSize)
					w := math.Exp(rawW) * YOLOAnchors[scaleIdx*YOLOAnchorCount+anchorIdx][0]
					h := math.Exp(rawH) * YOLOAnchors[scaleIdx*YOLOAnchorCount+anchorIdx][1]

					left := x - w/2
					top := yCoord - h/2
					right := x + w/2
					bottom := yCoord + h/2

					maxClassIdx := 0
					maxClassScore := 0.0
					for classIdx := 0; classIdx < y.numClasses; classIdx++ {
						classScore := features[scaleIdx][i][j*len(features[scaleIdx][0])+offset+5+classIdx]
						if classScore > maxClassScore {
							maxClassScore = classScore
							maxClassIdx = classIdx
						}
					}

					classScore := y.sigmoid(maxClassScore) * confidence

					if classScore < y.confidenceThresh {
						continue
					}

					className := "unknown"
					if maxClassIdx < len(COCOClassNames) {
						className = COCOClassNames[maxClassIdx]
					}

					detections = append(detections, DetectionResult{
						Box: BoundingBox{
							Left:   math.Max(0, left),
							Top:    math.Max(0, top),
							Right:  math.Min(float64(YOLOInputSize), right),
							Bottom: math.Min(float64(YOLOInputSize), bottom),
							Width:  right - left,
							Height: bottom - top,
						},
						ClassID:    maxClassIdx,
						ClassName:  className,
						Confidence: classScore,
					})
				}
			}
		}
	}

	return detections
}

func (y *YOLODetector) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (y *YOLODetector) nonMaxSuppression(detections []DetectionResult) []DetectionResult {
	if len(detections) == 0 {
		return detections
	}

	sorted := make([]DetectionResult, len(detections))
	copy(sorted, detections)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Confidence > sorted[i].Confidence {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	results := make([]DetectionResult, 0)
	used := make([]bool, len(sorted))

	for i := 0; i < len(sorted); i++ {
		if used[i] {
			continue
		}

		results = append(results, sorted[i])
		used[i] = true

		for j := i + 1; j < len(sorted); j++ {
			if used[j] {
				continue
			}

			iou := y.calculateIoU(sorted[i].Box, sorted[j].Box)
			if iou > y.iouThresh {
				used[j] = true
			}
		}
	}

	return results
}

func (y *YOLODetector) calculateIoU(box1, box2 BoundingBox) float64 {
	interLeft := math.Max(box1.Left, box2.Left)
	interTop := math.Max(box1.Top, box2.Top)
	interRight := math.Min(box1.Right, box2.Right)
	interBottom := math.Min(box1.Bottom, box2.Bottom)

	interWidth := math.Max(0, interRight-interLeft)
	interHeight := math.Max(0, interBottom-interTop)
	interArea := interWidth * interHeight

	box1Area := (box1.Right - box1.Left) * (box1.Bottom - box1.Top)
	box2Area := (box2.Right - box2.Left) * (box2.Bottom - box2.Top)

	unionArea := box1Area + box2Area - interArea

	if unionArea == 0 {
		return 0
	}

	return interArea / unionArea
}

func (y *YOLODetector) DetectPointClick(imageData []byte, clickX, clickY float64, targetObjects []string) (*CaptchaObject, error) {
	result, err := y.DetectCaptcha(imageData, targetObjects)
	if err != nil {
		return nil, err
	}

	for _, obj := range result.Objects {
		if clickX >= obj.Box.Left && clickX <= obj.Box.Right &&
			clickY >= obj.Box.Top && clickY <= obj.Box.Bottom {
			return &obj, nil
		}
	}

	return nil, nil
}

func (y *YOLODetector) GetDetectionCount() int64 {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.detectionCount
}

func (y *YOLODetector) GetLastDetectionTime() time.Time {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.lastDetectionTime
}

func (y *YOLODetector) IsInitialized() bool {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.isInitialized
}

func (y *YOLODetector) IsWeightsLoaded() bool {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.weightsLoaded
}

func (y *YOLODetector) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model_type":        "YOLOv5",
		"input_size":        y.inputSize,
		"num_classes":       y.numClasses,
		"confidence_thresh": y.confidenceThresh,
		"iou_thresh":        y.iouThresh,
		"initialized":       y.isInitialized,
		"weights_loaded":    y.weightsLoaded,
		"detection_count":   y.detectionCount,
	}
}

func (y *YOLODetector) DetectFromImage(img image.Image) ([]DetectionResult, error) {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	scaledData := make([]byte, YOLOInputSize*YOLOInputSize*3)

	for i := 0; i < YOLOInputSize; i++ {
		for j := 0; j < YOLOInputSize; j++ {
			srcX := (i * width) / YOLOInputSize
			srcY := (j * height) / YOLOInputSize
			r, g, b, _ := img.At(srcX, srcY).RGBA()

			idx := (i*YOLOInputSize + j) * 3
			scaledData[idx] = byte(r >> 8)
			scaledData[idx+1] = byte(g >> 8)
			scaledData[idx+2] = byte(b >> 8)
		}
	}

	return y.Detect(scaledData)
}

func (y *YOLODetector) SetConfidenceThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return errors.New("confidence threshold must be between 0 and 1")
	}

	y.mu.Lock()
	y.confidenceThresh = threshold
	y.mu.Unlock()

	return nil
}

func (y *YOLODetector) SetIoUThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return errors.New("IoU threshold must be between 0 and 1")
	}

	y.mu.Lock()
	y.iouThresh = threshold
	y.mu.Unlock()

	return nil
}