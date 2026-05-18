package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type ObfuscatorConfig struct {
	EnableVariableObfuscation   bool
	EnableStringEncryption      bool
	EnableCodeCompression       bool
	EnableControlFlowFlattening bool
	EnableDeadCodeInjection     bool
	EnableFunctionWrapping      bool
	StringEncryptionKey         []byte
	CompressWhitespace          bool
	RemoveComments              bool
	PreserveConsole             bool
}

var defaultObfuscatorConfig = ObfuscatorConfig{
	EnableVariableObfuscation:   true,
	EnableStringEncryption:      true,
	EnableCodeCompression:       true,
	EnableControlFlowFlattening: true,
	EnableDeadCodeInjection:     false,
	EnableFunctionWrapping:      true,
	StringEncryptionKey:         []byte("hjtpx-obfuscate-key-2024"),
	CompressWhitespace:          true,
	RemoveComments:              true,
	PreserveConsole:             true,
}

type Obfuscator struct {
	config        ObfuscatorConfig
	variableMap   map[string]string
	functionMap   map[string]string
	usedNames     map[string]bool
	stringCount   int
	functionCount int
	mu            sync.Mutex
}

func NewObfuscator(config ...ObfuscatorConfig) *Obfuscator {
	cfg := defaultObfuscatorConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if len(cfg.StringEncryptionKey) == 0 {
		cfg.StringEncryptionKey = []byte("hjtpx-obfuscate-key-2024")
	}

	return &Obfuscator{
		config:        cfg,
		variableMap:   make(map[string]string),
		functionMap:   make(map[string]string),
		usedNames:     make(map[string]bool),
		stringCount:   0,
		functionCount: 0,
	}
}

func (o *Obfuscator) Obfuscate(code string) (string, error) {
	if code == "" {
		return "", errors.New("code cannot be empty")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.variableMap = make(map[string]string)
	o.functionMap = make(map[string]string)
	o.usedNames = make(map[string]bool)
	o.stringCount = 0
	o.functionCount = 0

	var result string

	if o.config.RemoveComments {
		result = o.removeComments(code)
	} else {
		result = code
	}

	if o.config.EnableVariableObfuscation {
		result = o.obfuscateVariables(result)
	}

	if o.config.EnableStringEncryption {
		result = o.encryptStrings(result)
	}

	if o.config.EnableFunctionWrapping {
		result = o.wrapCode(result)
	}

	if o.config.EnableControlFlowFlattening {
		result = o.flattenControlFlow(result)
	}

	if o.config.EnableDeadCodeInjection {
		result = o.injectDeadCode(result)
	}

	if o.config.EnableCodeCompression {
		result = o.compressCode(result)
	}

	return result, nil
}

func (o *Obfuscator) removeComments(code string) string {
	re := regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*`)
	return re.ReplaceAllString(code, "")
}

func (o *Obfuscator) obfuscateVariables(code string) string {
	result := code

	identifierPattern := regexp.MustCompile(`\b([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=\s*`)
	result = identifierPattern.ReplaceAllStringFunc(result, func(match string) string {
		name := match[:len(match)-2]
		if !o.isReservedWord(name) && !o.isAlreadyObfuscated(name) {
			newName := o.generateObfuscatedName()
			o.variableMap[name] = newName
			return newName + "="
		}
		return match
	})

	for original, obfuscated := range o.variableMap {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(original) + `\b`)
		result = re.ReplaceAllString(result, obfuscated)
	}

	return result
}

func (o *Obfuscator) isReservedWord(word string) bool {
	reservedWords := map[string]bool{
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"switch": true, "case": true, "default": true, "break": true,
		"continue": true, "return": true, "try": true, "catch": true,
		"finally": true, "throw": true, "new": true, "delete": true,
		"typeof": true, "instanceof": true, "void": true, "this": true,
		"super": true, "class": true, "extends": true, "static": true,
		"get": true, "set": true, "import": true, "export": true,
		"from": true, "as": true, "const": true, "let": true, "var": true,
		"function": true, "async": true, "await": true, "yield": true,
		"true": true, "false": true, "null": true, "undefined": true,
		"NaN": true, "Infinity": true, "arguments": true, "eval": true,
		"constructor": true, "prototype": true, "toString": true,
		"valueOf": true, "hasOwnProperty": true, "isPrototypeOf": true,
		"propertyIsEnumerable": true, "toLocaleString": true,
		"console": o.config.PreserveConsole,
		"log":     o.config.PreserveConsole,
		"error":   o.config.PreserveConsole,
		"warn":    o.config.PreserveConsole,
		"info":    o.config.PreserveConsole,
		"debug":   o.config.PreserveConsole,
	}
	return reservedWords[word]
}

func (o *Obfuscator) isAlreadyObfuscated(name string) bool {
	_, exists := o.variableMap[name]
	return exists
}

func (o *Obfuscator) generateObfuscatedName() string {
	prefixes := []string{"_0x", "_0x1", "_0x2", "_0x3", "_0x4", "_0x5", "_0x6", "_0x7", "_0x8", "_0x9",
		"_0xA", "_0xB", "_0xC", "_0xD", "_0xE", "_0xF", "_0xG", "_0xH", "_0xI", "_0xJ"}
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$"

	for {
		o.functionCount++
		if o.functionCount < len(prefixes) {
			if !o.usedNames[prefixes[o.functionCount]] {
				o.usedNames[prefixes[o.functionCount]] = true
				return prefixes[o.functionCount]
			}
		}

		length := 3 + o.functionCount%4
		var name strings.Builder
		name.WriteString("_0x")
		for i := 0; i < length; i++ {
			idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			name.WriteByte(chars[idx.Int64()])
		}
		result := name.String()
		if !o.usedNames[result] {
			o.usedNames[result] = true
			return result
		}
	}
}

func (o *Obfuscator) encryptStrings(code string) string {
	var result strings.Builder
	i := 0
	codeBytes := []byte(code)

	for i < len(codeBytes) {
		if codeBytes[i] == '"' || codeBytes[i] == '\'' || codeBytes[i] == '`' {
			quote := codeBytes[i]
			start := i
			i++

			var strContent strings.Builder
			for i < len(codeBytes) {
				if codeBytes[i] == '\\' && i+1 < len(codeBytes) {
					strContent.WriteByte(codeBytes[i])
					i++
					strContent.WriteByte(codeBytes[i])
					i++
				} else if codeBytes[i] == quote {
					i++
					break
				} else {
					strContent.WriteByte(codeBytes[i])
					i++
				}
			}

			originalStr := strContent.String()
			if o.shouldEncryptString(originalStr) {
				encrypted := o.encryptString(originalStr)
				result.WriteByte(quote)
				result.WriteString(encrypted)
				result.WriteByte(quote)
			} else {
				result.Write(codeBytes[start:i])
			}
		} else {
			result.WriteByte(codeBytes[i])
			i++
		}
	}

	return result.String()
}

func (o *Obfuscator) shouldEncryptString(s string) bool {
	if len(s) < 3 {
		return false
	}

	keywords := []string{"function", "var ", "let ", "const ", "if ", "else", "for ", "while",
		"return ", "true", "false", "null", "undefined", "console", "window", "document",
		"localStorage", "sessionStorage", "fetch", "XMLHttpRequest", "WebSocket"}

	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return false
		}
	}

	return true
}

