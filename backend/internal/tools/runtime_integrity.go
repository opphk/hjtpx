package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type HashAlgorithm string

const (
	HashSHA256   HashAlgorithm = "sha256"
	HashSHA512   HashAlgorithm = "sha512"
	HashHMACSHA256 HashAlgorithm = "hmac_sha256"
)

type RuntimeIntegrity struct {
	config              *model.IntegrityConfig
	records             *model.ThreadSafeIntegrityChecker
	alertHandlers       []AlertHandler
	violationCallbacks  []func(model.IntegrityViolation)
	checkInterval       time.Duration
	stopChan            chan struct{}
	running             bool
	mu                  sync.RWMutex
	originalFunctions   map[string]string
	knownHashes         map[string]string
	dynamicCodePatterns []*regexp.Regexp
}

type AlertHandler interface {
	HandleAlert(alert *model.IntegrityAlert) error
}

type AlertHandlerFunc func(alert *model.IntegrityAlert) error

func (f AlertHandlerFunc) HandleAlert(alert *model.IntegrityAlert) error {
	return f(alert)
}

type WebhookAlertHandler struct {
	endpoint   string
	httpClient HTTPClient
	timeout    time.Duration
}

type HTTPClient interface {
	Post(url string, body []byte) error
}

type DefaultHTTPClient struct{}

func (c *DefaultHTTPClient) Post(url string, body []byte) error {
	return errors.New("http client not configured - implement for production use")
}

func NewWebhookAlertHandler(endpoint string) *WebhookAlertHandler {
	return &WebhookAlertHandler{
		endpoint:   endpoint,
		httpClient: &DefaultHTTPClient{},
		timeout:    10 * time.Second,
	}
}

func (h *WebhookAlertHandler) HandleAlert(alert *model.IntegrityAlert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	return h.httpClient.Post(h.endpoint, data)
}

type LogAlertHandler struct {
	logger Logger
}

type Logger interface {
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Info(format string, args ...interface{})
}

type DefaultLogger struct{}

func (l *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (l *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func NewLogAlertHandler() *LogAlertHandler {
	return &LogAlertHandler{
		logger: &DefaultLogger{},
	}
}

func (h *LogAlertHandler) HandleAlert(alert *model.IntegrityAlert) error {
	h.logger.Error("Integrity Alert: Type=%s, Severity=%d, Target=%s, Message=%s",
		alert.Type, alert.Severity, alert.Target, alert.Message)
	return nil
}

type IntegrityViolationHandler struct {
	handlers []AlertHandler
	mu       sync.Mutex
}

func NewIntegrityViolationHandler() *IntegrityViolationHandler {
	return &IntegrityViolationHandler{
		handlers: make([]AlertHandler, 0),
	}
}

func (h *IntegrityViolationHandler) AddHandler(handler AlertHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers = append(h.handlers, handler)
}

func (h *IntegrityViolationHandler) HandleViolation(violation *model.IntegrityViolation) {
	h.mu.Lock()
	defer h.mu.Unlock()

	alert := &model.IntegrityAlert{
		ID:        fmt.Sprintf("alert-%d-%s", time.Now().UnixNano(), violation.Type),
		Type:      violation.Type,
		Severity:  violation.Severity,
		Message:   fmt.Sprintf("Integrity violation detected: %s", violation.Type),
		Target:    violation.Target,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"expected_hash": violation.ExpectedHash,
			"actual_hash":   violation.ActualHash,
		},
	}

	for _, handler := range h.handlers {
		go func(h AlertHandler) {
			_ = h.HandleAlert(alert)
		}(handler)
	}
}

func NewRuntimeIntegrity(config *model.IntegrityConfig) *RuntimeIntegrity {
	if config == nil {
		config = model.NewIntegrityConfig()
	}

	ri := &RuntimeIntegrity{
		config:              config,
		records:             model.NewThreadSafeIntegrityChecker(),
		alertHandlers:       make([]AlertHandler, 0),
		violationCallbacks:  make([]func(model.IntegrityViolation), 0),
		checkInterval:       config.CheckInterval,
		stopChan:            make(chan struct{}),
		originalFunctions:   make(map[string]string),
		knownHashes:         make(map[string]string),
		dynamicCodePatterns: make([]*regexp.Regexp, 0),
	}

	ri.initializeDynamicCodePatterns()
	ri.initializeDefaultAlertHandlers()

	return ri
}

func (r *RuntimeIntegrity) initializeDynamicCodePatterns() {
	patterns := []string{
		`eval\s*\(`,
		`new\s+Function\s*\(`,
		`setTimeout\s*\(\s*['"]`,
		`setInterval\s*\(\s*['"]`,
		`document\s*\.\s*write\s*\(`,
		`innerHTML\s*=`,
		`outerHTML\s*=`,
		`insertAdjacentHTML`,
		`createElement\s*\(\s*['"]script['"]`,
		`setAttribute\s*\(\s*['"]src['"]`,
		`appendChild\s*\(\s*createElement\s*\(\s*['"]script['"]`,
		`import\s*\(\s*['"]`,
		`import\s*\(`,
		`Reflect\.import`,
		`System\.import`,
		`require\s*\(\s*['"]`,
		`define\s*\(`,
		`defineAsync\s*\(`,
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			r.dynamicCodePatterns = append(r.dynamicCodePatterns, re)
		}
	}
}

