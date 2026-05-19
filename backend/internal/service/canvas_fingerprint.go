package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type CanvasFingerprintService struct {
	config        *model.CanvasEnhancementConfig
	mu            sync.RWMutex
	fingerprintDB map[string]*model.CanvasFingerprintStability
	stabilityMu   sync.RWMutex
}

func NewCanvasFingerprintService() *CanvasFingerprintService {
	return &CanvasFingerprintService{
		config: &model.CanvasEnhancementConfig{
			EnableTextFingerprint: true,
			EnableImageAnalysis:  true,
			EnableStabilityTrack: true,
			EnableAnomalyDetect:  true,
			SampleTexts: []string{
				"Cwm fjord bank glyphs vext quiz, 😀",
				"Hello, World! こんにちは",
				"سلام دنیا مرحبا",
				"🏠🎉⭐",
			},
			ImageWidth:         280,
			ImageHeight:        60,
			StabilityThreshold: 0.8,
			AnomalyThreshold:   0.3,
		},
		fingerprintDB: make(map[string]*model.CanvasFingerprintStability),
	}
}

func NewCanvasFingerprintServiceWithConfig(config *model.CanvasEnhancementConfig) *CanvasFingerprintService {
	if config == nil {
		return NewCanvasFingerprintService()
	}

	if len(config.SampleTexts) == 0 {
		config.SampleTexts = []string{
			"Cwm fjord bank glyphs vext quiz",
			"Hello, World! こんにちは",
			"🏠🎉⭐",
		}
	}

	if config.ImageWidth == 0 {
		config.ImageWidth = 280
	}
	if config.ImageHeight == 0 {
		config.ImageHeight = 60
	}
	if config.StabilityThreshold == 0 {
		config.StabilityThreshold = 0.8
	}
	if config.AnomalyThreshold == 0 {
		config.AnomalyThreshold = 0.3
	}

	return &CanvasFingerprintService{
		config:        config,
		fingerprintDB: make(map[string]*model.CanvasFingerprintStability),
	}
}

func (s *CanvasFingerprintService) GenerateEnhancedFingerprint(info *model.EnvInfo) *model.CanvasFingerprintResult {
	result := &model.CanvasFingerprintResult{
		Success:          true,
		EnhancedFeatures: make(map[string]interface{}),
		RiskLevel:        "low",
		RiskScore:        0.0,
	}

	if info.CanvasFingerprint == "" {
		result.Success = false
		result.Error = "canvas fingerprint is empty"
		result.RiskLevel = "high"
		result.RiskScore = 50.0
		return result
	}

	baseFingerprint := info.CanvasFingerprint

	textFeatures := s.extractTextFeatures(baseFingerprint)
	if textFeatures != nil {
		result.EnhancedFeatures["text_fingerprint"] = textFeatures
	}

	gradientFeatures := s.extractGradientFeatures(baseFingerprint)
	if gradientFeatures != nil {
		result.EnhancedFeatures["gradient_features"] = gradientFeatures
	}

	bezierFeatures := s.extractBezierFeatures(baseFingerprint)
	if bezierFeatures != nil {
		result.EnhancedFeatures["bezier_features"] = bezierFeatures
	}

	arcFeatures := s.extractArcFeatures(baseFingerprint)
	if arcFeatures != nil {
		result.EnhancedFeatures["arc_features"] = arcFeatures
	}

	shadowFeatures := s.extractShadowFeatures(baseFingerprint)
	if shadowFeatures != nil {
		result.EnhancedFeatures["shadow_features"] = shadowFeatures
	}

	compositeFeatures := s.extractCompositeFeatures(baseFingerprint)
	if compositeFeatures != nil {
		result.EnhancedFeatures["composite_features"] = compositeFeatures
	}

	enhancedHash := s.computeEnhancedHash(baseFingerprint, result.EnhancedFeatures)
	result.Fingerprint = enhancedHash

	riskLevel, riskScore := s.analyzeCanvasRisk(result.EnhancedFeatures, info)
	result.RiskLevel = riskLevel
	result.RiskScore = riskScore

	if s.config.EnableAnomalyDetect {
		anomalies := s.detectCanvasAnomalies(enhancedHash, result.EnhancedFeatures)
		result.Anomalies = anomalies
		if len(anomalies) > 0 {
			result.RiskLevel = "medium"
			result.RiskScore += float64(len(anomalies)) * 10.0
		}
	}

	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	return result
}

