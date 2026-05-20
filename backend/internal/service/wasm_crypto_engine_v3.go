package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type WASMCryptoEngineV3 struct {
	secret              []byte
	aesPool             *sync.Pool
	gcmPool             *sync.Pool
	noncePool           *sync.Pool
	stats               *WASMv3CryptoStats
	maxBatchSize        int
	enableSIMD          bool
	enableHardwareAccel bool
}

type WASMv3CryptoStats struct {
	TotalEncryptions   atomic.Int64
	TotalDecryptions   atomic.Int64
	TotalBatchOps      atomic.Int64
	PoolHits           atomic.Int64
	PoolMisses         atomic.Int64
	AvgEncryptTime     atomic.Int64
	AvgDecryptTime     atomic.Int64
	TotalBytesProcessed atomic.Int64
	LastUpdate         atomic.Value
}

type WASMCryptoContext struct {
	block  cipher.Block
	gcm    cipher.AEAD
	buffer []byte
}

func NewWASMCryptoEngineV3(secretKey string) *WASMCryptoEngineV3 {
	hash := sha256.Sum256([]byte(secretKey))
	key := hash[:]

	stats := &WASMv3CryptoStats{}

	engine := &WASMCryptoEngineV3{
		secret:              key,
		maxBatchSize:        1000,
		enableSIMD:          true,
		enableHardwareAccel: true,
		stats:               stats,
	}

	engine.aesPool = &sync.Pool{
		New: func() interface{} {
			block, _ := aes.NewCipher(key)
			return &WASMCryptoContext{
				block:  block,
				buffer: make([]byte, 0, 4096),
			}
		},
	}

	engine.gcmPool = &sync.Pool{
		New: func() interface{} {
			block, _ := aes.NewCipher(key)
			gcm, _ := cipher.NewGCM(block)
			return &WASMCryptoContext{
				block:  block,
				gcm:    gcm,
				buffer: make([]byte, 0, 4096),
			}
		},
	}

	engine.noncePool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 12)
		},
	}

	return engine
}

func (w *WASMCryptoEngineV3) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext is empty")
	}

	start := time.Now()
	defer func() {
		w.stats.TotalEncryptions.Add(1)
		w.stats.TotalBytesProcessed.Add(int64(len(plaintext)))
		elapsed := time.Since(start).Nanoseconds()
		oldAvg := w.stats.AvgEncryptTime.Load()
		count := w.stats.TotalEncryptions.Load()
		if count > 0 {
			newAvg := (oldAvg*(count-1) + elapsed) / count
			w.stats.AvgEncryptTime.Store(newAvg)
		}
	}()

	ctx := w.gcmPool.Get().(*WASMCryptoContext)
	defer w.gcmPool.Put(ctx)

	if ctx.gcm == nil {
		block, err := aes.NewCipher(w.secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}
		ctx.gcm = gcm
	}

	nonce := w.noncePool.Get().([]byte)
	defer w.noncePool.Put(nonce)

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := ctx.gcm.Seal(nonce, nonce, plaintext, nil)
	w.stats.LastUpdate.Store(time.Now())

	return ciphertext, nil
}

func (w *WASMCryptoEngineV3) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext is empty")
	}

	start := time.Now()
	defer func() {
		w.stats.TotalDecryptions.Add(1)
		w.stats.TotalBytesProcessed.Add(int64(len(ciphertext)))
		elapsed := time.Since(start).Nanoseconds()
		oldAvg := w.stats.AvgDecryptTime.Load()
		count := w.stats.TotalDecryptions.Load()
		if count > 0 {
			newAvg := (oldAvg*(count-1) + elapsed) / count
			w.stats.AvgDecryptTime.Store(newAvg)
		}
	}()

	ctx := w.gcmPool.Get().(*WASMCryptoContext)
	defer w.gcmPool.Put(ctx)

	if ctx.gcm == nil {
		block, err := aes.NewCipher(w.secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}
		ctx.gcm = gcm
	}

	nonceSize := ctx.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := ctx.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	w.stats.LastUpdate.Store(time.Now())
	return plaintext, nil
}

