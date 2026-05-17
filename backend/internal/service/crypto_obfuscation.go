package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math/big"
	mathrand "math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/crypto"
)

var (
	ErrInvalidCiphertext    = errors.New("invalid ciphertext format")
	ErrDecryptionFailed    = errors.New("decryption failed")
	ErrInvalidKey          = errors.New("invalid key")
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrKeyGenerationFailed = errors.New("key generation failed")
	ErrObfuscationFailed  = errors.New("obfuscation failed")
)

type ObfuscationConfig struct {
	EnableJS obfuscationOptions `json:"enable_js"`
	EnableCaptcha obfuscationOptions `json:"enable_captcha"`
	EnableProtocol obfuscationOptions `json:"enable_protocol"`
	EnableDebugging debuggingOptions `json:"enable_debugging"`
}

type obfuscationOptions struct {
	Enabled bool `json:"enabled"`
	Level   int  `json:"level"`
}

type debuggingOptions struct {
	BreakpointDetection   bool `json:"breakpoint_detection"`
	ConsoleDetection      bool `json:"console_detection"`
	DebuggerDetection     bool `json:"debugger_detection"`
	AntiScreenshots        bool `json:"anti_screenshots"`
}

type JavaScriptObfuscator struct {
	config ObfuscationConfig
	mu     sync.RWMutex
}

type ObfuscatedCode struct {
	Code           string            `json:"code"`
	SourceMap      string            `json:"source_map,omitempty"`
	ObfuscationLevel int             `json:"obfuscation_level"`
	Techniques     []string          `json:"techniques"`
	Metrics        ObfuscationMetrics `json:"metrics"`
}

type ObfuscationMetrics struct {
	OriginalSize   int     `json:"original_size"`
	ObfuscatedSize int     `json:"obfuscated_size"`
	Ratio          float64 `json:"ratio"`
	DurationMs     int64   `json:"duration_ms"`
}

type ControlFlowFlat struct {
	Blocks     []ControlBlock `json:"blocks"`
	Edges      []ControlEdge  `json:"edges"`
	EntryBlock int            `json:"entry_block"`
}

type ControlBlock struct {
	ID       int           `json:"id"`
	Code     string        `json:"code"`
	Children []int         `json:"children"`
	Type     string        `json:"type"`
}

type ControlEdge struct {
	From   int    `json:"from"`
	To     int    `json:"to"`
	Cond   string `json:"condition,omitempty"`
}

type VariableMapping struct {
	Original string `json:"original"`
	Obfuscated string `json:"obfuscated"`
	Type     string `json:"type"`
}

type CaptchaEncryption struct {
	config CaptchaConfig
	keys   *CaptchaKeyManager
	mu     sync.RWMutex
}

type CaptchaConfig struct {
	EncryptionAlgorithm string `json:"algorithm"`
	KeyRotationPeriod   int    `json:"key_rotation_minutes"`
	EnableSteganography bool   `json:"enable_steganography"`
	ImageQuality        int    `json:"image_quality"`
}

type CaptchaKeyManager struct {
	CurrentKey  []byte
	PreviousKey []byte
	Version     int
	CreatedAt   time.Time
	Rotations   int
	mu          sync.RWMutex
}

type EncryptedCaptcha struct {
	ImageData   string            `json:"image_data"`
	KeyID       string            `json:"key_id"`
	Checksum    string            `json:"checksum"`
	Algorithm   string            `json:"algorithm"`
	Version     int               `json:"version"`
	Metadata    CaptchaMetadata   `json:"metadata"`
}

type CaptchaMetadata struct {
	Timestamp   int64  `json:"timestamp"`
	AppID       string `json:"app_id"`
	ChallengeID string `json:"challenge_id"`
}

type ProtocolEncryptor struct {
	config ProtocolConfig
	keys   *ProtocolKeyManager
	mu     sync.RWMutex
}

type ProtocolConfig struct {
	KeyExchangeMethod  string `json:"key_exchange"`
	SymmetricAlgo     string `json:"symmetric"`
	HMACAlgo          string `json:"hmac"`
	SessionTimeout    int    `json:"session_timeout"`
	EnableForwardSec  bool   `json:"enable_forward_secrecy"`
}

type ProtocolKeyManager struct {
	SessionKey      []byte
	MasterKey       []byte
	PublicKey       *rsa.PrivateKey
	SessionID       string
	CreatedAt       time.Time
	LastActivity    time.Time
	SequenceNumber  uint64
	mu              sync.RWMutex
}

type KeyExchangeResult struct {
	SessionID      string `json:"session_id"`
	EncryptedKey   string `json:"encrypted_key"`
	PublicKey      string `json:"public_key"`
	IV             string `json:"iv"`
	Algorithm      string `json:"algorithm"`
	Timestamp      int64  `json:"timestamp"`
	ExpiresAt      int64  `json:"expires_at"`
}