func (s *CanvasFingerprintService) extractTextFeatures(fingerprint string) *model.CanvasTextFingerprint {
	features := &model.CanvasTextFingerprint{
		TextHash: s.hashString(fingerprint),
		Features: make([]string, 0),
	}

	hasUnicode := false
	for _, r := range fingerprint {
		if r > 127 {
			hasUnicode = true
			break
		}
	}
	if hasUnicode {
		features.Features = append(features.Features, "unicode_present")
	}

	hasEmoji := false
	for _, r := range fingerprint {
		if r >= 0x1F300 && r <= 0x1F9FF {
			hasEmoji = true
			break
		}
	}
	if hasEmoji {
		features.Features = append(features.Features, "emoji_present")
	}

	if len(fingerprint) > 64 {
		features.Features = append(features.Features, "high_entropy")
	}

	hexCharCount := 0
	for _, c := range fingerprint {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hexCharCount++
		}
	}

	if float64(hexCharCount)/float64(len(fingerprint)) > 0.8 {
		features.Features = append(features.Features, "hex_dominant")
	}

	features.TextHash = s.hashString(fingerprint + ":" + strings.Join(features.Features, ","))

	return features
}

func (s *CanvasFingerprintService) extractGradientFeatures(fingerprint string) map[string]string {
	features := make(map[string]string)

	gradientIndicators := []string{
		"gradient_linear",
		"gradient_radial",
		"gradient_conic",
		"gradient_stop",
		"color_transition",
	}

	fingerprintLower := strings.ToLower(fingerprint)
	detected := 0
	for _, indicator := range gradientIndicators {
		if strings.Contains(fingerprintLower, indicator) {
			detected++
		}
	}

	features["gradient_score"] = fmt.Sprintf("%d", detected)
	features["has_gradient"] = "false"
	if detected > 0 {
		features["has_gradient"] = "true"
		features["gradient_type"] = "detected"
	}

	gradientHash := s.hashString(fingerprint + "_gradient")
	features["gradient_hash"] = gradientHash[:16]

	return features
}

func (s *CanvasFingerprintService) extractBezierFeatures(fingerprint string) map[string]string {
	features := make(map[string]string)

	bezierIndicators := []string{
		"bezier_quadratic",
		"bezier_cubic",
		"curve_to",
		"quadratic_curve",
	}

	fingerprintLower := strings.ToLower(fingerprint)
	detected := 0
	for _, indicator := range bezierIndicators {
		if strings.Contains(fingerprintLower, indicator) {
			detected++
		}
	}

	features["bezier_score"] = fmt.Sprintf("%d", detected)
	features["has_bezier"] = "false"
	if detected > 0 {
		features["has_bezier"] = "true"
	}

	bezierHash := s.hashString(fingerprint + "_bezier")
	features["bezier_hash"] = bezierHash[:16]

	return features
}

func (s *CanvasFingerprintService) extractArcFeatures(fingerprint string) map[string]string {
	features := make(map[string]string)

	arcIndicators := []string{
		"arc",
		"arc_to",
		"circle",
		"ellipse",
		"pie",
	}

	fingerprintLower := strings.ToLower(fingerprint)
	detected := 0
	for _, indicator := range arcIndicators {
		if strings.Contains(fingerprintLower, indicator) {
			detected++
		}
	}

	features["arc_score"] = fmt.Sprintf("%d", detected)
	features["has_arc"] = "false"
	if detected > 0 {
		features["has_arc"] = "true"
	}

	arcHash := s.hashString(fingerprint + "_arc")
	features["arc_hash"] = arcHash[:16]

	return features
}

func (s *CanvasFingerprintService) extractShadowFeatures(fingerprint string) map[string]string {
	features := make(map[string]string)

	shadowIndicators := []string{
		"shadow",
		"blur",
		"offset_x",
		"offset_y",
		"shadow_color",
	}

	fingerprintLower := strings.ToLower(fingerprint)
	detected := 0
	for _, indicator := range shadowIndicators {
		if strings.Contains(fingerprintLower, indicator) {
			detected++
		}
	}

	features["shadow_score"] = fmt.Sprintf("%d", detected)
	features["has_shadow"] = "false"
	if detected > 0 {
		features["has_shadow"] = "true"
	}

	shadowHash := s.hashString(fingerprint + "_shadow")
	features["shadow_hash"] = shadowHash[:16]

	return features
}

func (s *CanvasFingerprintService) extractCompositeFeatures(fingerprint string) map[string]string {
	features := make(map[string]string)

	compositeIndicators := []string{
		"source_over",
		"source_atop",
		"destination_over",
		"multiply",
		"screen",
		"overlay",
		"darken",
		"lighten",
		"color_dodge",
		"color_burn",
		"hard_light",
		"soft_light",
		"difference",
		"exclusion",
	}

	fingerprintLower := strings.ToLower(fingerprint)
	detected := make([]string, 0)
	for _, indicator := range compositeIndicators {
		if strings.Contains(fingerprintLower, indicator) {
			detected = append(detected, indicator)
		}
	}

	features["composite_count"] = fmt.Sprintf("%d", len(detected))
	features["has_composite"] = "false"
	if len(detected) > 0 {
		features["has_composite"] = "true"
		features["composite_types"] = strings.Join(detected, ",")
	}

	compositeHash := s.hashString(fingerprint + "_composite")
	features["composite_hash"] = compositeHash[:16]

	return features
}

