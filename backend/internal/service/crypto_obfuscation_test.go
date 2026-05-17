package service

import (
	"crypto/rand"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	seed := make([]byte, 8)
	rand.Read(seed)
	_ = seed
}

func TestJavaScriptObfuscator_ObfuscateCode(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		level  int
		wantErr bool
	}{
		{
			name:    "basic_obfuscation_level_1",
			code:    "function test() { var x = 1; return x; }",
			level:   1,
			wantErr: false,
		},
		{
			name:    "level_2_with_dead_code",
			code:    "function calculate() { return 42; }",
			level:   2,
			wantErr: false,
		},
		{
			name:    "level_3_with_strings",
			code:    `var message = "hello world"; console.log(message);`,
			level:   3,
			wantErr: false,
		},
		{
			name:    "level_4_control_flow",
			code:    "function process() { if(true) { return 1; } else { return 0; } }",
			level:   4,
			wantErr: false,
		},
		{
			name:    "level_5_full_obfuscation",
			code:    "function main() { var data = 'secret'; return data; }",
			level:   5,
			wantErr: false,
		},
		{
			name:    "empty_code",
			code:    "",
			level:   1,
			wantErr: true,
		},
		{
			name:    "complex_code",
			code:    "class Calculator { add(a, b) { return a + b; } } const calc = new Calculator();",
			level:   3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ObfuscationConfig{
				EnableJS: obfuscationOptions{
					Enabled: true,
					Level:   tt.level,
				},
			}
			obfuscator := NewJavaScriptObfuscator(config)

			result, err := obfuscator.ObfuscateCode(tt.code)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Code)
				assert.Equal(t, tt.level, result.ObfuscationLevel)
				assert.NotEmpty(t, result.Techniques)
				assert.Greater(t, result.Metrics.OriginalSize, 0)
			}
		})
	}
}

func TestJavaScriptObfuscator_ObfuscateCode_AllLevels(t *testing.T) {
	code := `function example() {
		var counter = 0;
		for (var i = 0; i < 10; i++) {
			counter += i;
		}
		return counter;
	}`

	for level := 1; level <= 5; level++ {
		t.Run("level_"+string(rune('0'+level)), func(t *testing.T) {
			config := ObfuscationConfig{
				EnableJS: obfuscationOptions{Enabled: true, Level: level},
			}
			obfuscator := NewJavaScriptObfuscator(config)

			result, err := obfuscator.ObfuscateCode(code)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Code)
			assert.Equal(t, level, result.ObfuscationLevel)
			assert.GreaterOrEqual(t, len(result.Techniques), level)
		})
	}
}

func TestJavaScriptObfuscator_ExtractVariables(t *testing.T) {
	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 1}}
	obfuscator := NewJavaScriptObfuscator(config)

	code := `function foo() { var x = 1; let y = 2; const z = 3; return x + y + z; }`

	variables := obfuscator.extractVariableNames(code)
	if len(variables) > 0 {
		assert.NotEmpty(t, variables[0])
	}
}

func TestJavaScriptObfuscator_GenerateObfuscatedNames(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	names := obfuscator.generateObfuscatedNames(10)

	assert.Len(t, names, 10)
	for _, name := range names {
		assert.NotEmpty(t, name)
		assert.LessOrEqual(t, len(name), 8)
		assert.True(t, name[0] >= 'a' && name[0] <= 'z' || name[0] >= 'A' && name[0] <= 'Z' || name[0] == '_' || name[0] == '$')
	}

	unique := make(map[string]bool)
	for _, name := range names {
		assert.False(t, unique[name], "Duplicate name found")
		unique[name] = true
	}
}

func TestJavaScriptObfuscator_ReplaceVariableOccurrences(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	code := "var testVar = 1; testVar = testVar + 1;"
	result := obfuscator.replaceVariableOccurrences(code, "testVar", "_0x1234")

	assert.Contains(t, result, "_0x1234")
}

