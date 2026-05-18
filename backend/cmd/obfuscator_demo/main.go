package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hjtpx/hjtpx/internal/tools"
)

func main() {
	testCode := `
(function(){
	var appKey = "test-app-key-2024";
	var apiEndpoint = "https://api.example.com/v1";
	var secretKey = "super-secret-key-for-hmac";

	function encryptData(data) {
		var encrypted = "";
		for(var i = 0; i < data.length; i++) {
			encrypted += String.fromCharCode(data.charCodeAt(i) ^ secretKey.charCodeAt(i % secretKey.length));
		}
		return encrypted;
	}

	function sendRequest(url, data) {
		console.log("Sending request to:", url);
		var encryptedData = encryptData(data);
		return fetch(url, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"X-App-Key": appKey,
				"X-Signature": encryptData(secretKey)
			},
			body: JSON.stringify({ data: encryptedData })
		});
	}

	function validateToken(token) {
		if(token && token.length > 32) {
			return true;
		}
		return false;
	}

	window.CaptchaSDK = {
		initialize: function(config) {
			this.config = config;
			console.log("Initialized with app key:", appKey);
		},
		verify: function(token) {
			return validateToken(token);
		},
		request: function(data) {
			return sendRequest(apiEndpoint + "/verify", data);
		}
	};
})();
`

	fmt.Println("========================================")
	fmt.Println("JavaScript混淆器测试")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Println("原始代码:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(testCode)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	config := tools.ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:      true,
		EnableCodeCompression:        true,
		EnableControlFlowFlattening: true,
		EnableDeadCodeInjection:     true,
		EnableFunctionWrapping:      true,
		EnableAdvancedAntiDebug:     true,
		EnableMemoryProtection:      true,
		EnableCodeIntegrity:        true,
		EnableDynamicAnalysis:      true,
		EnableTimingProtection:      true,
		EnableExceptionHandling:     true,
		StringEncryptionMethod:      "aes-gcm",
		PreserveConsole:             true,
	}

	obfuscator := tools.NewObfuscator(config)
	obfuscated, err := obfuscator.Obfuscate(testCode)
	if err != nil {
		fmt.Printf("混淆失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n混淆后代码:")
	fmt.Println(strings.Repeat("-", 50))
	if len(obfuscated) > 2000 {
		fmt.Println(obfuscated[:2000])
		fmt.Println("\n... [代码被截断] ...")
		fmt.Printf("\n完整代码已保存到 obfuscated_output.js (%d bytes)\n", len(obfuscated))
	} else {
		fmt.Println(obfuscated)
	}
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	os.WriteFile("obfuscated_output.js", []byte(obfuscated), 0644)

	stats := obfuscator.GetStats()
	fmt.Println("\n混淆统计:")
	fmt.Printf("- 变量混淆: %d 个\n", stats["variables_obfuscated"])
	fmt.Printf("- 字符串加密: %d 个\n", stats["strings_encrypted"])
	fmt.Printf("- 函数包装: %d 个\n", stats["functions_wrapped"])
	fmt.Printf("- 代码压缩率: %.2f%%\n", float64(len(testCode)-len(obfuscated))/float64(len(testCode))*100)

	analyzer := tools.AnalyzeCode(testCode)
	obfAnalyzer := tools.AnalyzeCode(obfuscated)

	metrics := analyzer.GetMetrics()
	obfMetrics := obfAnalyzer.GetMetrics()

	fmt.Println("\n代码分析对比:")
	fmt.Printf("- 原始行数: %v, 混淆后: %v\n", metrics["lines_of_code"], obfMetrics["lines_of_code"])
	fmt.Printf("- 原始函数: %v, 混淆后: %v\n", metrics["functions"], obfMetrics["functions"])
	fmt.Printf("- 原始字符串: %v, 混淆后: %v\n", metrics["strings"], obfMetrics["strings"])
	fmt.Printf("- 原始变量: %v, 混淆后: %v\n", metrics["variables"], obfMetrics["variables"])

	quality := tools.EstimateObfuscationQuality(testCode, obfuscated)
	fmt.Println("\n混淆质量评估:")
	fmt.Printf("- 原始熵值: %.2f\n", quality["entropy_original"])
	fmt.Printf("- 混淆后熵值: %.2f\n", quality["entropy_obfuscated"])
	fmt.Printf("- 熵值提升: %.2f%%\n", quality["entropy_improvement_percent"])
	fmt.Printf("- 不可读性: %.2f%%\n", quality["unreadability_percent"])
	fmt.Printf("- 总体质量: %.2f/100\n", quality["overall_quality"])

	valid, message := tools.ValidateObfuscatedCode(obfuscated)
	fmt.Println("\n代码验证:")
	fmt.Printf("- 有效性: %v\n", valid)
	fmt.Printf("- 消息: %s\n", message)

	fmt.Println("\n========================================")
	fmt.Println("测试完成！")
	fmt.Println("========================================")
}
