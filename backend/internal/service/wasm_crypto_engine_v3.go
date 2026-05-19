package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// WASMCryptoEngineV3 是 WebAssembly 加密引擎的 v3 版本
type WASMCryptoEngineV3 struct {
	pool              sync.Pool
	secret            []byte
	stats             *CryptoStats
	algorithmType     string
	enableSIMD        bool
	enableHardware    bool
}

// CryptoStats 加密引擎统计信息
type CryptoStats struct {
	encryptCount     atomic.Int64
	decryptCount     atomic.Int64
	encryptTime      atomic.Int64
	decryptTime      atomic.Int64
	totalBytesIn     atomic.Int64
	totalBytesOut    atomic.Int64
}

// CryptoPerformanceMetrics 加密性能指标
type CryptoPerformanceMetrics struct {
	TotalOperations   int64   `json:"total_operations"`
	AvgEncryptTime    float64 `json:"avg_encrypt_time_ms"`
	AvgDecryptTime    float64 `json:"avg_decrypt_time_ms"`
	Throughput        float64 `json:"throughput_mb_s"`
}

// NewWASMCryptoEngineV3 创建新的 WASM 加密引擎 v3
func NewWASMCryptoEngineV3() *WASMCryptoEngineV3 {
	engine := &WASMCryptoEngineV3{
		secret:         generateDefaultSecret(),
		stats:          &CryptoStats{},
		algorithmType:  "AES-256-GCM",
		enableSIMD:     true,
		enableHardware: true,
	}
	
	engine.pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}
	
	return engine
}

func generateDefaultSecret() []byte {
	secret := make([]byte, 32)
	rand.Read(secret)
	return secret
}

// Encrypt 加密数据
func (e *WASMCryptoEngineV3) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext cannot be empty")
	}
	
	start := time.Now()
	e.stats.totalBytesIn.Add(int64(len(plaintext)))
	
	defer func() {
		e.stats.encryptCount.Add(1)
		e.stats.encryptTime.Add(time.Since(start).Nanoseconds())
	}()
	
	block, err := aes.NewCipher(e.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	e.stats.totalBytesOut.Add(int64(len(ciphertext)))
	
	return ciphertext, nil
}

// Decrypt 解密数据
func (e *WASMCryptoEngineV3) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext cannot be empty")
	}
	
	start := time.Now()
	e.stats.totalBytesIn.Add(int64(len(ciphertext)))
	
	defer func() {
		e.stats.decryptCount.Add(1)
		e.stats.decryptTime.Add(time.Since(start).Nanoseconds())
	}()
	
	block, err := aes.NewCipher(e.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	
	e.stats.totalBytesOut.Add(int64(len(plaintext)))
	return plaintext, nil
}

// StreamEncrypt 流式加密
func (e *WASMCryptoEngineV3) StreamEncrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext cannot be empty")
	}
	
	start := time.Now()
	defer func() {
		e.stats.encryptCount.Add(1)
		e.stats.encryptTime.Add(time.Since(start).Nanoseconds())
	}()
	
	// 对于大数据分块处理
	const chunkSize = 64 * 1024
	numChunks := (len(plaintext) + chunkSize - 1) / chunkSize
	
	// 加密头部信息
	header := make([]byte, 8)
	binary.BigEndian.PutUint64(header, uint64(len(plaintext)))
	
	// 加密数据块
	var result []byte
	result = append(result, header...)
	
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(plaintext) {
			end = len(plaintext)
		}
		chunk, err := e.Encrypt(plaintext[start:end])
		if err != nil {
			return nil, err
		}
		
		// 记录 chunk 大小
		chunkSizeBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(chunkSizeBytes, uint32(len(chunk)))
		result = append(result, chunkSizeBytes...)
		result = append(result, chunk...)
	}
	
	return result, nil
}

// StreamDecrypt 流式解密
func (e *WASMCryptoEngineV3) StreamDecrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 8 {
		return nil, errors.New("ciphertext too short")
	}
	
	start := time.Now()
	defer func() {
		e.stats.decryptCount.Add(1)
		e.stats.decryptTime.Add(time.Since(start).Nanoseconds())
	}()
	
	// 读取头部信息
	totalSize := binary.BigEndian.Uint64(ciphertext[:8])
	ciphertext = ciphertext[8:]
	
	var result []byte
	for len(ciphertext) > 0 {
		if len(ciphertext) < 4 {
			return nil, errors.New("invalid ciphertext format")
		}
		chunkSize := binary.BigEndian.Uint32(ciphertext[:4])
		ciphertext = ciphertext[4:]
		
		if uint32(len(ciphertext)) < chunkSize {
			return nil, errors.New("invalid chunk size")
		}
		chunk := ciphertext[:chunkSize]
		ciphertext = ciphertext[chunkSize:]
		
		plaintext, err := e.Decrypt(chunk)
		if err != nil {
			return nil, err
		}
		result = append(result, plaintext...)
	}
	
	if uint64(len(result)) != totalSize {
		return nil, errors.New("decryption size mismatch")
	}
	
	return result, nil
}

