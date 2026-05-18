package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type EnhancedFingerprintAnalyzer struct {
	canvasAnalyzer  *CanvasFingerprintAnalyzer
	webglAnalyzer  *WebGLFingerprintAnalyzer
	fontAnalyzer   *FontFingerprintAnalyzer
	audioAnalyzer  *AudioFingerprintAnalyzer
	mediaAnalyzer  *MediaFingerprintAnalyzer
	browserMatcher *BrowserFingerprintMatcher
	mu             sync.RWMutex
}

type CanvasFingerprintAnalyzer struct{}

type CanvasFingerprint struct {
	Hash               string               `json:"hash"`
	RGBAHistogram      []int                `json:"rgba_histogram"`
	NoiseSignature     float64              `json:"noise_signature"`
	RenderingFeatures  *CanvasRenderingFeatures `json:"rendering_features"`
	TextMetrics        *CanvasTextMetrics   `json:"text_metrics"`
	GradientPatterns   []string             `json:"gradient_patterns"`
	PathPatterns       []string             `json:"path_patterns"`
	CompositeOps       []string             `json:"composite_ops"`
	UniquenessScore    float64              `json:"uniqueness_score"`
}

type CanvasRenderingFeatures struct {
	HasLinearGradient bool     `json:"has_linear_gradient"`
	HasRadialGradient bool     `json:"has_radial_gradient"`
	HasPatternFill    bool     `json:"has_pattern_fill"`
	HasShadow         bool     `json:"has_shadow"`
	ShadowBlur        float64  `json:"shadow_blur"`
	ShadowOffsetX     float64  `json:"shadow_offset_x"`
	ShadowOffsetY     float64  `json:"shadow_offset_y"`
	GlobalAlpha       float64  `json:"global_alpha"`
	MiterLimit        float64  `json:"miter_limit"`
	LineCap           string   `json:"line_cap"`
	LineJoin          string   `json:"line_join"`
	LineWidth         float64  `json:"line_width"`
}

type CanvasTextMetrics struct {
	Width               float64 `json:"width"`
	ActualBoundingBoxLeft   float64 `json:"actual_bounding_box_left"`
	ActualBoundingBoxRight  float64 `json:"actual_bounding_box_right"`
	ActualBoundingBoxAscent float64 `json:"actual_bounding_box_ascent"`
	ActualBoundingBoxDescent float64 `json:"actual_bounding_box_descent"`
	FontBoundingBoxAscent   float64 `json:"font_bounding_box_ascent"`
	FontBoundingBoxDescent  float64 `json:"font_bounding_box_descent"`
	EmHeightAscent         float64 `json:"em_height_ascent"`
	EmHeightDescent        float64 `json:"em_height_descent"`
	Baseline              float64 `json:"baseline"`
	CapHeight            float64 `json:"cap_height"`
}

func NewEnhancedFingerprintAnalyzer() *EnhancedFingerprintAnalyzer {
	return &EnhancedFingerprintAnalyzer{
		canvasAnalyzer:  NewCanvasFingerprintAnalyzer(),
		webglAnalyzer:  NewWebGLFingerprintAnalyzer(),
		fontAnalyzer:   NewFontFingerprintAnalyzer(),
		audioAnalyzer: NewAudioFingerprintAnalyzer(),
		mediaAnalyzer: NewMediaFingerprintAnalyzer(),
		browserMatcher: NewBrowserFingerprintMatcher(),
	}
}

func NewCanvasFingerprintAnalyzer() *CanvasFingerprintAnalyzer {
	return &CanvasFingerprintAnalyzer{}
}

func (c *CanvasFingerprintAnalyzer) Analyze(data map[string]interface{}) *CanvasFingerprint {
	fp := &CanvasFingerprint{
		GradientPatterns: make([]string, 0),
		PathPatterns:     make([]string, 0),
		CompositeOps:     make([]string, 0),
	}

	if hash, ok := data["canvas_hash"].(string); ok {
		fp.Hash = hash
	}

	if rgbaData, ok := data["canvas_rgba_histogram"].([]interface{}); ok {
		for _, v := range rgbaData {
			if fv, ok := toFloat64(v); ok {
				fp.RGBAHistogram = append(fp.RGBAHistogram, int(fv))
			}
		}
	}

	if noise, ok := toFloat64(data["canvas_noise_signature"]); ok {
		fp.NoiseSignature = noise
	}

	if features, ok := data["canvas_rendering_features"].(map[string]interface{}); ok {
		fp.RenderingFeatures = c.extractRenderingFeatures(features)
	}

	if textMetrics, ok := data["canvas_text_metrics"].(map[string]interface{}); ok {
		fp.TextMetrics = c.extractTextMetrics(textMetrics)
	}

	if gradients, ok := data["canvas_gradients"].([]interface{}); ok {
		for _, g := range gradients {
			if gs, ok := g.(string); ok {
				fp.GradientPatterns = append(fp.GradientPatterns, gs)
			}
		}
	}

	if paths, ok := data["canvas_paths"].([]interface{}); ok {
		for _, p := range paths {
			if ps, ok := p.(string); ok {
				fp.PathPatterns = append(fp.PathPatterns, ps)
			}
		}
	}

	if comps, ok := data["canvas_composite_ops"].([]interface{}); ok {
		for _, op := range comps {
			if ops, ok := op.(string); ok {
				fp.CompositeOps = append(fp.CompositeOps, ops)
			}
		}
	}

	fp.UniquenessScore = c.calculateUniquenessScore(fp)

	return fp
}

func (c *CanvasFingerprintAnalyzer) extractRenderingFeatures(data map[string]interface{}) *CanvasRenderingFeatures {
	features := &CanvasRenderingFeatures{}

	if v, ok := toFloat64(data["shadow_blur"]); ok {
		features.ShadowBlur = v
		features.HasShadow = v > 0
	}
	if v, ok := toFloat64(data["shadow_offset_x"]); ok {
		features.ShadowOffsetX = v
	}
	if v, ok := toFloat64(data["shadow_offset_y"]); ok {
		features.ShadowOffsetY = v
	}
	if v, ok := toFloat64(data["global_alpha"]); ok {
		features.GlobalAlpha = v
	}
	if v, ok := toFloat64(data["miter_limit"]); ok {
		features.MiterLimit = v
	}
	if v, ok := toFloat64(data["line_width"]); ok {
		features.LineWidth = v
	}
	if v, ok := data["line_cap"].(string); ok {
		features.LineCap = v
	}
	if v, ok := data["line_join"].(string); ok {
		features.LineJoin = v
	}
	features.HasLinearGradient, _ = data["has_linear_gradient"].(bool)
	features.HasRadialGradient, _ = data["has_radial_gradient"].(bool)
	features.HasPatternFill, _ = data["has_pattern_fill"].(bool)

	return features
}

func (c *CanvasFingerprintAnalyzer) extractTextMetrics(data map[string]interface{}) *CanvasTextMetrics {
	metrics := &CanvasTextMetrics{}

	fields := []string{
		"width", "actualBoundingBoxLeft", "actualBoundingBoxRight",
		"actualBoundingBoxAscent", "actualBoundingBoxDescent",
		"fontBoundingBoxAscent", "fontBoundingBoxDescent",
		"emHeightAscent", "emHeightDescent", "baseline", "capHeight",
	}

	metricsMap := map[string]*float64{
		"width":                   &metrics.Width,
		"actualBoundingBoxLeft":   &metrics.ActualBoundingBoxLeft,
		"actualBoundingBoxRight":  &metrics.ActualBoundingBoxRight,
		"actualBoundingBoxAscent": &metrics.ActualBoundingBoxAscent,
		"actualBoundingBoxDescent": &metrics.ActualBoundingBoxDescent,
		"fontBoundingBoxAscent":   &metrics.FontBoundingBoxAscent,
		"fontBoundingBoxDescent":  &metrics.FontBoundingBoxDescent,
		"emHeightAscent":          &metrics.EmHeightAscent,
		"emHeightDescent":         &metrics.EmHeightDescent,
		"baseline":                &metrics.Baseline,
		"capHeight":              &metrics.CapHeight,
	}

	for _, field := range fields {
		if v, ok := toFloat64(data[field]); ok {
			if ptr, exists := metricsMap[field]; exists {
				*ptr = v
			}
		}
	}

	return metrics
}

func (c *CanvasFingerprintAnalyzer) CalculateSimilarity(fp1, fp2 *CanvasFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	scores := make([]float64, 0)

	baseScore := 0.0
	if fp1.Hash == fp2.Hash {
		baseScore = 50.0
	} else {
		baseScore = c.calculateHashSimilarity(fp1.Hash, fp2.Hash) * 50.0
	}
	scores = append(scores, baseScore)

	histogramScore := c.calculateHistogramSimilarity(fp1.RGBAHistogram, fp2.RGBAHistogram)
	scores = append(scores, histogramScore*20.0)

	noiseScore := 0.0
	if fp1.NoiseSignature > 0 && fp2.NoiseSignature > 0 {
		noiseDiff := math.Abs(fp1.NoiseSignature - fp2.NoiseSignature)
		noiseScore = math.Max(0, 10-noiseDiff*100)
	}
	scores = append(scores, noiseScore)

	gradientScore := c.calculateStringArraySimilarity(fp1.GradientPatterns, fp2.GradientPatterns) * 10.0
	scores = append(scores, gradientScore)

	pathScore := c.calculateStringArraySimilarity(fp1.PathPatterns, fp2.PathPatterns) * 10.0
	scores = append(scores, pathScore)

	return math.Min(100, math.Max(0, average(scores)))
}

