package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type DeviceFingerprintTracker struct {
	fingerprints        map[string]*DeviceFingerprint
	deviceGroups        map[string][]string
	fingerprintHistory  map[string][]FingerprintHistoryEntry
	similarityCache     map[string]map[string]float64
	mu                  sync.RWMutex
	maxHistorySize      int
	maxSimilarityCache  int
}

type DeviceFingerprint struct {
	FingerprintID        string                 `json:"fingerprint_id"`
	DeviceID             string                 `json:"device_id"`
	UserAgent            string                 `json:"user_agent"`
	CanvasHash           string                 `json:"canvas_hash"`
	WebGLHash            string                 `json:"webgl_hash"`
	AudioHash            string                 `json:"audio_hash"`
	FontHash             string                 `json:"font_hash"`
	ScreenResolution     string                 `json:"screen_resolution"`
	Timezone             string                 `json:"timezone"`
	Language             string                 `json:"language"`
	Platform             string                 `json:"platform"`
	HardwareConcurrency  int                    `json:"hardware_concurrency"`
	DeviceMemory         float64                `json:"device_memory"`
	Plugins              []string               `json:"plugins"`
	WebRTCIPs            []string               `json:"webrtc_ips"`
	ConnectionType       string                 `json:"connection_type"`
	NetworkLatency       float64                `json:"network_latency"`
	Headers              map[string]string      `json:"headers"`
	CreatedAt            time.Time              `json:"created_at"`
	LastSeen             time.Time              `json:"last_seen"`
	RequestCount         int                    `json:"request_count"`
	AssociatedUserIDs    []string               `json:"associated_user_ids"`
	IsKnownBot           bool                   `json:"is_known_bot"`
	IsKnownVPN           bool                   `json:"is_known_vpn"`
	AnomalyScore         float64                `json:"anomaly_score"`
	RiskIndicators       []string               `json:"risk_indicators"`
	Confidence           float64                `json:"confidence"`
	DerivedFingerprints  []string               `json:"derived_fingerprints"`
}

type FingerprintHistoryEntry struct {
	FingerprintID  string    `json:"fingerprint_id"`
	Timestamp      time.Time `json:"timestamp"`
	IPAddress      string    `json:"ip_address"`
	UserID         string    `json:"user_id"`
	Action         string    `json:"action"`
	SessionID      string    `json:"session_id"`
}

type DeviceGroup struct {
	GroupID          string   `json:"group_id"`
	DeviceIDs        []string `json:"device_ids"`
	CommonFeatures   []string `json:"common_features"`
	CreatedAt        time.Time `json:"created_at"`
	LastUpdated      time.Time `json:"last_updated"`
	IsSuspicious     bool      `json:"is_suspicious"`
	SuspicionReason  string    `json:"suspicion_reason"`
}

type FingerprintComparison struct {
	FingerprintID1    string  `json:"fingerprint_id1"`
	FingerprintID2    string  `json:"fingerprint_id2"`
	SimilarityScore   float64 `json:"similarity_score"`
	MatchDetails      map[string]bool `json:"match_details"`
	Confidence        float64 `json:"confidence"`
}

type TrackingResult struct {
	FingerprintID      string      `json:"fingerprint_id"`
	IsNewDevice        bool        `json:"is_new_device"`
	MatchedDeviceID    string      `json:"matched_device_id"`
	SimilarityScore    float64     `json:"similarity_score"`
	DeviceGroupID      string      `json:"device_group_id"`
	IsSuspicious       bool        `json:"is_suspicious"`
	RiskScore          float64     `json:"risk_score"`
	Recommendations    []string    `json:"recommendations"`
}

func NewDeviceFingerprintTracker() *DeviceFingerprintTracker {
	return &DeviceFingerprintTracker{
		fingerprints:        make(map[string]*DeviceFingerprint),
		deviceGroups:        make(map[string][]string),
		fingerprintHistory:  make(map[string][]FingerprintHistoryEntry),
		similarityCache:     make(map[string]map[string]float64),
		maxHistorySize:      1000,
		maxSimilarityCache:  10000,
	}
}

