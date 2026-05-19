package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewRuntimeIntegrity(t *testing.T) {
	config := model.NewIntegrityConfig()
	ri := NewRuntimeIntegrity(config)

	if ri == nil {
		t.Fatal("Expected non-nil RuntimeIntegrity")
	}

	if ri.config != config {
		t.Errorf("Expected config %v, got %v", config, ri.config)
	}

	if ri.records == nil {
		t.Error("Expected non-nil records")
	}

	if ri.checkInterval != config.CheckInterval {
		t.Errorf("Expected checkInterval %v, got %v", config.CheckInterval, ri.checkInterval)
	}
}

func TestNewRuntimeIntegrityWithNilConfig(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	if ri == nil {
		t.Fatal("Expected non-nil RuntimeIntegrity with nil config")
	}

	if ri.config == nil {
		t.Error("Expected non-nil config")
	}

	if ri.config.EnableHashCheck != true {
		t.Error("Expected EnableHashCheck to be true by default")
	}
}

func TestRuntimeIntegrityCheckIntegrity(t *testing.T) {
	config := model.NewIntegrityConfig()
	ri := NewRuntimeIntegrity(config)

	result, err := ri.CheckIntegrity()
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IsValid != true {
		t.Errorf("Expected IsValid to be true, got false")
	}

	if result.Status != model.IntegrityStatusOK {
		t.Errorf("Expected status %s, got %s", model.IntegrityStatusOK, result.Status)
	}

	if len(result.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(result.Violations))
	}
}

func TestRuntimeIntegrityRegisterCodeHash(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	testCode := []byte("test code for hashing")
	targetName := "test_script.js"

	ri.RegisterCodeHash(targetName, testCode)

	record, exists := ri.records.GetRecord(targetName)
	if !exists {
		t.Fatal("Expected record to exist after registration")
	}

	if record.TargetName != targetName {
		t.Errorf("Expected target name %s, got %s", targetName, record.TargetName)
	}

	if record.OriginalHash == "" {
		t.Error("Expected non-empty original hash")
	}

	if record.CurrentHash != record.OriginalHash {
		t.Error("Expected current hash to match original hash after registration")
	}
}

func TestRuntimeIntegrityVerifyCodeIntegrity(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	testCode := []byte("original test code")
	targetName := "test_script.js"

	ri.RegisterCodeHash(targetName, testCode)

	result, err := ri.VerifyCodeIntegrity(targetName, testCode)
	if err != nil {
		t.Fatalf("VerifyCodeIntegrity failed: %v", err)
	}

	if !result.IsValid {
		t.Error("Expected valid integrity for unchanged code")
	}

	modifiedCode := []byte("modified test code")
	result, err = ri.VerifyCodeIntegrity(targetName, modifiedCode)
	if err != nil {
		t.Fatalf("VerifyCodeIntegrity failed for modified code: %v", err)
	}

	if result.IsValid {
		t.Error("Expected invalid integrity for modified code")
	}

	if len(result.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(result.Violations))
	}

	if result.Violations[0].Type != model.ViolationCodeHashMismatch {
		t.Errorf("Expected violation type %s, got %s", model.ViolationCodeHashMismatch, result.Violations[0].Type)
	}
}

func TestRuntimeIntegrityVerifyCodeIntegrityNotRegistered(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	testCode := []byte("test code")
	_, err := ri.VerifyCodeIntegrity("nonexistent", testCode)

	if err == nil {
		t.Error("Expected error for unregistered target")
	}
}

func TestRuntimeIntegrityGenerateHash(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	data := []byte("test data for hashing")
	hash1 := ri.GenerateHash(data)
	hash2 := ri.GenerateHash(data)

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	if hash1 != hash2 {
		t.Error("Expected deterministic hash")
	}

	differentData := []byte("different data")
	hash3 := ri.GenerateHash(differentData)

	if hash1 == hash3 {
		t.Error("Expected different hash for different data")
	}
}