func (r *RuntimeIntegrity) initializeDefaultAlertHandlers() {
	r.AddAlertHandler(NewLogAlertHandler())
}

func (r *RuntimeIntegrity) AddAlertHandler(handler AlertHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alertHandlers = append(r.alertHandlers, handler)
}

func (r *RuntimeIntegrity) AddViolationCallback(callback func(model.IntegrityViolation)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.violationCallbacks = append(r.violationCallbacks, callback)
}

func (r *RuntimeIntegrity) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return errors.New("integrity checker is already running")
	}
	r.running = true
	r.stopChan = make(chan struct{})
	r.mu.Unlock()

	go r.periodicCheck()

	return nil
}

func (r *RuntimeIntegrity) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return
	}

	r.running = false
	close(r.stopChan)
}

func (r *RuntimeIntegrity) periodicCheck() {
	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = r.CheckIntegrity()
		case <-r.stopChan:
			return
		}
	}
}

func (r *RuntimeIntegrity) CheckIntegrity() (*model.IntegrityCheckResult, error) {
	startTime := time.Now()
	result := model.NewIntegrityCheckResult()

	r.mu.RLock()
	enableHashCheck := r.config.EnableHashCheck
	enableDynamicCodeCheck := r.config.EnableDynamicCodeCheck
	enableMemoryCheck := r.config.EnableMemoryCheck
	enableFunctionHookCheck := r.config.EnableFunctionHookCheck
	enablePrototypeCheck := r.config.EnablePrototypeCheck
	r.mu.RUnlock()

	if enableHashCheck {
		hashResult := r.checkCodeHashes()
		if !hashResult.IsValid {
			result.Violations = append(result.Violations, hashResult.Violations...)
		}
	}

	if enableDynamicCodeCheck {
		dynamicResult := r.checkDynamicCodeLoading()
		if !dynamicResult.IsValid {
			result.Violations = append(result.Violations, dynamicResult.Violations...)
		}
	}

	if enableMemoryCheck {
		memoryResult := r.checkMemoryIntegrity()
		if !memoryResult.IsValid {
			result.Violations = append(result.Violations, memoryResult.Violations...)
		}
	}

	if enableFunctionHookCheck {
		hookResult := r.checkFunctionHooks()
		if !hookResult.IsValid {
			result.Violations = append(result.Violations, hookResult.Violations...)
		}
	}

	if enablePrototypeCheck {
		protoResult := r.checkPrototypeIntegrity()
		if !protoResult.IsValid {
			result.Violations = append(result.Violations, protoResult.Violations...)
		}
	}

	result.Duration = time.Since(startTime)
	result.CheckedAt = time.Now()

	if len(result.Violations) == 0 {
		result.IsValid = true
		result.Status = model.IntegrityStatusOK
	} else {
		result.IsValid = false
		result.Status = model.IntegrityStatusModified
		r.handleViolations(result.Violations)
	}

	return result, nil
}

func (r *RuntimeIntegrity) checkCodeHashes() *model.IntegrityCheckResult {
	result := model.NewIntegrityCheckResult()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for targetName, expectedHash := range r.knownHashes {
		if record, exists := r.records.GetRecord(targetName); exists {
			if record.CurrentHash != expectedHash {
				violation := model.IntegrityViolation{
					Type:         model.ViolationCodeHashMismatch,
					Severity:     9,
					Target:       targetName,
					ExpectedHash: expectedHash,
					ActualHash:   record.CurrentHash,
					Timestamp:    time.Now(),
				}
				result.AddViolation(violation)
				r.records.UpdateStats(string(model.ViolationCodeHashMismatch), true)
			} else {
				r.records.UpdateStats(string(model.ViolationCodeHashMismatch), false)
			}
		}
	}

	return result
}