func (t *DeviceFingerprintTracker) RegisterFingerprint(data map[string]interface{}) (*DeviceFingerprint, error) {
	fingerprint := t.createFingerprint(data)
	
	t.mu.Lock()
	defer t.mu.Unlock()

	existingFP, exists := t.fingerprints[fingerprint.FingerprintID]
	if exists {
		existingFP.LastSeen = time.Now()
		existingFP.RequestCount++
		t.updateFingerprintHistory(existingFP.FingerprintID, data)
		return existingFP, nil
	}

	t.fingerprints[fingerprint.FingerprintID] = fingerprint
	t.updateFingerprintHistory(fingerprint.FingerprintID, data)
	t.detectAndUpdateGroups(fingerprint)

	return fingerprint, nil
}

func (t *DeviceFingerprintTracker) createFingerprint(data map[string]interface{}) *DeviceFingerprint {
	fingerprint := &DeviceFingerprint{
		FingerprintID:       generateFingerprintID(data),
		UserAgent:          getString(data, "user_agent"),
		CanvasHash:         getString(data, "canvas_hash"),
		WebGLHash:          getString(data, "webgl_hash"),
		AudioHash:          getString(data, "audio_hash"),
		FontHash:           getString(data, "font_hash"),
		ScreenResolution:   getString(data, "screen_resolution"),
		Timezone:          getString(data, "timezone"),
		Language:          getString(data, "language"),
		Platform:          getString(data, "platform"),
		ConnectionType:    getString(data, "connection_type"),
		CreatedAt:         time.Now(),
		LastSeen:          time.Now(),
		RequestCount:      1,
		Plugins:           getStringSlice(data, "plugins"),
		WebRTCIPs:         getStringSlice(data, "webrtc_ips"),
		Headers:          getStringMap(data, "headers"),
		DerivedFingerprints: make([]string, 0),
	}

	if hwConcurrency, ok := data["hardware_concurrency"].(float64); ok {
		fingerprint.HardwareConcurrency = int(hwConcurrency)
	}
	if deviceMemory, ok := data["device_memory"].(float64); ok {
		fingerprint.DeviceMemory = deviceMemory
	}
	if networkLatency, ok := data["network_latency"].(float64); ok {
		fingerprint.NetworkLatency = networkLatency
	}

	t.detectBotIndicators(fingerprint)
	t.detectVPNIndicators(fingerprint)
	t.calculateConfidence(fingerprint)
	t.generateDerivedFingerprints(fingerprint)

	return fingerprint
}

