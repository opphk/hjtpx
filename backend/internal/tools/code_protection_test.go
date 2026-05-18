package tools

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestCryptoServiceCreation(t *testing.T) {
	crypto := NewCryptoService()
	if crypto == nil {
		t.Fatal("Expected crypto service to be created")
	}
	if crypto.secretKey == nil {
		t.Error("Secret key should be set")
	}
}

func TestCryptoServiceWithCustomKey(t *testing.T) {
	key := []byte("test-key-1234567890")
	crypto := NewCryptoService(key)
	if string(crypto.secretKey) != string(key) {
		t.Error("Custom key should be set")
	}
}

func TestCryptoServiceEncryptDecryptString(t *testing.T) {
	crypto := NewCryptoService()
	original := "Hello, World! 你好世界！"

	encrypted, err := crypto.EncryptString(original)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	if encrypted == original {
		t.Error("Encrypted string should differ from original")
	}

	decrypted, err := crypto.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("Decrypted string should match original, got %q", decrypted)
	}
}

func TestDecryptedParamsTypeHandling(t *testing.T) {
	crypto := NewCryptoService()
	params := map[string]interface{}{
		"username": "testuser",
		"password": "testpass123",
		"age":      float64(25),
		"active":   true,
	}

	encrypted, err := crypto.EncryptParams(params)
	if err != nil {
		t.Fatalf("EncryptParams failed: %v", err)
	}

	decrypted, err := crypto.DecryptParams(encrypted)
	if err != nil {
		t.Fatalf("DecryptParams failed: %v", err)
	}

	if decrypted["username"] != params["username"] {
		t.Error("Username should match")
	}
	if decrypted["password"] != params["password"] {
		t.Error("Password should match")
	}
	age, ok := decrypted["age"].(float64)
	if !ok {
		t.Error("Age should be float64 after JSON unmarshal")
	}
	if age != float64(25) {
		t.Error("Age should match")
	}
}

func TestCryptoServiceDecryptInvalidData(t *testing.T) {
	crypto := NewCryptoService()
	_, err := crypto.DecryptString("invalid-base64!")
	if err == nil {
		t.Error("Should return error for invalid base64")
	}
}

func TestCryptoServiceDecryptWrongKey(t *testing.T) {
	crypto1 := NewCryptoService([]byte("key1-1234567890ab"))
	crypto2 := NewCryptoService([]byte("key2-1234567890ab"))

	encrypted, err := crypto1.EncryptString("test data")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = crypto2.DecryptString(encrypted)
	if err == nil {
		t.Error("Decryption with wrong key should fail")
	}
}

func TestCryptoServiceSetSecretKey(t *testing.T) {
	crypto := NewCryptoService()
	newKey := []byte("new-key-1234567890")
	crypto.SetSecretKey(newKey)

	if string(crypto.secretKey) != string(newKey) {
		t.Error("Secret key should be updated")
	}
}

func TestProtectorCreation(t *testing.T) {
	protector := NewProtector()
	if protector == nil {
		t.Fatal("Expected protector to be created")
	}
	if protector.obfuscator == nil {
		t.Error("Obfuscator should be set")
	}
	if protector.crypto == nil {
		t.Error("Crypto service should be set")
	}
}

func TestProtectorProtect(t *testing.T) {
	protector := NewProtector()
	code := `function hello() { return "world"; }`

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Protect failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}

	if !strings.Contains(protected, "(function()") {
		t.Error("Protected code should be wrapped in IIFE")
	}

	if !strings.Contains(protected, "window.outerWidth") {
		t.Error("Protected code should include anti-debug code")
	}
}

func TestProtectorProtectWithLevel(t *testing.T) {
	protector := NewProtector()

	code := `var myVar = "test value";`

	result1, err := protector.ProtectWithLevel(code, 1)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}
	if result1 == "" {
		t.Error("Protected code should not be empty at level 1")
	}

	result3, err := protector.ProtectWithLevel(code, 3)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}
	if result3 == "" {
		t.Error("Protected code should not be empty at level 3")
	}
}

func TestProtectorEncryptAndProtect(t *testing.T) {
	protector := NewProtector()
	code := `function test() { return true; }`

	encrypted, err := protector.EncryptAndProtect(code)
	if err != nil {
		t.Fatalf("EncryptAndProtect failed: %v", err)
	}

	if encrypted == "" {
		t.Error("Encrypted code should not be empty")
	}
}