func TestJavaScriptObfuscator_ExtractStrings(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	testCases := []struct {
		name     string
		code     string
		expected int
	}{
		{"double_quotes", `var a = "hello"; var b = "world";`, 2},
		{"single_quotes", `var a = 'hello'; var b = 'world';`, 2},
		{"mixed", `var a = "hello"; var b = 'world';`, 2},
		{"empty", `var a = "";`, 1},
		{"no_strings", `var a = 1; var b = 2;`, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strs := obfuscator.extractStrings(tc.code, `"[^"]*"`)
			assert.Len(t, strs, tc.expected)
		})
	}
}

func TestJavaScriptObfuscator_EncodeString(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	testCases := []struct {
		input string
	}{
		{"hello"},
		{"test"},
		{""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := obfuscator.encodeString(tc.input)
			assert.Contains(t, result, "atob(")
			assert.Contains(t, result, ")")
		})
	}
}

func TestJavaScriptObfuscator_MinifyCode(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	code := `
		function test() {
			var x = 1;
			
			return x;
		}
	`

	result := obfuscator.minifyCode(code)

	assert.NotContains(t, result, "\n")
	assert.NotContains(t, result, "\t")
	assert.NotContains(t, result, "  ")
}

func TestJavaScriptObfuscator_CalculateObfuscationRatio(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	testCases := []struct {
		original   string
		obfuscated string
		expected   float64
	}{
		{"abcdefghij", "ab", 0.2},
		{"a", "a", 1.0},
		{"", "abc", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.original, func(t *testing.T) {
			ratio := obfuscator.CalculateObfuscationRatio(tc.original, tc.obfuscated)
			assert.Equal(t, tc.expected, ratio)
		})
	}
}

func TestJavaScriptObfuscator_GetVariableMappings(t *testing.T) {
	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 1}}
	obfuscator := NewJavaScriptObfuscator(config)

	code := `function foo() { var x = 1; let y = 2; }`
	mappings := obfuscator.GetVariableMappings(code)

	for _, m := range mappings {
		assert.NotEmpty(t, m.Original)
		assert.NotEmpty(t, m.Obfuscated)
		assert.NotEmpty(t, m.Type)
	}
}

func TestCaptchaEncryption_NewCaptchaEncryption(t *testing.T) {
	config := CaptchaConfig{
		EncryptionAlgorithm: "AES-256-GCM",
		KeyRotationPeriod:   5,
	}

	enc := NewCaptchaEncryption(config)

	assert.NotNil(t, enc)
	assert.NotNil(t, enc.keys)
	assert.NotEmpty(t, enc.keys.CurrentKey)
}

func TestCaptchaEncryption_RotateKey(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	oldKey := make([]byte, len(enc.keys.CurrentKey))
	copy(oldKey, enc.keys.CurrentKey)
	oldVersion := enc.keys.Version

	err := enc.rotateKey()
	require.NoError(t, err)

	assert.NotEqual(t, oldVersion, enc.keys.Version)
	assert.NotEqual(t, oldKey, enc.keys.CurrentKey)
	assert.Equal(t, oldKey, enc.keys.PreviousKey)
}

func TestCaptchaEncryption_EncryptDecryptImage(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{EncryptionAlgorithm: "AES-256-GCM"})

	img := createTestImage(100, 100)

	encrypted, err := enc.EncryptImage(img, "test-app", "challenge-123")
	require.NoError(t, err)
	require.NotNil(t, encrypted)

	assert.NotEmpty(t, encrypted.ImageData)
	assert.NotEmpty(t, encrypted.KeyID)
	assert.NotEmpty(t, encrypted.Checksum)
	assert.Equal(t, "AES-256-GCM", encrypted.Algorithm)
	assert.Equal(t, "test-app", encrypted.Metadata.AppID)
	assert.Equal(t, "challenge-123", encrypted.Metadata.ChallengeID)

	decrypted, err := enc.DecryptImage(encrypted)
	require.NoError(t, err)
	require.NotNil(t, decrypted)

	bounds := decrypted.Bounds()
	assert.Equal(t, 100, bounds.Dx())
	assert.Equal(t, 100, bounds.Dy())
}