func generateFingerprintID(data map[string]interface{}) string {
	hasher := sha256.New()

	fields := []string{
		"user_agent", "canvas_hash", "webgl_hash", "audio_hash",
		"font_hash", "screen_resolution", "timezone", "platform",
	}
	sort.Strings(fields)

	for _, field := range fields {
		if val, ok := data[field].(string); ok && val != "" {
			hasher.Write([]byte(val))
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))[:20]
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getStringSlice(data map[string]interface{}, key string) []string {
	if raw, ok := data[key].([]interface{}); ok {
		result := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}

func getStringMap(data map[string]interface{}, key string) map[string]string {
	if raw, ok := data[key].(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range raw {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	return make(map[string]string)
}

func (t *DeviceFingerprintTracker) detectBotIndicators(fingerprint *DeviceFingerprint) {
	uaLower := strings.ToLower(fingerprint.UserAgent)
	indicators := []string{}

	botPatterns := []struct {
		pattern     string
		description string
		score       float64
	}{
		{"headless", "Headless browser detected", 20},
		{"phantom", "PhantomJS detected", 25},
		{"puppeteer", "Puppeteer automation detected", 22},
		{"playwright", "Playwright automation detected", 22},
		{"selenium", "Selenium detected", 20},
		{"webdriver", "WebDriver property detected", 23},
		{"chrome-headless", "Chrome headless mode", 18},
		{"firefox-headless", "Firefox headless mode", 18},
	}

	totalScore := 0.0
	for _, pattern := range botPatterns {
		if strings.Contains(uaLower, pattern.pattern) {
			indicators = append(indicators, pattern.description)
			totalScore += pattern.score
		}
	}

	if len(fingerprint.Plugins) == 0 {
		indicators = append(indicators, "No plugins detected")
		totalScore += 15
	}

	if fingerprint.WebGLHash == "" {
		indicators = append(indicators, "Missing WebGL fingerprint")
		totalScore += 10
	}

	fingerprint.RiskIndicators = indicators
	fingerprint.IsKnownBot = len(indicators) > 2 || totalScore > 40
	fingerprint.AnomalyScore = math.Min(totalScore, 100)
}

func (t *DeviceFingerprintTracker) detectVPNIndicators(fingerprint *DeviceFingerprint) {
	vpnIndicators := []string{}
	isVPN := false

	vpnIPPatterns := []string{"45.33.", "104.238.", "107.170.", "142.4.", "162.247."}
	for _, ip := range fingerprint.WebRTCIPs {
		for _, pattern := range vpnIPPatterns {
			if strings.HasPrefix(ip, pattern) {
				vpnIndicators = append(vpnIndicators, "Known VPN IP range")
				isVPN = true
				break
			}
		}
	}

	if fingerprint.ConnectionType == "vpn" || fingerprint.ConnectionType == "cellular" {
		vpnIndicators = append(vpnIndicators, "Connection type indicates VPN")
		isVPN = true
	}

	if len(fingerprint.WebRTCIPs) > 1 {
		privateCount := 0
		publicCount := 0
		for _, ip := range fingerprint.WebRTCIPs {
			if isPrivateIP(ip) {
				privateCount++
			} else {
				publicCount++
			}
		}
		if privateCount > 0 && publicCount > 0 {
			vpnIndicators = append(vpnIndicators, "Mixed private/public IPs detected")
			isVPN = true
		}
	}

	fingerprint.IsKnownVPN = isVPN
	if isVPN {
		fingerprint.RiskIndicators = append(fingerprint.RiskIndicators, vpnIndicators...)
		fingerprint.AnomalyScore += 20
	}
}

func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.", "127.", "169.254.",
	}
	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

func (t *DeviceFingerprintTracker) calculateConfidence(fingerprint *DeviceFingerprint) {
	fields := 0
	complete := 0

	checks := []interface{}{
		fingerprint.CanvasHash,
		fingerprint.WebGLHash,
		fingerprint.AudioHash,
		fingerprint.FontHash,
		fingerprint.UserAgent,
		fingerprint.ScreenResolution,
	}

	for _, check := range checks {
		fields++
		if check != nil && check != "" {
			complete++
		}
	}

	if fields > 0 {
		fingerprint.Confidence = float64(complete) / float64(fields)
	}
}

func (t *DeviceFingerprintTracker) generateDerivedFingerprints(fingerprint *DeviceFingerprint) {
	derived := []string{}

	components := []string{
		fingerprint.CanvasHash,
		fingerprint.WebGLHash,
		fingerprint.AudioHash,
		fingerprint.FontHash,
	}

	for i := 0; i < len(components); i++ {
		for j := i + 1; j < len(components); j++ {
			if components[i] != "" && components[j] != "" {
				combined := components[i][:min(len(components[i]), 8)] + "_" + components[j][:min(len(components[j]), 8)]
				derived = append(derived, combined)
			}
		}
	}

	fingerprint.DerivedFingerprints = derived
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *DeviceFingerprintTracker) updateFingerprintHistory(fingerprintID string, data map[string]interface{}) {
	entry := FingerprintHistoryEntry{
		FingerprintID: fingerprintID,
		Timestamp:     time.Now(),
		IPAddress:     getString(data, "public_ip"),
		UserID:        getString(data, "user_id"),
		Action:        getString(data, "action"),
		SessionID:     getString(data, "session_id"),
	}

	t.fingerprintHistory[fingerprintID] = append(t.fingerprintHistory[fingerprintID], entry)
	if len(t.fingerprintHistory[fingerprintID]) > t.maxHistorySize {
		t.fingerprintHistory[fingerprintID] = t.fingerprintHistory[fingerprintID][len(t.fingerprintHistory[fingerprintID])-t.maxHistorySize:]
	}
}

func (t *DeviceFingerprintTracker) detectAndUpdateGroups(fingerprint *DeviceFingerprint) {
	for groupID, deviceIDs := range t.deviceGroups {
		for _, deviceID := range deviceIDs {
			otherFP, exists := t.fingerprints[deviceID]
			if !exists {
				continue
			}

			similarity := t.calculateFingerprintSimilarity(fingerprint, otherFP)
			if similarity > 0.7 {
				t.deviceGroups[groupID] = append(t.deviceGroups[groupID], fingerprint.FingerprintID)
				return
			}
		}
	}

	newGroupID := "group_" + fingerprint.FingerprintID[:8] + "_" + fmt.Sprintf("%d", time.Now().Unix())
	t.deviceGroups[newGroupID] = []string{fingerprint.FingerprintID}
}

func (t *DeviceFingerprintTracker) calculateFingerprintSimilarity(fp1, fp2 *DeviceFingerprint) float64 {
	if t.similarityCache[fp1.FingerprintID] != nil {
		if score, exists := t.similarityCache[fp1.FingerprintID][fp2.FingerprintID]; exists {
			return score
		}
	}

	matchCount := 0
	totalCount := 0

	features := []struct {
		f1 string
		f2 string
	}{
		{fp1.CanvasHash, fp2.CanvasHash},
		{fp1.WebGLHash, fp2.WebGLHash},
		{fp1.AudioHash, fp2.AudioHash},
		{fp1.FontHash, fp2.FontHash},
		{fp1.ScreenResolution, fp2.ScreenResolution},
		{fp1.Timezone, fp2.Timezone},
		{fp1.Language, fp2.Language},
		{fp1.Platform, fp2.Platform},
	}

	for _, feature := range features {
		if feature.f1 != "" || feature.f2 != "" {
			totalCount++
			if feature.f1 == feature.f2 {
				matchCount++
			}
		}
	}

	similarity := float64(matchCount) / float64(totalCount)

	if t.similarityCache[fp1.FingerprintID] == nil {
		t.similarityCache[fp1.FingerprintID] = make(map[string]float64)
	}
	t.similarityCache[fp1.FingerprintID][fp2.FingerprintID] = similarity

	return similarity
}

func (t *DeviceFingerprintTracker) TrackDevice(data map[string]interface{}) (*TrackingResult, error) {
	fingerprint, err := t.RegisterFingerprint(data)
	if err != nil {
		return nil, err
	}

	result := &TrackingResult{
		FingerprintID: fingerprint.FingerprintID,
		IsNewDevice:   true,
		RiskScore:     fingerprint.AnomalyScore,
	}

	t.mu.RLock()
	for _, fp := range t.fingerprints {
		if fp.FingerprintID != fingerprint.FingerprintID {
			similarity := t.calculateFingerprintSimilarity(fingerprint, fp)
			if similarity > 0.8 {
				result.IsNewDevice = false
				result.MatchedDeviceID = fp.FingerprintID
				result.SimilarityScore = similarity
				break
			}
		}
	}
	t.mu.RUnlock()

	t.detectDeviceGroup(fingerprint, result)
	t.assessRisk(fingerprint, result)

	return result, nil
}

func (t *DeviceFingerprintTracker) detectDeviceGroup(fingerprint *DeviceFingerprint, result *TrackingResult) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for groupID, deviceIDs := range t.deviceGroups {
		for _, deviceID := range deviceIDs {
			if deviceID == fingerprint.FingerprintID {
				result.DeviceGroupID = groupID
				if len(deviceIDs) > 5 {
					result.IsSuspicious = true
					result.Recommendations = append(result.Recommendations, "Multiple devices in same group")
				}
				return
			}
		}
	}
}

func (t *DeviceFingerprintTracker) assessRisk(fingerprint *DeviceFingerprint, result *TrackingResult) {
	if fingerprint.IsKnownBot {
		result.IsSuspicious = true
		result.RiskScore += 30
		result.Recommendations = append(result.Recommendations, "Known bot signature detected")
	}

	if fingerprint.IsKnownVPN {
		result.RiskScore += 20
		result.Recommendations = append(result.Recommendations, "VPN/proxy detected")
	}

	if len(fingerprint.AssociatedUserIDs) > 10 {
		result.IsSuspicious = true
		result.RiskScore += 15
		result.Recommendations = append(result.Recommendations, "Multiple users associated with device")
	}

	result.RiskScore = math.Min(result.RiskScore, 100)
}

func (t *DeviceFingerprintTracker) GetFingerprint(fingerprintID string) (*DeviceFingerprint, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fp, exists := t.fingerprints[fingerprintID]
	return fp, exists
}

func (t *DeviceFingerprintTracker) GetDeviceGroup(groupID string) (*DeviceGroup, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	deviceIDs, exists := t.deviceGroups[groupID]
	if !exists {
		return nil, false
	}

	group := &DeviceGroup{
		GroupID:     groupID,
		DeviceIDs:   deviceIDs,
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	commonFeatures := t.findCommonFeatures(deviceIDs)
	group.CommonFeatures = commonFeatures

	if len(deviceIDs) > 10 {
		group.IsSuspicious = true
		group.SuspicionReason = "More than 10 devices in group"
	}

	return group, true
}

func (t *DeviceFingerprintTracker) findCommonFeatures(deviceIDs []string) []string {
	features := make(map[string]int)

	for _, deviceID := range deviceIDs {
		if fp, exists := t.fingerprints[deviceID]; exists {
			if fp.UserAgent != "" {
				features["user_agent:"+fp.UserAgent[:min(20, len(fp.UserAgent))]]++
			}
			if fp.ScreenResolution != "" {
				features["resolution:"+fp.ScreenResolution]++
			}
			if fp.Platform != "" {
				features["platform:"+fp.Platform]++
			}
		}
	}

	common := []string{}
	threshold := len(deviceIDs) * 2 / 3
	for feature, count := range features {
		if count >= threshold {
			common = append(common, feature)
		}
	}

	return common
}

func (t *DeviceFingerprintTracker) CompareFingerprints(fp1ID, fp2ID string) (*FingerprintComparison, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fp1, exists := t.fingerprints[fp1ID]
	if !exists {
		return nil, errors.New("fingerprint 1 not found")
	}

	fp2, exists := t.fingerprints[fp2ID]
	if !exists {
		return nil, errors.New("fingerprint 2 not found")
	}

	similarity := t.calculateFingerprintSimilarity(fp1, fp2)

	matchDetails := map[string]bool{
		"canvas":         fp1.CanvasHash == fp2.CanvasHash,
		"webgl":          fp1.WebGLHash == fp2.WebGLHash,
		"audio":          fp1.AudioHash == fp2.AudioHash,
		"font":           fp1.FontHash == fp2.FontHash,
		"screen":         fp1.ScreenResolution == fp2.ScreenResolution,
		"timezone":       fp1.Timezone == fp2.Timezone,
		"language":       fp1.Language == fp2.Language,
		"platform":       fp1.Platform == fp2.Platform,
	}

	return &FingerprintComparison{
		FingerprintID1:  fp1ID,
		FingerprintID2:  fp2ID,
		SimilarityScore: similarity,
		MatchDetails:   matchDetails,
		Confidence:     (fp1.Confidence + fp2.Confidence) / 2,
	}, nil
}

func (t *DeviceFingerprintTracker) GetFingerprintsByUserID(userID string) []*DeviceFingerprint {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := []*DeviceFingerprint{}
	for _, fp := range t.fingerprints {
		for _, uid := range fp.AssociatedUserIDs {
			if uid == userID {
				result = append(result, fp)
				break
			}
		}
	}
	return result
}

func (t *DeviceFingerprintTracker) AssociateUser(fingerprintID, userID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	fp, exists := t.fingerprints[fingerprintID]
	if !exists {
		return errors.New("fingerprint not found")
	}

	for _, uid := range fp.AssociatedUserIDs {
		if uid == userID {
			return nil
		}
	}

	fp.AssociatedUserIDs = append(fp.AssociatedUserIDs, userID)
	return nil
}

func (t *DeviceFingerprintTracker) GetAllFingerprints() []*DeviceFingerprint {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*DeviceFingerprint, 0, len(t.fingerprints))
	for _, fp := range t.fingerprints {
		result = append(result, fp)
	}
	return result
}

func (t *DeviceFingerprintTracker) GetAllDeviceGroups() []*DeviceGroup {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*DeviceGroup, 0, len(t.deviceGroups))
	for groupID := range t.deviceGroups {
		group, _ := t.GetDeviceGroup(groupID)
		result = append(result, group)
	}
	return result
}

func (t *DeviceFingerprintTracker) GetFingerprintHistory(fingerprintID string) []FingerprintHistoryEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	history, exists := t.fingerprintHistory[fingerprintID]
	if !exists {
		return []FingerprintHistoryEntry{}
	}
	return history
}