type ProtocolMessage struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	Sequence    uint64 `json:"sequence"`
	Encrypted   string `json:"encrypted"`
	IV          string `json:"iv"`
	AuthTag     string `json:"auth_tag"`
	Timestamp   int64  `json:"timestamp"`
	Type        string `json:"type"`
}

type AntiDebug struct {
	config AntiDebugConfig
	mu     sync.RWMutex
}

type AntiDebugConfig struct {
	DetectionInterval    int      `json:"detection_interval_ms"`
	Actions             []string `json:"actions"`
	Severity            string   `json:"severity"`
	BreakpointDetection  bool     `json:"breakpoint_detection"`
	ConsoleDetection     bool     `json:"console_detection"`
	DebuggerDetection    bool     `json:"debugger_detection"`
	AntiScreenshots     bool     `json:"anti_screenshots"`
}

type DebugEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
	Severity  string    `json:"severity"`
}

func NewJavaScriptObfuscator(config ObfuscationConfig) *JavaScriptObfuscator {
	return &JavaScriptObfuscator{
		config: config,
	}
}

func (o *JavaScriptObfuscator) ObfuscateCode(code string) (*ObfuscatedCode, error) {
	startTime := time.Now()
	o.mu.Lock()
	defer o.mu.Unlock()

	if code == "" {
		return nil, ErrInvalidConfig
	}

	techniques := []string{}
	obfuscated := code
	level := o.config.EnableJS.Level

	if level >= 1 {
		obfuscated = o.obfuscateVariables(obfuscated)
		techniques = append(techniques, "variable_name_obfuscation")
	}

	if level >= 2 {
		obfuscated = o.insertDeadCode(obfuscated)
		techniques = append(techniques, "dead_code_injection")
	}

	if level >= 3 {
		obfuscated = o.obfuscateStrings(obfuscated)
		techniques = append(techniques, "string_encryption")
	}

	if level >= 4 {
		obfuscated = o.applyControlFlowFlattening(obfuscated)
		techniques = append(techniques, "control_flow_flattening")
	}

	if level >= 5 {
		obfuscated = o.applyPropertyProxy(obfuscated)
		techniques = append(techniques, "property_proxy")
	}

	obfuscated = o.minifyCode(obfuscated)
	techniques = append(techniques, "minification")

	metrics := ObfuscationMetrics{
		OriginalSize:   len(code),
		ObfuscatedSize: len(obfuscated),
		DurationMs:     time.Since(startTime).Milliseconds(),
	}
	if len(code) > 0 {
		metrics.Ratio = float64(len(code)-len(obfuscated)) / float64(len(code)) * 100
	}

	return &ObfuscatedCode{
		Code: obfuscated,
		ObfuscationLevel: level,
		Techniques: techniques,
		Metrics: metrics,
	}, nil
}

func (o *JavaScriptObfuscator) obfuscateVariables(code string) string {
	varNames := o.extractVariableNames(code)
	nameMapping := o.generateObfuscatedNames(len(varNames))

	result := code
	for i, original := range varNames {
		obfuscated := nameMapping[i]
		result = o.replaceVariableOccurrences(result, original, obfuscated)
	}

	return result
}

func (o *JavaScriptObfuscator) extractVariableNames(code string) []string {
	var names []string
	seen := make(map[string]bool)
	
	patterns := []string{
		`\b(var|let|const)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\b`,
		`\bfunction\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\b`,
		`\b([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=\s*function\b`,
	}
	
	for _, pattern := range patterns {
		matches := o.findAllMatches(code, pattern)
		for _, match := range matches {
			if len(match) >= 2 && !seen[match[1]] {
				names = append(names, match[1])
				seen[match[1]] = true
			}
		}
	}
	
	return names
}

func (o *JavaScriptObfuscator) generateObfuscatedNames(count int) []string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_$"
	names := make([]string, count)
	
	for i := 0; i < count; i++ {
		length := 3 + i%5
		name := make([]byte, length)
		for j := 0; j < length; j++ {
			name[j] = chars[mathrand.Intn(len(chars))]
		}
		names[i] = string(name)
	}
	
	return names
}

func (o *JavaScriptObfuscator) replaceVariableOccurrences(code, original, obfuscated string) string {
	result := strings.ReplaceAll(code, original, obfuscated)
	return result
}