func (s *CanvasFingerprintService) computeEnhancedHash(baseFingerprint string, features map[string]interface{}) string {
	combined := baseFingerprint

	for key, value := range features {
		if valueMap, ok := value.(map[string]string); ok {
			for k, v := range valueMap {
				combined += fmt.Sprintf("%s:%s:%s", key, k, v)
			}
		} else if textFP, ok := value.(*model.CanvasTextFingerprint); ok {
			combined += fmt.Sprintf("text:%s:%s", textFP.TextHash, strings.Join(textFP.Features, ","))
		}
	}

	h := sha256.New()
	h.Write([]byte(combined))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *CanvasFingerprintService) analyzeCanvasRisk(features map[string]interface{}, info *model.EnvInfo) (string, float64) {
	riskScore := 0.0

	if info.CanvasFingerprint == "" {
		return "high", 100.0
	}

	if len(info.CanvasFingerprint) < 32 {
		riskScore += 20.0
	}

	if len(info.CanvasFingerprint) > 128 {
		riskScore += 15.0
	}

	hasHexOnly := true
	for _, c := range info.CanvasFingerprint {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			hasHexOnly = false
			break
		}
	}
	if hasHexOnly && len(info.CanvasFingerprint) > 32 {
		riskScore += 25.0
	}

	repeatCount := 0
	maxRepeat := 0
	var lastChar rune
	for _, c := range info.CanvasFingerprint {
		if c == lastChar {
			repeatCount++
			if repeatCount > maxRepeat {
				maxRepeat = repeatCount
			}
		} else {
			repeatCount = 0
		}
		lastChar = c
	}
	if maxRepeat > len(info.CanvasFingerprint)/3 {
		riskScore += 30.0
	}

	softwarePatterns := []string{"swiftshader", "llvmpipe", "software", "virtual"}
	for _, pattern := range softwarePatterns {
		if strings.Contains(strings.ToLower(info.WebGLRenderer), pattern) {
			riskScore += 15.0
		}
	}

	if textFeatures, ok := features["text_fingerprint"].(*model.CanvasTextFingerprint); ok {
		if len(textFeatures.Features) == 0 {
			riskScore += 10.0
		}
		if textFeatures.TextHash == "" {
			riskScore += 15.0
		}
	}

	if riskScore < 20 {
		return "low", riskScore
	} else if riskScore < 50 {
		return "medium", riskScore
	}
	return "high", riskScore
}

func (s *CanvasFingerprintService) detectCanvasAnomalies(fingerprint string, features map[string]interface{}) []model.CanvasAnomaly {
	anomalies := make([]model.CanvasAnomaly, 0)

	if len(fingerprint) < 32 {
		anomalies = append(anomalies, model.CanvasAnomaly{
			Type:        "length",
			Severity:    "medium",
			Description: "Canvas指纹长度过短，可能被伪造或简化",
		})
	}

	if len(fingerprint) > 200 {
		anomalies = append(anomalies, model.CanvasAnomaly{
			Type:        "length",
			Severity:    "medium",
			Description: "Canvas指纹长度异常，可能包含额外数据",
		})
	}

	hexCount := 0
	for _, c := range fingerprint {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hexCount++
		}
	}

	if float64(hexCount)/float64(len(fingerprint)) > 0.95 {
		anomalies = append(anomalies, model.CanvasAnomaly{
			Type:        "format",
			Severity:    "low",
			Description: "指纹几乎完全是十六进制，可能缺乏渲染特征",
		})
	}

	uniqueChars := make(map[rune]bool)
	for _, c := range fingerprint {
		uniqueChars[c] = true
	}
	uniqueRatio := float64(len(uniqueChars)) / float64(len(fingerprint))

	if uniqueRatio < 0.1 {
		anomalies = append(anomalies, model.CanvasAnomaly{
			Type:        "entropy",
			Severity:    "high",
			Description: "指纹熵值过低，存在大量重复字符",
		})
	}

	if len(features) == 0 {
		anomalies = append(anomalies, model.CanvasAnomaly{
			Type:        "features",
			Severity:    "medium",
			Description: "未检测到Canvas渲染特征，可能被阻止",
		})
	}

	return anomalies
}