func (c *CanvasFingerprintAnalyzer) calculateHashSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	if minLen == 0 {
		return 0
	}

	matches := 0
	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(math.Max(float64(len(hash1)), float64(len(hash2))))
}

func (c *CanvasFingerprintAnalyzer) calculateHistogramSimilarity(hist1, hist2 []int) float64 {
	if len(hist1) == 0 || len(hist2) == 0 {
		return 0
	}

	minLen := len(hist1)
	if len(hist2) < minLen {
		minLen = len(hist2)
	}

	if minLen == 0 {
		return 0
	}

	sum := 0.0
	for i := 0; i < minLen; i++ {
		maxVal := float64(hist1[i])
		if hist2[i] > hist1[i] {
			maxVal = float64(hist2[i])
		}
		if maxVal > 0 {
			diff := math.Abs(float64(hist1[i]) - float64(hist2[i]))
			sum += 1.0 - diff/maxVal
		}
	}

	return sum / float64(minLen)
}

func (c *CanvasFingerprintAnalyzer) calculateStringArraySimilarity(arr1, arr2 []string) float64 {
	if len(arr1) == 0 && len(arr2) == 0 {
		return 1.0
	}
	if len(arr1) == 0 || len(arr2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, s := range arr1 {
		set1[s] = true
	}
	for _, s := range arr2 {
		set2[s] = true
	}

	common := 0
	for s := range set1 {
		if set2[s] {
			common++
		}
	}

	total := len(set1) + len(set2) - common
	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

func (c *CanvasFingerprintAnalyzer) calculateUniquenessScore(fp *CanvasFingerprint) float64 {
	score := 50.0

	if len(fp.RGBAHistogram) > 50 {
		entropy := c.calculateHistogramEntropy(fp.RGBAHistogram)
		score += entropy * 20
	}

	if fp.NoiseSignature > 0.001 && fp.NoiseSignature < 0.1 {
		score += 15
	}

	if len(fp.GradientPatterns) > 2 {
		score += 5
	}

	if len(fp.PathPatterns) > 3 {
		score += 5
	}

	if len(fp.CompositeOps) > 2 {
		score += 5
	}

	return math.Min(100, score)
}

func (c *CanvasFingerprintAnalyzer) calculateHistogramEntropy(hist []int) float64 {
	if len(hist) == 0 {
		return 0
	}

	total := 0
	for _, v := range hist {
		total += v
	}
	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, v := range hist {
		if v > 0 {
			p := float64(v) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(float64(len(hist)))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}
	return 0
}

type WebGLFingerprintAnalyzer struct{}

type WebGLFingerprint struct {
	Hash                  string           `json:"hash"`
	Vendor                string           `json:"vendor"`
	Renderer              string           `json:"renderer"`
	UnmaskedVendor        string           `json:"unmasked_vendor"`
	UnmaskedRenderer      string           `json:"unmasked_renderer"`
	MaxTextureSize        int              `json:"max_texture_size"`
	MaxRenderbufferSize   int              `json:"max_renderbuffer_size"`
	MaxVertexAttribs      int              `json:"max_vertex_attribs"`
	MaxViewportDims       []int            `json:"max_viewport_dims"`
	MaxCombinedTexUnits   int              `json:"max_combined_texture_units"`
	MaxCubeMapTextureSize int              `json:"max_cube_map_texture_size"`
	Max3DTextureSize      int              `json:"max_3d_texture_size"`
	MaxFramebufferWidth   int              `json:"max_framebuffer_width"`
	MaxFramebufferHeight  int              `json:"max_framebuffer_height"`
	MaxColorAttachments   int              `json:"max_color_attachments"`
	SupportedExtensions   []string         `json:"supported_extensions"`
	ExtensionCount        int              `json:"extension_count"`
	PrecisionFormats      []*PrecisionInfo `json:"precision_formats"`
	Antialiasing          bool             `json:"antialiasing"`
	PremultipliedAlpha    bool             `json:"premultiplied_alpha"`
	PreserveDrawingBuffer bool             `json:"preserve_drawing_buffer"`
	FailIfMajorPerfCaveat bool             `json:"fail_if_major_perf_caveat"`
	IsSoftwareRenderer    bool             `json:"is_software_renderer"`
	IsVirtualGPU          bool             `json:"is_virtual_gpu"`
	RendererFingerprint   string           `json:"renderer_fingerprint"`
	UniquenessScore       float64          `json:"uniqueness_score"`
}

type PrecisionInfo struct {
	Format        string  `json:"format"`
	Type          string  `json:"type"`
	RangeMin      int     `json:"range_min"`
	RangeMax      int     `json:"range_max"`
	PrecisionBits int     `json:"precision_bits"`
}

func NewWebGLFingerprintAnalyzer() *WebGLFingerprintAnalyzer {
	return &WebGLFingerprintAnalyzer{}
}

func (w *WebGLFingerprintAnalyzer) Analyze(data map[string]interface{}) *WebGLFingerprint {
	fp := &WebGLFingerprint{
		SupportedExtensions: make([]string, 0),
		PrecisionFormats:    make([]*PrecisionInfo, 0),
		MaxViewportDims:     make([]int, 0),
	}

	if hash, ok := data["webgl_hash"].(string); ok {
		fp.Hash = hash
	}
	if v, ok := data["webgl_vendor"].(string); ok {
		fp.Vendor = v
		fp.UnmaskedVendor = v
	}
	if v, ok := data["webgl_renderer"].(string); ok {
		fp.Renderer = v
		fp.UnmaskedRenderer = v
	}
	if v, ok := toInt(data["webgl_max_texture_size"]); ok {
		fp.MaxTextureSize = v
	}
	if v, ok := toInt(data["webgl_max_renderbuffer_size"]); ok {
		fp.MaxRenderbufferSize = v
	}
	if v, ok := toInt(data["webgl_max_vertex_attribs"]); ok {
		fp.MaxVertexAttribs = v
	}
	if v, ok := toInt(data["webgl_max_combined_texture_units"]); ok {
		fp.MaxCombinedTexUnits = v
	}
	if v, ok := toInt(data["webgl_max_cube_map_texture_size"]); ok {
		fp.MaxCubeMapTextureSize = v
	}
	if v, ok := toInt(data["webgl_max_3d_texture_size"]); ok {
		fp.Max3DTextureSize = v
	}
	if v, ok := toInt(data["webgl_max_framebuffer_width"]); ok {
		fp.MaxFramebufferWidth = v
	}
	if v, ok := toInt(data["webgl_max_framebuffer_height"]); ok {
		fp.MaxFramebufferHeight = v
	}
	if v, ok := toInt(data["webgl_max_color_attachments"]); ok {
		fp.MaxColorAttachments = v
	}

	if exts, ok := data["webgl_extensions"].([]interface{}); ok {
		for _, e := range exts {
			if es, ok := e.(string); ok {
				fp.SupportedExtensions = append(fp.SupportedExtensions, es)
			}
		}
		fp.ExtensionCount = len(fp.SupportedExtensions)
	}

	if precisions, ok := data["webgl_precision_formats"].([]interface{}); ok {
		for _, p := range precisions {
			if pm, ok := p.(map[string]interface{}); ok {
				info := &PrecisionInfo{}
				if format, ok := pm["format"].(string); ok {
					info.Format = format
				}
				if tp, ok := pm["type"].(string); ok {
					info.Type = tp
				}
				if v, ok := toInt(pm["range_min"]); ok {
					info.RangeMin = v
				}
				if v, ok := toInt(pm["range_max"]); ok {
					info.RangeMax = v
				}
				if v, ok := toInt(pm["precision_bits"]); ok {
					info.PrecisionBits = v
				}
				fp.PrecisionFormats = append(fp.PrecisionFormats, info)
			}
		}
	}

	if viewport, ok := data["webgl_max_viewport_dims"].([]interface{}); ok {
		for _, v := range viewport {
			if vi, ok := toInt(v); ok {
				fp.MaxViewportDims = append(fp.MaxViewportDims, vi)
			}
		}
	}

	fp.Antialiasing, _ = data["webgl_antialiasing"].(bool)
	fp.PremultipliedAlpha, _ = data["webgl_premultiplied_alpha"].(bool)
	fp.PreserveDrawingBuffer, _ = data["webgl_preserve_drawing_buffer"].(bool)
	fp.FailIfMajorPerfCaveat, _ = data["webgl_fail_if_major_perf_caveat"].(bool)

	fp.IsSoftwareRenderer = w.detectSoftwareRenderer(fp.Renderer)
	fp.IsVirtualGPU = w.detectVirtualGPU(fp.Renderer)
	fp.RendererFingerprint = w.generateRendererFingerprint(fp)
	fp.UniquenessScore = w.calculateUniquenessScore(fp)

	return fp
}

func (w *WebGLFingerprintAnalyzer) detectSoftwareRenderer(renderer string) bool {
	softwarePatterns := []string{
		"swiftshader", "llvmpipe", "mesa", "software",
		"google inc.", "microsoft basic",
	}

	lowerRenderer := strings.ToLower(renderer)
	for _, pattern := range softwarePatterns {
		if strings.Contains(lowerRenderer, pattern) {
			return true
		}
	}
	return false
}

func (w *WebGLFingerprintAnalyzer) detectVirtualGPU(renderer string) bool {
	virtualPatterns := []string{
		"vmware", "virtualbox", "virtual", "parallels",
		"qemu", "kvm", "hyper-v", "xen",
	}

	lowerRenderer := strings.ToLower(renderer)
	for _, pattern := range virtualPatterns {
		if strings.Contains(lowerRenderer, pattern) {
			return true
		}
	}
	return false
}

func (w *WebGLFingerprintAnalyzer) generateRendererFingerprint(fp *WebGLFingerprint) string {
	components := []string{
		fp.UnmaskedRenderer,
		fmt.Sprintf("%d", fp.MaxTextureSize),
		fmt.Sprintf("%d", fp.MaxVertexAttribs),
		fmt.Sprintf("%d", fp.ExtensionCount),
	}
	return strings.Join(components, "|")
}

func (w *WebGLFingerprintAnalyzer) CalculateSimilarity(fp1, fp2 *WebGLFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	scores := make([]float64, 0)

	if fp1.Hash == fp2.Hash {
		scores = append(scores, 40.0)
	} else {
		scores = append(scores, w.calculateHashSimilarity(fp1.Hash, fp2.Hash)*40.0)
	}

	rendererScore := 0.0
	if fp1.UnmaskedRenderer == fp2.UnmaskedRenderer && fp1.UnmaskedRenderer != "" {
		rendererScore = 25.0
	} else if fp1.RendererFingerprint == fp2.RendererFingerprint && fp1.RendererFingerprint != "" {
		rendererScore = 15.0
	}
	scores = append(scores, rendererScore)

	extScore := w.calculateExtensionSimilarity(fp1.SupportedExtensions, fp2.SupportedExtensions) * 20.0
	scores = append(scores, extScore)

	paramScore := w.calculateParameterSimilarity(fp1, fp2) * 15.0
	scores = append(scores, paramScore)

	return math.Min(100, math.Max(0, average(scores)))
}

func (w *WebGLFingerprintAnalyzer) calculateHashSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	if minLen == 0 {
		return 0
	}

	matches := 0
	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(math.Max(float64(len(hash1)), float64(len(hash2))))
}

func (w *WebGLFingerprintAnalyzer) calculateExtensionSimilarity(exts1, exts2 []string) float64 {
	if len(exts1) == 0 && len(exts2) == 0 {
		return 1.0
	}
	if len(exts1) == 0 || len(exts2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, e := range exts1 {
		set1[e] = true
	}
	for _, e := range exts2 {
		set2[e] = true
	}

	common := 0
	for e := range set1 {
		if set2[e] {
			common++
		}
	}

	total := len(set1) + len(set2) - common
	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

func (w *WebGLFingerprintAnalyzer) calculateParameterSimilarity(fp1, fp2 *WebGLFingerprint) float64 {
	scores := make([]float64, 0)

	if fp1.MaxTextureSize == fp2.MaxTextureSize && fp1.MaxTextureSize > 0 {
		scores = append(scores, 1.0)
	}
	if fp1.MaxVertexAttribs == fp2.MaxVertexAttribs && fp1.MaxVertexAttribs > 0 {
		scores = append(scores, 1.0)
	}
	if fp1.MaxCombinedTexUnits == fp2.MaxCombinedTexUnits && fp1.MaxCombinedTexUnits > 0 {
		scores = append(scores, 1.0)
	}

	if len(fp1.PrecisionFormats) > 0 && len(fp2.PrecisionFormats) > 0 {
		precisionScore := 0.0
		for _, p1 := range fp1.PrecisionFormats {
			for _, p2 := range fp2.PrecisionFormats {
				if p1.Format == p2.Format && p1.Type == p2.Type {
					if p1.PrecisionBits == p2.PrecisionBits {
						precisionScore++
					}
					break
				}
			}
		}
		maxPossible := math.Min(float64(len(fp1.PrecisionFormats)), float64(len(fp2.PrecisionFormats)))
		if maxPossible > 0 {
			scores = append(scores, precisionScore/maxPossible)
		}
	}

	if len(scores) == 0 {
		return 0
	}

	return average(scores)
}

func (w *WebGLFingerprintAnalyzer) calculateUniquenessScore(fp *WebGLFingerprint) float64 {
	score := 30.0

	if !fp.IsSoftwareRenderer && !fp.IsVirtualGPU {
		score += 20
	}

	if fp.ExtensionCount > 50 {
		score += 15
	} else if fp.ExtensionCount > 20 {
		score += 10
	} else if fp.ExtensionCount > 10 {
		score += 5
	}

	if fp.MaxTextureSize > 4096 {
		score += 10
	}
	if fp.MaxVertexAttribs > 8 {
		score += 5
	}

	if fp.UnmaskedRenderer != "" && fp.UnmaskedRenderer != fp.Renderer {
		score += 10
	}

	if len(fp.PrecisionFormats) > 3 {
		score += 5
	}

	if fp.Max3DTextureSize > 256 || fp.MaxCubeMapTextureSize > 4096 {
		score += 5
	}

	return math.Min(100, score)
}

type FontFingerprintAnalyzer struct{}

type FontFingerprint struct {
	Hash                  string   `json:"hash"`
	DetectedFonts         []string `json:"detected_fonts"`
	FontCount             int      `json:"font_count"`
	FontMetrics           []*FontMetricInfo `json:"font_metrics"`
	FontFamilyGroups      []string `json:"font_family_groups"`
	MonospaceFonts        []string `json:"monospace_fonts"`
	SerifFonts            []string `json:"serif_fonts"`
	SansSerifFonts        []string `json:"sans_serif_fonts"`
	DisplayFonts          []string `json:"display_fonts"`
	CJKFonts              []string `json:"cjk_fonts"`
	CommonFontsMissing    []string `json:"common_fonts_missing"`
	RareFonts             []string `json:"rare_fonts"`
	FontWidthPatterns     []string `json:"font_width_patterns"`
	IsLimitedFontSet      bool     `json:"is_limited_font_set"`
	IsSuspiciousFontSet   bool     `json:"is_suspicious_font_set"`
	UniquenessScore       float64  `json:"uniqueness_score"`
}

type FontMetricInfo struct {
	FontFamily string  `json:"font_family"`
	Width      float64 `json:"width"`
	Ascent     float64 `json:"ascent"`
	Descent    float64 `json:"descent"`
	XHeight    float64 `json:"x_height"`
	CapHeight  float64 `json:"cap_height"`
}

var commonFonts = []string{
	"Arial", "Helvetica", "Times New Roman", "Verdana", "Georgia",
	"Tahoma", "Trebuchet MS", "Courier New", "Impact", "Comic Sans MS",
}

var rareFonts = []string{
	"Brush Script MT", "Copperplate", "Papyrus", "OCR A Extended",
	"Viner Hand ITC", "Juice ITC", "Matura MT Script Capitals",
	"Freestyle Script", "French Script MT", "Bradley Hand ITC",
}

var cjkFonts = []string{
	"Microsoft YaHei", "SimSun", "SimHei", "Microsoft JhengHei",
	"PMingLiU", "MingLiU", "Microsoft YaHei UI", "Yu Gothic",
	"Meiryo", "Malgun Gothic", "Noto Sans CJK",
}

func NewFontFingerprintAnalyzer() *FontFingerprintAnalyzer {
	return &FontFingerprintAnalyzer{}
}

func (f *FontFingerprintAnalyzer) Analyze(data map[string]interface{}) *FontFingerprint {
	fp := &FontFingerprint{
		DetectedFonts:      make([]string, 0),
		FontMetrics:        make([]*FontMetricInfo, 0),
		FontFamilyGroups:   make([]string, 0),
		MonospaceFonts:     make([]string, 0),
		SerifFonts:         make([]string, 0),
		SansSerifFonts:     make([]string, 0),
		DisplayFonts:       make([]string, 0),
		CJKFonts:           make([]string, 0),
		CommonFontsMissing: make([]string, 0),
		RareFonts:          make([]string, 0),
		FontWidthPatterns:  make([]string, 0),
	}

	if hash, ok := data["font_hash"].(string); ok {
		fp.Hash = hash
	}

	if fonts, ok := data["detected_fonts"].([]interface{}); ok {
		for _, font := range fonts {
			if fontStr, ok := font.(string); ok {
				fp.DetectedFonts = append(fp.DetectedFonts, fontStr)
			}
		}
	}

	if metrics, ok := data["font_metrics"].([]interface{}); ok {
		for _, m := range metrics {
			if mm, ok := m.(map[string]interface{}); ok {
				info := &FontMetricInfo{}
				if family, ok := mm["font_family"].(string); ok {
					info.FontFamily = family
				}
				if v, ok := toFloat64(mm["width"]); ok {
					info.Width = v
				}
				if v, ok := toFloat64(mm["ascent"]); ok {
					info.Ascent = v
				}
				if v, ok := toFloat64(mm["descent"]); ok {
					info.Descent = v
				}
				if v, ok := toFloat64(mm["x_height"]); ok {
					info.XHeight = v
				}
				if v, ok := toFloat64(mm["cap_height"]); ok {
					info.CapHeight = v
				}
				fp.FontMetrics = append(fp.FontMetrics, info)
			}
		}
	}

	fp.FontCount = len(fp.DetectedFonts)
	f.categorizeFonts(fp)
	f.detectMissingFonts(fp)
	f.detectRareFonts(fp)
	f.detectSuspiciousFonts(fp)
	fp.UniquenessScore = f.calculateUniquenessScore(fp)

	return fp
}

func (f *FontFingerprintAnalyzer) categorizeFonts(fp *FontFingerprint) {
	monospacePatterns := []string{"mono", "courier", "consola", "code", "lucida console"}
	serifPatterns := []string{"serif", "times", "georgia", "palatino", "garamond", "cambria", "bookman"}
	sansSerifPatterns := []string{"sans", "arial", "helvetica", "segoe", "roboto", "verdana", "tahoma", "calibri"}
	displayPatterns := []string{"impact", "comic", "brush", "papyrus"}
	cjkPatterns := []string{"cjk", "chinese", "yahei", "mingliu", "simsun", "simhei", "noto sans cjk"}

	for _, font := range fp.DetectedFonts {
		lower := strings.ToLower(font)
		categorized := false

		for _, pattern := range cjkPatterns {
			if strings.Contains(lower, pattern) {
				fp.CJKFonts = append(fp.CJKFonts, font)
				categorized = true
				break
			}
		}
		if categorized {
			continue
		}

		for _, pattern := range monospacePatterns {
			if strings.Contains(lower, pattern) {
				fp.MonospaceFonts = append(fp.MonospaceFonts, font)
				categorized = true
				break
			}
		}
		if categorized {
			continue
		}

		for _, pattern := range serifPatterns {
			if strings.Contains(lower, pattern) {
				fp.SerifFonts = append(fp.SerifFonts, font)
				categorized = true
				break
			}
		}
		if categorized {
			continue
		}

		for _, pattern := range sansSerifPatterns {
			if strings.Contains(lower, pattern) {
				fp.SansSerifFonts = append(fp.SansSerifFonts, font)
				categorized = true
				break
			}
		}
		if categorized {
			continue
		}

		for _, pattern := range displayPatterns {
			if strings.Contains(lower, pattern) {
				fp.DisplayFonts = append(fp.DisplayFonts, font)
				break
			}
		}
	}
}

func (f *FontFingerprintAnalyzer) detectMissingFonts(fp *FontFingerprint) {
	detectedSet := make(map[string]bool)
	for _, font := range fp.DetectedFonts {
		detectedSet[strings.ToLower(font)] = true
	}

	for _, common := range commonFonts {
		found := false
		for _, detected := range fp.DetectedFonts {
			if strings.Contains(strings.ToLower(detected), strings.ToLower(common)) {
				found = true
				break
			}
		}
		if !found {
			fp.CommonFontsMissing = append(fp.CommonFontsMissing, common)
		}
	}
}

func (f *FontFingerprintAnalyzer) detectRareFonts(fp *FontFingerprint) {
	for _, detected := range fp.DetectedFonts {
		for _, rare := range rareFonts {
			if strings.Contains(strings.ToLower(detected), strings.ToLower(rare)) {
				fp.RareFonts = append(fp.RareFonts, detected)
				break
			}
		}
	}
}

func (f *FontFingerprintAnalyzer) detectSuspiciousFonts(fp *FontFingerprint) {
	suspiciousPatterns := []string{"fake", "test", "font1", "keyboard", "password"}

	for _, detected := range fp.DetectedFonts {
		lower := strings.ToLower(detected)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(lower, pattern) {
				fp.IsSuspiciousFontSet = true
				return
			}
		}
	}
}

func (f *FontFingerprintAnalyzer) CalculateSimilarity(fp1, fp2 *FontFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	scores := make([]float64, 0)

	if fp1.Hash == fp2.Hash {
		scores = append(scores, 40.0)
	} else {
		scores = append(scores, f.calculateFontListSimilarity(fp1.DetectedFonts, fp2.DetectedFonts)*40.0)
	}

	metricsScore := f.calculateMetricsSimilarity(fp1.FontMetrics, fp2.FontMetrics) * 30.0
	scores = append(scores, metricsScore)

	categoryScore := f.calculateCategorySimilarity(fp1, fp2) * 20.0
	scores = append(scores, categoryScore)

	rareScore := f.calculateRareFontSimilarity(fp1.RareFonts, fp2.RareFonts) * 10.0
	scores = append(scores, rareScore)

	return math.Min(100, math.Max(0, average(scores)))
}

func (f *FontFingerprintAnalyzer) calculateFontListSimilarity(fonts1, fonts2 []string) float64 {
	if len(fonts1) == 0 && len(fonts2) == 0 {
		return 1.0
	}
	if len(fonts1) == 0 || len(fonts2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, font := range fonts1 {
		set1[strings.ToLower(font)] = true
	}
	for _, font := range fonts2 {
		set2[strings.ToLower(font)] = true
	}

	common := 0
	for font := range set1 {
		if set2[font] {
			common++
		}
	}

	total := len(set1) + len(set2) - common
	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

func (f *FontFingerprintAnalyzer) calculateMetricsSimilarity(metrics1, metrics2 []*FontMetricInfo) float64 {
	if len(metrics1) == 0 && len(metrics2) == 0 {
		return 1.0
	}
	if len(metrics1) == 0 || len(metrics2) == 0 {
		return 0
	}

	scores := make([]float64, 0)

	for _, m1 := range metrics1 {
		for _, m2 := range metrics2 {
			if m1.FontFamily == m2.FontFamily {
				score := 0.0
				if m1.Width > 0 && m2.Width > 0 {
					widthDiff := math.Abs(m1.Width-m2.Width) / math.Max(m1.Width, m2.Width)
					score += 1.0 - widthDiff
				}
				if m1.Ascent > 0 && m2.Ascent > 0 {
					ascentDiff := math.Abs(m1.Ascent-m2.Ascent) / math.Max(m1.Ascent, m2.Ascent)
					score += 1.0 - ascentDiff
				}
				if m1.Descent > 0 && m2.Descent > 0 {
					descentDiff := math.Abs(m1.Descent-m2.Descent) / math.Max(m1.Descent, m2.Descent)
					score += 1.0 - descentDiff
				}
				scores = append(scores, score/3.0)
				break
			}
		}
	}

	if len(scores) == 0 {
		return 0
	}

	return average(scores)
}

func (f *FontFingerprintAnalyzer) calculateCategorySimilarity(fp1, fp2 *FontFingerprint) float64 {
	monospaceSim := f.calculateFontListSimilarity(fp1.MonospaceFonts, fp2.MonospaceFonts)
	serifSim := f.calculateFontListSimilarity(fp1.SerifFonts, fp2.SerifFonts)
	sansSerifSim := f.calculateFontListSimilarity(fp1.SansSerifFonts, fp2.SansSerifFonts)
	cjkSim := f.calculateFontListSimilarity(fp1.CJKFonts, fp2.CJKFonts)

	return (monospaceSim + serifSim + sansSerifSim + cjkSim) / 4.0
}

func (f *FontFingerprintAnalyzer) calculateRareFontSimilarity(rare1, rare2 []string) float64 {
	if len(rare1) == 0 && len(rare2) == 0 {
		return 1.0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, font := range rare1 {
		set1[strings.ToLower(font)] = true
	}
	for _, font := range rare2 {
		set2[strings.ToLower(font)] = true
	}

	common := 0
	for font := range set1 {
		if set2[font] {
			common++
		}
	}

	maxLen := math.Max(float64(len(rare1)), float64(len(rare2)))
	if maxLen == 0 {
		return 1.0
	}

	return float64(common) / maxLen
}

func (f *FontFingerprintAnalyzer) calculateUniquenessScore(fp *FontFingerprint) float64 {
	score := 30.0

	if fp.FontCount > 20 {
		score += 25
	} else if fp.FontCount > 10 {
		score += 15
	} else if fp.FontCount > 5 {
		score += 10
	}

	if len(fp.RareFonts) > 0 {
		score += 15
	}

	if len(fp.CJKFonts) > 0 {
		score += 10
	}

	if len(fp.CommonFontsMissing) > 3 {
		score += 5
	}

	if fp.IsSuspiciousFontSet {
		score -= 30
	}

	if len(fp.FontMetrics) > 5 {
		score += 10
	}

	return math.Min(100, math.Max(0, score))
}

type AudioFingerprintAnalyzer struct{}

type AudioFingerprint struct {
	Hash                   string              `json:"hash"`
	RendersCount           int                 `json:"renders_count"`
	ChannelData            []*AudioChannelInfo `json:"channel_data"`
	Variance               float64             `json:"variance"`
	NonZeroRatio           float64             `json:"non_zero_ratio"`
	PeakAmplitude          float64             `json:"peak_amplitude"`
	MeanAmplitude          float64             `json:"mean_amplitude"`
	RMSLevel               float64             `json:"rms_level"`
	DynamicRange           float64             `json:"dynamic_range"`
	ZeroCrossingRate       float64             `json:"zero_crossing_rate"`
	SpectralCentroid       float64             `json:"spectral_centroid"`
	IsStable               bool                `json:"is_stable"`
	IsSuspiciouslySilent   bool                `json:"is_suspiciously_silent"`
	IsIdenticalAcrossRenders bool              `json:"is_identical_across_renders"`
	UniquenessScore        float64              `json:"uniqueness_score"`
}

type AudioChannelInfo struct {
	ChannelIndex int       `json:"channel_index"`
	SampleCount  int       `json:"sample_count"`
	MinValue     float64   `json:"min_value"`
	MaxValue     float64   `json:"max_value"`
	MeanValue    float64   `json:"mean_value"`
	Variance     float64   `json:"variance"`
	Histogram    []int     `json:"histogram"`
}

func NewAudioFingerprintAnalyzer() *AudioFingerprintAnalyzer {
	return &AudioFingerprintAnalyzer{}
}

func (a *AudioFingerprintAnalyzer) Analyze(data map[string]interface{}) *AudioFingerprint {
	fp := &AudioFingerprint{
		ChannelData: make([]*AudioChannelInfo, 0),
	}

	if hash, ok := data["audio_hash"].(string); ok {
		fp.Hash = hash
	}

	if v, ok := toInt(data["audio_renders_count"]); ok {
		fp.RendersCount = v
	}

	if channels, ok := data["audio_channels"].([]interface{}); ok {
		for _, ch := range channels {
			if cm, ok := ch.(map[string]interface{}); ok {
				info := &AudioChannelInfo{}
				if idx, ok := toInt(cm["channel_index"]); ok {
					info.ChannelIndex = idx
				}
				if cnt, ok := toInt(cm["sample_count"]); ok {
					info.SampleCount = cnt
				}
				if v, ok := toFloat64(cm["min_value"]); ok {
					info.MinValue = v
				}
				if v, ok := toFloat64(cm["max_value"]); ok {
					info.MaxValue = v
				}
				if v, ok := toFloat64(cm["mean_value"]); ok {
					info.MeanValue = v
				}
				if v, ok := toFloat64(cm["variance"]); ok {
					info.Variance = v
				}
				if hist, ok := cm["histogram"].([]interface{}); ok {
					for _, h := range hist {
						if hv, ok := toInt(h); ok {
							info.Histogram = append(info.Histogram, hv)
						}
					}
				}
				fp.ChannelData = append(fp.ChannelData, info)
			}
		}
	}

	if v, ok := toFloat64(data["audio_variance"]); ok {
		fp.Variance = v
	}
	if v, ok := toFloat64(data["audio_non_zero_ratio"]); ok {
		fp.NonZeroRatio = v
	}
	if v, ok := toFloat64(data["audio_peak_amplitude"]); ok {
		fp.PeakAmplitude = v
	}
	if v, ok := toFloat64(data["audio_mean_amplitude"]); ok {
		fp.MeanAmplitude = v
	}
	if v, ok := toFloat64(data["audio_rms_level"]); ok {
		fp.RMSLevel = v
	}
	if v, ok := toFloat64(data["audio_dynamic_range"]); ok {
		fp.DynamicRange = v
	}
	if v, ok := toFloat64(data["audio_zero_crossing_rate"]); ok {
		fp.ZeroCrossingRate = v
	}
	if v, ok := toFloat64(data["audio_spectral_centroid"]); ok {
		fp.SpectralCentroid = v
	}

	fp.IsStable, _ = data["audio_is_stable"].(bool)
	fp.IsSuspiciouslySilent, _ = data["audio_is_suspiciously_silent"].(bool)
	fp.IsIdenticalAcrossRenders, _ = data["audio_is_identical_across_renders"].(bool)

	fp.UniquenessScore = a.calculateUniquenessScore(fp)

	return fp
}

func (a *AudioFingerprintAnalyzer) CalculateSimilarity(fp1, fp2 *AudioFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	scores := make([]float64, 0)

	if fp1.Hash == fp2.Hash {
		scores = append(scores, 50.0)
	} else {
		scores = append(scores, a.calculateHashSimilarity(fp1.Hash, fp2.Hash)*50.0)
	}

	metricScore := a.calculateMetricSimilarity(fp1, fp2) * 30.0
	scores = append(scores, metricScore)

	channelScore := a.calculateChannelSimilarity(fp1.ChannelData, fp2.ChannelData) * 20.0
	scores = append(scores, channelScore)

	return math.Min(100, math.Max(0, average(scores)))
}

func (a *AudioFingerprintAnalyzer) calculateHashSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	if minLen == 0 {
		return 0
	}

	matches := 0
	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(math.Max(float64(len(hash1)), float64(len(hash2))))
}

func (a *AudioFingerprintAnalyzer) calculateMetricSimilarity(fp1, fp2 *AudioFingerprint) float64 {
	scores := make([]float64, 0)

	if fp1.Variance > 0 && fp2.Variance > 0 {
		varianceRatio := math.Min(fp1.Variance, fp2.Variance) / math.Max(fp1.Variance, fp2.Variance)
		scores = append(scores, varianceRatio)
	}

	if fp1.NonZeroRatio > 0 && fp2.NonZeroRatio > 0 {
		nonZeroRatio := math.Min(fp1.NonZeroRatio, fp2.NonZeroRatio) / math.Max(fp1.NonZeroRatio, fp2.NonZeroRatio)
		scores = append(scores, nonZeroRatio)
	}

	if fp1.RMSLevel > 0 && fp2.RMSLevel > 0 {
		rmsRatio := math.Min(fp1.RMSLevel, fp2.RMSLevel) / math.Max(fp1.RMSLevel, fp2.RMSLevel)
		scores = append(scores, rmsRatio)
	}

	if fp1.DynamicRange > 0 && fp2.DynamicRange > 0 {
		drRatio := math.Min(fp1.DynamicRange, fp2.DynamicRange) / math.Max(fp1.DynamicRange, fp2.DynamicRange)
		scores = append(scores, drRatio)
	}

	if len(scores) == 0 {
		return 0
	}

	return average(scores)
}

func (a *AudioFingerprintAnalyzer) calculateChannelSimilarity(ch1, ch2 []*AudioChannelInfo) float64 {
	if len(ch1) == 0 && len(ch2) == 0 {
		return 1.0
	}
	if len(ch1) == 0 || len(ch2) == 0 {
		return 0
	}

	scores := make([]float64, 0)

	for _, c1 := range ch1 {
		for _, c2 := range ch2 {
			if c1.ChannelIndex == c2.ChannelIndex {
				score := 0.0
				if c1.Variance > 0 && c2.Variance > 0 {
					varianceScore := math.Min(c1.Variance, c2.Variance) / math.Max(c1.Variance, c2.Variance)
					score += varianceScore
				}
				if c1.SampleCount == c2.SampleCount && c1.SampleCount > 0 {
					score += 1.0
				}
				scores = append(scores, score/2.0)
				break
			}
		}
	}

	if len(scores) == 0 {
		return 0
	}

	return average(scores)
}

func (a *AudioFingerprintAnalyzer) calculateUniquenessScore(fp *AudioFingerprint) float64 {
	score := 50.0

	if fp.IsIdenticalAcrossRenders {
		score -= 30
	}

	if fp.IsSuspiciouslySilent {
		score -= 20
	}

	if fp.Variance > 0.0001 {
		score += 15
	}

	if fp.NonZeroRatio > 0.3 {
		score += 10
	}

	if fp.DynamicRange > 0.1 {
		score += 10
	}

	if len(fp.ChannelData) > 1 {
		score += 5
	}

	return math.Min(100, math.Max(0, score))
}

type MediaFingerprintAnalyzer struct{}

type MediaFingerprint struct {
	Hash                string                `json:"hash"`
	VideoCodecs         []string              `json:"video_codecs"`
	AudioCodecs         []string              `json:"audio_codecs"`
	MediaDevices        []*MediaDeviceInfo    `json:"media_devices"`
	SupportedFormats    []*MediaFormatInfo    `json:"supported_formats"`
	MediaSourceSupport  *MediaSourceSupport    `json:"media_source_support"`
	VideoCapabilities   *VideoCapabilities    `json:"video_capabilities"`
	AudioCapabilities   *AudioCapabilities     `json:"audio_capabilities"`
	DeviceLabels        []string              `json:"device_labels"`
	HasCamera           bool                  `json:"has_camera"`
	HasMicrophone       bool                  `json:"has_microphone"`
	HasSpeaker          bool                  `json:"has_speaker"`
	DeviceCount         int                   `json:"device_count"`
	UniquenessScore    float64               `json:"uniqueness_score"`
}

type MediaDeviceInfo struct {
	DeviceID   string `json:"device_id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	GroupID    string `json:"group_id"`
	IsDefault  bool   `json:"is_default"`
}

type MediaFormatInfo struct {
	MimeType    string `json:"mime_type"`
	HasAudio    bool   `json:"has_audio"`
	HasVideo    bool   `json:"has_video"`
	IsSupported bool   `json:"is_supported"`
}

type MediaSourceSupport struct {
	MediaSource       bool `json:"media_source"`
	SourceBuffer      bool `json:"source_buffer"`
	WebRTC            bool `json:"webrtc"`
	MediaStream       bool `json:"media_stream"`
	MediaRecorder     bool `json:"media_recorder"`
}

type VideoCapabilities struct {
	MaxWidth       int      `json:"max_width"`
	MaxHeight      int      `json:"max_height"`
	MaxFrameRate   float64  `json:"max_frame_rate"`
	SupportedResolutions []string `json:"supported_resolutions"`
}

type AudioCapabilities struct {
	SampleRates    []int    `json:"sample_rates"`
	ChannelCounts  []int    `json:"channel_counts"`
	MaxChannelCount int     `json:"max_channel_count"`
}

func NewMediaFingerprintAnalyzer() *MediaFingerprintAnalyzer {
	return &MediaFingerprintAnalyzer{}
}

func (m *MediaFingerprintAnalyzer) Analyze(data map[string]interface{}) *MediaFingerprint {
	fp := &MediaFingerprint{
		VideoCodecs:        make([]string, 0),
		AudioCodecs:        make([]string, 0),
		MediaDevices:       make([]*MediaDeviceInfo, 0),
		SupportedFormats:   make([]*MediaFormatInfo, 0),
		DeviceLabels:       make([]string, 0),
	}

	if hash, ok := data["media_hash"].(string); ok {
		fp.Hash = hash
	}

	if codecs, ok := data["video_codecs"].([]interface{}); ok {
		for _, c := range codecs {
			if cs, ok := c.(string); ok {
				fp.VideoCodecs = append(fp.VideoCodecs, cs)
			}
		}
	}

	if codecs, ok := data["audio_codecs"].([]interface{}); ok {
		for _, c := range codecs {
			if cs, ok := c.(string); ok {
				fp.AudioCodecs = append(fp.AudioCodecs, cs)
			}
		}
	}

	if devices, ok := data["media_devices"].([]interface{}); ok {
		for _, d := range devices {
			if dm, ok := d.(map[string]interface{}); ok {
				info := &MediaDeviceInfo{}
				if id, ok := dm["device_id"].(string); ok {
					info.DeviceID = id
				}
				if label, ok := dm["label"].(string); ok {
					info.Label = label
					fp.DeviceLabels = append(fp.DeviceLabels, label)
				}
				if kind, ok := dm["kind"].(string); ok {
					info.Kind = kind
				}
				if gid, ok := dm["group_id"].(string); ok {
					info.GroupID = gid
				}
				info.IsDefault, _ = dm["is_default"].(bool)
				fp.MediaDevices = append(fp.MediaDevices, info)
			}
		}
	}

	if formats, ok := data["supported_formats"].([]interface{}); ok {
		for _, f := range formats {
			if fm, ok := f.(map[string]interface{}); ok {
				info := &MediaFormatInfo{}
				if mt, ok := fm["mime_type"].(string); ok {
					info.MimeType = mt
				}
				info.HasAudio, _ = fm["has_audio"].(bool)
				info.HasVideo, _ = fm["has_video"].(bool)
				info.IsSupported, _ = fm["is_supported"].(bool)
				fp.SupportedFormats = append(fp.SupportedFormats, info)
			}
		}
	}

	if source, ok := data["media_source_support"].(map[string]interface{}); ok {
		fp.MediaSourceSupport = &MediaSourceSupport{}
		fp.MediaSourceSupport.MediaSource, _ = source["media_source"].(bool)
		fp.MediaSourceSupport.SourceBuffer, _ = source["source_buffer"].(bool)
		fp.MediaSourceSupport.WebRTC, _ = source["webrtc"].(bool)
		fp.MediaSourceSupport.MediaStream, _ = source["media_stream"].(bool)
		fp.MediaSourceSupport.MediaRecorder, _ = source["media_recorder"].(bool)
	}

	if video, ok := data["video_capabilities"].(map[string]interface{}); ok {
		fp.VideoCapabilities = &VideoCapabilities{
			SupportedResolutions: make([]string, 0),
		}
		if w, ok := toInt(video["max_width"]); ok {
			fp.VideoCapabilities.MaxWidth = w
		}
		if h, ok := toInt(video["max_height"]); ok {
			fp.VideoCapabilities.MaxHeight = h
		}
		if r, ok := toFloat64(video["max_frame_rate"]); ok {
			fp.VideoCapabilities.MaxFrameRate = r
		}
		if res, ok := video["supported_resolutions"].([]interface{}); ok {
			for _, r := range res {
				if rs, ok := r.(string); ok {
					fp.VideoCapabilities.SupportedResolutions = append(
						fp.VideoCapabilities.SupportedResolutions, rs,
					)
				}
			}
		}
	}

	if audio, ok := data["audio_capabilities"].(map[string]interface{}); ok {
		fp.AudioCapabilities = &AudioCapabilities{
			SampleRates:   make([]int, 0),
			ChannelCounts: make([]int, 0),
		}
		if rates, ok := audio["sample_rates"].([]interface{}); ok {
			for _, r := range rates {
				if ri, ok := toInt(r); ok {
					fp.AudioCapabilities.SampleRates = append(
						fp.AudioCapabilities.SampleRates, ri,
					)
				}
			}
		}
		if counts, ok := audio["channel_counts"].([]interface{}); ok {
			for _, c := range counts {
				if ci, ok := toInt(c); ok {
					fp.AudioCapabilities.ChannelCounts = append(
						fp.AudioCapabilities.ChannelCounts, ci,
					)
				}
			}
		}
		if mc, ok := toInt(audio["max_channel_count"]); ok {
			fp.AudioCapabilities.MaxChannelCount = mc
		}
	}

	for _, device := range fp.MediaDevices {
		switch device.Kind {
		case "videoinput":
			fp.HasCamera = true
		case "audioinput":
			fp.HasMicrophone = true
		case "audiooutput":
			fp.HasSpeaker = true
		}
	}

	fp.DeviceCount = len(fp.MediaDevices)
	fp.UniquenessScore = m.calculateUniquenessScore(fp)

	return fp
}

func (m *MediaFingerprintAnalyzer) CalculateSimilarity(fp1, fp2 *MediaFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	scores := make([]float64, 0)

	if fp1.Hash == fp2.Hash {
		scores = append(scores, 40.0)
	} else {
		scores = append(scores, m.calculateHashSimilarity(fp1.Hash, fp2.Hash)*40.0)
	}

	deviceScore := m.calculateDeviceSimilarity(fp1.MediaDevices, fp2.MediaDevices) * 30.0
	scores = append(scores, deviceScore)

	codecScore := m.calculateCodecSimilarity(fp1.VideoCodecs, fp2.VideoCodecs, fp1.AudioCodecs, fp2.AudioCodecs) * 20.0
	scores = append(scores, codecScore)

	formatScore := m.calculateFormatSimilarity(fp1.SupportedFormats, fp2.SupportedFormats) * 10.0
	scores = append(scores, formatScore)

	return math.Min(100, math.Max(0, average(scores)))
}

func (m *MediaFingerprintAnalyzer) calculateHashSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}
	if minLen == 0 {
		return 0
	}

	matches := 0
	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(math.Max(float64(len(hash1)), float64(len(hash2))))
}

func (m *MediaFingerprintAnalyzer) calculateDeviceSimilarity(dev1, dev2 []*MediaDeviceInfo) float64 {
	if len(dev1) == 0 && len(dev2) == 0 {
		return 1.0
	}
	if len(dev1) == 0 || len(dev2) == 0 {
		return 0
	}

	kindCounts1 := make(map[string]int)
	kindCounts2 := make(map[string]int)
	for _, d := range dev1 {
		kindCounts1[d.Kind]++
	}
	for _, d := range dev2 {
		kindCounts2[d.Kind]++
	}

	scores := make([]float64, 0)
	for kind, count1 := range kindCounts1 {
		count2, exists := kindCounts2[kind]
		if exists {
			minCount := math.Min(float64(count1), float64(count2))
			maxCount := math.Max(float64(count1), float64(count2))
			if maxCount > 0 {
				scores = append(scores, minCount/maxCount)
			}
		}
	}

	if len(scores) == 0 {
		return 0
	}

	return average(scores)
}

func (m *MediaFingerprintAnalyzer) calculateCodecSimilarity(v1, v2, a1, a2 []string) float64 {
	videoSim := m.calculateStringArraySimilarity(v1, v2)
	audioSim := m.calculateStringArraySimilarity(a1, a2)

	return (videoSim + audioSim) / 2.0
}

func (m *MediaFingerprintAnalyzer) calculateStringArraySimilarity(arr1, arr2 []string) float64 {
	if len(arr1) == 0 && len(arr2) == 0 {
		return 1.0
	}
	if len(arr1) == 0 || len(arr2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, s := range arr1 {
		set1[s] = true
	}
	for _, s := range arr2 {
		set2[s] = true
	}

	common := 0
	for s := range set1 {
		if set2[s] {
			common++
		}
	}

	total := len(set1) + len(set2) - common
	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

func (m *MediaFingerprintAnalyzer) calculateFormatSimilarity(formats1, formats2 []*MediaFormatInfo) float64 {
	if len(formats1) == 0 && len(formats2) == 0 {
		return 1.0
	}
	if len(formats1) == 0 || len(formats2) == 0 {
		return 0
	}

	supported1 := 0
	supported2 := 0
	for _, f := range formats1 {
		if f.IsSupported {
			supported1++
		}
	}
	for _, f := range formats2 {
		if f.IsSupported {
			supported2++
		}
	}

	maxSupported := math.Max(float64(supported1), float64(supported2))
	if maxSupported == 0 {
		return 0
	}

	return math.Min(float64(supported1), float64(supported2)) / maxSupported
}

func (m *MediaFingerprintAnalyzer) calculateUniquenessScore(fp *MediaFingerprint) float64 {
	score := 30.0

	if fp.DeviceCount > 3 {
		score += 20
	} else if fp.DeviceCount > 1 {
		score += 10
	}

	if fp.HasCamera && fp.HasMicrophone && fp.HasSpeaker {
		score += 15
	}

	if len(fp.VideoCodecs) > 3 {
		score += 10
	}

	if len(fp.AudioCodecs) > 3 {
		score += 10
	}

	if fp.VideoCapabilities != nil && fp.VideoCapabilities.MaxWidth > 1920 {
		score += 5
	}

	if fp.AudioCapabilities != nil && fp.AudioCapabilities.MaxChannelCount > 2 {
		score += 5
	}

	if len(fp.DeviceLabels) > 0 {
		score += 10
	}

	return math.Min(100, score)
}

type BrowserFingerprintMatcher struct {
	knownBrowsers map[string]*BrowserSignature
}

type BrowserSignature struct {
	Name           string
	Patterns       []*regexp.Regexp
	CanvasPatterns []string
	WebGLPatterns  []string
	FontPatterns   []string
	AudioPatterns  []string
	Weight         float64
}

func NewBrowserFingerprintMatcher() *BrowserFingerprintMatcher {
	matcher := &BrowserFingerprintMatcher{
		knownBrowsers: make(map[string]*BrowserSignature),
	}
	matcher.initializeSignatures()
	return matcher
}

func (b *BrowserFingerprintMatcher) initializeSignatures() {
	b.knownBrowsers["Chrome_Windows"] = &BrowserSignature{
		Name:   "Chrome on Windows",
		Weight: 0.95,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)chrome.*windows`),
			regexp.MustCompile(`(?i)chrome/\d+\.\d+\.\d+\.\d+`),
		},
		CanvasPatterns: []string{"chrome", "intel", "win32"},
		WebGLPatterns:  []string{"google", "angle", "inteldx9", "inteldx11"},
		FontPatterns:   []string{"segoe ui", "arial"},
	}

	b.knownBrowsers["Chrome_Mac"] = &BrowserSignature{
		Name:   "Chrome on macOS",
		Weight: 0.95,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)chrome.*mac os`),
			regexp.MustCompile(`(?i)chrome/\d+\.\d+\.\d+\.\d+`),
		},
		CanvasPatterns: []string{"chrome", "intel", "mac os"},
		WebGLPatterns:  []string{"google", "apple", "intel iris"},
		FontPatterns:   []string{"helvetica", "arial", "lucida"},
	}

	b.knownBrowsers["Chrome_Linux"] = &BrowserSignature{
		Name:   "Chrome on Linux",
		Weight: 0.90,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)chrome.*linux`),
		},
		CanvasPatterns: []string{"chrome", "swiftshader", "mesa"},
		WebGLPatterns:  []string{"google", "swiftshader", "mesa"},
		FontPatterns:   []string{"dejavu", "ubuntu", "liberation"},
	}

	b.knownBrowsers["Firefox_Windows"] = &BrowserSignature{
		Name:   "Firefox on Windows",
		Weight: 0.90,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)firefox.*windows`),
			regexp.MustCompile(`(?i)firefox/\d+\.\d+`),
		},
		CanvasPatterns: []string{"firefox", "cairo", "win32"},
		WebGLPatterns:  []string{"mozilla", "firefox", "direct3d"},
		FontPatterns:   []string{"segoe ui", "arial", "times new roman"},
	}

	b.knownBrowsers["Firefox_Mac"] = &BrowserSignature{
		Name:   "Firefox on macOS",
		Weight: 0.90,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)firefox.*mac os`),
		},
		CanvasPatterns: []string{"firefox", "core graphics", "mac os"},
		WebGLPatterns:  []string{"mozilla", "firefox", "apple"},
		FontPatterns:   []string{"helvetica", "arial", "lucida"},
	}

	b.knownBrowsers["Safari_Mac"] = &BrowserSignature{
		Name:   "Safari on macOS",
		Weight: 0.95,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)safari.*mac os`),
			regexp.MustCompile(`(?i)version/.*safari`),
		},
		CanvasPatterns: []string{"safari", "core graphics", "mac os"},
		WebGLPatterns:  []string{"apple", "safari", "intel"},
		FontPatterns:   []string{"helvetica", "arial", "lucida", "times"},
	}

	b.knownBrowsers["Edge_Chromium"] = &BrowserSignature{
		Name:   "Microsoft Edge (Chromium)",
		Weight: 0.92,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)edg(e)?/`),
			regexp.MustCompile(`(?i)edge.*chrome`),
		},
		CanvasPatterns: []string{"chrome", "edge", "win32"},
		WebGLPatterns:  []string{"google", "edge", "direct3d"},
		FontPatterns:   []string{"segoe ui", "arial", "cambria"},
	}

	b.knownBrowsers["Chrome_Android"] = &BrowserSignature{
		Name:   "Chrome on Android",
		Weight: 0.88,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)chrome.*android`),
			regexp.MustCompile(`(?i)linux.*android`),
		},
		CanvasPatterns: []string{"chrome", "android", "qualcomm"},
		WebGLPatterns:  []string{"google", "qualcomm", "adreno"},
		FontPatterns:   []string{"roboto", "arial", "noto"},
	}

	b.knownBrowsers["Safari_iOS"] = &BrowserSignature{
		Name:   "Safari on iOS",
		Weight: 0.90,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)version/.*mobile/.*safari`),
			regexp.MustCompile(`(?i)iphone|ipad|ipod`),
		},
		CanvasPatterns: []string{"safari", "core animation", "ios"},
		WebGLPatterns:  []string{"apple", "safari", "gpu"},
		FontPatterns:   []string{"helvetica", "arial", "times"},
	}

	b.knownBrowsers["Headless_Chrome"] = &BrowserSignature{
		Name:   "Headless Chrome",
		Weight: 0.70,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless.*chrome`),
			regexp.MustCompile(`(?i)chrome.*headless`),
		},
		CanvasPatterns: []string{"swiftshader", "headless"},
		WebGLPatterns:  []string{"swiftshader", "software"},
		FontPatterns:   []string{},
	}

	b.knownBrowsers["Puppeteer"] = &BrowserSignature{
		Name:   "Puppeteer",
		Weight: 0.60,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)puppeteer`),
		},
		CanvasPatterns: []string{"swiftshader", "headless"},
		WebGLPatterns:  []string{"swiftshader", "software"},
		FontPatterns:   []string{},
	}

	b.knownBrowsers["Playwright"] = &BrowserSignature{
		Name:   "Playwright",
		Weight: 0.60,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)playwright`),
		},
		CanvasPatterns: []string{"swiftshader", "headless"},
		WebGLPatterns:  []string{"swiftshader", "software"},
		FontPatterns:   []string{},
	}

	b.knownBrowsers["Selenium"] = &BrowserSignature{
		Name:   "Selenium",
		Weight: 0.50,
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)selenium|webdriver`),
		},
		CanvasPatterns: []string{"swiftshader", "mesa"},
		WebGLPatterns:  []string{"swiftshader", "software"},
		FontPatterns:   []string{},
	}
}

