package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

type FingerprintAdvanced struct{}

func (f *FingerprintAdvanced) AnalyzeCanvasV2(canvasData string) CanvasFingerprintResult {
	result := CanvasFingerprintResult{}

	if canvasData == "" {
		result.Valid = false
		return result
	}

	result.Valid = true
	result.Hash = f.calculateHash(canvasData)
	result.Length = len(canvasData)

	entropy := f.calculateEntropy([]byte(canvasData))
	result.Entropy = entropy
	result.Quality = f.evaluateQuality(entropy, len(canvasData))

	return result
}

func (f *FingerprintAdvanced) AnalyzeWebGL(webglData map[string]interface{}) WebGLFingerprintResult {
	result := WebGLFingerprintResult{}

	if vendor, ok := webglData["vendor"].(string); ok {
		result.Vendor = vendor
	}
	if renderer, ok := webglData["renderer"].(string); ok {
		result.Renderer = renderer
	}
	if extensions, ok := webglData["extensions"].([]string); ok {
		result.ExtensionCount = len(extensions)
		result.Extensions = extensions
	}
	if params, ok := webglData["params"].(map[string]interface{}); ok {
		result.Params = params
	}

	fingerprintStr := result.Vendor + result.Renderer +
		strings.Join(result.Extensions, ",")
	result.Hash = f.calculateHash(fingerprintStr)

	result.Uniqueness = f.evaluateUniqueness(result)

	return result
}

func (f *FingerprintAdvanced) AnalyzeAudioContext(audioData map[string]interface{}) AudioFingerprintResult {
	result := AudioFingerprintResult{}

	if frequencyData, ok := audioData["frequencyData"].([]float64); ok {
		result.FrequencyData = frequencyData
		result.Hash = f.calculateHashFromFloatArray(frequencyData)
	}

	if waveformData, ok := audioData["waveformData"].([]float64); ok {
		result.WaveformData = waveformData
	}

	result.Characteristics = f.extractAudioCharacteristics(result.FrequencyData)

	return result
}

func (f *FingerprintAdvanced) AnalyzeFonts(fonts []string, baseFonts []string) FontFingerprintResult {
	result := FontFingerprintResult{}

	fontSet := make(map[string]bool)
	baseSet := make(map[string]bool)

	for _, font := range fonts {
		fontSet[strings.ToLower(font)] = true
	}
	for _, font := range baseFonts {
		baseSet[strings.ToLower(font)] = true
	}

	uniqueFonts := []string{}
	for font := range fontSet {
		if !baseSet[font] {
			uniqueFonts = append(uniqueFonts, font)
		}
	}

	result.TotalFonts = len(fonts)
	result.UniqueFonts = uniqueFonts
	result.UniqueCount = len(uniqueFonts)
	result.Hash = f.calculateHash(strings.Join(uniqueFonts, ","))

	result.Risk = f.evaluateFontRisk(uniqueFonts, len(fonts))

	return result
}

func (f *FingerprintAdvanced) AnalyzeHardwareConcurrency(concurrency int) HardwareFingerprintResult {
	result := HardwareFingerprintResult{}

	result.Concurrency = concurrency

	if concurrency < 1 || concurrency > 64 {
		result.Suspicious = true
		result.Risk = "high"
	} else if concurrency > 32 {
		result.Suspicious = true
		result.Risk = "medium"
	} else {
		result.Suspicious = false
		result.Risk = "low"
	}

	return result
}

func (f *FingerprintAdvanced) CalculateDeviceFingerprint(fingerprints map[string]string) string {
	combined := ""
	for key, value := range fingerprints {
		combined += key + ":" + value + "|"
	}
	return f.calculateHash(combined)
}

func (f *FingerprintAdvanced) calculateHash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (f *FingerprintAdvanced) calculateHashFromFloatArray(data []float64) string {
	str := ""
	for _, v := range data {
		str += string(rune(int(v * 1000)))
	}
	return f.calculateHash(str)
}

func (f *FingerprintAdvanced) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	frequency := make(map[byte]int)
	for _, b := range data {
		frequency[b]++
	}

	entropy := 0.0
	for _, count := range frequency {
		p := float64(count) / float64(len(data))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (f *FingerprintAdvanced) evaluateQuality(entropy float64, length int) string {
	maxEntropy := 8.0
	normalizedEntropy := entropy / maxEntropy
	normalizedLength := math.Min(float64(length)/10000.0, 1.0)

	score := (normalizedEntropy + normalizedLength) / 2.0

	if score > 0.8 {
		return "high"
	} else if score > 0.5 {
		return "medium"
	}
	return "low"
}

func (f *FingerprintAdvanced) evaluateUniqueness(result WebGLFingerprintResult) float64 {
	score := 0.0

	if result.Vendor != "" && result.Renderer != "" {
		score += 0.3
	}

	if result.ExtensionCount > 50 {
		score += 0.3
	} else if result.ExtensionCount > 20 {
		score += 0.2
	}

	paramsLen := 0
	for _, v := range result.Params {
		paramsLen += len(fmt.Sprintf("%v", v))
	}
	if paramsLen > 1000 {
		score += 0.4
	} else if paramsLen > 500 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (f *FingerprintAdvanced) extractAudioCharacteristics(frequencyData []float64) map[string]float64 {
	characteristics := make(map[string]float64)

	if len(frequencyData) == 0 {
		return characteristics
	}

	sum := 0.0
	max := 0.0
	for _, v := range frequencyData {
		sum += v
		if v > max {
			max = v
		}
	}

	characteristics["mean"] = sum / float64(len(frequencyData))
	characteristics["max"] = max
	characteristics["variance"] = f.calculateVariance(frequencyData, characteristics["mean"])

	return characteristics
}

func (f *FingerprintAdvanced) evaluateFontRisk(uniqueFonts []string, totalFonts int) string {
	uniqueRatio := float64(len(uniqueFonts)) / float64(totalFonts)

	if uniqueRatio > 0.3 {
		return "high"
	} else if uniqueRatio > 0.15 {
		return "medium"
	}
	return "low"
}

func (f *FingerprintAdvanced) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

type CanvasFingerprintResult struct {
	Valid   bool
	Hash    string
	Length  int
	Entropy float64
	Quality string
}

type WebGLFingerprintResult struct {
	Vendor         string
	Renderer       string
	ExtensionCount int
	Extensions     []string
	Params         map[string]interface{}
	Hash           string
	Uniqueness     float64
}

type AudioFingerprintResult struct {
	FrequencyData   []float64
	WaveformData    []float64
	Hash            string
	Characteristics map[string]float64
}

type FontFingerprintResult struct {
	TotalFonts  int
	UniqueFonts []string
	UniqueCount int
	Hash        string
	Risk        string
}

type HardwareFingerprintResult struct {
	Concurrency int
	Suspicious  bool
	Risk        string
}