func (t *DeviceFingerprintTracker) CleanupOldFingerprints(maxAge time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, fp := range t.fingerprints {
		if fp.LastSeen.Before(cutoff) && fp.RequestCount < 5 {
			delete(t.fingerprints, id)
			delete(t.fingerprintHistory, id)
			removed++
		}
	}

	return removed
}

func (t *DeviceFingerprintTracker) ExportFingerprints() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	type ExportData struct {
		Fingerprints     map[string]*DeviceFingerprint `json:"fingerprints"`
		DeviceGroups     map[string][]string           `json:"device_groups"`
		ExportedAt       time.Time                    `json:"exported_at"`
	}

	data := ExportData{
		Fingerprints: t.fingerprints,
		DeviceGroups: t.deviceGroups,
		ExportedAt:   time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

func (t *DeviceFingerprintTracker) ImportFingerprints(data []byte) error {
	type ImportData struct {
		Fingerprints map[string]*DeviceFingerprint `json:"fingerprints"`
		DeviceGroups map[string][]string           `json:"device_groups"`
	}

	var importData ImportData
	if err := json.Unmarshal(data, &importData); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for id, fp := range importData.Fingerprints {
		fp.FingerprintID = id
		t.fingerprints[id] = fp
	}

	for groupID, deviceIDs := range importData.DeviceGroups {
		t.deviceGroups[groupID] = deviceIDs
	}

	return nil
}

func (t *DeviceFingerprintTracker) GetTrackerStatistics() *TrackerStatistics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &TrackerStatistics{
		TotalFingerprints: len(t.fingerprints),
		TotalDeviceGroups: len(t.deviceGroups),
		BotCount:          0,
		VPNCount:          0,
		AvgConfidence:     0,
		AvgAnomalyScore:   0,
	}

	totalConfidence := 0.0
	totalAnomalyScore := 0.0

	for _, fp := range t.fingerprints {
		if fp.IsKnownBot {
			stats.BotCount++
		}
		if fp.IsKnownVPN {
			stats.VPNCount++
		}
		totalConfidence += fp.Confidence
		totalAnomalyScore += fp.AnomalyScore
	}

	if stats.TotalFingerprints > 0 {
		stats.AvgConfidence = totalConfidence / float64(stats.TotalFingerprints)
		stats.AvgAnomalyScore = totalAnomalyScore / float64(stats.TotalFingerprints)
	}

	return stats
}

