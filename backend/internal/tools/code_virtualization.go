package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

type CodeVirtualizer struct {
	vmInstructions map[string]*VMInstruction
	vmContext      *VMContext
	key            []byte
}

type VMInstruction struct {
	Opcode    byte
	Name      string
	Operands  []string
	Execute   func(*VMContext, []interface{}) interface{}
}

type VMContext struct {
	Registers [16]int64
	Stack     []interface{}
	Heap      map[int64][]byte
	IP        int
	Variables map[string]interface{}
	Functions map[string]*VMFunction
	CallStack []int
}

type VMFunction struct {
	Name      string
	Params    []string
	LocalVars []string
	Code      []byte
}

type VirtualizedCode struct {
	Instructions []byte
	Constants   []interface{}
	Functions   map[string]*VMFunction
	Metadata    *CodeMetadata
}

type CodeMetadata struct {
	OriginalSize   int
	VirtualizedAt  time.Time
	Version        string
	ProtectionLevel int
	ObfuscationSeed int64
}

func NewCodeVirtualizer(key ...[]byte) *CodeVirtualizer {
	cv := &CodeVirtualizer{
		vmInstructions: make(map[string]*VMInstruction),
		vmContext:      &VMContext{
			Stack:      make([]interface{}, 0),
			Heap:      make(map[int64][]byte),
			Variables: make(map[string]interface{}),
			Functions: make(map[string]*VMFunction),
		},
	}

	if len(key) > 0 && len(key[0]) > 0 {
		cv.key = key[0]
	} else {
		cv.key = []byte("vm-key-2024-hjtpx")
	}

	cv.initInstructions()
	return cv
}

func (cv *CodeVirtualizer) initInstructions() {
	cv.vmInstructions["NOP"] = &VMInstruction{
		Opcode:   0x00,
		Name:     "NOP",
		Operands: []string{},
		Execute:  func(ctx *VMContext, ops []interface{}) interface{} { return nil },
	}

	cv.vmInstructions["LOAD_CONST"] = &VMInstruction{
		Opcode:   0x01,
		Name:     "LOAD_CONST",
		Operands: []string{"index"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 {
				return ops[0]
			}
			return nil
		},
	}

	cv.vmInstructions["STORE_VAR"] = &VMInstruction{
		Opcode:   0x02,
		Name:     "STORE_VAR",
		Operands: []string{"name"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 && len(ops) > 1 {
				name := ops[0].(string)
				ctx.Variables[name] = ops[1]
			}
			return nil
		},
	}

	cv.vmInstructions["LOAD_VAR"] = &VMInstruction{
		Opcode:   0x03,
		Name:     "LOAD_VAR",
		Operands: []string{"name"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 {
				name := ops[0].(string)
				return ctx.Variables[name]
			}
			return nil
		},
	}

	cv.vmInstructions["ADD"] = &VMInstruction{
		Opcode:   0x04,
		Name:     "ADD",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return add(a, b)
			}
			return nil
		},
	}

	cv.vmInstructions["SUB"] = &VMInstruction{
		Opcode:   0x05,
		Name:     "SUB",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return sub(a, b)
			}
			return nil
		},
	}

	cv.vmInstructions["MUL"] = &VMInstruction{
		Opcode:   0x06,
		Name:     "MUL",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return mul(a, b)
			}
			return nil
		},
	}

	cv.vmInstructions["DIV"] = &VMInstruction{
		Opcode:   0x07,
		Name:     "DIV",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return div(a, b)
			}
			return nil
		},
	}

	cv.vmInstructions["JUMP"] = &VMInstruction{
		Opcode:   0x08,
		Name:     "JUMP",
		Operands: []string{"target"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 {
				ctx.IP = ops[0].(int)
			}
			return nil
		},
	}

	cv.vmInstructions["JUMP_IF_TRUE"] = &VMInstruction{
		Opcode:   0x09,
		Name:     "JUMP_IF_TRUE",
		Operands: []string{"target"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) > 0 && len(ops) > 0 {
				cond := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				if cond == true {
					ctx.IP = ops[0].(int)
				}
			}
			return nil
		},
	}

	cv.vmInstructions["JUMP_IF_FALSE"] = &VMInstruction{
		Opcode:   0x0A,
		Name:     "JUMP_IF_FALSE",
		Operands: []string{"target"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) > 0 && len(ops) > 0 {
				cond := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				if cond == false {
					ctx.IP = ops[0].(int)
				}
			}
			return nil
		},
	}

	cv.vmInstructions["CALL"] = &VMInstruction{
		Opcode:   0x0B,
		Name:     "CALL",
		Operands: []string{"func_name"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 {
				name := ops[0].(string)
				if fn, exists := ctx.Functions[name]; exists {
					ctx.CallStack = append(ctx.CallStack, ctx.IP)
					ctx.IP = int(binary.BigEndian.Uint64(fn.Code))
				}
			}
			return nil
		},
	}

	cv.vmInstructions["RETURN"] = &VMInstruction{
		Opcode:   0x0C,
		Name:     "RETURN",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.CallStack) > 0 {
				ctx.IP = ctx.CallStack[len(ctx.CallStack)-1]
				ctx.CallStack = ctx.CallStack[:len(ctx.CallStack)-1]
			}
			return nil
		},
	}

	cv.vmInstructions["PUSH"] = &VMInstruction{
		Opcode:   0x0D,
		Name:     "PUSH",
		Operands: []string{"value"},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ops) > 0 {
				ctx.Stack = append(ctx.Stack, ops[0])
			}
			return nil
		},
	}

	cv.vmInstructions["POP"] = &VMInstruction{
		Opcode:   0x0E,
		Name:     "POP",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) > 0 {
				val := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				return val
			}
			return nil
		},
	}

	cv.vmInstructions["AND"] = &VMInstruction{
		Opcode:   0x10,
		Name:     "AND",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return and(a, b)
			}
			return false
		},
	}

	cv.vmInstructions["OR"] = &VMInstruction{
		Opcode:   0x11,
		Name:     "OR",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return or(a, b)
			}
			return false
		},
	}

	cv.vmInstructions["NOT"] = &VMInstruction{
		Opcode:   0x12,
		Name:     "NOT",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) > 0 {
				val := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				return not(val)
			}
			return true
		},
	}

	cv.vmInstructions["XOR"] = &VMInstruction{
		Opcode:   0x13,
		Name:     "XOR",
		Operands: []string{},
		Execute: func(ctx *VMContext, ops []interface{}) interface{} {
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				return xor(a, b)
			}
			return 0
		},
	}

	cv.vmInstructions["HALT"] = &VMInstruction{
		Opcode:   0xFF,
		Name:     "HALT",
		Operands: []string{},
		Execute:  func(ctx *VMContext, ops []interface{}) interface{} { return nil },
	}
}

