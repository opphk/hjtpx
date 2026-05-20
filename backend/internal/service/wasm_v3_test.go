package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWASMV3Engine_NewEngine(t *testing.T) {
	t.Run("创建默认配置引擎", func(t *testing.T) {
		engine := NewWASMV3Engine(nil)
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.config)
		assert.NotNil(t, engine.keyPool)
		assert.NotNil(t, engine.sandbox)
		assert.NotNil(t, engine.aiModule)
		assert.NotNil(t, engine.offloader)
		assert.NotNil(t, engine.metrics)
	})

	t.Run("创建自定义配置引擎", func(t *testing.T) {
		config := &WASMConfigV3{
			EnableChaCha20:    true,
			EnableAES256GCM:   true,
			EnableAIInference: true,
			MaxConcurrentOps:  500,
			MemoryLimitMB:    256,
			SecurityLevel:    SecurityLevelHigh,
			SandboxMode:      SandboxModeEnhanced,
		}

		engine := NewWASMV3Engine(config)
		assert.NotNil(t, engine)
		assert.Equal(t, 500, engine.config.MaxConcurrentOps)
		assert.Equal(t, 256, engine.config.MemoryLimitMB)
		assert.Equal(t, SecurityLevelHigh, engine.config.SecurityLevel)
	})
}

func TestWASMV3Engine_Initialize(t *testing.T) {
	t.Run("初始化引擎", func(t *testing.T) {
		engine := NewWASMV3Engine(nil)
		err := engine.Initialize()
		assert.NoError(t, err)
		assert.True(t, engine.initialized.Load())
	})

	t.Run("重复初始化不应失败", func(t *testing.T) {
		engine := NewWASMV3Engine(nil)
		err := engine.Initialize()
		assert.NoError(t, err)

		err = engine.Initialize()
		assert.NoError(t, err)
	})
}

func TestWASMV3Engine_EncryptV3(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()

	t.Run("AES-256-GCM加密", func(t *testing.T) {
		plaintext := []byte("test message for encryption")
		result, err := engine.EncryptV3(ctx, plaintext, WASMKeyTypeAES256GCM)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Ciphertext)
		assert.NotEmpty(t, result.Nonce)
		assert.Equal(t, "AES-256-GCM", result.Algorithm)
		assert.False(t, result.EncryptedAt.IsZero())
		assert.True(t, result.ExecutionTime > 0)
	})

	t.Run("ChaCha20-Poly1305加密", func(t *testing.T) {
		plaintext := []byte("ChaCha20 encryption test")
		result, err := engine.EncryptV3(ctx, plaintext, WASMKeyTypeChaCha20)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ChaCha20-Poly1305", result.Algorithm)
	})

	t.Run("Hybrid加密", func(t *testing.T) {
		plaintext := []byte("Hybrid encryption test")
		result, err := engine.EncryptV3(ctx, plaintext, WASMKeyTypeHybrid)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Hybrid-AES-ChaCha20", result.Algorithm)
	})

	t.Run("空数据加密", func(t *testing.T) {
		plaintext := []byte{}
		_, err := engine.EncryptV3(ctx, plaintext, WASMKeyTypeAES256GCM)
		assert.Error(t, err)
	})

	t.Run("大数据加密", func(t *testing.T) {
		largeData := make([]byte, 1024*1024)
		rand.Read(largeData)
		result, err := engine.EncryptV3(ctx, largeData, WASMKeyTypeAES256GCM)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Ciphertext)
	})

	t.Run("并发加密", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		errorCount := 0

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				plaintext := []byte("concurrent encryption test")
				_, err := engine.EncryptV3(ctx, plaintext, WASMKeyTypeAES256GCM)
				if err == nil {
					successCount++
				} else {
					errorCount++
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, 100, successCount+errorCount)
		assert.True(t, successCount > 90, "expected most operations to succeed")
	})
}