func (r *RuntimeIntegrity) checkDynamicCodeLoading() *model.IntegrityCheckResult {
	result := model.NewIntegrityCheckResult()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pattern := range r.dynamicCodePatterns {
		matches := pattern.FindAllString(r.getCurrentRuntimeState(), -1)
		if len(matches) > 0 {
			violation := model.IntegrityViolation{
				Type:       model.ViolationDynamicCodeLoad,
				Severity:   7,
				Target:     fmt.Sprintf("dynamic_pattern:%s", pattern.String()),
				Timestamp:  time.Now(),
				StackTrace: r.getStackTrace(),
			}
			violation.SetMetadata(map[string]interface{}{
				"matched_count": len(matches),
				"pattern":       pattern.String(),
			})
			result.AddViolation(violation)
			r.records.UpdateStats(string(model.ViolationDynamicCodeLoad), true)
		} else {
			r.records.UpdateStats(string(model.ViolationDynamicCodeLoad), false)
		}
	}

	return result
}

func (r *RuntimeIntegrity) checkMemoryIntegrity() *model.IntegrityCheckResult {
	result := model.IntegrityCheckResult{
		IsValid:    true,
		Status:     model.IntegrityStatusOK,
		Violations: make([]model.IntegrityViolation, 0),
	}

	criticalRegions := []string{
		"Object.prototype",
		"Function.prototype",
		"Array.prototype",
		"String.prototype",
		"Number.prototype",
		"Boolean.prototype",
	}

	for _, region := range criticalRegions {
		if r.isRegionModified(region) {
			violation := model.IntegrityViolation{
				Type:       model.ViolationMemoryModification,
				Severity:   8,
				Target:     region,
				Timestamp:  time.Now(),
				StackTrace: r.getStackTrace(),
			}
			result.AddViolation(violation)
			r.records.UpdateStats(string(model.ViolationMemoryModification), true)
		} else {
			r.records.UpdateStats(string(model.ViolationMemoryModification), false)
		}
	}

	return &result
}

func (r *RuntimeIntegrity) isRegionModified(region string) bool {
	hashKey := fmt.Sprintf("proto_hash_%s", region)

	r.mu.Lock()
	defer r.mu.Unlock()

	originalHash, exists := r.originalFunctions[hashKey]
	currentHash := r.calculateObjectHash(region)

	if !exists {
		r.originalFunctions[hashKey] = currentHash
		return false
	}

	if currentHash != originalHash && originalHash != "" {
		return true
	}

	return false
}

func (r *RuntimeIntegrity) checkFunctionHooks() *model.IntegrityCheckResult {
	result := model.IntegrityCheckResult{
		IsValid:    true,
		Status:     model.IntegrityStatusOK,
		Violations: make([]model.IntegrityViolation, 0),
	}

	criticalFunctions := []string{
		"eval",
		"Function",
		"setTimeout",
		"setInterval",
		"document.write",
		"fetch",
		"XMLHttpRequest",
	}

	for _, fn := range criticalFunctions {
		if r.isFunctionHooked(fn) {
			violation := model.IntegrityViolation{
				Type:       model.ViolationFunctionHook,
				Severity:   8,
				Target:     fn,
				Timestamp:  time.Now(),
				StackTrace: r.getStackTrace(),
			}
			result.AddViolation(violation)
			r.records.UpdateStats(string(model.ViolationFunctionHook), true)
		} else {
			r.records.UpdateStats(string(model.ViolationFunctionHook), false)
		}
	}

	return &result
}

func (r *RuntimeIntegrity) isFunctionHooked(functionName string) bool {
	hashKey := fmt.Sprintf("fn_hash_%s", functionName)

	r.mu.Lock()
	defer r.mu.Unlock()

	originalHash, exists := r.originalFunctions[hashKey]
	currentHash := r.calculateFunctionHash(functionName)

	if !exists {
		r.originalFunctions[hashKey] = currentHash
		return false
	}

	if currentHash != originalHash && originalHash != "" {
		return true
	}

	return false
}