func (s *CanvasFingerprintService) ExtractTextFingerprint(sampleTexts []string) []*model.CanvasTextFingerprint {
	if len(sampleTexts) == 0 {
		sampleTexts = s.config.SampleTexts
	}

	textFingerprints := make([]*model.CanvasTextFingerprint, 0, len(sampleTexts))

	for _, text := range sampleTexts {
		fp := s.generateTextFingerprint(text)
		textFingerprints = append(textFingerprints, fp)
	}

	return textFingerprints
}

func (s *CanvasFingerprintService) generateTextFingerprint(text string) *model.CanvasTextFingerprint {
	fp := &model.CanvasTextFingerprint{
		TextContent: text,
		TextHash:    s.hashString(text),
		Features:    make([]string, 0),
	}

	fontFamilies := []string{"Arial", "Helvetica", "Times New Roman", "Courier New", "Georgia"}
	fp.FontFamily = fontFamilies[abs(int(hashStringToInt(text)%int64(len(fontFamilies))))]

	fontSizes := []int{12, 14, 16, 18, 20, 24}
	fp.FontSize = fontSizes[abs(int(hashStringToInt(text+"size")%int64(len(fontSizes))))]

	fontWeights := []string{"normal", "bold", "lighter", "bolder"}
	fp.FontWeight = fontWeights[abs(int(hashStringToInt(text+"weight")%int64(len(fontWeights))))]

	fillStyles := []string{"#000000", "#333333", "#666666", "#999999"}
	fp.FillStyle = fillStyles[abs(int(hashStringToInt(text+"fill")%int64(len(fillStyles))))]

	strokeStyles := []string{"#ffffff", "#cccccc", "transparent"}
	fp.StrokeStyle = strokeStyles[abs(int(hashStringToInt(text+"stroke")%int64(len(strokeStyles))))]

	if len(text) > 20 {
		fp.Features = append(fp.Features, "long_text")
	}

	runeCount := 0
	for range text {
		runeCount++
	}
	if runeCount != len(text) {
		fp.Features = append(fp.Features, "multi_byte")
	}

	hasSpecialChars := false
	for _, r := range text {
		if r > 127 || (r >= '0' && r <= '9') == false && (r >= 'a' && r <= 'z') == false && (r >= 'A' && r <= 'Z') == false {
			hasSpecialChars = true
			break
		}
	}
	if hasSpecialChars {
		fp.Features = append(fp.Features, "special_chars")
	}

	fp.TextHash = s.hashString(fmt.Sprintf("%s:%s:%d:%s:%s:%s:%s",
		fp.TextContent, fp.FontFamily, fp.FontSize, fp.FontWeight, fp.FillStyle, fp.StrokeStyle,
		strings.Join(fp.Features, ",")))

	return fp
}

func (s *CanvasFingerprintService) AnalyzeImageData(imageData []byte) (*model.CanvasImageData, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	img, err := png.Decode(strings.NewReader(string(imageData)))
	if err != nil {
		img = s.simulateImageAnalysis(imageData)
		if img == nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
	}

	result := &model.CanvasImageData{
		Width:              img.Bounds().Dx(),
		Height:             img.Bounds().Dy(),
		DataHash:           s.hashString(string(imageData)),
		PixelCount:         img.Bounds().Dx() * img.Bounds().Dy(),
		ColorDistribution:  make([]model.ColorInfo, 0),
		Histogram:          make([]int, 256),
		Entropy:            0.0,
	}

	colorCounts := make(map[string]int)
	totalPixels := 0

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pixel := img.At(x, y)
			r, g, b, a := rgbaToInt(pixel)

			colorKey := fmt.Sprintf("%d,%d,%d,%d", r, g, b, a)
			colorCounts[colorKey]++
			totalPixels++

			luminance := int(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
			if luminance < 256 {
				result.Histogram[luminance]++
			}
		}
	}

	if totalPixels > 0 {
		sortedColors := make([]struct {
			key   string
			count int
		}, 0, len(colorCounts))

		for key, count := range colorCounts {
			sortedColors = append(sortedColors, struct {
				key   string
				count int
			}{key, count})
		}

		sort.Slice(sortedColors, func(i, j int) bool {
			return sortedColors[i].count > sortedColors[j].count
		})

		topColors := sortedColors
		if len(topColors) > 20 {
			topColors = sortedColors[:20]
		}

		for _, c := range topColors {
			var r, g, b, a int
			fmt.Sscanf(c.key, "%d,%d,%d,%d", &r, &g, &b, &a)

			result.ColorDistribution = append(result.ColorDistribution, model.ColorInfo{
				Color:     fmt.Sprintf("#%02x%02x%02x", r, g, b),
				R:         r,
				G:         g,
				B:         b,
				A:         a,
				Count:     c.count,
				Percentage: float64(c.count) / float64(totalPixels) * 100,
			})
		}
	}

	result.Entropy = s.calculateImageEntropy(result.Histogram)

	return result, nil
}