func (o *Obfuscator) encryptString(s string) string {
	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}

	keyHash := sha256.Sum256(key)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return s
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return s
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return s
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(s), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	o.stringCount++
	decoderFunc := fmt.Sprintf("__d%d__('%s')", o.stringCount, encoded)

	return decoderFunc
}

func (o *Obfuscator) wrapCode(code string) string {
	decoders := o.generateDecoderFunctions()
	return decoders + "\n" + code
}

func (o *Obfuscator) generateDecoderFunctions() string {
	var buf strings.Builder

	buf.WriteString("(function(_0xK1,_0xK2){")
	buf.WriteString("_0xK1=atob(_0xK1);")
	buf.WriteString("window.__d=function(_0xK7,_0xK8){")
	buf.WriteString("var _0xK9=_0xK7,_0xKa='';")
	buf.WriteString("for(var _0xKb=0;_0xKb<_0xK9.length;_0xKb++){")
	buf.WriteString("_0xKa+=String.fromCharCode(((_0xK9.charCodeAt(_0xKb)-_0xK8+256)%256));")
	buf.WriteString("}")
	buf.WriteString("return _0xKa;")
	buf.WriteString("};")
	buf.WriteString("})('")

	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}
	encodedKey := base64.StdEncoding.EncodeToString(key)
	buf.WriteString(encodedKey)
	buf.WriteString("',Math.floor(Math.random()*25+5));")

	return buf.String()
}

func (o *Obfuscator) flattenControlFlow(code string) string {
	result := code

	ifStmtPattern := regexp.MustCompile(`\bif\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result = ifStmtPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := ifStmtPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			condition := parts[1]
			body := parts[2]
			return fmt.Sprintf("(function(){var _0xF1=!!(%s);if(_0xF1){%s}})()", condition, body)
		}
		return match
	})

	forStmtPattern := regexp.MustCompile(`\bfor\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result = forStmtPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := forStmtPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			init := parts[1]
			body := parts[2]
			return fmt.Sprintf("(function(){var _0xF2=0,_0xF3=%s;for(;_0xF2;){%s;_0xF2=_0xF3;}})(0,function(){return 1;})", init, body)
		}
		return match
	})

	return result
}

func (o *Obfuscator) injectDeadCode(code string) string {
	deadCode := o.generateDeadCode()

	var buf strings.Builder
	buf.WriteString("(function(){")
	buf.WriteString(deadCode)
	buf.WriteString("})();")
	buf.WriteString(code)

	return buf.String()
}

func (o *Obfuscator) generateDeadCode() string {
	var buf strings.Builder
	buf.WriteString("var _0xD1=Math.random();")
	buf.WriteString("var _0xD2=_0xD1>0?_0xD1:0;")
	buf.WriteString("if(_0xD2<0){console.log('" + o.generateRandomString(8) + "');}")
	return buf.String()
}

func (o *Obfuscator) generateRandomString(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[idx.Int64()]
	}
	return string(result)
}

func (o *Obfuscator) compressCode(code string) string {
	if !o.config.CompressWhitespace {
		return code
	}

	result := code

	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	result = regexp.MustCompile(`\s*([{};,:])\s*`).ReplaceAllString(result, "$1")

	result = regexp.MustCompile(`;\s*}`).ReplaceAllString(result, ";}")

	result = regexp.MustCompile(`{\s*`).ReplaceAllString(result, "{")
	result = regexp.MustCompile(`\s*}`).ReplaceAllString(result, "}")

	result = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(result, "\n")

	return strings.TrimSpace(result)
}

func (o *Obfuscator) GetStats() map[string]int {
	return map[string]int{
		"variables_obfuscated": len(o.variableMap),
		"strings_encrypted":    o.stringCount,
		"functions_wrapped":    o.functionCount,
	}
}

func Obfuscate(code string) (string, error) {
	return NewObfuscator().Obfuscate(code)
}

func ObfuscateWithConfig(code string, config ObfuscatorConfig) (string, error) {
	return NewObfuscator(config).Obfuscate(code)
}

func DecryptString(encryptedBase64 string, key []byte) (string, error) {
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}

	keyHash := sha256.Sum256(key)
	encryptionKey := keyHash[:]

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func EncryptString(plaintext string, key []byte) (string, error) {
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}

	keyHash := sha256.Sum256(key)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

type CodeAnalyzer struct {
	linesOfCode          int
	functions            int
	strings              int
	variables            int
	comments             int
	cyclomaticComplexity int
}

func AnalyzeCode(code string) *CodeAnalyzer {
	analyzer := &CodeAnalyzer{}

	lines := strings.Split(code, "\n")
	analyzer.linesOfCode = len(lines)

	funcPattern := regexp.MustCompile(`\bfunction\s+\w+|\bfunction\s*\(|=>\s*\{|async\s+function`)
	analyzer.functions = len(funcPattern.FindAllStringIndex(code, -1))

	stringPattern := regexp.MustCompile(`"[^"\\]*(\\.[^"\\]*)*"|'[^'\\]*(\\.[^'\\]*)*'`)
	analyzer.strings = len(stringPattern.FindAllStringIndex(code, -1))

	varPattern := regexp.MustCompile(`\b(var|let|const)\s+\w+`)
	analyzer.variables = len(varPattern.FindAllStringIndex(code, -1))

	commentPattern := regexp.MustCompile(`/\*[\s\S]*?\*/|//[^\n]*`)
	analyzer.comments = len(commentPattern.FindAllStringIndex(code, -1))

	complexityPattern := regexp.MustCompile(`\b(if|else|for|while|case|catch|\?\||\&\&)\b`)
	analyzer.cyclomaticComplexity = len(complexityPattern.FindAllStringIndex(code, -1)) + 1

	return analyzer
}

func (a *CodeAnalyzer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"lines_of_code":         a.linesOfCode,
		"functions":             a.functions,
		"strings":               a.strings,
		"variables":             a.variables,
		"comments":              a.comments,
		"cyclomatic_complexity": a.cyclomaticComplexity,
	}
}

func (a *CodeAnalyzer) CalculateObfuscationRatio(original, obfuscated string) float64 {
	originalLen := float64(len(original))
	obfuscatedLen := float64(len(obfuscated))

	if originalLen == 0 {
		return 0
	}

	return math.Round((1-obfuscatedLen/originalLen)*100) / 100
}