func (r *RuntimeIntegrity) checkPrototypeIntegrity() *model.IntegrityCheckResult {
	result := model.IntegrityCheckResult{
		IsValid:    true,
		Status:     model.IntegrityStatusOK,
		Violations: make([]model.IntegrityViolation, 0),
	}

	prototypes := []string{
		"Object.prototype.toString",
		"Object.prototype.hasOwnProperty",
		"Function.prototype.call",
		"Function.prototype.apply",
		"Array.prototype.push",
		"Array.prototype.pop",
		"String.prototype.charAt",
	}

	for _, proto := range prototypes {
		parts := strings.Split(proto, ".")
		if len(parts) < 2 {
			continue
		}

		hashKey := fmt.Sprintf("proto_integrity_%s", strings.Join(parts, "_"))

		r.mu.Lock()
		originalHash, exists := r.originalFunctions[hashKey]
		currentHash := r.calculatePropertyHash(proto)

		if !exists {
			r.originalFunctions[hashKey] = currentHash
			r.mu.Unlock()
			continue
		}
		r.mu.Unlock()

		if currentHash != originalHash {
			violation := model.IntegrityViolation{
				Type:       model.ViolationPrototypeModification,
				Severity:   7,
				Target:     proto,
				Timestamp:  time.Now(),
				StackTrace: r.getStackTrace(),
			}
			result.AddViolation(violation)
			r.records.UpdateStats(string(model.ViolationPrototypeModification), true)
		} else {
			r.records.UpdateStats(string(model.ViolationPrototypeModification), false)
		}
	}

	return &result
}

func (r *RuntimeIntegrity) calculateObjectHash(objName string) string {
	hashData := fmt.Sprintf("object:%s:%d", objName, time.Now().UnixNano()/int64(time.Millisecond))
	return r.computeHash([]byte(hashData))
}

func (r *RuntimeIntegrity) calculateFunctionHash(fnName string) string {
	hashData := fmt.Sprintf("function:%s:%d", fnName, time.Now().UnixNano()/int64(time.Millisecond))
	return r.computeHash([]byte(hashData))
}

func (r *RuntimeIntegrity) calculatePropertyHash(propName string) string {
	hashData := fmt.Sprintf("property:%s:%d", propName, time.Now().UnixNano()/int64(time.Millisecond))
	return r.computeHash([]byte(hashData))
}

func (r *RuntimeIntegrity) computeHash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (r *RuntimeIntegrity) getCurrentRuntimeState() string {
	return fmt.Sprintf("runtime_state_%d", time.Now().UnixNano())
}

func (r *RuntimeIntegrity) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (r *RuntimeIntegrity) handleViolations(violations []model.IntegrityViolation) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler := &IntegrityViolationHandler{}
	for _, h := range r.alertHandlers {
		handler.AddHandler(h)
	}

	for _, violation := range violations {
		v := violation
		handler.HandleViolation(&v)

		for _, callback := range r.violationCallbacks {
			callback(violation)
		}
	}
}

func (r *RuntimeIntegrity) VerifyHash(data []byte, expectedHash string) bool {
	actualHash := r.GenerateHash(data)
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}

func (r *RuntimeIntegrity) GenerateHash(data []byte) string {
	return r.computeHash(data)
}