func (s *CanvasFingerprintService) simulateImageAnalysis(data []byte) image.Image {
	width := 280
	height := 60

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) % len(data)
			val := int(data[offset])

			r := uint8((val * 13) % 256)
			g := uint8((val * 17) % 256)
			b := uint8((val * 19) % 256)

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img
}

func (s *CanvasFingerprintService) calculateImageEntropy(histogram []int) float64 {
	totalPixels := 0
	for _, count := range histogram {
		totalPixels += count
	}

	if totalPixels == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(totalPixels)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (s *CanvasFingerprintService) AnalyzeStability(fingerprintID string, sessionID string) (*model.CanvasStabilityResult, error) {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	stored, exists := s.fingerprintDB[fingerprintID]
	now := time.Now()

	if !exists {
		stored = &model.CanvasFingerprintStability{
			FingerprintID:  fingerprintID,
			SessionID:      sessionID,
			FirstSeen:      now,
			LastSeen:       now,
			HitCount:       1,
			StabilityScore: 0.0,
			Variations:     make([]string, 0),
			IsStable:       false,
			Confidence:     0.0,
		}
		s.fingerprintDB[fingerprintID] = stored
	} else {
		stored.LastSeen = now
		stored.HitCount++

		if stored.SessionID != sessionID {
			if !containsString(stored.Variations, sessionID) {
				stored.Variations = append(stored.Variations, sessionID)
			}
		}
	}

	stored.StabilityScore = s.calculateStabilityScore(stored)
	stored.IsStable = stored.StabilityScore >= s.config.StabilityThreshold

	if stored.HitCount > 0 && len(stored.Variations) >= 0 {
		stored.Confidence = math.Min(float64(stored.HitCount)*0.1, 1.0)
	}

	return &model.CanvasStabilityResult{
		DeviceID:          fingerprintID,
		SessionCount:      stored.HitCount,
		UniqueFingerprints: len(stored.Variations) + 1,
		StabilityScore:    stored.StabilityScore,
		IsTrusted:         stored.IsStable && stored.Confidence >= 0.8,
		FirstSeen:         stored.FirstSeen,
		LastSeen:          stored.LastSeen,
	}, nil
}

func (s *CanvasFingerprintService) calculateStabilityScore(stored *model.CanvasFingerprintStability) float64 {
	baseScore := 1.0

	hitDecay := math.Max(0, 1.0-float64(stored.HitCount)*0.02)
	baseScore *= hitDecay

	if len(stored.Variations) > 0 {
		variationPenalty := float64(len(stored.Variations)) * 0.15
		baseScore *= math.Max(0.1, 1.0-variationPenalty)
	}

	age := time.Since(stored.FirstSeen)
	ageDays := age.Hours() / 24
	if ageDays > 7 {
		baseScore *= 1.2
		if baseScore > 1.0 {
			baseScore = 1.0
		}
	}

	return math.Max(0, math.Min(1.0, baseScore))
}

func (s *CanvasFingerprintService) CompareFingerprints(hash1, hash2 string) *model.CanvasFingerprintComparison {
	comparison := &model.CanvasFingerprintComparison{
		Hash1:            hash1,
		Hash2:            hash2,
		Similarity:       s.calculateSimilarity(hash1, hash2),
		CommonFeatures:   make([]string, 0),
		DifferentFeatures: make([]string, 0),
		IsSameDevice:     false,
		Confidence:       0.0,
	}

	comparison.IsSameDevice = comparison.Similarity >= 85.0
	comparison.Confidence = comparison.Similarity / 100.0

	return comparison
}

func (s *CanvasFingerprintService) calculateSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 100.0
	}

	if len(hash1) != len(hash2) {
		return 0.0
	}

	if len(hash1) == 0 {
		return 0.0
	}

	matches := 0
	for i := 0; i < len(hash1) && i < len(hash2); i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(hash1)) * 100.0
}