func (b *BrowserFingerprintMatcher) Match(data map[string]interface{}) *BrowserMatchResult {
	result := &BrowserMatchResult{
		MatchedSignatures: make([]*SignatureMatch, 0),
		Confidence:        0,
	}

	ua := getStringValue(data, "user_agent")
	canvasHash := getStringValue(data, "canvas_hash")
	webglRenderer := getStringValue(data, "webgl_renderer")
	detectedFonts := getStringArrayValue(data, "detected_fonts")

	for _, sig := range b.knownBrowsers {
		score := 0.0
		matchedFeatures := make([]string, 0)

		for _, pattern := range sig.Patterns {
			if pattern.MatchString(ua) {
				score += 30
				matchedFeatures = append(matchedFeatures, "user_agent")
				break
			}
		}

		for _, canvasPattern := range sig.CanvasPatterns {
			if strings.Contains(strings.ToLower(canvasHash), strings.ToLower(canvasPattern)) {
				score += 15
				matchedFeatures = append(matchedFeatures, "canvas:"+canvasPattern)
				break
			}
		}

		for _, webglPattern := range sig.WebGLPatterns {
			if strings.Contains(strings.ToLower(webglRenderer), strings.ToLower(webglPattern)) {
				score += 15
				matchedFeatures = append(matchedFeatures, "webgl:"+webglPattern)
				break
			}
		}

		for _, fontPattern := range sig.FontPatterns {
			for _, font := range detectedFonts {
				if strings.Contains(strings.ToLower(font), strings.ToLower(fontPattern)) {
					score += 5
					matchedFeatures = append(matchedFeatures, "font:"+fontPattern)
					break
				}
			}
		}

		if score > 0 {
			normalizedScore := score / 100.0 * sig.Weight
			if normalizedScore > 0.1 {
				result.MatchedSignatures = append(result.MatchedSignatures, &SignatureMatch{
					SignatureName:    sig.Name,
					MatchScore:       normalizedScore,
					MatchedFeatures:  matchedFeatures,
				})
			}
		}
	}

	sort.Slice(result.MatchedSignatures, func(i, j int) bool {
		return result.MatchedSignatures[i].MatchScore > result.MatchedSignatures[j].MatchScore
	})

	if len(result.MatchedSignatures) > 0 {
		result.BestMatch = result.MatchedSignatures[0].SignatureName
		result.Confidence = result.MatchedSignatures[0].MatchScore
	}

	return result
}