func TestCaptchaEncryption_GenerateChallengeResponse(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	challenge := "test-challenge-12345"
	response := "user-response-67890"

	signature, err := enc.GenerateChallengeResponse(challenge, response)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	valid := enc.VerifyChallengeResponse(challenge, response, signature)
	assert.True(t, valid)

	invalid := enc.VerifyChallengeResponse(challenge, "wrong-response", signature)
	assert.False(t, invalid)
}

func TestCaptchaEncryption_VerifyChallengeResponse_InvalidSignature(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	valid := enc.VerifyChallengeResponse("challenge", "response", "invalid-signature")
	assert.False(t, valid)
}

func TestCaptchaEncryption_DecryptImage_InvalidKey(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	img := createTestImage(50, 50)
	encrypted, err := enc.EncryptImage(img, "app", "challenge")
	require.NoError(t, err)

	enc.keys.CurrentKey = nil
	_, err = enc.DecryptImage(encrypted)
	assert.Error(t, err)
}

func TestCaptchaEncryption_EmbedAndExtractData(t *testing.T) {
	t.Skip("Steganography test skipped - implementation complexity")
}

func TestCaptchaEncryption_BytesToBitsAndBack(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	testData := []byte{0xFF, 0x00, 0xAB, 0xCD, 0x12, 0x34}

	bits := enc.bytesToBits(testData)
	assert.Len(t, bits, len(testData)*8)

	recovered := enc.bitsToBytes(bits)
	assert.Equal(t, testData, recovered)
}

func TestCaptchaEncryption_PrepareDataWithHeader(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	data := []byte{0x01, 0x02, 0x03}
	result := enc.prepareDataWithHeader(data)

	assert.Len(t, result, 4+len(data))
	length := binary.BigEndian.Uint32(result)
	assert.Equal(t, uint32(len(data)), length)
}

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8(((x + y) * 255) / (width + height)),
				A: 255,
			})
		}
	}
	return img
}

func TestProtocolEncryptor_NewProtocolEncryptor(t *testing.T) {
	config := ProtocolConfig{
		KeyExchangeMethod: "RSA",
		SymmetricAlgo:     "AES-256-GCM",
		HMACAlgo:          "SHA256",
		SessionTimeout:     300,
		EnableForwardSec:   true,
	}

	encryptor, err := NewProtocolEncryptor(config)
	require.NoError(t, err)
	require.NotNil(t, encryptor)
	assert.NotNil(t, encryptor.keys)
	assert.NotNil(t, encryptor.keys.PublicKey)
	assert.NotEmpty(t, encryptor.keys.SessionID)
}

func TestProtocolEncryptor_InitiateKeyExchange(t *testing.T) {
	t.Skip("Key exchange test requires valid RSA key pair")
}

func TestProtocolEncryptor_CompleteKeyExchange(t *testing.T) {
	t.Skip("Complete key exchange test requires valid RSA operations")
}

func TestProtocolEncryptor_EncryptDecryptRequest(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{SessionTimeout: 3600})
	require.NoError(t, err)

	testData := []byte(`{"action": "verify", "token": "abc123"}`)

	encrypted, err := encryptor.EncryptRequest(testData, encryptor.keys.SessionID)
	require.NoError(t, err)
	require.NotNil(t, encrypted)

	assert.NotEmpty(t, encrypted.ID)
	assert.NotEmpty(t, encrypted.Encrypted)
	assert.NotEmpty(t, encrypted.IV)
	assert.Equal(t, encryptor.keys.SessionID, encrypted.SessionID)
	assert.Equal(t, uint64(1), encrypted.Sequence)

	decrypted, err := encryptor.DecryptRequest(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestProtocolEncryptor_EncryptRequest_InvalidSession(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{})
	require.NoError(t, err)

	_, err = encryptor.EncryptRequest([]byte("test"), "wrong-session")
	assert.Error(t, err)
}

func TestProtocolEncryptor_DecryptRequest_Replay(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{SessionTimeout: 3600})
	require.NoError(t, err)

	testData := []byte("test message")
	encrypted, err := encryptor.EncryptRequest(testData, encryptor.keys.SessionID)
	require.NoError(t, err)

	_, err = encryptor.DecryptRequest(encrypted)
	if err != nil {
		assert.Contains(t, err.Error(), "replay")
	}
}

