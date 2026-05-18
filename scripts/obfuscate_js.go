package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/backend/internal/tools"
)

type BuildConfig struct {
	InputFile    string
	OutputFile   string
	Level        int
	EnableAll    bool
	Verbose      bool
	ShowStats    bool
	GenerateReport bool
	Validate     bool
}

type ObfuscationReport struct {
	Version       string    `json:"version"`
	BuildTime     string    `json:"build_time"`
	InputFile     string    `json:"input_file"`
	OutputFile    string    `json:"output_file"`
	Level         int       `json:"level"`
	OriginalSize  int       `json:"original_size"`
	ObfuscatedSize int     `json:"obfuscated_size"`
	Compression   float64   `json:"compression_percent"`
	Features      []string  `json:"features_enabled"`
	Checksum      string    `json:"sha256_checksum"`
	Stats        map[string]int `json:"stats"`
}

func main() {
	config := BuildConfig{}

	flag.StringVar(&config.InputFile, "input", "", "输入JavaScript文件")
	flag.StringVar(&config.OutputFile, "output", "", "输出JavaScript文件")
	flag.IntVar(&config.Level, "level", 2, "混淆级别 (1-3)")
	flag.BoolVar(&config.EnableAll, "enable-all", false, "启用所有保护特性")
	flag.BoolVar(&config.Verbose, "verbose", false, "详细输出")
	flag.BoolVar(&config.ShowStats, "stats", true, "显示统计信息")
	flag.BoolVar(&config.GenerateReport, "report", false, "生成报告")
	flag.BoolVar(&config.Validate, "validate", true, "验证输出")
	flag.Parse()

	if config.InputFile == "" {
		fmt.Println("错误: 必须指定输入文件 (-input)")
		flag.Usage()
		os.Exit(1)
	}

	if config.Verbose {
		fmt.Printf("=== JavaScript 混淆构建工具 ===\n")
		fmt.Printf("版本: 3.0\n")
		fmt.Printf("时间: %s\n\n", time.Now().Format(time.RFC3339))
	}

	// 读取输入文件
	if config.Verbose {
		fmt.Printf("[INFO] 读取输入文件: %s\n", config.InputFile)
	}

	code, err := os.ReadFile(config.InputFile)
	if err != nil {
		fmt.Printf("[ERROR] 读取文件失败: %v\n", err)
		os.Exit(1)
	}

	originalSize := len(code)

	if config.Verbose {
		fmt.Printf("[INFO] 原始文件大小: %d bytes\n", originalSize)
	}

	// 配置混淆器
	if config.OutputFile == "" {
		ext := filepath.Ext(config.InputFile)
		base := strings.TrimSuffix(config.InputFile, ext)
		config.OutputFile = base + ".obfuscated" + ext
	}

	// 选择混淆器类型
	var obfuscated string

	if config.Level >= 3 || config.EnableAll {
		// 使用增强版混淆器
		if config.Verbose {
			fmt.Printf("[INFO] 使用增强版混淆器\n")
		}

		enhancedConfig := tools.EnhancedObfuscatorConfig{
			EnableVariableObfuscation:    true,
			EnableStringEncryption:        true,
			EnableCodeCompression:        true,
			EnableControlFlowFlattening: config.Level >= 2,
			EnableDeadCodeInjection:     config.Level >= 3 || config.EnableAll,
			EnableFunctionWrapping:      true,
			CompressWhitespace:          true,
			RemoveComments:              true,
			PreserveConsole:            true,
			StringEncryptionMethod:      "aes-gcm",
			EnableNameMangling:          true,
			EnableAdvancedAntiDebug:     config.EnableAll,
			EnableSelfDestruct:         config.EnableAll,
			EnableMemoryProtection:     config.EnableAll || config.Level >= 2,
			EnableCodeIntegrity:        true,
			EnableSHA256Checksum:       true,
			EnableDynamicAnalysis:      config.EnableAll,
			EnableTimingProtection:     config.EnableAll || config.Level >= 2,
			EnableAdvancedIntegrity:    config.EnableAll,
			EnableStackTraceObfuscation: config.Level >= 2,
			EnableExceptionHandling:     true,
			EnableConsoleOverride:      config.Level >= 2,
			EnhancedEncryptionLevel:   3,
			EnablePerformanceOptimization: true,
		}

		obfuscator := tools.NewEnhancedObfuscator(enhancedConfig)
		obfuscated, err = obfuscator.Obfuscate(string(code))

		if config.ShowStats {
			stats := obfuscator.GetStats()
			if stats != nil {
				fmt.Printf("\n[STATS] 混淆统计:\n")
				fmt.Printf("  - 变量混淆: %d 个\n", stats.VariablesObfuscated)
				fmt.Printf("  - 字符串加密: %d 个\n", stats.StringsEncrypted)
				fmt.Printf("  - 函数包装: %d 个\n", stats.FunctionsWrapped)
				fmt.Printf("  - 死代码块: %d 个\n", stats.DeadCodeBlocks)
				fmt.Printf("  - 控制流平坦化: %d 处\n", stats.ControlFlowFlattened)
			}
		}
	} else {
		// 使用标准混淆器
		if config.Verbose {
			fmt.Printf("[INFO] 使用标准混淆器\n")
		}

		obfuscatorConfig := tools.ObfuscatorConfig{
			EnableVariableObfuscation:    true,
			EnableStringEncryption:        true,
			EnableCodeCompression:        true,
			EnableControlFlowFlattening: config.Level >= 2,
			EnableDeadCodeInjection:     config.Level >= 3,
			EnableFunctionWrapping:      true,
			CompressWhitespace:          true,
			RemoveComments:              true,
			PreserveConsole:            true,
			StringEncryptionMethod:      "aes-gcm",
			EnableNameMangling:          true,
		}

		obfuscator := tools.NewObfuscator(obfuscatorConfig)
		obfuscated, err = obfuscator.Obfuscate(string(code))

		if config.ShowStats {
			stats := obfuscator.GetStats()
			fmt.Printf("\n[STATS] 混淆统计:\n")
			fmt.Printf("  - 变量混淆: %d 个\n", stats["variables_obfuscated"])
			fmt.Printf("  - 字符串加密: %d 个\n", stats["strings_encrypted"])
			fmt.Printf("  - 函数包装: %d 个\n", stats["functions_wrapped"])
		}
	}

	if err != nil {
		fmt.Printf("[ERROR] 混淆失败: %v\n", err)
		os.Exit(1)
	}

	// 计算SHA-256校验和
	checksum := sha256.Sum256([]byte(obfuscated))
	checksumStr := hex.EncodeToString(checksum[:])

	// 验证输出
	if config.Validate {
		if config.Verbose {
			fmt.Printf("\n[INFO] 验证混淆后的代码...\n")
		}

		valid, msg := validateObfuscatedCode(obfuscated)
		if !valid {
			fmt.Printf("[WARNING] 代码验证: %s\n", msg)
		} else if config.Verbose {
			fmt.Printf("[INFO] 代码验证通过\n")
		}
	}

	// 写入输出文件
	if config.Verbose {
		fmt.Printf("\n[INFO] 写入输出文件: %s\n", config.OutputFile)
	}

	err = os.WriteFile(config.OutputFile, []byte(obfuscated), 0644)
	if err != nil {
		fmt.Printf("[ERROR] 写入文件失败: %v\n", err)
		os.Exit(1)
	}

	// 显示结果
	fmt.Printf("\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("混淆完成!\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("输入文件: %s\n", config.InputFile)
	fmt.Printf("输出文件: %s\n", config.OutputFile)
	fmt.Printf("原始大小: %d bytes\n", originalSize)
	fmt.Printf("混淆后: %d bytes\n", len(obfuscated))
	fmt.Printf("压缩率: %.2f%%\n", float64(originalSize-len(obfuscated))/float64(originalSize)*100)
	fmt.Printf("SHA-256: %s\n", checksumStr)
	fmt.Printf("===========================================\n")

	// 生成报告
	if config.GenerateReport {
		report := generateReport(config, originalSize, len(obfuscated), checksumStr)
		reportFile := config.OutputFile + ".report.json"

		reportJSON, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			os.WriteFile(reportFile, reportJSON, 0644)
			fmt.Printf("\n[INFO] 报告已生成: %s\n", reportFile)
		}
	}

	if config.Verbose {
		fmt.Printf("\n[SUCCESS] 构建完成!\n")
	}
}

