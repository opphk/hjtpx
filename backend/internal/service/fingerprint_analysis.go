package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

type FingerprintAnalysis struct {
	FingerprintID       string                 `json:"fingerprint_id"`
	IP                 string                 `json:"ip"`
	CanvasHash         string                 `json:"canvas_hash"`
	WebGLHash          string                 `json:"webgl_hash"`
	AudioHash          string                 `json:"audio_hash"`
	FontHash           string                 `json:"font_hash"`
	PluginHash         string                 `json:"plugin_hash"`
	UserAgent          string                 `json:"user_agent"`
	ScreenResolution   string                 `json:"screen_resolution"`
	Timezone           string                 `json:"timezone"`
	Language           string                 `json:"language"`
	Platform           string                 `json:"platform"`
	HardwareConcurrency int                   `json:"hardware_concurrency"`
	DeviceMemory       float64                `json:"device_memory"`
	FirstSeen          time.Time             `json:"first_seen"`
	LastSeen           time.Time             `json:"last_seen"`
	RequestCount       int                   `json:"request_count"`
	Similarity         float64               `json:"similarity"`
	RiskIndicators     []string              `json:"risk_indicators"`
	AnomalyScore       float64               `json:"anomaly_score"`
	Confidence         float64               `json:"confidence"`
	ClusterID          string                `json:"cluster_id"`
	IsKnownBot         bool                  `json:"is_known_bot"`
	IsKnownVPN         bool                  `json:"is_known_vpn"`
}

type FingerprintDatabase struct {
	fingerprints map[string]*FingerprintAnalysis
	clusters     map[string][]string
	similarityIndex map[string][]string
	mu           sync.RWMutex
	stats        *AnalysisStats
}

type AnalysisStats struct {
	TotalFingerprints int64                  `json:"total_fingerprints"`
	BotFingerprints   int64                  `json:"bot_fingerprints"`
	VPNFingerprints   int64                  `json:"vpn_fingerprints"`
	AvgAnomalyScore   float64                `json:"avg_anomaly_score"`
	HighRiskCount     int64                  `json:"high_risk_count"`
	MediumRiskCount   int64                  `json:"medium_risk_count"`
	LowRiskCount      int64                  `json:"low_risk_count"`
	ClustersCount     int                    `json:"clusters_count"`
}

type SimilarityResult struct {
	FingerprintID string  `json:"fingerprint_id"`
	Similarity   float64 `json:"similarity"`
	CommonFields []string `json:"common_fields"`
	DiffFields   []string `json:"diff_fields"`
}

type AnomalyResult struct {
	IsAnomaly    bool     `json:"is_anomaly"`
	AnomalyType  string   `json:"anomaly_type"`
	Score        float64  `json:"score"`
	Indicators   []string `json:"indicators"`
	Reasons      []string `json:"reasons"`
	Severity     string   `json:"severity"`
}

