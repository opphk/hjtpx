package fuzzing

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type FuzzTestResult struct {
	TotalTests     int
	PassedTests    int
	FailedTests    int
	CrashedTests   int
	PanicDetected  bool
	MemoryLeakHits int
	TimeoutHits    int
	Duration       time.Duration
	Bugs           []FuzzBug
}

type FuzzBug struct {
	Input       string
	BugType     string
	Severity    string
	Description string
}

type FuzzConfig struct {
	MaxIterations   int
	Timeout         time.Duration
	MaxInputSize    int
	EnableCoverage  bool
	ParallelWorkers int
}

var DefaultFuzzConfig = FuzzConfig{
	MaxIterations:   10000,
	Timeout:         30 * time.Second,
	MaxInputSize:    1024 * 1024,
	EnableCoverage:  true,
	ParallelWorkers: 4,
}

type CaptchaFuzzer struct {
	config     FuzzConfig
	generators []InputGenerator
	oracles    []BugOracle
}

type InputGenerator func() []byte

type BugOracle func(input []byte, result interface{}) *FuzzBug

func NewCaptchaFuzzer(config FuzzConfig) *CaptchaFuzzer {
	return &CaptchaFuzzer{
		config:     config,
		generators: []InputGenerator{},
		oracles:    []BugOracle{},
	}
}

func (f *CaptchaFuzzer) RegisterGenerator(name string, gen InputGenerator) {
	f.generators = append(f.generators, gen)
}

func (f *CaptchaFuzzer) RegisterOracle(oracle BugOracle) {
	f.oracles = append(f.oracles, oracle)
}

func (f *CaptchaFuzzer) Run() *FuzzTestResult {
	result := &FuzzTestResult{
		Bugs: []FuzzBug{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.config.Timeout)
	defer cancel()

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	for i := 0; i < f.config.MaxIterations && ctx.Err() == nil; i++ {
		result.TotalTests++
		input := f.generateInput()
		testResult := f.executeWithTimeout(input)

		if testResult.Timeout {
			result.TimeoutHits++
			continue
		}

		for _, oracle := range f.oracles {
			if bug := oracle(input, testResult.Result); bug != nil {
				result.FailedTests++
				result.Bugs = append(result.Bugs, *bug)
				break
			}
		}

		if len(result.Bugs) == 0 || result.Bugs[len(result.Bugs)-1].BugType == "" {
			result.PassedTests++
		}
	}

	return result
}

type TestResult struct {
	Timeout bool
	Panic   bool
	Result  interface{}
}

func (f *CaptchaFuzzer) generateInput() []byte {
	if len(f.generators) == 0 {
		return generateRandomInput(f.config.MaxInputSize)
	}

	gen := f.generators[rand.Intn(len(f.generators))]
	input := gen()

	if len(input) > f.config.MaxInputSize {
		input = input[:f.config.MaxInputSize]
	}

	return input
}

func generateRandomInput(maxSize int) []byte {
	size := rand.Intn(maxSize) + 1
	input := make([]byte, size)
	rand.Read(input)
	return input
}

func (f *CaptchaFuzzer) executeWithTimeout(input []byte) TestResult {
	resultCh := make(chan TestResult, 1)

	go func() {
		result := f.executeTest(input)
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result
	case <-time.After(5 * time.Second):
		return TestResult{Timeout: true}
	}
}

func (f *CaptchaFuzzer) executeTest(input []byte) TestResult {
	return TestResult{
		Result: map[string]interface{}{
			"input_size": len(input),
		},
	}
}

type CaptchaInputGenerator struct{}

func NewCaptchaInputGenerator() *CaptchaInputGenerator {
	return &CaptchaInputGenerator{}
}

func (g *CaptchaInputGenerator) GenerateSliderInput() []byte {
	return []byte(fmt.Sprintf(`{"session_id": "test_%d", "trajectory": [[0,50,0],[10,48,50]]}`, time.Now().UnixNano()))
}

func (g *CaptchaInputGenerator) GenerateMalformedInput() []byte {
	templates := []string{
		`{"session_id": "%s", "invalid_field": "%s"}`,
		`{`,
		`}`,
		`{"session_id":`,
		`null`,
		`""`,
		`{}`,
		`[[]]`,
	}

	template := templates[rand.Intn(len(templates))]
	return []byte(fmt.Sprintf(template, g.randomString(10), g.randomString(20)))
}

func (g *CaptchaInputGenerator) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func SliderVerificationOracle(input []byte, result interface{}) *FuzzBug {
	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return &FuzzBug{
			Input:       truncateString(string(input), 200),
			BugType:     "parse_error",
			Severity:    "medium",
			Description:  "Failed to parse slider verification input",
		}
	}

	if trajectory, ok := data["trajectory"].([]interface{}); ok {
		for i, t := range trajectory {
			if pt, ok := t.([]interface{}); ok && len(pt) >= 2 {
				for j, p := range pt {
					if f, ok := p.(float64); ok {
						if f < 0 {
							return &FuzzBug{
								Input:       truncateString(string(input), 200),
								BugType:     "invalid_trajectory",
								Severity:    "high",
								Description: fmt.Sprintf("Negative coordinate at index %d, point %d", i, j),
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func RunSliderFuzzTests() *FuzzTestResult {
	fuzzer := NewCaptchaFuzzer(DefaultFuzzConfig)
	generator := NewCaptchaInputGenerator()

	fuzzer.RegisterGenerator("slider", generator.GenerateSliderInput)
	fuzzer.RegisterGenerator("malformed", generator.GenerateMalformedInput)

	fuzzer.RegisterOracle(SliderVerificationOracle)

	return fuzzer.Run()
}

func RunAllFuzzTests() map[string]*FuzzTestResult {
	results := make(map[string]*FuzzTestResult)
	results["slider"] = RunSliderFuzzTests()
	return results
}

func GenerateFuzzReport(results map[string]*FuzzTestResult) string {
	var buf bytes.Buffer

	buf.WriteString("# Fuzzing Test Report\n\n")
	buf.WriteString(fmt.Sprintf("Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	totalTests := 0
	totalPassed := 0

	for name, result := range results {
		totalTests += result.TotalTests
		totalPassed += result.PassedTests

		buf.WriteString(fmt.Sprintf("## %s Fuzz Tests\n", strings.ToUpper(name)))
		buf.WriteString(fmt.Sprintf("- Total: %d\n", result.TotalTests))
		buf.WriteString(fmt.Sprintf("- Passed: %d\n", result.PassedTests))
		buf.WriteString(fmt.Sprintf("- Failed: %d\n", result.FailedTests))
		buf.WriteString(fmt.Sprintf("- Duration: %s\n\n", result.Duration))
	}

	if totalTests > 0 {
		passRate := float64(totalPassed) / float64(totalTests) * 100
		buf.WriteString(fmt.Sprintf("- Pass Rate: %.2f%%\n", passRate))
	}

	return buf.String()
}