func (o *JavaScriptObfuscator) findAllMatches(code, pattern string) [][]string {
	var matches [][]string
	
	start := 0
	for {
		idx := strings.Index(code[start:], pattern)
		if idx == -1 {
			break
		}
		start += idx
		
		remaining := code[start:]
		for i := 0; i < len(remaining); i++ {
			if i > 1000 {
				break
			}
			if remaining[i] == '{' || remaining[i] == '\n' {
				matchStr := remaining[:i]
				if strings.HasPrefix(matchStr, "var ") || 
				   strings.HasPrefix(matchStr, "let ") || 
				   strings.HasPrefix(matchStr, "const ") ||
				   strings.HasPrefix(matchStr, "function ") {
					matches = append(matches, []string{matchStr, o.extractVarName(matchStr)})
				}
				break
			}
		}
		start++
	}
	
	return matches
}

func (o *JavaScriptObfuscator) extractVarName(code string) string {
	parts := strings.Fields(code)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (o *JavaScriptObfuscator) insertDeadCode(code string) string {
	deadCodePatterns := []string{
		`(function(){var _0x%x=Date.now();if(_0x%x%%3===0){return true;}})();`,
		`(function(){try{eval('" + "'.repeat(3));}catch(e){}})();`,
		`var _0x%x=%d;while(_0x%x-->0){if(_0x%x%%7===0){break;}}`,
	}
	
	var buf bytes.Buffer
	buf.WriteString(code)
	
	for _, pattern := range deadCodePatterns {
		seed := mathrand.Int63()
		formatted := fmt.Sprintf(pattern, seed, seed, seed, seed)
		buf.WriteString(";" + formatted)
	}
	
	return buf.String()
}

func (o *JavaScriptObfuscator) obfuscateStrings(code string) string {
	stringPattern := `"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'`

	matches := o.extractStrings(code, stringPattern)

	result := code
	for i, match := range matches {
		encoded := o.encodeString(match)
		_ = fmt.Sprintf("_0x%x", i)
		replacement := fmt.Sprintf("(function(){var _=%s;return _.charAt?_.split(''):_.slice(0);})()", encoded)
		result = strings.Replace(result, match, replacement, 1)
	}

	return result
}

func (o *JavaScriptObfuscator) extractStrings(code, pattern string) []string {
	var result []string
	inString := false
	var current bytes.Buffer
	var quote byte

	for i := 0; i < len(code); i++ {
		c := code[i]

		if !inString && (c == '"' || c == '\'') {
			inString = true
			quote = c
			current.Reset()
			current.WriteByte(c)
		} else if inString {
			current.WriteByte(c)
			if c == quote && (i == 0 || code[i-1] != '\\') {
				inString = false
				result = append(result, current.String())
			}
		}
	}

	return result
}

func (o *JavaScriptObfuscator) encodeString(s string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	return fmt.Sprintf("atob('%s')", encoded)
}

func (o *JavaScriptObfuscator) applyControlFlowFlattening(code string) string {
	flow := o.generateControlFlow(code)
	
	flattened := fmt.Sprintf(`
(function(_ctx){
	var _states = %d;
	var _current = 0;
	var _stack = [];
	
	function _dispatch(){
		switch(_current){
%s		}
	}
	
	_ctx._dispatch = _dispatch;
	return _ctx;
})(this);
`, flow.EntryBlock, o.generateSwitchCases(flow))
	
	return code + "\n" + flattened
}

func (o *JavaScriptObfuscator) generateControlFlow(code string) ControlFlowFlat {
	return ControlFlowFlat{
		Blocks: []ControlBlock{
			{ID: 0, Type: "entry", Code: "var _0x0=1;", Children: []int{1, 2}},
			{ID: 1, Type: "block", Code: "var _0x1=2;", Children: []int{3}},
			{ID: 2, Type: "block", Code: "var _0x2=3;", Children: []int{3}},
			{ID: 3, Type: "exit", Code: "return true;", Children: []int{}},
		},
		Edges: []ControlEdge{
			{From: 0, To: 1, Cond: "true"},
			{From: 0, To: 2, Cond: "false"},
			{From: 1, To: 3},
			{From: 2, To: 3},
		},
		EntryBlock: 0,
	}
}

func (o *JavaScriptObfuscator) generateSwitchCases(flow ControlFlowFlat) string {
	var buf bytes.Buffer
	for _, block := range flow.Blocks {
		fmt.Fprintf(&buf, "\t\tcase %d:\n\t\t\t%s\n", block.ID, block.Code)
		if len(block.Children) > 0 {
			next := block.Children[mathrand.Intn(len(block.Children))]
			fmt.Fprintf(&buf, "\t\t\t_current = %d;\n\t\t\tbreak;\n", next)
		} else {
			fmt.Fprintf(&buf, "\t\t\treturn;\n")
		}
	}
	return buf.String()
}

func (o *JavaScriptObfuscator) applyPropertyProxy(code string) string {
	proxyCode := `
(function(){
	var _origDescriptors = {};
	var _props = Object.getOwnPropertyNames(window);
	_props.forEach(function(_p){
		try{
			var _d = Object.getOwnPropertyDescriptor(window, _p);
			if(_d && _d.configurable){
				_origDescriptors[_p] = _d;
				Object.defineProperty(window, _p, {
					get: function(){
						return _origDescriptors[_p].get ? _origDescriptors[_p].get() : undefined;
					},
					set: function(_v){
						return _origDescriptors[_p].set ? _origDescriptors[_p].set(_v) : undefined;
					},
					configurable: true,
					enumerable: _d.enumerable
				});
			}
		}catch(e){}
	});
})();
`
	return code + "\n" + proxyCode
}

func (o *JavaScriptObfuscator) minifyCode(code string) string {
	code = strings.ReplaceAll(code, "\n", "")
	code = strings.ReplaceAll(code, "\r", "")
	code = strings.ReplaceAll(code, "\t", "")
	
	for strings.Contains(code, "  ") {
		code = strings.ReplaceAll(code, "  ", " ")
	}
	
	return code
}

func (o *JavaScriptObfuscator) GetVariableMappings(code string) []VariableMapping {
	varNames := o.extractVariableNames(code)
	mappings := make([]VariableMapping, len(varNames))
	
	for i, original := range varNames {
		mappings[i] = VariableMapping{
			Original:   original,
			Obfuscated: fmt.Sprintf("_0x%x", i),
			Type:      "local_variable",
		}
	}
	
	return mappings
}

func (o *JavaScriptObfuscator) Deobfuscate(obfuscatedCode, mapping string) (string, error) {
	var mappings []VariableMapping
	if err := json.Unmarshal([]byte(mapping), &mappings); err != nil {
		return "", fmt.Errorf("invalid mapping format: %w", err)
	}
	
	result := obfuscatedCode
	for _, m := range mappings {
		result = strings.ReplaceAll(result, m.Obfuscated, m.Original)
	}
	
	return result, nil
}

func (o *JavaScriptObfuscator) CalculateObfuscationRatio(original, obfuscated string) float64 {
	if len(original) == 0 {
		return 0
	}
	return float64(len(obfuscated)) / float64(len(original))
}

func NewCaptchaEncryption(config CaptchaConfig) *CaptchaEncryption {
	c := &CaptchaEncryption{
		config: config,
		keys: &CaptchaKeyManager{
			CreatedAt: time.Now(),
		},
	}
	c.rotateKey()
	return c
}

func (c *CaptchaEncryption) rotateKey() error {
	c.keys.mu.Lock()
	defer c.keys.mu.Unlock()

	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return ErrKeyGenerationFailed
	}

	c.keys.PreviousKey = c.keys.CurrentKey
	c.keys.CurrentKey = newKey
	c.keys.Version++
	c.keys.CreatedAt = time.Now()
	c.keys.Rotations++

	return nil
}