func (s *CanvasFingerprintService) DetectSpoofing(originalHash, receivedHash string, features map[string]interface{}) *model.CanvasAntiFingerprintResult {
	result := &model.CanvasAntiFingerprintResult{
		IsSpoofed:          false,
		SpoofingIndicators: make([]string, 0),
		ConsistencyScore:  100.0,
		Recommendation:    "allow",
	}

	similarity := s.calculateSimilarity(originalHash, receivedHash)

	if similarity < 30.0 {
		result.IsSpoofed = true
		result.SpoofingIndicators = append(result.SpoofingIndicators,
			"指纹哈希相似度过低，可能来自不同设备或被伪造")
		result.ConsistencyScore -= 50.0
	}

	if len(originalHash) != len(receivedHash) {
		result.SpoofingIndicators = append(result.SpoofingIndicators,
			"指纹长度不匹配")
		result.ConsistencyScore -= 20.0
		result.IsSpoofed = true
	}

	originalEntropy := s.calculateStringEntropy(originalHash)
	receivedEntropy := s.calculateStringEntropy(receivedHash)

	if math.Abs(originalEntropy-receivedEntropy) > 2.0 {
		result.SpoofingIndicators = append(result.SpoofingIndicators,
			"指纹熵值差异显著")
		result.ConsistencyScore -= 15.0
	}

	if len(features) == 0 && originalHash != "" {
		result.SpoofingIndicators = append(result.SpoofingIndicators,
			"缺少增强特征，可能被移除")
		result.ConsistencyScore -= 10.0
	}

	if result.ConsistencyScore < 50 {
		result.Recommendation = "block"
		result.IsSpoofed = true
	} else if result.ConsistencyScore < 70 {
		result.Recommendation = "review"
	}

	return result
}

func (s *CanvasFingerprintService) calculateStringEntropy(str string) float64 {
	if len(str) == 0 {
		return 0.0
	}

	charCounts := make(map[byte]int)
	for _, c := range str {
		charCounts[byte(c)]++
	}

	entropy := 0.0
	for _, count := range charCounts {
		p := float64(count) / float64(len(str))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (s *CanvasFingerprintService) GenerateRenderAnalysis(canvasHash string, enhancedFeatures map[string]interface{}) *model.CanvasRenderAnalysis {
	analysis := &model.CanvasRenderAnalysis{
		CanvasHash:      canvasHash,
		Features:        make([]model.CanvasFeature, 0),
		ComplexityScore: 0.0,
		UniquenessScore: 0.0,
		Anomalies:       make([]model.CanvasAnomaly, 0),
	}

	for featureType, featureName := range map[model.CanvasFeatureType]string{
		model.CanvasFeatureText:         "text",
		model.CanvasFeatureGradient:     "gradient",
		model.CanvasFeatureBezierCurve:  "bezier",
		model.CanvasFeatureArc:          "arc",
		model.CanvasFeatureShadow:       "shadow",
		model.CanvasFeatureComposite:    "composite",
	} {
		featureKey := fmt.Sprintf("%s_features", featureName)
		if _, ok := enhancedFeatures[featureKey]; ok {
			analysis.Features = append(analysis.Features, model.CanvasFeature{
				Type:       featureType,
				Name:       featureName,
				DataHash:   s.hashString(canvasHash + featureName),
				Properties: make(map[string]string),
			})
			analysis.ComplexityScore += 10.0
		}
	}

	if textFP, ok := enhancedFeatures["text_fingerprint"].(*model.CanvasTextFingerprint); ok {
		analysis.TextFingerprint = textFP
		analysis.ComplexityScore += 15.0
		analysis.UniquenessScore += 10.0
	}

	analysis.UniquenessScore = s.calculateUniquenessScore(analysis.Features, canvasHash)

	if analysis.ComplexityScore > 100 {
		analysis.ComplexityScore = 100
	}
	if analysis.UniquenessScore > 100 {
		analysis.UniquenessScore = 100
	}

	return analysis
}

func (s *CanvasFingerprintService) calculateUniquenessScore(features []model.CanvasFeature, hash string) float64 {
	score := 0.0

	score += float64(len(features)) * 8.0

	if len(hash) > 64 {
		score += 15.0
	}

	uniqueChars := make(map[byte]bool)
	for _, c := range hash {
		uniqueChars[byte(c)] = true
	}
	score += float64(len(uniqueChars)) * 0.5

	return math.Min(score, 100.0)
}

func (s *CanvasFingerprintService) ExtractImageDataFromCanvas(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gradientValue := uint8((x + y) % 256)
			img.Set(x, y, color.RGBA{
				R: gradientValue,
				G: uint8((int(gradientValue) * 7) % 256),
				B: uint8((int(gradientValue) * 13) % 256),
				A: 255,
			})
		}
	}

	var buf []byte
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.At(x, y)
			r, g, b, a := rgbaToInt(pixel)
			buf = append(buf, byte(r), byte(g), byte(b), byte(a))
		}
	}

	return buf, nil
}

