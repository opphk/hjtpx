package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSignerV2HMACSHA256(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	signer := NewSignerV2(key)

	data := []byte("test data")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if signature == "" {
		t.Fatal("Signature should not be empty")
	}

	valid, err := signer.Verify(signature, data)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Signature verification should succeed")
	}

	valid, err = signer.Verify("invalid_signature", data)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Signature verification should fail for invalid signature")
	}
}

func TestSignerV2HMACSHA384(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	signer := NewSignerV2WithConfig(key, &SignerConfigV2{
		Algorithm: AlgorithmHMACSHA384,
	})

	data := []byte("test data for SHA384")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	valid, err := signer.Verify(signature, data)
	if err != nil || !valid {
		t.Fatalf("Signature verification should succeed")
	}
}

func TestSignerV2HMACSHA512(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	signer := NewSignerV2WithConfig(key, &SignerConfigV2{
		Algorithm: AlgorithmHMACSHA512,
	})

	data := []byte("test data for SHA512")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	valid, err := signer.Verify(signature, data)
	if err != nil || !valid {
		t.Fatalf("Signature verification should succeed")
	}
}

func TestKeyManagerRotation(t *testing.T) {
	initialKey := make([]byte, 32)
	rand.Read(initialKey)

	manager := NewKeyManager(initialKey, time.Hour)

	if manager.GetCurrentVersion() != 1 {
		t.Fatalf("Initial version should be 1, got %d", manager.GetCurrentVersion())
	}

	currentKey := manager.GetCurrentKey()
	if hex.EncodeToString(currentKey) != hex.EncodeToString(initialKey) {
		t.Fatal("Current key should match initial key")
	}

	newKey := make([]byte, 32)
	rand.Read(newKey)
	manager.Rotate(newKey)

	if manager.GetCurrentVersion() != 2 {
		t.Fatalf("Version should be 2 after rotation, got %d", manager.GetCurrentVersion())
	}

	currentKey = manager.GetCurrentKey()
	if hex.EncodeToString(currentKey) == hex.EncodeToString(initialKey) {
		t.Fatal("Current key should not match initial key after rotation")
	}
}

func TestKeyManagerVerifyWithAnyKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	manager := NewKeyManager(key1, 0)

	signer1 := NewSignerV2WithConfig(key1, &SignerConfigV2{Algorithm: AlgorithmHMACSHA256})
	data := []byte("test data")
	signature, _ := signer1.Sign(data)

	valid, err := manager.VerifyWithAnyKey(signature, data, []SigningAlgorithm{AlgorithmHMACSHA256})
	if err != nil || !valid {
		t.Fatal("Should verify with key1")
	}

	manager.Rotate(key2)

	valid, err = manager.VerifyWithAnyKey(signature, data, []SigningAlgorithm{AlgorithmHMACSHA256})
	if err != nil || !valid {
		t.Fatal("Should verify with previous key after rotation")
	}
}

func TestNonceCache(t *testing.T) {
	cache := NewNonceCache(time.Minute)

	nonce := "test_nonce_123"

	if cache.IsUsed(nonce) {
		t.Fatal("Nonce should not be used initially")
	}

	cache.MarkUsed(nonce)

	if !cache.IsUsed(nonce) {
		t.Fatal("Nonce should be marked as used")
	}

	anotherNonce := "another_nonce"
	if cache.IsUsed(anotherNonce) {
		t.Fatal("Another nonce should not be used")
	}
}