func (c *CaptchaEncryption) EncryptImage(img image.Image, appID, challengeID string) (*EncryptedCaptcha, error) {
	c.keys.mu.RLock()
	key := c.keys.CurrentKey
	version := c.keys.Version
	c.keys.mu.RUnlock()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	encrypted, err := c.encryptWithKey(buf.Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	checksum := crypto.HashSHA256(encrypted)

	return &EncryptedCaptcha{
		ImageData: base64.StdEncoding.EncodeToString(encrypted),
		KeyID:     fmt.Sprintf("key_%d_%d", version, time.Now().Unix()),
		Checksum:  checksum,
		Algorithm: c.config.EncryptionAlgorithm,
		Version:   version,
		Metadata: CaptchaMetadata{
			Timestamp:   time.Now().Unix(),
			AppID:       appID,
			ChallengeID: challengeID,
		},
	}, nil
}

func (c *CaptchaEncryption) encryptWithKey(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (c *CaptchaEncryption) DecryptImage(encrypted *EncryptedCaptcha) (image.Image, error) {
	c.keys.mu.RLock()
	var key []byte
	if c.keys.Version == encrypted.Version {
		key = c.keys.CurrentKey
	} else {
		key = c.keys.PreviousKey
	}
	c.keys.mu.RUnlock()

	if key == nil {
		return nil, ErrInvalidKey
	}

	data, err := base64.StdEncoding.DecodeString(encrypted.ImageData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	decrypted, err := c.decryptWithKey(data, key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(decrypted))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

func (c *CaptchaEncryption) decryptWithKey(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (c *CaptchaEncryption) EncryptWithSteganography(img image.Image, secret []byte, appID, challengeID string) (*EncryptedCaptcha, error) {
	encrypted, err := c.EncryptImage(img, appID, challengeID)
	if err != nil {
		return nil, err
	}

	stegoImg := c.embedData(img, secret)
	
	var buf bytes.Buffer
	if err := png.Encode(&buf, stegoImg); err != nil {
		return nil, err
	}

	encrypted.ImageData = base64.StdEncoding.EncodeToString(buf.Bytes())
	encrypted.Algorithm = "steganography_" + c.config.EncryptionAlgorithm

	return encrypted, nil
}

func (c *CaptchaEncryption) embedData(img image.Image, data []byte) image.Image {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	
	dataWithHeader := c.prepareDataWithHeader(data)
	
	copy(result.Pix, img.(*image.RGBA).Pix)
	
	dataBits := c.bytesToBits(dataWithHeader)
	
	bitIndex := 0
	for y := bounds.Min.Y; y < bounds.Max.Y && bitIndex < len(dataBits); y++ {
		for x := bounds.Min.X; x < bounds.Max.X && bitIndex < len(dataBits); x++ {
			idx := (y-bounds.Min.Y)*result.Stride + (x-bounds.Min.X)*4
			
			result.Pix[idx] = (result.Pix[idx] & 0xFE) | dataBits[bitIndex]
			bitIndex++
		}
	}
	
	return result
}

func (c *CaptchaEncryption) prepareDataWithHeader(data []byte) []byte {
	length := uint32(len(data))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)
	return append(header, data...)
}

func (c *CaptchaEncryption) bytesToBits(data []byte) []byte {
	bits := make([]byte, len(data)*8)
	for i, b := range data {
		for j := 7; j >= 0; j-- {
			bits[i*8+(7-j)] = (b >> j) & 1
		}
	}
	return bits
}

func (c *CaptchaEncryption) extractData(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	
	headerBits := make([]byte, 32)
	bitIndex := 0
	
	for y := bounds.Min.Y; y < bounds.Max.Y && bitIndex < 32; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && bitIndex < 32; x++ {
			offset := (y-bounds.Min.Y)*img.(*image.RGBA).Stride + (x-bounds.Min.X)*4
			headerBits[bitIndex] = img.(*image.RGBA).Pix[offset] & 1
			bitIndex++
		}
	}
	
	header := c.bitsToBytes(headerBits)
	length := binary.BigEndian.Uint32(header)
	
	if length > 1024*1024 {
		return nil, errors.New("invalid data length")
	}
	
	dataBits := make([]byte, length*8)
	bitIndex = 0
	
	for y := bounds.Min.Y; y < bounds.Max.Y && bitIndex < int(length)*8; y++ {
		for x := bounds.Min.X + 4; x < bounds.Max.X && bitIndex < int(length)*8; x++ {
			offset := (y-bounds.Min.Y)*img.(*image.RGBA).Stride + (x-bounds.Min.X)*4
			dataBits[bitIndex] = img.(*image.RGBA).Pix[offset] & 1
			bitIndex++
		}
	}
	
	return c.bitsToBytes(dataBits), nil
}

func (c *CaptchaEncryption) bitsToBytes(bits []byte) []byte {
	result := make([]byte, len(bits)/8)
	for i := 0; i < len(result); i++ {
		for j := 0; j < 8; j++ {
			result[i] |= bits[i*8+j] << (7 - j)
		}
	}
	return result
}

func (c *CaptchaEncryption) GenerateChallengeResponse(challenge, response string) (string, error) {
	c.keys.mu.RLock()
	key := c.keys.CurrentKey
	c.keys.mu.RUnlock()

	h := sha256.New()
	h.Write([]byte(challenge))
	h.Write([]byte(response))
	h.Write(key)
	
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (c *CaptchaEncryption) VerifyChallengeResponse(challenge, response, expected string) bool {
	generated, err := c.GenerateChallengeResponse(challenge, response)
	if err != nil {
		return false
	}
	return crypto.ConstantTimeCompare(generated, expected)
}

func NewProtocolEncryptor(config ProtocolConfig) (*ProtocolEncryptor, error) {
	p := &ProtocolEncryptor{
		config: config,
		keys: &ProtocolKeyManager{
			CreatedAt: time.Now(),
		},
	}

	if err := p.initializeKeys(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *ProtocolEncryptor) initializeKeys() error {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ErrKeyGenerationFailed
	}

	p.keys.PublicKey = privateKey
	p.keys.MasterKey = make([]byte, 32)
	if _, err := rand.Read(p.keys.MasterKey); err != nil {
		return ErrKeyGenerationFailed
	}

	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		return ErrKeyGenerationFailed
	}
	p.keys.SessionKey = sessionKey

	sessionID := make([]byte, 16)
	if _, err := rand.Read(sessionID); err != nil {
		return ErrKeyGenerationFailed
	}
	p.keys.SessionID = base64.StdEncoding.EncodeToString(sessionID)

	p.keys.CreatedAt = time.Now()
	p.keys.LastActivity = time.Now()
	p.keys.SequenceNumber = 0

	return nil
}

func (p *ProtocolEncryptor) InitiateKeyExchange(clientPublicKey string) (*KeyExchangeResult, error) {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()

	clientKey, err := crypto.ParseRSAPublicKeyFromPEM(clientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid client public key: %w", err)
	}

	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		return nil, ErrKeyGenerationFailed
	}

	encryptedKey, err := crypto.RSAEncrypt(sessionKey, clientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt session key: %w", err)
	}

	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return nil, ErrKeyGenerationFailed
	}

	p.keys.SessionKey = sessionKey

	sessionIDBytes := make([]byte, 16)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return nil, ErrKeyGenerationFailed
	}
	p.keys.SessionID = base64.StdEncoding.EncodeToString(sessionIDBytes)
	p.keys.CreatedAt = time.Now()
	p.keys.LastActivity = time.Now()
	p.keys.SequenceNumber = 0

	publicKeyPEM, err := crypto.ExportRSAPrivateKeyToPEM(p.keys.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to export public key: %w", err)
	}

	return &KeyExchangeResult{
		SessionID:    p.keys.SessionID,
		EncryptedKey: base64.StdEncoding.EncodeToString(encryptedKey),
		PublicKey:    publicKeyPEM,
		IV:           base64.StdEncoding.EncodeToString(iv),
		Algorithm:    p.config.SymmetricAlgo,
		Timestamp:    time.Now().Unix(),
		ExpiresAt:    time.Now().Add(time.Duration(p.config.SessionTimeout) * time.Second).Unix(),
	}, nil
}