type BrowserMatchResult struct {
	BestMatch         string             `json:"best_match"`
	Confidence        float64            `json:"confidence"`
	MatchedSignatures []*SignatureMatch  `json:"matched_signatures"`
}

type SignatureMatch struct {
	SignatureName   string   `json:"signature_name"`
	MatchScore      float64  `json:"match_score"`
	MatchedFeatures []string `json:"matched_features"`
}

func (e *EnhancedFingerprintAnalyzer) AnalyzeEnhanced(data map[string]interface{}) *EnhancedFingerprintResult {
	result := &EnhancedFingerprintResult{
		Timestamp:           getCurrentTimestamp(),
		CanvasFingerprint:   e.canvasAnalyzer.Analyze(data),
		WebGLFingerprint:    e.webglAnalyzer.Analyze(data),
		FontFingerprint:     e.fontAnalyzer.Analyze(data),
		AudioFingerprint:    e.audioAnalyzer.Analyze(data),
		MediaFingerprint:    e.mediaAnalyzer.Analyze(data),
		BrowserMatch:        e.browserMatcher.Match(data),
	}

	result.CombinedUniquenessScore = e.calculateCombinedUniquenessScore(result)
	result.Fingerprint = e.generateEnhancedFingerprint(result)
	result.EntropyBits = e.calculateEntropyBits(result)

	return result
}