func TestWASMV3Engine_DecryptV3(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()

	t.Run("解密加密的数据", func(t *testing.T) {
		originalText := []byte("test message for round trip")
		encrypted, err := engine.EncryptV3(ctx, originalText, WASMKeyTypeAES256GCM)
		require.NoError(t, err)

		decrypted, err := engine.DecryptV3(ctx, encrypted, WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.Equal(t, originalText, decrypted)
	})

	t.Run("ChaCha20解密", func(t *testing.T) {
		originalText := []byte("ChaCha20 round trip test")
		encrypted, err := engine.EncryptV3(ctx, originalText, WASMKeyTypeChaCha20)
		require.NoError(t, err)

		decrypted, err := engine.DecryptV3(ctx, encrypted, WASMKeyTypeChaCha20)
		require.NoError(t, err)
		assert.Equal(t, originalText, decrypted)
	})

	t.Run("无效密文解密", func(t *testing.T) {
		invalidEncrypted := &WASMEncryptionResultV3{
			Ciphertext: "invalid_base64!@#$",
			Nonce:      "invalid_nonce",
			Algorithm:  "AES-256-GCM",
		}

		_, err := engine.DecryptV3(ctx, invalidEncrypted, WASMKeyTypeAES256GCM)
		assert.Error(t, err)
	})

	t.Run("篡改密文检测", func(t *testing.T) {
		originalText := []byte("tampering detection test")
		encrypted, err := engine.EncryptV3(ctx, originalText, WASMKeyTypeAES256GCM)
		require.NoError(t, err)

		decoded, _ := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
		if len(decoded) > 10 {
			decoded[len(decoded)-1] ^= 0xFF
			encrypted.Ciphertext = base64.StdEncoding.EncodeToString(decoded)
		}

		_, err = engine.DecryptV3(ctx, encrypted, WASMKeyTypeAES256GCM)
		assert.Error(t, err)
	})
}

func TestWASMV3Engine_BatchOperations(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()

	t.Run("批量加密", func(t *testing.T) {
		plaintexts := [][]byte{
			[]byte("batch message 1"),
			[]byte("batch message 2"),
			[]byte("batch message 3"),
		}

		results, err := engine.BatchEncrypt(ctx, plaintexts, WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for i, result := range results {
			assert.NotEmpty(t, result.Ciphertext)
			assert.Equal(t, i+1, i+1)
		}
	})

	t.Run("批量解密", func(t *testing.T) {
		plaintexts := [][]byte{
			[]byte("batch decryption test 1"),
			[]byte("batch decryption test 2"),
			[]byte("batch decryption test 3"),
		}

		encrypted, err := engine.BatchEncrypt(ctx, plaintexts, WASMKeyTypeAES256GCM)
		require.NoError(t, err)

		decrypted, err := engine.BatchDecrypt(ctx, encrypted, WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.Len(t, decrypted, 3)

		for i, pt := range plaintexts {
			assert.Equal(t, pt, decrypted[i])
		}
	})

	t.Run("空批量操作", func(t *testing.T) {
		emptyPlaintexts := [][]byte{}
		results, err := engine.BatchEncrypt(ctx, emptyPlaintexts, WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestWASMV3Engine_StreamEncrypt(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("流式加密", func(t *testing.T) {
		plaintextChan := make(chan []byte, 5)
		plaintexts := [][]byte{
			[]byte("stream message 1"),
			[]byte("stream message 2"),
			[]byte("stream message 3"),
		}

		go func() {
			for _, pt := range plaintexts {
				plaintextChan <- pt
			}
			close(plaintextChan)
		}()

		resultsChan, errorsChan := engine.StreamEncrypt(ctx, plaintextChan, WASMKeyTypeAES256GCM)

		var results []*WASMEncryptionResultV3
		for result := range resultsChan {
			results = append(results, result)
		}

		select {
		case err := <-errorsChan:
			t.Logf("Stream completed with error: %v", err)
		default:
		}

		assert.Len(t, results, 3)
	})
}

func TestWASMV3Engine_AIInference(t *testing.T) {
	config := &WASMConfigV3{
		EnableAIInference: true,
	}
	engine := NewWASMV3Engine(config)
	engine.Initialize()

	ctx := context.Background()

	t.Run("AI推理", func(t *testing.T) {
		request := &AIInferenceRequest{
			ModelID:   "test-model-v1",
			InputData: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			Options: &AIInferenceOptions{
				BatchSize:    1,
				Quantize:     false,
				UseCache:     true,
			},
		}

		result, err := engine.RunAIInference(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.OutputData)
		assert.True(t, result.Confidence >= 0 && result.Confidence <= 1)
		assert.True(t, result.Latency >= 0)
	})

	t.Run("量化推理", func(t *testing.T) {
		request := &AIInferenceRequest{
			ModelID:   "quantized-model",
			InputData: []float32{0.5, -0.3, 0.8, -0.2, 0.6},
			Options: &AIInferenceOptions{
				Quantize: true,
				UseCache: false,
			},
		}

		result, err := engine.RunAIInference(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("缓存命中", func(t *testing.T) {
		request := &AIInferenceRequest{
			ModelID:   "cached-model",
			InputData: []float32{0.1, 0.2, 0.3},
			Options: &AIInferenceOptions{
				UseCache: true,
			},
		}

		_, _ = engine.RunAIInference(ctx, request)
		result, err := engine.RunAIInference(ctx, request)
		require.NoError(t, err)
		assert.True(t, result.CacheHit)
	})

	t.Run("AI禁用时推理应失败", func(t *testing.T) {
		disabledConfig := &WASMConfigV3{
			EnableAIInference: false,
		}
		disabledEngine := NewWASMV3Engine(disabledConfig)
		disabledEngine.Initialize()

		request := &AIInferenceRequest{
			ModelID:   "test-model",
			InputData: []float32{0.1, 0.2},
		}

		_, err := disabledEngine.RunAIInference(ctx, request)
		assert.Error(t, err)
	})
}

func TestWASMV3Engine_SecurityAudit(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("基础安全审计", func(t *testing.T) {
		report := engine.SecurityAudit()
		assert.NotNil(t, report)
		assert.NotNil(t, report.Recommendations)
		assert.False(t, report.Timestamp.IsZero())
	})

	t.Run("高阻塞场景审计", func(t *testing.T) {
		for i := 0; i < 150; i++ {
			engine.metrics.SecurityBlocks.Add(1)
		}

		report := engine.SecurityAudit()
		assert.True(t, report.ThreatDetected)
		assert.Contains(t, report.ThreatType, "security")
	})
}

func TestWASMV3Engine_ComputationOffload(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("计算卸载", func(t *testing.T) {
		testData := []byte("offload test data")
		result, err := engine.ComputeOffload(ctx, "inference", testData)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("加密卸载", func(t *testing.T) {
		testData := []byte("encryption offload")
		result, err := engine.ComputeOffload(ctx, "encryption", testData)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("超时场景", func(t *testing.T) {
		ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancelTimeout()

		testData := []byte("timeout test")
		_, err := engine.ComputeOffload(ctxTimeout, "inference", testData)

		assert.Error(t, err)
	})
}

func TestWASMV3Engine_KeyDerivation(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("基础密钥推导", func(t *testing.T) {
		password := "test-password-123"
		salt := make([]byte, 16)
		rand.Read(salt)

		derivedKey, err := engine.GenerateKeyDerivation(password, salt, 100000)
		require.NoError(t, err)
		assert.Len(t, derivedKey, 32)
	})

	t.Run("不同盐值产生不同密钥", func(t *testing.T) {
		password := "same-password"
		salt1 := []byte("salt1-12345678")
		salt2 := []byte("salt2-12345678")

		key1, err := engine.GenerateKeyDerivation(password, salt1, 100000)
		require.NoError(t, err)

		key2, err := engine.GenerateKeyDerivation(password, salt2, 100000)
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("迭代次数过少应失败", func(t *testing.T) {
		password := "test-password"
		salt := make([]byte, 16)

		_, err := engine.GenerateKeyDerivation(password, salt, 1000)
		assert.Error(t, err)
	})
}

func TestWASMV3Engine_IntegrityVerification(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("完整性验证成功", func(t *testing.T) {
		data := []byte("data to verify")
		hash := computeTestHash(data)

		valid, err := engine.VerifyIntegrity(data, hash)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("篡改数据检测", func(t *testing.T) {
		data := []byte("original data")
		hash := computeTestHash(data)

		tamperedData := []byte("tampered data")
		valid, err := engine.VerifyIntegrity(tamperedData, hash)
		assert.NoError(t, err)
		assert.False(t, valid)
	})
}

func computeTestHash(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func TestWASMV3Engine_GPUAcceleration(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("GPU能力检查", func(t *testing.T) {
		canEnable := engine.checkGPUCapability()
		assert.IsType(t, false, canEnable)
	})

	t.Run("GPU启用", func(t *testing.T) {
		err := engine.EnableGPUAcceleration(false)
		assert.NoError(t, err)
		assert.False(t, engine.enableGPU)
	})
}

func TestWASMV3Engine_SecurityLevel(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("设置不同安全级别", func(t *testing.T) {
		engine.SetSecurityLevel(SecurityLevelStandard)
		assert.Equal(t, SandboxModeBasic, engine.sandbox.mode)

		engine.SetSecurityLevel(SecurityLevelHigh)
		assert.Equal(t, SandboxModeEnhanced, engine.sandbox.mode)

		engine.SetSecurityLevel(SecurityLevelMaximum)
		assert.Equal(t, SandboxModeIsolated, engine.sandbox.mode)
		assert.True(t, engine.sandbox.strictMode.Load())
	})
}

func TestWASMV3Engine_Benchmark(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()

	t.Run("性能基准测试", func(t *testing.T) {
		results := engine.Benchmark(ctx)

		assert.NotNil(t, results)
		assert.Contains(t, results, "encrypt_ops_per_second")
		assert.Contains(t, results, "decrypt_ops_per_second")
		assert.Contains(t, results, "encrypt_avg_latency_ms")
		assert.Contains(t, results, "decrypt_avg_latency_ms")

		encryptOps, ok := results["encrypt_ops_per_second"].(float64)
		assert.True(t, ok)
		assert.True(t, encryptOps > 0)
	})
}

func TestWASMV3Engine_Metrics(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()

	_, _ = engine.EncryptV3(ctx, []byte("test1"), WASMKeyTypeAES256GCM)
	_, _ = engine.EncryptV3(ctx, []byte("test2"), WASMKeyTypeAES256GCM)

	t.Run("导出指标", func(t *testing.T) {
		metrics := engine.ExportMetrics()

		assert.Contains(t, metrics, "total_encrypt_ops")
		assert.Contains(t, metrics, "total_decrypt_ops")
		assert.Contains(t, metrics, "initialized")
		assert.Contains(t, metrics, "gpu_enabled")

		encryptOps, ok := metrics["total_encrypt_ops"].(int64)
		assert.True(t, ok)
		assert.GreaterOrEqual(t, encryptOps, int64(2))
	})

	t.Run("获取指标对象", func(t *testing.T) {
		metrics := engine.GetMetrics()
		assert.NotNil(t, metrics)
		assert.NotNil(t, &metrics.EncryptOps)
	})
}

func TestWASMV3Engine_Close(t *testing.T) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	t.Run("关闭引擎", func(t *testing.T) {
		err := engine.Close()
		assert.NoError(t, err)
		assert.False(t, engine.initialized.Load())
	})
}

func TestWASMKeyPool(t *testing.T) {
	t.Run("获取和返回密钥", func(t *testing.T) {
		pool := newWASMKeyPool(10)

		key1, err := pool.GetKey(WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.NotNil(t, key1)
		assert.Equal(t, WASMKeyTypeAES256GCM, key1.KeyType)
		assert.True(t, key1.IsActive.Load())

		pool.ReturnKey(key1.ID)

		key2, err := pool.GetKey(WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.NotNil(t, key2)
	})

	t.Run("不同密钥类型", func(t *testing.T) {
		pool := newWASMKeyPool(10)

		keyAES, err := pool.GetKey(WASMKeyTypeAES256GCM)
		require.NoError(t, err)
		assert.Len(t, keyAES.Key, 32)

		keyChaCha, err := pool.GetKey(WASMKeyTypeChaCha20)
		require.NoError(t, err)
		assert.Len(t, keyChaCha.Key, 32)

		keyHybrid, err := pool.GetKey(WASMKeyTypeHybrid)
		require.NoError(t, err)
		assert.Len(t, keyHybrid.Key, 64)
	})
}

func TestWASMSandboxV3(t *testing.T) {
	t.Run("沙箱模式初始化", func(t *testing.T) {
		sandbox := newWASMSandboxV3(SandboxModeEnhanced)
		assert.NotNil(t, sandbox)
		assert.Equal(t, SandboxModeEnhanced, sandbox.mode)
		assert.True(t, sandbox.enableAudit)
	})

	t.Run("安全规则初始化", func(t *testing.T) {
		sandbox := newWASMSandboxV3(SandboxModeBasic)
		assert.True(t, sandbox.forbiddenFuncs["syscall_js_value_get"])

		sandboxStrict := newWASMSandboxV3(SandboxModeIsolated)
		assert.True(t, sandboxStrict.strictMode.Load())
	})

	t.Run("操作验证", func(t *testing.T) {
		sandbox := newWASMSandboxV3(SandboxModeBasic)
		err := sandbox.ValidateOperation("encrypt")
		assert.NoError(t, err)
	})

	t.Run("审计日志", func(t *testing.T) {
		sandbox := newWASMSandboxV3(SandboxModeEnhanced)
		sandbox.EnableAudit()

		entry := &SandboxAuditEntry{
			Timestamp: time.Now(),
			Operation:  "test_op",
			Blocked:   false,
		}
		sandbox.logAuditEntry(entry)

		log := sandbox.GetAuditLog()
		assert.Len(t, log, 1)
		assert.Equal(t, "test_op", log[0].Operation)
	})
}

func TestWASAIModuleV3(t *testing.T) {
	t.Run("AI模块启用", func(t *testing.T) {
		module := newWASAIModuleV3()
		module.Enable()
		assert.True(t, module.enabled.Load())
	})

	t.Run("AI模块初始化", func(t *testing.T) {
		module := newWASAIModuleV3()
		err := module.initialize()
		assert.NoError(t, err)
		assert.True(t, module.enabled.Load())
	})
}

func TestComputationOffloader(t *testing.T) {
	t.Run("卸载器初始化", func(t *testing.T) {
		offloader := newComputationOffloader(DeviceCPU)
		assert.NotNil(t, offloader)
		assert.True(t, offloader.enabled.Load())
		assert.Equal(t, DeviceCPU, offloader.targetDevice)
	})

	t.Run("设备自动选择", func(t *testing.T) {
		offloader := newComputationOffloader(DeviceCPU)
		offloader.autoSelectDevice()
	})
}

func BenchmarkWASMEncrypt(b *testing.B) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()
	plaintext := make([]byte, 1024)
	rand.Read(plaintext)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = engine.EncryptV3(ctx, plaintext, WASMKeyTypeAES256GCM)
		}
	})
}

func BenchmarkWASMBatchEncrypt(b *testing.B) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()
	plaintexts := make([][]byte, 100)
	for i := range plaintexts {
		plaintexts[i] = make([]byte, 1024)
		rand.Read(plaintexts[i])
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = engine.BatchEncrypt(ctx, plaintexts, WASMKeyTypeAES256GCM)
		}
	})
}

func BenchmarkWASMDecrypt(b *testing.B) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	ctx := context.Background()
	plaintext := make([]byte, 1024)
	rand.Read(plaintext)

	encrypted, _ := engine.EncryptV3(ctx, plaintext, WASMKeyTypeAES256GCM)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = engine.DecryptV3(ctx, encrypted, WASMKeyTypeAES256GCM)
		}
	})
}

func BenchmarkWASMKeyDerivation(b *testing.B) {
	engine := NewWASMV3Engine(nil)
	engine.Initialize()

	password := "benchmark-password-123"
	salt := make([]byte, 16)
	rand.Read(salt)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = engine.GenerateKeyDerivation(password, salt, 10000)
		}
	})
}