func (p *ProtocolEncryptor) CompleteKeyExchange(encryptedKey, iv string) error {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()

	encryptedKeyBytes, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return fmt.Errorf("invalid encrypted key: %w", err)
	}

	sessionKey, err := crypto.RSADecrypt(encryptedKeyBytes, p.keys.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt session key: %w", err)
	}

	p.keys.SessionKey = sessionKey
	p.keys.LastActivity = time.Now()

	return nil
}

func (p *ProtocolEncryptor) EncryptRequest(data []byte, sessionID string) (*ProtocolMessage, error) {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()

	if sessionID != p.keys.SessionID {
		return nil, errors.New("invalid session")
	}

	if time.Since(p.keys.LastActivity) > time.Duration(p.config.SessionTimeout)*time.Second {
		return nil, errors.New("session expired")
	}

	p.keys.SequenceNumber++

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, ErrKeyGenerationFailed
	}

	encrypted, err := p.encryptAEAD(data, nonce)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return &ProtocolMessage{
		ID:        fmt.Sprintf("msg_%d_%d", p.keys.SequenceNumber, time.Now().UnixNano()),
		SessionID: sessionID,
		Sequence:  p.keys.SequenceNumber,
		Encrypted: base64.StdEncoding.EncodeToString(encrypted),
		IV:       base64.StdEncoding.EncodeToString(nonce),
		Timestamp: time.Now().Unix(),
		Type:      "request",
	}, nil
}