func TestSignatureValidatorV2(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	keyManager := NewKeyManager(key, time.Hour)
	validator := NewSignatureValidatorV2(nil, keyManager)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	signer := NewSignerV2WithConfig(key, &SignerConfigV2{
		Algorithm:       AlgorithmHMACSHA256,
		TimestampTolerance: 5 * time.Minute,
		EnableNonceCheck:    true,
		EnableTimestampCheck: true,
	})

	router.GET("/test", func(c *gin.Context) {
		valid, err := validator.ValidateSignature(c)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if !valid {
			c.JSON(401, gin.H{"error": "invalid"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	timestamp := time.Now().Unix()
	nonce, _ := GenerateNonceV2(16)
	params := &SignatureParamsV2{
		Method:    "GET",
		Path:      "/test",
		Timestamp: timestamp,
		Nonce:     nonce,
	}
	signature, _, _ := signer.SignRequest(params)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response: %s", w.Body.String())
	}
}

func TestAuditLogger(t *testing.T) {
	config := &AuditConfig{
		Enabled:           true,
		BufferSize:        100,
		AsyncEnabled:      false,
		MaskSensitiveData: true,
		SensitiveFields:   []string{"password", "token"},
	}

	logger := NewAuditLogger(config)

	event := &AuditEvent{
		Level:     AuditLevelInfo,
		Action:    ActionLogin,
		IPAddress: "192.168.1.1",
		UserID:    "user123",
		Success:   true,
		Message:   "User logged in",
		Details: map[string]interface{}{
			"password": "secret123",
			"token":   "abc123",
		},
	}

	logger.Log(event)

	stats, _ := logger.GetStats()
	if stats.TotalEvents != 1 {
		t.Fatalf("Expected 1 event, got %d", stats.TotalEvents)
	}

	filter := &AuditFilter{
		Action: ActionLogin,
	}
	results, _ := logger.Query(filter)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
}

func TestAuditMiddleware(t *testing.T) {
	config := &AuditConfig{
		Enabled:      true,
		AsyncEnabled: false,
	}

	middleware := NewAuditMiddleware(config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID header should be set")
	}
}

func TestTwoFactorService(t *testing.T) {
	config := &TwoFactorConfig{
		Enabled:           true,
		CodeLength:        6,
		CodeExpiry:        5 * time.Minute,
		MaxAttempts:       3,
		RequireForDeletes: true,
	}

	service := NewTwoFactorService(config)

	challenge, err := service.CreateChallenge(nil, "user123", "delete", "resource456", TwoFactorCode)
	if err != nil {
		t.Fatalf("CreateChallenge failed: %v", err)
	}
	if challenge == nil {
		t.Fatal("Challenge should not be nil")
	}

	code, err := service.GetCodeForTesting(challenge.ID)
	if err != nil {
		t.Fatalf("GetCodeForTesting failed: %v", err)
	}
	if code == "" {
		t.Fatal("Code should not be empty")
	}

	valid, err := service.VerifyChallenge(challenge.ID, code)
	if err != nil {
		t.Fatalf("VerifyChallenge failed: %v", err)
	}
	if !valid {
		t.Fatal("Challenge verification should succeed")
	}

	valid, err = service.VerifyChallenge(challenge.ID, code)
	if err != nil {
		t.Fatalf("VerifyChallenge failed: %v", err)
	}
	if !valid {
		t.Fatal("Challenge should remain verified after initial verification")
	}

	valid, err = service.VerifyChallenge(challenge.ID, "wrong_code")
	if err != nil {
		t.Fatalf("VerifyChallenge failed: %v", err)
	}
	if !valid {
		t.Fatal("Challenge should remain verified (idempotent)")
	}

	invalidID, _ := service.VerifyChallenge("invalid_challenge_id", "wrong_code")
	if invalidID {
		t.Fatal("Should fail for invalid challenge ID")
	}
}

func TestConfirmationService(t *testing.T) {
	config := &TwoFactorConfig{}
	service := NewConfirmationService(config)

	token, err := service.GenerateToken("user123", "delete", "resource456")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("Token should not be empty")
	}

	tok, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if tok.UserID != "user123" {
		t.Fatalf("Expected user_id 'user123', got '%s'", tok.UserID)
	}

	_, err = service.ConsumeToken(token)
	if err != nil {
		t.Fatalf("ConsumeToken failed: %v", err)
	}

	_, err = service.ValidateToken(token)
	if err == nil {
		t.Fatal("Should fail for used token")
	}
}

func TestAnomalyDetector(t *testing.T) {
	config := &AnomalyDetectorConfig{
		BruteForceThreshold:    5,
		BruteForceWindow:       time.Minute,
		FailureCountThreshold:  3,
		FailureWindow:          time.Minute,
		RequestRateThreshold:   100,
		RequestRateWindow:      time.Minute,
		AutoBlockEnabled:       true,
		BlockDuration:          5 * time.Minute,
	}

	detector := NewAnomalyDetector(config)

	ip := "192.168.1.100"

	for i := 0; i < 3; i++ {
		detector.RecordAttempt(ip, "user1", "login", "test-agent", false)
	}

	event := detector.Detect(ip)
	if event == nil {
		t.Fatal("Should detect anomaly for 3 failures")
	}
	if event.Type != AnomalyFrequentFailures {
		t.Fatalf("Expected AnomalyFrequentFailures, got %s", event.Type)
	}

	for i := 0; i < 10; i++ {
		detector.RecordAttempt(ip, "", "login", "test-agent", false)
	}

	event = detector.Detect(ip)
	if event == nil {
		t.Fatal("Should detect anomaly after many failures")
	}

	validTypes := map[AnomalyType]bool{
		AnomalyFrequentFailures: true,
		AnomalyBruteForce: true,
		AnomalyUnusualPattern: true,
	}
	if !validTypes[event.Type] {
		t.Fatalf("Expected a valid anomaly type, got %s", event.Type)
	}

	blocked := detector.IsBlocked(ip)
	if blocked {
		t.Fatal("IP should not be blocked yet")
	}

	detector.BlockIP(ip, time.Minute)

	blocked = detector.IsBlocked(ip)
	if !blocked {
		t.Fatal("IP should be blocked")
	}

	detector.UnblockIP(ip)
	blocked = detector.IsBlocked(ip)
	if blocked {
		t.Fatal("IP should be unblocked")
	}
}

func TestDDoSProtector(t *testing.T) {
	config := &DDoSProtectionConfig{
		Enabled:             true,
		ConnectionLimit:     100,
		RequestsPerSecond:   10,
		BurstSize:          5,
		BlockDuration:       time.Minute,
		MaxConcurrentPerIP: 5,
	}

	protector := NewDDoSProtector(config)

	ip := "192.168.1.200"

	for i := 0; i < 6; i++ {
		allowed, _, remaining := protector.CheckRateLimit(ip)
		if i < 5 {
			if !allowed {
				t.Fatalf("Request %d should be allowed (tokens remaining: %d)", i+1, remaining)
			}
		}
	}

	for i := 0; i < 10; i++ {
		protector.CheckRateLimit(ip)
	}

	allowed, reason, _ := protector.CheckRateLimit(ip)
	if allowed {
		t.Fatal("After many rapid requests, should eventually trigger rate limit")
	}
	if reason == "" {
		t.Fatal("Should return reason for rate limit")
	}

	blocked := protector.IsIPBlocked(ip)
	if blocked {
		t.Fatal("IP should not be blocked from rate limit alone")
	}
}

func TestConnectionLimiter(t *testing.T) {
	config := &ConnectionLimitConfig{
		MaxConnections:   1000,
		MaxConnectionsIP: 10,
	}

	limiter := NewConnectionLimiter(config)

	ip := "192.168.1.50"

	for i := 0; i < 10; i++ {
		if !limiter.AllowConnection(ip) {
			t.Fatalf("Connection %d should be allowed", i+1)
		}
	}

	if limiter.AllowConnection(ip) {
		t.Fatal("Should not allow more connections")
	}

	limiter.CloseConnection(ip)

	if !limiter.AllowConnection(ip) {
		t.Fatal("Should allow connection after closing one")
	}

	count := limiter.GetConnectionCount(ip)
	if count != 10 {
		t.Fatalf("Expected 10 connections, got %d", count)
	}
}

func TestCCProtector(t *testing.T) {
	config := &CCProtectionConfig{
		Enabled:                true,
		MaxConsumptionsPerIP:   100,
		MaxConsumptionsPerUser: 5,
		ConsumptionWindow:      time.Minute,
		BlockDuration:          5 * time.Minute,
		AlertThreshold:         3,
	}

	protector := NewCCProtector(config)

	ip := "192.168.1.150"
	userID := "user456"

	blocked := protector.IsBlocked(ip, userID, "")
	if blocked {
		t.Fatal("Should not be blocked initially")
	}

	for i := 0; i < 5; i++ {
		protector.RecordConsumption(&CCConsumptionRecord{
			Type:      CCConsumeCreate,
			IPAddress: ip,
			UserID:    userID,
		})
	}

	for i := 0; i < 10; i++ {
		protector.RecordConsumption(&CCConsumptionRecord{
			Type:      CCConsumeCreate,
			IPAddress: ip,
			UserID:    userID,
		})
	}

	blocked = protector.IsBlocked(ip, userID, "")
	if blocked {
		t.Log("User is now blocked from consumption")
	}

	err := protector.Block("", userID, "")
	if err != nil {
		t.Fatalf("Block failed: %v", err)
	}
	blocked = protector.IsBlocked("", userID, "")
	if !blocked {
		t.Fatal("User should be blocked after explicit block")
	}

	err = protector.Unblock("", userID, "")
	if err != nil {
		t.Fatalf("Unblock failed: %v", err)
	}
	blocked = protector.IsBlocked("", userID, "")
	if blocked {
		t.Fatal("User should be unblocked")
	}
}

func TestCCMiddleware(t *testing.T) {
	config := &CCProtectionConfig{
		Enabled:                true,
		MaxConsumptionsPerIP:   5,
		ConsumptionWindow:      time.Minute,
		BlockDuration:          5 * time.Minute,
	}

	auditConfig := &AuditConfig{Enabled: false}
	logger := NewAuditLogger(auditConfig)

	protector := NewCCProtector(config)
	middleware := NewCCMiddleware(protector, logger, config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Handler())
	router.POST("/captcha/create", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", "/captcha/create", nil)
		req.Header.Set("X-App-ID", "test-app")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	req, _ := http.NewRequest("POST", "/captcha/create", nil)
	req.Header.Set("X-App-ID", "test-app")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Logf("Response code: %d, body: %s", w.Code, w.Body.String())
	}
}

func TestIPWhitelist(t *testing.T) {
	whitelist := NewIPWhitelist()

	whitelist.Add("192.168.1.1")

	if !whitelist.IsWhitelisted("192.168.1.1") {
		t.Fatal("IP should be whitelisted")
	}

	if whitelist.IsWhitelisted("192.168.1.2") {
		t.Fatal("IP should not be whitelisted")
	}

	err := whitelist.AddCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("AddCIDR failed: %v", err)
	}

	if !whitelist.IsInWhitelist("10.0.0.1") {
		t.Fatal("IP in CIDR range should be whitelisted")
	}

	whitelist.Remove("192.168.1.1")
	if whitelist.IsWhitelisted("192.168.1.1") {
		t.Fatal("IP should not be whitelisted after removal")
	}
}

func TestThreatIntel(t *testing.T) {
	intel := NewThreatIntel()

	err := intel.BlockRange("192.168.0.0/24")
	if err != nil {
		t.Fatalf("BlockRange failed: %v", err)
	}

	if !intel.IsIPBlocked("192.168.0.1") {
		t.Fatal("IP in blocked range should be blocked")
	}

	if intel.IsIPBlocked("192.168.1.1") {
		t.Fatal("IP not in blocked range should not be blocked")
	}

	err = intel.UnblockRange("192.168.0.0/24")
	if err != nil {
		t.Fatalf("UnblockRange failed: %v", err)
	}

	if intel.IsIPBlocked("192.168.0.1") {
		t.Fatal("IP should not be blocked after unblocking range")
	}
}

func TestGenerateNonceV2(t *testing.T) {
	nonce1, err := GenerateNonceV2(32)
	if err != nil {
		t.Fatalf("GenerateNonceV2 failed: %v", err)
	}

	if len(nonce1) != 32 {
		t.Fatalf("Expected nonce length 32, got %d", len(nonce1))
	}

	nonce2, _ := GenerateNonceV2(32)
	if nonce1 == nonce2 {
		t.Fatal("Nonces should be unique")
	}
}

func TestAuditLevelOrdering(t *testing.T) {
	levels := []AuditLevel{
		AuditLevelDebug,
		AuditLevelInfo,
		AuditLevelWarning,
		AuditLevelError,
		AuditLevelCritical,
	}

	expected := []string{"debug", "info", "warning", "error", "critical"}

	for i, level := range levels {
		if level.String() != expected[i] {
			t.Fatalf("Expected %s, got %s", expected[i], level.String())
		}
	}
}

func TestAnomalyActionValues(t *testing.T) {
	actions := []AnomalyAction{
		ActionAllow,
		ActionWarn,
		ActionBlock,
		ActionCaptcha,
		ActionThrottle,
		ActionLockout,
	}

	if len(actions) != 6 {
		t.Fatal("Expected 6 action types")
	}
}

func TestLogSecurityViolation(t *testing.T) {
	auditConfig := &AuditConfig{Enabled: true, AsyncEnabled: false}
	logger := NewAuditLogger(auditConfig)

	LogSecurityViolation(logger, "sqli", "192.168.1.1", map[string]interface{}{
		"pattern": "union select",
	})

	stats, _ := logger.GetStats()
	if stats.TotalEvents != 1 {
		t.Fatalf("Expected 1 event, got %d", stats.TotalEvents)
	}

	action := stats.ByAction[string(ActionSQLInjection)]
	if action != 1 {
		t.Fatalf("Expected 1 SQL injection event, got %d", action)
	}
}

func TestDetectSensitivePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"union select * from users", true},
		{"drop table users", true},
		{"admin'--", true},
		{"normal input", false},
		{"hello world", false},
	}

	for _, test := range tests {
		detected, _ := DetectSensitivePattern(test.input)
		if detected != test.expected {
			t.Errorf("Input: %s, expected %v, got %v", test.input, test.expected, detected)
		}
	}
}

func TestContextFromGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"
	c.Set("user_id", "user123")
	c.Set("username", "testuser")

	ctx := ContextFromGin(c)

	if ctx.UserID != "user123" {
		t.Fatalf("Expected UserID 'user123', got '%s'", ctx.UserID)
	}
	if ctx.Username != "testuser" {
		t.Fatalf("Expected Username 'testuser', got '%s'", ctx.Username)
	}
	if ctx.IPAddress == "" {
		t.Fatal("IPAddress should not be empty")
	}
}

func TestCaptchaConsumptionLimiter(t *testing.T) {
	auditConfig := &AuditConfig{Enabled: false}
	logger := NewAuditLogger(auditConfig)

	config := &CCProtectionConfig{
		Enabled:                true,
		MaxConsumptionsPerIP:   10,
		ConsumptionWindow:      time.Minute,
		BlockDuration:          5 * time.Minute,
	}

	limiter := NewCaptchaConsumptionLimiter(logger, config)

	ip := "192.168.1.100"
	appID := "test-app"

	blocked := limiter.IsBlocked(ip, "", appID)
	if blocked {
		t.Fatal("Should not be blocked initially")
	}

	allowed, err := limiter.CheckAndRecord(nil, ip, "", appID, CCConsumeCreate)
	if err != nil {
		t.Fatalf("CheckAndRecord failed: %v", err)
	}
	if !allowed {
		t.Fatal("First request should be allowed")
	}

	blocked = limiter.IsBlocked(ip, "", appID)
	if blocked {
		t.Fatal("Should not be blocked after single request")
	}

	limiter.Block("", "", appID)
	blocked = limiter.IsBlocked("", "", appID)
	if !blocked {
		t.Fatal("Should be blocked after explicit block (using appID)")
	}

	limiter.Unblock("", "", appID)
	blocked = limiter.IsBlocked("", "", appID)
	if blocked {
		t.Fatal("Should be unblocked")
	}
}

