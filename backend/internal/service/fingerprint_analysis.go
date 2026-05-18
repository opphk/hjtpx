package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type FingerprintAnalysis struct {
	FingerprintID       string    `json:"fingerprint_id"`
	IP                  string    `json:"ip"`
	CanvasHash          string    `json:"canvas_hash"`
	WebGLHash           string    `json:"webgl_hash"`
	AudioHash           string    `json:"audio_hash"`
	FontHash            string    `json:"font_hash"`
	PluginHash          string    `json:"plugin_hash"`
	UserAgent           string    `json:"user_agent"`
	ScreenResolution    string    `json:"screen_resolution"`
	Timezone            string    `json:"timezone"`
	Language            string    `json:"language"`
	Platform            string    `json:"platform"`
	HardwareConcurrency int       `json:"hardware_concurrency"`
	DeviceMemory        float64   `json:"device_memory"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	RequestCount        int       `json:"request_count"`
	Similarity          float64   `json:"similarity"`
	RiskIndicators      []string  `json:"risk_indicators"`
	AnomalyScore        float64   `json:"anomaly_score"`
	Confidence          float64   `json:"confidence"`
	ClusterID           string    `json:"cluster_id"`
	IsKnownBot          bool      `json:"is_known_bot"`
	IsKnownVPN          bool      `json:"is_known_vpn"`
}

type FingerprintDatabase struct {
	fingerprints    map[string]*FingerprintAnalysis
	clusters        map[string][]string
	similarityIndex map[string][]string
	mu              sync.RWMutex
	stats           *AnalysisStats
}

type AnalysisStats struct {
	TotalFingerprints int64   `json:"total_fingerprints"`
	BotFingerprints   int64   `json:"bot_fingerprints"`
	VPNFingerprints   int64   `json:"vpn_fingerprints"`
	AvgAnomalyScore   float64 `json:"avg_anomaly_score"`
	HighRiskCount     int64   `json:"high_risk_count"`
	MediumRiskCount   int64   `json:"medium_risk_count"`
	LowRiskCount      int64   `json:"low_risk_count"`
	ClustersCount     int     `json:"clusters_count"`
}

type SimilarityResult struct {
	FingerprintID string   `json:"fingerprint_id"`
	Similarity    float64  `json:"similarity"`
	CommonFields  []string `json:"common_fields"`
	DiffFields    []string `json:"diff_fields"`
}

type AnomalyResult struct {
	IsAnomaly   bool     `json:"is_anomaly"`
	AnomalyType string   `json:"anomaly_type"`
	Score       float64  `json:"score"`
	Indicators  []string `json:"indicators"`
	Reasons     []string `json:"reasons"`
	Severity    string   `json:"severity"`
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
		fingerprints:    make(map[string]*FingerprintAnalysis),
		clusters:        make(map[string][]string),
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
		FingerprintIDs: fpIDs,
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
		Clusters     map[string][]string             `json:"clusters"`
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
	database            *FingerprintDatabase
	knownBots           map[string]bool
	knownVPNs           map[string]bool
	confidenceThreshold float64
	mu                  sync.RWMutex
}

func NewFingerprintAnalyzer() *FingerprintAnalyzer {
	return &FingerprintAnalyzer{
		database:            NewFingerprintDatabase(),
		knownBots:           make(map[string]bool),
		knownVPNs:           make(map[string]bool),
		confidenceThreshold: 0.85,
	}
}

func (a *FingerprintAnalyzer) AnalyzeFingerprint(data map[string]interface{}) (*FingerprintAnalysis, *AnomalyResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fp := &FingerprintAnalysis{
		FingerprintID:    generateFingerprintID(data),
		CanvasHash:       getString(data, "canvas_hash"),
		WebGLHash:        getString(data, "webgl_hash"),
		AudioHash:        getString(data, "audio_hash"),
		FontHash:         getString(data, "font_hash"),
		PluginHash:       getString(data, "plugin_hash"),
		UserAgent:        getString(data, "user_agent"),
		ScreenResolution: getString(data, "screen_resolution"),
		Timezone:         getString(data, "timezone"),
		Language:         getString(data, "language"),
		Platform:         getString(data, "platform"),
		FirstSeen:        time.Now(),
		LastSeen:         time.Now(),
		RequestCount:     1,
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
		pattern   string
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
	BaseAnalysis         *FingerprintAnalysis
	NetworkAnalysis      map[string]interface{} `json:"network_analysis"`
	BehavioralAnalysis   map[string]interface{} `json:"behavioral_analysis"`
	HistoricalComparison map[string]interface{} `json:"historical_comparison"`
	AccuracyScore        float64                `json:"accuracy_score"`
	PredictionScore      float64                `json:"prediction_score"`
}

func (a *FingerprintAnalyzer) AnalyzeWithExtendedMetrics(data map[string]interface{}) (*ExtendedFingerprintAnalysis, error) {
	base, anomaly, err := a.AnalyzeFingerprint(data)
	if err != nil {
		return nil, err
	}

	extended := &ExtendedFingerprintAnalysis{
		BaseAnalysis:         base,
		NetworkAnalysis:      a.analyzeNetworkMetrics(data),
		BehavioralAnalysis:   a.analyzeBehavioralMetrics(data),
		HistoricalComparison: a.compareWithHistory(base),
		AccuracyScore:        a.calculateAccuracyScore(base),
		PredictionScore:      a.calculatePredictionScore(base, anomaly),
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

type EnhancedFingerprintMetrics struct {
	CanvasMetrics       *CanvasMetrics       `json:"canvas_metrics"`
	WebGLMetrics        *WebGLMetrics        `json:"webgl_metrics"`
	FontMetrics         *FontMetrics         `json:"font_metrics"`
	ScreenMetrics       *ScreenMetrics       `json:"screen_metrics"`
	UniquenessScore     float64              `json:"uniqueness_score"`
	BrowserSignature    string               `json:"browser_signature"`
	MultiBrowserCompare *MultiBrowserCompare `json:"multi_browser_compare"`
}

type CanvasMetrics struct {
	Hash                 string   `json:"hash"`
	RgbaDistribution     []int    `json:"rgba_distribution"`
	NoiseLevel           float64  `json:"noise_level"`
	RenderingConsistency float64  `json:"rendering_consistency"`
	IsHeadlessRenderer   bool     `json:"is_headless_renderer"`
	SoftwareRenderer     bool     `json:"software_renderer"`
	Details              []string `json:"details"`
}

type WebGLMetrics struct {
	Hash                string   `json:"hash"`
	Vendor              string   `json:"vendor"`
	Renderer            string   `json:"renderer"`
	MaxTextureSize      int      `json:"max_texture_size"`
	MaxRenderbufferSize int      `json:"max_renderbuffer_size"`
	MaxVertexAttribs    int      `json:"max_vertex_attribs"`
	SupportedExtensions int      `json:"supported_extensions"`
	UnmaskedVendor      string   `json:"unmasked_vendor"`
	UnmaskedRenderer    string   `json:"unmasked_renderer"`
	IsSoftwareRenderer  bool     `json:"is_software_renderer"`
	IsVirtualGPU        bool     `json:"is_virtual_gpu"`
	PrecisionLoss       bool     `json:"precision_loss"`
	Details             []string `json:"details"`
}

type FontMetrics struct {
	Hash                string   `json:"hash"`
	DetectedFonts       []string `json:"detected_fonts"`
	FontCount           int      `json:"font_count"`
	CommonFontMissing   []string `json:"common_font_missing"`
	FontFamilyDiversity float64  `json:"font_family_diversity"`
	IsLimitedFontSet    bool     `json:"is_limited_font_set"`
}

type ScreenMetrics struct {
	Resolution         string  `json:"resolution"`
	ColorDepth         int     `json:"color_depth"`
	PixelRatio         float64 `json:"pixel_ratio"`
	AvailWidth         int     `json:"avail_width"`
	AvailHeight        int     `json:"avail_height"`
	DevicePixelRatio   float64 `json:"device_pixel_ratio"`
	Orientation        string  `json:"orientation"`
	IsCommonResolution bool    `json:"is_common_resolution"`
}

type MultiBrowserCompare struct {
	ComparedBrowsers  []string  `json:"compared_browsers"`
	SimilarityScores  []float64 `json:"similarity_scores"`
	IsUniqueSignature bool      `json:"is_unique_signature"`
	CollisionRisk     float64   `json:"collision_risk"`
}

func (a *FingerprintAnalyzer) AnalyzeEnhancedMetrics(data map[string]interface{}) (*EnhancedFingerprintMetrics, error) {
	metrics := &EnhancedFingerprintMetrics{}

	metrics.CanvasMetrics = a.analyzeCanvasEnhanced(data)
	metrics.WebGLMetrics = a.analyzeWebGLEnhanced(data)
	metrics.FontMetrics = a.analyzeFontsEnhanced(data)
	metrics.ScreenMetrics = a.analyzeScreenEnhanced(data)
	metrics.BrowserSignature = a.generateBrowserSignature(data)
	metrics.MultiBrowserCompare = a.compareWithKnownBrowsers(data)
	metrics.UniquenessScore = a.calculateUniquenessScore(metrics)

	return metrics, nil
}

func (a *FingerprintAnalyzer) analyzeCanvasEnhanced(data map[string]interface{}) *CanvasMetrics {
	metrics := &CanvasMetrics{
		Details: make([]string, 0),
	}

	if hash, ok := data["canvas_hash"].(string); ok {
		metrics.Hash = hash
	}

	if rgbaData, ok := data["canvas_rgba_distribution"].([]interface{}); ok {
		for _, v := range rgbaData {
			if fv, ok := v.(float64); ok {
				metrics.RgbaDistribution = append(metrics.RgbaDistribution, int(fv))
			}
		}
	}

	if noiseLevel, ok := data["canvas_noise_level"].(float64); ok {
		metrics.NoiseLevel = noiseLevel
		if noiseLevel < 0.01 {
			metrics.IsHeadlessRenderer = true
			metrics.Details = append(metrics.Details, "low_noise_headless")
		}
	}

	if consistency, ok := data["canvas_rendering_consistency"].(float64); ok {
		metrics.RenderingConsistency = consistency
		if consistency > 0.99 {
			metrics.Details = append(metrics.Details, "too_consistent_rendering")
		}
	}

	if renderer, ok := data["canvas_renderer"].(string); ok {
		if strings.Contains(strings.ToLower(renderer), "swiftshader") ||
			strings.Contains(strings.ToLower(renderer), "llvmpipe") ||
			strings.Contains(strings.ToLower(renderer), "mesa") {
			metrics.SoftwareRenderer = true
			metrics.IsHeadlessRenderer = true
			metrics.Details = append(metrics.Details, "software_renderer_detected")
		}
	}

	return metrics
}

func (a *FingerprintAnalyzer) analyzeWebGLEnhanced(data map[string]interface{}) *WebGLMetrics {
	metrics := &WebGLMetrics{
		Details: make([]string, 0),
	}

	if hash, ok := data["webgl_hash"].(string); ok {
		metrics.Hash = hash
	}

	if vendor, ok := data["webgl_vendor"].(string); ok {
		metrics.Vendor = vendor
		metrics.UnmaskedVendor = vendor
	}

	if renderer, ok := data["webgl_renderer"].(string); ok {
		metrics.Renderer = renderer
		metrics.UnmaskedRenderer = renderer

		softwarePatterns := []string{"swiftshader", "llvmpipe", "mesa", "virtual", "software"}
		for _, pattern := range softwarePatterns {
			if strings.Contains(strings.ToLower(renderer), pattern) {
				metrics.IsSoftwareRenderer = true
				metrics.Details = append(metrics.Details, "software_renderer:"+pattern)
				break
			}
		}

		virtualPatterns := []string{"virtual", "vmware", "virtualbox", "parallels", "qemu", "kvm"}
		for _, pattern := range virtualPatterns {
			if strings.Contains(strings.ToLower(renderer), pattern) {
				metrics.IsVirtualGPU = true
				metrics.Details = append(metrics.Details, "virtual_gpu:"+pattern)
				break
			}
		}
	}

	if maxTexSize, ok := data["webgl_max_texture_size"].(float64); ok {
		metrics.MaxTextureSize = int(maxTexSize)
		if maxTexSize <= 1024 {
			metrics.Details = append(metrics.Details, "limited_texture_size")
		}
	}

	if maxRbSize, ok := data["webgl_max_renderbuffer_size"].(float64); ok {
		metrics.MaxRenderbufferSize = int(maxRbSize)
	}

	if maxAttribs, ok := data["webgl_max_vertex_attribs"].(float64); ok {
		metrics.MaxVertexAttribs = int(maxAttribs)
		if maxAttribs <= 8 {
			metrics.Details = append(metrics.Details, "limited_vertex_attribs")
		}
	}

	if extensions, ok := data["webgl_extensions_count"].(float64); ok {
		metrics.SupportedExtensions = int(extensions)
		if extensions < 10 {
			metrics.Details = append(metrics.Details, "limited_extensions")
		}
	}

	if precision, ok := data["webgl_precision_loss"].(bool); ok && precision {
		metrics.PrecisionLoss = true
		metrics.Details = append(metrics.Details, "precision_loss_detected")
	}

	return metrics
}

func (a *FingerprintAnalyzer) analyzeFontsEnhanced(data map[string]interface{}) *FontMetrics {
	metrics := &FontMetrics{
		DetectedFonts:     make([]string, 0),
		CommonFontMissing: make([]string, 0),
	}

	if hash, ok := data["font_hash"].(string); ok {
		metrics.Hash = hash
	}

	if fonts, ok := data["detected_fonts"].([]interface{}); ok {
		for _, font := range fonts {
			if fontStr, ok := font.(string); ok {
				metrics.DetectedFonts = append(metrics.DetectedFonts, fontStr)
			}
		}
		metrics.FontCount = len(metrics.DetectedFonts)
	}

	commonFonts := []string{"Arial", "Helvetica", "Times New Roman", "Verdana", "Georgia", "Tahoma", "Segoe UI", "Roboto", "Open Sans"}
	for _, common := range commonFonts {
		found := false
		for _, detected := range metrics.DetectedFonts {
			if strings.Contains(strings.ToLower(detected), strings.ToLower(common)) {
				found = true
				break
			}
		}
		if !found {
			metrics.CommonFontMissing = append(metrics.CommonFontMissing, common)
		}
	}

	if metrics.FontCount < 3 {
		metrics.IsLimitedFontSet = true
	}

	if metrics.FontCount > 0 {
		fontFamilies := make(map[string]bool)
		for _, font := range metrics.DetectedFonts {
			family := strings.Split(font, " ")[0]
			fontFamilies[family] = true
		}
		metrics.FontFamilyDiversity = float64(len(fontFamilies)) / float64(metrics.FontCount)
	}

	return metrics
}

func (a *FingerprintAnalyzer) analyzeScreenEnhanced(data map[string]interface{}) *ScreenMetrics {
	metrics := &ScreenMetrics{}

	if resolution, ok := data["screen_resolution"].(string); ok {
		metrics.Resolution = resolution
	}

	if colorDepth, ok := data["screen_color_depth"].(float64); ok {
		metrics.ColorDepth = int(colorDepth)
	}

	if pixelRatio, ok := data["screen_pixel_ratio"].(float64); ok {
		metrics.PixelRatio = pixelRatio
		metrics.DevicePixelRatio = pixelRatio
	}

	if availWidth, ok := data["screen_avail_width"].(float64); ok {
		metrics.AvailWidth = int(availWidth)
	}

	if availHeight, ok := data["screen_avail_height"].(float64); ok {
		metrics.AvailHeight = int(availHeight)
	}

	if orientation, ok := data["screen_orientation"].(string); ok {
		metrics.Orientation = orientation
	}

	commonResolutions := []string{
		"1920x1080", "1366x768", "1536x864", "1440x900", "1280x720",
		"1280x800", "1600x900", "2560x1440", "3840x2160",
		"1680x1050", "1280x1024", "1024x768",
	}

	for _, common := range commonResolutions {
		if metrics.Resolution == common {
			metrics.IsCommonResolution = true
			break
		}
	}

	return metrics
}

func (a *FingerprintAnalyzer) generateBrowserSignature(data map[string]interface{}) string {
	components := make([]string, 0)

	if ua, ok := data["user_agent"].(string); ok {
		maxLen := len(ua)
		if maxLen > 50 {
			maxLen = 50
		}
		components = append(components, "ua:"+ua[:maxLen])
	}

	if canvas, ok := data["canvas_hash"].(string); ok {
		maxLen := len(canvas)
		if maxLen > 16 {
			maxLen = 16
		}
		components = append(components, "cnv:"+canvas[:maxLen])
	}

	if webgl, ok := data["webgl_renderer"].(string); ok {
		maxLen := len(webgl)
		if maxLen > 30 {
			maxLen = 30
		}
		components = append(components, "wgl:"+webgl[:maxLen])
	}

	if fonts, ok := data["detected_fonts"].([]interface{}); ok {
		fontCount := len(fonts)
		components = append(components, fmt.Sprintf("fc:%d", fontCount))
	}

	if screen, ok := data["screen_resolution"].(string); ok {
		components = append(components, "scr:"+screen)
	}

	if tz, ok := data["timezone"].(string); ok {
		components = append(components, "tz:"+tz)
	}

	return strings.Join(components, "|")
}

func (a *FingerprintAnalyzer) compareWithKnownBrowsers(data map[string]interface{}) *MultiBrowserCompare {
	compare := &MultiBrowserCompare{
		ComparedBrowsers: make([]string, 0),
		SimilarityScores: make([]float64, 0),
	}

	knownBrowserPatterns := map[string][]string{
		"Chrome":  {"Chrome", "Chromium"},
		"Firefox": {"Firefox", "Gecko"},
		"Safari":  {"Safari", "WebKit"},
		"Edge":    {"Edge", "Edg"},
		"Opera":   {"Opera", "OPR"},
	}

	ua := getString(data, "user_agent")

	for browser, patterns := range knownBrowserPatterns {
		matchCount := 0
		for _, pattern := range patterns {
			if strings.Contains(ua, pattern) {
				matchCount++
			}
		}
		if matchCount > 0 {
			compare.ComparedBrowsers = append(compare.ComparedBrowsers, browser)
		}
	}

	compare.IsUniqueSignature = len(compare.ComparedBrowsers) == 1

	if len(compare.ComparedBrowsers) > 1 {
		compare.CollisionRisk = 0.7
	} else if len(compare.ComparedBrowsers) == 0 {
		compare.CollisionRisk = 0.5
	} else {
		compare.CollisionRisk = 0.1
	}

	return compare
}

func (a *FingerprintAnalyzer) calculateUniquenessScore(metrics *EnhancedFingerprintMetrics) float64 {
	score := 50.0

	if metrics.CanvasMetrics != nil && metrics.CanvasMetrics.Hash != "" {
		if !metrics.CanvasMetrics.IsHeadlessRenderer && !metrics.CanvasMetrics.SoftwareRenderer {
			score += 15
		}
		if metrics.CanvasMetrics.NoiseLevel > 0.01 {
			score += 10
		}
	}

	if metrics.WebGLMetrics != nil {
		if !metrics.WebGLMetrics.IsSoftwareRenderer && !metrics.WebGLMetrics.IsVirtualGPU {
			score += 15
		}
		if metrics.WebGLMetrics.SupportedExtensions > 10 {
			score += 5
		}
	}

	if metrics.FontMetrics != nil {
		if !metrics.FontMetrics.IsLimitedFontSet {
			score += 10
		}
		if metrics.FontMetrics.FontFamilyDiversity > 0.5 {
			score += 5
		}
	}

	if metrics.ScreenMetrics != nil && !metrics.ScreenMetrics.IsCommonResolution {
		score += 5
	}

	if metrics.MultiBrowserCompare != nil && metrics.MultiBrowserCompare.IsUniqueSignature {
		score += 10
	}

	return math.Min(score, 100)
}

type CanvasSimilarityAnalyzer struct {
	database *FingerprintDatabase
}

func NewCanvasSimilarityAnalyzer(db *FingerprintDatabase) *CanvasSimilarityAnalyzer {
	return &CanvasSimilarityAnalyzer{
		database: db,
	}
}

func (c *CanvasSimilarityAnalyzer) CalculateCanvasSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}

	if hash1 == hash2 {
		return 100.0
	}

	similarChars := 0
	minLen := len(hash1)
	if len(hash2) < minLen {
		minLen = len(hash2)
	}

	for i := 0; i < minLen; i++ {
		if hash1[i] == hash2[i] {
			similarChars++
		}
	}

	avgLen := (len(hash1) + len(hash2)) / 2
	return float64(similarChars) / float64(avgLen) * 100
}

func (c *CanvasSimilarityAnalyzer) FindSimilarCanvas(hash string, threshold float64) []*FingerprintAnalysis {
	similar := make([]*FingerprintAnalysis, 0)

	fingerprints := c.database.GetAllFingerprints()
	for _, fp := range fingerprints {
		if fp.CanvasHash == "" {
			continue
		}

		similarity := c.CalculateCanvasSimilarity(hash, fp.CanvasHash)
		if similarity >= threshold {
			similar = append(similar, fp)
		}
	}

	sort.Slice(similar, func(i, j int) bool {
		iSim := c.CalculateCanvasSimilarity(hash, similar[i].CanvasHash)
		jSim := c.CalculateCanvasSimilarity(hash, similar[j].CanvasHash)
		return iSim > jSim
	})

	return similar
}

func (c *CanvasSimilarityAnalyzer) AnalyzeCanvasFingerprint(data map[string]interface{}) *CanvasMetrics {
	metrics := &CanvasMetrics{
		Details: make([]string, 0),
	}

	if hash, ok := data["canvas_hash"].(string); ok {
		metrics.Hash = hash
	}

	if rgbaData, ok := data["canvas_rgba_distribution"].([]interface{}); ok {
		for _, v := range rgbaData {
			if fv, ok := v.(float64); ok {
				metrics.RgbaDistribution = append(metrics.RgbaDistribution, int(fv))
			}
		}
	}

	if noiseLevel, ok := data["canvas_noise_level"].(float64); ok {
		metrics.NoiseLevel = noiseLevel
		if noiseLevel < 0.005 {
			metrics.IsHeadlessRenderer = true
			metrics.Details = append(metrics.Details, "possible_headless")
		}
	}

	if consistency, ok := data["canvas_rendering_consistency"].(float64); ok {
		metrics.RenderingConsistency = consistency
		if consistency > 0.999 {
			metrics.Details = append(metrics.Details, "suspiciously_consistent")
		}
	}

	return metrics
}

func (c *CanvasSimilarityAnalyzer) CalculateHistogramSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}

	hist1 := c.hashToHistogram(hash1)
	hist2 := c.hashToHistogram(hash2)

	return c.cosineSimilarity(hist1, hist2) * 100
}

func (c *CanvasSimilarityAnalyzer) hashToHistogram(hash string) []int {
	histogram := make([]int, 16)
	
	for i := 0; i < len(hash); i++ {
		nibble := 0
		if hash[i] >= '0' && hash[i] <= '9' {
			nibble = int(hash[i] - '0')
		} else if hash[i] >= 'a' && hash[i] <= 'f' {
			nibble = int(hash[i] - 'a' + 10)
		} else if hash[i] >= 'A' && hash[i] <= 'F' {
			nibble = int(hash[i] - 'A' + 10)
		}
		histogram[nibble]++
	}
	
	return histogram
}

func (c *CanvasSimilarityAnalyzer) cosineSimilarity(vec1, vec2 []int) float64 {
	dotProduct := 0
	norm1 := 0
	norm2 := 0
	
	minLen := len(vec1)
	if len(vec2) < minLen {
		minLen = len(vec2)
	}
	
	for i := 0; i < minLen; i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	
	if norm1 == 0 || norm2 == 0 {
		return 0
	}
	
	return float64(dotProduct) / (math.Sqrt(float64(norm1)) * math.Sqrt(float64(norm2)))
}

func (c *CanvasSimilarityAnalyzer) CalculateEnhancedSimilarity(hash1, hash2 string) float64 {
	if hash1 == "" || hash2 == "" {
		return 0
	}
	
	if hash1 == hash2 {
		return 100.0
	}
	
	exactMatch := c.CalculateCanvasSimilarity(hash1, hash2)
	histogramMatch := c.CalculateHistogramSimilarity(hash1, hash2)
	
	return (exactMatch*0.6 + histogramMatch*0.4)
}

func (c *CanvasSimilarityAnalyzer) AnalyzeHashStability(hashSamples []string) *CanvasStabilityResult {
	result := &CanvasStabilityResult{
		SampleCount:    len(hashSamples),
		IsStable:       true,
		StabilityScore: 100.0,
		Issues:         make([]string, 0),
	}
	
	if len(hashSamples) < 2 {
		result.Issues = append(result.Issues, "insufficient_samples")
		result.IsStable = false
		result.StabilityScore = 0
		return result
	}
	
	referenceHash := hashSamples[0]
	totalSimilarity := 0.0
	matchCount := 0
	
	for i := 1; i < len(hashSamples); i++ {
		similarity := c.CalculateEnhancedSimilarity(referenceHash, hashSamples[i])
		totalSimilarity += similarity
		
		if hashSamples[i] == referenceHash {
			matchCount++
		}
	}
	
	result.AvgSimilarity = totalSimilarity / float64(len(hashSamples)-1)
	result.ExactMatchRatio = float64(matchCount) / float64(len(hashSamples)-1)
	
	if result.AvgSimilarity < 95 {
		result.IsStable = false
		result.StabilityScore = result.AvgSimilarity
		result.Issues = append(result.Issues, "low_average_similarity")
	}
	
	if result.ExactMatchRatio < 0.8 {
		result.Issues = append(result.Issues, "inconsistent_hash_generation")
	}
	
	if result.AvgSimilarity > 99.9 && len(hashSamples) > 5 {
		result.Issues = append(result.Issues, "suspiciously_identical_hashes")
	}
	
	return result
}

type CanvasStabilityResult struct {
	SampleCount      int      `json:"sample_count"`
	IsStable         bool     `json:"is_stable"`
	StabilityScore   float64  `json:"stability_score"`
	AvgSimilarity    float64  `json:"avg_similarity"`
	ExactMatchRatio  float64  `json:"exact_match_ratio"`
	Issues           []string `json:"issues"`
}

func (c *CanvasSimilarityAnalyzer) DetectHashTampering(hash string, expectedLength int) *TamperingDetection {
	result := &TamperingDetection{
		IsTampered:      false,
		Confidence:      0.0,
		Indicators:      make([]string, 0),
	}
	
	if hash == "" {
		result.IsTampered = true
		result.Confidence = 0.9
		result.Indicators = append(result.Indicators, "empty_hash")
		return result
	}
	
	if expectedLength > 0 && len(hash) != expectedLength {
		result.IsTampered = true
		result.Confidence = 0.85
		result.Indicators = append(result.Indicators, "invalid_length")
	}
	
	hexPattern := regexp.MustCompile("^[0-9a-fA-F]+$")
	if !hexPattern.MatchString(hash) {
		result.IsTampered = true
		result.Confidence = 0.95
		result.Indicators = append(result.Indicators, "non_hex_characters")
	}
	
	if len(hash) > 0 {
		histogram := c.hashToHistogram(hash)
		entropy := c.calculateEntropy(histogram)
		
		if entropy < 2.0 {
			result.IsTampered = true
			result.Confidence = math.Min(0.8 + (2.0-entropy)*0.1, 0.95)
			result.Indicators = append(result.Indicators, "low_entropy")
		}
		
		if entropy > 3.9 {
			result.Indicators = append(result.Indicators, "unusually_high_entropy")
		}
	}
	
	if len(result.Indicators) > 0 {
		result.IsTampered = true
		result.Confidence = math.Min(0.5+float64(len(result.Indicators))*0.15, 0.95)
	}
	
	return result
}

func (c *CanvasSimilarityAnalyzer) calculateEntropy(histogram []int) float64 {
	total := 0
	for _, count := range histogram {
		total += count
	}
	
	if total == 0 {
		return 0
	}
	
	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			prob := float64(count) / float64(total)
			entropy -= prob * math.Log2(prob)
		}
	}
	
	return entropy / 4.0
}

type TamperingDetection struct {
	IsTampered bool      `json:"is_tampered"`
	Confidence float64   `json:"confidence"`
	Indicators []string  `json:"indicators"`
}

type WebGLAnalyzer struct {
	database            *FingerprintDatabase
	knownVendors        map[string]bool
	knownRenderers      map[string]bool
	blacklistedPatterns []string
	expectedExtensions  map[string][]string
}

func NewWebGLAnalyzer(db *FingerprintDatabase) *WebGLAnalyzer {
	return &WebGLAnalyzer{
		database:            db,
		knownVendors:        initKnownVendors(),
		knownRenderers:      initKnownRenderers(),
		blacklistedPatterns: initBlacklistedPatterns(),
		expectedExtensions:  initExpectedExtensions(),
	}
}

func initKnownVendors() map[string]bool {
	return map[string]bool{
		"NVIDIA Corporation":             true,
		"ATI Technologies Inc.":          true,
		"Advanced Micro Devices, Inc.":   true,
		"Intel Inc.":                     true,
		"Intel(R) Corporation":           true,
		"Google Inc.":                    true,
		"Microsoft Corporation":          true,
		"Apple Inc.":                     true,
		"ARM Ltd.":                       true,
		"Qualcomm":                       true,
		"Imagination Technologies":       true,
		"Mesa project":                   true,
		"VMware, Inc.":                   true,
		"VirtualBox":                     true,
	}
}

func initKnownRenderers() map[string]bool {
	return map[string]bool{
		"GeForce":                        true,
		"Radeon":                         true,
		"Intel(R) HD Graphics":           true,
		"Intel(R) UHD Graphics":          true,
		"Apple M1":                       true,
		"Apple M2":                       true,
		"SwiftShader":                    true,
		"llvmpipe":                       true,
		"ANGLE":                          true,
		"WebKit WebGL":                   true,
		"Chromium":                       true,
		"Mesa":                           true,
		"VirtualBox":                     true,
		"VMware":                         true,
		"NVIDIA":                         true,
	}
}

func initBlacklistedPatterns() []string {
	return []string{
		"fake",
		"mock",
		"test",
		"emulator",
		"virtual",
		"spoof",
		"none",
		"unknown",
		"undefined",
	}
}

func initExpectedExtensions() map[string][]string {
	return map[string][]string{
		"webgl": {
			"GL_EXT_blend_minmax",
			"GL_EXT_color_buffer_float",
			"GL_EXT_frag_depth",
			"GL_EXT_shader_texture_lod",
			"GL_EXT_sRGB",
			"GL_OES_standard_derivatives",
			"GL_OES_texture_float",
			"GL_OES_texture_float_linear",
			"GL_OES_texture_half_float",
			"GL_OES_texture_half_float_linear",
			"GL_OES_vertex_array_object",
			"GL_ANGLE_instanced_arrays",
			"GL_WEBGL_compressed_texture_s3tc",
			"GL_WEBGL_depth_texture",
			"GL_WEBGL_lose_context",
		},
		"webgl2": {
			"GL_EXT_color_buffer_float",
			"GL_EXT_float_blend",
			"GL_EXT_frag_depth",
			"GL_EXT_shader_texture_lod",
			"GL_EXT_sRGB",
			"GL_OES_standard_derivatives",
			"GL_WEBGL_compressed_texture_s3tc",
			"GL_WEBGL_compressed_texture_s3tc_srgb",
			"GL_WEBGL_depth_texture",
			"GL_WEBGL_lose_context",
		},
	}
}

func (w *WebGLAnalyzer) AnalyzeWebGLFingerprint(data map[string]interface{}) *WebGLAnalysisResult {
	result := &WebGLAnalysisResult{
		IsTampered:       false,
		TamperingScore:   0.0,
		Confidence:       0.0,
		VendorAnalysis:   &VendorAnalysis{},
		RendererAnalysis: &RendererAnalysis{},
		ExtensionsAnalysis: &ExtensionsAnalysis{},
		Capabilities:     &WebGLCapabilities{},
		Warnings:         make([]string, 0),
		Errors:           make([]string, 0),
	}

	w.analyzeVendor(data, result)
	w.analyzeRenderer(data, result)
	w.analyzeExtensions(data, result)
	w.analyzeCapabilities(data, result)

	if len(result.Errors) > 0 {
		result.IsTampered = true
		result.TamperingScore = math.Min(50.0+float64(len(result.Errors))*15.0, 95.0)
		result.Confidence = math.Min(0.5+float64(len(result.Errors))*0.15, 0.95)
	} else if len(result.Warnings) > 2 {
		result.IsTampered = true
		result.TamperingScore = math.Min(20.0+float64(len(result.Warnings))*10.0, 50.0)
		result.Confidence = math.Min(0.3+float64(len(result.Warnings))*0.1, 0.7)
	}

	return result
}

func (w *WebGLAnalyzer) analyzeVendor(data map[string]interface{}, result *WebGLAnalysisResult) {
	vendor := getString(data, "webgl_vendor")
	unmaskedVendor := getString(data, "webgl_unmasked_vendor")

	result.VendorAnalysis.Vendor = vendor
	result.VendorAnalysis.UnmaskedVendor = unmaskedVendor

	if vendor == "" {
		result.Errors = append(result.Errors, "missing_vendor")
		return
	}

	if unmaskedVendor == "" {
		result.Warnings = append(result.Warnings, "missing_unmasked_vendor")
	}

	if !w.knownVendors[vendor] && !w.knownVendors[unmaskedVendor] {
		result.Warnings = append(result.Warnings, "unknown_vendor:"+vendor)
	}

	if vendor != unmaskedVendor && vendor != "" && unmaskedVendor != "" {
		similarity := w.stringSimilarity(vendor, unmaskedVendor)
		if similarity < 0.5 {
			result.Warnings = append(result.Warnings, "vendor_mismatch")
		}
	}

	for _, pattern := range w.blacklistedPatterns {
		if strings.Contains(strings.ToLower(vendor), pattern) {
			result.Errors = append(result.Errors, "blacklisted_vendor_pattern:"+pattern)
			return
		}
	}
}

func (w *WebGLAnalyzer) analyzeRenderer(data map[string]interface{}, result *WebGLAnalysisResult) {
	renderer := getString(data, "webgl_renderer")
	unmaskedRenderer := getString(data, "webgl_unmasked_renderer")

	result.RendererAnalysis.Renderer = renderer
	result.RendererAnalysis.UnmaskedRenderer = unmaskedRenderer

	if renderer == "" {
		result.Errors = append(result.Errors, "missing_renderer")
		return
	}

	if unmaskedRenderer == "" {
		result.Warnings = append(result.Warnings, "missing_unmasked_renderer")
	}

	found := false
	for knownRenderer := range w.knownRenderers {
		if strings.Contains(renderer, knownRenderer) || strings.Contains(unmaskedRenderer, knownRenderer) {
			found = true
			break
		}
	}
	if !found {
		result.Warnings = append(result.Warnings, "unknown_renderer:"+renderer)
	}

	if renderer != unmaskedRenderer && renderer != "" && unmaskedRenderer != "" {
		similarity := w.stringSimilarity(renderer, unmaskedRenderer)
		if similarity < 0.3 {
			result.Errors = append(result.Errors, "renderer_mismatch")
		}
	}

	softwarePatterns := []string{"swiftshader", "llvmpipe", "mesa"}
	for _, pattern := range softwarePatterns {
		if strings.Contains(strings.ToLower(renderer), pattern) {
			result.RendererAnalysis.IsSoftwareRenderer = true
			result.Warnings = append(result.Warnings, "software_renderer:"+pattern)
			break
		}
	}

	virtualPatterns := []string{"virtualbox", "vmware", "qemu", "kvm", "parallels", "hyperv"}
	for _, pattern := range virtualPatterns {
		if strings.Contains(strings.ToLower(renderer), pattern) {
			result.RendererAnalysis.IsVirtualGPU = true
			result.Warnings = append(result.Warnings, "virtual_gpu:"+pattern)
			break
		}
	}
}

func (w *WebGLAnalyzer) analyzeExtensions(data map[string]interface{}, result *WebGLAnalysisResult) {
	var extensions []string
	if extData, ok := data["webgl_extensions"].([]interface{}); ok {
		for _, ext := range extData {
			if extStr, ok := ext.(string); ok {
				extensions = append(extensions, extStr)
			}
		}
	}

	result.ExtensionsAnalysis.ExtensionCount = len(extensions)
	result.ExtensionsAnalysis.Extensions = extensions

	if len(extensions) == 0 {
		result.Errors = append(result.Errors, "no_extensions")
		return
	}

	if len(extensions) < 5 {
		result.Warnings = append(result.Warnings, "unusually_few_extensions")
	}

	webglVersion := getString(data, "webgl_version")
	expectedExtensions := w.expectedExtensions[webglVersion]
	if len(expectedExtensions) > 0 {
		missingExtensions := make([]string, 0)
		for _, expected := range expectedExtensions {
			found := false
			for _, ext := range extensions {
				if strings.Contains(ext, expected) || strings.Contains(expected, ext) {
					found = true
					break
				}
			}
			if !found {
				missingExtensions = append(missingExtensions, expected)
			}
		}
		if len(missingExtensions) > len(expectedExtensions)/2 {
			result.Warnings = append(result.Warnings, "many_expected_extensions_missing")
		}
		result.ExtensionsAnalysis.MissingExpected = missingExtensions
	}

	suspiciousExtensions := []string{"WEBGL_debug_renderer_info", "WEBGL_lose_context"}
	foundSuspicious := false
	for _, ext := range extensions {
		for _, suspicious := range suspiciousExtensions {
			if strings.Contains(ext, suspicious) {
				foundSuspicious = true
				break
			}
		}
	}
	if !foundSuspicious && len(extensions) > 0 {
		result.Warnings = append(result.Warnings, "missing_debug_extensions")
	}
}

func (w *WebGLAnalyzer) analyzeCapabilities(data map[string]interface{}, result *WebGLAnalysisResult) {
	if maxTextureSize, ok := data["webgl_max_texture_size"].(float64); ok {
		result.Capabilities.MaxTextureSize = int(maxTextureSize)
		if maxTextureSize < 2048 {
			result.Warnings = append(result.Warnings, "small_max_texture_size")
		}
	}

	if maxRenderbufferSize, ok := data["webgl_max_renderbuffer_size"].(float64); ok {
		result.Capabilities.MaxRenderbufferSize = int(maxRenderbufferSize)
	}

	if maxVertexAttribs, ok := data["webgl_max_vertex_attribs"].(float64); ok {
		result.Capabilities.MaxVertexAttribs = int(maxVertexAttribs)
		if maxVertexAttribs < 16 {
			result.Warnings = append(result.Warnings, "limited_vertex_attribs")
		}
	}

	if precisionLoss, ok := data["webgl_precision_loss"].(bool); ok && precisionLoss {
		result.Capabilities.PrecisionLoss = true
		result.Warnings = append(result.Warnings, "precision_loss_detected")
	}
}

func (w *WebGLAnalyzer) stringSimilarity(s1, s2 string) float64 {
	if s1 == "" || s2 == "" {
		return 0
	}

	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	matchCount := 0
	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			matchCount++
		}
	}

	avgLen := (len(s1) + len(s2)) / 2
	return float64(matchCount) / float64(avgLen)
}

func (w *WebGLAnalyzer) CompareWebGLFingerprints(fp1, fp2 *WebGLMetrics) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	totalScore := 0.0
	weightSum := 0.0

	if fp1.Vendor != "" && fp2.Vendor != "" {
		if fp1.Vendor == fp2.Vendor {
			totalScore += 25
		}
		weightSum += 25
	}

	if fp1.Renderer != "" && fp2.Renderer != "" {
		if fp1.Renderer == fp2.Renderer {
			totalScore += 30
		}
		weightSum += 30
	}

	if fp1.MaxTextureSize > 0 && fp2.MaxTextureSize > 0 {
		if fp1.MaxTextureSize == fp2.MaxTextureSize {
			totalScore += 15
		}
		weightSum += 15
	}

	if fp1.MaxRenderbufferSize > 0 && fp2.MaxRenderbufferSize > 0 {
		if fp1.MaxRenderbufferSize == fp2.MaxRenderbufferSize {
			totalScore += 10
		}
		weightSum += 10
	}

	if fp1.MaxVertexAttribs > 0 && fp2.MaxVertexAttribs > 0 {
		if fp1.MaxVertexAttribs == fp2.MaxVertexAttribs {
			totalScore += 10
		}
		weightSum += 10
	}

	if fp1.SupportedExtensions > 0 && fp2.SupportedExtensions > 0 {
		diff := float64(intAbs(fp1.SupportedExtensions - fp2.SupportedExtensions))
		similarity := math.Max(0, 1.0-diff/50.0)
		totalScore += similarity * 10
		weightSum += 10
	}

	if weightSum == 0 {
		return 0
	}

	return (totalScore / weightSum) * 100
}

func intAbs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

type WebGLAnalysisResult struct {
	IsTampered         bool                `json:"is_tampered"`
	TamperingScore     float64             `json:"tampering_score"`
	Confidence         float64             `json:"confidence"`
	VendorAnalysis     *VendorAnalysis     `json:"vendor_analysis"`
	RendererAnalysis   *RendererAnalysis   `json:"renderer_analysis"`
	ExtensionsAnalysis *ExtensionsAnalysis `json:"extensions_analysis"`
	Capabilities       *WebGLCapabilities  `json:"capabilities"`
	Warnings           []string            `json:"warnings"`
	Errors             []string            `json:"errors"`
}

type VendorAnalysis struct {
	Vendor         string `json:"vendor"`
	UnmaskedVendor string `json:"unmasked_vendor"`
	IsKnownVendor  bool   `json:"is_known_vendor"`
}

type RendererAnalysis struct {
	Renderer          string `json:"renderer"`
	UnmaskedRenderer  string `json:"unmasked_renderer"`
	IsKnownRenderer   bool   `json:"is_known_renderer"`
	IsSoftwareRenderer bool  `json:"is_software_renderer"`
	IsVirtualGPU      bool   `json:"is_virtual_gpu"`
}

type ExtensionsAnalysis struct {
	ExtensionCount     int      `json:"extension_count"`
	Extensions         []string `json:"extensions"`
	MissingExpected    []string `json:"missing_expected"`
	HasDebugExtensions bool     `json:"has_debug_extensions"`
}

type WebGLCapabilities struct {
	MaxTextureSize      int  `json:"max_texture_size"`
	MaxRenderbufferSize int  `json:"max_renderbuffer_size"`
	MaxVertexAttribs    int  `json:"max_vertex_attribs"`
	PrecisionLoss       bool `json:"precision_loss"`
}

type FontAnalyzer struct {
	database             *FingerprintDatabase
	commonFonts          map[string]bool
	rareFonts            map[string]bool
	platformFonts        map[string][]string
	expectedFontCounts   map[string]int
}

func NewFontAnalyzer(db *FingerprintDatabase) *FontAnalyzer {
	return &FontAnalyzer{
		database:           db,
		commonFonts:        initCommonFonts(),
		rareFonts:          initRareFonts(),
		platformFonts:      initPlatformFonts(),
		expectedFontCounts: initExpectedFontCounts(),
	}
}

func initCommonFonts() map[string]bool {
	return map[string]bool{
		"Arial":              true,
		"Helvetica":          true,
		"Times New Roman":    true,
		"Times":              true,
		"Courier New":        true,
		"Courier":            true,
		"Verdana":            true,
		"Georgia":            true,
		"Comic Sans MS":      true,
		"Trebuchet MS":       true,
		"Arial Black":        true,
		"Impact":             true,
		"Lucida Console":     true,
		"Lucida Sans Unicode": true,
		"Palatino Linotype":  true,
		"Garamond":           true,
		"Book Antiqua":       true,
		"Microsoft YaHei":    true,
		"SimHei":             true,
		"SimSun":             true,
		"KaiTi":              true,
		"NSimSun":            true,
		"STSong":             true,
		"STHeiti":            true,
		"Roboto":             true,
		"Open Sans":          true,
		"Lato":               true,
		"Segoe UI":           true,
		"Ubuntu":             true,
		"Cantarell":          true,
		"DejaVu Sans":        true,
		"DejaVu Serif":       true,
	}
}

func initRareFonts() map[string]bool {
	return map[string]bool{
		"Comic Neue":         true,
		"Fira Code":          true,
		"JetBrains Mono":     true,
		"SF Pro Display":     true,
		"SF Pro Text":        true,
		"SF Mono":            true,
		"PingFang SC":        true,
		"PingFang TC":        true,
		"PingFang HK":        true,
		"Hiragino Sans":      true,
		"Yu Gothic":          true,
		"Yu Mincho":          true,
		"Meiryo":             true,
		"Malgun Gothic":      true,
		"Apple SD Gothic Neo": true,
	}
}

func initPlatformFonts() map[string][]string {
	return map[string][]string{
		"windows": {
			"Arial", "Arial Black", "Comic Sans MS", "Courier New",
			"Georgia", "Impact", "Times New Roman", "Trebuchet MS",
			"Verdana", "Microsoft YaHei", "SimHei", "SimSun", "KaiTi",
		},
		"macos": {
			"Arial", "Helvetica", "Times New Roman", "Georgia",
			"Verdana", "SF Pro Display", "SF Pro Text", "PingFang SC",
			"Hiragino Sans", "Yu Gothic", "Yu Mincho",
		},
		"linux": {
			"DejaVu Sans", "DejaVu Serif", "Ubuntu", "Cantarell",
			"Liberation Sans", "Liberation Serif",
		},
		"android": {
			"Roboto", "Open Sans", "Noto Sans",
		},
		"ios": {
			"SF Pro Display", "SF Pro Text", "PingFang SC",
		},
	}
}

func initExpectedFontCounts() map[string]int {
	return map[string]int{
		"windows": 25,
		"macos":   20,
		"linux":   15,
		"android": 12,
		"ios":     15,
	}
}

func (f *FontAnalyzer) AnalyzeFontFingerprint(data map[string]interface{}) *FontAnalysisResult {
	result := &FontAnalysisResult{
		IsTampered:       false,
		Confidence:       0.0,
		FontAnalysis:     &DetailedFontAnalysis{},
		RenderingAnalysis: &FontRenderingAnalysis{},
		PlatformMatch:    &PlatformFontMatch{},
		Warnings:         make([]string, 0),
		Errors:           make([]string, 0),
	}

	f.extractFontData(data, result)
	f.analyzeFontPatterns(result)
	f.analyzeRenderingConsistency(data, result)
	f.analyzePlatformMatch(data, result)

	f.calculateConfidence(result)

	return result
}

func (f *FontAnalyzer) extractFontData(data map[string]interface{}, result *FontAnalysisResult) {
	if fonts, ok := data["detected_fonts"].([]interface{}); ok {
		for _, font := range fonts {
			if fontStr, ok := font.(string); ok {
				result.FontAnalysis.DetectedFonts = append(result.FontAnalysis.DetectedFonts, fontStr)
			}
		}
	}
	result.FontAnalysis.FontCount = len(result.FontAnalysis.DetectedFonts)

	if hash, ok := data["font_hash"].(string); ok {
		result.FontAnalysis.FontHash = hash
	}
}

func (f *FontAnalyzer) analyzeFontPatterns(result *FontAnalysisResult) {
	commonCount := 0
	rareCount := 0
	missingCommon := make([]string, 0)

	for _, font := range result.FontAnalysis.DetectedFonts {
		lowerFont := strings.ToLower(font)
		
		for common := range f.commonFonts {
			if strings.Contains(lowerFont, strings.ToLower(common)) {
				commonCount++
				break
			}
		}
		
		for rare := range f.rareFonts {
			if strings.Contains(lowerFont, strings.ToLower(rare)) {
				rareCount++
				break
			}
		}
	}

	for common := range f.commonFonts {
		found := false
		for _, font := range result.FontAnalysis.DetectedFonts {
			if strings.Contains(strings.ToLower(font), strings.ToLower(common)) {
				found = true
				break
			}
		}
		if !found {
			missingCommon = append(missingCommon, common)
		}
	}

	result.FontAnalysis.CommonFontCount = commonCount
	result.FontAnalysis.RareFontCount = rareCount
	result.FontAnalysis.MissingCommonFonts = missingCommon

	if len(missingCommon) > 5 {
		result.Warnings = append(result.Warnings, "many_common_fonts_missing")
	}

	if commonCount == 0 && result.FontAnalysis.FontCount > 0 {
		result.Errors = append(result.Errors, "no_common_fonts_detected")
	}

	if result.FontAnalysis.FontCount < 3 {
		result.Warnings = append(result.Warnings, "very_few_fonts_detected")
		result.FontAnalysis.IsLimitedFontSet = true
	}

	if result.FontAnalysis.FontCount > 0 {
		fontFamilies := make(map[string]bool)
		for _, font := range result.FontAnalysis.DetectedFonts {
			family := strings.Split(font, " ")[0]
			fontFamilies[family] = true
		}
		result.FontAnalysis.FontFamilyDiversity = float64(len(fontFamilies)) / float64(result.FontAnalysis.FontCount)
	}
}

func (f *FontAnalyzer) analyzeRenderingConsistency(data map[string]interface{}, result *FontAnalysisResult) {
	if renderingData, ok := data["font_rendering_data"].([]interface{}); ok {
		result.RenderingAnalysis.RenderingSamples = len(renderingData)
		
		if len(renderingData) > 1 {
			var variance float64
			sum := 0.0
			for _, sample := range renderingData {
				if val, ok := sample.(float64); ok {
					sum += val
				}
			}
			avg := sum / float64(len(renderingData))
			
			for _, sample := range renderingData {
				if val, ok := sample.(float64); ok {
					variance += math.Pow(val-avg, 2)
				}
			}
			variance /= float64(len(renderingData))
			
			result.RenderingAnalysis.RenderingVariance = variance
			
			if variance < 0.001 {
				result.Warnings = append(result.Warnings, "rendering_too_consistent")
				result.RenderingAnalysis.IsSuspiciouslyConsistent = true
			}
		}
	}

	if antialiasing, ok := data["font_antialiasing"].(bool); ok {
		result.RenderingAnalysis.AntialiasingEnabled = antialiasing
		if !antialiasing {
			result.Warnings = append(result.Warnings, "no_antialiasing")
		}
	}

	if subpixel, ok := data["font_subpixel_rendering"].(bool); ok {
		result.RenderingAnalysis.SubpixelRendering = subpixel
	}
}

func (f *FontAnalyzer) analyzePlatformMatch(data map[string]interface{}, result *FontAnalysisResult) {
	platform := getString(data, "platform")
	if platform == "" {
		platform = getString(data, "os")
	}
	result.PlatformMatch.Platform = platform

	expectedFonts := f.platformFonts[platform]
	if len(expectedFonts) > 0 {
		matchCount := 0
		for _, expected := range expectedFonts {
			for _, detected := range result.FontAnalysis.DetectedFonts {
				if strings.Contains(strings.ToLower(detected), strings.ToLower(expected)) {
					matchCount++
					break
				}
			}
		}
		result.PlatformMatch.ExpectedFontCount = len(expectedFonts)
		result.PlatformMatch.MatchedFontCount = matchCount
		result.PlatformMatch.MatchRatio = float64(matchCount) / float64(len(expectedFonts))

		if result.PlatformMatch.MatchRatio < 0.3 {
			result.Warnings = append(result.Warnings, "low_platform_font_match")
		}
	}

	expectedCount := f.expectedFontCounts[platform]
	if expectedCount > 0 {
		result.PlatformMatch.ExpectedTotalCount = expectedCount
		if result.FontAnalysis.FontCount < expectedCount/2 {
			result.Warnings = append(result.Warnings, "font_count_below_expected")
		}
	}
}

func (f *FontAnalyzer) calculateConfidence(result *FontAnalysisResult) {
	baseScore := 50.0

	if len(result.Errors) > 0 {
		baseScore -= float64(len(result.Errors)) * 20
	}

	if len(result.Warnings) > 0 {
		baseScore -= float64(len(result.Warnings)) * 5
	}

	if result.FontAnalysis.FontCount >= 10 {
		baseScore += 15
	} else if result.FontAnalysis.FontCount >= 5 {
		baseScore += 5
	}

	if result.FontAnalysis.CommonFontCount >= 5 {
		baseScore += 10
	}

	if result.PlatformMatch.MatchRatio > 0.5 {
		baseScore += 10
	}

	if result.FontAnalysis.FontFamilyDiversity > 0.5 {
		baseScore += 10
	}

	result.Confidence = math.Max(0, math.Min(100, baseScore))

	if result.Confidence < 30 {
		result.IsTampered = true
	}
}

func (f *FontAnalyzer) CompareFontFingerprints(fp1, fp2 *FontMetrics) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	if fp1.Hash == "" || fp2.Hash == "" {
		return 0
	}

	if fp1.Hash == fp2.Hash {
		return 100.0
	}

	totalScore := 0.0
	weightSum := 0.0

	if fp1.FontCount > 0 && fp2.FontCount > 0 {
		countDiff := math.Abs(float64(fp1.FontCount - fp2.FontCount))
		maxFontCount := float64(fp1.FontCount)
		if float64(fp2.FontCount) > maxFontCount {
			maxFontCount = float64(fp2.FontCount)
		}
		countSimilarity := math.Max(0, 1.0-countDiff/maxFontCount)
		totalScore += countSimilarity * 30
		weightSum += 30
	}

	if len(fp1.DetectedFonts) > 0 && len(fp2.DetectedFonts) > 0 {
		commonFonts := 0
		fontSet1 := make(map[string]bool)
		for _, font := range fp1.DetectedFonts {
			fontSet1[strings.ToLower(font)] = true
		}
		for _, font := range fp2.DetectedFonts {
			if fontSet1[strings.ToLower(font)] {
				commonFonts++
			}
		}
		maxLen := len(fp1.DetectedFonts)
		if len(fp2.DetectedFonts) > maxLen {
			maxLen = len(fp2.DetectedFonts)
		}
		similarity := float64(commonFonts) / float64(maxLen)
		totalScore += similarity * 50
		weightSum += 50
	}

	if fp1.FontFamilyDiversity > 0 && fp2.FontFamilyDiversity > 0 {
		diff := math.Abs(fp1.FontFamilyDiversity - fp2.FontFamilyDiversity)
		similarity := math.Max(0, 1.0-diff)
		totalScore += similarity * 20
		weightSum += 20
	}

	if weightSum == 0 {
		return 0
	}

	return (totalScore / weightSum) * 100
}

type FontAnalysisResult struct {
	IsTampered         bool                      `json:"is_tampered"`
	Confidence         float64                   `json:"confidence"`
	FontAnalysis       *DetailedFontAnalysis     `json:"font_analysis"`
	RenderingAnalysis  *FontRenderingAnalysis    `json:"rendering_analysis"`
	PlatformMatch      *PlatformFontMatch        `json:"platform_match"`
	Warnings           []string                  `json:"warnings"`
	Errors             []string                  `json:"errors"`
}

type DetailedFontAnalysis struct {
	FontHash            string   `json:"font_hash"`
	DetectedFonts       []string `json:"detected_fonts"`
	FontCount           int      `json:"font_count"`
	CommonFontCount     int      `json:"common_font_count"`
	RareFontCount       int      `json:"rare_font_count"`
	MissingCommonFonts  []string `json:"missing_common_fonts"`
	FontFamilyDiversity float64  `json:"font_family_diversity"`
	IsLimitedFontSet    bool     `json:"is_limited_font_set"`
}

type FontRenderingAnalysis struct {
	RenderingSamples         int     `json:"rendering_samples"`
	RenderingVariance        float64 `json:"rendering_variance"`
	IsSuspiciouslyConsistent bool    `json:"is_suspiciously_consistent"`
	AntialiasingEnabled      bool    `json:"antialiasing_enabled"`
	SubpixelRendering        bool    `json:"subpixel_rendering"`
}

type PlatformFontMatch struct {
	Platform           string  `json:"platform"`
	ExpectedFontCount  int     `json:"expected_font_count"`
	MatchedFontCount   int     `json:"matched_font_count"`
	MatchRatio         float64 `json:"match_ratio"`
	ExpectedTotalCount int     `json:"expected_total_count"`
}

type FingerprintStabilityAnalyzer struct {
	database       *FingerprintDatabase
	historyStorage map[string][]*FingerprintAnalysis
	maxHistorySize int
}

func NewFingerprintStabilityAnalyzer(db *FingerprintDatabase) *FingerprintStabilityAnalyzer {
	return &FingerprintStabilityAnalyzer{
		database:       db,
		historyStorage: make(map[string][]*FingerprintAnalysis),
		maxHistorySize: 50,
	}
}

func (s *FingerprintStabilityAnalyzer) TrackFingerprint(fp *FingerprintAnalysis) {
	s.historyStorage[fp.FingerprintID] = append(s.historyStorage[fp.FingerprintID], fp)
	
	if len(s.historyStorage[fp.FingerprintID]) > s.maxHistorySize {
		s.historyStorage[fp.FingerprintID] = s.historyStorage[fp.FingerprintID][1:]
	}
}

func (s *FingerprintStabilityAnalyzer) AnalyzeStability(fingerprintID string) *StabilityAnalysisResult {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 2 {
		return &StabilityAnalysisResult{
			IsStable:            false,
			StabilityScore:      0.0,
			InsufficientSamples: true,
			AnalysisHistory:     make([]*SingleAnalysisResult, 0),
		}
	}

	result := &StabilityAnalysisResult{
		IsStable:      true,
		StabilityScore: 100.0,
		AnalysisHistory: make([]*SingleAnalysisResult, 0),
	}

	reference := history[0]
	totalSimilarity := 0.0
	matchCount := 0

	for i := 1; i < len(history); i++ {
		similarity := s.calculateOverallSimilarity(reference, history[i])
		totalSimilarity += similarity
		
		if similarity == 100.0 {
			matchCount++
		}

		singleResult := &SingleAnalysisResult{
			Timestamp:      history[i].LastSeen,
			Similarity:     similarity,
			IsConsistent:   similarity >= 95,
			DiffFields:     s.findDiffFields(reference, history[i]),
			ScoreBreakdown: s.calculateScoreBreakdown(reference, history[i]),
		}
		result.AnalysisHistory = append(result.AnalysisHistory, singleResult)
	}

	result.AverageSimilarity = totalSimilarity / float64(len(history)-1)
	result.ExactMatchRatio = float64(matchCount) / float64(len(history)-1)
	result.SampleCount = len(history)
	result.TimeSpanMinutes = history[len(history)-1].LastSeen.Sub(history[0].FirstSeen).Minutes()

	if result.AverageSimilarity < 95 {
		result.IsStable = false
		result.StabilityScore = result.AverageSimilarity
	}

	if result.ExactMatchRatio < 0.7 {
		result.IsStable = false
		result.Warnings = append(result.Warnings, "frequent_fingerprint_changes")
	}

	if result.AverageSimilarity > 99.9 && len(history) > 10 {
		result.Warnings = append(result.Warnings, "suspiciously_consistent")
	}

	if len(result.Warnings) > 0 {
		result.StabilityScore = math.Max(result.StabilityScore-10, 0)
	}

	return result
}

func (s *FingerprintStabilityAnalyzer) calculateOverallSimilarity(fp1, fp2 *FingerprintAnalysis) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	fields := []struct {
		name   string
		val1   string
		val2   string
		weight float64
	}{
		{"canvas", fp1.CanvasHash, fp2.CanvasHash, 25},
		{"webgl", fp1.WebGLHash, fp2.WebGLHash, 25},
		{"audio", fp1.AudioHash, fp2.AudioHash, 15},
		{"fonts", fp1.FontHash, fp2.FontHash, 15},
		{"user_agent", fp1.UserAgent, fp2.UserAgent, 20},
		{"screen", fp1.ScreenResolution, fp2.ScreenResolution, 10},
		{"timezone", fp1.Timezone, fp2.Timezone, 5},
		{"language", fp1.Language, fp2.Language, 5},
	}

	totalWeight := 0.0
	matchWeight := 0.0

	for _, field := range fields {
		totalWeight += field.weight
		if field.val1 != "" && field.val2 != "" && field.val1 == field.val2 {
			matchWeight += field.weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return (matchWeight / totalWeight) * 100
}

func (s *FingerprintStabilityAnalyzer) findDiffFields(fp1, fp2 *FingerprintAnalysis) []string {
	diffs := make([]string, 0)

	if fp1.CanvasHash != fp2.CanvasHash {
		diffs = append(diffs, "canvas")
	}
	if fp1.WebGLHash != fp2.WebGLHash {
		diffs = append(diffs, "webgl")
	}
	if fp1.AudioHash != fp2.AudioHash {
		diffs = append(diffs, "audio")
	}
	if fp1.FontHash != fp2.FontHash {
		diffs = append(diffs, "fonts")
	}
	if fp1.UserAgent != fp2.UserAgent {
		diffs = append(diffs, "user_agent")
	}
	if fp1.ScreenResolution != fp2.ScreenResolution {
		diffs = append(diffs, "screen")
	}
	if fp1.Timezone != fp2.Timezone {
		diffs = append(diffs, "timezone")
	}

	return diffs
}

func (s *FingerprintStabilityAnalyzer) calculateScoreBreakdown(fp1, fp2 *FingerprintAnalysis) map[string]float64 {
	breakdown := make(map[string]float64)

	if fp1.CanvasHash != "" && fp2.CanvasHash != "" {
		breakdown["canvas"] = map[bool]float64{true: 100, false: 0}[fp1.CanvasHash == fp2.CanvasHash]
	}
	if fp1.WebGLHash != "" && fp2.WebGLHash != "" {
		breakdown["webgl"] = map[bool]float64{true: 100, false: 0}[fp1.WebGLHash == fp2.WebGLHash]
	}
	if fp1.AudioHash != "" && fp2.AudioHash != "" {
		breakdown["audio"] = map[bool]float64{true: 100, false: 0}[fp1.AudioHash == fp2.AudioHash]
	}
	if fp1.FontHash != "" && fp2.FontHash != "" {
		breakdown["fonts"] = map[bool]float64{true: 100, false: 0}[fp1.FontHash == fp2.FontHash]
	}
	if fp1.UserAgent != "" && fp2.UserAgent != "" {
		breakdown["user_agent"] = map[bool]float64{true: 100, false: 0}[fp1.UserAgent == fp2.UserAgent]
	}

	return breakdown
}

func (s *FingerprintStabilityAnalyzer) AnalyzeTemporalStability(fingerprintID string) *TemporalStabilityResult {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 3 {
		return &TemporalStabilityResult{
			IsStable:            false,
			InsufficientSamples: true,
		}
	}

	result := &TemporalStabilityResult{
		IsStable:          true,
		TimeSegments:      make([]*TimeSegmentAnalysis, 0),
		TotalDurationMinutes: history[len(history)-1].LastSeen.Sub(history[0].FirstSeen).Minutes(),
	}

	segmentSize := 5
	for i := 0; i < len(history); i += segmentSize {
		end := i + segmentSize
		if end > len(history) {
			end = len(history)
		}

		segment := history[i:end]
		if len(segment) < 2 {
			continue
		}

		segmentAnalysis := &TimeSegmentAnalysis{
			Start:    segment[0].FirstSeen,
			End:      segment[len(segment)-1].LastSeen,
			Samples:  len(segment),
		}

		totalSimilarity := 0.0
		reference := segment[0]
		for j := 1; j < len(segment); j++ {
			totalSimilarity += s.calculateOverallSimilarity(reference, segment[j])
		}
		segmentAnalysis.AvgSimilarity = totalSimilarity / float64(len(segment)-1)
		segmentAnalysis.IsStable = segmentAnalysis.AvgSimilarity >= 95

		result.TimeSegments = append(result.TimeSegments, segmentAnalysis)

		if !segmentAnalysis.IsStable {
			result.IsStable = false
		}
	}

	if len(result.TimeSegments) > 1 {
		similarityChanges := make([]float64, 0)
		for i := 1; i < len(result.TimeSegments); i++ {
			change := math.Abs(result.TimeSegments[i].AvgSimilarity - result.TimeSegments[i-1].AvgSimilarity)
			similarityChanges = append(similarityChanges, change)
		}

		if len(similarityChanges) > 0 {
			avgChange := 0.0
			for _, change := range similarityChanges {
				avgChange += change
			}
			avgChange /= float64(len(similarityChanges))
			result.AvgSegmentChange = avgChange

			if avgChange > 10 {
				result.IsStable = false
				result.Warnings = append(result.Warnings, "significant_temporal_changes")
			}
		}
	}

	result.CalculateOverallScore()

	return result
}

func (s *FingerprintStabilityAnalyzer) DetectStabilityAnomalies(fingerprintID string) []*StabilityAnomaly {
	history, exists := s.historyStorage[fingerprintID]
	if !exists || len(history) < 5 {
		return nil
	}

	anomalies := make([]*StabilityAnomaly, 0)

	for i := 1; i < len(history); i++ {
		similarity := s.calculateOverallSimilarity(history[i-1], history[i])
		
		if similarity < 80 {
			anomaly := &StabilityAnomaly{
				Timestamp:      history[i].LastSeen,
				AnomalyType:    "fingerprint_drift",
				Similarity:     similarity,
				Threshold:      80,
				DiffFields:     s.findDiffFields(history[i-1], history[i]),
				PreviousFP:     history[i-1].FingerprintID,
				CurrentFP:      history[i].FingerprintID,
				Severity:       s.calculateSeverity(similarity),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (s *FingerprintStabilityAnalyzer) calculateSeverity(similarity float64) string {
	if similarity < 50 {
		return "critical"
	} else if similarity < 70 {
		return "high"
	} else if similarity < 80 {
		return "medium"
	}
	return "low"
}

type StabilityAnalysisResult struct {
	IsStable            bool                    `json:"is_stable"`
	StabilityScore      float64                 `json:"stability_score"`
	AverageSimilarity   float64                 `json:"average_similarity"`
	ExactMatchRatio     float64                 `json:"exact_match_ratio"`
	SampleCount         int                     `json:"sample_count"`
	TimeSpanMinutes     float64                 `json:"time_span_minutes"`
	InsufficientSamples bool                   `json:"insufficient_samples"`
	AnalysisHistory     []*SingleAnalysisResult `json:"analysis_history"`
	Warnings            []string                `json:"warnings"`
}

type SingleAnalysisResult struct {
	Timestamp      time.Time              `json:"timestamp"`
	Similarity     float64                `json:"similarity"`
	IsConsistent   bool                   `json:"is_consistent"`
	DiffFields     []string               `json:"diff_fields"`
	ScoreBreakdown map[string]float64     `json:"score_breakdown"`
}

type TemporalStabilityResult struct {
	IsStable               bool                     `json:"is_stable"`
	TotalDurationMinutes   float64                  `json:"total_duration_minutes"`
	TimeSegments           []*TimeSegmentAnalysis   `json:"time_segments"`
	AvgSegmentChange       float64                  `json:"avg_segment_change"`
	InsufficientSamples    bool                    `json:"insufficient_samples"`
	Warnings               []string                 `json:"warnings"`
	OverallScore           float64                  `json:"overall_score"`
}

func (t *TemporalStabilityResult) CalculateOverallScore() {
	if len(t.TimeSegments) == 0 {
		t.OverallScore = 0
		return
	}

	totalScore := 0.0
	for _, segment := range t.TimeSegments {
		totalScore += segment.AvgSimilarity
	}
	t.OverallScore = totalScore / float64(len(t.TimeSegments))

	if len(t.Warnings) > 0 {
		t.OverallScore = math.Max(t.OverallScore-10, 0)
	}
}

type TimeSegmentAnalysis struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Samples      int       `json:"samples"`
	AvgSimilarity float64  `json:"avg_similarity"`
	IsStable     bool      `json:"is_stable"`
}

type StabilityAnomaly struct {
	Timestamp      time.Time `json:"timestamp"`
	AnomalyType    string    `json:"anomaly_type"`
	Similarity     float64   `json:"similarity"`
	Threshold      float64   `json:"threshold"`
	DiffFields     []string  `json:"diff_fields"`
	PreviousFP     string    `json:"previous_fingerprint"`
	CurrentFP      string    `json:"current_fingerprint"`
	Severity       string    `json:"severity"`
}