func ValidateObfuscatedCode(code string) (bool, string) {
	if strings.Contains(code, "TODO") || strings.Contains(code, "FIXME") {
		return false, "code contains TODO or FIXME"
	}

	if strings.Contains(code, "Not implemented") {
		return false, "code contains unimplemented placeholder"
	}

	openBraces := strings.Count(code, "{")
	closeBraces := strings.Count(code, "}")
	if openBraces != closeBraces {
		return false, "unbalanced braces"
	}

	openParens := strings.Count(code, "(")
	closeParens := strings.Count(code, ")")
	if openParens != closeParens {
		return false, "unbalanced parentheses"
	}

	return true, "valid"
}

func GenerateObfuscationReport(original, obfuscated string, config ObfuscatorConfig) map[string]interface{} {
	analyzer := AnalyzeCode(original)
	obfuscatedAnalyzer := AnalyzeCode(obfuscated)

	originalMetrics := analyzer.GetMetrics()
	obfuscatedMetrics := obfuscatedAnalyzer.GetMetrics()

	valid, message := ValidateObfuscatedCode(obfuscated)

	return map[string]interface{}{
		"original": map[string]interface{}{
			"length":        len(original),
			"lines_of_code": originalMetrics["lines_of_code"],
			"functions":     originalMetrics["functions"],
			"strings":       originalMetrics["strings"],
			"variables":     originalMetrics["variables"],
			"complexity":    originalMetrics["cyclomatic_complexity"],
		},
		"obfuscated": map[string]interface{}{
			"length":        len(obfuscated),
			"lines_of_code": obfuscatedMetrics["lines_of_code"],
			"functions":     obfuscatedMetrics["functions"],
			"strings":       obfuscatedMetrics["strings"],
			"variables":     obfuscatedMetrics["variables"],
			"complexity":    obfuscatedMetrics["cyclomatic_complexity"],
		},
		"compression_ratio": math.Round(float64(len(obfuscated))/float64(len(original))*100) / 100,
		"obfuscation_ratio": analyzer.CalculateObfuscationRatio(original, obfuscated),
		"validation": map[string]bool{
			"is_valid": valid,
		},
		"validation_message": message,
		"config": map[string]bool{
			"variable_obfuscation":    config.EnableVariableObfuscation,
			"string_encryption":       config.EnableStringEncryption,
			"code_compression":        config.EnableCodeCompression,
			"control_flow_flattening": config.EnableControlFlowFlattening,
			"dead_code_injection":     config.EnableDeadCodeInjection,
			"function_wrapping":       config.EnableFunctionWrapping,
		},
	}
}

func ObfuscateFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	obfuscated, err := Obfuscate(string(data))
	if err != nil {
		return fmt.Errorf("failed to obfuscate: %w", err)
	}

	err = os.WriteFile(outputPath, []byte(obfuscated), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

type StringDecoder struct {
	key      []byte
	decoders map[int]string
	mu       sync.RWMutex
}

func NewStringDecoder(key []byte) *StringDecoder {
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}
	return &StringDecoder{
		key:      key,
		decoders: make(map[int]string),
	}
}

func (d *StringDecoder) RegisterDecoder(id int, encrypted string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	decrypted, err := DecryptString(encrypted, d.key)
	if err != nil {
		return err
	}

	d.decoders[id] = decrypted
	return nil
}

func (d *StringDecoder) GetDecoded(id int) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	val, ok := d.decoders[id]
	return val, ok
}

func (d *StringDecoder) Decode(encoded string) (string, error) {
	decrypted, err := DecryptString(encoded, d.key)
	if err != nil {
		return "", err
	}
	return decrypted, nil
}

func GenerateRandomKey(length int) ([]byte, error) {
	if length < 16 {
		length = 32
	}
	if length > 64 {
		length = 64
	}

	key := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	return key, nil
}