type ClusterInfo struct {
	ClusterID      string    `json:"cluster_id"`
	Size           int       `json:"size"`
	CommonFeatures []string  `json:"common_features"`
	RiskLevel      string    `json:"risk_level"`
	FingerprintIDs []string  `json:"fingerprint_ids"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
}

func NewFingerprintDatabase() *FingerprintDatabase {
	return &FingerprintDatabase{
		fingerprints:   make(map[string]*FingerprintAnalysis),
		clusters:       make(map[string][]string),
		similarityIndex: make(map[string][]string),
		stats: &AnalysisStats{
			TotalFingerprints: 0,
			BotFingerprints:   0,
			VPNFingerprints:   0,
			AvgAnomalyScore:   0,
		},
	}
}

func (db *FingerprintDatabase) AddFingerprint(fp *FingerprintAnalysis) {
	db.mu.Lock()
	defer db.mu.Unlock()

	fp.FirstSeen = time.Now()
	fp.LastSeen = time.Now()
	fp.RequestCount = 1
	fp.ClusterID = db.assignCluster(fp)
	db.fingerprints[fp.FingerprintID] = fp
	db.updateSimilarityIndex(fp)
	db.updateStats()
}

func (db *FingerprintDatabase) UpdateFingerprint(fpID string, updateFn func(*FingerprintAnalysis)) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if fp, exists := db.fingerprints[fpID]; exists {
		updateFn(fp)
		fp.LastSeen = time.Now()
		fp.RequestCount++
	}
}

func (db *FingerprintDatabase) GetFingerprint(fpID string) (*FingerprintAnalysis, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fp, exists := db.fingerprints[fpID]
	return fp, exists
}

func (db *FingerprintDatabase) GetAllFingerprints() []*FingerprintAnalysis {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]*FingerprintAnalysis, 0, len(db.fingerprints))
	for _, fp := range db.fingerprints {
		result = append(result, fp)
	}
	return result
}

func (db *FingerprintDatabase) CalculateSimilarity(fp1, fp2 *FingerprintAnalysis) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	fields := []struct {
		name   string
		val1   interface{}
		val2   interface{}
		weight float64
	}{
		{"canvas", fp1.CanvasHash, fp2.CanvasHash, 0.15},
		{"webgl", fp1.WebGLHash, fp2.WebGLHash, 0.15},
		{"audio", fp1.AudioHash, fp2.AudioHash, 0.10},
		{"fonts", fp1.FontHash, fp2.FontHash, 0.10},
		{"plugins", fp1.PluginHash, fp2.PluginHash, 0.05},
		{"user_agent", fp1.UserAgent, fp2.UserAgent, 0.15},
		{"screen", fp1.ScreenResolution, fp2.ScreenResolution, 0.10},
		{"timezone", fp1.Timezone, fp2.Timezone, 0.05},
		{"language", fp1.Language, fp2.Language, 0.05},
		{"platform", fp1.Platform, fp2.Platform, 0.10},
	}

	totalWeight := 0.0
	matchWeight := 0.0

	for _, field := range fields {
		totalWeight += field.weight
		if fmt.Sprintf("%v", field.val1) == fmt.Sprintf("%v", field.val2) &&
		   fmt.Sprintf("%v", field.val1) != "" &&
		   fmt.Sprintf("%v", field.val2) != "" {
			matchWeight += field.weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return matchWeight / totalWeight * 100
}

func (db *FingerprintDatabase) FindSimilarFingerprints(fpID string, threshold float64) []SimilarityResult {
	db.mu.RLock()
	defer db.mu.RUnlock()

	target, exists := db.fingerprints[fpID]
	if !exists {
		return nil
	}

	results := make([]SimilarityResult, 0)

	for id, fp := range db.fingerprints {
		if id == fpID {
			continue
		}

		similarity := db.CalculateSimilarity(target, fp)
		if similarity >= threshold {
			commonFields := db.getCommonFields(target, fp)
			diffFields := db.getDiffFields(target, fp)
			results = append(results, SimilarityResult{
				FingerprintID: id,
				Similarity:    similarity,
				CommonFields:  commonFields,
				DiffFields:    diffFields,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return results
}

func (db *FingerprintDatabase) getCommonFields(fp1, fp2 *FingerprintAnalysis) []string {
	fields := []string{}

	if fp1.CanvasHash == fp2.CanvasHash && fp1.CanvasHash != "" {
		fields = append(fields, "canvas")
	}
	if fp1.WebGLHash == fp2.WebGLHash && fp1.WebGLHash != "" {
		fields = append(fields, "webgl")
	}
	if fp1.AudioHash == fp2.AudioHash && fp1.AudioHash != "" {
		fields = append(fields, "audio")
	}
	if fp1.FontHash == fp2.FontHash && fp1.FontHash != "" {
		fields = append(fields, "fonts")
	}
	if fp1.UserAgent == fp2.UserAgent && fp1.UserAgent != "" {
		fields = append(fields, "user_agent")
	}
	if fp1.ScreenResolution == fp2.ScreenResolution && fp1.ScreenResolution != "" {
		fields = append(fields, "screen")
	}
	if fp1.Timezone == fp2.Timezone && fp1.Timezone != "" {
		fields = append(fields, "timezone")
	}
	if fp1.Language == fp2.Language && fp1.Language != "" {
		fields = append(fields, "language")
	}

	return fields
}

func (db *FingerprintDatabase) getDiffFields(fp1, fp2 *FingerprintAnalysis) []string {
	fields := []string{}

	if fp1.CanvasHash != fp2.CanvasHash {
		fields = append(fields, "canvas")
	}
	if fp1.WebGLHash != fp2.WebGLHash {
		fields = append(fields, "webgl")
	}
	if fp1.AudioHash != fp2.AudioHash {
		fields = append(fields, "audio")
	}
	if fp1.IP != fp2.IP {
		fields = append(fields, "ip")
	}

	return fields
}

func (db *FingerprintDatabase) DetectAnomaly(fpID string) *AnomalyResult {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fp, exists := db.fingerprints[fpID]
	if !exists {
		return &AnomalyResult{
			IsAnomaly:   false,
			AnomalyType: "not_found",
			Score:       0,
		}
	}

	result := &AnomalyResult{
		Indicators: make([]string, 0),
		Reasons:    make([]string, 0),
	}

	if fp.RequestCount == 0 {
		result.Indicators = append(result.Indicators, "no_requests")
		result.Reasons = append(result.Reasons, "指纹无请求记录")
	}

	similarFps := db.FindSimilarFingerprints(fpID, 80)
	if len(similarFps) > 10 {
		result.Indicators = append(result.Indicators, "high_similarity_count")
		result.Reasons = append(result.Reasons, fmt.Sprintf("发现%d个相似度>80%%的指纹", len(similarFps)))
		result.Score += 20
	}

	knownBotPatterns := []string{"headless", "phantom", "puppeteer", "playwright", "selenium", "webdriver"}
	for _, pattern := range knownBotPatterns {
		if strings.Contains(strings.ToLower(fp.UserAgent), pattern) {
			result.Indicators = append(result.Indicators, "known_bot_pattern")
			result.Reasons = append(result.Reasons, fmt.Sprintf("检测到自动化工具标识: %s", pattern))
			result.Score += 30
			result.IsAnomaly = true
			result.AnomalyType = "automation"
		}
	}

	if fp.CanvasHash == "" || fp.WebGLHash == "" {
		result.Indicators = append(result.Indicators, "missing_fingerprint")
		result.Reasons = append(result.Reasons, "缺少关键指纹数据")
		result.Score += 25
	}

	timeSinceLastSeen := time.Since(fp.LastSeen)
	if fp.RequestCount > 100 && timeSinceLastSeen < 1*time.Minute {
		result.Indicators = append(result.Indicators, "high_frequency")
		result.Reasons = append(result.Reasons, fmt.Sprintf("高频请求: %d次请求在短时间内", fp.RequestCount))
		result.Score += 35
	}

	if fp.Similarity > 95 && len(similarFps) > 5 {
		result.Indicators = append(result.Indicators, "fingerprint_collision")
		result.Reasons = append(result.Reasons, "检测到指纹冲突，可能是指纹复用攻击")
		result.Score += 40
	}

	result.Score = math.Min(result.Score, 100)
	result.IsAnomaly = result.Score > 30
	result.Severity = db.getSeverity(result.Score)

	if result.Score > 0 && result.AnomalyType == "" {
		result.AnomalyType = "general"
	}

	return result
}

func (db *FingerprintDatabase) getSeverity(score float64) string {
	if score >= 70 {
		return "high"
	} else if score >= 40 {
		return "medium"
	}
	return "low"
}

func (db *FingerprintDatabase) assignCluster(fp *FingerprintAnalysis) string {
	bestMatch := ""
	bestSimilarity := 0.0

	for _, existingFP := range db.fingerprints {
		similarity := db.CalculateSimilarity(fp, existingFP)
		if similarity > bestSimilarity && similarity > 70 {
			bestSimilarity = similarity
			bestMatch = existingFP.ClusterID
		}
	}

	if bestMatch != "" {
		db.clusters[bestMatch] = append(db.clusters[bestMatch], fp.FingerprintID)
		return bestMatch
	}

	clusterID := fmt.Sprintf("cluster_%d", len(db.clusters)+1)
	db.clusters[clusterID] = []string{fp.FingerprintID}
	return clusterID
}

func (db *FingerprintDatabase) updateSimilarityIndex(fp *FingerprintAnalysis) {
	for otherID, otherFP := range db.fingerprints {
		if otherID == fp.FingerprintID {
			continue
		}

		similarity := db.CalculateSimilarity(fp, otherFP)
		if similarity > 60 {
			db.similarityIndex[fp.FingerprintID] = append(
				db.similarityIndex[fp.FingerprintID], otherID,
			)
			db.similarityIndex[otherID] = append(
				db.similarityIndex[otherID], fp.FingerprintID,
			)
		}
	}
}

func (db *FingerprintDatabase) updateStats() {
	stats := &AnalysisStats{
		TotalFingerprints: int64(len(db.fingerprints)),
		HighRiskCount:     0,
		MediumRiskCount:   0,
		LowRiskCount:      0,
		ClustersCount:     len(db.clusters),
	}

	var totalAnomalyScore float64

	for _, fp := range db.fingerprints {
		if fp.IsKnownBot {
			stats.BotFingerprints++
		}
		if fp.IsKnownVPN {
			stats.VPNFingerprints++
		}

		totalAnomalyScore += fp.AnomalyScore

		if fp.AnomalyScore >= 70 {
			stats.HighRiskCount++
		} else if fp.AnomalyScore >= 40 {
			stats.MediumRiskCount++
		} else {
			stats.LowRiskCount++
		}
	}

	if stats.TotalFingerprints > 0 {
		stats.AvgAnomalyScore = totalAnomalyScore / float64(stats.TotalFingerprints)
	}

	db.stats = stats
}

func (db *FingerprintDatabase) GetStats() *AnalysisStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.stats
}

func (db *FingerprintDatabase) GetCluster(clusterID string) *ClusterInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fpIDs, exists := db.clusters[clusterID]
	if !exists {
		return nil
	}

	info := &ClusterInfo{
		ClusterID:      clusterID,
		Size:           len(fpIDs),
		FingerprintIDs:  fpIDs,
		CommonFeatures: make([]string, 0),
		RiskLevel:      "low",
	}

	var firstSeen, lastSeen time.Time
	featureCounts := make(map[string]int)

	for _, fpID := range fpIDs {
		if fp, exists := db.fingerprints[fpID]; exists {
			if firstSeen.IsZero() || fp.FirstSeen.Before(firstSeen) {
				firstSeen = fp.FirstSeen
			}
			if lastSeen.IsZero() || fp.LastSeen.After(lastSeen) {
				lastSeen = fp.LastSeen
			}

			if fp.UserAgent != "" {
				featureCounts["user_agent"]++
			}
			if fp.CanvasHash != "" {
				featureCounts["canvas"]++
			}
			if fp.WebGLHash != "" {
				featureCounts["webgl"]++
			}
			if fp.Timezone != "" {
				featureCounts["timezone"]++
			}
		}
	}

	info.FirstSeen = firstSeen
	info.LastSeen = lastSeen

	for feature, count := range featureCounts {
		if float64(count)/float64(len(fpIDs)) > 0.7 {
			info.CommonFeatures = append(info.CommonFeatures, feature)
		}
	}

	highRisk := 0
	for _, fpID := range fpIDs {
		if fp, exists := db.fingerprints[fpID]; exists {
			if fp.AnomalyScore >= 70 {
				highRisk++
			}
		}
	}

	if float64(highRisk)/float64(len(fpIDs)) > 0.5 {
		info.RiskLevel = "high"
	} else if float64(highRisk)/float64(len(fpIDs)) > 0.2 {
		info.RiskLevel = "medium"
	}

	return info
}

func (db *FingerprintDatabase) GetAllClusters() []*ClusterInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	clusters := make([]*ClusterInfo, 0, len(db.clusters))
	for clusterID := range db.clusters {
		info := db.GetCluster(clusterID)
		if info != nil {
			clusters = append(clusters, info)
		}
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Size > clusters[j].Size
	})

	return clusters
}

func (db *FingerprintDatabase) RemoveFingerprint(fpID string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if fp, exists := db.fingerprints[fpID]; exists {
		clusterID := fp.ClusterID
		if fps, exists := db.clusters[clusterID]; exists {
			newFps := make([]string, 0)
			for _, id := range fps {
				if id != fpID {
					newFps = append(newFps, id)
				}
			}
			if len(newFps) > 0 {
				db.clusters[clusterID] = newFps
			} else {
				delete(db.clusters, clusterID)
			}
		}

		delete(db.similarityIndex, fpID)
		for otherID, similar := range db.similarityIndex {
			newSimilar := make([]string, 0)
			for _, simID := range similar {
				if simID != fpID {
					newSimilar = append(newSimilar, simID)
				}
			}
			db.similarityIndex[otherID] = newSimilar
		}
	}

	delete(db.fingerprints, fpID)
	db.updateStats()
}

func (db *FingerprintDatabase) CleanupOldData(maxAge time.Duration) int {
	db.mu.Lock()
	defer db.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for fpID, fp := range db.fingerprints {
		if fp.LastSeen.Before(cutoff) && fp.RequestCount < 5 {
			delete(db.fingerprints, fpID)
			removed++
		}
	}

	db.updateStatsLocked()
	return removed
}

func (db *FingerprintDatabase) updateStatsLocked() {
	stats := &AnalysisStats{
		TotalFingerprints: int64(len(db.fingerprints)),
		HighRiskCount:     0,
		MediumRiskCount:   0,
		LowRiskCount:      0,
		ClustersCount:     len(db.clusters),
	}

	var totalAnomalyScore float64

	for _, fp := range db.fingerprints {
		if fp.IsKnownBot {
			stats.BotFingerprints++
		}
		if fp.IsKnownVPN {
			stats.VPNFingerprints++
		}

		totalAnomalyScore += fp.AnomalyScore

		if fp.AnomalyScore >= 70 {
			stats.HighRiskCount++
		} else if fp.AnomalyScore >= 40 {
			stats.MediumRiskCount++
		} else {
			stats.LowRiskCount++
		}
	}

	if stats.TotalFingerprints > 0 {
		stats.AvgAnomalyScore = totalAnomalyScore / float64(stats.TotalFingerprints)
	}

	db.stats = stats
}

func (db *FingerprintDatabase) ExportData() ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data := map[string]interface{}{
		"fingerprints": db.fingerprints,
		"clusters":     db.clusters,
		"stats":        db.stats,
		"exported_at":  time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

func (db *FingerprintDatabase) ImportData(data []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	type ImportData struct {
		Fingerprints map[string]*FingerprintAnalysis `json:"fingerprints"`
		Clusters     map[string][]string            `json:"clusters"`
	}

	var importData ImportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return err
	}

	for id, fp := range importData.Fingerprints {
		fp.FingerprintID = id
		db.fingerprints[id] = fp
	}

	for clusterID, fpIDs := range importData.Clusters {
		db.clusters[clusterID] = fpIDs
	}

	db.updateStatsLocked()
	return nil
}

type FingerprintAnalyzer struct {
	database    *FingerprintDatabase
	knownBots   map[string]bool
	knownVPNs    map[string]bool
	confidenceThreshold float64
	mu          sync.RWMutex
}

func NewFingerprintAnalyzer() *FingerprintAnalyzer {
	return &FingerprintAnalyzer{
		database:          NewFingerprintDatabase(),
		knownBots:         make(map[string]bool),
		knownVPNs:         make(map[string]bool),
		confidenceThreshold: 0.85,
	}
}

func (a *FingerprintAnalyzer) AnalyzeFingerprint(data map[string]interface{}) (*FingerprintAnalysis, *AnomalyResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fp := &FingerprintAnalysis{
		FingerprintID: generateFingerprintID(data),
		CanvasHash:    getString(data, "canvas_hash"),
		WebGLHash:     getString(data, "webgl_hash"),
		AudioHash:     getString(data, "audio_hash"),
		FontHash:      getString(data, "font_hash"),
		PluginHash:    getString(data, "plugin_hash"),
		UserAgent:     getString(data, "user_agent"),
		ScreenResolution: getString(data, "screen_resolution"),
		Timezone:      getString(data, "timezone"),
		Language:      getString(data, "language"),
		Platform:      getString(data, "platform"),
		FirstSeen:     time.Now(),
		LastSeen:      time.Now(),
		RequestCount:  1,
	}

	if hwConcurrency, ok := data["hardware_concurrency"].(float64); ok {
		fp.HardwareConcurrency = int(hwConcurrency)
	}
	if deviceMemory, ok := data["device_memory"].(float64); ok {
		fp.DeviceMemory = deviceMemory
	}

	a.detectBotIndicators(fp)
	a.detectVPNIndicators(fp)

	anomaly := a.database.DetectAnomaly(fp.FingerprintID)
	fp.AnomalyScore = anomaly.Score
	fp.RiskIndicators = anomaly.Indicators
	fp.Confidence = a.calculateConfidence(fp)

	similarFps := a.database.FindSimilarFingerprints(fp.FingerprintID, 70)
	if len(similarFps) > 0 {
		fp.Similarity = similarFps[0].Similarity
	}

	a.database.AddFingerprint(fp)

	return fp, anomaly, nil
}

func (a *FingerprintAnalyzer) detectBotIndicators(fp *FingerprintAnalysis) {
	botPatterns := []struct {
		pattern  string
		indicator string
	}{
		{"headless", "headless_browser"},
		{"phantom", "phantom_js"},
		{"puppeteer", "puppeteer"},
		{"playwright", "playwright"},
		{"selenium", "selenium"},
		{"webdriver", "webdriver"},
		{"chrome-headless", "chrome_headless"},
		{"firefox-headless", "firefox_headless"},
	}

	uaLower := strings.ToLower(fp.UserAgent)
	for _, bp := range botPatterns {
		if strings.Contains(uaLower, bp.pattern) {
			fp.RiskIndicators = append(fp.RiskIndicators, bp.indicator)
			fp.IsKnownBot = true
			fp.AnomalyScore = math.Max(fp.AnomalyScore, 50)
		}
	}

	if fp.CanvasHash == "" && fp.WebGLHash == "" {
		fp.RiskIndicators = append(fp.RiskIndicators, "missing_fingerprints")
		fp.AnomalyScore = math.Max(fp.AnomalyScore, 40)
	}
}

func (a *FingerprintAnalyzer) detectVPNIndicators(fp *FingerprintAnalysis) {
	if len(fp.RiskIndicators) == 0 {
		return
	}

	vpnIndicators := []string{"proxy_detected", "vpn_detected", "tor_exit_node", "ip_mismatch"}
	for _, indicator := range vpnIndicators {
		for _, risk := range fp.RiskIndicators {
			if strings.Contains(strings.ToLower(risk), indicator) {
				fp.IsKnownVPN = true
				fp.AnomalyScore = math.Max(fp.AnomalyScore, 45)
				return
			}
		}
	}
}

func (a *FingerprintAnalyzer) calculateConfidence(fp *FingerprintAnalysis) float64 {
	fields := 0
	complete := 0

	if fp.CanvasHash != "" {
		fields++
		complete++
	}
	if fp.WebGLHash != "" {
		fields++
		complete++
	}
	if fp.AudioHash != "" {
		fields++
		complete++
	}
	if fp.FontHash != "" {
		fields++
		complete++
	}
	if fp.UserAgent != "" {
		fields++
		complete++
	}
	if fp.ScreenResolution != "" {
		fields++
		complete++
	}

	if fields == 0 {
		return 0
	}

	return float64(complete) / float64(fields)
}

func (a *FingerprintAnalyzer) GetFingerprint(fpID string) (*FingerprintAnalysis, bool) {
	return a.database.GetFingerprint(fpID)
}

func (a *FingerprintAnalyzer) GetSimilarFingerprints(fpID string, threshold float64) []SimilarityResult {
	return a.database.FindSimilarFingerprints(fpID, threshold)
}

func (a *FingerprintAnalyzer) GetAnomaly(fpID string) *AnomalyResult {
	return a.database.DetectAnomaly(fpID)
}

func (a *FingerprintAnalyzer) GetDatabase() *FingerprintDatabase {
	return a.database
}

func (a *FingerprintAnalyzer) GetStats() *AnalysisStats {
	return a.database.GetStats()
}

func (a *FingerprintAnalyzer) GetClusters() []*ClusterInfo {
	return a.database.GetAllClusters()
}

func generateFingerprintID(data map[string]interface{}) string {
	hasher := sha256.New()

	if ua, ok := data["user_agent"].(string); ok {
		hasher.Write([]byte(ua))
	}
	if canvas, ok := data["canvas_hash"].(string); ok {
		hasher.Write([]byte(canvas))
	}
	if webgl, ok := data["webgl_hash"].(string); ok {
		hasher.Write([]byte(webgl))
	}
	if screen, ok := data["screen_resolution"].(string); ok {
		hasher.Write([]byte(screen))
	}
	if timezone, ok := data["timezone"].(string); ok {
		hasher.Write([]byte(timezone))
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)[:16] + fmt.Sprintf("_%d", rand.Intn(10000))
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

type ExtendedFingerprintAnalysis struct {
	BaseAnalysis *FingerprintAnalysis
	NetworkAnalysis map[string]interface{} `json:"network_analysis"`
	BehavioralAnalysis map[string]interface{} `json:"behavioral_analysis"`
	HistoricalComparison map[string]interface{} `json:"historical_comparison"`
	AccuracyScore float64 `json:"accuracy_score"`
	PredictionScore float64 `json:"prediction_score"`
}

func (a *FingerprintAnalyzer) AnalyzeWithExtendedMetrics(data map[string]interface{}) (*ExtendedFingerprintAnalysis, error) {
	base, anomaly, err := a.AnalyzeFingerprint(data)
	if err != nil {
		return nil, err
	}

	extended := &ExtendedFingerprintAnalysis{
		BaseAnalysis:          base,
		NetworkAnalysis:       a.analyzeNetworkMetrics(data),
		BehavioralAnalysis:    a.analyzeBehavioralMetrics(data),
		HistoricalComparison:  a.compareWithHistory(base),
		AccuracyScore:         a.calculateAccuracyScore(base),
		PredictionScore:       a.calculatePredictionScore(base, anomaly),
	}

	return extended, nil
}

func (a *FingerprintAnalyzer) analyzeNetworkMetrics(data map[string]interface{}) map[string]interface{} {
	metrics := make(map[string]interface{})

	if webrtcIPs, ok := data["webrtc_ips"].([]interface{}); ok {
		metrics["webrtc_ip_count"] = len(webrtcIPs)
		metrics["webrtc_leak_risk"] = len(webrtcIPs) > 1
	}

	if connType, ok := data["connection_type"].(string); ok {
		metrics["connection_type"] = connType
		metrics["is_vpn_type"] = connType == "vpn" || connType == "cellular"
	}

	return metrics
}

func (a *FingerprintAnalyzer) analyzeBehavioralMetrics(data map[string]interface{}) map[string]interface{} {
	metrics := make(map[string]interface{})

	if requestInterval, ok := data["request_interval"].(float64); ok {
		metrics["avg_request_interval"] = requestInterval
		metrics["is_automated"] = requestInterval < 1.0
	}

	if pathPattern, ok := data["request_paths"].([]interface{}); ok {
		metrics["unique_paths"] = len(pathPattern)
		metrics["path_diversity"] = float64(len(pathPattern)) / 100.0
	}

	return metrics
}

func (a *FingerprintAnalyzer) compareWithHistory(base *FingerprintAnalysis) map[string]interface{} {
	comparison := make(map[string]interface{})

	similar := a.database.FindSimilarFingerprints(base.FingerprintID, 80)
	comparison["similar_fingerprints_count"] = len(similar)
	comparison["similarity_score"] = base.Similarity

	if len(similar) > 0 {
		comparison["first_similar_seen"] = similar[0].FingerprintID
	}

	return comparison
}

func (a *FingerprintAnalyzer) calculateAccuracyScore(fp *FingerprintAnalysis) float64 {
	score := 0.0
	components := 0

	if fp.CanvasHash != "" {
		score += 20
	}
	components += 20

	if fp.WebGLHash != "" {
		score += 20
	}
	components += 20

	if fp.AudioHash != "" {
		score += 15
	}
	components += 15

	if fp.FontHash != "" {
		score += 15
	}
	components += 15

	if fp.UserAgent != "" {
		score += 15
	}
	components += 15

	if fp.ScreenResolution != "" {
		score += 15
	}
	components += 15

	return math.Min(score, 100)
}

func (a *FingerprintAnalyzer) calculatePredictionScore(fp *FingerprintAnalysis, anomaly *AnomalyResult) float64 {
	score := 50.0

	if fp.IsKnownBot {
		score += 30
	}
	if fp.IsKnownVPN {
		score += 20
	}
	if fp.AnomalyScore > 70 {
		score += 25
	} else if fp.AnomalyScore > 40 {
		score += 15
	}
	if fp.Confidence > 0.9 {
		score += 10
	}

	if fp.RequestCount > 100 {
		score += 5
	}

	return math.Min(score, 100)
}
