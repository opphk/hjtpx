package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type IntegrityChecker struct {
	whitelist    map[string]string
	cache        *IntegrityCache
	monitorMutex sync.RWMutex
}

type IntegrityCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
}

type CacheEntry struct {
	Hash      string
	Timestamp time.Time
	TTL       time.Duration
}

func NewIntegrityChecker() *IntegrityChecker {
	return &IntegrityChecker{
		whitelist: make(map[string]string),
		cache: &IntegrityCache{
			entries: make(map[string]*CacheEntry),
		},
	}
}

func (ic *IntegrityChecker) CalculateFileHash(filePath string) (string, error) {
	if cached := ic.getFromCache(filePath); cached != "" {
		return cached, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()

	buffer := make([]byte, 32*1024)
	for {
		bytesRead, err := file.Read(buffer)
		if bytesRead > 0 {
			hasher.Write(buffer[:bytesRead])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	ic.putToCache(filePath, hash)

	return hash, nil
}

func (ic *IntegrityChecker) CalculateCodeHash(code string) string {
	hasher := sha256.New()
	hasher.Write([]byte(code))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (ic *IntegrityChecker) VerifyFileIntegrity(filePath, expectedHash string) (bool, error) {
	actualHash, err := ic.CalculateFileHash(filePath)
	if err != nil {
		return false, err
	}

	return actualHash == expectedHash, nil
}

func (ic *IntegrityChecker) AddToWhitelist(filePath, hash string) {
	ic.monitorMutex.Lock()
	defer ic.monitorMutex.Unlock()
	ic.whitelist[filePath] = hash
}

func (ic *IntegrityChecker) RemoveFromWhitelist(filePath string) {
	ic.monitorMutex.Lock()
	defer ic.monitorMutex.Unlock()
	delete(ic.whitelist, filePath)
}

func (ic *IntegrityChecker) IsInWhitelist(filePath, hash string) bool {
	ic.monitorMutex.RLock()
	defer ic.monitorMutex.RUnlock()

	if expectedHash, exists := ic.whitelist[filePath]; exists {
		return expectedHash == hash
	}
	return false
}

func (ic *IntegrityChecker) DetectTampering(filePath string) (*TamperingReport, error) {
	report := &TamperingReport{
		FilePath:   filePath,
		Timestamp: time.Now(),
	}

	currentHash, err := ic.CalculateFileHash(filePath)
	if err != nil {
		report.Error = err.Error()
		return report, err
	}
	report.CurrentHash = currentHash

	ic.monitorMutex.RLock()
	if expectedHash, exists := ic.whitelist[filePath]; exists {
		report.IsTampered = currentHash != expectedHash
		report.ExpectedHash = expectedHash
	} else {
		report.IsTampered = false
		report.InWhitelist = false
	}
	ic.monitorMutex.RUnlock()

	return report, nil
}

func (ic *IntegrityChecker) MonitorDirectory(dirPath string, extensions []string) ([]*TamperingReport, error) {
	reports := []*TamperingReport{}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if len(extensions) > 0 {
			ext := getFileExtension(entry.Name())
			if !contains(ext, extensions) {
				continue
			}
		}

		filePath := fmt.Sprintf("%s/%s", dirPath, entry.Name())
		report, err := ic.DetectTampering(filePath)
		if err != nil {
			continue
		}

		if report.IsTampered {
			reports = append(reports, report)
		}
	}

	return reports, nil
}

func (ic *IntegrityChecker) GenerateIntegrityReport(filePaths []string) *IntegrityReport {
	report := &IntegrityReport{
		Timestamp:      time.Now(),
		TotalFiles:     len(filePaths),
		ValidFiles:     0,
		TamperedFiles:  0,
		ErrorFiles:     0,
		FileDetails:    make([]*FileIntegrityStatus, 0, len(filePaths)),
	}

	for _, filePath := range filePaths {
		status := &FileIntegrityStatus{
			FilePath: filePath,
		}

		hash, err := ic.CalculateFileHash(filePath)
		if err != nil {
			status.HasError = true
			status.Error = err.Error()
			report.ErrorFiles++
			report.FileDetails = append(report.FileDetails, status)
			continue
		}
		status.CurrentHash = hash

		if ic.IsInWhitelist(filePath, hash) {
			status.IsValid = true
			status.InWhitelist = true
			report.ValidFiles++
		} else {
			status.IsValid = false
			status.InWhitelist = false
			report.TamperedFiles++
		}

		report.FileDetails = append(report.FileDetails, status)
	}

	return report
}

func (ic *IntegrityChecker) CreateIntegrityManifest(dirPath string, extensions []string) (*IntegrityManifest, error) {
	manifest := &IntegrityManifest{
		CreatedAt: time.Now(),
		Version:   "1.0",
		Files:     make(map[string]string),
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if len(extensions) > 0 {
			ext := getFileExtension(entry.Name())
			if !contains(ext, extensions) {
				continue
			}
		}

		filePath := fmt.Sprintf("%s/%s", dirPath, entry.Name())
		hash, err := ic.CalculateFileHash(filePath)
		if err != nil {
			continue
		}

		manifest.Files[filePath] = hash
		manifest.FileCount++

		ic.AddToWhitelist(filePath, hash)
	}

	manifestStr := ic.serializeManifest(manifest)
	manifest.ManifestHash = ic.CalculateCodeHash(manifestStr)

	return manifest, nil
}

func (ic *IntegrityChecker) VerifyIntegrityManifest(manifest *IntegrityManifest) *IntegrityReport {
	return ic.GenerateIntegrityReport(func() []string {
		paths := make([]string, 0, len(manifest.Files))
		for path := range manifest.Files {
			paths = append(paths, path)
		}
		return paths
	}())
}

func (ic *IntegrityChecker) serializeManifest(manifest *IntegrityManifest) string {
	result := fmt.Sprintf("version=%s,count=%d,created=%s",
		manifest.Version, manifest.FileCount, manifest.CreatedAt.Format(time.RFC3339))

	for path, hash := range manifest.Files {
		result += fmt.Sprintf(",%s=%s", path, hash)
	}

	return result
}

func (ic *IntegrityChecker) getFromCache(key string) string {
	ic.cache.mutex.RLock()
	defer ic.cache.mutex.RUnlock()

	if entry, exists := ic.cache.entries[key]; exists {
		if time.Since(entry.Timestamp) < entry.TTL {
			return entry.Hash
		}
	}
	return ""
}

func (ic *IntegrityChecker) putToCache(key, hash string) {
	ic.cache.mutex.Lock()
	defer ic.cache.mutex.Unlock()

	ic.cache.entries[key] = &CacheEntry{
		Hash:      hash,
		Timestamp: time.Now(),
		TTL:       5 * time.Minute,
	}
}

func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + parts[len(parts)-1]
}

func contains(item string, list []string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

type TamperingReport struct {
	FilePath     string
	CurrentHash  string
	ExpectedHash string
	IsTampered   bool
	InWhitelist  bool
	Error        string
	Timestamp    time.Time
}

type IntegrityReport struct {
	Timestamp     time.Time
	TotalFiles    int
	ValidFiles    int
	TamperedFiles int
	ErrorFiles    int
	FileDetails   []*FileIntegrityStatus
}

type FileIntegrityStatus struct {
	FilePath      string
	CurrentHash   string
	IsValid       bool
	InWhitelist   bool
	HasError      bool
	Error         string
}

type IntegrityManifest struct {
	Version      string
	CreatedAt    time.Time
	ManifestHash string
	FileCount    int
	Files        map[string]string
}

func ParseIntegrityManifest(manifestStr string) (*IntegrityManifest, error) {
	manifest := &IntegrityManifest{
		Files: make(map[string]string),
	}

	return manifest, nil
}

func ExportManifestAsJSON(manifest *IntegrityManifest) (string, error) {
	result := fmt.Sprintf(`{
		"version": "%s",
		"created_at": "%s",
		"manifest_hash": "%s",
		"file_count": %d,
		"files": {`,
		manifest.Version,
		manifest.CreatedAt.Format(time.RFC3339),
		manifest.ManifestHash,
		manifest.FileCount)

	first := true
	for path, hash := range manifest.Files {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s": "%s"`, path, hash)
		first = false
	}

	result += "}}"

	return result, nil
}