func (s *CanvasFingerprintService) hashString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func hashStringToInt(str string) int64 {
	h := fnv.New64a()
	h.Write([]byte(str))
	return int64(h.Sum64())
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func rgbaToInt(c color.Color) (r, g, b, a int) {
	rr, gg, bb, aa := c.RGBA()
	return int(rr), int(gg), int(bb), int(aa)
}

type SimulatedCanvasRenderer struct {
	width  int
	height int
	data   []byte
}

func NewSimulatedCanvasRenderer(width, height int) *SimulatedCanvasRenderer {
	return &SimulatedCanvasRenderer{
		width:  width,
		height: height,
		data:   make([]byte, width*height*4),
	}
}

func (r *SimulatedCanvasRenderer) FillText(text string, x, y int, font string, size int) {
	for i, char := range text {
		offset := (y*256 + x + i) % len(r.data)
		r.data[offset] = byte(char % 256)
		r.data[offset+1] = byte((char >> 8) % 256)
	}
}

func (r *SimulatedCanvasRenderer) DrawGradient(x1, y1, x2, y2 int, color1, color2 [4]byte) {
	for i := 0; i < r.height*r.width*4; i += 4 {
		t := float64(i) / float64(len(r.data))
		r.data[i] = byte(float64(color1[0])*(1-t) + float64(color2[0])*t)
		r.data[i+1] = byte(float64(color1[1])*(1-t) + float64(color2[1])*t)
		r.data[i+2] = byte(float64(color1[2])*(1-t) + float64(color2[2])*t)
		r.data[i+3] = 255
	}
}

func (r *SimulatedCanvasRenderer) DrawBezierCurve(p0, p1, p2, p3 [2]int) {
	for t := 0.0; t <= 1.0; t += 0.01 {
		x := int(math.Pow(1-t, 3)*float64(p0[0]) + 3*math.Pow(1-t, 2)*t*float64(p1[0]) +
			3*(1-t)*math.Pow(t, 2)*float64(p2[0]) + math.Pow(t, 3)*float64(p3[0]))
		y := int(math.Pow(1-t, 3)*float64(p0[1]) + 3*math.Pow(1-t, 2)*t*float64(p1[1]) +
			3*(1-t)*math.Pow(t, 2)*float64(p2[1]) + math.Pow(t, 3)*float64(p3[1]))

		offset := (y*256 + x) % len(r.data)
		r.data[offset] = 255
		r.data[offset+1] = 255
		r.data[offset+2] = 255
	}
}

func (r *SimulatedCanvasRenderer) DrawArc(cx, cy, radius int, startAngle, endAngle float64) {
	for angle := startAngle; angle <= endAngle; angle += 0.01 {
		x := cx + int(float64(radius)*math.Cos(angle))
		y := cy + int(float64(radius)*math.Sin(angle))

		offset := (y*256 + x) % len(r.data)
		if offset >= 0 && offset+2 < len(r.data) {
			r.data[offset] = 200
			r.data[offset+1] = 200
			r.data[offset+2] = 200
		}
	}
}

func (r *SimulatedCanvasRenderer) ApplyShadow(offsetX, offsetY, blur int, color [4]byte) {
	for i := 0; i < len(r.data); i += 4 {
		shadowOffset := ((i/4 + offsetY*256 + offsetX) * 4) % len(r.data)
		if shadowOffset >= 0 && shadowOffset+3 < len(r.data) {
			r.data[shadowOffset] = color[0]
			r.data[shadowOffset+1] = color[1]
			r.data[shadowOffset+2] = color[2]
			r.data[shadowOffset+3] = color[3]
		}
	}
}

func (r *SimulatedCanvasRenderer) Composite(operation string) {
	switch operation {
	case "multiply":
		for i := 0; i < len(r.data); i += 4 {
			r.data[i] = byte((int(r.data[i]) * 128) / 255)
			r.data[i+1] = byte((int(r.data[i+1]) * 128) / 255)
			r.data[i+2] = byte((int(r.data[i+2]) * 128) / 255)
		}
	case "screen":
		for i := 0; i < len(r.data); i += 4 {
			r.data[i] = byte(255 - ((255-int(r.data[i]))*(255-128))/255)
			r.data[i+1] = byte(255 - ((255-int(r.data[i+1]))*(255-128))/255)
			r.data[i+2] = byte(255 - ((255-int(r.data[i+2]))*(255-128))/255)
		}
	}
}

func (r *SimulatedCanvasRenderer) GetImageData() []byte {
	return r.data
}

func (r *SimulatedCanvasRenderer) GetHash() string {
	h := sha256.New()
	h.Write(r.data)
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateSimulatedCanvasFingerprint(userAgent string, screenWidth, screenHeight int) string {
	renderer := NewSimulatedCanvasRenderer(280, 60)

	renderer.FillText("Cwm fjord bank glyphs", 10, 20, "Arial", 14)
	renderer.FillText("Hello World!", 10, 40, "Helvetica", 16)

	renderer.DrawGradient(0, 0, 280, 60, [4]byte{0, 100, 200, 255}, [4]byte{200, 100, 0, 255})

	renderer.DrawBezierCurve([2]int{10, 30}, [2]int{50, 10}, [2]int{100, 50}, [2]int{140, 30})

	renderer.DrawArc(200, 30, 20, 0, math.Pi*2)

	renderer.ApplyShadow(2, 2, 5, [4]byte{0, 0, 0, 100})

	renderer.Composite("multiply")

	baseHash := renderer.GetHash()

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%dx%d:%d",
		baseHash, userAgent, screenWidth, screenHeight, time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))
}

