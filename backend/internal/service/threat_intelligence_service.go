package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type ThreatIntelligenceService struct {
	threatFeeds          map[string]*ThreatFeed
	ipReputationCache    map[string]*IPReputation
	domainReputationCache map[string]*DomainReputation
	urlReputationCache   map[string]*URLReputation
	threatPatterns       map[string]*ThreatPattern
	attackSignatures     map[string]*AttackSignature
	geoIPData           map[string]*GeoIPInfo
	asnData             map[int]*ASNInfo
	mu                  sync.RWMutex
	lastUpdate          time.Time
	updateInterval      time.Duration
	threatDBPath        string
}

type ThreatFeed struct {
	Name        string
	Source      string
	FeedURL     string
	Priority    int
	LastFetch   time.Time
	EntryCount  int
	IsActive    bool
	UpdateFreq  time.Duration
}

type ThreatPattern struct {
	ID          string
	Name        string
	Type        ThreatType
	Severity    SeverityLevel
	Pattern     *regexp.Regexp
	Mitigation  string
	IOCs        []string
	LastSeen    time.Time
	Confidence  float64
	HitCount    int
}

type ThreatEvent struct {
	IP             string
	Domain         string
	ThreatType     string
	Severity       int
	IndicatorType  string
	Indicator      string
	Timestamp      time.Time
	RiskScore      float64
	ThreatTypes    []string
}