func (r *RuntimeIntegrity) RegisterCodeHash(targetName string, code []byte) {
	hash := r.GenerateHash(code)
	r.knownHashes[targetName] = hash

	record := &model.CodeIntegrityRecord{
		TargetName:   targetName,
		TargetType:   "code",
		OriginalHash: hash,
		CurrentHash:  hash,
		Version:      "1.0",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	r.records.AddRecord(record)
}

func (r *RuntimeIntegrity) VerifyCodeIntegrity(targetName string, code []byte) (*model.IntegrityCheckResult, error) {
	result := model.NewIntegrityCheckResult()

	r.mu.RLock()
	expectedHash, exists := r.knownHashes[targetName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("target '%s' not registered", targetName)
	}

	actualHash := r.GenerateHash(code)
	result.HashValue = actualHash
	result.TargetName = targetName

	if !hmac.Equal([]byte(actualHash), []byte(expectedHash)) {
		violation := model.IntegrityViolation{
			Type:         model.ViolationCodeHashMismatch,
			Severity:     9,
			Target:       targetName,
			ExpectedHash: expectedHash,
			ActualHash:   actualHash,
			Timestamp:    time.Now(),
		}
		result.AddViolation(violation)

		r.handleViolations([]model.IntegrityViolation{violation})

		r.mu.RLock()
		record, _ := r.records.GetRecord(targetName)
		r.mu.RUnlock()

		if record != nil {
			record.CurrentHash = actualHash
			record.ViolationCount++
			r.records.UpdateRecord(targetName, record)
		}
	}

	return result, nil
}

func (r *RuntimeIntegrity) GetStats() *model.IntegrityStats {
	return r.records.GetStats()
}

func (r *RuntimeIntegrity) GetRecord(targetName string) (*model.CodeIntegrityRecord, bool) {
	return r.records.GetRecord(targetName)
}

func (r *RuntimeIntegrity) SetConfig(config *model.IntegrityConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
	if config.CheckInterval > 0 {
		r.checkInterval = config.CheckInterval
	}
}

func (r *RuntimeIntegrity) GetConfig() *model.IntegrityConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *RuntimeIntegrity) GenerateIntegrityReport(sessionID string) (*model.IntegrityReport, error) {
	result, err := r.CheckIntegrity()
	if err != nil {
		return nil, err
	}

	report := &model.IntegrityReport{
		SessionID:     sessionID,
		CheckResults:  []*model.IntegrityCheckResult{result},
		GeneratedAt:   time.Now(),
		Summary:       r.GetStats(),
	}

	if result.IsValid {
		report.OverallStatus = model.IntegrityStatusOK
	} else {
		report.OverallStatus = model.IntegrityStatusModified
	}

	return report, nil
}

func (r *RuntimeIntegrity) ExportAlertAsJSON() (string, error) {
	report, err := r.GenerateIntegrityReport("")
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

type HashBasedIntegrity struct {
	algorithm HashAlgorithm
	secretKey []byte
}

func NewHashBasedIntegrity(algorithm HashAlgorithm) *HashBasedIntegrity {
	return &HashBasedIntegrity{
		algorithm: algorithm,
	}
}

func (h *HashBasedIntegrity) SetSecretKey(key []byte) {
	h.secretKey = key
}

func (h *HashBasedIntegrity) ComputeHash(data []byte) string {
	switch h.algorithm {
	case HashSHA256:
		return h.sha256Hash(data)
	case HashSHA512:
		return h.sha512Hash(data)
	case HashHMACSHA256:
		return h.hmacSHA256Hash(data)
	default:
		return h.sha256Hash(data)
	}
}

func (h *HashBasedIntegrity) sha256Hash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (h *HashBasedIntegrity) sha512Hash(data []byte) string {
	hasher := sha512New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (h *HashBasedIntegrity) hmacSHA256Hash(data []byte) string {
	if len(h.secretKey) == 0 {
		h.secretKey = []byte("default-secret-key")
	}
	mac := hmac.New(sha256.New, h.secretKey)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *HashBasedIntegrity) VerifyHash(data []byte, expectedHash string) bool {
	actualHash := h.ComputeHash(data)
	return hmac.Equal([]byte(actualHash), []byte(expectedHash))
}

func (h *HashBasedIntegrity) GenerateIntegrityToken(data []byte, timestamp int64) string {
	tokenData := fmt.Sprintf("%s:%d", string(data), timestamp)
	hash := h.ComputeHash([]byte(tokenData))
	return base64.StdEncoding.EncodeToString([]byte(hash + ":" + fmt.Sprintf("%d", timestamp)))
}

func (h *HashBasedIntegrity) VerifyIntegrityToken(data []byte, token string, maxAge time.Duration) bool {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return false
	}

	decodedStr := string(decoded)
	parts := strings.SplitN(decodedStr, ":", 2)
	if len(parts) != 2 {
		return false
	}

	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}

	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > maxAge {
		return false
	}

	expectedToken := h.GenerateIntegrityToken(data, timestamp)
	return token == expectedToken
}

func sha512New() hash.Hash {
	return sha256.New()
}

func ComputeFileHash(filePath string, algorithm HashAlgorithm) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func VerifyFileIntegrity(filePath, expectedHash string) (bool, error) {
	actualHash, err := ComputeFileHash(filePath, HashSHA256)
	if err != nil {
		return false, err
	}

	return hmac.Equal([]byte(actualHash), []byte(expectedHash)), nil
}

func GenerateMultiHash(data []byte) map[HashAlgorithm]string {
	return map[HashAlgorithm]string{
		HashSHA256:      sha256Hash(data),
		HashHMACSHA256:  hmacHash(data, []byte("default-key")),
	}
}

func sha256Hash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func hmacHash(data, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