func SimulateTextFingerprintRendering() (string, error) {
	renderer := NewSimulatedCanvasRenderer(280, 60)

	sampleTexts := []string{
		"Cwm fjord bank glyphs vext quiz, 😀",
		"Hello, World! こんにちは",
		"سلام دنیا مرحبا",
		"🏠🎉⭐",
	}

	for i, text := range sampleTexts {
		renderer.FillText(text, 10, 15+i*15, "Arial", 12)
	}

	hash := renderer.GetHash()
	return hash, nil
}

func SimulateImageDataExtraction() ([]byte, error) {
	renderer := NewSimulatedCanvasRenderer(280, 60)

	renderer.DrawGradient(0, 0, 280, 60, [4]byte{100, 150, 200, 255}, [4]byte{50, 100, 150, 255})

	for i := 0; i < 10; i++ {
		x := rand.Intn(260) + 10
		y := rand.Intn(40) + 10
		renderer.FillText(fmt.Sprintf("●"), x, y, "Arial", 16)
	}

	return renderer.GetImageData(), nil
}

func (s *CanvasFingerprintService) GetConfig() *model.CanvasEnhancementConfig {
	return s.config
}

func (s *CanvasFingerprintService) UpdateConfig(config *model.CanvasEnhancementConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.EnableTextFingerprint = config.EnableTextFingerprint
	if config.EnableImageAnalysis {
		s.config.EnableImageAnalysis = config.EnableImageAnalysis
	}
	if config.EnableStabilityTrack {
		s.config.EnableStabilityTrack = config.EnableStabilityTrack
	}
	if config.EnableAnomalyDetect {
		s.config.EnableAnomalyDetect = config.EnableAnomalyDetect
	}
	if len(config.SampleTexts) > 0 {
		s.config.SampleTexts = config.SampleTexts
	}
	if config.ImageWidth > 0 {
		s.config.ImageWidth = config.ImageWidth
	}
	if config.ImageHeight > 0 {
		s.config.ImageHeight = config.ImageHeight
	}
	if config.StabilityThreshold > 0 {
		s.config.StabilityThreshold = config.StabilityThreshold
	}
	if config.AnomalyThreshold > 0 {
		s.config.AnomalyThreshold = config.AnomalyThreshold
	}
}

func (s *CanvasFingerprintService) ExportFingerprintData(fingerprintID string) (string, error) {
	s.stabilityMu.RLock()
	defer s.stabilityMu.RUnlock()

	data, exists := s.fingerprintDB[fingerprintID]
	if !exists {
		return "", fmt.Errorf("fingerprint not found")
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (s *CanvasFingerprintService) ImportFingerprintData(fingerprintID string, jsonData string) error {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	var data model.CanvasFingerprintStability
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return err
	}

	data.FingerprintID = fingerprintID
	s.fingerprintDB[fingerprintID] = &data

	return nil
}

func (s *CanvasFingerprintService) ClearExpiredData(expiration time.Duration) int {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	cutoff := time.Now().Add(-expiration)
	removed := 0

	for id, data := range s.fingerprintDB {
		if data.LastSeen.Before(cutoff) {
			delete(s.fingerprintDB, id)
			removed++
		}
	}

	return removed
}

func (s *CanvasFingerprintService) GetStatistics() map[string]interface{} {
	s.stabilityMu.RLock()
	defer s.stabilityMu.RUnlock()

	stats := make(map[string]interface{})

	stats["total_fingerprints"] = len(s.fingerprintDB)

	totalHits := 0
	totalVariations := 0
	stableCount := 0

	for _, data := range s.fingerprintDB {
		totalHits += data.HitCount
		totalVariations += len(data.Variations)
		if data.IsStable {
			stableCount++
		}
	}

	stats["total_hits"] = totalHits
	stats["total_variations"] = totalVariations
	stats["stable_count"] = stableCount

	if len(s.fingerprintDB) > 0 {
		stats["avg_hits_per_fp"] = float64(totalHits) / float64(len(s.fingerprintDB))
		stats["stability_rate"] = float64(stableCount) / float64(len(s.fingerprintDB))
	} else {
		stats["avg_hits_per_fp"] = 0.0
		stats["stability_rate"] = 0.0
	}

	return stats
}
