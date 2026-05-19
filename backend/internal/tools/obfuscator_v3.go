package tools

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// ObfuscatorV3 高级JavaScript混淆器v3
type ObfuscatorV3 struct {
	options *ObfuscatorOptions
}

// ObfuscatorOptions 混淆选项
type ObfuscatorOptions struct {
	ControlFlowFlattening bool
	StringEncryption      bool
	DeadCodeInjection      bool
	VariableNameObfuscation bool
	FunctionReordering    bool
	AntiDebug            bool
	StringArray          bool
}

// NewObfuscatorV3 创建混淆器v3实例
func NewObfuscatorV3() *ObfuscatorV3 {
	return &ObfuscatorV3{
		options: &ObfuscatorOptions{
			ControlFlowFlattening: true,
			StringEncryption:      true,
			DeadCodeInjection:      true,
			VariableNameObfuscation: true,
			FunctionReordering:    true,
			AntiDebug:            true,
			StringArray:          true,
		},
	}
}

// Obfuscate 执行高级混淆
func (o *ObfuscatorV3) Obfuscate(jsCode string) (string, error) {
	result := jsCode
	
	// 1. 字符串加密
	if o.options.StringEncryption {
		result = o.encryptStrings(result)
	}
	
	// 2. 字符串数组化
	if o.options.StringArray {
		result = o.convertToStringArray(result)
	}
	
	// 3. 变量名混淆
	if o.options.VariableNameObfuscation {
		result = o.obfuscateVariables(result)
	}
	
	// 4. 控制流扁平化
	if o.options.ControlFlowFlattening {
		result = o.flattenControlFlow(result)
	}
	
	// 5. 死代码注入
	if o.options.DeadCodeInjection {
		result = o.injectDeadCode(result)
	}
	
	// 6. 反调试
	if o.options.AntiDebug {
		result = o.addAntiDebug(result)
	}
	
	// 7. 压缩空格
	result = o.minify(result)
	
	return result, nil
}

// encryptStrings 加密字符串
func (o *ObfuscatorV3) encryptStrings(code string) string {
	// 找到所有字符串字面量
	stringRegex := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)
	matches := stringRegex.FindAllStringIndex(code, -1)
	
	if len(matches) == 0 {
		return code
	}
	
	result := code
	offset := 0
	
	for _, match := range matches {
		original := code[match[0]:match[1]]
		content := code[match[0]+1:match[1]-1]
		
		// 生成随机密钥
		key := generateRandomString(16)
		
		// 加密内容
		encrypted := o.encryptString(content, key)
		
		// 替换为解密函数调用
		replacement := fmt.Sprintf("__d('%s','%s')", encrypted, key)
		
		// 应用偏移
		start := match[0] + offset
		end := match[1] + offset
		
		result = result[:start] + replacement + result[end:]
		offset += len(replacement) - len(original)
	}
	
	// 添加解密函数
	decoder := o.generateDecoderFunction()
	result = decoder + result
	
	return result
}

// convertToStringArray 转换为字符串数组
func (o *ObfuscatorV3) convertToStringArray(code string) string {
	// 找到所有字符串
	stringRegex := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"|'([^'\\]*(\\.[^'\\]*)*)'`)
	matches := stringRegex.FindAllString(code, -1)
	
	if len(matches) == 0 {
		return code
	}
	
	// 创建字符串数组
	stringList := make([]string, 0, len(matches))
	stringMap := make(map[string]int)
	
	for _, match := range matches {
		if _, exists := stringMap[match]; !exists {
			stringMap[match] = len(stringList)
			stringList = append(stringList, match)
		}
	}
	
	// 生成数组定义
	arrayDef := fmt.Sprintf("var __a=[%s];", 
		strings.Join(func() []string {
			result := make([]string, len(stringList))
			for i, s := range stringList {
				result[i] = fmt.Sprintf("'%s'", s)
			}
			return result
		}(), ","))
	
	// 替换字符串引用
	result := code
	for _, match := range matches {
		if idx, exists := stringMap[match]; exists {
			replacement := fmt.Sprintf("__a[%d]", idx)
			result = strings.Replace(result, match, replacement, 1)
		}
	}
	
	return arrayDef + result
}

// obfuscateVariables 混淆变量名
func (o *ObfuscatorV3) obfuscateVariables(code string) string {
	// 找到所有变量声明
	varRegex := regexp.MustCompile(`\b(var|let|const)\s+(\w+)\s*=`)
	
	// 生成映射表
	replacements := make(map[string]string)
	counter := 0
	
	matches := varRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			original := match[2]
			if _, exists := replacements[original]; !exists {
				replacements[original] = fmt.Sprintf("_%d", counter)
				counter++
			}
		}
	}
	
	// 应用替换
	result := code
	for original, replacement := range replacements {
		result = strings.ReplaceAll(result, original, replacement)
	}
	
	return result
}