func add(a, b interface{}) interface{} {
	switch a.(type) {
	case int:
		return a.(int) + toInt(b)
	case int64:
		return a.(int64) + toInt64(b)
	case float64:
		return a.(float64) + toFloat64(b)
	case string:
		return a.(string) + toString(b)
	default:
		return 0
	}
}

func sub(a, b interface{}) interface{} {
	switch a.(type) {
	case int:
		return a.(int) - toInt(b)
	case int64:
		return a.(int64) - toInt64(b)
	case float64:
		return a.(float64) - toFloat64(b)
	default:
		return 0
	}
}

func mul(a, b interface{}) interface{} {
	switch a.(type) {
	case int:
		return a.(int) * toInt(b)
	case int64:
		return a.(int64) * toInt64(b)
	case float64:
		return a.(float64) * toFloat64(b)
	default:
		return 0
	}
}

func div(a, b interface{}) interface{} {
	switch a.(type) {
	case int:
		return a.(int) / toInt(b)
	case int64:
		return a.(int64) / toInt64(b)
	case float64:
		return a.(float64) / toFloat64(b)
	default:
		return 0
	}
}

func and(a, b interface{}) bool {
	return toBool(a) && toBool(b)
}

func or(a, b interface{}) bool {
	return toBool(a) || toBool(b)
}

func not(a interface{}) bool {
	return !toBool(a)
}

func xor(a, b interface{}) interface{} {
	return toInt64(a) ^ toInt64(b)
}

func toInt(v interface{}) int {
	switch v.(type) {
	case int:
		return v.(int)
	case int64:
		return int(v.(int64))
	case float64:
		return int(v.(float64))
	default:
		return 0
	}
}

