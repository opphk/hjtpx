package service

import (
	"net/http"
	"regexp"
	"strings"
)

type AutomationDetectionService struct {
	seleniumPatterns   []*regexp.Regexp
	headlessPatterns   []*regexp.Regexp
	phantomJSPatterns  []*regexp.Regexp
	playwrightPatterns []*regexp.Regexp
	puppeteerPatterns  []*regexp.Regexp
	genericBotPatterns []*regexp.Regexp
}

func NewAutomationDetectionService() *AutomationDetectionService {
	return &AutomationDetectionService{
		seleniumPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)selenium`),
			regexp.MustCompile(`(?i)webdriver`),
			regexp.MustCompile(`(?i)Selenium\.prototype`),
		},
		headlessPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless`),
			regexp.MustCompile(`(?i)HeadlessChrome`),
			regexp.MustCompile(`(?i)HeadlessFirefox`),
		},
		phantomJSPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)phantomjs`),
			regexp.MustCompile(`(?i)phantom.js`),
		},
		playwrightPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)playwright`),
			regexp.MustCompile(`(?i)pw\.chromium`),
			regexp.MustCompile(`(?i)pw\.firefox`),
			regexp.MustCompile(`(?i)pw\.webkit`),
		},
		puppeteerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)puppeteer-extra`),
		},
		genericBotPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)bot`),
			regexp.MustCompile(`(?i)crawler`),
			regexp.MustCompile(`(?i)spider`),
		},
	}
}

func (s *AutomationDetectionService) DetectAutomationTool(req *http.Request, frontendData map[string]interface{}) AutomationDetectionResult {
	result := AutomationDetectionResult{
		ToolType:             "",
		IsAutomated:          false,
		Confidence:           0,
		DetectionMethods:     []string{},
		Indicators:           []string{},
		Score:                0,
		BehavioralIndicators: nil,
	}

	userAgent := req.Header.Get("User-Agent")

	if s.detectSelenium(userAgent, frontendData, &result) {
		return result
	}

	if s.detectPhantomJS(userAgent, &result) {
		return result
	}

	if s.detectHeadless(userAgent, frontendData, &result) {
		return result
	}

	if s.detectPlaywright(userAgent, frontendData, &result) {
		return result
	}

	if s.detectPuppeteer(userAgent, frontendData, &result) {
		return result
	}

	return result
}

func (s *AutomationDetectionService) detectSelenium(userAgent string, frontendData map[string]interface{}, result *AutomationDetectionResult) bool {
	evidence := []string{}
	score := 0.0
	methods := []string{}

	for _, pattern := range s.seleniumPatterns {
		if pattern.MatchString(userAgent) {
			evidence = append(evidence, "User-Agent contains selenium pattern: "+pattern.String())
			methods = append(methods, "user_agent_pattern")
			score += 15
		}
	}

	if frontendData != nil {
		if navigator, ok := frontendData["navigator"].(map[string]interface{}); ok {
			if webdriver, ok := navigator["webdriver"].(bool); ok && webdriver {
				evidence = append(evidence, "Navigator.webdriver is true")
				methods = append(methods, "frontend_webdriver_flag")
				score += 25
			}
		}
	}

	if len(evidence) > 0 {
		result.ToolType = ToolSelenium
		result.Indicators = evidence
		result.DetectionMethods = methods
		result.Score = score
		result.Confidence = score / 40.0
		result.IsAutomated = score >= 20
		return true
	}

	return false
}

func (s *AutomationDetectionService) detectPhantomJS(userAgent string, result *AutomationDetectionResult) bool {
	evidence := []string{}
	score := 0.0
	methods := []string{}

	for _, pattern := range s.phantomJSPatterns {
		if pattern.MatchString(userAgent) {
			evidence = append(evidence, "User-Agent contains PhantomJS pattern: "+pattern.String())
			methods = append(methods, "user_agent_pattern")
			score += 35
		}
	}

	if len(evidence) > 0 {
		result.ToolType = ToolPhantomJS
		result.Indicators = evidence
		result.DetectionMethods = methods
		result.Score = score
		result.Confidence = score / 35.0
		result.IsAutomated = score >= 20
		return true
	}

	return false
}

func (s *AutomationDetectionService) detectHeadless(userAgent string, frontendData map[string]interface{}, result *AutomationDetectionResult) bool {
	evidence := []string{}
	score := 0.0
	methods := []string{}

	for _, pattern := range s.headlessPatterns {
		if pattern.MatchString(userAgent) {
			evidence = append(evidence, "User-Agent contains headless pattern: "+pattern.String())
			methods = append(methods, "user_agent_pattern")
			score += 25
		}
	}

	if frontendData != nil {
		if webdriver, ok := frontendData["webdriver"].(bool); ok && webdriver {
			evidence = append(evidence, "Webdriver flag is true")
			methods = append(methods, "frontend_webdriver_flag")
			score += 15
		}
	}

	if len(evidence) > 0 {
		result.ToolType = ToolHeadless
		result.Indicators = evidence
		result.DetectionMethods = methods
		result.Score = score
		result.Confidence = score / 40.0
		result.IsAutomated = score >= 20
		return true
	}

	return false
}

func (s *AutomationDetectionService) detectPlaywright(userAgent string, frontendData map[string]interface{}, result *AutomationDetectionResult) bool {
	evidence := []string{}
	score := 0.0
	methods := []string{}

	for _, pattern := range s.playwrightPatterns {
		if pattern.MatchString(userAgent) {
			evidence = append(evidence, "User-Agent contains playwright pattern: "+pattern.String())
			methods = append(methods, "user_agent_pattern")
			score += 20
		}
	}

	if frontendData != nil {
		if browserName, ok := frontendData["browserName"].(string); ok && strings.Contains(strings.ToLower(browserName), "playwright") {
			evidence = append(evidence, "Browser name contains playwright")
			methods = append(methods, "frontend_browser_name")
			score += 20
		}
	}

	if len(evidence) > 0 {
		result.ToolType = ToolPlaywright
		result.Indicators = evidence
		result.DetectionMethods = methods
		result.Score = score
		result.Confidence = score / 40.0
		result.IsAutomated = score >= 20
		return true
	}

	return false
}

func (s *AutomationDetectionService) detectPuppeteer(userAgent string, frontendData map[string]interface{}, result *AutomationDetectionResult) bool {
	evidence := []string{}
	score := 0.0
	methods := []string{}

	for _, pattern := range s.puppeteerPatterns {
		if pattern.MatchString(userAgent) {
			evidence = append(evidence, "User-Agent contains puppeteer pattern: "+pattern.String())
			methods = append(methods, "user_agent_pattern")
			score += 25
		}
	}

	if frontendData != nil {
		if puppeteer, ok := frontendData["puppeteer"].(bool); ok && puppeteer {
			evidence = append(evidence, "Puppeteer flag is true")
			methods = append(methods, "frontend_puppeteer_flag")
			score += 20
		}
	}

	if len(evidence) > 0 {
		result.ToolType = ToolPuppeteer
		result.Indicators = evidence
		result.DetectionMethods = methods
		result.Score = score
		result.Confidence = score / 45.0
		result.IsAutomated = score >= 20
		return true
	}

	return false
}

func (s *AutomationDetectionService) IsAutomatedRequest(req *http.Request, frontendData map[string]interface{}) bool {
	result := s.DetectAutomationTool(req, frontendData)
	return result.IsAutomated
}

func (s *AutomationDetectionService) GetRiskLevel(score float64) string {
	switch {
	case score >= 40:
		return "high"
	case score >= 20:
		return "medium"
	case score >= 10:
		return "low"
	default:
		return "none"
	}
}
