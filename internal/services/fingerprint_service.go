package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type FingerprintService struct {
	config    *FingerprintConfig
	browserDB *BrowserDatabase
	deviceDB  *DeviceDatabase
	osDB      *OSDatabase
}

type FingerprintConfig struct {
	EnableDeepParse   bool
	CacheExpiration   time.Duration
	MaxCacheSize      int
	EnableBotDetect   bool
}

var DefaultFingerprintConfig = &FingerprintConfig{
	EnableDeepParse:   true,
	CacheExpiration:   1 * time.Hour,
	MaxCacheSize:      10000,
	EnableBotDetect:   true,
}

type FingerprintResult struct {
	IsValid           bool            `json:"is_valid"`
	DeviceType        DeviceType       `json:"device_type"`
	DeviceBrand       string           `json:"device_brand,omitempty"`
	DeviceModel       string           `json:"device_model,omitempty"`
	Browser           BrowserInfo      `json:"browser"`
	OperatingSystem   OSInfo           `json:"operating_system"`
	HardwareInfo      *HardwareInfo    `json:"hardware_info,omitempty"`
	RiskScore         float64          `json:"risk_score"`
	IsBot             bool             `json:"is_bot"`
	Confidence        float64          `json:"confidence"`
	Fingerprint       string           `json:"fingerprint"`
	RawData           *RawFingerprint  `json:"raw_data"`
	Warnings          []string         `json:"warnings,omitempty"`
}

type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeTablet  DeviceType = "tablet"
	DeviceTypeTV      DeviceType = "tv"
	DeviceTypeBot     DeviceType = "bot"
	DeviceTypeUnknown DeviceType = "unknown"
)

type BrowserInfo struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	FullName   string  `json:"full_name"`
	Engine     string  `json:"engine,omitempty"`
	IsModern   bool    `json:"is_modern"`
	IsOutdated bool    `json:"is_outdated"`
}

type OSInfo struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	FullName   string  `json:"full_name"`
	Family     string  `json:"family"`
	Is64Bit    bool    `json:"is_64bit"`
}

type HardwareInfo struct {
	Architecture string  `json:"architecture,omitempty"`
	Cores        int     `json:"cores,omitempty"`
	Memory       int64   `json:"memory,omitempty"`
	GPU          string  `json:"gpu,omitempty"`
	ScreenWidth  int     `json:"screen_width,omitempty"`
	ScreenHeight int     `json:"screen_height,omitempty"`
	PixelRatio   float64 `json:"pixel_ratio,omitempty"`
}

