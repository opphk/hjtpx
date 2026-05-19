package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ObfuscatorV2Config struct {
	EnableAdvancedStringEncryption   bool
	EnableEnhancedControlFlowFlattening bool
	EnableCodeSplitting              bool
	EnableEnhancedDeadCodeInjection  bool
	StringEncryptionKey              []byte
	EncryptionRounds                int
	SplitFragments                   int
	DeadCodeRatio                    float64
	EnableStringPooling              bool
	EnableStringCompression          bool
	EnableControlFlowReordering      bool
	EnableBogusControlFlow           bool
	EnableSubstitution               bool
	EnableReachableCodeShuffling     bool
	PreserveFunctionNames            []string
}

var defaultObfuscatorV2Config = ObfuscatorV2Config{
	EnableAdvancedStringEncryption:    true,
	EnableEnhancedControlFlowFlattening: true,
	EnableCodeSplitting:               true,
	EnableEnhancedDeadCodeInjection:   true,
	StringEncryptionKey:               []byte("hjtpx-v2-obfuscate-key-2024"),
	EncryptionRounds:                  3,
	SplitFragments:                    3,
	DeadCodeRatio:                     0.3,
	EnableStringPooling:               true,
	EnableStringCompression:           true,
	EnableControlFlowReordering:       true,
	EnableBogusControlFlow:            true,
	EnableSubstitution:                true,
	EnableReachableCodeShuffling:      true,
	PreserveFunctionNames:             []string{},
}

type ObfuscatorV2 struct {
	config        ObfuscatorV2Config
	stringPool    map[string]string
	fragmentMap   map[string]string
	mu            sync.Mutex
	stats         ObfuscationStats
	stringCount   int
	functionCount int
}

type ObfuscationStats struct {
	StringsEncrypted      int
	FragmentsCreated      int
	DeadCodeInjected      int
	ControlFlowFlattened  int
}

func NewObfuscatorV2(config ...ObfuscatorV2Config) *ObfuscatorV2 {
	cfg := defaultObfuscatorV2Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if len(cfg.StringEncryptionKey) == 0 {
		cfg.StringEncryptionKey = []byte("hjtpx-v2-obfuscate-key-2024")
	}

	return &ObfuscatorV2{
		config:        cfg,
		stringPool:    make(map[string]string),
		fragmentMap:   make(map[string]string),
		stats:         ObfuscationStats{},
		stringCount:   0,
		functionCount: 0,
	}
}

func (o *ObfuscatorV2) Obfuscate(code string) (string, error) {
	if code == "" {
		return "", errors.New("code cannot be empty")
	}

	o.stringPool = make(map[string]string)
	o.fragmentMap = make(map[string]string)
	o.stats = ObfuscationStats{}

	var result = code

	if o.config.EnableAdvancedStringEncryption {
		result = o.encryptStringsAdvanced(result)
	}

	if o.config.EnableEnhancedControlFlowFlattening {
		result = o.flattenControlFlowEnhanced(result)
	}

	if o.config.EnableCodeSplitting {
		result = o.splitCode(result)
	}

	if o.config.EnableEnhancedDeadCodeInjection {
		result = o.injectDeadCodeEnhanced(result)
	}

	if o.config.EnableControlFlowReordering {
		result = o.reorderControlFlow(result)
	}

	if o.config.EnableBogusControlFlow {
		result = o.injectBogusControlFlow(result)
	}

	return result, nil
}

// ==================== 字符串加密混淆增强 ====================

func (o *ObfuscatorV2) encryptStringsAdvanced(code string) string {
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
			if o.shouldEncryptStringV2(originalStr) {
				encrypted := o.encryptStringWithRounds(originalStr)
				result.WriteByte(quote)
				result.WriteString(encrypted)
				result.WriteByte(quote)

				o.stats.StringsEncrypted++
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

func (o *ObfuscatorV2) encryptStringWithRounds(s string) string {
	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-v2-obfuscate-key-2024")
	}

	rounds := o.config.EncryptionRounds
	if rounds < 1 {
		rounds = 1
	}
	if rounds > 5 {
		rounds = 5
	}

	encrypted := []byte(s)

	for round := 0; round < rounds; round++ {
		roundKey := append(key, byte(round))
		keyHash := sha256.Sum256(roundKey)

		xorKey := make([]byte, len(encrypted))
		for i := range xorKey {
			xorKey[i] = keyHash[i%32]
		}

		for i := range encrypted {
			encrypted[i] ^= xorKey[i%len(xorKey)]
		}

		encrypted = o.scrambleBytesV2(encrypted, round)
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)
	o.stringCount++

	return fmt.Sprintf("__v2d%d__('%s')", o.stringCount, encoded)
}

func (o *ObfuscatorV2) scrambleBytesV2(data []byte, seed int) []byte {
	result := make([]byte, len(data))
	for i := range data {
		j := (i*7 + seed*13) % len(data)
		result[j] = data[i]
	}
	return result
}

func (o *ObfuscatorV2) shouldEncryptStringV2(s string) bool {
	if len(s) < 3 {
		return false
	}

	keywords := []string{
		"function", "var ", "let ", "const ", "if ", "else", "for ", "while",
		"return ", "true", "false", "null", "undefined", "console", "window", "document",
		"localStorage", "sessionStorage", "fetch", "XMLHttpRequest", "WebSocket",
		"prototype", "constructor", "toString", "valueOf", "hasOwnProperty",
	}

	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return false
		}
	}

	return true
}