func TestRuntimeIntegrityVerifyHash(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	data := []byte("test data")
	expectedHash := ri.GenerateHash(data)

	if !ri.VerifyHash(data, expectedHash) {
		t.Error("Expected hash verification to succeed")
	}

	wrongHash := "wrong_hash_value"
	if ri.VerifyHash(data, wrongHash) {
		t.Error("Expected hash verification to fail for wrong hash")
	}
}

func TestRuntimeIntegrityAddAlertHandler(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	handler := &mockAlertHandler{}
	ri.AddAlertHandler(handler)

	if len(ri.alertHandlers) != 2 {
		t.Errorf("Expected 2 handlers (including default), got %d", len(ri.alertHandlers))
	}
}

func TestRuntimeIntegrityAddViolationCallback(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	violationCalled := false
	var receivedViolation model.IntegrityViolation

	callback := func(v model.IntegrityViolation) {
		violationCalled = true
		receivedViolation = v
	}

	ri.AddViolationCallback(callback)

	testCode := []byte("test")
	ri.RegisterCodeHash("test", testCode)

	modifiedCode := []byte("modified")
	ri.VerifyCodeIntegrity("test", modifiedCode)

	if !violationCalled {
		t.Error("Expected violation callback to be called")
	}

	_ = receivedViolation
}

func TestRuntimeIntegrityStartStop(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	err := ri.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !ri.running {
		t.Error("Expected running to be true after Start")
	}

	err = ri.Start()
	if err == nil {
		t.Error("Expected error when starting already running checker")
	}

	ri.Stop()

	if ri.running {
		t.Error("Expected running to be false after Stop")
	}
}

func TestRuntimeIntegrityGetSetConfig(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	newConfig := &model.IntegrityConfig{
		EnableHashCheck:        false,
		EnableDynamicCodeCheck: true,
		CheckInterval:          60 * time.Second,
	}

	ri.SetConfig(newConfig)

	retrievedConfig := ri.GetConfig()
	if retrievedConfig.EnableHashCheck != false {
		t.Error("Expected EnableHashCheck to be false after SetConfig")
	}

	if retrievedConfig.CheckInterval != 60*time.Second {
		t.Errorf("Expected CheckInterval 60s, got %v", retrievedConfig.CheckInterval)
	}
}

func TestRuntimeIntegrityGetStats(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	stats := ri.GetStats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.TotalChecks != 0 {
		t.Errorf("Expected 0 total checks initially, got %d", stats.TotalChecks)
	}

	testCode := []byte("test")
	ri.RegisterCodeHash("test", testCode)
	ri.CheckIntegrity()

	stats = ri.GetStats()
	if stats.TotalChecks == 0 {
		t.Error("Expected total checks to be updated")
	}
}

