package service

import (
	"strings"
	"time"
)

type FingerprintRequest struct {
	UserAgent string
	IPAddress string
	Headers   map[string]string
}

type FingerprintFeatures struct {
	Webdriver bool
	Headless  bool
	AutomationFramework bool
	VPN       bool
	Proxy     bool
	Tor       bool
	VM        bool
}

type FingerprintResult struct {
	Features   *FingerprintFeatures
	IsBot      bool
	Confidence float64
	Fingerprint string
	ThreatLevel float64
	Recommendations []string
}

type BotFingerprintV2 struct {
	baseService *FingerprintService
}

func NewBotFingerprintV2() *BotFingerprintV2 {
	return &BotFingerprintV2{
		baseService: NewFingerprintService(),
	}
}

func (bf *BotFingerprintV2) AnalyzeRequest(req *FingerprintRequest) *FingerprintResult {
	result := &FingerprintResult{
		Features:        &FingerprintFeatures{},
		IsBot:           false,
		Confidence:      0.0,
		Fingerprint:     "",
		ThreatLevel:     0.0,
		Recommendations: []string{},
	}

	features := result.Features

	if req.Headers != nil {
		if _, exists := req.Headers["webdriver"]; exists {
			features.Webdriver = true
			result.Confidence += 0.6
		}

		if ua := strings.ToLower(req.UserAgent); ua != "" {
			if strings.Contains(ua, "headlesschrome") || 
			   strings.Contains(ua, "headless") ||
			   strings.Contains(ua, "phantom") {
				features.Headless = true
				result.Confidence += 0.3
			}

			if strings.Contains(ua, "selenium") ||
			   strings.Contains(ua, "puppeteer") ||
			   strings.Contains(ua, "playwright") {
				features.AutomationFramework = true
				result.Confidence += 0.5
			}

			if strings.Contains(ua, "vmware") ||
			   strings.Contains(ua, "virtualbox") ||
			   strings.Contains(ua, "parallels") ||
			   strings.Contains(ua, "qemu") {
				features.VM = true
				result.Confidence += 0.2
			}
		}

		if _, exists := req.Headers["x-forwarded-for"]; exists {
			result.Confidence += 0.1
		}
		if _, exists := req.Headers["via"]; exists {
			if strings.Contains(strings.ToLower(req.Headers["via"]), "tor") {
				features.Tor = true
				result.Confidence += 0.3
			}
			features.Proxy = true
			result.Confidence += 0.2
		}
	}

	if result.Confidence > 0.7 {
		result.IsBot = true
		result.ThreatLevel = result.Confidence
		result.Recommendations = append(result.Recommendations, "High bot probability detected")
	}

	if features.Webdriver {
		result.Recommendations = append(result.Recommendations, "WebDriver automation detected")
	}
	if features.Headless {
		result.Recommendations = append(result.Recommendations, "Headless browser detected")
	}
	if features.AutomationFramework {
		result.Recommendations = append(result.Recommendations, "Automation framework detected")
	}

	fingerprint := bf.generateFingerprint(req)
	result.Fingerprint = fingerprint

	return result
}

func (bf *BotFingerprintV2) generateFingerprint(req *FingerprintRequest) string {
	data := req.UserAgent + req.IPAddress + time.Now().Format(time.RFC3339Nano)
	
	hash := 0
	for i := 0; i < len(data); i++ {
		hash = ((hash << 5) - hash) + int(data[i])
		hash = hash & hash
	}

	fingerprint := ""
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < 32; i++ {
		fingerprint += string(chars[((hash+i)*31+i*17)%len(chars)])
	}

	return fingerprint
}