// ==================== 控制流扁平化增强 ====================

func (o *ObfuscatorV2) flattenControlFlowEnhanced(code string) string {
	result := code

	result = o.flattenIfEnhanced(result)
	result = o.flattenForEnhanced(result)
	result = o.flattenWhileEnhanced(result)
	result = o.addStateMachineWrapper(result)

	o.stats.ControlFlowFlattened++

	return result
}

func (o *ObfuscatorV2) flattenIfEnhanced(code string) string {
	ifPattern := regexp.MustCompile(`if\s*\(([^)]+)\)\s*\{([^}]+)\}\s*else\s*\{([^}]+)\}`)
	
	result := ifPattern.ReplaceAllStringFunc(code, func(match string) string {
		groups := ifPattern.FindStringSubmatch(match)
		if len(groups) < 4 {
			return match
		}

		condition := groups[1]
		thenBlock := groups[2]
		elseBlock := groups[3]

		stateVar := o.generateStateVar()
		stateInit := o.generateStateVar()

		flatControl := fmt.Sprintf(`var %s=%s?1:2;var %s=0;switch(%s){case 1:{var %s=1;%s;%s=1;break;}case 2:{var %s=1;%s;%s=2;break;}}`,
			stateInit, condition, stateVar, stateVar,
			stateVar+"_executed", thenBlock, stateVar,
			stateVar+"_executed", elseBlock, stateVar)

		return flatControl
	})

	return result
}

func (o *ObfuscatorV2) flattenForEnhanced(code string) string {
	forPattern := regexp.MustCompile(`for\s*\(([^;]+);([^;]+);([^)]+)\)\s*\{([^}]+)\}`)

	result := forPattern.ReplaceAllStringFunc(code, func(match string) string {
		groups := forPattern.FindStringSubmatch(match)
		if len(groups) < 5 {
			return match
		}

		condition := groups[2]
		update := groups[3]
		body := groups[4]

		stateVar := o.generateStateVar()
		counterVar := o.generateStateVar()

		flatFor := fmt.Sprintf(`var %s=0,%s;for(;%s;){if(!(%s))break;var %s=1;%s;%s++;%s=%s?1:0;}`,
			counterVar, stateVar, condition, condition,
			stateVar+"_active", body, counterVar, stateVar, update)

		return flatFor
	})

	return result
}

func (o *ObfuscatorV2) flattenWhileEnhanced(code string) string {
	whilePattern := regexp.MustCompile(`while\s*\(([^)]+)\)\s*\{([^}]+)\}`)

	result := whilePattern.ReplaceAllStringFunc(code, func(match string) string {
		groups := whilePattern.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}

		condition := groups[1]
		body := groups[2]

		stateVar := o.generateStateVar()

		flatWhile := fmt.Sprintf(`var %s=0;for(;%s||%s<1;){if(!(%s))break;var %s=1;%s;%s++;}`,
			stateVar, condition, stateVar, condition,
			stateVar+"_active", body, stateVar)

		return flatWhile
	})

	return result
}

func (o *ObfuscatorV2) addStateMachineWrapper(code string) string {
	stateVar := o.generateStateVar()
	dispatchVar := o.generateStateVar()

	wrapper := fmt.Sprintf(`
(function(){
	var %s=0,%s={};
	%s.dispatch=function(_0xs){return %s[_0xs]&&%s[_0xs]()};
	try{
		%s
	}catch(_0xe){console.error(_0xe);}
})();
`, stateVar, dispatchVar, dispatchVar, stateVar, dispatchVar, code)

	return wrapper
}

func (o *ObfuscatorV2) generateStateVar() string {
	o.functionCount++
	return fmt.Sprintf("_0xSF%d", o.functionCount)
}