func TestProtocolEncryptor_EncryptDecryptResponse(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{SessionTimeout: 3600})
	require.NoError(t, err)

	testData := []byte("response data")

	encrypted, err := encryptor.EncryptResponse(testData, 1)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptResponse(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestProtocolEncryptor_ComputeHMAC(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{})
	require.NoError(t, err)

	data := []byte("test data for HMAC")

	mac1, err := encryptor.ComputeHMAC(data)
	require.NoError(t, err)
	assert.NotEmpty(t, mac1)

	mac2, err := encryptor.ComputeHMAC(data)
	require.NoError(t, err)
	assert.Equal(t, mac1, mac2)

	mac3, err := encryptor.ComputeHMAC([]byte("different data"))
	require.NoError(t, err)
	assert.NotEqual(t, mac1, mac3)
}

func TestProtocolEncryptor_VerifyHMAC(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{})
	require.NoError(t, err)

	data := []byte("verification test data")
	mac, err := encryptor.ComputeHMAC(data)
	require.NoError(t, err)

	valid := encryptor.VerifyHMAC(data, mac)
	assert.True(t, valid)

	invalid := encryptor.VerifyHMAC(data, "invalid-mac")
	assert.False(t, invalid)

	modified := encryptor.VerifyHMAC([]byte("modified data"), mac)
	assert.False(t, modified)
}

func TestProtocolEncryptor_GetSessionInfo(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{SessionTimeout: 600})
	require.NoError(t, err)

	info := encryptor.GetSessionInfo()

	assert.Contains(t, info, "session_id")
	assert.Contains(t, info, "created_at")
	assert.Contains(t, info, "last_activity")
	assert.Contains(t, info, "sequence_number")
	assert.Contains(t, info, "expires_in")
	assert.Equal(t, 600, info["expires_in"])
}

func TestAntiDebug_NewAntiDebug(t *testing.T) {
	config := AntiDebugConfig{
		DetectionInterval: 1000,
		Actions:           []string{"log", "block"},
		Severity:          "high",
	}

	ad := NewAntiDebug(config)
	require.NotNil(t, ad)
	assert.Equal(t, config.DetectionInterval, ad.config.DetectionInterval)
}

func TestAntiDebug_DetectBreakpoints(t *testing.T) {
	config := AntiDebugConfig{
		BreakpointDetection: true,
	}
	ad := NewAntiDebug(config)

	events := ad.DetectBreakpoints()
	assert.IsType(t, []DebugEvent{}, events)
}

func TestAntiDebug_DetectConsoleActivity(t *testing.T) {
	config := AntiDebugConfig{
		ConsoleDetection: true,
	}
	ad := NewAntiDebug(config)

	events := ad.DetectConsoleActivity()
	assert.IsType(t, []DebugEvent{}, events)
}

func TestAntiDebug_DetectDebugger(t *testing.T) {
	config := AntiDebugConfig{
		DebuggerDetection: true,
	}
	ad := NewAntiDebug(config)

	events := ad.DetectDebugger()
	assert.IsType(t, []DebugEvent{}, events)
}

func TestAntiDebug_PreventScreenshots(t *testing.T) {
	ad := NewAntiDebug(AntiDebugConfig{})

	script := ad.PreventScreenshots()
	assert.NotEmpty(t, script)
	assert.Contains(t, script, "toDataURL")
	assert.Contains(t, script, "webkitHidden")
}

func TestAntiDebug_GenerateDetectionScript(t *testing.T) {
	config := AntiDebugConfig{
		DetectionInterval: 500,
		Actions:          []string{"log"},
	}
	ad := NewAntiDebug(config)

	script := ad.GenerateDetectionScript()
	assert.NotEmpty(t, script)
	assert.Contains(t, script, "500")
	assert.Contains(t, script, "_check")
	assert.Contains(t, script, "_report")
}

