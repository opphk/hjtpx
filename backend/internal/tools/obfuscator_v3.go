package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type ObfuscatorV3 struct {
	options         *ObfuscatorOptions
	keyGenerator    *keyGenerator
	stringRegistry  *stringRegistry
	controlFlowMgr  *controlFlowManager
	antiDebugMgr    *antiDebugManager
}

type ObfuscatorOptions struct {
	ControlFlowFlattening     bool
	StringEncryption          bool
	DeadCodeInjection         bool
	VariableNameObfuscation   bool
	FunctionReordering         bool
	AntiDebug                 bool
	StringArray               bool
	EncryptFunctionNames      bool
	EncryptNumbers            bool
	DomainLock                bool
	SelfDefending             bool
	DisableConsoleOutput      bool
	AntiTamper                bool
	DebugProtection           bool
	LiveRelocation            bool
}

type keyGenerator struct {
	mu      sync.Mutex
	keys    map[string]bool
	counter uint64
}

func newKeyGenerator() *keyGenerator {
	return &keyGenerator{
		keys:    make(map[string]bool),
		counter: 0,
	}
}

func (kg *keyGenerator) generate() string {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	kg.counter++
	buf := make([]byte, 32)
	rand.Read(buf)
	key := fmt.Sprintf("_0x%x", kg.counter)
	kg.keys[key] = true
	return hex.EncodeToString(buf)[:32]
}

type stringRegistry struct {
	mu       sync.RWMutex
	strings  []string
	indices  map[string]int
	encoded  map[int]string
}

func newStringRegistry() *stringRegistry {
	return &stringRegistry{
		strings: make([]string, 0),
		indices: make(map[string]int),
		encoded: make(map[int]string),
	}
}

func (sr *stringRegistry) add(s string) int {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if idx, exists := sr.indices[s]; exists {
		return idx
	}
	idx := len(sr.strings)
	sr.strings = append(sr.strings, s)
	sr.indices[s] = idx
	sr.encoded[idx] = sr.encodeString(s)
	return idx
}

func (sr *stringRegistry) encodeString(s string) string {
	encoded := make([]byte, len(s)*2)
	for i, c := range s {
		encoded[i*2] = byte((int(c) >> 8) & 0xFF)
		encoded[i*2+1] = byte(int(c) & 0xFF)
	}
	return base64.StdEncoding.EncodeToString(encoded)
}

func (sr *stringRegistry) getEncoded(idx int) string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.encoded[idx]
}

type controlFlowManager struct {
	mu      sync.Mutex
	blocks  map[string]*controlBlock
	counter int
}

type controlBlock struct {
	ID        string
	Body      string
	Condition string
	NextBlock string
}

func newControlFlowManager() *controlFlowManager {
	return &controlFlowManager{
		blocks:  make(map[string]*controlBlock),
		counter: 0,
	}
}

func (cfm *controlFlowManager) createBlock(body, condition string) string {
	cfm.mu.Lock()
	defer cfm.mu.Unlock()
	cfm.counter++
	id := fmt.Sprintf("_blk%d", cfm.counter)
	cfm.blocks[id] = &controlBlock{
		ID:        id,
		Body:      body,
		Condition: condition,
	}
	return id
}

type antiDebugManager struct {
	techniques []antiDebugTechnique
}

type antiDebugTechnique struct {
	Name    string
	Enabled bool
	Weight  int
}

func newAntiDebugManager() *antiDebugManager {
	return &antiDebugManager{
		techniques: []antiDebugTechnique{
			{Name: "debugger_check", Enabled: true, Weight: 10},
			{Name: "console_check", Enabled: true, Weight: 8},
			{Name: "time_check", Enabled: true, Weight: 7},
			{Name: "function_wrapping", Enabled: true, Weight: 9},
			{Name: "eval_manipulation", Enabled: true, Weight: 6},
			{Name: "devtools_detection", Enabled: true, Weight: 8},
			{Name: "performance_timing", Enabled: true, Weight: 7},
			{Name: "stack_depth_check", Enabled: true, Weight: 6},
		},
	}
}

func NewObfuscatorV3() *ObfuscatorV3 {
	return &ObfuscatorV3{
		options: &ObfuscatorOptions{
			ControlFlowFlattening:   true,
			StringEncryption:        true,
			DeadCodeInjection:       true,
			VariableNameObfuscation: true,
			FunctionReordering:      true,
			AntiDebug:               true,
			StringArray:             true,
			EncryptFunctionNames:    true,
			EncryptNumbers:          true,
			DomainLock:              true,
			SelfDefending:           true,
			DisableConsoleOutput:   true,
			AntiTamper:              true,
			DebugProtection:         true,
			LiveRelocation:          true,
		},
		keyGenerator:   newKeyGenerator(),
		stringRegistry: newStringRegistry(),
		controlFlowMgr: newControlFlowManager(),
		antiDebugMgr:   newAntiDebugManager(),
	}
}