// ==================== 代码分割混淆 ====================

func (o *ObfuscatorV2) splitCode(code string) string {
	if len(code) < 50 {
		return code
	}

	fragments := o.config.SplitFragments
	if fragments < 2 {
		fragments = 2
	}
	if fragments > 10 {
		fragments = 10
	}

	statements := o.splitIntoStatements(code)
	if len(statements) < fragments {
		fragments = len(statements)
	}

	if fragments < 2 {
		return code
	}

	fragmentSize := len(statements) / fragments
	var codeFragments []string

	for i := 0; i < fragments; i++ {
		start := i * fragmentSize
		end := start + fragmentSize
		if i == fragments-1 {
			end = len(statements)
		}

		fragment := strings.Join(statements[start:end], "")
		if fragment != "" {
			codeFragments = append(codeFragments, fragment)
		}
	}

	return o.assembleFragments(codeFragments)
}

func (o *ObfuscatorV2) splitIntoStatements(code string) []string {
	var statements []string
	var current strings.Builder
	braceCount := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(code); i++ {
		c := code[i]

		if !inString && (c == '"' || c == '\'' || c == '`') {
			inString = true
			stringChar = c
			current.WriteByte(c)
		} else if inString && c == stringChar && (i == 0 || code[i-1] != '\\') {
			inString = false
			stringChar = 0
			current.WriteByte(c)
		} else if inString {
			current.WriteByte(c)
		} else {
			if c == '{' {
				braceCount++
			} else if c == '}' {
				braceCount--
			}

			current.WriteByte(c)

			if braceCount == 0 && (c == ';' || c == '\n') {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" && stmt != ";" {
					statements = append(statements, stmt+"\n")
				}
				current.Reset()
			}
		}
	}

	remaining := strings.TrimSpace(current.String())
	if remaining != "" && remaining != ";" {
		statements = append(statements, remaining+"\n")
	}

	return statements
}

func (o *ObfuscatorV2) assembleFragments(fragments []string) string {
	if len(fragments) == 0 {
		return ""
	}

	var result strings.Builder

	loaderVar := o.generateStateVar()
	executorVar := o.generateStateVar()

	result.WriteString(fmt.Sprintf("(function(){var %s=[];", loaderVar))

	for i, fragment := range fragments {
		fragmentKey := fmt.Sprintf("f%d", i)
		fragmentVar := o.generateStateVar()

		encodedFragment := o.encodeFragment(fragment)
		fragmentToken := fmt.Sprintf("__frg_%s__('%s')", fragmentKey, encodedFragment)

		result.WriteString(fmt.Sprintf("var %s=%s;", fragmentVar, fragmentToken))
		result.WriteString(fmt.Sprintf("%s.push(%s);", loaderVar, fragmentVar))

		o.stats.FragmentsCreated++
	}

	result.WriteString(fmt.Sprintf("var %s=%s.join('');eval(%s);", executorVar, loaderVar, executorVar))
	result.WriteString("})();")

	return result.String()
}

func (o *ObfuscatorV2) encodeFragment(fragment string) string {
	key := o.config.StringEncryptionKey
	if len(key) == 0 {
		key = []byte("hjtpx-v2-fragment-key")
	}

	keyHash := sha256.Sum256(key)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return base64.StdEncoding.EncodeToString([]byte(fragment))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return base64.StdEncoding.EncodeToString([]byte(fragment))
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(fragment))
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(fragment), nil)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	var obfuscated strings.Builder
	for i, c := range encoded {
		obfuscated.WriteRune(c + rune(i%3))
	}

	return obfuscated.String()
}

// ==================== 死代码注入增强 ====================

func (o *ObfuscatorV2) injectDeadCodeEnhanced(code string) string {
	ratio := o.config.DeadCodeRatio
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	estimatedLines := strings.Count(code, "\n")
	deadCodeCount := int(float64(estimatedLines) * ratio)

	if deadCodeCount < 1 {
		deadCodeCount = 1
	}
	if deadCodeCount > 20 {
		deadCodeCount = 20
	}

	result := code

	for i := 0; i < deadCodeCount; i++ {
		deadCode := o.generateDeadCode()
		result = o.injectDeadCodeAtRandom(result, deadCode)

		o.stats.DeadCodeInjected++
	}

	return result
}

func (o *ObfuscatorV2) generateDeadCode() string {
	deadCodeTypes := []func() string{
		o.generateDeadVariableDecl,
		o.generateDeadConditional,
		o.generateDeadLoop,
		o.generateDeadFunction,
		o.generateDeadExpression,
	}

	deadCodeGenerator := deadCodeTypes[o.functionCount%len(deadCodeTypes)]
	o.functionCount++

	return deadCodeGenerator()
}