func TestRuntimeIntegrityGenerateIntegrityReport(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	report, err := ri.GenerateIntegrityReport("test-session-123")
	if err != nil {
		t.Fatalf("GenerateIntegrityReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if report.SessionID != "test-session-123" {
		t.Errorf("Expected session ID 'test-session-123', got '%s'", report.SessionID)
	}

	if report.OverallStatus != model.IntegrityStatusOK {
		t.Errorf("Expected status OK, got %s", report.OverallStatus)
	}

	if report.Summary == nil {
		t.Error("Expected non-nil summary")
	}

	if len(report.CheckResults) != 1 {
		t.Errorf("Expected 1 check result, got %d", len(report.CheckResults))
	}
}

func TestRuntimeIntegrityExportAlertAsJSON(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	jsonStr, err := ri.ExportAlertAsJSON()
	if err != nil {
		t.Fatalf("ExportAlertAsJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	var report model.IntegrityReport
	err = json.Unmarshal([]byte(jsonStr), &report)
	if err != nil {
		t.Fatalf("Failed to unmarshal exported JSON: %v", err)
	}
}

func TestRuntimeIntegrityGetRecord(t *testing.T) {
	ri := NewRuntimeIntegrity(nil)

	ri.RegisterCodeHash("test1", []byte("code1"))

	record, exists := ri.GetRecord("test1")
	if !exists {
		t.Fatal("Expected record to exist")
	}

	if record.TargetName != "test1" {
		t.Errorf("Expected target name 'test1', got '%s'", record.TargetName)
	}

	_, exists = ri.GetRecord("nonexistent")
	if exists {
		t.Error("Expected no record for nonexistent target")
	}
}

func TestHashBasedIntegrity(t *testing.T) {
	hi := NewHashBasedIntegrity(HashSHA256)

	data := []byte("test data")

	hash1 := hi.ComputeHash(data)
	hash2 := hi.ComputeHash(data)

	if hash1 != hash2 {
		t.Error("Expected deterministic hash")
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	hi2 := NewHashBasedIntegrity(HashSHA256)
	differentHash := hi2.ComputeHash([]byte("different data"))

	if hash1 == differentHash {
		t.Error("Expected different hash for different data")
	}
}

func TestHashBasedIntegrityWithSecretKey(t *testing.T) {
	hi := NewHashBasedIntegrity(HashHMACSHA256)

	secretKey := []byte("my-secret-key")
	hi.SetSecretKey(secretKey)

	data := []byte("test data")

	hash1 := hi.ComputeHash(data)
	hash2 := hi.ComputeHash(data)

	if hash1 != hash2 {
		t.Error("Expected deterministic HMAC hash")
	}

	hi.SetSecretKey([]byte("different-key"))
	hash3 := hi.ComputeHash(data)

	if hash1 == hash3 {
		t.Error("Expected different hash with different key")
	}
}

func TestHashBasedIntegrityVerifyHash(t *testing.T) {
	hi := NewHashBasedIntegrity(HashSHA256)

	data := []byte("test data")
	hash := hi.ComputeHash(data)

	if !hi.VerifyHash(data, hash) {
		t.Error("Expected hash verification to succeed")
	}

	if hi.VerifyHash(data, "invalid-hash") {
		t.Error("Expected hash verification to fail for invalid hash")
	}

	if hi.VerifyHash([]byte("different data"), hash) {
		t.Error("Expected hash verification to fail for different data")
	}
}

func TestHashBasedIntegrityGenerateIntegrityToken(t *testing.T) {
	hi := NewHashBasedIntegrity(HashHMACSHA256)

	data := []byte("test data")
	timestamp := time.Now().Unix()

	token := hi.GenerateIntegrityToken(data, timestamp)
	if token == "" {
		t.Error("Expected non-empty token")
	}

	maxAge := 1 * time.Hour
	if !hi.VerifyIntegrityToken(data, token, maxAge) {
		t.Error("Expected token verification to succeed within max age")
	}

	oldTimestamp := time.Now().Add(-2 * time.Hour).Unix()
	oldToken := hi.GenerateIntegrityToken(data, oldTimestamp)

	if hi.VerifyIntegrityToken(data, oldToken, maxAge) {
		t.Error("Expected token verification to fail for expired token")
	}
}

func TestHashBasedIntegrityGenerateIntegrityTokenDifferentData(t *testing.T) {
	hi := NewHashBasedIntegrity(HashHMACSHA256)

	data1 := []byte("data1")
	data2 := []byte("data2")
	timestamp := time.Now().Unix()

	token1 := hi.GenerateIntegrityToken(data1, timestamp)

	if hi.VerifyIntegrityToken(data2, token1, 1*time.Hour) {
		t.Error("Expected token verification to fail for different data")
	}
}

func TestThreadSafeIntegrityChecker(t *testing.T) {
	checker := model.NewThreadSafeIntegrityChecker()

	record := &model.CodeIntegrityRecord{
		TargetName:   "test.js",
		TargetType:   "code",
		OriginalHash: "abc123",
		CurrentHash:  "abc123",
	}

	checker.AddRecord(record)

	retrieved, exists := checker.GetRecord("test.js")
	if !exists {
		t.Fatal("Expected record to exist")
	}

	if retrieved.TargetName != "test.js" {
		t.Errorf("Expected target name 'test.js', got '%s'", retrieved.TargetName)
	}

	allRecords := checker.GetAllRecords()
	if len(allRecords) != 1 {
		t.Errorf("Expected 1 record, got %d", len(allRecords))
	}

	retrieved.CurrentHash = "modified"
	checker.UpdateRecord("test.js", retrieved)

	updated, _ := checker.GetRecord("test.js")
	if updated.CurrentHash != "modified" {
		t.Error("Expected record to be updated")
	}

	checker.DeleteRecord("test.js")
	_, exists = checker.GetRecord("test.js")
	if exists {
		t.Error("Expected record to be deleted")
	}
}

func TestThreadSafeIntegrityCheckerConcurrent(t *testing.T) {
	checker := model.NewThreadSafeIntegrityChecker()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			targetName := fmt.Sprintf("target_%d", id%10)

			record := &model.CodeIntegrityRecord{
				TargetName:   targetName,
				TargetType:   "code",
				OriginalHash: fmt.Sprintf("hash_%d", id),
				CurrentHash:  fmt.Sprintf("hash_%d", id),
			}
			checker.AddRecord(record)

			checker.GetRecord(targetName)
			checker.GetAllRecords()

			checker.UpdateStats("test_violation", id%2 == 0)

			stats := checker.GetStats()
			if stats != nil {
				_ = stats.TotalChecks
			}
		}(i)
	}

	wg.Wait()

	stats := checker.GetStats()
	if stats.TotalChecks != int64(numGoroutines) {
		t.Errorf("Expected %d total checks, got %d", numGoroutines, stats.TotalChecks)
	}

	if stats.PassedChecks+stats.FailedChecks != int64(numGoroutines) {
		t.Errorf("Expected %d passed+failed checks, got %d",
			numGoroutines, stats.PassedChecks+stats.FailedChecks)
	}
}

func TestIntegrityCheckResult(t *testing.T) {
	result := model.NewIntegrityCheckResult()

	if result.IsValid != true {
		t.Error("Expected IsValid to be true initially")
	}

	if result.Status != model.IntegrityStatusOK {
		t.Error("Expected IntegrityStatusOK status initially")
	}

	if len(result.Violations) != 0 {
		t.Error("Expected empty violations initially")
	}

	if result.Metadata == nil {
		t.Error("Expected non-nil metadata")
	}

	result.SetMetadata("key1", "value1")
	if result.Metadata["key1"] != "value1" {
		t.Error("Expected metadata to be set")
	}
}

func TestIntegrityCheckResultAddViolation(t *testing.T) {
	result := model.NewIntegrityCheckResult()

	violation := model.IntegrityViolation{
		Type:    model.ViolationCodeHashMismatch,
		Severity: 9,
		Target:  "test.js",
	}

	result.AddViolation(violation)

	if result.IsValid {
		t.Error("Expected IsValid to be false after adding violation")
	}

	if result.Status != model.IntegrityStatusModified {
		t.Error("Expected status to be modified")
	}

	if len(result.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(result.Violations))
	}
}

func TestIntegrityViolationSetGetMetadata(t *testing.T) {
	violation := model.IntegrityViolation{}

	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := violation.SetMetadata(data)
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	retrieved, err := violation.GetMetadata()
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if retrieved["key1"] != "value1" {
		t.Error("Expected key1 to be 'value1'")
	}

	if retrieved["key2"].(float64) != 123 {
		t.Error("Expected key2 to be 123")
	}

	if retrieved["key3"] != true {
		t.Error("Expected key3 to be true")
	}
}

func TestIntegrityViolationGetMetadataEmpty(t *testing.T) {
	violation := model.IntegrityViolation{}

	metadata, err := violation.GetMetadata()
	if err != nil {
		t.Fatalf("GetMetadata failed for empty violation: %v", err)
	}

	if len(metadata) != 0 {
		t.Error("Expected empty metadata")
	}
}

func TestNewIntegrityConfig(t *testing.T) {
	config := model.NewIntegrityConfig()

	if config.EnableHashCheck != true {
		t.Error("Expected EnableHashCheck to be true")
	}

	if config.EnableDynamicCodeCheck != true {
		t.Error("Expected EnableDynamicCodeCheck to be true")
	}

	if config.EnableMemoryCheck != true {
		t.Error("Expected EnableMemoryCheck to be true")
	}

	if config.CheckInterval != 30*time.Second {
		t.Errorf("Expected CheckInterval 30s, got %v", config.CheckInterval)
	}

	if config.AlertThreshold != 3 {
		t.Errorf("Expected AlertThreshold 3, got %d", config.AlertThreshold)
	}

	if len(config.AlertChannels) != 2 {
		t.Errorf("Expected 2 alert channels, got %d", len(config.AlertChannels))
	}
}

func TestNewIntegrityCheckResult(t *testing.T) {
	result := model.NewIntegrityCheckResult()

	if result.Violations == nil {
		t.Error("Expected non-nil violations slice")
	}

	if result.Metadata == nil {
		t.Error("Expected non-nil metadata")
	}

	if result.CheckedAt.IsZero() {
		t.Error("Expected non-zero CheckedAt")
	}
}

func TestComputeFileHash(t *testing.T) {
	tmpFile := "/tmp/integrity_test_file.txt"
	testContent := []byte("test content for file hashing")

	err := writeTestFile(tmpFile, testContent)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	defer osRemove(t, tmpFile)

	hash, err := ComputeFileHash(tmpFile, HashSHA256)
	if err != nil {
		t.Fatalf("ComputeFileHash failed: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	hash2, err := ComputeFileHash(tmpFile, HashSHA256)
	if err != nil {
		t.Fatalf("ComputeFileHash second call failed: %v", err)
	}

	if hash != hash2 {
		t.Error("Expected deterministic hash")
	}
}

func TestComputeFileHashFileNotFound(t *testing.T) {
	_, err := ComputeFileHash("/nonexistent/file/path", HashSHA256)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestVerifyFileIntegrity(t *testing.T) {
	tmpFile := "/tmp/integrity_verify_test.txt"
	testContent := []byte("content for integrity verification")

	err := writeTestFile(tmpFile, testContent)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	defer osRemove(t, tmpFile)

	expectedHash, err := ComputeFileHash(tmpFile, HashSHA256)
	if err != nil {
		t.Fatalf("ComputeFileHash failed: %v", err)
	}

	valid, err := VerifyFileIntegrity(tmpFile, expectedHash)
	if err != nil {
		t.Fatalf("VerifyFileIntegrity failed: %v", err)
	}

	if !valid {
		t.Error("Expected verification to succeed for correct hash")
	}

	valid, err = VerifyFileIntegrity(tmpFile, "invalid_hash")
	if err != nil {
		t.Fatalf("VerifyFileIntegrity failed for invalid hash: %v", err)
	}

	if valid {
		t.Error("Expected verification to fail for invalid hash")
	}
}

func TestGenerateMultiHash(t *testing.T) {
	data := []byte("multi-hash test data")

	hashes := GenerateMultiHash(data)

	if hashes == nil {
		t.Fatal("Expected non-nil hash map")
	}

	if len(hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(hashes))
	}

	if hashes[HashSHA256] == "" {
		t.Error("Expected non-empty SHA256 hash")
	}

	if hashes[HashHMACSHA256] == "" {
		t.Error("Expected non-empty HMAC hash")
	}

	if hashes[HashSHA256] == hashes[HashHMACSHA256] {
		t.Error("Expected different hashes for different algorithms")
	}
}

func TestIntegrityCheckResultCalculateHash(t *testing.T) {
	result := model.NewIntegrityCheckResult()

	data := []byte("test data for hash calculation")
	hash := result.CalculateHash(data)

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	hash2 := result.CalculateHash(data)
	if hash != hash2 {
		t.Error("Expected deterministic hash")
	}
}

type mockAlertHandler struct {
	alerts []*model.IntegrityAlert
}

func (h *mockAlertHandler) HandleAlert(alert *model.IntegrityAlert) error {
	h.alerts = append(h.alerts, alert)
	return nil
}

func TestLogAlertHandler(t *testing.T) {
	handler := NewLogAlertHandler()

	alert := &model.IntegrityAlert{
		ID:       "test-alert-1",
		Type:     model.ViolationCodeHashMismatch,
		Severity: 9,
		Target:   "test.js",
		Message:  "Hash mismatch detected",
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(alert)
	if err != nil {
		t.Fatalf("HandleAlert failed: %v", err)
	}
}

func TestWebhookAlertHandler(t *testing.T) {
	handler := NewWebhookAlertHandler("https://example.com/webhook")

	alert := &model.IntegrityAlert{
		ID:       "test-alert-2",
		Type:     model.ViolationDynamicCodeLoad,
		Severity: 7,
		Target:   "eval()",
		Message:  "Dynamic code loading detected",
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(alert)
	if err == nil {
		t.Log("Expected error due to unimplemented HTTP client")
	}
}

func TestIntegrityViolationHandler(t *testing.T) {
	handler := NewIntegrityViolationHandler()

	mockHandler := &mockAlertHandler{}
	handler.AddHandler(mockHandler)

	violation := &model.IntegrityViolation{
		Type:       model.ViolationMemoryModification,
		Severity:   8,
		Target:     "Object.prototype",
		Timestamp:  time.Now(),
	}

	handler.HandleViolation(violation)

	time.Sleep(10 * time.Millisecond)

	if len(mockHandler.alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(mockHandler.alerts))
	}
}

func TestAlertHandlerFunc(t *testing.T) {
	alertReceived := false
	var receivedAlert *model.IntegrityAlert

	handler := AlertHandlerFunc(func(alert *model.IntegrityAlert) error {
		alertReceived = true
		receivedAlert = alert
		return nil
	})

	alert := &model.IntegrityAlert{
		ID:       "test-alert-func",
		Type:     model.ViolationFunctionHook,
		Severity: 8,
		Target:   "eval",
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(alert)
	if err != nil {
		t.Fatalf("HandleAlert failed: %v", err)
	}

	if !alertReceived {
		t.Error("Expected alert to be received by handler function")
	}

	if receivedAlert.ID != alert.ID {
		t.Error("Expected received alert ID to match")
	}
}

func TestHashAlgorithmConstants(t *testing.T) {
	if HashSHA256 != "sha256" {
		t.Errorf("Expected HashSHA256 to be 'sha256', got '%s'", HashSHA256)
	}

	if HashSHA512 != "sha512" {
		t.Errorf("Expected HashSHA512 to be 'sha512', got '%s'", HashSHA512)
	}

	if HashHMACSHA256 != "hmac_sha256" {
		t.Errorf("Expected HashHMACSHA256 to be 'hmac_sha256', got '%s'", HashHMACSHA256)
	}
}

func TestIntegrityStatusConstants(t *testing.T) {
	if model.IntegrityStatusOK != "ok" {
		t.Errorf("Expected IntegrityStatusOK to be 'ok', got '%s'", model.IntegrityStatusOK)
	}

	if model.IntegrityStatusModified != "modified" {
		t.Errorf("Expected IntegrityStatusModified to be 'modified', got '%s'", model.IntegrityStatusModified)
	}

	if model.IntegrityStatusTampered != "tampered" {
		t.Errorf("Expected IntegrityStatusTampered to be 'tampered', got '%s'", model.IntegrityStatusTampered)
	}
}

func TestIntegrityViolationTypeConstants(t *testing.T) {
	expectedTypes := map[model.IntegrityViolationType]string{
		model.ViolationCodeHashMismatch:          "code_hash_mismatch",
		model.ViolationDynamicCodeLoad:            "dynamic_code_load",
		model.ViolationMemoryModification:        "memory_modification",
		model.ViolationFunctionHook:              "function_hook",
		model.ViolationPrototypeModification:      "prototype_modification",
		model.ViolationObjectPropertyChange:       "object_property_change",
	}

	for vtype, expected := range expectedTypes {
		if string(vtype) != expected {
			t.Errorf("Expected %s, got %s", expected, vtype)
		}
	}
}

func writeTestFile(path string, content []byte) error {
	f, err := osCreate(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}

func osRemove(t *testing.T, path string) {
	t.Helper()
	err := os.Remove(path)
	if err != nil {
		t.Logf("Failed to remove test file: %v", err)
	}
}

func osCreate(path string) (*os.File, error) {
	return os.Create(path)
}

var _ model.IntegrityStatus = model.IntegrityStatus("")
var _ model.IntegrityStatus = model.IntegrityStatusOK