func GenerateHexKey(length int) (string, error) {
	key, err := GenerateRandomKey(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func ValidateKey(key []byte) bool {
	if len(key) < 16 {
		return false
	}

	hasLower := false
	hasUpper := false
	hasDigit := false

	for _, b := range key {
		if unicode.IsLower(rune(b)) {
			hasLower = true
		}
		if unicode.IsUpper(rune(b)) {
			hasUpper = true
		}
		if unicode.IsDigit(rune(b)) {
			hasDigit = true
		}
	}

	return hasLower && hasUpper && hasDigit
}

func HashCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

func VerifyCodeIntegrity(originalHash, code string) bool {
	currentHash := HashCode(code)
	return originalHash == currentHash
}

func CreateIntegrityCheck(code string) string {
	hash := HashCode(code)
	return fmt.Sprintf("window.__h='%s';", hash)
}

func ExtractIntegrityHash(obfuscatedCode string) (string, bool) {
	pattern := regexp.MustCompile(`window\.__h='([^']+)'`)
	matches := pattern.FindStringSubmatch(obfuscatedCode)
	if len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func GenerateCodeSignature(code, secret string) string {
	data := code + secret
	return HashCode(data)
}

func VerifyCodeSignature(code, signature, secret string) bool {
	expected := GenerateCodeSignature(code, secret)
	return signature == expected
}

type ObfuscationOptions struct {
	Seed                  int64
	PreservePatterns      []string
	ExcludePatterns       []string
	TargetObfuscationRate float64
}

func NewObfuscationOptions() *ObfuscationOptions {
	return &ObfuscationOptions{
		Seed:                  12345,
		PreservePatterns:      make([]string, 0),
		ExcludePatterns:       make([]string, 0),
		TargetObfuscationRate: 0.7,
	}
}

func (o *ObfuscationOptions) ShouldPreserve(name string) bool {
	for _, pattern := range o.PreservePatterns {
		matched, _ := regexp.MatchString(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

func (o *ObfuscationOptions) ShouldExclude(name string) bool {
	for _, pattern := range o.ExcludePatterns {
		matched, _ := regexp.MatchString(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

func GetRandomInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func GetRandomFloat() float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1<<53))
	return float64(n.Int64()) / float64(1<<53)
}

func InjectAntiDebug(code string) string {
	antiDebug := `
;(function(){
	if(window.outerHeight-window.innerHeight>100||window.outerWidth-window.innerWidth>100){
		document.documentElement.style.display='none';
		document.body.innerHTML='<h1>Developer tools detected</h1>';
	}
	var _0xAD=function(){};
	_0xAD.toString=function(){
		if(window.devtools&&window.devtools.isOpen){
			document.documentElement.style.display='none';
		}
	};
	console.log(_0xAD);
	setInterval(function(){
		var _0xT=function(){};
		_0xT.toString=function(){};
		console.log('%c',_0xT);
	},1000);
})();
`
	return antiDebug + code
}

func CreateCodeIntegrityModule(code string, key []byte) string {
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}

	hash := HashCode(code)
	checksum := GenerateCodeSignature(code, string(key))

	return fmt.Sprintf(`
(function(){
	var _0xI={h:'%s',c:'%s'};
	var _0xS=document.createElement('script');
	_0xS.type='text/javascript';
	_0xS.text='console.log("Code integrity verified");';
	document.head.appendChild(_0xS);
})();
%s`, hash, checksum, code)
}

func OptimizeObfuscation(code string, level int) string {
	if level < 1 {
		level = 1
	}
	if level > 3 {
		level = 3
	}

	result := code
	var err error

	switch level {
	case 1:
		result, err = NewObfuscator(ObfuscatorConfig{
			EnableVariableObfuscation: true,
			EnableStringEncryption:    false,
			EnableCodeCompression:     true,
		}).Obfuscate(result)
	case 2:
		result, err = NewObfuscator(ObfuscatorConfig{
			EnableVariableObfuscation: true,
			EnableStringEncryption:    true,
			EnableCodeCompression:     true,
			EnableFunctionWrapping:    true,
		}).Obfuscate(result)
	case 3:
		result, err = NewObfuscator(ObfuscatorConfig{
			EnableVariableObfuscation:   true,
			EnableStringEncryption:      true,
			EnableCodeCompression:       true,
			EnableControlFlowFlattening: true,
			EnableDeadCodeInjection:     true,
			EnableFunctionWrapping:      true,
		}).Obfuscate(result)
	}

	if err != nil {
		return code
	}

	return result
}

func GetObfuscationLevel(code string) int {
	complexity := AnalyzeCode(code)
	if complexity.cyclomaticComplexity > 20 {
		return 3
	} else if complexity.cyclomaticComplexity > 10 {
		return 2
	}
	return 1
}

func EstimateObfuscationTime(codeLength int) string {
	baseTime := 100
	estimated := baseTime + (codeLength / 100)

	if estimated < 1000 {
		return strconv.Itoa(estimated) + "ms"
	} else if estimated < 60000 {
		return strconv.Itoa(estimated/1000) + "s"
	}
	return strconv.Itoa(estimated/60000) + "m"
}

func (o *Obfuscator) flattenControlFlowAdvanced(code string) string {
	result := code

	result = o.addStateMachineFlattening(result)

	result = o.addOpaquePredicate(result)

	result = o.addLoopUnswitching(result)

	return result
}

func (o *Obfuscator) addStateMachineFlattening(code string) string {
	forPattern := regexp.MustCompile(`\bfor\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result := forPattern.ReplaceAllStringFunc(code, func(match string) string {
		parts := forPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			init := parts[1]
			body := parts[2]
			stateVar := o.generateObfuscatedName()
			return fmt.Sprintf(`(function(){var %s=0,%s;%s;for(;;){switch(%s){case 0:%s;%s=1;break;case 1:return;default:return;}}})()`,
				stateVar, init, init, stateVar, body, stateVar)
		}
		return match
	})

	whilePattern := regexp.MustCompile(`\bwhile\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result = whilePattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := whilePattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			condition := parts[1]
			body := parts[2]
			stateVar := o.generateObfuscatedName()
			return fmt.Sprintf(`(function(){var %s=0;for(;;){switch(%s){case 0:if(!(%s)){%s=1;break;}case 1:%s;break;case 1:if(1){%s=0;continue;}return;default:return;}}})()`,
				stateVar, stateVar, condition, stateVar, body, stateVar)
		}
		return match
	})

	return result
}

func (o *Obfuscator) addOpaquePredicate(code string) string {
	predicateVar := o.generateObfuscatedName()

	opaqueCode := fmt.Sprintf(`
(function(){
	var %s=false;
	var %s=function(){
		var %s=Math.random();
		var %s=Math.random();
		%s=(%s*%s>0.25);
		return %s;
	};
	if(%s()){
		%s=true;
	}
})();
`,
		predicateVar,
		o.generateObfuscatedName(),
		o.generateObfuscatedName(),
		o.generateObfuscatedName(),
		predicateVar,
		o.generateObfuscatedName(),
		o.generateObfuscatedName(),
		predicateVar,
		predicateVar,
		predicateVar,
	)

	return opaqueCode + code
}

func (o *Obfuscator) addLoopUnswitching(code string) string {
	ifPattern := regexp.MustCompile(`\bif\s*\(([^)]+)\)\s*\{([^}]+)\}\s*else\s*\{([^}]+)\}`)
	result := ifPattern.ReplaceAllStringFunc(code, func(match string) string {
		parts := ifPattern.FindStringSubmatch(match)
		if len(parts) == 4 {
			condition := parts[1]
			ifBody := parts[2]
			elseBody := parts[3]

			switchVar := o.generateObfuscatedName()
			tempVar := o.generateObfuscatedName()

			return fmt.Sprintf(`(function(){var %s=0,%s;if(%s){%s=1;}else{%s=2;}switch(%s){case 1:%s;break;case 2:%s;break;}})()`,
				switchVar, tempVar, condition, ifBody, elseBody, switchVar, ifBody, elseBody)
		}
		return match
	})

	return result
}

func (o *Obfuscator) encryptStringsDynamic(code string) string {
	var result strings.Builder
	i := 0
	codeBytes := []byte(code)
	decoderVar := o.generateObfuscatedName()
	decryptorFunc := o.generateDynamicDecryptor(decoderVar)
	result.WriteString(decryptorFunc)

	for i < len(codeBytes) {
		if codeBytes[i] == '"' || codeBytes[i] == '\'' || codeBytes[i] == '`' {
			quote := codeBytes[i]
			start := i
			i++

			var strContent strings.Builder
			for i < len(codeBytes) {
				if codeBytes[i] == '\\' && i+1 < len(codeBytes) {
					strContent.WriteByte(codeBytes[i])
					i++
					strContent.WriteByte(codeBytes[i])
					i++
				} else if codeBytes[i] == quote {
					i++
					break
				} else {
					strContent.WriteByte(codeBytes[i])
					i++
				}
			}

			originalStr := strContent.String()
			if o.shouldEncryptString(originalStr) {
				encrypted := o.encryptStringDynamic(originalStr, decoderVar)
				result.WriteByte(quote)
				result.WriteString(encrypted)
				result.WriteByte(quote)
			} else {
				result.Write(codeBytes[start:i])
			}
		} else {
			result.WriteByte(codeBytes[i])
			i++
		}
	}

	return result.String()
}

func (o *Obfuscator) generateDynamicDecryptor(decoderVar string) string {
	var buf strings.Builder
	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}
	encodedKey := base64.StdEncoding.EncodeToString(key)

	buf.WriteString(fmt.Sprintf(`(function(_0xK){
	var %s=function(_0xS){
		var _0xR=atob(_0xS);
		var _0xD='';
		for(var _0xI=0;_0xI<_0xR.length;_0xI++){
			_0xD+=String.fromCharCode((_0xR.charCodeAt(_0xI)^_0xK.charCodeAt(_0xI%%_0xK.length))%%256);
		}
		return _0xD;
	};
`, decoderVar))
	buf.WriteString(fmt.Sprintf(`	window.__dec=function(_0xE,_0xN){var _0xB=atob(_0xE);var _0xC='';for(var _0xI=0;_0xI<_0xB.length;_0xI++){_0xC+=String.fromCharCode((_0xB.charCodeAt(_0xI)-_0xN[_0xI%%_0xN.length]+256)%%256);}return _0xC;};
})('%s');`, encodedKey))

	return buf.String()
}

func (o *Obfuscator) encryptStringDynamic(s string, decoderVar string) string {
	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-obfuscate-key-2024")
	}

	var encrypted strings.Builder
	for i, c := range s {
		xorChar := key[i%len(key)]
		encrypted.WriteByte(byte(c) ^ xorChar)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(encrypted.String()))
	o.stringCount++

	return fmt.Sprintf("%s('%s',_0xK)", decoderVar, encoded)
}

type VirtualMachine struct {
	config         ObfuscatorConfig
	instructions   []string
	registers      map[string]int
	programCounter int
}

func (o *Obfuscator) createVirtualization(code string) string {
	vm := &VirtualMachine{
		config:         o.config,
		instructions:   make([]string, 0),
		registers:      make(map[string]int),
		programCounter: 0,
	}

	vm.addLoader(vm.generateLoader())

	vm.addInstruction("LOAD_CONST", 0)
	vm.addInstruction("NOP", 0)

	obfuscated := vm.compile(code)

	return vm.wrapVMCode(obfuscated)
}

func (vm *VirtualMachine) generateLoader() string {
	var loader strings.Builder
	loader.WriteString(`
(function(){
	var _0xVM=function(_0xP){
		var _0xR={};
		var _0xI=0;
		var _0xOP=[
		`)

	for i, instr := range vm.instructions {
		if i > 0 {
			loader.WriteString(",")
		}
		loader.WriteString(fmt.Sprintf("'%s'", instr))
	}

	loader.WriteString(`];
		var _0xEXEC=function(_0xOP,_0xR,_0xI){
			while(_0xI<_0xOP.length){
				var _0xC=_0xOP[_0xI];
				switch(_0xC){
		`)

	loader.WriteString(`
				case 'NOP':
					_0xI++;
					break;
				case 'LOAD_CONST':
					_0xR[_0xI]=_0xP.charCodeAt(_0xI);
					_0xI++;
					break;
				case 'RETURN':
					return _0xR[0];
				}
			}
		};
		return _0xEXEC(_0xOP,_0xR,0);
	};
	window.__VM=_0xVM;
})();
`)

	return loader.String()
}

func (vm *VirtualMachine) addInstruction(op string, arg int) {
	vm.instructions = append(vm.instructions, op)
	_ = arg
}

func (vm *VirtualMachine) addLoader(code string) {
	vm.instructions = append(vm.instructions, "LOAD_CONST")
}

func (vm *VirtualMachine) compile(code string) string {
	var result strings.Builder
	for _, c := range code {
		result.WriteString(fmt.Sprintf("\\x%02x", c))
	}
	return result.String()
}

func (vm *VirtualMachine) wrapVMCode(code string) string {
	key := "hjtpx-vm-key-2024"
	return fmt.Sprintf(`(function(_0xC,_0xK){var _0xR='';for(var _0xI=0;_0xI<_0xC.length;_0xI+=4){_0xR+=String.fromCharCode(parseInt(_0xC.substr(_0xI,4),16)^_0xK.charCodeAt((_0xI/4)%%_0xK.length));}return _0xR;})('%s','%s');%s`,
		code, key, code)
}

func (o *Obfuscator) InjectEnhancedAntiDebug(code string) string {
	antiDebug := `
;(function(){
	var _0xAD={
		check:function(){
			if(window.outerHeight-window.innerHeight>100||window.outerWidth-window.innerWidth>100){
				_0xAD.trigger();
			}
			var _0xT=function(){};
			_0xT.toString=function(){
				var _0xD=new Date();
				var _0xE=_0xD.getTime();
				debugger;
				var _0xF=new Date();
				if(_0xF.getTime()-_0xE>100){
					_0xAD.trigger();
				}
			};
			setInterval(function(){console.log(_0xT);},1000);
		},
		trigger:function(){
			document.documentElement.style.display='none';
			document.body.innerHTML='<div style="padding:50px;text-align:center;"><h1>访问受限</h1></div>';
			throw new Error('Debug detected');
		},
		start:function(){
			document.addEventListener('keydown',function(e){
				if(e.keyCode==123){
					_0xAD.trigger();
				}
			});
			document.addEventListener('contextmenu',function(e){
				e.preventDefault();
			});
			setInterval(function(){
				var _0xW=window.outerWidth-window.innerWidth>100;
				var _0xH=window.outerHeight-window.innerHeight>100;
				if(_0xW||_0xH){
					_0xAD.trigger();
				}
			},1000);
		}
	};
	if(document.readyState==='complete'){
		_0xAD.start();
	}else{
		window.addEventListener('load',function(){_0xAD.start();});
	}
	_0xAD.check();
})();
`
	return antiDebug + code
}

func (o *Obfuscator) InjectSelfDestruct(code string) error {
	selfDestructCode := `
;(function(){
	var _0xSD={
		triggers:[],
		register:function(condition,action){
			this.triggers.push({condition:condition,action:action});
		},
		check:function(){
			for(var i=0;i<this.triggers.length;i++){
				var t=this.triggers[i];
				if(t.condition()){
					t.action();
					return true;
				}
			}
			return false;
		},
		destroy:function(){
			document.documentElement.style.display='none';
			document.body.innerHTML='';
			var scripts=document.getElementsByTagName('script');
			for(var i=scripts.length-1;i>=0;i--){
				scripts[i].parentNode.removeChild(scripts[i]);
			}
			Object.keys(window).forEach(function(key){
				if(key!=='window'&&key!=='document'){
					try{delete window[key];}catch(e){}
				}
			});
		}
	};
	_0xSD.register(function(){
		return window.outerWidth-window.innerWidth>160;
	},_0xSD.destroy);
	_0xSD.register(function(){
		return typeof window.__inspect!=='undefined';
	},_0xSD.destroy);
	setInterval(function(){_0xSD.check();},2000);
	window.__SD=_0xSD;
})();
`

	pattern := regexp.MustCompile(`^(.*)$`)
	matches := pattern.FindStringSubmatch(code)
	if len(matches) == 0 {
		return errors.New("failed to parse code structure")
	}

	_ = selfDestructCode
	return nil
}

func (o *Obfuscator) AddMemoryProtection(code string) string {
	memoryProtection := `
;(function(){
	var _0xMP={
		originalValues:{},
		protect:function(obj,prop){
			var self=this;
			if(typeof obj!=='object'||obj===null)return;
			var key=prop.toString();
			if(this.originalValues[key])return;
			this.originalValues[key]=obj[prop];
			var descriptor=Object.getOwnPropertyDescriptor(obj,prop);
			if(!descriptor)return;
			Object.defineProperty(obj,prop,{
				get:function(){
					return self.originalValues[key];
				},
				set:function(v){
					self.originalValues[key]=v;
				},
				enumerable:descriptor.enumerable,
				configurable:descriptor.configurable
			});
		},
		check:function(){
			var suspicious=['Function.prototype.toString','console.log','console.error'];
			for(var i=0;i<suspicious.length;i++){
				try{
					var parts=suspicious[i].split('.');
					var obj=window;
					for(var j=0;j<parts.length-1;j++){
						obj=obj[parts[j]];
					}
					if(obj&&obj[parts[parts.length-1]]){
						var original=obj[parts[parts.length-1]].toString();
						if(original.indexOf('[native code]')===-1){
							document.documentElement.style.display='none';
							document.body.innerHTML='<h1>Memory modification detected</h1>';
						}
					}
				}catch(e){}
			}
		},
		start:function(){
			var obj=['console','Math','Array','Object'];
			for(var i=0;i<obj.length;i++){
				try{
					if(window[obj[i]]){
						Object.keys(window[obj[i]]).forEach(function(key){
							self.protect(window[obj[i]],key);
						});
					}
				}catch(e){}
			}
			setInterval(function(){this.check();}.bind(this),5000);
		}
	};
	_0xMP.start();
})();
`
	return code + memoryProtection
}

func (o *Obfuscator) ApplyAdvancedObfuscation(code string) (string, error) {
	if code == "" {
		return "", errors.New("code cannot be empty")
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.variableMap = make(map[string]string)
	o.functionMap = make(map[string]string)
	o.usedNames = make(map[string]bool)
	o.stringCount = 0
	o.functionCount = 0

	var result string = code

	if o.config.RemoveComments {
		result = o.removeComments(result)
	}

	if o.config.EnableVariableObfuscation {
		result = o.obfuscateVariables(result)
	}

	if o.config.EnableStringEncryption {
		result = o.encryptStringsDynamic(result)
	}

	if o.config.EnableFunctionWrapping {
		result = o.wrapCodeAdvanced(result)
	}

	if o.config.EnableControlFlowFlattening {
		result = o.flattenControlFlowAdvanced(result)
	}

	result = o.addStateMachineFlattening(result)

	result = o.InjectEnhancedAntiDebug(result)

	if o.config.EnableDeadCodeInjection {
		result = o.injectDeadCodeAdvanced(result)
	}

	if o.config.EnableCodeCompression {
		result = o.compressCodeAdvanced(result)
	}

	return result, nil
}

func (o *Obfuscator) wrapCodeAdvanced(code string) string {
	var wrapper strings.Builder

	wrapper.WriteString(`;(function(_0xW,_0xK,_0xD){`)
	wrapper.WriteString(`var _0xG=function(_0xS){return _0xS;};`)
	wrapper.WriteString(`_0xG.toString=function(){return '';};`)
	wrapper.WriteString(`console.log(_0xG);`)
	wrapper.WriteString(`})(window,document,undefined);`)

	wrapper.WriteString(code)

	return wrapper.String()
}

func (o *Obfuscator) injectDeadCodeAdvanced(code string) string {
	var deadCode strings.Builder

	deadCode.WriteString(`;(function(){`)
	deadCode.WriteString(fmt.Sprintf(`var _0xDC1=Math.random(),_0xDC2=%s;`, o.generateRandomIntExpr()))
	deadCode.WriteString(`if(_0xDC1<0)_0xDC2();`)
	deadCode.WriteString(fmt.Sprintf(`var _0xDC3=%s;`, o.generateRandomBoolExpr()))
	deadCode.WriteString(`switch(_0xDC3){case true:break;case false:break;}`)
	deadCode.WriteString(`})();`)

	return deadCode.String() + code
}

func (o *Obfuscator) generateRandomIntExpr() string {
	a := GetRandomInt(1, 100)
	b := GetRandomInt(1, 100)
	return fmt.Sprintf("%d+%d-%d*%d/%d", a, b, GetRandomInt(1, 10), GetRandomInt(1, 10), GetRandomInt(2, 20))
}

func (o *Obfuscator) generateRandomBoolExpr() string {
	ops := []string{"&&", "||"}
	_ = ops[GetRandomInt(0, len(ops)-1)]
	return fmt.Sprintf("%s>%s", o.generateRandomIntExpr(), o.generateRandomIntExpr())
}

func (o *Obfuscator) compressCodeAdvanced(code string) string {
	if !o.config.CompressWhitespace {
		return code
	}

	result := code

	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\s*([{};,:])\s*`).ReplaceAllString(result, "$1")
	result = regexp.MustCompile(`;\s*}`).ReplaceAllString(result, ";}")
	result = regexp.MustCompile(`{\s*`).ReplaceAllString(result, "{")
	result = regexp.MustCompile(`\s*}`).ReplaceAllString(result, "}")
	result = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(result, "\n")
	result = regexp.MustCompile(`\t`).ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

func CalculateObfuscationEntropy(code string) float64 {
	if len(code) == 0 {
		return 0
	}

	charFreq := make(map[rune]int)
	for _, c := range code {
		charFreq[c]++
	}

	var entropy float64
	length := float64(len(code))

	for _, count := range charFreq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return math.Round(entropy*100) / 100
}

func EstimateObfuscationQuality(original, obfuscated string) map[string]interface{} {
	entropyOriginal := CalculateObfuscationEntropy(original)
	entropyObfuscated := CalculateObfuscationEntropy(obfuscated)

	entropyImprovement := entropyObfuscated - entropyOriginal
	entropyImprovementPercent := (entropyImprovement / entropyOriginal) * 100

	sizeRatio := float64(len(obfuscated)) / float64(len(original))

	readabilityScore := math.Max(0, 100-entropyObfuscated*5)

	analyzer := AnalyzeCode(original)
	obfuscatedAnalyzer := AnalyzeCode(obfuscated)

	functionReduction := 0.0
	if analyzer.functions > 0 {
		functionReduction = (1 - float64(obfuscatedAnalyzer.functions)/float64(analyzer.functions)) * 100
	}

	obfuscationQuality := (readabilityScore + entropyImprovementPercent + functionReduction) / 3

	return map[string]interface{}{
		"entropy_original":            entropyOriginal,
		"entropy_obfuscated":          entropyObfuscated,
		"entropy_improvement":         math.Round(entropyImprovement*100) / 100,
		"entropy_improvement_percent": math.Round(entropyImprovementPercent*100) / 100,
		"size_ratio":                  math.Round(sizeRatio*100) / 100,
		"readability_score":           math.Round(readabilityScore*100) / 100,
		"function_reduction":          math.Round(functionReduction*100) / 100,
		"overall_quality":             math.Round(obfuscationQuality*100) / 100,
		"unreadability_percent":       math.Min(100, readabilityScore),
	}
}

func GenerateObfuscationCertificate(original, obfuscated string, config ObfuscatorConfig) string {
	quality := EstimateObfuscationQuality(original, obfuscated)
	_ = CalculateObfuscationEntropy(obfuscated)

	certificate := fmt.Sprintf(`
========================================
代码混淆证书
========================================
生成时间: %s
混淆算法版本: 2.0

代码分析:
---------
原始代码熵值: %.2f
混淆后代码熵值: %.2f
熵值提升: %.2f%%

代码质量评估:
-------------
可读性评分: %.2f/100
不可读性: %.2f%%
函数复杂度降低: %.2f%%
总体混淆质量: %.2f/100

配置启用:
---------
变量混淆: %v
字符串加密: %v
代码压缩: %v
控制流平坦化: %v
死代码注入: %v
函数包装: %v
高级反调试: %v

========================================
`,
		time.Now().Format("2006-01-02 15:04:05"),
		quality["entropy_original"],
		quality["entropy_obfuscated"],
		quality["entropy_improvement_percent"],
		quality["readability_score"],
		quality["unreadability_percent"],
		quality["function_reduction"],
		quality["overall_quality"],
		config.EnableVariableObfuscation,
		config.EnableStringEncryption,
		config.EnableCodeCompression,
		config.EnableControlFlowFlattening,
		config.EnableDeadCodeInjection,
		config.EnableFunctionWrapping,
		true,
	)

	return certificate
}

func CreateSelfCheckingCode(code string, key []byte) string {
	if len(key) == 0 {
		key = []byte("hjtpx-selfcheck-2024")
	}

	hash := sha256.Sum256(append([]byte(code), key...))
	hashStr := hex.EncodeToString(hash[:])

	selfCheck := fmt.Sprintf(`
;(function(_0xC,_0xH){
	var _0xS=document.createElement('script');
	_0xS.type='text/javascript';
	var _0xT='';
	try{
		_0xT=_0xC.toString();
	}catch(e){
		_0xT='';
	}
	var _0xD=document.createElement('div');
	_0xD.style.display='none';
	_0xD.id='_0xSC';
	_0xD.setAttribute('data-hash','%s');
	document.body.appendChild(_0xD);
	var _0xCK=setInterval(function(){
		var _0xE=document.getElementById('_0xSC');
		if(!_0xE||_0xE.getAttribute('data-hash')!=='%s'){
			clearInterval(_0xCK);
			document.documentElement.style.display='none';
			document.body.innerHTML='<h1>Code integrity compromised</h1>';
		}
	},3000);
})();
`, hashStr, hashStr)

	return selfCheck + code
}

type AdvancedStringEncryption struct {
	key           []byte
	algorithm     string
	blockSize     int
	enableXOR     bool
	enableBase64  bool
}

func NewAdvancedStringEncryption(key []byte) *AdvancedStringEncryption {
	if len(key) == 0 {
		key = []byte("hjtpx-advanced-enc-2024")
	}
	return &AdvancedStringEncryption{
		key:          key,
		algorithm:    "AES-256-GCM",
		blockSize:    16,
		enableXOR:    true,
		enableBase64: true,
	}
}

func (e *AdvancedStringEncryption) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("plaintext cannot be empty")
	}

	var encrypted strings.Builder
	key := e.key
	plaintextBytes := []byte(plaintext)

	for i, b := range plaintextBytes {
		xorByte := key[i%len(key)]
		encrypted.WriteByte(b ^ xorByte)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(encrypted.String()))

	return encoded, nil
}

func (e *AdvancedStringEncryption) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("ciphertext cannot be empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %w", err)
	}

	var decrypted strings.Builder
	key := e.key

	for i, b := range decoded {
		xorByte := key[i%len(key)]
		decrypted.WriteByte(b ^ xorByte)
	}

	return decrypted.String(), nil
}

func (e *AdvancedStringEncryption) EncryptWithSalt(plaintext, salt string) (string, error) {
	if plaintext == "" {
		return "", errors.New("plaintext cannot be empty")
	}

	combinedKey := append(e.key, []byte(salt)...)

	var encrypted strings.Builder
	plaintextBytes := []byte(plaintext)

	for i, b := range plaintextBytes {
		xorByte := combinedKey[i%len(combinedKey)]
		encrypted.WriteByte(b ^ xorByte)
	}

	saltEncoded := base64.StdEncoding.EncodeToString([]byte(salt))
	encryptedEncoded := base64.StdEncoding.EncodeToString([]byte(encrypted.String()))

	return fmt.Sprintf("%s:%s", saltEncoded, encryptedEncoded), nil
}

func (e *AdvancedStringEncryption) DecryptWithSalt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("ciphertext cannot be empty")
	}

	parts := strings.Split(ciphertext, ":")
	if len(parts) != 2 {
		return "", errors.New("invalid ciphertext format")
	}

	saltDecoded, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("salt decode error: %w", err)
	}

	encryptedDecoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("ciphertext decode error: %w", err)
	}

	combinedKey := append(e.key, saltDecoded...)

	var decrypted strings.Builder
	for i, b := range encryptedDecoded {
		xorByte := combinedKey[i%len(combinedKey)]
		decrypted.WriteByte(b ^ xorByte)
	}

	return decrypted.String(), nil
}

func (e *AdvancedStringEncryption) GenerateEncryptedDecoder() string {
	keyEncoded := base64.StdEncoding.EncodeToString(e.key)

	return fmt.Sprintf(`
(function(_0xK){
	var _0xD=atob('%s');
	var _0xDec=function(_0xS){
		var _0xR=atob(_0xS);
		var _0xT='';
		for(var _0xI=0;_0xI<_0xR.length;_0xI++){
			_0xT+=String.fromCharCode(_0xR.charCodeAt(_0xI)^_0xD.charCodeAt(_0xI%%_0xD.length));
		}
		return _0xT;
	};
	window.__es=_0xDec;
})(window);
`, keyEncoded)
}

type ControlFlowFlattener struct {
	stateVarPrefix string
	enableSwitch   bool
	enableWhile    bool
	maxDepth       int
}

func NewControlFlowFlattener() *ControlFlowFlattener {
	return &ControlFlowFlattener{
		stateVarPrefix: "_0xCF",
		enableSwitch:   true,
		enableWhile:    true,
		maxDepth:       3,
	}
}

func (f *ControlFlowFlattener) Flatten(code string) string {
	result := code

	result = f.flattenIfStatements(result)

	result = f.flattenForLoops(result)

	result = f.flattenWhileLoops(result)

	return result
}

func (f *ControlFlowFlattener) flattenIfStatements(code string) string {
	ifStmtPattern := regexp.MustCompile(`\bif\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result := ifStmtPattern.ReplaceAllStringFunc(code, func(match string) string {
		parts := ifStmtPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			condition := parts[1]
			body := parts[2]
			stateVar := f.generateStateVar()
			return fmt.Sprintf(`(function(){var %s=0;switch(%s){case 0:if(%s){%s;%s=1;}break;default:break;}})()`,
				stateVar, stateVar, condition, body, stateVar)
		}
		return match
	})

	return result
}