func (o *ObfuscatorV2) generateDeadVariableDecl() string {
	varName := o.generateObfuscatedNameV2()
	varValue := o.generateRandomValue()

	return fmt.Sprintf("var %s=%s;", varName, varValue)
}

func (o *ObfuscatorV2) generateDeadConditional() string {
	varName := o.generateObfuscatedNameV2()
	condition := o.generateObfuscatedNameV2()

	return fmt.Sprintf("if(%s&&%s){var %s=%s;}", varName, condition, o.generateObfuscatedNameV2(), o.generateRandomValue())
}

func (o *ObfuscatorV2) generateDeadLoop() string {
	varName := o.generateObfuscatedNameV2()

	return fmt.Sprintf("for(var %s=0;%s<%d;%s++){var %s=%s*%s;}", varName, varName, o.randomInt(10, 100), varName, o.generateObfuscatedNameV2(), varName, varName)
}

func (o *ObfuscatorV2) generateDeadFunction() string {
	funcName := o.generateObfuscatedNameV2()
	paramName := o.generateObfuscatedNameV2()

	return fmt.Sprintf("function %s(%s){var %s=%s*2;return %s;}", funcName, paramName, o.generateObfuscatedNameV2(), paramName, o.generateObfuscatedNameV2())
}

func (o *ObfuscatorV2) generateDeadExpression() string {
	varName := o.generateObfuscatedNameV2()
	a := o.generateObfuscatedNameV2()
	b := o.generateObfuscatedNameV2()

	return fmt.Sprintf("var %s=%s+%s-%s*%s/%s;", varName, a, b, a, b, o.generateObfuscatedNameV2())
}

func (o *ObfuscatorV2) injectDeadCodeAtRandom(code string, deadCode string) string {
	positions := o.findInjectionPoints(code)
	if len(positions) == 0 {
		return code + "\n" + deadCode
	}

	position := positions[o.functionCount%len(positions)]
	o.functionCount++

	prefix := code[:position]
	suffix := code[position:]

	return prefix + deadCode + "\n" + suffix
}

func (o *ObfuscatorV2) findInjectionPoints(code string) []int {
	var points []int
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "function ") ||
			strings.HasPrefix(trimmed, "var ") ||
			strings.HasPrefix(trimmed, "if ") ||
			strings.HasPrefix(trimmed, "for ") ||
			strings.HasPrefix(trimmed, "while ") {
			if i > 0 {
				points = append(points, len(strings.Join(lines[:i], "\n"))+1)
			}
		}
	}

	if len(points) == 0 {
		points = append(points, 0)
	}

	return points
}

func (o *ObfuscatorV2) generateObfuscatedNameV2() string {
	prefixes := []string{"_0x", "_0x1", "_0x2", "_0x3", "_0x4"}
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$"

	o.functionCount++
	if o.functionCount < len(prefixes) {
		return prefixes[o.functionCount]
	}

	length := 3 + o.functionCount%4
	var name strings.Builder
	name.WriteString("_0x")
	for i := 0; i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		name.WriteByte(chars[idx.Int64()])
	}

	return name.String()
}

func (o *ObfuscatorV2) generateRandomValue() string {
	types := []func() string{
		func() string {
			return fmt.Sprintf("%d", o.randomInt(0, 1000))
		},
		func() string {
			return fmt.Sprintf("'%s'", o.generateRandomString(5))
		},
		func() string {
			return fmt.Sprintf("[%d,%d,%d]", o.randomInt(0, 100), o.randomInt(0, 100), o.randomInt(0, 100))
		},
		func() string {
			return fmt.Sprintf("{a:%d,b:'%s'}", o.randomInt(0, 100), o.generateRandomString(3))
		},
	}

	return types[o.functionCount%len(types)]()
}

func (o *ObfuscatorV2) randomInt(min, max int) int {
	return min + int(time.Now().UnixNano()%int64(max-min))
}

func (o *ObfuscatorV2) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result strings.Builder
	for i := 0; i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result.WriteByte(charset[idx.Int64()])
	}
	return result.String()
}

// ==================== 控制流重排序 ====================

func (o *ObfuscatorV2) reorderControlFlow(code string) string {
	if !o.config.EnableReachableCodeShuffling {
		return code
	}

	blocks := o.extractReachableBlocks(code)
	if len(blocks) < 2 {
		return code
	}

	shuffled := o.shuffleBlocks(blocks)

	return o.reconstructCode(shuffled)
}