type EnhancedFingerprintResult struct {
	Timestamp              int64                  `json:"timestamp"`
	CanvasFingerprint      *CanvasFingerprint     `json:"canvas_fingerprint"`
	WebGLFingerprint       *WebGLFingerprint      `json:"webgl_fingerprint"`
	FontFingerprint        *FontFingerprint       `json:"font_fingerprint"`
	AudioFingerprint       *AudioFingerprint      `json:"audio_fingerprint"`
	MediaFingerprint       *MediaFingerprint      `json:"media_fingerprint"`
	BrowserMatch           *BrowserMatchResult    `json:"browser_match"`
	CombinedUniquenessScore float64              `json:"combined_uniqueness_score"`
	Fingerprint            string                 `json:"fingerprint"`
	EntropyBits            float64               `json:"entropy_bits"`
}

func (e *EnhancedFingerprintAnalyzer) calculateCombinedUniquenessScore(result *EnhancedFingerprintResult) float64 {
	scores := make([]float64, 0)

	if result.CanvasFingerprint != nil {
		scores = append(scores, result.CanvasFingerprint.UniquenessScore)
	}
	if result.WebGLFingerprint != nil {
		scores = append(scores, result.WebGLFingerprint.UniquenessScore)
	}
	if result.FontFingerprint != nil {
		scores = append(scores, result.FontFingerprint.UniquenessScore)
	}
	if result.AudioFingerprint != nil {
		scores = append(scores, result.AudioFingerprint.UniquenessScore)
	}
	if result.MediaFingerprint != nil {
		scores = append(scores, result.MediaFingerprint.UniquenessScore)
	}

	if len(scores) == 0 {
		return 0
	}

	combined := average(scores)

	if result.BrowserMatch != nil && result.BrowserMatch.Confidence > 0.8 {
		if strings.Contains(strings.ToLower(result.BrowserMatch.BestMatch), "headless") ||
			strings.Contains(strings.ToLower(result.BrowserMatch.BestMatch), "puppeteer") ||
			strings.Contains(strings.ToLower(result.BrowserMatch.BestMatch), "playwright") ||
			strings.Contains(strings.ToLower(result.BrowserMatch.BestMatch), "selenium") {
			combined -= 30
		}
	}

	return math.Min(100, math.Max(0, combined))
}

