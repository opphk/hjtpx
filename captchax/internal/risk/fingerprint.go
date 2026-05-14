package risk

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DeviceFingerprint struct {
	UserAgent       string
	OS              string
	OSVersion       string
	Browser         string
	BrowserVersion  string
	DeviceType      DeviceType
	ScreenWidth     int
	ScreenHeight    int
	ColorDepth      int
	Timezone        string
	Language        string
	Plugins         []string
	CanvasHash      string
	WebGLRenderer   string
	Platform        string
	CookieEnabled   bool
	DoNotTrack      bool
	AcceptLanguage  string
	Hash            string
	IsBot           bool
}

type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeTablet  DeviceType = "tablet"
	DeviceTypeBot     DeviceType = "bot"
	DeviceTypeUnknown DeviceType = "unknown"
)

type FingerprintRisk struct {
	Score           int
	Factors         []RiskFactor
	IsTorBrowser    bool
	IsVPNDetected   bool
	IsProxyDetected bool
	IsBot           bool
	IsSuspiciousUA  bool
}

func NewDeviceFingerprint(userAgent, acceptLanguage string, screenWidth, screenHeight int) *DeviceFingerprint {
	fp := &DeviceFingerprint{
		UserAgent:      userAgent,
		AcceptLanguage: acceptLanguage,
		ScreenWidth:    screenWidth,
		ScreenHeight:   screenHeight,
	}

	fp.parseUserAgent()
	fp.detectDeviceType()
	fp.Hash = fp.GenerateHash()

	return fp
}