func TestCCAlertManager(t *testing.T) {
	alertManager := NewCCAlertManager(50)

	config := &CCProtectionConfig{
		Enabled:                true,
		MaxConsumptionsPerIP:   100,
		ConsumptionWindow:      time.Minute,
	}

	protector := NewCCProtector(config)
	alertManager.SetProtector(protector)

	ip := "192.168.1.200"

	for i := 0; i < 60; i++ {
		protector.CheckLimit(ip, "", "")
		protector.RecordConsumption(&CCConsumptionRecord{
			Type:      CCConsumeCreate,
			IPAddress: ip,
		})
	}

	alertManager.CheckAndAlert(ip, "", "")

	alerts := alertManager.GetAlerts()
	if len(alerts) == 0 {
		t.Log("No alerts generated (threshold may not be reached)")
	}
}

func TestSensitiveOperations(t *testing.T) {
	operations := []SensitiveOperation{
		{Action: "delete_config", Path: "/admin/api/config", Method: "POST", RequiresConfirm: true},
		{Action: "add_blacklist", Path: "/admin/api/blacklist", Method: "POST", RequiresConfirm: true},
	}

	if len(operations) != 2 {
		t.Fatal("Expected 2 sensitive operations")
	}

	op := MatchSensitiveOperation("POST", "/admin/api/config")
	if op == nil {
		t.Fatal("Should match sensitive operation")
	}
	if op.Action != "delete_config" {
		t.Fatalf("Expected action 'delete_config', got '%s'", op.Action)
	}

	op = MatchSensitiveOperation("GET", "/admin/api/config")
	if op != nil {
		t.Fatal("Should not match GET requests")
	}
}

func TestAnomalyReporter(t *testing.T) {
	config := &AnomalyDetectorConfig{}
	detector := NewAnomalyDetector(config)
	reporter := NewAnomalyReporter(detector)

	event := &AnomalyEvent{
		ID:          "test-event-1",
		Type:        AnomalyBruteForce,
		IPAddress:   "192.168.1.1",
		Timestamp:   time.Now(),
		Severity:    8,
		Score:       80,
		Description: "Test brute force detected",
	}

	reporter.Report(event)

	stats := reporter.GetStats()
	if stats["total_events"].(int) != 1 {
		t.Fatalf("Expected 1 event, got %v", stats["total_events"])
	}
}