func (e *EnhancedFingerprintAnalyzer) generateEnhancedFingerprint(result *EnhancedFingerprintResult) string {
	components := make([]string, 0)

	if result.CanvasFingerprint != nil && result.CanvasFingerprint.Hash != "" {
		hashLen := len(result.CanvasFingerprint.Hash)
		if hashLen > 16 {
			components = append(components, "cnv:"+result.CanvasFingerprint.Hash[:16])
		} else {
			components = append(components, "cnv:"+result.CanvasFingerprint.Hash)
		}
	}

	if result.WebGLFingerprint != nil && result.WebGLFingerprint.UnmaskedRenderer != "" {
		rendererHash := sha256.Sum256([]byte(result.WebGLFingerprint.UnmaskedRenderer))
		components = append(components, "wgl:"+hex.EncodeToString(rendererHash[:8]))
	}

	if result.FontFingerprint != nil && len(result.FontFingerprint.DetectedFonts) > 0 {
		fontHash := sha256.Sum256([]byte(strings.Join(result.FontFingerprint.DetectedFonts, ",")))
		components = append(components, "fnt:"+hex.EncodeToString(fontHash[:8]))
	}

	if result.AudioFingerprint != nil && result.AudioFingerprint.Hash != "" {
		hashLen := len(result.AudioFingerprint.Hash)
		if hashLen > 16 {
			components = append(components, "aud:"+result.AudioFingerprint.Hash[:16])
		} else {
			components = append(components, "aud:"+result.AudioFingerprint.Hash)
		}
	}

	if result.MediaFingerprint != nil && len(result.MediaFingerprint.MediaDevices) > 0 {
		deviceCount := len(result.MediaFingerprint.MediaDevices)
		components = append(components, fmt.Sprintf("med:%d", deviceCount))
	}

	combined := strings.Join(components, "|")
	fullHash := sha256.Sum256([]byte(combined))

	return hex.EncodeToString(fullHash[:16])
}