func toInt64(v interface{}) int64 {
	switch v.(type) {
	case int:
		return int64(v.(int))
	case int64:
		return v.(int64)
	case float64:
		return int64(v.(float64))
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch v.(type) {
	case int:
		return float64(v.(int))
	case int64:
		return float64(v.(int64))
	case float64:
		return v.(float64)
	default:
		return 0.0
	}
}

func toString(v interface{}) string {
	switch v.(type) {
	case string:
		return v.(string)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

func toBool(v interface{}) bool {
	switch v.(type) {
	case bool:
		return v.(bool)
	case int, int64:
		return toInt64(v) != 0
	case float64:
		return v.(float64) != 0
	case string:
		return v.(string) != ""
	default:
		return false
	}
}

func (cv *CodeVirtualizer) Virtualize(code string) (*VirtualizedCode, error) {
	instructions := []byte{}
	constants := []interface{}{}

	variableRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	variables := make(map[string]bool)
	matches := variableRegex.FindAllString(code, -1)
	for _, m := range matches {
		if !isKeyword(m) {
			variables[m] = true
		}
	}

	for varName := range variables {
		instructions = append(instructions, 0x03)
		constantIndex := len(constants)
		constants = append(constants, varName)
		instructions = append(instructions, byte(constantIndex))
	}

	encryptedInstructions := make([]byte, len(instructions))
	for i, b := range instructions {
		encryptedInstructions[i] = b ^ cv.key[i%len(cv.key)]
	}

	return &VirtualizedCode{
		Instructions: encryptedInstructions,
		Constants:    constants,
		Functions:    make(map[string]*VMFunction),
		Metadata: &CodeMetadata{
			OriginalSize:    len(code),
			VirtualizedAt:    time.Now(),
			Version:         "1.0",
			ProtectionLevel: 3,
			ObfuscationSeed: time.Now().UnixNano(),
		},
	}, nil
}

func (cv *CodeVirtualizer) Execute(vcode *VirtualizedCode) error {
	ctx := cv.vmContext
	ctx.Reset()

	instructions := make([]byte, len(vcode.Instructions))
	for i, b := range vcode.Instructions {
		instructions[i] = b ^ cv.key[i%len(cv.key)]
	}

	ctx.IP = 0
	for ctx.IP < len(instructions) {
		opcode := instructions[ctx.IP]
		ctx.IP++

		var result interface{}
		switch opcode {
		case 0x00:
			result = nil
		case 0x01:
			constantIndex := int(instructions[ctx.IP])
			ctx.IP++
			if constantIndex < len(vcode.Constants) {
				result = vcode.Constants[constantIndex]
				ctx.Stack = append(ctx.Stack, result)
			}
		case 0x02:
			if len(ctx.Stack) > 0 {
				name := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				if len(ctx.Stack) > 0 {
					value := ctx.Stack[len(ctx.Stack)-1]
					ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
					if nameStr, ok := name.(string); ok {
						ctx.Variables[nameStr] = value
					}
				}
			}
		case 0x03:
			if len(ctx.Stack) > 0 {
				name := ctx.Stack[len(ctx.Stack)-1]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
				if nameStr, ok := name.(string); ok {
					result = ctx.Variables[nameStr]
					ctx.Stack = append(ctx.Stack, result)
				}
			}
		case 0x04:
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				result = add(a, b)
				ctx.Stack = append(ctx.Stack, result)
			}
		case 0x05:
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				result = sub(a, b)
				ctx.Stack = append(ctx.Stack, result)
			}
		case 0x06:
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				result = mul(a, b)
				ctx.Stack = append(ctx.Stack, result)
			}
		case 0x07:
			if len(ctx.Stack) >= 2 {
				b := ctx.Stack[len(ctx.Stack)-1]
				a := ctx.Stack[len(ctx.Stack)-2]
				ctx.Stack = ctx.Stack[:len(ctx.Stack)-2]
				result = div(a, b)
				ctx.Stack = append(ctx.Stack, result)
			}
		case 0xFF:
			return nil
		default:
			return fmt.Errorf("unknown opcode: %x", opcode)
		}
	}

	return nil
}

func (vm *VMContext) Reset() {
	vm.Registers = [16]int64{}
	vm.Stack = make([]interface{}, 0)
	vm.IP = 0
}

func isKeyword(s string) bool {
	keywords := []string{"if", "else", "for", "while", "return", "function", "var", "let", "const",
		"true", "false", "null", "undefined", "break", "continue", "switch", "case"}
	for _, kw := range keywords {
		if s == kw {
			return true
		}
	}
	return false
}

func (cv *CodeVirtualizer) GenerateObfuscatedCode(code string, level int) (string, error) {
	var obfuscated strings.Builder

	obfuscated.WriteString("(function(){")

	if level >= 1 {
		obfuscated.WriteString(cv.obfuscateStrings(code))
	}

	if level >= 2 {
		obfuscated.WriteString(cv.addDeadCode())
		obfuscated.WriteString(cv.addFakeLogic())
	}

	if level >= 3 {
		obfuscated.WriteString(cv.addAntiDebug())
		obfuscated.WriteString(cv.addSelfModification())
	}

	obfuscated.WriteString(code)
	obfuscated.WriteString("})();")

	return obfuscated.String(), nil
}

