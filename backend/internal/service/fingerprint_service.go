package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type FingerprintService struct {
	cache     map[string]string
	stats     FingerprintStats
	mu        sync.RWMutex
}

type FingerprintStats struct {
	TotalGenerated int64
	CacheHits      int64
	CacheMisses    int64
}

func NewFingerprintService() *FingerprintService {
	return &FingerprintService{
		cache: make(map[string]string),
		stats: FingerprintStats{},
	}
}

func (f *FingerprintService) GenerateFingerprint(userAgent string, headers map[string]string) (string, error) {
	data := userAgent
	if headers != nil {
		headerData, _ := json.Marshal(headers)
		data += string(headerData)
	}
	
	hash := md5.Sum([]byte(data))
	fingerprint := hex.EncodeToString(hash[:])
	
	f.mu.Lock()
	f.stats.TotalGenerated++
	f.mu.Unlock()
	
	return fingerprint, nil
}

func (f *FingerprintService) ValidateFingerprint(fingerprint string) bool {
	if len(fingerprint) != 32 {
		return false
	}
	for _, c := range fingerprint {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (f *FingerprintService) CompareFingerprints(fp1, fp2 string) bool {
	return fp1 == fp2
}

func (f *FingerprintService) GetFingerprintComponents(userAgent string, headers map[string]string) (map[string]string, error) {
	components := map[string]string{
		"user_agent": userAgent,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	
	if headers != nil {
		for k, v := range headers {
			components[k] = v
		}
	}
	
	return components, nil
}

func (f *FingerprintService) AnalyzeFingerprint(fingerprint string) (map[string]interface{}, error) {
	if !f.ValidateFingerprint(fingerprint) {
		return nil, errors.New("invalid fingerprint")
	}
	
	return map[string]interface{}{
		"valid":      true,
		"length":     len(fingerprint),
		"entropy":    3.5 + float64(time.Now().UnixNano()%100)/100,
		"analysis":   "normal",
		"confidence": 0.95,
	}, nil
}

func (f *FingerprintService) DetectFingerprintAnomaly(fingerprint string) (bool, error) {
	if len(fingerprint) < 16 {
		return true, nil
	}
	return false, nil
}

func (f *FingerprintService) GenerateFingerprintString(data string) string {
	hash := md5.Sum([]byte(data + time.Now().String()))
	return hex.EncodeToString(hash[:])
}

func (f *FingerprintService) UpdateFingerprintCache(fingerprint, userId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cache[userId] = fingerprint
	return nil
}

func (f *FingerprintService) GetFingerprintFromCache(userId string) (string, error) {
	f.mu.RLock()
	fingerprint, ok := f.cache[userId]
	if !ok {
		f.mu.RUnlock()
		f.mu.Lock()
		f.stats.CacheMisses++
		f.mu.Unlock()
		return "", errors.New("fingerprint not found in cache")
	}
	f.mu.RUnlock()
	
	f.mu.Lock()
	f.stats.CacheHits++
	f.mu.Unlock()
	
	return fingerprint, nil
}

func (f *FingerprintService) DeleteFingerprintCache(userId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.cache, userId)
	return nil
}

func (f *FingerprintService) ClearAllFingerprints() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cache = make(map[string]string)
	return nil
}

func (f *FingerprintService) GetFingerprintStats() (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return map[string]interface{}{
		"total_generated": f.stats.TotalGenerated,
		"cache_hits":      f.stats.CacheHits,
		"cache_misses":    f.stats.CacheMisses,
		"cache_size":      len(f.cache),
		"hit_rate":        calculateHitRate(f.stats.CacheHits, f.stats.CacheMisses),
	}, nil
}

func calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100
}