type RawFingerprint struct {
	UserAgent      string            `json:"user_agent"`
	AcceptLanguage string            `json:"accept_language,omitempty"`
	AcceptEncoding string            `json:"accept_encoding,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
}

type BrowserDatabase struct {
	mu sync.RWMutex
	browsers map[string]*BrowserPattern
}

type BrowserPattern struct {
	Name      string
	Regex     *regexp.Regexp
	VersionGroups []int
	Engine    string
	ModernVersions map[string]int
	OutdatedVersions map[string]int
}

type DeviceDatabase struct {
	mu sync.RWMutex
	mobilePatterns  []DevicePattern
	tabletPatterns  []DevicePattern
	botPatterns     []BotPattern
}

type DevicePattern struct {
	Type    DeviceType
	Brand   string
	Model   string
	Regex   *regexp.Regexp
}

type BotPattern struct {
	Name   string
	Regex  *regexp.Regexp
}

type OSDatabase struct {
	mu sync.RWMutex
	osPatterns []OSPattern
}

type OSPattern struct {
	Name    string
	Family  string
	Regex   *regexp.Regexp
	VersionGroups []int
}

func NewFingerprintService(config *FingerprintConfig) *FingerprintService {
	if config == nil {
		config = DefaultFingerprintConfig
	}

	service := &FingerprintService{
		config:    config,
		browserDB: NewBrowserDatabase(),
		deviceDB:  NewDeviceDatabase(),
		osDB:      NewOSDatabase(),
	}

	return service
}

func (s *FingerprintService) ParseFingerprint(ctx context.Context, req *FingerprintRequest) (*FingerprintResult, error) {
	result := &FingerprintResult{
		IsValid:   true,
		Fingerprint: generateFingerprint(req.UserAgent),
		RawData: &RawFingerprint{
			UserAgent:      req.UserAgent,
			AcceptLanguage: req.AcceptLanguage,
			AcceptEncoding: req.AcceptEncoding,
			Headers:        req.Headers,
		},
		Warnings: make([]string, 0),
	}

	if req.UserAgent == "" {
		result.Warnings = append(result.Warnings, "Empty User-Agent")
		result.RiskScore = 0.5
		result.Confidence = 0.3
		return result, nil
	}

	browser := s.browserDB.DetectBrowser(req.UserAgent)
	result.Browser = browser

	os := s.osDB.DetectOS(req.UserAgent)
	result.OperatingSystem = os

	deviceType := s.deviceDB.DetectDeviceType(req.UserAgent)
	result.DeviceType = deviceType

	if device, found := s.deviceDB.DetectDevice(req.UserAgent); found {
		result.DeviceBrand = device.Brand
		result.DeviceModel = device.Model
	}

	if s.config.EnableBotDetect {
		if botInfo := s.deviceDB.DetectBot(req.UserAgent); botInfo != nil {
			result.IsBot = true
			result.RiskScore = 0.8
			result.DeviceType = DeviceTypeBot
		}
	}

	result.RiskScore = s.calculateRiskScore(result)
	result.Confidence = s.calculateConfidence(result, browser, os)
	result.IsValid = result.RiskScore < 0.7

	return result, nil
}

func (s *FingerprintService) DetectBrowser(ua string) BrowserInfo {
	return s.browserDB.DetectBrowser(ua)
}

func (s *FingerprintService) DetectOS(ua string) OSInfo {
	return s.osDB.DetectOS(ua)
}

func (s *FingerprintService) DetectDeviceType(ua string) DeviceType {
	return s.deviceDB.DetectDeviceType(ua)
}

func (s *FingerprintService) GenerateDeviceFingerprint(ctx context.Context, req *FingerprintRequest) (string, error) {
	components := []string{
		req.UserAgent,
		req.AcceptLanguage,
		req.AcceptEncoding,
	}

	if req.Headers != nil {
		if accept := req.Headers["Accept"]; accept != "" {
			components = append(components, accept)
		}
		if acceptEnc := req.Headers["Accept-Encoding"]; acceptEnc != "" {
			components = append(components, acceptEnc)
		}
	}

	fingerprint := generateFingerprintHash(strings.Join(components, "|"))

	return fingerprint, nil
}

func (s *FingerprintService) calculateRiskScore(result *FingerprintResult) float64 {
	score := 0.0

	if result.IsBot {
		return 1.0
	}

	if result.Browser.Name == "" || result.Browser.Name == "unknown" {
		score += 0.3
		result.Warnings = append(result.Warnings, "Unrecognized browser")
	}

	if result.OperatingSystem.Name == "" || result.OperatingSystem.Name == "unknown" {
		score += 0.2
		result.Warnings = append(result.Warnings, "Unrecognized operating system")
	}

	if result.Browser.IsOutdated {
		score += 0.1
		result.Warnings = append(result.Warnings, "Browser version is outdated")
	}

	if result.DeviceType == DeviceTypeUnknown {
		score += 0.2
		result.Warnings = append(result.Warnings, "Unable to determine device type")
	}

	if strings.Contains(strings.ToLower(result.RawData.UserAgent), "headless") {
		score += 0.5
		result.Warnings = append(result.Warnings, "Headless browser detected")
	}

	if strings.Contains(strings.ToLower(result.RawData.UserAgent), "phantom") {
		score += 0.5
		result.Warnings = append(result.Warnings, "PhantomJS detected")
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (s *FingerprintService) calculateConfidence(result *FingerprintResult, browser BrowserInfo, os OSInfo) float64 {
	confidence := 0.3

	if browser.Name != "" && browser.Name != "unknown" {
		confidence += 0.3
	}

	if os.Name != "" && os.Name != "unknown" {
		confidence += 0.2
	}

	if result.DeviceType != DeviceTypeUnknown {
		confidence += 0.1
	}

	if result.DeviceBrand != "" {
		confidence += 0.1
	}

	if !result.Browser.IsOutdated && !result.Browser.IsModern {
		confidence += 0.1
	}

	if len(result.Warnings) == 0 {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func generateFingerprint(ua string) string {
	if ua == "" {
		return ""
	}

	var hash uint64
	for i := 0; i < len(ua) && i < 100; i++ {
		hash = hash*31 + uint64(ua[i])
	}

	return fmt.Sprintf("%016x", hash)
}

func generateFingerprintHash(data string) string {
	var hash uint64
	for i := 0; i < len(data); i++ {
		hash = hash*31 + uint64(data[i])
	}

	return fmt.Sprintf("%016x", hash)
}

func NewBrowserDatabase() *BrowserDatabase {
	db := &BrowserDatabase{
		browsers: make(map[string]*BrowserPattern),
	}

	db.browsers = map[string]*BrowserPattern{
		"chrome": {
			Name: "Chrome",
			Regex: regexp.MustCompile(`(?i)Chrome/(\d+)\.(\d+)\.(\d+)\.(\d+)`),
			VersionGroups: []int{1},
			Engine: "Blink",
			ModernVersions: map[string]int{"120": 1},
			OutdatedVersions: map[string]int{"70": 1},
		},
		"firefox": {
			Name: "Firefox",
			Regex: regexp.MustCompile(`(?i)Firefox/(\d+)\.(\d+)`),
			VersionGroups: []int{1},
			Engine: "Gecko",
			ModernVersions: map[string]int{"120": 1},
			OutdatedVersions: map[string]int{"70": 1},
		},
		"safari": {
			Name: "Safari",
			Regex: regexp.MustCompile(`(?i)Version/(\d+)\.(\d+).*Safari`),
			VersionGroups: []int{1},
			Engine: "WebKit",
			ModernVersions: map[string]int{"17": 1},
			OutdatedVersions: map[string]int{"14": 1},
		},
		"edge": {
			Name: "Edge",
			Regex: regexp.MustCompile(`(?i)Edg(e|A|iOS)?/(\d+)\.(\d+)\.(\d+)\.(\d+)`),
			VersionGroups: []int{2},
			Engine: "Blink",
			ModernVersions: map[string]int{"120": 1},
			OutdatedVersions: map[string]int{"90": 1},
		},
		"opera": {
			Name: "Opera",
			Regex: regexp.MustCompile(`(?i)(Opera|OPR)/(\d+)\.(\d+)`),
			VersionGroups: []int{2},
			Engine: "Blink",
			ModernVersions: map[string]int{"100": 1},
			OutdatedVersions: map[string]int{"70": 1},
		},
		"samsung": {
			Name: "Samsung Browser",
			Regex: regexp.MustCompile(`(?i)SamsungBrowser/(\d+)\.(\d+)`),
			VersionGroups: []int{1},
			Engine: "Blink",
			ModernVersions: map[string]int{"22": 1},
			OutdatedVersions: map[string]int{"15": 1},
		},
		"ie": {
			Name: "Internet Explorer",
			Regex: regexp.MustCompile(`(?i)MSIE (\d+)\.(\d+)`),
			VersionGroups: []int{1},
			Engine: "Trident",
			OutdatedVersions: map[string]int{"11": 1},
		},
		"ie_edge": {
			Name: "Internet Explorer",
			Regex: regexp.MustCompile(`(?i)Trident/(\d+)\.(\d+).*rv:(\d+)\.(\d+)`),
			VersionGroups: []int{3},
			Engine: "Trident",
			OutdatedVersions: map[string]int{"11": 1},
		},
	}

	return db
}

func (db *BrowserDatabase) DetectBrowser(ua string) BrowserInfo {
	if ua == "" {
		return BrowserInfo{Name: "unknown"}
	}

	browserNames := []string{"chrome", "firefox", "safari", "edge", "opera", "samsung", "ie", "ie_edge"}

	for _, name := range browserNames {
		pattern, ok := db.browsers[name]
		if !ok {
			continue
		}

		matches := pattern.Regex.FindStringSubmatch(ua)
		if len(matches) > 1 {
			version := matches[pattern.VersionGroups[0]]

			return BrowserInfo{
				Name:     pattern.Name,
				Version:  version,
				FullName: fmt.Sprintf("%s %s", pattern.Name, version),
				Engine:   pattern.Engine,
				IsModern: pattern.checkModern(version),
				IsOutdated: pattern.checkOutdated(version),
			}
		}
	}

	return BrowserInfo{Name: "unknown"}
}

func (p *BrowserPattern) checkModern(version string) bool {
	if p.ModernVersions == nil {
		return false
	}

	var majorVersion int
	fmt.Sscanf(version, "%d", &majorVersion)

	for minVersion := range p.ModernVersions {
		var threshold int
		fmt.Sscanf(minVersion, "%d", &threshold)
		if majorVersion >= threshold {
			return true
		}
	}

	return false
}

func (p *BrowserPattern) checkOutdated(version string) bool {
	if p.OutdatedVersions == nil {
		return false
	}

	var majorVersion int
	fmt.Sscanf(version, "%d", &majorVersion)

	for maxVersion := range p.OutdatedVersions {
		var threshold int
		fmt.Sscanf(maxVersion, "%d", &threshold)
		if majorVersion <= threshold {
			return true
		}
	}

	return false
}

func NewDeviceDatabase() *DeviceDatabase {
	db := &DeviceDatabase{
		mobilePatterns: []DevicePattern{
			{Type: DeviceTypeMobile, Brand: "Apple", Model: "iPhone", Regex: regexp.MustCompile(`(?i)iPhone`)},
			{Type: DeviceTypeMobile, Brand: "Apple", Model: "iPod", Regex: regexp.MustCompile(`(?i)iPod`)},
			{Type: DeviceTypeMobile, Brand: "Samsung", Model: "Galaxy", Regex: regexp.MustCompile(`(?i)Android.*Samsung|SM-`)},
			{Type: DeviceTypeMobile, Brand: "Google", Model: "Pixel", Regex: regexp.MustCompile(`(?i)Android.*Pixel`)},
			{Type: DeviceTypeMobile, Brand: "Huawei", Model: "Various", Regex: regexp.MustCompile(`(?i)Android.*HUAWEI|HONOR`)},
			{Type: DeviceTypeMobile, Brand: "Xiaomi", Model: "Various", Regex: regexp.MustCompile(`(?i)Android.*Mi |Redmi |Xiaomi`)},
			{Type: DeviceTypeMobile, Brand: "OPPO", Model: "Various", Regex: regexp.MustCompile(`(?i)Android.*OPPO|CPH`)},
			{Type: DeviceTypeMobile, Brand: "Vivo", Model: "Various", Regex: regexp.MustCompile(`(?i)Android.*Vivo|Vivo`)},
		},
		tabletPatterns: []DevicePattern{
			{Type: DeviceTypeTablet, Brand: "Apple", Model: "iPad", Regex: regexp.MustCompile(`(?i)iPad`)},
			{Type: DeviceTypeTablet, Brand: "Samsung", Model: "Galaxy Tab", Regex: regexp.MustCompile(`(?i)Android.*Tablet|SM-T`)},
			{Type: DeviceTypeTablet, Brand: "Amazon", Model: "Fire", Regex: regexp.MustCompile(`(?i)Android.*Kindle|Fire.*Build`)},
			{Type: DeviceTypeTablet, Brand: "Microsoft", Model: "Surface", Regex: regexp.MustCompile(`(?i)Windows.*Touch|Surface`)},
		},
		botPatterns: []BotPattern{
			{Name: "Googlebot", Regex: regexp.MustCompile(`(?i)Googlebot|Googlebot-Image|Googlebot-News`)},
			{Name: "Bingbot", Regex: regexp.MustCompile(`(?i)bingbot|BingPreview`)},
			{Name: "Yahoo Slurp", Regex: regexp.MustCompile(`(?i)Yahoo! Slurp`)},
			{Name: "DuckDuckBot", Regex: regexp.MustCompile(`(?i)DuckDuckBot`)},
			{Name: "Baiduspider", Regex: regexp.MustCompile(`(?i)Baiduspider`)},
			{Name: "Yandex Bot", Regex: regexp.MustCompile(`(?i)YandexBot|YandexImages`)},
			{Name: "Facebook Bot", Regex: regexp.MustCompile(`(?i)facebookexternalhit|Facebot`)},
			{Name: "Twitter Bot", Regex: regexp.MustCompile(`(?i)Twitterbot`)},
			{Name: "Apple Bot", Regex: regexp.MustCompile(`(?i)Applebot`)},
			{Name: "LinkedIn Bot", Regex: regexp.MustCompile(`(?i)linkedinbot`)},
			{Name: "Slack Bot", Regex: regexp.MustCompile(`(?i)Slackbot`)},
			{Name: "Discord Bot", Regex: regexp.MustCompile(`(?i)Discordbot`)},
			{Name: "curl", Regex: regexp.MustCompile(`(?i)curl/`)},
			{Name: "wget", Regex: regexp.MustCompile(`(?i)Wget`)},
			{Name: "Python urllib", Regex: regexp.MustCompile(`(?i)Python-urllib|python-requests`)},
		},
	}

	return db
}

func (db *DeviceDatabase) DetectDeviceType(ua string) DeviceType {
	if ua == "" {
		return DeviceTypeUnknown
	}

	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "android") {
		if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
			return DeviceTypeTablet
		}
		if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipod") ||
			strings.Contains(uaLower, "android") || strings.Contains(uaLower, "mobile") {
			return DeviceTypeMobile
		}
	}

	for _, pattern := range db.mobilePatterns {
		if pattern.Regex.MatchString(ua) {
			return DeviceTypeMobile
		}
	}

	for _, pattern := range db.tabletPatterns {
		if pattern.Regex.MatchString(ua) {
			return DeviceTypeTablet
		}
	}

	pcPatterns := regexp.MustCompile(`(?i)(Windows NT|Macintosh|X11|Linux)`)
	if pcPatterns.MatchString(ua) && !strings.Contains(uaLower, "mobile") && !strings.Contains(uaLower, "android") {
		return DeviceTypeDesktop
	}

	iosPattern := regexp.MustCompile(`(?i)iPhone|iPod|iPad`)
	if iosPattern.MatchString(ua) {
		if strings.Contains(uaLower, "ipad") {
			return DeviceTypeTablet
		}
		return DeviceTypeMobile
	}

	return DeviceTypeDesktop
}

func (db *DeviceDatabase) DetectDevice(ua string) (DevicePattern, bool) {
	if ua == "" {
		return DevicePattern{}, false
	}

	for _, pattern := range db.mobilePatterns {
		if pattern.Regex.MatchString(ua) {
			return pattern, true
		}
	}

	for _, pattern := range db.tabletPatterns {
		if pattern.Regex.MatchString(ua) {
			return pattern, true
		}
	}

	return DevicePattern{}, false
}

func (db *DeviceDatabase) DetectBot(ua string) *BotPattern {
	if ua == "" {
		return nil
	}

	for _, pattern := range db.botPatterns {
		if pattern.Regex.MatchString(ua) {
			return &pattern
		}
	}

	return nil
}

func NewOSDatabase() *OSDatabase {
	db := &OSDatabase{
		osPatterns: []OSPattern{
			{
				Name:    "Windows",
				Family:  "Windows",
				Regex:   regexp.MustCompile(`(?i)Windows NT (\d+)\.(\d+)`),
				VersionGroups: []int{1},
			},
			{
				Name:    "macOS",
				Family:  "Apple",
				Regex:   regexp.MustCompile(`(?i)Mac OS X (\d+)[._](\d+)[._]?(\d+)?`),
				VersionGroups: []int{1, 2},
			},
			{
				Name:    "iOS",
				Family:  "Apple",
				Regex:   regexp.MustCompile(`(?i)iPhone OS (\d+)[._](\d+)[._]?(\d+)?`),
				VersionGroups: []int{1, 2},
			},
			{
				Name:    "Android",
				Family:  "Linux",
				Regex:   regexp.MustCompile(`(?i)Android ?(\d+)\.?(\d+)?\.?(\d+)?`),
				VersionGroups: []int{1, 2},
			},
			{
				Name:    "Linux",
				Family:  "Linux",
				Regex:   regexp.MustCompile(`(?i)Linux|X11`),
				VersionGroups: []int{},
			},
			{
				Name:    "Chrome OS",
				Family:  "Linux",
				Regex:   regexp.MustCompile(`(?i)CrOS`),
				VersionGroups: []int{},
			},
		},
	}

	return db
}

func (db *OSDatabase) DetectOS(ua string) OSInfo {
	if ua == "" {
		return OSInfo{Name: "unknown"}
	}

	is64Bit := strings.Contains(strings.ToLower(ua), "wow64") ||
		strings.Contains(strings.ToLower(ua), "win64") ||
		strings.Contains(strings.ToLower(ua), "x64") ||
		strings.Contains(strings.ToLower(ua), "amd64")

	if strings.Contains(strings.ToLower(ua), "android") {
		androidPattern := regexp.MustCompile(`(?i)Android ?(\d+)\.?(\d+)?\.?(\d+)?`)
		androidMatch := androidPattern.FindStringSubmatch(ua)
		if len(androidMatch) > 1 {
			version := androidMatch[1]
			if len(androidMatch) > 2 && androidMatch[2] != "" {
				version = fmt.Sprintf("%s.%s", androidMatch[1], androidMatch[2])
			}
			return OSInfo{
				Name:     "Android",
				Version:  version,
				FullName: fmt.Sprintf("Android %s", version),
				Family:   "Linux",
				Is64Bit:  is64Bit,
			}
		}
	}

	for _, pattern := range db.osPatterns {
		if pattern.Name == "Android" || pattern.Name == "Linux" {
			continue
		}
		matches := pattern.Regex.FindStringSubmatch(ua)
		if len(matches) > 0 {
			version := ""
			if len(pattern.VersionGroups) >= 2 {
				version = fmt.Sprintf("%s.%s", matches[pattern.VersionGroups[0]], matches[pattern.VersionGroups[1]])
			} else if len(pattern.VersionGroups) == 1 {
				version = matches[pattern.VersionGroups[0]]
			}

			fullName := fmt.Sprintf("%s %s", pattern.Name, version)
			if version == "" {
				fullName = pattern.Name
			}

			return OSInfo{
				Name:     pattern.Name,
				Version:  version,
				FullName: strings.TrimSpace(fullName),
				Family:   pattern.Family,
				Is64Bit:  is64Bit,
			}
		}
	}

	return OSInfo{Name: "unknown"}
}

type FingerprintRequest struct {
	UserAgent      string
	AcceptLanguage string
	AcceptEncoding string
	Headers        map[string]string
}

func (s *FingerprintService) ParseBatch(ctx context.Context, requests []*FingerprintRequest) ([]*FingerprintResult, error) {
	results := make([]*FingerprintResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)
	errMu := sync.Mutex{}

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, request *FingerprintRequest) {
			defer wg.Done()

			result, err := s.ParseFingerprint(ctx, request)
			mu.Lock()
			results[idx] = result
			if err != nil {
				errMu.Lock()
				errs = append(errs, fmt.Errorf("request %d: %w", idx, err))
				errMu.Unlock()
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()

	if len(errs) > 0 {
		return results, errs[0]
	}

	return results, nil
}

func (s *FingerprintService) ValidateUserAgent(ua string) (bool, string) {
	if ua == "" {
		return false, "Empty User-Agent"
	}

	if len(ua) < 10 {
		return false, "User-Agent too short"
	}

	if len(ua) > 500 {
		return false, "User-Agent suspiciously long"
	}

	suspiciousPatterns := []string{
		"$(", "&&", "|", ">", "<", "`",
	}

	uaLower := strings.ToLower(ua)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(uaLower, pattern) {
			return false, fmt.Sprintf("Suspicious pattern detected: %s", pattern)
		}
	}

	return true, ""
}