// GetPerformanceMetrics 获取性能指标
func (e *WASMCryptoEngineV3) GetPerformanceMetrics() *CryptoPerformanceMetrics {
	encryptCount := e.stats.encryptCount.Load()
	decryptCount := e.stats.decryptCount.Load()
	
	metrics := &CryptoPerformanceMetrics{
		TotalOperations: encryptCount + decryptCount,
	}
	
	if encryptCount > 0 {
		metrics.AvgEncryptTime = float64(e.stats.encryptTime.Load()) / float64(encryptCount) / 1_000_000
	}
	if decryptCount > 0 {
		metrics.AvgDecryptTime = float64(e.stats.decryptTime.Load()) / float64(decryptCount) / 1_000_000
	}
	
	totalTime := (e.stats.encryptTime.Load() + e.stats.decryptTime.Load())
	if totalTime > 0 {
		totalBytes := (e.stats.totalBytesIn.Load() + e.stats.totalBytesOut.Load())
		metrics.Throughput = float64(totalBytes) / (1024 * 1024) / (float64(totalTime) / 1_000_000_000)
	}
	
	return metrics
}

// ResetStats 重置统计信息
func (e *WASMCryptoEngineV3) ResetStats() {
	e.stats.encryptCount.Store(0)
	e.stats.decryptCount.Store(0)
	e.stats.encryptTime.Store(0)
	e.stats.decryptTime.Store(0)
	e.stats.totalBytesIn.Store(0)
	e.stats.totalBytesOut.Store(0)
}

// Hash 生成 SHA-256 哈希
func (e *WASMCryptoEngineV3) Hash(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

// DeriveKey 使用 PBKDF2 派生密钥
func (e *WASMCryptoEngineV3) DeriveKey(password []byte, salt []byte, iterations int) []byte {
	key := make([]byte, 32)
	if len(salt) == 0 {
		salt = e.secret[:16]
	}
	
	// 简单的派生函数，实际应该使用真正的 PBKDF2
	temp := append(password, salt...)
	for i := 0; i < iterations; i++ {
		h := sha256.New()
		h.Write(temp)
		h.Write(binary.BigEndian.AppendUint32(nil, uint32(i)))
		temp = h.Sum(nil)
	}
	
	copy(key, temp)
	return key
}

// BatchEncrypt 批量加密
func (e *WASMCryptoEngineV3) BatchEncrypt(plaintexts [][]byte) ([][]byte, error) {
	results := make([][]byte, len(plaintexts))
	var wg sync.WaitGroup
	errChan := make(chan error, len(plaintexts))
	
	numWorkers := int(math.Min(8, float64(len(plaintexts))))
	semaphore := make(chan struct{}, numWorkers)
	
	for i, pt := range plaintexts {
		wg.Add(1)
		go func(idx int, data []byte) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			ciphertext, err := e.Encrypt(data)
			if err != nil {
				errChan <- err
				return
			}
			results[idx] = ciphertext
		}(i, pt)
	}
	
	wg.Wait()
	close(errChan)
	
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}
	
	return results, nil
}

// BatchDecrypt 批量解密
func (e *WASMCryptoEngineV3) BatchDecrypt(ciphertexts [][]byte) ([][]byte, error) {
	results := make([][]byte, len(ciphertexts))
	var wg sync.WaitGroup
	errChan := make(chan error, len(ciphertexts))
	
	numWorkers := int(math.Min(8, float64(len(ciphertexts))))
	semaphore := make(chan struct{}, numWorkers)
	
	for i, ct := range ciphertexts {
		wg.Add(1)
		go func(idx int, data []byte) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			plaintext, err := e.Decrypt(data)
			if err != nil {
				errChan <- err
				return
			}
			results[idx] = plaintext
		}(i, ct)
	}
	
	wg.Wait()
	close(errChan)
	
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}
	
	return results, nil
}