func (o *ObfuscatorV3) Obfuscate(jsCode string) (string, error) {
	if jsCode == "" {
		return "", fmt.Errorf("empty code")
	}

	result := jsCode

	if o.options.StringArray {
		result = o.convertToStringArray(result)
	}

	if o.options.StringEncryption {
		result = o.encryptStringsEnhanced(result)
	}

	if o.options.VariableNameObfuscation {
		result = o.obfuscateVariablesAdvanced(result)
	}

	if o.options.ControlFlowFlattening {
		result = o.flattenControlFlowAdvanced(result)
	}

	if o.options.FunctionReordering {
		result = o.reorderFunctions(result)
	}

	if o.options.DeadCodeInjection {
		result = o.injectDeadCodeAdvanced(result)
	}

	if o.options.AntiDebug {
		result = o.addAntiDebugAdvanced(result)
	}

	if o.options.EncryptFunctionNames {
		result = o.encryptFunctionNames(result)
	}

	if o.options.EncryptNumbers {
		result = o.encryptNumbers(result)
	}

	if o.options.DomainLock {
		result = o.addDomainLock(result)
	}

	if o.options.SelfDefending {
		result = o.addSelfDefendingCode(result)
	}

	if o.options.DisableConsoleOutput {
		result = o.disableConsoleOutput(result)
	}

	if o.options.LiveRelocation {
		result = o.addLiveRelocation(result)
	}

	result = o.minifyAdvanced(result)

	return result, nil
}

func (o *ObfuscatorV3) convertToStringArray(code string) string {
	stringRegex := regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'`)
	matches := stringRegex.FindAllString(code, -1)

	if len(matches) == 0 {
		return code
	}

	stringList := make([]string, 0, len(matches))
	stringMap := make(map[string]int)

	for _, match := range matches {
		if _, exists := stringMap[match]; !exists {
			stringMap[match] = len(stringList)
			stringList = append(stringList, match)
		}
	}

	var encodedStrings []string
	for _, s := range stringList {
		idx := stringMap[s]
		o.stringRegistry.add(s)
		encoded := o.stringRegistry.getEncoded(idx)
		encodedStrings = append(encodedStrings, fmt.Sprintf("'%s'", encoded))
	}

	arrayName := o.keyGenerator.generate()[:8]
	arrayDef := fmt.Sprintf("var %s=[%s];", arrayName, strings.Join(encodedStrings, ","))

	result := code
	for _, match := range matches {
		if idx, exists := stringMap[match]; exists {
			replacement := fmt.Sprintf("%s[%d]", arrayName, idx)
			result = strings.Replace(result, match, replacement, 1)
		}
	}

	decoder := o.generateDecoderFunction(arrayName)
	return decoder + arrayDef + result
}

func (o *ObfuscatorV3) encryptStringsEnhanced(code string) string {
	stringRegex := regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'`)
	matches := stringRegex.FindAllStringIndex(code, -1)

	if len(matches) == 0 {
		return code
	}

	encryptionKey := o.keyGenerator.generate()
	result := code
	offset := 0

	for _, match := range matches {
		original := code[match[0]:match[1]]
		if len(original) <= 2 {
			continue
		}

		content := code[match[0]+1 : match[1]-1]
		encrypted := o.encryptStringAES(content, encryptionKey)

		funcName := fmt.Sprintf("__x%s", o.keyGenerator.generate()[:6])
		replacement := fmt.Sprintf("%s('%s')", funcName, encrypted)

		start := match[0] + offset
		end := match[1] + offset

		result = result[:start] + replacement + result[end:]
		offset += len(replacement) - len(original)
	}

	decoder := o.generateAESDecoderFunction(encryptionKey)
	return decoder + result
}