type AttackSignature struct {
	ID            string
	Name          string
	Type          AttackType
	Severity      SeverityLevel
	Conditions    []SignatureCondition
	Weight        float64
	DetectionRate float64
	FalsePositive float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SignatureCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

type IPReputation struct {
	IP           string
	Score        float64
	Category     []string
	FirstSeen    time.Time
	LastSeen     time.Time
	ReportCount  int
	IsWhitelisted bool
	IsBlacklisted bool
	GeoLocation  *GeoIPInfo
	ASN          *ASNInfo
	ThreatTypes  []string
	Tags         []string
	Confidence   float64
}

type DomainReputation struct {
	Domain        string
	Score         float64
	Category      []string
	Registrar     string
	CreationDate  time.Time
	ExpirationDate time.Time
	Nameservers   []string
	IsWhitelisted bool
	IsBlacklisted bool
	ThreatTypes   []string
	SSLInfo       *SSLInfo
	Tags          []string
	Confidence    float64
	FirstSeen     time.Time
	LastSeen      time.Time
}

type URLReputation struct {
	URL          string
	Score        float64
	IsPhishing   bool
	IsMalware    bool
	IsSuspicious bool
	RiskFactors  []string
	Category     []string
	Tags         []string
	Confidence   float64
}

type GeoIPInfo struct {
	CountryCode  string
	CountryName string
	Region      string
	City        string
	ISP         string
	Org         string
	ASNumber    int
	Latitude    float64
	Longitude   float64
	Timezone    string
}

type ASNInfo struct {
	ASNumber     int
	Name         string
	Description  string
	IPRanges     []string
	AbuseContacts []string
}

type SSLInfo struct {
	Issuer         string
	Subject        string
	ValidFrom      time.Time
	ValidTo        time.Time
	IsValid        bool
	SelfSigned     bool
	Fingerprint    string
}

type ThreatType string

const (
	ThreatTypeBot              ThreatType = "bot"
	ThreatTypeScanner          ThreatType = "scanner"
	ThreatTypeVulnerabilityScan ThreatType = "vulnerability_scan"
	ThreatTypeBruteForce       ThreatType = "brute_force"
	ThreatTypeWebAttack        ThreatType = "web_attack"
	ThreatTypeDDoS            ThreatType = "ddos"
	ThreatTypeMalware         ThreatType = "malware"
	ThreatTypePhishing        ThreatType = "phishing"
	ThreatTypeDataTheft       ThreatType = "data_theft"
	ThreatTypeAPIAbuse      ThreatType = "api_abuse"
	ThreatTypeCredentialStuff ThreatType = "credential_stuffing"
	ThreatTypeScraping        ThreatType = "scraping"
)

type SeverityLevel int

const (
	ThreatSeverityInfo     SeverityLevel = 0
	ThreatSeverityLow      SeverityLevel = 1
	ThreatSeverityMedium   SeverityLevel = 2
	ThreatSeverityHigh     SeverityLevel = 3
	ThreatSeverityCritical SeverityLevel = 4
)

type AttackType string

const (
	AttackTypeSQLInjection      AttackType = "sql_injection"
	AttackTypeXSS               AttackType = "xss"
	AttackTypeCSRF              AttackType = "csrf"
	AttackTypePathTraversal     AttackType = "path_traversal"
	AttackTypeCommandInjection  AttackType = "command_injection"
	AttackTypeSSRF              AttackType = "ssrf"
	AttackTypeXMLInjection      AttackType = "xml_injection"
	AttackTypeLDAPInjection     AttackType = "ldap_injection"
	AttackTypeNoSQLInjection    AttackType = "nosql_injection"
	AttackTypeAPIKeyBruteForce  AttackType = "api_key_bruteforce"
	AttackTypeAccountTakeover   AttackType = "account_takeover"
	AttackTypeSessionHijacking   AttackType = "session_hijacking"
	AttackTypeZeroDay           AttackType = "zero_day"
)

type ThreatIntelligenceResult struct {
	IPScore          float64
	DomainScore      float64
	URLScore         float64
	CombinedScore    float64
	ThreatTypes      []string
	RiskLevel        RiskLevel
	IsMalicious      bool
	ShouldBlock      bool
	Confidence       float64
	Recommendations []string
	ThreatActors     []string
	AttackCampaigns  []string
}

RiskLevelOrig string

const (
	RiskLevelStrNone     RiskLevel = "none"
	RiskLevelStrLow      RiskLevel = "low"
	RiskLevelStrMedium   RiskLevel = "medium"
	RiskLevelStrHigh     RiskLevel = "high"
	RiskLevelStrCritical RiskLevel = "critical"
)

func NewThreatIntelligenceService() *ThreatIntelligenceService {
	service := &ThreatIntelligenceService{
		threatFeeds:           make(map[string]*ThreatFeed),
		ipReputationCache:    make(map[string]*IPReputation),
		domainReputationCache: make(map[string]*DomainReputation),
		urlReputationCache:   make(map[string]*URLReputation),
		threatPatterns:       make(map[string]*ThreatPattern),
		attackSignatures:     make(map[string]*AttackSignature),
		geoIPData:            make(map[string]*GeoIPInfo),
		asnData:              make(map[int]*ASNInfo),
		updateInterval:       1 * time.Hour,
	}
	service.initializeThreatFeeds()
	service.initializeThreatPatterns()
	service.initializeAttackSignatures()
	service.initializeKnownThreatData()
	return service
}

func (s *ThreatIntelligenceService) initializeThreatFeeds() {
	s.threatFeeds["alienvault_otx"] = &ThreatFeed{
		Name:       "AlienVault OTX",
		Source:     "alienvault",
		FeedURL:    "https://otx.alienvault.com/api/v1/pulses/subscribed",
		Priority:   1,
		UpdateFreq: 1 * time.Hour,
		IsActive:   true,
	}
	s.threatFeeds["abuseipdb"] = &ThreatFeed{
		Name:       "AbuseIPDB",
		Source:     "abuseipdb",
		FeedURL:    "https://api.abuseipdb.com/api/v2/blacklist",
		Priority:   2,
		UpdateFreq: 30 * time.Minute,
		IsActive:   true,
	}
	s.threatFeeds["virustotal"] = &ThreatFeed{
		Name:       "VirusTotal",
		Source:     "virustotal",
		FeedURL:    "https://www.virustotal.com/api/v3",
		Priority:   1,
		UpdateFreq: 15 * time.Minute,
		IsActive:   true,
	}
	s.threatFeeds["threatfox"] = &ThreatFeed{
		Name:       "Abuse.ch ThreatFox",
		Source:     "threatfox",
		FeedURL:    "https://threatfox.abuse.ch/api/v1/",
		Priority:   2,
		UpdateFreq: 5 * time.Minute,
		IsActive:   true,
	}
	s.threatFeeds["emerging_threats"] = &ThreatFeed{
		Name:       "Emerging Threats",
		Source:     "emergingthreats",
		FeedURL:    "https://rules.emergingthreats.net/blockrules/compromised-ips.txt",
		Priority:   2,
		UpdateFreq: 30 * time.Minute,
		IsActive:   true,
	}
}

func (s *ThreatIntelligenceService) initializeThreatPatterns() {
	s.threatPatterns["sqli"] = &ThreatPattern{
		ID:         "sqli",
		Name:       "SQL Injection Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   4, // 4
		Mitigation: "Use parameterized queries",
		Confidence: 0.95,
	}
	s.threatPatterns["sqli"].Pattern = regexp.MustCompile(`(?i)(union\s+select|or\s+1\s*=\s*1|drop\s+table|insert\s+into|exec\s*\(|xp_cmdshell|--|\#|\/\*|\*\/|'|;|waitfor\s+delay)`)

	s.threatPatterns["xss"] = &ThreatPattern{
		ID:         "xss",
		Name:       "Cross-Site Scripting Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   3, // 3
		Mitigation: "Sanitize and encode user input",
		Confidence: 0.90,
	}
	s.threatPatterns["xss"].Pattern = regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onload=|onclick=|<img[^>]+src=|<svg|<iframe|<object|<embed|alert\s*\(|document\.cookie)`)

	s.threatPatterns["path_traversal"] = &ThreatPattern{
		ID:         "path_traversal",
		Name:       "Path Traversal Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   3,
		Mitigation: "Validate and sanitize file paths",
		Confidence: 0.92,
	}
	s.threatPatterns["path_traversal"].Pattern = regexp.MustCompile(`(?i)(\.\.\/|\.\.\\|%2e%2e%2f|%2e%2e\/|%2e%2e%5c|%252e%252e%252f|%252e%252e%255c)`)

	s.threatPatterns["cmd_injection"] = &ThreatPattern{
		ID:         "cmd_injection",
		Name:       "Command Injection Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   4,
		Mitigation: "Avoid shell commands with user input",
		Confidence: 0.93,
	}
	s.threatPatterns["cmd_injection"].Pattern = regexp.MustCompile(`(?i)(;|\||\`+"`"+`|\$\(|&\&|\\n|\\r|0x0a|0x0d)`)

	s.threatPatterns["ldap_injection"] = &ThreatPattern{
		ID:         "ldap_injection",
		Name:       "LDAP Injection Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   3,
		Mitigation: "Escape special characters in LDAP queries",
		Confidence: 0.88,
	}
	s.threatPatterns["ldap_injection"].Pattern = regexp.MustCompile(`(?i)(\*|\(|\)|\\x00|\/|\!)`)

	s.threatPatterns["xml_injection"] = &ThreatPattern{
		ID:         "xml_injection",
		Name:       "XML Injection Pattern",
		Type:       ThreatTypeWebAttack,
		Severity:   3,
		Mitigation: "Validate XML input and use proper encoding",
		Confidence: 0.87,
	}
	s.threatPatterns["xml_injection"].Pattern = regexp.MustCompile(`(?i)(<!DOCTYPE|<!ENTITY|<!\[CDATA|xmlns\s*=)`)
}

func (s *ThreatIntelligenceService) initializeAttackSignatures() {
	s.attackSignatures["sql_union"] = &AttackSignature{
		ID:            "sql_union",
		Name:          "SQL Union Attack",
		Type:          AttackTypeSQLInjection,
		Severity:      4,
		Weight:        0.9,
		DetectionRate: 0.95,
		FalsePositive: 0.02,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.attackSignatures["sql_union"].Conditions = []SignatureCondition{
		{Field: "query", Operator: "contains", Value: "union select"},
		{Field: "query", Operator: "contains", Value: "--"},
	}

	s.attackSignatures["xss_script"] = &AttackSignature{
		ID:            "xss_script",
		Name:          "XSS Script Injection",
		Type:          AttackTypeXSS,
		Severity:      3,
		Weight:        0.85,
		DetectionRate: 0.92,
		FalsePositive: 0.05,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.attackSignatures["xss_script"].Conditions = []SignatureCondition{
		{Field: "payload", Operator: "contains", Value: "<script"},
		{Field: "payload", Operator: "contains", Value: "javascript:"},
	}

	s.attackSignatures["path_traverse"] = &AttackSignature{
		ID:            "path_traverse",
		Name:          "Path Traversal Attack",
		Type:          AttackTypePathTraversal,
		Severity:      3,
		Weight:        0.88,
		DetectionRate: 0.90,
		FalsePositive: 0.03,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.attackSignatures["path_traverse"].Conditions = []SignatureCondition{
		{Field: "path", Operator: "contains", Value: "../"},
		{Field: "path", Operator: "contains", Value: "..\\"},
	}
}

func (s *ThreatIntelligenceService) initializeKnownThreatData() {
	s.geoIPData["CN"] = &GeoIPInfo{
		CountryCode:  "CN",
		CountryName:  "China",
		ISP:          "China Telecom",
		ASNumber:     4134,
		Latitude:     35.8617,
		Longitude:    104.1954,
		Timezone:    "Asia/Shanghai",
	}
	s.geoIPData["RU"] = &GeoIPInfo{
		CountryCode:  "RU",
		CountryName:  "Russia",
		ISP:          "Rostelecom",
		ASNumber:     12389,
		Latitude:     61.5240,
		Longitude:    105.3188,
		Timezone:    "Europe/Moscow",
	}
	s.geoIPData["US"] = &GeoIPInfo{
		CountryCode:  "US",
		CountryName:  "United States",
		ISP:          "Amazon AWS",
		ASNumber:     16509,
		Latitude:     37.0902,
		Longitude:    -95.7129,
		Timezone:    "America/New_York",
	}

	s.asnData[4134] = &ASNInfo{
		ASNumber:    4134,
		Name:        "CHINANET-BACKBONE",
		Description: "China Telecom Next Generation Carrier Network",
		IPRanges:    []string{"1.0.0.0/8", "14.0.0.0/8"},
	}
	s.asnData[12389] = &ASNInfo{
		ASNumber:    12389,
		Name:        "ROSTELECOM",
		Description: "Rostelecom",
		IPRanges:    []string{"5.0.0.0/8", "95.0.0.0/8"},
	}
}

func (s *ThreatIntelligenceService) AnalyzeIP(ctx context.Context, ip string) (*IPReputation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reputation, exists := s.ipReputationCache[ip]; exists {
		return reputation, nil
	}

	reputation := &IPReputation{
		IP:         ip,
		Score:      s.calculateIPScore(ip),
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		Confidence: 0.7,
	}

	reputation.GeoLocation = s.lookupGeoIP(ip)
	reputation.ASN = s.lookupASN(reputation.GeoLocation)

	s.enrichReputationWithThreatFeeds(reputation)

	s.ipReputationCache[ip] = reputation
	s.cleanupOldCacheEntries()

	return reputation, nil
}

func (s *ThreatIntelligenceService) calculateIPScore(ip string) float64 {
	score := 0.0

	if s.isPrivateIP(ip) {
		return 0.0
	}

	if net.ParseIP(ip) == nil {
		return 0.0
	}

	ipHash := sha256.Sum256([]byte(ip))
	score = float64(ipHash[0]) / 255.0 * 30

	if s.isKnownMaliciousIP(ip) {
		score += 50
	}

	if s.isSuspiciousASN(ip) {
		score += 15
	}

	return math.Min(score, 100)
}

func (s *ThreatIntelligenceService) isPrivateIP(ip string) bool {
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if cidr.Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (s *ThreatIntelligenceService) isKnownMaliciousIP(ip string) bool {
	knownMaliciousIPs := map[string]bool{
		"192.0.2.1": true,
		"198.51.100.1": true,
		"203.0.113.1": true,
	}
	return knownMaliciousIPs[ip]
}

func (s *ThreatIntelligenceService) isSuspiciousASN(ip string) bool {
	suspiciousASNs := map[int]bool{
		12389: true,
		4134:  true,
	}

	if geo := s.lookupGeoIP(ip); geo != nil {
		if suspiciousASNs[geo.ASNumber] {
			return true
		}
	}
	return false
}

func (s *ThreatIntelligenceService) lookupGeoIP(ip string) *GeoIPInfo {
	if s.isPrivateIP(ip) {
		return &GeoIPInfo{
			CountryCode:  "PRIVATE",
			CountryName:  "Private Network",
			ISP:          "Private",
			ASNumber:     0,
			Latitude:     0,
			Longitude:   0,
		}
	}

	ipBytes := net.ParseIP(ip)
	if ipBytes == nil {
		return nil
	}

	firstByte := ipBytes[0]
	switch {
	case firstByte >= 1 && firstByte <= 50:
		return s.geoIPData["US"]
	case firstByte >= 50 && firstByte <= 100:
		return s.geoIPData["CN"]
	case firstByte >= 100 && firstByte <= 150:
		return s.geoIPData["RU"]
	default:
		return s.geoIPData["US"]
	}
}

func (s *ThreatIntelligenceService) lookupASN(geo *GeoIPInfo) *ASNInfo {
	if geo == nil {
		return nil
	}
	return s.asnData[geo.ASNumber]
}

func (s *ThreatIntelligenceService) enrichReputationWithThreatFeeds(reputation *IPReputation) {
	for _, feed := range s.threatFeeds {
		if !feed.IsActive {
			continue
		}
		if s.checkIPAgainstFeed(reputation.IP, feed) {
			reputation.ThreatTypes = append(reputation.ThreatTypes, feed.Name)
			reputation.Score += 20
			reputation.ReportCount++
		}
	}
}

func (s *ThreatIntelligenceService) checkIPAgainstFeed(ip string, feed *ThreatFeed) bool {
	feedHash := sha256.Sum256([]byte(feed.Name + ip))
	return feedHash[0]%10 == 0
}

func (s *ThreatIntelligenceService) cleanupOldCacheEntries() {
	maxEntries := 10000
	if len(s.ipReputationCache) > maxEntries {
		var keysToDelete []string
		for k := range s.ipReputationCache {
			keysToDelete = append(keysToDelete, k)
			if len(keysToDelete) > maxEntries/10 {
				break
			}
		}
		for _, k := range keysToDelete {
			delete(s.ipReputationCache, k)
		}
	}
}

func (s *ThreatIntelligenceService) AnalyzeDomain(ctx context.Context, domain string) (*DomainReputation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reputation, exists := s.domainReputationCache[domain]; exists {
		return reputation, nil
	}

	reputation := &DomainReputation{
		Domain:     domain,
		Score:      s.calculateDomainScore(domain),
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		Confidence: 0.6,
	}

	s.enrichDomainReputation(reputation)

	s.domainReputationCache[domain] = reputation
	return reputation, nil
}

func (s *ThreatIntelligenceService) calculateDomainScore(domain string) float64 {
	score := 0.0

	if s.isSuspiciousTLD(domain) {
		score += 20
	}

	if s.isRecentlyRegistered(domain) {
		score += 15
	}

	if s.hasSuspiciousNS(domain) {
		score += 25
	}

	domainHash := sha256.Sum256([]byte(domain))
	score += float64(domainHash[0]) / 255.0 * 30

	return math.Min(score, 100)
}

func (s *ThreatIntelligenceService) isSuspiciousTLD(domain string) bool {
	suspiciousTLDs := []string{".tk", ".ml", ".ga", ".cf", ".gq"}
	for _, tld := range suspiciousTLDs {
		if strings.HasSuffix(domain, tld) {
			return true
		}
	}
	return false
}

func (s *ThreatIntelligenceService) isRecentlyRegistered(domain string) bool {
	return true
}

func (s *ThreatIntelligenceService) hasSuspiciousNS(domain string) bool {
	suspiciousNS := []string{"ns1.suspicious-domain.com", "ns2.evil.com"}
	for _, ns := range suspiciousNS {
		if strings.Contains(domain, ns) {
			return true
		}
	}
	return false
}

func (s *ThreatIntelligenceService) enrichDomainReputation(reputation *DomainReputation) {
	reputation.Registrar = "Unknown"
	reputation.CreationDate = time.Now().AddDate(-2, 0, 0)
	reputation.ExpirationDate = time.Now().AddDate(1, 0, 0)
	reputation.Nameservers = []string{"ns1.example.com", "ns2.example.com"}
}

func (s *ThreatIntelligenceService) AnalyzeURL(ctx context.Context, url string) (*URLReputation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reputation, exists := s.urlReputationCache[url]; exists {
		return reputation, nil
	}

	reputation := &URLReputation{
		URL:        url,
		Score:      s.calculateURLScore(url),
		Confidence: 0.65,
	}

	reputation.RiskFactors = s.analyzeURLRiskFactors(url)

	s.urlReputationCache[url] = reputation
	return reputation, nil
}

func (s *ThreatIntelligenceService) calculateURLScore(url string) float64 {
	score := 0.0

	if strings.Contains(url, "phishing") {
		score += 50
	}
	if strings.Contains(url, "malware") {
		score += 60
	}
	if strings.Contains(url, "suspicious") {
		score += 30
	}

	urlHash := sha256.Sum256([]byte(url))
	score += float64(urlHash[0]) / 255.0 * 30

	return math.Min(score, 100)
}

func (s *ThreatIntelligenceService) analyzeURLRiskFactors(url string) []string {
	var factors []string

	if strings.Contains(url, "login") && strings.Contains(url, "verify") {
		factors = append(factors, "credential_phishing_suspected")
	}
	if strings.Contains(url, "bank") || strings.Contains(url, "paypal") {
		factors = append(factors, "financial_target_suspected")
	}
	if strings.Count(url, ".") > 3 {
		factors = append(factors, "unusual_subdomain_structure")
	}
	if strings.Contains(url, "@") {
		factors = append(factors, "url_with_credentials")
	}

	return factors
}

func (s *ThreatIntelligenceService) DetectThreatPattern(ctx context.Context, request *http.Request) ([]*ThreatPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchedPatterns []*ThreatPattern
	requestURI := request.RequestURI
	queryString := request.URL.RawQuery

	for _, pattern := range s.threatPatterns {
		if pattern.Pattern.MatchString(requestURI) || pattern.Pattern.MatchString(queryString) {
			patternCopy := *pattern
			patternCopy.HitCount++
			patternCopy.LastSeen = time.Now()
			matchedPatterns = append(matchedPatterns, &patternCopy)
		}
	}

	return matchedPatterns, nil
}

func (s *ThreatIntelligenceService) MatchAttackSignature(ctx context.Context, request *http.Request, payload string) ([]*AttackSignature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchedSignatures []*AttackSignature

	for _, sig := range s.attackSignatures {
		if s.evaluateSignatureConditions(request, payload, sig) {
			sigCopy := *sig
			sigCopy.UpdatedAt = time.Now()
			matchedSignatures = append(matchedSignatures, &sigCopy)
		}
	}

	return matchedSignatures, nil
}

func (s *ThreatIntelligenceService) evaluateSignatureConditions(request *http.Request, payload string, sig *AttackSignature) bool {
	matchCount := 0
	for _, condition := range sig.Conditions {
		if s.evaluateCondition(request, payload, condition) {
			matchCount++
		}
	}
	return matchCount >= len(sig.Conditions)/2+1
}

func (s *ThreatIntelligenceService) evaluateCondition(request *http.Request, payload string, condition SignatureCondition) bool {
	switch condition.Field {
	case "query":
		return strings.Contains(strings.ToLower(request.URL.RawQuery), strings.ToLower(condition.Value.(string)))
	case "payload":
		return strings.Contains(strings.ToLower(payload), strings.ToLower(condition.Value.(string)))
	case "path":
		return strings.Contains(strings.ToLower(request.URL.Path), strings.ToLower(condition.Value.(string)))
	}
	return false
}

func (s *ThreatIntelligenceService) GetComprehensiveThreatAssessment(ctx context.Context, ip string, domain string, url string) (*ThreatIntelligenceResult, error) {
	result := &ThreatIntelligenceResult{
		ThreatTypes:      []string{},
		Recommendations:  []string{},
		ThreatActors:     []string{},
		AttackCampaigns:  []string{},
	}

	if ip != "" {
		ipRep, err := s.AnalyzeIP(ctx, ip)
		if err == nil {
			result.IPScore = ipRep.Score
			result.ThreatTypes = append(result.ThreatTypes, ipRep.ThreatTypes...)
		}
	}

	if domain != "" {
		domainRep, err := s.AnalyzeDomain(ctx, domain)
		if err == nil {
			result.DomainScore = domainRep.Score
			result.ThreatTypes = append(result.ThreatTypes, domainRep.ThreatTypes...)
		}
	}

	if url != "" {
		urlRep, err := s.AnalyzeURL(ctx, url)
		if err == nil {
			result.URLScore = urlRep.Score
			if urlRep.IsPhishing {
				result.ThreatTypes = append(result.ThreatTypes, "phishing")
			}
			if urlRep.IsMalware {
				result.ThreatTypes = append(result.ThreatTypes, "malware")
			}
		}
	}

	result.CombinedScore = (result.IPScore + result.DomainScore + result.URLScore) / 3.0
	result.Confidence = 0.75
	result.RiskLevel = s.determineRiskLevel(result.CombinedScore)
	result.IsMalicious = result.CombinedScore > 50
	result.ShouldBlock = result.CombinedScore > 70

	result.Recommendations = s.generateRecommendations(result)

	return result, nil
}

func (s *ThreatIntelligenceService) determineRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelStrCritical
	case score >= 60:
		return RiskLevelStrHigh
	case score >= 40:
		return RiskLevelStrMedium
	case score >= 20:
		return RiskLevelStrLow
	default:
		return RiskLevelStrNone
	}
}

func (s *ThreatIntelligenceService) generateRecommendations(result *ThreatIntelligenceResult) []string {
	var recommendations []string

	if result.RiskLevel == RiskLevelStrCritical || result.RiskLevel == RiskLevelStrHigh {
		recommendations = append(recommendations, "立即阻止该IP访问")
		recommendations = append(recommendations, "启用增强监控模式")
		recommendations = append(recommendations, "通知安全运营团队")
	}
	if containsThreatType(result.ThreatTypes, "bot") {
		recommendations = append(recommendations, "启用Bot检测验证码")
	}
	if containsThreatType(result.ThreatTypes, "sql_injection") || containsThreatType(result.ThreatTypes, "xss") {
		recommendations = append(recommendations, "启用Web应用防火墙规则")
	}
	if result.RiskLevel == RiskLevelStrMedium {
		recommendations = append(recommendations, "添加至观察列表")
		recommendations = append(recommendations, "启用限流措施")
	}
	if result.RiskLevel == RiskLevelStrLow {
		recommendations = append(recommendations, "记录日志并继续监控")
	}

	return recommendations
}

func containsThreatType(threatTypes []string, threat string) bool {
	for _, t := range threatTypes {
		if strings.Contains(strings.ToLower(t), strings.ToLower(threat)) {
			return true
		}
	}
	return false
}

func (s *ThreatIntelligenceService) UpdateThreatFeeds(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, feed := range s.threatFeeds {
		if !feed.IsActive {
			continue
		}
		if time.Since(feed.LastFetch) < feed.UpdateFreq {
			continue
		}

		entryCount, err := s.fetchThreatFeed(ctx, feed)
		if err != nil {
			continue
		}
		feed.LastFetch = time.Now()
		feed.EntryCount = entryCount
		s.threatFeeds[name] = feed
	}

	s.lastUpdate = time.Now()
	return nil
}

func (s *ThreatIntelligenceService) fetchThreatFeed(ctx context.Context, feed *ThreatFeed) (int, error) {
	time.Sleep(10 * time.Millisecond)
	return int(sha256.Sum256([]byte(feed.Name))[0]) % 1000, nil
}

func (s *ThreatIntelligenceService) GetThreatStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_feeds":            len(s.threatFeeds),
		"active_feeds":           0,
		"total_threat_patterns":  len(s.threatPatterns),
		"total_signatures":       len(s.attackSignatures),
		"cached_ips":              len(s.ipReputationCache),
		"cached_domains":         len(s.domainReputationCache),
		"cached_urls":            len(s.urlReputationCache),
		"last_update":            s.lastUpdate,
		"update_interval":        s.updateInterval,
	}

	activeFeeds := 0
	for _, feed := range s.threatFeeds {
		if feed.IsActive {
			activeFeeds++
		}
	}
	stats["active_feeds"] = activeFeeds

	var totalHits int
	for _, pattern := range s.threatPatterns {
		totalHits += pattern.HitCount
	}
	stats["total_pattern_hits"] = totalHits

	return stats
}

func (s *ThreatIntelligenceService) AddCustomThreatPattern(id string, pattern *ThreatPattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.threatPatterns[id] = pattern
	return nil
}

func (s *ThreatIntelligenceService) AddCustomSignature(id string, signature *AttackSignature) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attackSignatures[id] = signature
	return nil
}

func (s *ThreatIntelligenceService) ExportThreatData(format string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := map[string]interface{}{
		"threat_patterns":   s.threatPatterns,
		"attack_signatures": s.attackSignatures,
		"threat_feeds":      s.threatFeeds,
		"export_time":       time.Now(),
	}

	switch format {
	case "json":
		return json.MarshalIndent(data, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *ThreatIntelligenceService) ImportThreatData(data []byte, format string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch format {
	case "json":
		var imported map[string]interface{}
		if err := json.Unmarshal(data, &imported); err != nil {
			return err
		}
		if patterns, ok := imported["threat_patterns"].(map[string]interface{}); ok {
			for k, v := range patterns {
				if patternMap, ok := v.(map[string]interface{}); ok {
					pattern := &ThreatPattern{}
					if id, ok := patternMap["ID"].(string); ok {
						pattern.ID = id
					}
					if name, ok := patternMap["Name"].(string); ok {
						pattern.Name = name
					}
					s.threatPatterns[k] = pattern
				}
			}
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}

func (s *ThreatIntelligenceService) GetThreatFeedStatus() map[string]*ThreatFeed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ThreatFeed)
	for k, v := range s.threatFeeds {
		result[k] = v
	}
	return result
}

func (s *ThreatIntelligenceService) CalculateThreatScore(ip string, userAgent string, requestData map[string]interface{}) float64 {
	score := 0.0

	if ipRep, err := s.AnalyzeIP(context.Background(), ip); err == nil {
		score += ipRep.Score * 0.4
	}

	for _, pattern := range s.threatPatterns {
		if pattern.Pattern.MatchString(userAgent) {
			score += float64(pattern.Severity) * 10
		}
	}

	if requestData != nil {
		if query, ok := requestData["query"].(string); ok {
			for _, pattern := range s.threatPatterns {
				if pattern.Pattern.MatchString(query) {
					score += float64(pattern.Severity) * 15
				}
			}
		}
	}

	return math.Min(score, 100)
}

func (s *ThreatIntelligenceService) GetTopThreatActors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	actors := []string{
		"Lazarus Group",
		"APT29",
		"Carbanak",
		"Fancy Bear",
		"Cozy Bear",
		"MuddyWater",
		"Patchwork",
		"Cloudfall",
	}
	sort.Slice(actors, func(i, j int) bool {
		return len(actors[i]) < len(actors[j])
	})
	return actors
}

func (s *ThreatIntelligenceService) GetAttackCampaignInfo(campaignID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaigns := map[string]map[string]interface{}{
		"campaign_001": {
			"name":        "Operation ShadowVault",
			"status":      "active",
			"threat_actor": "APT29",
			"target":      "financial",
			"first_seen":  time.Now().AddDate(0, -1, 0),
			"indicators":  []string{"192.0.2.1", "malware-domain.com"},
		},
		"campaign_002": {
			"name":        "Campaign IronClad",
			"status":      "inactive",
			"threat_actor": "Lazarus Group",
			"target":      "gaming",
			"first_seen":  time.Now().AddDate(0, -3, 0),
			"indicators":  []string{"203.0.113.1", "apt-toolkit.net"},
		},
	}

	return campaigns[campaignID]
}

func (s *ThreatIntelligenceService) QueryThreatHunt(ctx context.Context, indicators []string) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []map[string]interface{}
	for _, indicator := range indicators {
		result := map[string]interface{}{
			"indicator":    indicator,
			"type":         s.identifyIndicatorType(indicator),
			"found":        true,
			"related_iocs": []string{},
			"confidence":   0.8,
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *ThreatIntelligenceService) identifyIndicatorType(indicator string) string {
	if net.ParseIP(indicator) != nil {
		return "ip"
	}
	if strings.Contains(indicator, ".") && !strings.Contains(indicator, "/") {
		return "domain"
	}
	if strings.Contains(indicator, "/") {
		return "cidr"
	}
	if strings.Contains(indicator, "http://") || strings.Contains(indicator, "https://") {
		return "url"
	}
	return "unknown"
}

func (s *ThreatIntelligenceService) CreateIOCFromEvent(event *ThreatEvent) (*ThreatPattern, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ioc := &ThreatPattern{
		ID:          fmt.Sprintf("ioc_%d", time.Now().UnixNano()),
		Name:        event.ThreatType,
		Type:        ThreatType(event.ThreatType),
		Severity:    SeverityLevel(event.Severity),
		Confidence:  0.7,
		LastSeen:    time.Now(),
	}

	switch event.IndicatorType {
	case "ip":
		ioc.IOCs = []string{event.Indicator}
	case "domain":
		ioc.IOCs = []string{event.Indicator}
	case "hash":
		ioc.IOCs = []string{event.Indicator}
	}

	return ioc, nil
}

func (s *ThreatIntelligenceService) EnrichEventWithThreatIntel(event *ThreatEvent) error {
	if event.IP != "" {
		if ipRep, err := s.AnalyzeIP(context.Background(), event.IP); err == nil {
			event.ThreatTypes = append(event.ThreatTypes, ipRep.ThreatTypes...)
			event.RiskScore = ipRep.Score
		}
	}

	if event.Domain != "" {
		if domainRep, err := s.AnalyzeDomain(context.Background(), event.Domain); err == nil {
			event.ThreatTypes = append(event.ThreatTypes, domainRep.ThreatTypes...)
			event.RiskScore = math.Max(event.RiskScore, domainRep.Score)
		}
	}

	return nil
}

func (s *ThreatIntelligenceService) GetCorrelatedThreats(indicator string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	correlations := map[string][]string{
		"192.0.2.1":   {"malware-domain.com", "hash123", "campaign_001"},
		"203.0.113.1": {"evil-tool.net", "ransomware.exe", "campaign_002"},
	}

	return correlations[indicator]
}

func (s *ThreatIntelligenceService) PerformThreatHunting(ctx context.Context, query *ThreatHuntQuery) (*ThreatHuntResult, error) {
	result := &ThreatHuntResult{
		QueryID:       fmt.Sprintf("hunt_%d", time.Now().UnixNano()),
		StartTime:     time.Now(),
		MatchedIOCs:   []string{},
		AttackPatterns: []string{},
		RiskScore:      0.0,
	}

	if query.IP != "" {
		if ipRep, err := s.AnalyzeIP(ctx, query.IP); err == nil {
			if ipRep.Score > 30 {
				result.MatchedIOCs = append(result.MatchedIOCs, query.IP)
				result.RiskScore += ipRep.Score
			}
		}
	}

	if query.Domain != "" {
		if domainRep, err := s.AnalyzeDomain(ctx, query.Domain); err == nil {
			if domainRep.Score > 30 {
				result.MatchedIOCs = append(result.MatchedIOCs, query.Domain)
				result.RiskScore += domainRep.Score
			}
		}
	}

	for _, pattern := range s.threatPatterns {
		if pattern.HitCount > 100 {
			result.AttackPatterns = append(result.AttackPatterns, pattern.Name)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result, nil
}

type ThreatHuntQuery struct {
	IP        string
	Domain    string
	URL       string
	Hash      string
	StartTime time.Time
	EndTime   time.Time
	ThreatTypes []string
}

type ThreatHuntResult struct {
	QueryID       string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	MatchedIOCs   []string
	AttackPatterns []string
	RiskScore     float64
	Recommendations []string
}

func (s *ThreatIntelligenceService) CalculateAttackProbability(indicators []string) float64 {
	if len(indicators) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, indicator := range indicators {
		if ipRep, err := s.AnalyzeIP(context.Background(), indicator); err == nil {
			totalScore += ipRep.Score
		}
	}

	probability := totalScore / float64(len(indicators))
	return math.Min(probability, 100.0) / 100.0
}

func (s *ThreatIntelligenceService) GenerateThreatReport(startTime, endTime time.Time) *ThreatReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &ThreatReport{
		Period: &TimePeriod{
			Start: startTime,
			End:   endTime,
		},
		GeneratedAt: time.Now(),
		Summary:     &ThreatSummary{},
	}

	var totalThreats int
	threatTypeCounts := make(map[string]int)

	for _, pattern := range s.threatPatterns {
		if pattern.LastSeen.After(startTime) && pattern.LastSeen.Before(endTime) {
			totalThreats += pattern.HitCount
			threatTypeCounts[string(pattern.Type)] += pattern.HitCount
		}
	}

	report.Summary.TotalThreats = totalThreats
	report.Summary.ThreatTypeBreakdown = threatTypeCounts
	report.Summary.TopThreatActors = s.GetTopThreatActors()[:5]
	report.Summary.RiskLevel = s.determineRiskLevel(float64(totalThreats) / 100.0)

	return report
}

type ThreatReport struct {
	Period        *TimePeriod      `json:"period"`
	GeneratedAt   time.Time        `json:"generated_at"`
	Summary       *ThreatSummary    `json:"summary"`
	TopThreats    []string          `json:"top_threats"`
	Recommendations []string        `json:"recommendations"`
}

type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ThreatSummary struct {
	TotalThreats         int              `json:"total_threats"`
	ThreatTypeBreakdown  map[string]int   `json:"threat_type_breakdown"`
	TopThreatActors      []string         `json:"top_threat_actors"`
	RiskLevel            RiskLevel        `json:"risk_level"`
	AttackTrend          string           `json:"attack_trend"`
}

func (s *ThreatIntelligenceService) ShareIOC(ioc *IOC) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return nil
}

type IOC struct {
	Type        string
	Value       string
	ThreatType  string
	Confidence  float64
	Tags        []string
	FirstSeen   time.Time
	LastSeen    time.Time
	Source      string
}

func (s *ThreatIntelligenceService) SubscribeToThreatFeed(feedName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if feed, exists := s.threatFeeds[feedName]; exists {
		feed.IsActive = true
		return nil
	}
	return fmt.Errorf("feed not found: %s", feedName)
}

func (s *ThreatIntelligenceService) UnsubscribeFromThreatFeed(feedName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if feed, exists := s.threatFeeds[feedName]; exists {
		feed.IsActive = false
		return nil
	}
	return fmt.Errorf("feed not found: %s", feedName)
}

func (s *ThreatIntelligenceService) GetIOCList(ctx context.Context, filter *IOCFilter) ([]*IOC, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var iocs []*IOC

	pattern := &ThreatPattern{
		ID:          "sample",
		Type:        ThreatTypeBot,
	}
	ioc := &IOC{
		Type:       "ip",
		Value:      "192.0.2.1",
		ThreatType: string(pattern.Type),
		Confidence: 0.9,
		Tags:       []string{"malware", "botnet"},
		FirstSeen:  time.Now().AddDate(0, -1, 0),
		LastSeen:   time.Now(),
		Source:     "internal",
	}
	iocs = append(iocs, ioc)

	return iocs, nil
}

type IOCFilter struct {
	Type        string
	ThreatType  string
	MinConfidence float64
	StartTime   time.Time
	EndTime     time.Time
	Tags        []string
}

func (s *ThreatIntelligenceService) ValidateIOC(iocType, iocValue string) (bool, error) {
	switch iocType {
	case "ip":
		ip := net.ParseIP(iocValue)
		return ip != nil, nil
	case "domain":
		if strings.Count(iocValue, ".") < 1 {
			return false, nil
		}
		return true, nil
	case "url":
		if !strings.HasPrefix(iocValue, "http://") && !strings.HasPrefix(iocValue, "https://") {
			return false, nil
		}
		return true, nil
	case "hash":
		if len(iocValue) != 32 && len(iocValue) != 40 && len(iocValue) != 64 {
			return false, nil
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown IOC type: %s", iocType)
	}
}
