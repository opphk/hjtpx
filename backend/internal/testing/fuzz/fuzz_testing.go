package fuzz

import (
	"math/rand"
	"testing"
	"time"
)

// FuzzingConfig 配置模糊测试参数
type FuzzingConfig struct {
	Iterations  int
	Seed        int64
	Timeout     time.Duration
	Verbose     bool
	MinLength   int
	MaxLength   int
}

// DefaultFuzzingConfig 默认配置
func DefaultFuzzingConfig() *FuzzingConfig {
	return &FuzzingConfig{
		Iterations: 10000,
		Seed:       time.Now().UnixNano(),
		Timeout:    5 * time.Minute,
		Verbose:    false,
		MinLength:  0,
		MaxLength:  1024,
	}
}

// FuzzingResult 模糊测试结果
type FuzzingResult struct {
	TotalIterations int
	Failures        int
	Panics          int
	Timeouts        int
	CrashInputs     [][]byte
	Elapsed         time.Duration
}

// Fuzzer 模糊测试器
type Fuzzer struct {
	config *FuzzingConfig
	rng    *rand.Rand
}

// NewFuzzer 创建模糊测试器
func NewFuzzer(config *FuzzingConfig) *Fuzzer {
	if config == nil {
		config = DefaultFuzzingConfig()
	}
	return &Fuzzer{
		config: config,
		rng:    rand.New(rand.NewSource(config.Seed)),
	}
}

// FuzzBytes 生成随机字节切片
func (f *Fuzzer) FuzzBytes(length int) []byte {
	if length <= 0 {
		length = f.rng.Intn(f.config.MaxLength-f.config.MinLength+1) + f.config.MinLength
	}
	b := make([]byte, length)
	f.rng.Read(b)
	return b
}

// FuzzString 生成随机字符串
func (f *Fuzzer) FuzzString(length int) string {
	return string(f.FuzzBytes(length))
}

// FuzzInt 生成随机整数
func (f *Fuzzer) FuzzInt() int {
	return f.rng.Int()
}

// FuzzInt64 生成随机int64
func (f *Fuzzer) FuzzInt64() int64 {
	return f.rng.Int63()
}

// FuzzFloat64 生成随机float64
func (f *Fuzzer) FuzzFloat64() float64 {
	return f.rng.Float64()
}

// FuzzBool 生成随机布尔值
func (f *Fuzzer) FuzzBool() bool {
	return f.rng.Float32() < 0.5
}

// Run 运行模糊测试
func (f *Fuzzer) Run(t *testing.T, name string, testFn func([]byte) error) *FuzzingResult {
	t.Helper()
	
	result := &FuzzingResult{
		TotalIterations: f.config.Iterations,
		CrashInputs:     make([][]byte, 0),
	}
	
	start := time.Now()
	defer func() {
		result.Elapsed = time.Since(start)
	}()
	
	for i := 0; i < f.config.Iterations; i++ {
		input := f.FuzzBytes(0)
		
		func() {
			defer func() {
				if r := recover(); r != nil {
					result.Panics++
					result.CrashInputs = append(result.CrashInputs, input)
					if f.config.Verbose {
						t.Logf("[%s] Panic at iteration %d with input: %v", name, i, input)
					}
				}
			}()
			
			err := testFn(input)
			if err != nil {
				result.Failures++
				if f.config.Verbose {
					t.Logf("[%s] Failure at iteration %d: %v", name, i, err)
				}
			}
		}()
	}
	
	return result
}

// EdgeCase 生成边缘情况输入
func EdgeCases() [][]byte {
	return [][]byte{
		{},
		{0x00},
		{0xFF},
		{0x00, 0x00},
		{0xFF, 0xFF},
		[]byte("nil"),
		[]byte("null"),
		[]byte("undefined"),
		[]byte(""),
		[]byte(" "),
		[]byte("\x00"),
		[]byte("\\x00"),
		[]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		[]byte("<script>alert('xss')</script>"),
		[]byte("' OR '1'='1"),
		[]byte(github.com/hjtpx/hjtpx/../../../etc/passwd"),
	}
}