func (e *EnhancedFingerprintAnalyzer) calculateEntropyBits(result *EnhancedFingerprintResult) float64 {
	bits := 0.0

	if result.CanvasFingerprint != nil && result.CanvasFingerprint.Hash != "" {
		bits += 20
	}

	if result.WebGLFingerprint != nil && result.WebGLFingerprint.UniquenessScore > 50 {
		bits += 15
	}

	if result.FontFingerprint != nil && result.FontFingerprint.FontCount > 10 {
		bits += 10
	}

	if result.AudioFingerprint != nil && !result.AudioFingerprint.IsIdenticalAcrossRenders {
		bits += 8
	}

	if result.MediaFingerprint != nil && result.MediaFingerprint.DeviceCount > 2 {
		bits += 5
	}

	return bits
}

func (e *EnhancedFingerprintAnalyzer) CalculateFingerprintSimilarity(data1, data2 map[string]interface{}) *FingerprintSimilarityResult {
	result := &FingerprintSimilarityResult{
		Components: make([]*ComponentSimilarity, 0),
	}

	canvas1 := e.canvasAnalyzer.Analyze(data1)
	canvas2 := e.canvasAnalyzer.Analyze(data2)
	canvasSim := e.canvasAnalyzer.CalculateSimilarity(canvas1, canvas2)
	result.Components = append(result.Components, &ComponentSimilarity{
		Name:        "canvas",
		Score:       canvasSim,
		Weight:      0.25,
	})

	webgl1 := e.webglAnalyzer.Analyze(data1)
	webgl2 := e.webglAnalyzer.Analyze(data2)
	webglSim := e.webglAnalyzer.CalculateSimilarity(webgl1, webgl2)
	result.Components = append(result.Components, &ComponentSimilarity{
		Name:        "webgl",
		Score:       webglSim,
		Weight:      0.25,
	})

	font1 := e.fontAnalyzer.Analyze(data1)
	font2 := e.fontAnalyzer.Analyze(data2)
	fontSim := e.fontAnalyzer.CalculateSimilarity(font1, font2)
	result.Components = append(result.Components, &ComponentSimilarity{
		Name:        "fonts",
		Score:       fontSim,
		Weight:      0.20,
	})

	audio1 := e.audioAnalyzer.Analyze(data1)
	audio2 := e.audioAnalyzer.Analyze(data2)
	audioSim := e.audioAnalyzer.CalculateSimilarity(audio1, audio2)
	result.Components = append(result.Components, &ComponentSimilarity{
		Name:        "audio",
		Score:       audioSim,
		Weight:      0.15,
	})

	media1 := e.mediaAnalyzer.Analyze(data1)
	media2 := e.mediaAnalyzer.Analyze(data2)
	mediaSim := e.mediaAnalyzer.CalculateSimilarity(media1, media2)
	result.Components = append(result.Components, &ComponentSimilarity{
		Name:        "media",
		Score:       mediaSim,
		Weight:      0.15,
	})

	result.OverallSimilarity = result.calculateOverallSimilarity()

	return result
}

type FingerprintSimilarityResult struct {
	OverallSimilarity  float64               `json:"overall_similarity"`
	Components          []*ComponentSimilarity `json:"components"`
}

type ComponentSimilarity struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Weight float64 `json:"weight"`
}

func (r *FingerprintSimilarityResult) calculateOverallSimilarity() float64 {
	if len(r.Components) == 0 {
		return 0
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for _, comp := range r.Components {
		if comp.Score > 0 {
			weightedSum += comp.Score * comp.Weight
			totalWeight += comp.Weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

func getStringValue(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getStringArrayValue(data map[string]interface{}, key string) []string {
	result := make([]string, 0)
	if arr, ok := data[key].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	default:
		return 0, false
	}
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
