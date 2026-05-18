package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
	"sync"
	"time"
)

type HashAlgorithm string

const (
	HashSHA256 HashAlgorithm = "sha256"
	HashSHA512 HashAlgorithm = "sha512"
	HashHMAC256 HashAlgorithm = "hmac-sha256"
)

type IntegrityService struct {
	algorithm HashAlgorithm
	secretKey []byte
	cache     map[string]string
	mu        sync.RWMutex
}

func NewIntegrityService() *IntegrityService {
	return &IntegrityService{
		algorithm: HashSHA256,
		secretKey:  []byte("hjtpx-integrity-key-2024"),
		cache:      make(map[string]string),
	}
}

func (is *IntegrityService) SetAlgorithm(algorithm HashAlgorithm) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.algorithm = algorithm
}

func (is *IntegrityService) SetSecretKey(key []byte) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.secretKey = key
}

func (is *IntegrityService) CalculateHash(data string) string {
	is.mu.RLock()
	algorithm := is.algorithm
	secretKey := make([]byte, len(is.secretKey))
	copy(secretKey, is.secretKey)
	is.mu.RUnlock()

	var h hash.Hash
	
	switch algorithm {
	case HashSHA256:
		h = sha256.New()
		h.Write([]byte(data))
	case HashSHA512:
		h = sha512.New()
		h.Write([]byte(data))
	case HashHMAC256:
		h = hmac.New(sha256.New, secretKey)
		h.Write([]byte(data))
	default:
		h = sha256.New()
		h.Write([]byte(data))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (is *IntegrityService) CalculateHashB64(data string) string {
	hash := is.CalculateHash(data)
	bytes, _ := hex.DecodeString(hash)
	return base64.StdEncoding.EncodeToString(bytes)
}

func (is *IntegrityService) VerifyHash(data, expectedHash string) bool {
	actualHash := is.CalculateHash(data)
	return actualHash == expectedHash
}

func (is *IntegrityService) VerifyHashB64(data, expectedB64 string) bool {
	actualB64 := is.CalculateHashB64(data)
	return actualB64 == expectedB64
}

func (is *IntegrityService) CreateChecksum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (is *IntegrityService) VerifyChecksum(data []byte, checksum string) bool {
	actualChecksum := is.CreateChecksum(data)
	return actualChecksum == checksum
}

func (is *IntegrityService) GenerateIntegrityToken(data string, ttl time.Duration) (string, error) {
	timestamp := time.Now().Add(ttl).Unix()
	
	tokenData := fmt.Sprintf("%s:%d", data, timestamp)
	hash := is.CalculateHash(tokenData)
	
	token := fmt.Sprintf("%s:%d", hash, timestamp)
	return base64.URLEncoding.EncodeToString([]byte(token)), nil
}

func (is *IntegrityService) VerifyIntegrityToken(token string) (bool, error) {
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false, fmt.Errorf("failed to decode token: %w", err)
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid token format")
	}

	timestampStr := parts[1]
	var timestamp int64
	fmt.Sscanf(timestampStr, "%d", &timestamp)

	if time.Now().Unix() > timestamp {
		return false, fmt.Errorf("token expired")
	}

	return true, nil
}

func (is *IntegrityService) CacheHash(data string) string {
	is.mu.Lock()
	defer is.mu.Unlock()

	if hash, exists := is.cache[data]; exists {
		return hash
	}

	is.mu.Unlock()
	hash := is.calculateHashInternal(data)
	is.mu.Lock()

	is.cache[data] = hash
	
	if len(is.cache) > 1000 {
		is.cleanupCache()
	}

	return hash
}

func (is *IntegrityService) calculateHashInternal(data string) string {
	var h hash.Hash
	h = sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (is *IntegrityService) cleanupCache() {
	count := len(is.cache) / 4
	keys := make([]string, 0, count)
	for k := range is.cache {
		keys = append(keys, k)
		if len(keys) >= count {
			break
		}
	}
	
	for _, k := range keys {
		delete(is.cache, k)
	}
}

func (is *IntegrityService) ClearCache() {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.cache = make(map[string]string)
}

func (is *IntegrityService) GetCacheSize() int {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return len(is.cache)
}

func (is *IntegrityService) CreateMultipleHashes(data string) map[string]string {
	hashes := make(map[string]string)

	is.mu.RLock()
	secretKey := make([]byte, len(is.secretKey))
	copy(secretKey, is.secretKey)
	is.mu.RUnlock()

	h256 := sha256.New()
	h256.Write([]byte(data))
	hashes["sha256"] = hex.EncodeToString(h256.Sum(nil))

	h512 := sha512.New()
	h512.Write([]byte(data))
	hashes["sha512"] = hex.EncodeToString(h512.Sum(nil))

	hHmac := hmac.New(sha256.New, secretKey)
	hHmac.Write([]byte(data))
	hashes["hmac-sha256"] = hex.EncodeToString(hHmac.Sum(nil))

	return hashes
}

func (is *IntegrityService) VerifyMultipleHashes(data string, hashes map[string]string) bool {
	calculated := is.CreateMultipleHashes(data)
	
	for algo, expectedHash := range hashes {
		if actualHash, exists := calculated[algo]; exists {
			if actualHash != expectedHash {
				return false
			}
		}
	}
	
	return true
}

func (is *IntegrityService) GenerateMerkleRoot(items []string) string {
	if len(items) == 0 {
		return ""
	}

	if len(items) == 1 {
		return is.CalculateHash(items[0])
	}

	var pairs []string
	for i := 0; i < len(items); i += 2 {
		if i+1 < len(items) {
			pairHash := is.CalculateHash(items[i] + items[i+1])
			pairs = append(pairs, pairHash)
		} else {
			pairs = append(pairs, is.CalculateHash(items[i]))
		}
	}

	return is.GenerateMerkleRoot(pairs)
}

func (is *IntegrityService) CreateIntegrityReport(data string) map[string]interface{} {
	report := make(map[string]interface{})

	report["hash"] = is.CalculateHash(data)
	report["hash_b64"] = is.CalculateHashB64(data)
	report["algorithm"] = string(is.algorithm)
	report["length"] = len(data)
	report["timestamp"] = time.Now().Unix()
	
	hashes := is.CreateMultipleHashes(data)
	report["hashes"] = hashes

	return report
}

type IntegrityChecker struct {
	service *IntegrityService
	history []IntegrityRecord
	mu      sync.RWMutex
}

type IntegrityRecord struct {
	Data      string    `json:"data"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

func NewIntegrityChecker() *IntegrityChecker {
	return &IntegrityChecker{
		service: NewIntegrityService(),
		history: make([]IntegrityRecord, 0),
	}
}

func (ic *IntegrityChecker) Check(data string) (bool, string) {
	hash := ic.service.CalculateHash(data)
	valid := ic.service.VerifyHash(data, hash)

	ic.mu.Lock()
	ic.history = append(ic.history, IntegrityRecord{
		Data:      data,
		Hash:      hash,
		Timestamp: time.Now(),
		Status:    func() string { if valid { return "valid" } else { return "invalid" } }(),
	})
	ic.mu.Unlock()

	return valid, hash
}

func (ic *IntegrityChecker) GetHistory() []IntegrityRecord {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return append([]IntegrityRecord{}, ic.history...)
}

func (ic *IntegrityChecker) ClearHistory() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.history = make([]IntegrityRecord, 0)
}

func (ic *IntegrityChecker) GetStatistics() map[string]interface{} {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_checks"] = len(ic.history)
	stats["cache_size"] = ic.service.GetCacheSize()

	validCount := 0
	invalidCount := 0
	for _, record := range ic.history {
		if record.Status == "valid" {
			validCount++
		} else {
			invalidCount++
		}
	}
	stats["valid_checks"] = validCount
	stats["invalid_checks"] = invalidCount

	return stats
}