func (o *ObfuscatorV3) encryptStringAES(content, key string) string {
	keyHash := sha256.Sum256([]byte(key))
	aesKey := keyHash[:32]

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return content
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return content
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	ciphertext := gcm.Seal(nonce, nonce, []byte(content), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func (o *ObfuscatorV3) generateAESDecoderFunction(key string) string {
	keyHash := sha256.Sum256([]byte(key))
	keyHex := hex.EncodeToString(keyHash[:])
	keyVar := fmt.Sprintf("_k%s", o.keyGenerator.generate()[:4])

	return fmt.Sprintf(`
var %s='%s';
var _d=function(_s){
	var _h=[];
	for(var _i=0;_i<_s.length;_i+=2){
		_h.push(String.fromCharCode(parseInt(_s.substr(_i,2),16)));
	}
	return _h.join('');
};
var _e=function(_c){
	var _k=window['%s']||'';
	var _b=atob(_c);
	var _r=[];
	for(var _i=0;_i<_b.length;_i++){
		_r.push(String.fromCharCode(_b.charCodeAt(_i)^_k.charCodeAt(_i%%_k.length)));
	}
	return _r.join('');
};
`, keyVar, keyHex[:32], keyVar)
}

func (o *ObfuscatorV3) generateDecoderFunction(arrayName string) string {
	decoded := o.keyGenerator.generate()[:6]

	return fmt.Sprintf(`
var %s=function(_i){
	var _s=window['%s']||[];
	var _e='';
	for(var _c=0;_c<_s[_i].length;_c+=2){
		_e+=String.fromCharCode(parseInt(_s[_i].substr(_c,2),16));
	}
	return _e;
};
`, decoded, o.keyGenerator.generate()[:8])
}

func (o *ObfuscatorV3) obfuscateVariablesAdvanced(code string) string {
	varDeclRegex := regexp.MustCompile(`\b(var|let|const)\s+(\w+)\b`)
	funcNameRegex := regexp.MustCompile(`function\s+(\w+)\s*\(`)
	propRegex := regexp.MustCompile(`\.(\w+)\s*=`)

	varDecls := varDeclRegex.FindAllStringSubmatch(code, -1)
	funcNames := funcNameRegex.FindAllStringSubmatch(code, -1)
	props := propRegex.FindAllStringSubmatch(code, -1)

	replacements := make(map[string]string)

	for _, match := range varDecls {
		if len(match) >= 3 {
			original := match[2]
			if len(original) > 2 && original != "undefined" && original != "null" {
				replacements[original] = o.generateMangledName(len(replacements))
			}
		}
	}

	for _, match := range funcNames {
		if len(match) >= 2 {
			original := match[1]
			if len(original) > 2 && original != "constructor" {
				replacements[original] = o.generateMangledName(len(replacements))
			}
		}
	}

	for _, match := range props {
		if len(match) >= 2 {
			original := match[1]
			if len(original) > 3 {
				replacements[original] = o.generateMangledName(len(replacements))
			}
		}
	}

	result := code
	for original, replacement := range replacements {
		result = regexp.MustCompile(`\b`+regexp.QuoteMeta(original)+`\b`).ReplaceAllString(result, replacement)
	}

	return result
}

func (o *ObfuscatorV3) generateMangledName(index int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := ""
	n := index
	for n >= len(charset) {
		result = string(charset[n%len(charset)]) + result
		n = n/len(charset) - 1
	}
	result = string(charset[n%len(charset)]) + result
	return "_" + result
}

func (o *ObfuscatorV3) flattenControlFlowAdvanced(code string) string {
	funcRegex := regexp.MustCompile(`function\s+(\w+)\s*\(([^)]*)\)\s*\{((?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*)\}`)

	matches := funcRegex.FindAllStringSubmatchIndex(code, -1)
	if len(matches) == 0 {
		return code
	}

	result := code
	offset := 0

	for _, match := range matches {
		if len(match) < 7 {
			continue
		}

		funcName := code[match[2]:match[3]]
		funcBody := code[match[6]:match[7]]
		params := code[match[4]:match[5]]

		if len(funcBody) > 200 && len(funcBody) < 2000 {
			flattened := o.flattenFunctionBodyAdvanced(funcBody, funcName)

			stateVar := o.keyGenerator.generate()[:6]
			stateArrName := o.keyGenerator.generate()[:6]

			blockInit := o.generateStateBlocks(flattened, stateArrName)

			dispatcher := fmt.Sprintf(`
var %s=0;
var %s=[%s];
(function(){
	var _fn=arguments.callee;
	var _st=setInterval(function(){
		if(%s>=%s.length){clearInterval(_st);return;}
		var _blk=%s[%s++];
		try{if(_blk)eval(_blk);}catch(_e){clearInterval(_st);throw _e;}
	},1);
})();
`, stateVar, stateArrName, blockInit, stateVar, stateArrName, stateArrName, stateVar)

			start := match[0] + offset
			end := match[1] + offset

			originalFunc := code[match[0]:match[1]]
			newFunc := fmt.Sprintf("function %s(%s){%s}", funcName, params, dispatcher)

			result = result[:start] + newFunc + result[end:]
			offset += len(newFunc) - len(originalFunc)
		}
	}

	return result
}

func (o *ObfuscatorV3) generateStateBlocks(statements []string, arrName string) string {
	var blocks []string
	for _, stmt := range statements {
		encoded := o.encodeBlock(stmt)
		blocks = append(blocks, fmt.Sprintf("'%s'", encoded))
	}
	return strings.Join(blocks, ",")
}

func (o *ObfuscatorV3) encodeBlock(block string) string {
	encoded := make([]byte, len(block)*2)
	for i, c := range block {
		encoded[i*2] = byte((int(c) >> 8) & 0xFF)
		encoded[i*2+1] = byte(int(c) & 0xFF)
	}
	return base64.StdEncoding.EncodeToString(encoded)
}

func (o *ObfuscatorV3) flattenFunctionBodyAdvanced(body, funcName string) []string {
	statementRegex := regexp.MustCompile(`(?:[^;{}]+|\{[^}]*\})+`)
	matches := statementRegex.FindAllString(body, -1)

	var statements []string
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if len(trimmed) > 0 && trimmed != "{" && trimmed != "}" {
			statements = append(statements, trimmed)
		}
	}

	if len(statements) == 0 {
		statements = append(statements, body)
	}

	return statements
}

func (o *ObfuscatorV3) reorderFunctions(code string) string {
	funcRegex := regexp.MustCompile(`function\s+(\w+)\s*\(([^)]*)\)\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)

	matches := funcRegex.FindAllStringSubmatchIndex(code, -1)
	if len(matches) < 2 {
		return code
	}

	var funcs []struct {
		name    string
		params  string
		body    string
		start   int
		end     int
	}

	for _, match := range matches {
		if len(match) >= 8 {
			name := code[match[2]:match[3]]
			params := code[match[4]:match[5]]
			body := code[match[6]:match[7]]
			start := match[0]
			end := match[1]
			funcs = append(funcs, struct {
				name    string
				params  string
				body    string
				start   int
				end     int
			}{name, params, body, start, end})
		}
	}

	if len(funcs) < 2 {
		return code
	}

	_ = o.keyGenerator.generate()[:8]
	order := o.generateRandomOrder(len(funcs))

	var newCode string
	var declarations []string

	for _, idx := range order {
		f := funcs[idx]
		newCode += code[f.start:f.end] + ";"
	}

	for _, idx := range order {
		f := funcs[idx]
		declarations = append(declarations, fmt.Sprintf("function %s(%s){%s}", f.name, f.params, f.body))
	}

	for i := len(code) - 1; i >= 0; i-- {
		if code[i] == ';' || code[i] == '}' {
			insertPos := i + 1
			declCode := strings.Join(declarations, ";")
			newCode = code[:insertPos] + declCode + ";" + newCode
			break
		}
	}

	return newCode
}

func (o *ObfuscatorV3) generateRandomOrder(n int) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}

	for i := n - 1; i > 0; i-- {
		j := randInt(i + 1)
		order[i], order[j] = order[j], order[i]
	}

	return order
}

func (o *ObfuscatorV3) injectDeadCodeAdvanced(code string) string {
	deadCodePatterns := []string{
		`var %s=0;for(var %s=0;%s<%d;%s++){%s+=Math.random()*0.001;}`,
		`(function(){var %s=Date.now();if(%s-1>0){console.log('');}})();`,
		`try{throw new Error('%s');}catch(%s){var %s=%s.toString();}`,
		`var %s=[],%s=%s.length;for(var %s=0;%s<%s;%s++){%s.push(%s[%s]);}`,
		`(function(){var %s=window;var %s=%s.innerWidth;var %s=%s.innerHeight;})();`,
	}

	funcRegex := regexp.MustCompile(`function\s+\w+\s*\([^)]*\)\s*\{`)
	matches := funcRegex.FindAllStringIndex(code, -1)

	if len(matches) == 0 {
		return code
	}

	var deadCode string
	vars := []string{
		o.keyGenerator.generate()[:4],
		o.keyGenerator.generate()[:4],
		o.keyGenerator.generate()[:4],
		o.keyGenerator.generate()[:4],
	}

	patternIdx := randInt(len(deadCodePatterns))
	pattern := deadCodePatterns[patternIdx]

	for i := 0; i < len(vars) && i < 6; i++ {
		var arg interface{}
		switch i {
		case 0, 1, 2, 3:
			arg = vars[i]
		case 4:
			arg = 100 + randInt(900)
		case 5:
			arg = vars[0]
		}
		deadCode += fmt.Sprintf(pattern, arg) + ";"
	}

	insertionPoints := make([]int, 0)
	for _, match := range matches {
		if randFloat() > 0.5 {
			insertionPoints = append(insertionPoints, match[1])
		}
	}

	result := code
	offset := 0
	for _, point := range insertionPoints {
		pos := point + offset
		result = result[:pos] + deadCode + result[pos:]
		offset += len(deadCode)
	}

	return result
}

func randFloat() float64 {
	num, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return float64(num.Int64()) / 10000.0
}

func (o *ObfuscatorV3) addAntiDebugAdvanced(code string) string {
	antiDebug := fmt.Sprintf(`
(function(){
	var _ct=0;
	var _ct2=Date.now();
	var _fn=function(){
		var _t=Date.now();
		if(_t-_ct2<%d){
			_ct++;
			if(_ct>%d){
				document.documentElement.innerHTML='';
				window.location='about:blank';
			}
		}
		_ct2=_t;
	};
	setInterval(_fn,%d);

	var _ch=function(){
		var _d=Object.defineProperty({},"property",{get:function(){throw new Error();}});
	};
	(function(){
		var _w=window;
		var _i=0;
		setInterval(function(){
			_i++;
			if(_i>%d){
				try{_w.eval('debugger');}catch(e){}
				try{_ch();}catch(e){}
				_i=0;
			}
		},50);
	})();

	var _cs=['%s','%s','%s'];
	var _cf=function(_m){
		var _o=console[_m];
		console[_m]=function(){
			if(arguments.length>0){return;}
			_o.apply(console,arguments);
		};
	};
	_cs.forEach(_cf);

	(function(){
		var _t=true;
		var _f=function(){
			if(!_t)return;
			var _e=false;
			var _s=function(){
				_e=true;
			};
			try{_s();}catch(e){}
			var _w=function(){
				_e=false;
			};
			try{_w();}catch(e){}
			setTimeout(_f,1000);
		};
		_f();
	})();
})();
`, 50+randInt(50), 3+randInt(5), 100+randInt(100), 100+randInt(200),
		o.keyGenerator.generate()[:6],
		o.keyGenerator.generate()[:6],
		o.keyGenerator.generate()[:6])

	return code + antiDebug
}

func (o *ObfuscatorV3) encryptFunctionNames(code string) string {
	funcRegex := regexp.MustCompile(`function\s+(\w+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(code, -1)

	if len(matches) == 0 {
		return code
	}

	replacements := make(map[string]string)

	for _, match := range matches {
		if len(match) >= 2 {
			original := match[1]
			if len(original) > 2 && original != "constructor" && original != "toString" {
				replacements[original] = o.keyGenerator.generate()[:10]
			}
		}
	}

	result := code
	for original, replacement := range replacements {
		result = regexp.MustCompile(`\b`+regexp.QuoteMeta(original)+`\b`).ReplaceAllString(result, replacement)
	}

	return result
}

func (o *ObfuscatorV3) encryptNumbers(code string) string {
	numberRegex := regexp.MustCompile(`\b(\d+)\b`)
	matches := numberRegex.FindAllStringSubmatchIndex(code, -1)

	if len(matches) == 0 {
		return code
	}

	result := code
	offset := 0

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		original := code[match[2]:match[3]]
		num, err := strconv.ParseInt(original, 10, 64)
		if err != nil || num < 10 || num > 1000000 {
			continue
		}

		expr := o.generateNumberExpression(num)

		start := match[2] + offset
		end := match[3] + offset

		result = result[:start] + expr + result[end:]
		offset += len(expr) - len(original)
	}

	return result
}

func (o *ObfuscatorV3) generateNumberExpression(n int64) string {
	if n < 100 {
		nInt := int(n)
		return fmt.Sprintf("(%d*%d-%d)", nInt/2+1, 2, (nInt/2+1)*2-nInt)
	}

	base := int64(1)
	for base*base <= n {
		base++
	}
	base--

	expr := fmt.Sprintf("%d*%d", base, base)
	remainder := n - base*base

	if remainder > 0 {
		expr += fmt.Sprintf("+%d", remainder)
	} else if remainder < 0 {
		expr += fmt.Sprintf("-%d", -remainder)
	}

	return fmt.Sprintf("(%s)", expr)
}

func (o *ObfuscatorV3) addDomainLock(code string) string {
	domains := []string{
		"localhost",
		"127.0.0.1",
		"example.com",
	}

	lockCode := fmt.Sprintf(`
(function(){
	var _dl=['%s'];
	var _ch=window.location.hostname;
	var _ok=false;
	for(var _i=0;_i<_dl.length;_i++){
		if(_ch.indexOf(_dl[_i])!==-1){_ok=true;break;}
	}
	if(!_ok){
		document.body.innerHTML='';
		document.body.style.background='white';
		var _m=document.createElement('div');
		_m.style.cssText='position:fixed;top:50%%;left:50%%;transform:translate(-50%%,-50%%);font-family:sans-serif;font-size:16px;color:#333;';
		_m.textContent='Access Denied';
		document.body.appendChild(_m);
	}
})();
`, strings.Join(domains, "','"))

	return lockCode + code
}

func (o *ObfuscatorV3) addSelfDefendingCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	hashStr := hex.EncodeToString(hash[:16])

	selfDefend := fmt.Sprintf(`
(function(){
	var _h='%s';
	var _s=document.scripts||document.getElementsByTagName('script');
	var _c='';
	for(var _i=0;_i<_s.length;_i++){
		if(_s[_i].src){_c+=_s[_i].src;}
	}
	var _sh=sha256(_c||'%s');
	if(_sh!==_h){
		document.body.innerHTML='';
	}
})();
`, hashStr, o.keyGenerator.generate()[:16])

	return code + selfDefend
}

func (o *ObfuscatorV3) disableConsoleOutput(code string) string {
	disableCode := `
(function(){
	var _m=['log','warn','error','info','debug','table'];
	var _f=function(_n){
		var _o=console[_n];
		console[_n]=function(){
			if(arguments&&arguments.length===0){return;}
			_o.apply(console,arguments);
		};
	};
	_m.forEach(_f);
	Object.defineProperty(console,'__proto__',{set:function(){}});
})();
`

	return disableCode + code
}

func (o *ObfuscatorV3) addLiveRelocation(code string) string {
	relocationCode := `
(function(){
	var _r=['appendChild','insertBefore','replaceChild','removeChild'];
	var _e=Element.prototype;
	var _o={};
	_r.forEach(function(_n){
		if(_e[_n]){
			_o[_n]=_e[_n];
			_e[_n]=function(_c){
				if(_c&&_c.nodeType===1){
					try{_o[_n].call(this,_c);}catch(e){}
				}
			};
		}
	});
})();
`

	return relocationCode + code
}

func (o *ObfuscatorV3) minifyAdvanced(code string) string {
	patterns := []struct {
		from *regexp.Regexp
		to   string
	}{
		{regexp.MustCompile(`\s+`), " "},
		{regexp.MustCompile(`\s*([{};,:])\s*`), "$1"},
		{regexp.MustCompile(`\s*\(\s*`), "("},
		{regexp.MustCompile(`\s*\)\s*`), ")"},
		{regexp.MustCompile(`;\s*}`), "}"},
		{regexp.MustCompile(`{\s*`), "{"},
	}

	result := code
	for _, p := range patterns {
		result = p.from.ReplaceAllString(result, p.to)
	}

	return strings.Trim(result, " \t\n")
}

func (o *ObfuscatorV3) GetObfuscationStats() map[string]interface{} {
	return map[string]interface{}{
		"features_enabled": map[string]bool{
			"string_array":              o.options.StringArray,
			"string_encryption":         o.options.StringEncryption,
			"variable_obfuscation":      o.options.VariableNameObfuscation,
			"control_flow_flattening":   o.options.ControlFlowFlattening,
			"function_reordering":      o.options.FunctionReordering,
			"dead_code_injection":       o.options.DeadCodeInjection,
			"anti_debug":                o.options.AntiDebug,
			"function_name_encryption":  o.options.EncryptFunctionNames,
			"number_encryption":         o.options.EncryptNumbers,
			"domain_lock":               o.options.DomainLock,
			"self_defending":            o.options.SelfDefending,
			"console_disabling":         o.options.DisableConsoleOutput,
			"live_relocation":           o.options.LiveRelocation,
		},
		"version": "3.0.0",
		"security_level": "maximum",
	}
}

func generateRandomString(length int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[num.Int64()]
	}
	return string(result)
}

func randInt(max int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(num.Int64())
}