func TestProtectorGenerateIntegrityCheck(t *testing.T) {
	protector := NewProtector()
	code := `function test() { return true; }`

	check := protector.GenerateIntegrityCheck(code)
	if check == "" {
		t.Error("Integrity check should not be empty")
	}

	if !strings.Contains(check, "window.__h=") && !strings.Contains(check, "__h") {
		t.Error("Integrity check should include hash")
	}
}

func TestProtectorVerifyIntegrity(t *testing.T) {
	protector := NewProtector()
	code := `function test() { return true; }`

	hash := sha256HashForTest(code)

	valid := protector.VerifyIntegrity(code, hash)
	if !valid {
		t.Error("Code should verify against its hash")
	}

	invalid := protector.VerifyIntegrity(code+"modified", hash)
	if invalid {
		t.Error("Modified code should not verify")
	}
}

func sha256HashForTest(code string) string {
	hash := sha256.Sum256([]byte(code))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func TestParameterProtectorCreation(t *testing.T) {
	protector := NewParameterProtector()
	if protector == nil {
		t.Fatal("Expected parameter protector to be created")
	}
}

func TestParameterProtectorEncryptDecrypt(t *testing.T) {
	protector := NewParameterProtector()
	params := map[string]interface{}{
		"action": "login",
		"user":   "testuser",
	}

	encrypted, err := protector.EncryptRequestParams(params)
	if err != nil {
		t.Fatalf("EncryptRequestParams failed: %v", err)
	}

	if encrypted["data"] == "" {
		t.Error("Encrypted data should not be empty")
	}

	decrypted, err := protector.DecryptRequestParams(encrypted["data"])
	if err != nil {
		t.Fatalf("DecryptRequestParams failed: %v", err)
	}

	if decrypted["action"] != "login" {
		t.Errorf("Action should be login, got %v", decrypted["action"])
	}
	if decrypted["user"] != "testuser" {
		t.Errorf("User should be testuser, got %v", decrypted["user"])
	}
}

func TestParameterProtectorDecryptInvalidData(t *testing.T) {
	protector := NewParameterProtector()
	_, err := protector.DecryptRequestParams("invalid-data!!!")
	if err == nil {
		t.Error("Should return error for invalid encrypted data")
	}
}

func TestSignatureValidatorCreation(t *testing.T) {
	key := []byte("test-secret-key-12345678")
	validator := NewSignatureValidator(key)
	if validator == nil {
		t.Fatal("Expected signature validator to be created")
	}
}

func TestSignatureValidatorSetTimestampTTL(t *testing.T) {
	key := []byte("test-secret-key-12345678")
	validator := NewSignatureValidator(key)
	validator.SetTimestampTTL(600)

	if validator.timestampTTL != 600 {
		t.Error("Timestamp TTL should be updated")
	}
}

func TestSignatureValidatorGenerateSignature(t *testing.T) {
	key := []byte("test-secret-key-12345678")
	validator := NewSignatureValidator(key)

	sig, ts, nonce, err := validator.GenerateSignature("POST", "/api/test", []byte(`{"data":"test"}`))
	if err != nil {
		t.Fatalf("GenerateSignature failed: %v", err)
	}

	if sig == "" {
		t.Error("Signature should not be empty")
	}
	if ts == "" {
		t.Error("Timestamp should not be empty")
	}
	if nonce == "" {
		t.Error("Nonce should not be empty")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	bytes1, err := GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("GenerateRandomBytes failed: %v", err)
	}
	if len(bytes1) != 16 {
		t.Errorf("Expected 16 bytes, got %d", len(bytes1))
	}

	bytes2, _ := GenerateRandomBytes(16)
	if string(bytes1) == string(bytes2) {
		t.Error("Generated bytes should be unique")
	}
}

func TestGenerateRandomStringForCrypto(t *testing.T) {
	str := GenerateRandomString(32)
	if len(str) != 32 {
		t.Errorf("Expected 32 chars, got %d", len(str))
	}

	str2 := GenerateRandomString(32)
	if str == str2 {
		t.Error("Generated strings should be unique")
	}
}

func TestMaskSensitiveData(t *testing.T) {
	{
		result := MaskSensitiveData("password123", 2)
		if !strings.HasPrefix(result, "pa") {
			t.Errorf("Should start with visible chars, got %s", result)
		}
		if !strings.HasSuffix(result, strings.Repeat("*", len(result)-2)) {
			t.Errorf("Should end with stars, got %s", result)
		}
	}

	{
		result := MaskSensitiveData("secret", 0)
		if result != strings.Repeat("*", 6) {
			t.Errorf("Should be all stars, got %s", result)
		}
	}

	{
		result := MaskSensitiveData("abc", 3)
		if result != "***" {
			t.Errorf("Should be all stars, got %s", result)
		}
	}

	{
		result := MaskSensitiveData("", 2)
		if result != "" {
			t.Errorf("Empty string should return empty, got %s", result)
		}
	}
}

func TestSanitizeLogOutput(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{`password="secret123"`, "REDACTED"},
		{`token="abc123"`, "REDACTED"},
		{`username="user"`, "user"},
	}

	for _, tc := range tests {
		result := SanitizeLogOutput(tc.input)
		if tc.contains == "REDACTED" {
			if strings.Contains(result, "secret") || strings.Contains(result, "abc") {
				t.Errorf("Sensitive data should be redacted in %q", result)
			}
		}
	}
}

