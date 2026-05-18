package main

import (
	"fmt"
	tools "github.com/hjtpx/hjtpx/internal/tools"
)

func main() {
	sampleCode := `
		function calculateSum(a, b) {
			return a + b;
		}

		function validateInput(data) {
			if (data.length < 5) {
				return false;
			}
			return true;
		}

		console.log("Starting application");
		var result = calculateSum(10, 20);
		console.log("Result:", result);
	`

	fmt.Println("Original code length:", len(sampleCode))
	fmt.Println("\n=== Testing Basic Obfuscation ===")

	obfuscated, err := tools.Obfuscate(sampleCode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Obfuscated code length: %d\n", len(obfuscated))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(obfuscated))/float64(len(sampleCode))*100)

	fmt.Println("\n=== Testing Full Protection ===")

	fullProtected, err := tools.NewObfuscator(tools.ObfuscatorConfig{
		EnableVariableObfuscation:    true,
		EnableStringEncryption:       true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
		EnableAdvancedAntiDebug:     true,
		EnableMemoryProtection:      true,
		EnableCodeIntegrity:        true,
	}).ApplyFullProtection(sampleCode)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Full protected code length: %d\n", len(fullProtected))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(fullProtected))/float64(len(sampleCode))*100)

	fmt.Println("\n=== Testing Maximum Protection ===")

	maxProtected, err := tools.NewObfuscator(tools.ObfuscatorConfig{
		EnableVariableObfuscation:          true,
		EnableStringEncryption:             true,
		EnableCodeCompression:              true,
		EnableControlFlowFlattening:        true,
		EnableAdvancedAntiDebug:           true,
		EnableMemoryProtection:             true,
		EnableCodeIntegrity:               true,
		EnableNumberObfuscation:            true,
		EnableBooleanObfuscation:          true,
		EnableArrayLiteralObfuscation:      true,
	}).ApplyMaximumProtection(sampleCode)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Maximum protected code length: %d\n", len(maxProtected))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(maxProtected))/float64(len(sampleCode))*100)

	fmt.Println("\n=== Testing Integrity Hashes ===")

	hashes, err := tools.GenerateIntegrityHashes(sampleCode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("SHA256: %s\n", hashes.SHA256)
	fmt.Printf("SHA384: %s\n", hashes.SHA384)
	fmt.Printf("SHA512: %s\n", hashes.SHA512)
	fmt.Printf("MD5: %s\n", hashes.MD5)

	fmt.Println("\n=== Testing Quality Score ===")

	report := tools.GenerateCodeObfuscationReport(sampleCode, tools.ObfuscatorConfig{
		EnableVariableObfuscation:    true,
		EnableStringEncryption:       true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
	})

	fmt.Printf("Original size: %d\n", report["original_size"])
	fmt.Printf("Obfuscated size: %d\n", report["obfuscated_size"])
	fmt.Printf("Quality score: %.2f\n", report["quality_score"])

	entropy := tools.CalculateObfuscationEntropy(maxProtected)
	fmt.Printf("Entropy: %.2f\n", entropy)

	fmt.Println("\n=== Testing LRU Cache ===")

	cache := tools.NewLRUCache(3)
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")

	val, exists := cache.Get("key1")
	fmt.Printf("key1 exists: %v, value: %s\n", exists, val)

	cache.Put("key4", "value4")

	_, exists = cache.Get("key1")
	fmt.Printf("key1 after eviction: %v\n", exists)

	_, exists = cache.Get("key2")
	fmt.Printf("key2 after eviction: %v\n", exists)

	stats := cache.GetStats()
	fmt.Printf("Cache stats: %v\n", stats)

	fmt.Println("\n=== All tests completed successfully ===")
}