func (cv *CodeVirtualizer) obfuscateStrings(code string) string {
	stringRegex := regexp.MustCompile(`"[^"]*"|'[^']*'`)
	matches := stringRegex.FindAllString(code, -1)

	result := code
	for _, match := range matches {
		if len(match) > 2 {
			inner := match[1 : len(match)-1]
			encoded := base64.StdEncoding.EncodeToString([]byte(inner))
			result = strings.Replace(result, match, "atob('"+encoded+"')", 1)
		}
	}

	return result
}

func (cv *CodeVirtualizer) addDeadCode() string {
	return `
(function(){
var _0x1234=Math.random()>0.5?function(){return true;}:function(){return false;};
var _0xabcd=_0x1234();
if(_0xabcd){console.log('');}
})();
`
}

func (cv *CodeVirtualizer) addFakeLogic() string {
	return `
(function(){
var _0xfake1=function(){return false;};
var _0xfake2=function(){return true;};
if(typeof window==='undefined'){_0xfake1();}
})();
`
}

func (cv *CodeVirtualizer) addAntiDebug() string {
	return `
!function(){
var _0xT=window.outerWidth-window.innerWidth,o=window.outerHeight-window.innerHeight;
if(t>0||o>0){document.documentElement.style.display='none';}
setInterval(function(){try{var _0xC=console.log.toString().length;if(_0xC>16){document.documentElement.style.display='none';}}catch(e){}},1000);
}();
`
}

func (cv *CodeVirtualizer) addSelfModification() string {
	return `
(function(){
var _0xcode=document.currentScript?document.currentScript.textContent:'';
if(_0xcode.indexOf('debugger')>-1){
document.documentElement.style.display='none';
}
})();
`
}

func (cv *CodeVirtualizer) EncryptVMCode(vcode *VirtualizedCode) ([]byte, error) {
	jsonData, err := cv.serializeCode(vcode)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cv.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)
	return ciphertext, nil
}

func (cv *CodeVirtualizer) DecryptVMCode(encrypted []byte) (*VirtualizedCode, error) {
	block, err := aes.NewCipher(cv.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return cv.deserializeCode(plaintext)
}

func (cv *CodeVirtualizer) serializeCode(vcode *VirtualizedCode) ([]byte, error) {
	data := fmt.Sprintf("Instructions:%x|Constants:%v|Metadata:%v",
		vcode.Instructions, vcode.Constants, vcode.Metadata)
	return []byte(data), nil
}

func (cv *CodeVirtualizer) deserializeCode(data []byte) (*VirtualizedCode, error) {
	return &VirtualizedCode{
		Instructions: data,
		Constants:    []interface{}{},
		Metadata: &CodeMetadata{
			OriginalSize: len(data),
			VirtualizedAt: time.Now(),
			Version: "1.0",
		},
	}, nil
}

type ProtectionLevel int

const (
	LevelBasic ProtectionLevel = iota
	LevelStandard
	LevelEnhanced
	LevelMaximum
)

func (cv *CodeVirtualizer) ProtectWithLevel(code string, level ProtectionLevel) (string, error) {
	switch level {
	case LevelBasic:
		return cv.obfuscateBasic(code)
	case LevelStandard:
		return cv.obfuscateStandard(code)
	case LevelEnhanced:
		return cv.obfuscateEnhanced(code)
	case LevelMaximum:
		return cv.obfuscateMaximum(code)
	default:
		return cv.obfuscateBasic(code)
	}
}

func (cv *CodeVirtualizer) obfuscateBasic(code string) (string, error) {
	return cv.obfuscateStrings(code), nil
}

func (cv *CodeVirtualizer) obfuscateStandard(code string) (string, error) {
	obfuscated := cv.obfuscateStrings(code)
	obfuscated += cv.addDeadCode()
	return obfuscated, nil
}

func (cv *CodeVirtualizer) obfuscateEnhanced(code string) (string, error) {
	return cv.GenerateObfuscatedCode(code, 2)
}

func (cv *CodeVirtualizer) obfuscateMaximum(code string) (string, error) {
	return cv.GenerateObfuscatedCode(code, 3)
}

func (cv *CodeVirtualizer) VirtualizeAdvanced(code string, complexity int) (*VirtualizedCode, error) {
	instructions := []byte{}

	obfuscatedCode := cv.generateAdvancedObfuscation(code, complexity)

	constantPool := make([]interface{}, 0)
	for _, char := range obfuscatedCode {
		constantPool = append(constantPool, int(char))
	}

	encryptedInstructions := make([]byte, len(instructions))
	for i, b := range instructions {
		encryptedInstructions[i] = b ^ cv.key[i%len(cv.key)]
	}

	metadata := &CodeMetadata{
		OriginalSize:    len(code),
		VirtualizedAt:    time.Now(),
		Version:         "2.0",
		ProtectionLevel: complexity,
		ObfuscationSeed: time.Now().UnixNano(),
	}

	return &VirtualizedCode{
		Instructions: encryptedInstructions,
		Constants:    constantPool,
		Functions:    make(map[string]*VMFunction),
		Metadata:    metadata,
	}, nil
}

func (cv *CodeVirtualizer) generateAdvancedObfuscation(code string, complexity int) string {
	var result strings.Builder

	for i := 0; i < len(code); i++ {
		char := code[i]
		obfuscated := int(char) ^ int(cv.key[i%len(cv.key)])
		result.WriteString(fmt.Sprintf("\\x%02x", obfuscated))
	}

	if complexity >= 2 {
		result.WriteString(cv.addPolymorphicWrapper())
	}

	if complexity >= 3 {
		result.WriteString(cv.addSelfChecking())
	}

	return result.String()
}

func (cv *CodeVirtualizer) addPolymorphicWrapper() string {
	return `
(function(_0xK){
var _0xR=[];
for(var _0xI=0;_0xI<_0xK.length;_0xI++){
_0xR.push(String.fromCharCode(_0xK.charCodeAt(_0xI)^(_0xI%255)));
}
return _0xR.join('');
})
`
}

func (cv *CodeVirtualizer) addSelfChecking() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf(`
(function(){
var _0xT=%d;
var _0xC=Date.now();
if(Math.abs(_0xC-_0xT)>3600000){
document.documentElement.style.display='none';
}
})();
`, timestamp)
}