// flattenControlFlow 扁平化控制流
func (o *ObfuscatorV3) flattenControlFlow(code string) string {
	// 找到函数定义
	funcRegex := regexp.MustCompile(`function\s+(\w+)\s*\([^)]*\)\s*\{([^}]*(?:\{[^}]*\}[^}]*)*)\}`)
	
	matches := funcRegex.FindAllStringSubmatchIndex(code, -1)
	if len(matches) == 0 {
		return code
	}
	
	result := code
	
	for _, match := range matches {
		if len(match) >= 4 {
			funcBody := code[match[4]:match[5]]
			funcName := code[match[2]:match[3]]
			
			// 简单的控制流扁平化
			flattened := o.flattenFunctionBody(funcBody)
			newFunc := fmt.Sprintf("function %s(){%s}", funcName, flattened)
			
			// 替换原函数
			start := match[0]
			end := match[1]
			result = result[:start] + newFunc + result[end:]
		}
	}
	
	return result
}

// flattenFunctionBody 扁平化函数体
func (o *ObfuscatorV3) flattenFunctionBody(body string) string {
	// 检测if-else结构
	ifRegex := regexp.MustCompile(`if\s*\([^)]+\)\s*\{([^}]+)\}\s*else\s*\{([^}]+)\}`)
	
	matches := ifRegex.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return body
	}
	
	result := body
	
	for _, match := range matches {
		if len(match) >= 4 {
			ifBranch := body[match[2]:match[3]]
			elseBranch := body[match[4]:match[5]]
			
			// 生成随机数来决定分支
			randomVar := fmt.Sprintf("_r%d", randInt(1000))
			
			// 生成扁平化代码
			flattened := fmt.Sprintf(
				"var %s=Math.random()>0.5;if(%s){%s}else{%s}",
				randomVar, randomVar, ifBranch, elseBranch,
			)
			
			// 替换原代码
			start := match[0]
			end := match[1]
			result = result[:start] + flattened + result[end:]
		}
	}
	
	return result
}

// injectDeadCode 注入死代码
func (o *ObfuscatorV3) injectDeadCode(code string) string {
	// 生成随机无用代码
	deadCode := []string{
		"var _=0;for(var __=0;__<0;__++){_+=__;}",
		"var _x=Math.random();if(_x<-1){console.log('dead');}",
		"try{throw new Error();}catch(_e){}",
		"(function(){var _=0;})();",
	}
	
	// 在函数开头插入
	funcRegex := regexp.MustCompile(`function\s+\w+\s*\([^)]*\)\s*\{`)
	
	matches := funcRegex.FindAllStringIndex(code, -1)
	if len(matches) == 0 {
		return code
	}
	
	result := code
	deadIdx := randInt(len(deadCode))
	insertionPoint := matches[0][1]
	
	result = result[:insertionPoint] + deadCode[deadIdx] + result[insertionPoint:]
	
	return result
}

// addAntiDebug 添加反调试代码
func (o *ObfuscatorV3) addAntiDebug(code string) string {
	antiDebug := `
(function(){
	var _ct=0;
	setInterval(function(){
		var _d=new Date();
		if(console.clear||console.debug){
			_ct++;
			if(_ct>3){
				window.location='about:blank';
			}
		}
	},1000);
	
	(function(){
		var _w=window;
		var _ct=0;
		setInterval(function(){
			_ct++;
			if(_ct>100){
				_w.close();
			}
		},100);
	})();
})();
`
	
	// 在代码末尾添加
	return code + antiDebug
}

// minify 压缩代码
func (o *ObfuscatorV3) minify(code string) string {
	// 移除多余空格和换行
	code = regexp.MustCompile(`\s+`).ReplaceAllString(code, " ")
	code = regexp.MustCompile(`\s*([{};,:])\s*`).ReplaceAllString(code, "$1")
	return code
}

// generateDecoderFunction 生成解码函数
func (o *ObfuscatorV3) generateDecoderFunction() string {
	return `
var _k='ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
function __d(s,k){
	var _b=[];
	for(var _i=0;_i<s.length;_i+=2){
		_b.push(parseInt(s.substr(_i,2),36));
	}
	var _r='';
	for(var _i=0;_i<_b.length;_i++){
		_r+=String.fromCharCode(_b[_i]^k.charCodeAt(_i%k.length));
	}
	return _r;
}
`
}

// encryptString 加密字符串
func (o *ObfuscatorV3) encryptString(content, key string) string {
	// 简单的XOR加密
	result := make([]byte, len(content))
	for i, c := range content {
		result[i] = byte(c) ^ byte(key[i%len(key)])
	}
	return base64.StdEncoding.EncodeToString(result)
}

// 辅助函数
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