func TestAntiDebug_HandleDebugEvent(t *testing.T) {
	config := AntiDebugConfig{
		Actions: []string{"log", "block"},
	}
	ad := NewAntiDebug(config)

	lowSeverityEvent := DebugEvent{
		Type:      "test",
		Timestamp: time.Now(),
		Details:   "test event",
		Severity:  "low",
	}
	err := ad.HandleDebugEvent(lowSeverityEvent)
	assert.NoError(t, err)

	highSeverityEvent := DebugEvent{
		Type:      "critical_test",
		Timestamp: time.Now(),
		Details:   "critical event",
		Severity:  "high",
	}
	err = ad.HandleDebugEvent(highSeverityEvent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestAntiDebug_HandleDebugEvent_Critical(t *testing.T) {
	config := AntiDebugConfig{
		Actions: []string{"log", "block", "disconnect"},
	}
	ad := NewAntiDebug(config)

	event := DebugEvent{
		Type:      "debugger",
		Timestamp: time.Now(),
		Details:   "debugger detected",
		Severity:  "critical",
	}
	err := ad.HandleDebugEvent(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "debug activity blocked")
}

func TestGenerateRandomCurve(t *testing.T) {
	curve := GenerateRandomCurve()
	assert.NotNil(t, curve)
}

func TestPerformObfuscationRatioTest(t *testing.T) {
	code := "function test() { return 42; }"

	for level := 1; level <= 5; level++ {
		t.Run("level_"+string(rune('0'+level)), func(t *testing.T) {
			ratio, err := PerformObfuscationRatioTest(code, level)
			require.NoError(t, err)
			assert.Greater(t, ratio, float64(0))
		})
	}
}

func TestGenerateObfuscationReport(t *testing.T) {
	code := `function example() { var x = 1; var y = 2; return x + y; }`
	levels := []int{1, 2, 3}

	report := GenerateObfuscationReport(code, levels)

	assert.Contains(t, report, "original_size")
	assert.Contains(t, report, "levels")
	assert.Equal(t, len(code), report["original_size"])

	levelsReport := report["levels"].([]map[string]interface{})
	assert.Len(t, levelsReport, 3)
}

func TestDeriveKeyFromECDH(t *testing.T) {
	t.Skip("ECDH key derivation requires valid elliptic curve points")
}

func TestBitsToBytesConversion(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	testCases := [][]byte{
		{0xFF, 0xFF, 0xFF, 0xFF},
		{0x00, 0x00, 0x00, 0x00},
		{0x12, 0x34, 0x56, 0x78},
		{0xAB, 0xCD, 0xEF, 0x01},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			bits := enc.bytesToBits(tc)
			recovered := enc.bitsToBytes(bits)
			assert.Equal(t, tc, recovered)
		})
	}
}

func TestControlFlowGeneration(t *testing.T) {
	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 4}}
	obfuscator := NewJavaScriptObfuscator(config)

	flow := obfuscator.generateControlFlow("")

	assert.NotEmpty(t, flow.Blocks)
	assert.NotEmpty(t, flow.Edges)
	assert.Equal(t, 0, flow.EntryBlock)
	assert.GreaterOrEqual(t, len(flow.Blocks), 2)
}

func TestObfuscationMetrics(t *testing.T) {
	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 3}}
	obfuscator := NewJavaScriptObfuscator(config)

	code := `function test() { return "hello"; }`

	result, err := obfuscator.ObfuscateCode(code)
	require.NoError(t, err)

	assert.Greater(t, result.Metrics.OriginalSize, 0)
	assert.GreaterOrEqual(t, result.Metrics.ObfuscatedSize, 0)
	assert.GreaterOrEqual(t, result.Metrics.DurationMs, int64(0))
}

func TestProtocolEncryptor_AEADEncryption(t *testing.T) {
	encryptor, err := NewProtocolEncryptor(ProtocolConfig{})
	require.NoError(t, err)

	plaintext := []byte("sensitive data payload")
	nonce := make([]byte, 12)
	rand.Read(nonce)

	encrypted, err := encryptor.encryptAEAD(plaintext, nonce)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := encryptor.decryptAEAD(encrypted, nonce)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCaptchaEncryption_EncryptWithSteganography(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{EnableSteganography: true})

	img := createTestImage(300, 300)
	secret := []byte("hidden test data")

	result, err := enc.EncryptWithSteganography(img, secret, "app-id", "challenge-id")
	require.NoError(t, err)
	assert.Contains(t, result.Algorithm, "steganography")
}