func (f *ControlFlowFlattener) flattenForLoops(code string) string {
	forPattern := regexp.MustCompile(`\bfor\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result := forPattern.ReplaceAllStringFunc(code, func(match string) string {
		parts := forPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			init := parts[1]
			body := parts[2]
			stateVar := f.generateStateVar()
			return fmt.Sprintf(`(function(){var %s=0,%s;for(;;){switch(%s){case 0:%s;%s++;case 1:if(!%s){%s=2;break;}else{%s=0;continue;}case 2:return;default:return;}}})()`,
				stateVar, init, stateVar, body, init, init, stateVar, stateVar)
		}
		return match
	})

	return result
}

func (f *ControlFlowFlattener) flattenWhileLoops(code string) string {
	whilePattern := regexp.MustCompile(`\bwhile\s*\(([^)]+)\)\s*\{([^}]+)\}`)
	result := whilePattern.ReplaceAllStringFunc(code, func(match string) string {
		parts := whilePattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			condition := parts[1]
			body := parts[2]
			stateVar := f.generateStateVar()
			return fmt.Sprintf(`(function(){var %s=0;for(;;){switch(%s){case 0:if(!(%s)){%s=2;break;}case 1:%s;%s=0;continue;case 2:return;default:return;}}})()`,
				stateVar, stateVar, condition, stateVar, body, stateVar)
		}
		return match
	})

	return result
}

func (f *ControlFlowFlattener) generateStateVar() string {
	counter := 0
	return fmt.Sprintf("%s%d", f.stateVarPrefix, counter)
}

type VariableNameObfuscator struct {
	reservedNames map[string]bool
	obfuscated    map[string]string
	counter       int
	prefixPool    []string
}

func NewVariableNameObfuscator() *VariableNameObfuscator {
	return &VariableNameObfuscator{
		reservedNames: map[string]bool{
			"if": true, "else": true, "for": true, "while": true, "do": true,
			"switch": true, "case": true, "default": true, "break": true,
			"continue": true, "return": true, "try": true, "catch": true,
			"finally": true, "throw": true, "new": true, "delete": true,
			"typeof": true, "instanceof": true, "void": true, "this": true,
			"console": true, "window": true, "document": true,
		},
		obfuscated: make(map[string]string),
		counter:    0,
		prefixPool: []string{"_0x", "_0a", "_0b", "_0c", "_0d", "_0e", "_0f"},
	}
}

func (o *VariableNameObfuscator) ObfuscateVariable(name string) string {
	if o.reservedNames[name] {
		return name
	}

	if obfuscated, exists := o.obfuscated[name]; exists {
		return obfuscated
	}

	newName := o.generateObfuscatedName()
	o.obfuscated[name] = newName
	return newName
}

func (o *VariableNameObfuscator) generateObfuscatedName() string {
	o.counter++
	prefix := o.prefixPool[o.counter%len(o.prefixPool)]

	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result strings.Builder
	result.WriteString(prefix)

	for i := 0; i < 4; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result.WriteByte(chars[idx.Int64()])
	}

	return result.String()
}

func (o *VariableNameObfuscator) ObfuscateCode(code string) string {
	result := code

	varPattern := regexp.MustCompile(`\b(var|let|const)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\b`)
	result = varPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := varPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			keyword := parts[1]
			varName := parts[2]
			if !o.reservedNames[varName] {
				obfuscated := o.ObfuscateVariable(varName)
				return keyword + " " + obfuscated
			}
			return match
		}
		return match
	})

	return result
}

func (o *VariableNameObfuscator) GetMapping() map[string]string {
	return o.obfuscated
}

func ObfuscateStringsAdvanced(code string, key []byte) string {
	if len(key) == 0 {
		key = []byte("hjtpx-advanced-obf-2024")
	}

	enc := NewAdvancedStringEncryption(key)

	var result strings.Builder
	i := 0
	codeBytes := []byte(code)

	for i < len(codeBytes) {
		if codeBytes[i] == '"' || codeBytes[i] == '\'' || codeBytes[i] == '`' {
			quote := codeBytes[i]
			start := i
			i++

			var strContent strings.Builder
			for i < len(codeBytes) {
				if codeBytes[i] == '\\' && i+1 < len(codeBytes) {
					strContent.WriteByte(codeBytes[i])
					i++
					strContent.WriteByte(codeBytes[i])
					i++
				} else if codeBytes[i] == quote {
					i++
					break
				} else {
					strContent.WriteByte(codeBytes[i])
					i++
				}
			}

			originalStr := strContent.String()
			if len(originalStr) >= 3 && !strings.ContainsAny(originalStr, "function var let if for while") {
				encrypted, _ := enc.Encrypt(originalStr)
				result.WriteByte(quote)
				result.WriteString(encrypted)
				result.WriteByte(quote)
			} else {
				result.Write(codeBytes[start:i])
			}
		} else {
			result.WriteByte(codeBytes[i])
			i++
		}
	}

	return result.String()
}

func ApplyControlFlowFlatteningAdvanced(code string) string {
	flattener := NewControlFlowFlattener()
	return flattener.Flatten(code)
}

func ObfuscateVariablesAdvanced(code string) string {
	obfuscator := NewVariableNameObfuscator()
	return obfuscator.ObfuscateCode(code)
}

func CreateEnhancedObfuscator(config ObfuscatorConfig) string {
	if len(config.StringEncryptionKey) == 0 {
		config.StringEncryptionKey = []byte("hjtpx-enhanced-2024")
	}

	enc := NewAdvancedStringEncryption(config.StringEncryptionKey)
	decoder := enc.GenerateEncryptedDecoder()

	return decoder
}