func (cv *CodeVirtualizer) GenerateRandomKey(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func (cv *CodeVirtualizer) HashCode(s string) int64 {
	h := int64(0)
	for i := 0; i < len(s); i++ {
		h = 31*h + int64(s[i])
	}
	return h
}

func (cv *CodeVirtualizer) EncryptCode(code string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(code), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cv *CodeVirtualizer) DecryptCode(encrypted string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (cv *CodeVirtualizer) GenerateVMCode(code string) (string, error) {
	vm := NewCodeVirtualizer()
	vmCode, err := vm.Virtualize(code)
	if err != nil {
		return "", err
	}

	encrypted, err := vm.EncryptVMCode(vmCode)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)

	jsCode := fmt.Sprintf(`
(function(){
var _0xE="%s";
var _0xK="%s";
var _0xD=atob(_0xE);
var _0xVM=new Function(_0xD);
return _0xVM();
})();
`, encoded, string(cv.key))

	return jsCode, nil
}

func (cv *CodeVirtualizer) ProtectWithAdvancedObfuscation(code string, level int) (string, error) {
	var result string
	var err error

	switch level {
	case 1:
		result, err = cv.obfuscateBasic(code)
	case 2:
		result, err = cv.obfuscateStandard(code)
	case 3:
		result, err = cv.obfuscateEnhanced(code)
	case 4:
		result, err = cv.obfuscateMaximum(code)
	default:
		result, err = cv.obfuscateBasic(code)
	}

	if err != nil {
		return "", err
	}

	if level >= 2 {
		result = cv.addRuntimeChecks(result)
	}

	if level >= 3 {
		result = cv.addDynamicDecryption(result)
	}

	return result, nil
}

func (cv *CodeVirtualizer) addRuntimeChecks(code string) string {
	checksum := cv.HashCode(code)
	checks := fmt.Sprintf(`
(function(){
var _0xH=%d;
var _0xC=0;
for(var _0xI=0;_0xI<document.scripts.length;_0xI++){
var _0xS=document.scripts[_0xI];
if(_0xS.src&&_0xS.src.indexOf('hjtpx')>-1){
_0xC++;
}
}
if(_0xC===0){
document.documentElement.style.display='none';
}
})();
`, checksum)

	return checks + code
}

func (cv *CodeVirtualizer) addDynamicDecryption(code string) string {
	encrypted, err := cv.EncryptCode(code, cv.key)
	if err != nil {
		encrypted = code
	}
	return fmt.Sprintf(`
(function(){
var _0xE="%s";
var _0xK="%s";
try{
var _0xD=atob(_0xE);
var _0xVM=new Function(_0xD);
_0xVM();
}catch(_0xErr){
document.documentElement.style.display='none';
}
})();
`, encrypted, string(cv.key))
}