func (o *ObfuscatorV2) extractReachableBlocks(code string) []string {
	var blocks []string
	var current strings.Builder
	braceCount := 0
	inFunction := false

	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "function ") {
			inFunction = true
		}

		current.WriteString(line + "\n")

		for _, c := range line {
			if c == '{' {
				braceCount++
			} else if c == '}' {
				braceCount--
			}
		}

		if braceCount == 0 && inFunction && (strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "};")) {
			blocks = append(blocks, current.String())
			current.Reset()
			inFunction = false
		}
	}

	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}

	return blocks
}

func (o *ObfuscatorV2) shuffleBlocks(blocks []string) []string {
	shuffled := make([]string, len(blocks))
	copy(shuffled, blocks)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := int(time.Now().UnixNano()) % (i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func (o *ObfuscatorV2) reconstructCode(blocks []string) string {
	return strings.Join(blocks, "\n")
}

// ==================== 虚假控制流 ====================

func (o *ObfuscatorV2) injectBogusControlFlow(code string) string {
	if !o.config.EnableBogusControlFlow {
		return code
	}

	bogusCode := o.generateBogusControlFlow()

	lines := strings.Split(code, "\n")
	if len(lines) < 2 {
		return bogusCode + code
	}

	insertPoint := len(lines) / 2
	before := strings.Join(lines[:insertPoint], "\n")
	after := strings.Join(lines[insertPoint:], "\n")

	return before + "\n" + bogusCode + "\n" + after
}

func (o *ObfuscatorV2) generateBogusControlFlow() string {
	varName := o.generateObfuscatedNameV2()
	condition := o.generateObfuscatedNameV2()

	bogus := fmt.Sprintf(`
(function(){
	var %s=Math.random()>0.5;
	var %s=%s?%d:%d;
	if(%s){
		var %s=%s*%s;
	}else{
		var %s=%s-%s;
	}
})();
`, varName, condition, varName, o.randomInt(1, 100), o.randomInt(1, 100),
		condition,
		o.generateObfuscatedNameV2(), condition, varName,
		o.generateObfuscatedNameV2(), condition, varName)

	return bogus
}

// ==================== 辅助函数 ====================

func (o *ObfuscatorV2) GetStats() map[string]int {
	return map[string]int{
		"strings_encrypted":    o.stats.StringsEncrypted,
		"fragments_created":   o.stats.FragmentsCreated,
		"dead_code_injected":   o.stats.DeadCodeInjected,
		"control_flow_flattened": o.stats.ControlFlowFlattened,
	}
}

func (o *ObfuscatorV2) ResetStats() {
	o.stats = ObfuscationStats{}
}

// ==================== 便捷函数 ====================

func ObfuscateWithConfigV2(code string, config ObfuscatorV2Config) (string, error) {
	obfuscator := NewObfuscatorV2(config)
	return obfuscator.Obfuscate(code)
}

func GenerateV2ObfuscationReport(original, obfuscated string, config ObfuscatorV2Config) map[string]interface{} {
	report := map[string]interface{}{
		"original_length":   len(original),
		"obfuscated_length": len(obfuscated),
		"compression_ratio": float64(len(obfuscated)) / float64(len(original)),
		"config": map[string]interface{}{
			"advanced_string_encryption":     config.EnableAdvancedStringEncryption,
			"enhanced_control_flow_flattening": config.EnableEnhancedControlFlowFlattening,
			"code_splitting":                 config.EnableCodeSplitting,
			"enhanced_dead_code_injection":   config.EnableEnhancedDeadCodeInjection,
			"encryption_rounds":              config.EncryptionRounds,
			"split_fragments":                config.SplitFragments,
			"dead_code_ratio":                config.DeadCodeRatio,
		},
	}

	return report
}

func EstimateV2ObfuscationStrength(code string, config ObfuscatorV2Config) float64 {
	score := 0.0

	if config.EnableAdvancedStringEncryption {
		score += 30.0
		score += float64(config.EncryptionRounds) * 5.0
	}

	if config.EnableEnhancedControlFlowFlattening {
		score += 25.0
	}

	if config.EnableCodeSplitting {
		score += 15.0
		score += float64(config.SplitFragments) * 2.0
	}

	if config.EnableEnhancedDeadCodeInjection {
		score += 20.0
		score += config.DeadCodeRatio * 10.0
	}

	if config.EnableControlFlowReordering {
		score += 10.0
	}

	if config.EnableBogusControlFlow {
		score += 5.0
	}

	return math.Min(score, 100.0)
}