type TrackerStatistics struct {
	TotalFingerprints int     `json:"total_fingerprints"`
	TotalDeviceGroups int     `json:"total_device_groups"`
	BotCount          int     `json:"bot_count"`
	VPNCount          int     `json:"vpn_count"`
	AvgConfidence     float64 `json:"avg_confidence"`
	AvgAnomalyScore   float64 `json:"avg_anomaly_score"`
}

func (t *DeviceFingerprintTracker) DetectSuspiciousGroups() []*DeviceGroup {
	t.mu.RLock()
	defer t.mu.RUnlock()

	suspicious := []*DeviceGroup{}
	for groupID, deviceIDs := range t.deviceGroups {
		if len(deviceIDs) > 5 {
			group, _ := t.GetDeviceGroup(groupID)
			group.IsSuspicious = true
			group.SuspicionReason = fmt.Sprintf("%d devices in group", len(deviceIDs))
			suspicious = append(suspicious, group)
		}
	}
	return suspicious
}

func (t *DeviceFingerprintTracker) FindRelatedFingerprints(fingerprintID string, threshold float64) []*FingerprintComparison {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fp, exists := t.fingerprints[fingerprintID]
	if !exists {
		return []*FingerprintComparison{}
	}

	results := []*FingerprintComparison{}
	for _, otherFP := range t.fingerprints {
		if otherFP.FingerprintID == fingerprintID {
			continue
		}

		similarity := t.calculateFingerprintSimilarity(fp, otherFP)
		if similarity >= threshold {
			results = append(results, &FingerprintComparison{
				FingerprintID1:  fingerprintID,
				FingerprintID2:  otherFP.FingerprintID,
				SimilarityScore: similarity,
				Confidence:     (fp.Confidence + otherFP.Confidence) / 2,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].SimilarityScore > results[j].SimilarityScore
	})

	return results
}