func (p *ProtocolEncryptor) DecryptRequest(msg *ProtocolMessage) ([]byte, error) {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()

	if msg.SessionID != p.keys.SessionID {
		return nil, errors.New("invalid session")
	}

	if msg.Sequence < p.keys.SequenceNumber {
		return nil, errors.New("replay detected")
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(msg.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(msg.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	plaintext, err := p.decryptAEAD(encryptedBytes, nonce)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	p.keys.SequenceNumber = msg.Sequence
	p.keys.LastActivity = time.Now()

	return plaintext, nil
}

func (p *ProtocolEncryptor) EncryptResponse(data []byte, requestSeq uint64) (*ProtocolMessage, error) {
	return p.EncryptRequest(data, p.keys.SessionID)
}

func (p *ProtocolEncryptor) DecryptResponse(msg *ProtocolMessage) ([]byte, error) {
	return p.DecryptRequest(msg)
}

func (p *ProtocolEncryptor) encryptAEAD(plaintext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.keys.SessionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte(p.keys.SessionID))
	return ciphertext, nil
}

func (p *ProtocolEncryptor) decryptAEAD(ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.keys.SessionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, []byte(p.keys.SessionID))
}

func (p *ProtocolEncryptor) ComputeHMAC(data []byte) (string, error) {
	p.keys.mu.RLock()
	defer p.keys.mu.RUnlock()

	h := hmac.New(sha256.New, p.keys.SessionKey)
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (p *ProtocolEncryptor) VerifyHMAC(data []byte, expectedMAC string) bool {
	mac, err := p.ComputeHMAC(data)
	if err != nil {
		return false
	}
	return crypto.ConstantTimeCompare(mac, expectedMAC)
}

func (p *ProtocolEncryptor) GetSessionInfo() map[string]interface{} {
	p.keys.mu.RLock()
	defer p.keys.mu.RUnlock()

	return map[string]interface{}{
		"session_id":      p.keys.SessionID,
		"created_at":      p.keys.CreatedAt.Unix(),
		"last_activity":   p.keys.LastActivity.Unix(),
		"sequence_number": p.keys.SequenceNumber,
		"expires_in":      p.config.SessionTimeout,
	}
}

func NewAntiDebug(config AntiDebugConfig) *AntiDebug {
	return &AntiDebug{
		config: config,
	}
}

func (a *AntiDebug) DetectBreakpoints() []DebugEvent {
	var events []DebugEvent

	if a.isDebuggerAttached() {
		events = append(events, DebugEvent{
			Type:      "breakpoint",
			Timestamp: time.Now(),
			Details:   "Potential breakpoint detected via timing analysis",
			Severity:  "high",
		})
	}

	if a.config.BreakpointDetection && a.checkBreakpointInjection() {
		events = append(events, DebugEvent{
			Type:      "breakpoint_injection",
			Timestamp: time.Now(),
			Details:   "Breakpoint injection attempt detected",
			Severity:  "critical",
		})
	}

	return events
}

func (a *AntiDebug) isDebuggerAttached() bool {
	start := time.Now()
	
	iterations := 100000
	for i := 0; i < iterations; i++ {
		_ = i * i
	}
	
	elapsed := time.Since(start)
	
	if elapsed > 100*time.Millisecond {
		return true
	}
	
	return false
}

func (a *AntiDebug) checkBreakpointInjection() bool {
	testFunc := func() {}
	
	testFunc()
	
	return false
}

func (a *AntiDebug) DetectConsoleActivity() []DebugEvent {
	var events []DebugEvent

	if a.config.ConsoleDetection {
		if a.isConsoleOpen() {
			events = append(events, DebugEvent{
				Type:      "console",
				Timestamp: time.Now(),
				Details:   "Developer console opened",
				Severity:  "medium",
			})
		}

		if a.detectConsoleCommands() {
			events = append(events, DebugEvent{
				Type:      "console_command",
				Timestamp: time.Now(),
				Details:   "Suspicious console commands detected",
				Severity:  "high",
			})
		}
	}

	return events
}

func (a *AntiDebug) isConsoleOpen() bool {
	threshold := 160
	width := 0
	if width > threshold {
		return true
	}
	return false
}

func (a *AntiDebug) detectConsoleCommands() bool {
	return false
}

func (a *AntiDebug) DetectDebugger() []DebugEvent {
	var events []DebugEvent

	if !a.config.DebuggerDetection {
		return events
	}

	if a.detectDevTools() {
		events = append(events, DebugEvent{
			Type:      "devtools",
			Timestamp: time.Now(),
			Details:   "Developer tools opened",
			Severity:  "high",
		})
	}

	if a.detectDebuggerKeywords() {
		events = append(events, DebugEvent{
			Type:      "debugger_keyword",
			Timestamp: time.Now(),
			Details:   "debugger keyword usage detected",
			Severity:  "critical",
		})
	}

	if a.detectFunctionOverrides() {
		events = append(events, DebugEvent{
			Type:      "function_override",
			Timestamp: time.Now(),
			Details:   "Console function override detected",
			Severity:  "medium",
		})
	}

	return events
}

func (a *AntiDebug) detectDevTools() bool {
	width := 0
	height := 0
	return width > 0 && height > 0
}

func (a *AntiDebug) detectDebuggerKeywords() bool {
	return false
}

func (a *AntiDebug) detectFunctionOverrides() bool {
	consoleOverrides := []string{"log", "warn", "error", "debug", "info"}
	for _, method := range consoleOverrides {
		if method == "" {
			return true
		}
	}
	return false
}

func (a *AntiDebug) PreventScreenshots() string {
	return `
(function(){
	var _origCreateElement = document.createElement.bind(document);
	var _screenshotBlocked = false;
	
	document.addEventListener('webkitvisibilitychange', function(){
		if(document.webkitHidden && !_screenshotBlocked){
			_screenshotBlocked = true;
		}
	});
	
	Object.defineProperty(HTMLCanvasElement.prototype, 'toDataURL', {
		configurable: false,
		get: function(){
			var _origFunc = this._obfuscated_getContext;
			if(!_origFunc){
				var _ctx = this.getContext('2d');
				if(_ctx && _ctx.canvas){
					var _data = _ctx.getImageData(0, 0, this.width, this.height);
					_ctx.putImageData(_data, 0, 0);
					_data = _ctx.getImageData(0, 0, this.width, this.height);
					for(var _i = 0; _i < _data.data.length; _i += 4){
						_data.data[_i] = (_data.data[_i] + 128) % 256;
						_data.data[_i+1] = (_data.data[_i+1] + 128) % 256;
						_data.data[_i+2] = (_data.data[_i+2] + 128) % 256;
					}
					_ctx.putImageData(_data, 0, 0);
				}
			}
			return function(){
				if(_screenshotBlocked){
					throw new Error('Screenshot blocked');
				}
				return _origFunc.apply(this, arguments);
			};
		}
	});
	
	document.addEventListener('copy', function(e){
		if(document.webkitHidden){
			e.preventDefault();
			return false;
		}
	});
	
	document.addEventListener('cut', function(e){
		if(document.webkitHidden){
			e.preventDefault();
			return false;
		}
	});
	
	document.addEventListener('contextmenu', function(e){
		if(document.webkitHidden){
			e.preventDefault();
			return false;
		}
	});
})();
`
}

func (a *AntiDebug) GenerateDetectionScript() string {
	return `
(function(){
	var _a = {
		_cfg: {
			interval: ` + fmt.Sprintf("%d", a.config.DetectionInterval) + `,
			actions: ` + fmt.Sprintf("%v", a.config.Actions) + `
		},
		_events: [],
		_start: function(){
			var _t = this;
			setInterval(function(){
				_t._check();
			}, _t._cfg.interval);
		},
		_check: function(){
			var _d = Date.now();
			var _r = Math.sqrt(2);
			if(Date.now() - _d > 100){
				this._report('timing_anomaly');
			}
		},
		_report: function(_t){
			this._events.push({type: _t, time: Date.now()});
			if(this._events.length > 100){
				this._events.shift();
			}
		}
	};
	_a._start();
	return _a;
})();
`
}

func (a *AntiDebug) HandleDebugEvent(event DebugEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, action := range a.config.Actions {
		switch action {
		case "log":
			fmt.Printf("[DEBUG] %s: %s\n", event.Type, event.Details)
		case "alert":
		case "block":
			if event.Severity == "critical" || event.Severity == "high" {
				return fmt.Errorf("debug activity blocked: %s", event.Type)
			}
		case "disconnect":
			if event.Severity == "critical" {
				return fmt.Errorf("session terminated due to debug activity: %s", event.Type)
			}
		}
	}

	return nil
}

func GenerateRandomCurve() elliptic.Curve {
	curves := []elliptic.Curve{elliptic.P256(), elliptic.P384(), elliptic.P521()}
	return curves[mathrand.Intn(len(curves))]
}

func DeriveKeyFromECDH(priv *ecdsaPrivateKey, pubKey []byte) ([]byte, error) {
	px, py := elliptic.Unmarshal(GenerateRandomCurve(), pubKey)
	if px == nil {
		return nil, errors.New("invalid public key")
	}

	x, _ := priv.Curve.ScalarMult(px, py, priv.D.Bytes())
	if x == nil {
		return nil, errors.New("key derivation failed")
	}

	hash := sha256.New()
	hash.Write(x.Bytes())
	return hash.Sum(nil), nil
}

type ecdsaPrivateKey struct {
	Curve elliptic.Curve
	D     *big.Int
}

func PerformObfuscationRatioTest(originalCode string, level int) (float64, error) {
	config := ObfuscationConfig{
		EnableJS: obfuscationOptions{
			Enabled: true,
			Level:   level,
		},
	}
	
	obfuscator := NewJavaScriptObfuscator(config)
	result, err := obfuscator.ObfuscateCode(originalCode)
	if err != nil {
		return 0, err
	}

	return obfuscator.CalculateObfuscationRatio(originalCode, result.Code), nil
}

func GenerateObfuscationReport(code string, levels []int) map[string]interface{} {
	report := map[string]interface{}{
		"original_size": len(code),
		"levels":        []map[string]interface{}{},
	}

	for _, level := range levels {
		ratio, err := PerformObfuscationRatioTest(code, level)
		if err != nil {
			continue
		}

		levelReport := map[string]interface{}{
			"level":       level,
			"ratio":       ratio,
			"size_saved":  int(float64(len(code)) * (1 - ratio)),
		}
		report["levels"] = append(report["levels"].([]map[string]interface{}), levelReport)
	}

	return report
}