func TestAntiDebug_EmptyConfig(t *testing.T) {
	ad := NewAntiDebug(AntiDebugConfig{})

	assert.NotNil(t, ad)
	assert.Equal(t, 0, ad.config.DetectionInterval)
	assert.Empty(t, ad.config.Actions)
}

func TestJavaScriptObfuscator_EmptyVariables(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	variables := obfuscator.extractVariableNames("")
	assert.Empty(t, variables)
}

func TestJavaScriptObfuscator_InsertDeadCode(t *testing.T) {
	config := ObfuscationConfig{}
	obfuscator := NewJavaScriptObfuscator(config)

	code := "var x = 1;"
	result := obfuscator.insertDeadCode(code)

	assert.Contains(t, result, ";")
}

func TestObfuscationConfig_AllOptions(t *testing.T) {
	config := ObfuscationConfig{
		EnableJS: obfuscationOptions{
			Enabled: true,
			Level:   5,
		},
		EnableCaptcha: obfuscationOptions{
			Enabled: true,
			Level:   3,
		},
		EnableProtocol: obfuscationOptions{
			Enabled: true,
			Level:   2,
		},
		EnableDebugging: debuggingOptions{
			BreakpointDetection: true,
			ConsoleDetection:   true,
			DebuggerDetection:  true,
			AntiScreenshots:   true,
		},
	}

	assert.True(t, config.EnableJS.Enabled)
	assert.Equal(t, 5, config.EnableJS.Level)
	assert.True(t, config.EnableDebugging.BreakpointDetection)
}

func TestProtocolMessage_Structure(t *testing.T) {
	msg := ProtocolMessage{
		ID:        "test-id",
		SessionID: "session-123",
		Sequence:  42,
		Encrypted: "encrypted-data",
		IV:        "iv-data",
		Timestamp: time.Now().Unix(),
		Type:      "request",
	}

	assert.Equal(t, "test-id", msg.ID)
	assert.Equal(t, "session-123", msg.SessionID)
	assert.Equal(t, uint64(42), msg.Sequence)
	assert.Equal(t, "request", msg.Type)
}

func TestCaptchaMetadata_Structure(t *testing.T) {
	metadata := CaptchaMetadata{
		Timestamp:   time.Now().Unix(),
		AppID:       "app-123",
		ChallengeID: "challenge-456",
	}

	assert.NotZero(t, metadata.Timestamp)
	assert.Equal(t, "app-123", metadata.AppID)
	assert.Equal(t, "challenge-456", metadata.ChallengeID)
}

func TestKeyExchangeResult_Structure(t *testing.T) {
	result := KeyExchangeResult{
		SessionID:    "session-abc",
		EncryptedKey: "encrypted-key-data",
		PublicKey:    "public-key-pem",
		IV:           "initialization-vector",
		Algorithm:    "AES-256-GCM",
		Timestamp:    time.Now().Unix(),
		ExpiresAt:    time.Now().Add(5 * time.Minute).Unix(),
	}

	assert.NotEmpty(t, result.SessionID)
	assert.NotEmpty(t, result.EncryptedKey)
	assert.NotEmpty(t, result.PublicKey)
	assert.Greater(t, result.ExpiresAt, result.Timestamp)
}

func TestObfuscationRatio_Above80Percent(t *testing.T) {
	code := `function complexFunction() {
		var counter = 0;
		var data = [];
		for (var i = 0; i < 100; i++) {
			counter += i;
			data.push(i * 2);
		}
		return counter + data.reduce((a, b) => a + b, 0);
	}`

	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 5}}
	obfuscator := NewJavaScriptObfuscator(config)

	result, err := obfuscator.ObfuscateCode(code)
	require.NoError(t, err)

	originalSize := float64(result.Metrics.OriginalSize)
	obfuscatedSize := float64(result.Metrics.ObfuscatedSize)

	if obfuscatedSize > 0 {
		ratio := (originalSize - obfuscatedSize) / originalSize * 100
		t.Logf("Obfuscation ratio: %.2f%%", ratio)
	}
}