func (fp *DeviceFingerprint) parseUserAgent() {
	ua := strings.ToLower(fp.UserAgent)

	fp.IsBot = fp.detectBot(ua)

	if strings.Contains(ua, "windows") {
		fp.OS = "Windows"
		fp.Platform = "Win32"
		if strings.Contains(ua, "windows nt 10") {
			fp.OSVersion = "10"
		} else if strings.Contains(ua, "windows nt 6.3") {
			fp.OSVersion = "8.1"
		} else if strings.Contains(ua, "windows nt 6.2") {
			fp.OSVersion = "8"
		} else if strings.Contains(ua, "windows nt 6.1") {
			fp.OSVersion = "7"
		}
	} else if strings.Contains(ua, "mac os x") || strings.Contains(ua, "macos") {
		fp.OS = "macOS"
		fp.Platform = "MacIntel"
		versionMatch := regexp.MustCompile(`mac os x (\d+[._]\d+)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.OSVersion = strings.Replace(versionMatch[1], "_", ".", -1)
		}
	} else if strings.Contains(ua, "linux") {
		fp.OS = "Linux"
		fp.Platform = "Linux x86_64"
		if strings.Contains(ua, "android") {
			fp.OS = "Android"
			versionMatch := regexp.MustCompile(`android (\d+(\.\d+)?)`).FindStringSubmatch(ua)
			if len(versionMatch) > 1 {
				fp.OSVersion = versionMatch[1]
			}
		}
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") {
		fp.OS = "iOS"
		fp.Platform = "iPhone"
		versionMatch := regexp.MustCompile(`os (\d+[._]\d+)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.OSVersion = strings.Replace(versionMatch[1], "_", ".", -1)
		}
	}

	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") && !strings.Contains(ua, "opr") {
		fp.Browser = "Chrome"
		versionMatch := regexp.MustCompile(`chrome/(\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	} else if strings.Contains(ua, "firefox") {
		fp.Browser = "Firefox"
		versionMatch := regexp.MustCompile(`firefox/(\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	} else if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		fp.Browser = "Safari"
		versionMatch := regexp.MustCompile(`version/(\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	} else if strings.Contains(ua, "edg") {
		fp.Browser = "Edge"
		versionMatch := regexp.MustCompile(`edg/(\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	} else if strings.Contains(ua, "opr") || strings.Contains(ua, "opera") {
		fp.Browser = "Opera"
		versionMatch := regexp.MustCompile(`(?:opera|opr)[/ ](\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	} else if strings.Contains(ua, "msie") || strings.Contains(ua, "trident") {
		fp.Browser = "IE"
		versionMatch := regexp.MustCompile(`(?:msie |rv:)(\d+(\.\d+)?)`).FindStringSubmatch(ua)
		if len(versionMatch) > 1 {
			fp.BrowserVersion = versionMatch[1]
		}
	}
}

func (fp *DeviceFingerprint) detectBot(ua string) bool {
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "curl", "wget", "python",
		"headless", "phantom", "selenium", "puppeteer", "playwright",
		"apache-httpclient", "okhttp", "go-http-client",
	}

	uaLower := strings.ToLower(ua)
	for _, pattern := range botPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}

	return false
}

func (fp *DeviceFingerprint) detectDeviceType() {
	ua := strings.ToLower(fp.UserAgent)

	if regexp.MustCompile(`mobile|android|iphone|ipod|blackberry|windows phone`).MatchString(ua) {
		if regexp.MustCompile(`tablet|ipad|nexus (7|9|10)|sm-t|playbook`).MatchString(ua) {
			fp.DeviceType = DeviceTypeTablet
		} else {
			fp.DeviceType = DeviceTypeMobile
		}
	} else {
		fp.DeviceType = DeviceTypeDesktop
	}
}

func (fp *DeviceFingerprint) GenerateHash() string {
	components := []string{
		fp.UserAgent,
		fp.OS,
		fp.Browser,
		strconv.Itoa(fp.ScreenWidth),
		strconv.Itoa(fp.ScreenHeight),
		fp.Platform,
		fp.Timezone,
		fp.Language,
	}

	combined := strings.Join(components, "|")

	hash := 0
	for i, c := range combined {
		hash = hash*31 + int(c) + i
	}

	return strconv.FormatInt(int64(hash), 16)
}

func (e *RiskEngine) AnalyzeDeviceFingerprint(fingerprint *DeviceFingerprint) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if fingerprint == nil {
		return score, factors
	}

	if fingerprint.IsBot {
		score += 50
		factors = append(factors, RiskFactor{
			Name:   "bot_detected",
			Weight: 50,
			Reason: "检测到机器人User-Agent",
		})
	}

	if e.isTorBrowser(fingerprint.UserAgent) {
		score += 20
		factors = append(factors, RiskFactor{
			Name:   "tor_browser",
			Weight: 20,
			Reason: "检测到Tor浏览器",
		})
	}

	if e.isSuspiciousUserAgent(fingerprint.UserAgent) {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "suspicious_ua",
			Weight: 15,
			Reason: "User-Agent存在可疑特征",
		})
	}

	if fingerprint.ScreenWidth == 0 || fingerprint.ScreenHeight == 0 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "missing_screen_info",
			Weight: 10,
			Reason: "缺少屏幕分辨率信息",
		})
	}

	aspectRatio := float64(fingerprint.ScreenWidth) / float64(fingerprint.ScreenHeight+1)
	if aspectRatio < 0.5 || aspectRatio > 3.0 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "abnormal_aspect_ratio",
			Weight: 10,
			Reason: "屏幕宽高比异常",
		})
	}

	if fingerprint.Timezone == "" {
		score += 5
		factors = append(factors, RiskFactor{
			Name:   "missing_timezone",
			Weight: 5,
			Reason: "缺少时区信息",
		})
	}

	if fingerprint.Language == "" && fingerprint.AcceptLanguage == "" {
		score += 5
		factors = append(factors, RiskFactor{
			Name:   "missing_language",
			Weight: 5,
			Reason: "缺少语言设置",
		})
	}

	if fingerprint.Browser == "" || fingerprint.Browser == "unknown" {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "unknown_browser",
			Weight: 15,
			Reason: "无法识别的浏览器",
		})
	}

	return score, factors
}

func (e *RiskEngine) isTorBrowser(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	torIndicators := []string{
		"tor",
		"onion",
		"firefox/60",
		"firefox/78",
	}

	for _, indicator := range torIndicators {
		if strings.Contains(ua, indicator) {
			return true
		}
	}

	return false
}

func (e *RiskEngine) isSuspiciousUserAgent(userAgent string) bool {
	ua := strings.ToLower(userAgent)

	if regexp.MustCompile(`^mozilla/\d+\.\d+\s*$$`).MatchString(userAgent) {
		return true
	}

	if len(userAgent) < 20 {
		return true
	}

	suspiciousPatterns := []string{
		"模拟器",
		"emulator",
		"test",
		"debug",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(ua, pattern) {
			return true
		}
	}

	return false
}

func (e *RiskEngine) CompareFingerprints(fp1, fp2 *DeviceFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0.0
	}

	similarity := 0.0
	totalWeight := 0.0

	uaWeight := 20.0
	if fp1.UserAgent == fp2.UserAgent {
		similarity += uaWeight
	} else if fp1.Browser == fp2.Browser && fp1.OS == fp2.OS {
		similarity += uaWeight * 0.8
	} else if fp1.OS == fp2.OS {
		similarity += uaWeight * 0.5
	}
	totalWeight += uaWeight

	screenWeight := 15.0
	if fp1.ScreenWidth == fp2.ScreenWidth && fp1.ScreenHeight == fp2.ScreenHeight {
		similarity += screenWeight
	} else {
		widthDiff := math.Abs(float64(fp1.ScreenWidth-fp2.ScreenWidth)) / float64(max(fp1.ScreenWidth, fp2.ScreenWidth)+1)
		heightDiff := math.Abs(float64(fp1.ScreenHeight-fp2.ScreenHeight)) / float64(max(fp1.ScreenHeight, fp2.ScreenHeight)+1)
		screenSimilarity := screenWeight * (1.0 - (widthDiff+heightDiff)/2)
		similarity += screenSimilarity
	}
	totalWeight += screenWeight

	platformWeight := 10.0
	if fp1.Platform == fp2.Platform {
		similarity += platformWeight
	} else if fp1.OS == fp2.OS {
		similarity += platformWeight * 0.7
	}
	totalWeight += platformWeight

	timezoneWeight := 10.0
	if fp1.Timezone == fp2.Timezone {
		similarity += timezoneWeight
	}
	totalWeight += timezoneWeight

	languageWeight := 10.0
	if fp1.Language == fp2.Language {
		similarity += languageWeight
	} else if fp1.AcceptLanguage == fp2.AcceptLanguage {
		similarity += languageWeight * 0.8
	}
	totalWeight += languageWeight

	canvasWeight := 15.0
	if fp1.CanvasHash != "" && fp2.CanvasHash != "" {
		if fp1.CanvasHash == fp2.CanvasHash {
			similarity += canvasWeight
		}
	} else {
		similarity += canvasWeight * 0.5
	}
	totalWeight += canvasWeight

	webglWeight := 10.0
	if fp1.WebGLRenderer != "" && fp2.WebGLRenderer != "" {
		if fp1.WebGLRenderer == fp2.WebGLRenderer {
			similarity += webglWeight
		}
	} else {
		similarity += webglWeight * 0.5
	}
	totalWeight += webglWeight

	if totalWeight > 0 {
		return similarity / totalWeight
	}

	return 0.0
}

func (e *RiskEngine) DetectFingerprintAnomaly(current *DeviceFingerprint, historical []*DeviceFingerprint) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(historical) == 0 {
		return score, factors
	}

	suspiciousChanges := 0
	for _, hist := range historical {
		similarity := e.CompareFingerprints(current, hist)

		if current.OS != hist.OS && similarity < 0.5 {
			suspiciousChanges++
		}

		if current.Browser != hist.Browser && similarity < 0.5 {
			suspiciousChanges++
		}

		if math.Abs(float64(current.ScreenWidth-hist.ScreenWidth)) > 100 {
			suspiciousChanges++
		}

		if math.Abs(float64(current.ScreenHeight-hist.ScreenHeight)) > 100 {
			suspiciousChanges++
		}

		if current.Platform != hist.Platform {
			suspiciousChanges++
		}
	}

	if suspiciousChanges > len(historical)/2 && suspiciousChanges >= 2 {
		score += 30
		factors = append(factors, RiskFactor{
			Name:   "device_mismatch",
			Weight: 30,
			Reason: "当前设备与历史设备指纹存在显著差异",
		})
	}

	if current.Browser != "" && current.Browser != "unknown" {
		browserChange := 0
		for _, hist := range historical {
			if hist.Browser != "" && hist.Browser != "unknown" && hist.Browser != current.Browser {
				browserChange++
			}
		}
		if browserChange > len(historical)/3 && browserChange >= 2 {
			score += 20
			factors = append(factors, RiskFactor{
				Name:   "browser_switching",
				Weight: 20,
				Reason: "检测到频繁切换浏览器",
			})
		}
	}

	if current.ScreenWidth < 320 || current.ScreenHeight < 480 {
		if current.ScreenWidth > 0 && current.ScreenHeight > 0 {
			aspectRatio := float64(current.ScreenWidth) / float64(current.ScreenHeight)
			if aspectRatio < 0.3 || aspectRatio > 3.0 {
				score += 15
				factors = append(factors, RiskFactor{
					Name:   "invalid_screen_size",
					Weight: 15,
					Reason: "屏幕尺寸异常，可能是自动化工具",
				})
			}
		}
	}

	return score, factors
}

func (e *RiskEngine) AnalyzeFingerprintTimezone(fingerprint *DeviceFingerprint, localTime time.Time) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if fingerprint.Timezone == "" {
		return score, factors
	}

	detectedZone, err := time.LoadLocation(fingerprint.Timezone)
	if err != nil {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "invalid_timezone",
			Weight: 10,
			Reason: "时区信息无效",
		})
		return score, factors
	}

	currentOffset := localTime.In(detectedZone).Format("-0700")
	_ = currentOffset

	utcOffset := detectedZone.String()
	if strings.Contains(utcOffset, "UTC") || strings.Contains(utcOffset, "GMT") {
		if !strings.Contains(fingerprint.Timezone, "utc") && !strings.Contains(fingerprint.Timezone, "gmt") {
			score += 5
			factors = append(factors, RiskFactor{
				Name:   "timezone_mismatch",
				Weight: 5,
				Reason: "时区设置与系统时区不匹配",
			})
		}
	}

	return score, factors
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