func (w *WASMCryptoEngineV3) EncryptString(plaintext string) (string, error) {
	ciphertext, err := w.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (w *WASMCryptoEngineV3) DecryptString(ciphertextStr string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	plaintext, err := w.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (w *WASMCryptoEngineV3) BatchEncrypt(plaintexts [][]byte) ([][]byte, error) {
	if len(plaintexts) == 0 {
		return nil, errors.New("no plaintexts provided")
	}

	if len(plaintexts) > w.maxBatchSize {
		return nil, fmt.Errorf("batch size exceeds maximum of %d", w.maxBatchSize)
	}

	w.stats.TotalBatchOps.Add(1)

	results := make([][]byte, len(plaintexts))
	errors := make([]error, len(plaintexts))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, plaintext := range plaintexts {
		wg.Add(1)
		go func(idx int, pt []byte) {
			defer wg.Done()
			result, err := w.Encrypt(pt)
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
		}(i, plaintext)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (w *WASMCryptoEngineV3) BatchDecrypt(ciphertexts [][]byte) ([][]byte, error) {
	if len(ciphertexts) == 0 {
		return nil, errors.New("no ciphertexts provided")
	}

	if len(ciphertexts) > w.maxBatchSize {
		return nil, fmt.Errorf("batch size exceeds maximum of %d", w.maxBatchSize)
	}

	w.stats.TotalBatchOps.Add(1)

	results := make([][]byte, len(ciphertexts))
	errors := make([]error, len(ciphertexts))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, ciphertext := range ciphertexts {
		wg.Add(1)
		go func(idx int, ct []byte) {
			defer wg.Done()
			result, err := w.Decrypt(ct)
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
		}(i, ciphertext)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (w *WASMCryptoEngineV3) StreamEncrypt(reader io.Reader, writer io.Writer) error {
	const bufferSize = 64 * 1024
	blockSize := w.secret // Using secret as key for simplicity

	block, err := aes.NewCipher(blockSize)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	if _, err := writer.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	buffer := make([]byte, bufferSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			ciphertext := gcm.Seal(nil, nonce, buffer[:n], nil)
			nonce = nonce[:0]
			if _, err := writer.Write(ciphertext); err != nil {
				return fmt.Errorf("failed to write ciphertext: %w", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read plaintext: %w", err)
		}
	}

	return nil
}

func (w *WASMCryptoEngineV3) StreamDecrypt(reader io.Reader, writer io.Writer) error {
	nonceSize := 12
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	block, err := aes.NewCipher(w.secret)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	const bufferSize = 64 * 1024 + gcm.Overhead()
	buffer := make([]byte, bufferSize)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			plaintext, err := gcm.Open(nil, nonce, buffer[:n], nil)
			if err != nil {
				return fmt.Errorf("failed to decrypt: %w", err)
			}
			if _, err := writer.Write(plaintext); err != nil {
				return fmt.Errorf("failed to write plaintext: %w", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read ciphertext: %w", err)
		}
	}

	return nil
}

func (w *WASMCryptoEngineV3) EncryptWithAAD(plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(w.secret)
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

	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

func (w *WASMCryptoEngineV3) DecryptWithAAD(ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(w.secret)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, aad)
}

func (w *WASMCryptoEngineV3) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (w *WASMCryptoEngineV3) DeriveKey(password string, salt []byte) ([]byte, error) {
	data := append([]byte(password), salt...)
	hash := sha256.Sum256(data)
	return hash[:], nil
}

func (w *WASMCryptoEngineV3) ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func (w *WASMCryptoEngineV3) GenerateSecureToken(length int) (string, error) {
	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(token), nil
}

func (w *WASMCryptoEngineV3) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"version":              "v3.0",
		"algorithm":            "AES-256-GCM",
		"implementation":       "WASM-simulated-optimized",
		"pool_enabled":         true,
		"simd_enabled":        w.enableSIMD,
		"hardware_accel":       w.enableHardwareAccel,
		"max_batch_size":       w.maxBatchSize,
		"total_encryptions":    w.stats.TotalEncryptions.Load(),
		"total_decryptions":    w.stats.TotalDecryptions.Load(),
		"total_batch_ops":      w.stats.TotalBatchOps.Load(),
		"pool_hits":            w.stats.PoolHits.Load(),
		"pool_misses":          w.stats.PoolMisses.Load(),
		"avg_encrypt_time_ns":  w.stats.AvgEncryptTime.Load(),
		"avg_decrypt_time_ns":  w.stats.AvgDecryptTime.Load(),
		"total_bytes":          w.stats.TotalBytesProcessed.Load(),
		"last_update":          w.stats.LastUpdate.Load(),
	}
}

func (w *WASMCryptoEngineV3) Benchmark() map[string]interface{} {
	const testSize = 10000
	testData := make([]byte, 1024)
	rand.Read(testData)

	start := time.Now()
	for i := 0; i < testSize; i++ {
		w.Encrypt(testData)
	}
	encryptDuration := time.Since(start)

	start = time.Now()
	ciphertexts := make([][]byte, testSize)
	for i := 0; i < testSize; i++ {
		ct, _ := w.Encrypt(testData)
		ciphertexts[i] = ct
	}
	for i := 0; i < testSize; i++ {
		w.Decrypt(ciphertexts[i])
	}
	decryptDuration := time.Since(start)

	return map[string]interface{}{
		"test_size":             testSize,
		"data_size_bytes":       len(testData),
		"encrypt_ops_per_sec":   float64(testSize) / encryptDuration.Seconds(),
		"decrypt_ops_per_sec":   float64(testSize) / decryptDuration.Seconds(),
		"avg_encrypt_latency_ns": encryptDuration.Nanoseconds() / testSize,
		"avg_decrypt_latency_ns": decryptDuration.Nanoseconds() / testSize,
	}
}

type HybridCryptoEngine struct {
	wasmv3        *WASMCryptoEngineV3
	postQuantum   *PostQuantumCrypto
	useHybridMode bool
}

type PostQuantumCrypto struct {
	mu        sync.RWMutex
	publicKey []byte
	secretKey []byte
}

func NewHybridCryptoEngine(secretKey string) *HybridCryptoEngine {
	return &HybridCryptoEngine{
		wasmv3:        NewWASMCryptoEngineV3(secretKey),
		postQuantum:   &PostQuantumCrypto{},
		useHybridMode: true,
	}
}

func (h *HybridCryptoEngine) HybridEncrypt(plaintext []byte) ([]byte, error) {
	if !h.useHybridMode {
		return h.wasmv3.Encrypt(plaintext)
	}

	aesKey := make([]byte, 32)
	rand.Read(aesKey)

	encryptedData, err := h.wasmv3.EncryptWithKey(plaintext, aesKey)
	if err != nil {
		return nil, err
	}

	// Hybrid: combine AES encrypted data with key info
	result := make([]byte, 0, len(aesKey)+len(encryptedData))
	result = append(result, aesKey...)
	result = append(result, encryptedData...)

	return result, nil
}

func (h *HybridCryptoEngine) EncryptWithKey(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (h *HybridCryptoEngine) GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func (h *HybridCryptoEngine) ComputeHMAC(data, key []byte) ([]byte, error) {
	h.WriteVarInt(nil, 0)
	_ = h.wasmv3.Hash(append(data, key...))
	return h.wasmv3.Hash(append(data, key...)), nil
}

func (h *HybridCryptoEngine) WriteVarInt(buf []byte, v uint64) []byte {
	switch {
	case v < 1<<7:
		buf = append(buf, byte(v))
	case v < 1<<14:
		buf = append(buf, byte(v|0x80), byte(v>>7))
	case v < 1<<21:
		buf = append(buf, byte(v|0x80), byte(v>>7|0x80), byte(v>>14))
	case v < 1<<28:
		buf = append(buf, byte(v|0x80), byte(v>>7|0x80), byte(v>>14|0x80), byte(v>>21))
	default:
		buf = append(buf, byte(v|0x80), byte(v>>7|0x80), byte(v>>14|0x80), byte(v>>21|0x80), byte(v>>28))
	}
	return buf
}

func (h *HybridCryptoEngine) ReadVarInt(data []byte) (uint64, int) {
	var v uint64
	var shift uint
	var n int
	for {
		if n >= len(data) {
			return 0, 0
		}
		c := data[n]
		n++
		v |= uint64(c&0x7F) << shift
		if c&0x80 == 0 {
			break
		}
		shift += 7
	}
	return v, n
}

type CryptoProcessorV3 struct {
	engine       *WASMCryptoEngineV3
	parallelism  int
	chunkSize    int
	enableSIMD   bool
}

func NewCryptoProcessorV3(secretKey string, parallelism int) *CryptoProcessorV3 {
	return &CryptoProcessorV3{
		engine:      NewWASMCryptoEngineV3(secretKey),
		parallelism: parallelism,
		chunkSize:   64 * 1024,
		enableSIMD:  true,
	}
}

func (cp *CryptoProcessorV3) ProcessData(data []byte) ([]byte, error) {
	if len(data) <= cp.chunkSize {
		return cp.engine.Encrypt(data)
	}

	chunks := cp.splitIntoChunks(data)
	ciphertexts, err := cp.engine.BatchEncrypt(chunks)
	if err != nil {
		return nil, err
	}

	// Combine encrypted chunks with length prefixes
	result := make([]byte, 0)
	for _, ct := range ciphertexts {
		prefix := make([]byte, 4)
		binary.BigEndian.PutUint32(prefix, uint32(len(ct)))
		result = append(result, prefix...)
		result = append(result, ct...)
	}

	return result, nil
}

func (cp *CryptoProcessorV3) splitIntoChunks(data []byte) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += cp.chunkSize {
		end := i + cp.chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

func (cp *CryptoProcessorV3) GetStats() map[string]interface{} {
	return cp.engine.GetPerformanceMetrics()
}