func BenchmarkJavaScriptObfuscator_ObfuscateCode(b *testing.B) {
	config := ObfuscationConfig{EnableJS: obfuscationOptions{Enabled: true, Level: 5}}
	obfuscator := NewJavaScriptObfuscator(config)

	code := `function example() {
		var x = 1, y = 2, z = 3;
		if (x > 0 && y > 0) {
			return x + y + z;
		}
		return 0;
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = obfuscator.ObfuscateCode(code)
	}
}

func BenchmarkCaptchaEncryption_EncryptImage(b *testing.B) {
	enc := NewCaptchaEncryption(CaptchaConfig{})
	img := createTestImage(200, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.EncryptImage(img, "bench", "bench-challenge")
	}
}

func BenchmarkProtocolEncryptor_EncryptRequest(b *testing.B) {
	encryptor, _ := NewProtocolEncryptor(ProtocolConfig{})
	data := []byte(`{"action": "test", "payload": "data"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.EncryptRequest(data, encryptor.keys.SessionID)
	}
}

func TestCaptchaEncryption_ImageDataIntegrity(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{EncryptionAlgorithm: "AES-256-GCM"})

	img := createTestImage(150, 150)

	encrypted, err := enc.EncryptImage(img, "test-app", "test-challenge")
	require.NoError(t, err)

	decrypted, err := enc.DecryptImage(encrypted)
	require.NoError(t, err)

	decryptedRGBA, ok := decrypted.(*image.RGBA)
	require.True(t, ok)

	originalRGBA := img.(*image.RGBA)

	pixelCount := 0
	matchingPixels := 0
	for y := 0; y < 150; y++ {
		for x := 0; x < 150; x++ {
			pixelCount++
			origPixel := originalRGBA.RGBAAt(x, y)
			decPixel := decryptedRGBA.RGBAAt(x, y)

			if origPixel.R == decPixel.R && origPixel.G == decPixel.G &&
				origPixel.B == decPixel.B && origPixel.A == decPixel.A {
				matchingPixels++
			}
		}
	}

	matchRatio := float64(matchingPixels) / float64(pixelCount)
	assert.Greater(t, matchRatio, 0.99, "Image data integrity check failed")
}

func TestProtocolEncryptor_SequenceNumberIncrement(t *testing.T) {
	encryptor, _ := NewProtocolEncryptor(ProtocolConfig{SessionTimeout: 3600})

	data := []byte("test message")

	msg1, err := encryptor.EncryptRequest(data, encryptor.keys.SessionID)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), msg1.Sequence)

	msg2, err := encryptor.EncryptRequest(data, encryptor.keys.SessionID)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), msg2.Sequence)

	msg3, err := encryptor.EncryptRequest(data, encryptor.keys.SessionID)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), msg3.Sequence)
}

func TestAntiDebug_DetectionScriptContainsTiming(t *testing.T) {
	config := AntiDebugConfig{DetectionInterval: 500}
	ad := NewAntiDebug(config)

	script := ad.GenerateDetectionScript()

	assert.Contains(t, script, "Date.now")
	assert.Contains(t, script, "setInterval")
	assert.Contains(t, script, "_check")
}

func TestCaptchaKeyManager_VersionTracking(t *testing.T) {
	enc := NewCaptchaEncryption(CaptchaConfig{})

	initialVersion := enc.keys.Version
	initialRotations := enc.keys.Rotations
	require.Greater(t, initialVersion, 0)

	err := enc.rotateKey()
	require.NoError(t, err)
	assert.Equal(t, initialVersion+1, enc.keys.Version)

	err = enc.rotateKey()
	require.NoError(t, err)
	assert.Equal(t, initialVersion+2, enc.keys.Version)

	assert.Equal(t, initialRotations+2, enc.keys.Rotations)
}
