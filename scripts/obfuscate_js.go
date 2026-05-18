package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hjtpx/hjtpx/backend/internal/tools"
)

func main() {
	inputFile := flag.String("input", "", "Input JavaScript file")
	outputFile := flag.String("output", "", "Output JavaScript file")
	level := flag.Int("level", 2, "Obfuscation level (1-3)")
	enableAll := flag.Bool("enable-all", false, "Enable all protection features")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("错误: 必须指定输入文件")
		os.Exit(1)
	}

	code, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("错误: 读取文件失败: %v\n", err)
		os.Exit(1)
	}

	config := tools.ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:        true,
		EnableCodeCompression:        true,
		EnableControlFlowFlattening: true,
		EnableDeadCodeInjection:     false,
		EnableFunctionWrapping:      true,
		CompressWhitespace:          true,
		RemoveComments:              true,
		PreserveConsole:             true,
		StringEncryptionMethod:      "aes-gcm",
		EnableNameMangling:          true,
	}

	switch *level {
	case 1:
		config.EnableControlFlowFlattening = false
		config.EnableStringEncryption = false
		config.EnableDeadCodeInjection = false
	case 2:
		config.EnableDeadCodeInjection = true
	case 3:
		config.EnableDeadCodeInjection = true
	}

	if *enableAll {
		config.EnableAdvancedAntiDebug = true
		config.EnableSelfDestruct = true
		config.EnableMemoryProtection = true
		config.EnableCodeIntegrity = true
		config.EnableDynamicAnalysis = true
		config.EnableTimingProtection = true
		config.EnableExceptionHandling = true
	}

	obfuscator := tools.NewObfuscator(config)
	result, err := obfuscator.Obfuscate(string(code))
	if err != nil {
		fmt.Printf("错误: 混淆失败: %v\n", err)
		os.Exit(1)
	}

	if *outputFile == "" {
		*outputFile = *inputFile
	}

	err = os.WriteFile(*outputFile, []byte(result), 0644)
	if err != nil {
		fmt.Printf("错误: 写入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("混淆完成: %s\n", *outputFile)
	fmt.Printf("原始大小: %d bytes\n", len(code))
	fmt.Printf("混淆后: %d bytes\n", len(result))

	stats := obfuscator.GetStats()
	fmt.Println("\n混淆统计:")
	fmt.Printf("- 变量混淆: %d 个\n", stats["variables_obfuscated"])
	fmt.Printf("- 字符串加密: %d 个\n", stats["strings_encrypted"])
	fmt.Printf("- 函数包装: %d 个\n", stats["functions_wrapped"])
}

func init() {
	if !strings.HasPrefix(os.Args[0], "go-") {
		flag.CommandLine.Parse(os.Args[1:])
	}
}