func TestFullEncryptionDecryptionCycle(t *testing.T) {
	key := []byte("test-key-for-full-cycle")
	crypto := NewCryptoService(key)

	testData := map[string]interface{}{
		"string":   "Hello, World! 你好！",
		"number":   42,
		"float":    3.14159,
		"boolean":  true,
		"null":     nil,
		"array":    []int{1, 2, 3},
		"nested":   map[string]interface{}{"key": "value"},
		"unicode":  "日本語テスト",
		"special":  "!@#$%^&*()",
		"longtext": strings.Repeat("Lorem ipsum ", 100),
	}

	encrypted, err := crypto.EncryptParams(testData)
	if err != nil {
		t.Fatalf("EncryptParams failed: %v", err)
	}

	decrypted, err := crypto.DecryptParams(encrypted)
	if err != nil {
		t.Fatalf("DecryptParams failed: %v", err)
	}

	jsonOriginal, _ := json.Marshal(testData)
	jsonDecrypted, _ := json.Marshal(decrypted)

	if string(jsonOriginal) != string(jsonDecrypted) {
		t.Error("Decrypted data should match original")
	}
}

func TestProtectorWithDifferentKeys(t *testing.T) {
	key1 := []byte("key-one-1234567890ab")
	key2 := []byte("key-two-1234567890ab")

	protector1 := NewProtector(key1)
	protector2 := NewProtector(key2)

	code := `var secret = "password123";`

	encrypted1, err := protector1.crypto.EncryptString(code)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = protector2.crypto.DecryptString(encrypted1)
	if err == nil {
		t.Error("Decryption with different key should fail")
	}

	decrypted, err := protector1.crypto.DecryptString(encrypted1)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != code {
		t.Error("Decrypted code should match original")
	}
}

func TestNonceTracking(t *testing.T) {
	crypto := NewCryptoService()

	nonce1 := GenerateRandomString(16)
	nonce2 := GenerateRandomString(16)

	if crypto.IsNonceUsed(nonce1) {
		t.Error("New nonce should not be marked as used")
	}

	crypto.MarkNonceUsed(nonce1)

	if !crypto.IsNonceUsed(nonce1) {
		t.Error("Marked nonce should be marked as used")
	}

	if crypto.IsNonceUsed(nonce2) {
		t.Error("Different nonce should not be marked as used")
	}
}

func TestParameterProtectorWithCustomKey(t *testing.T) {
	key := []byte("custom-encryption-key-12")
	protector := NewParameterProtector(key)

	params := map[string]interface{}{
		"data": "sensitive",
	}

	encrypted, err := protector.EncryptRequestParams(params)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := protector.DecryptRequestParams(encrypted["data"])
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted["data"] != "sensitive" {
		t.Errorf("Data should be 'sensitive', got %v", decrypted["data"])
	}
}

func TestMultipleEncryptDecryptCycles(t *testing.T) {
	crypto := NewCryptoService()

	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"iteration": i,
			"data":      GenerateRandomString(32),
		}

		encrypted, err := crypto.EncryptParams(data)
		if err != nil {
			t.Fatalf("Encrypt failed at iteration %d: %v", i, err)
		}

		decrypted, err := crypto.DecryptParams(encrypted)
		if err != nil {
			t.Fatalf("Decrypt failed at iteration %d: %v", i, err)
		}

		if decrypted["iteration"].(float64) != float64(i) {
			t.Error("Iteration should match")
		}
	}
}