func validateObfuscatedCode(code string) (bool, string) {
	// 检查TODO/FIXME
	if strings.Contains(code, "TODO") || strings.Contains(code, "FIXME") {
		return false, "代码包含TODO或FIXME"
	}

	// 检查括号平衡
	openBraces := strings.Count(code, "{")
	closeBraces := strings.Count(code, "}")
	if openBraces != closeBraces {
		return false, fmt.Sprintf("大括号不平衡: %d vs %d", openBraces, closeBraces)
	}

	openParens := strings.Count(code, "(")
	closeParens := strings.Count(code, ")")
	if openParens != closeParens {
		return false, fmt.Sprintf("小括号不平衡: %d vs %d", openParens, closeParens)
	}

	// 检查未加密的敏感字符串
	sensitivePatterns := []string{
		`password\s*=\s*["'][^"']+["']`,
		`api[_-]?key\s*=\s*["'][^"']+["']`,
		`secret\s*=\s*["'][^"']+["']`,
	}

	for _, pattern := range sensitivePatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(code) {
			return false, fmt.Sprintf("发现未加密的敏感数据: %s", pattern)
		}
	}

	return true, "验证通过"
}

func generateReport(config BuildConfig, originalSize, obfuscatedSize int, checksum string) ObfuscationReport {
	features := []string{}

	if config.Level >= 1 {
		features = append(features, "变量名混淆")
		features = append(features, "代码压缩")
	}

	if config.Level >= 2 {
		features = append(features, "字符串加密")
		features = append(features, "控制流平坦化")
		features = append(features, "函数包装")
		features = append(features, "内存保护")
	}

	if config.Level >= 3 || config.EnableAll {
		features = append(features, "死代码注入")
		features = append(features, "高级反调试")
		features = append(features, "动态分析检测")
		features = append(features, "时间保护")
		features = append(features, "SHA-256完整性校验")
	}

	if config.EnableAll {
		features = append(features, "自毁机制")
		features = append(features, "堆栈跟踪混淆")
		features = append(features, "性能优化")
	}

	return ObfuscationReport{
		Version:        "3.0",
		BuildTime:      time.Now().Format(time.RFC3339),
		InputFile:      config.InputFile,
		OutputFile:     config.OutputFile,
		Level:          config.Level,
		OriginalSize:   originalSize,
		ObfuscatedSize: obfuscatedSize,
		Compression:    float64(originalSize-obfuscatedSize) / float64(originalSize) * 100,
		Features:      features,
		Checksum:      checksum,
		Stats: map[string]int{
			"compression_percent": int(float64(originalSize-obfuscatedSize) / float64(originalSize) * 100),
		},
	}
}
