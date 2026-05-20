package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type TrajectoryEncryptionService struct {
	mu        sync.RWMutex
	key       []byte
	salt      int64
	algorithm string
	version   string
}

type EncryptedTrajectoryData struct {
	Version      string                 `json:"v"`
	Timestamp    int64                  `json:"ts"`
	Encrypted    string                 `json:"enc"`
	Checksum     string                 `json:"cs"`
	Compressed   []byte                 `json:"cp,omitempty"`
	Features     *TrajectoryFeatures    `json:"ft,omitempty"`
	DeviceInfo   *DeviceInfo           `json:"di,omitempty"`
	Summary      *TrajectorySummary     `json:"sm,omitempty"`
}

type TrajectoryFeatures struct {
	TotalDistance     float64 `json:"td"`
	TotalDuration     int64   `json:"tdr"`
	AvgVelocity       float64 `json:"av"`
	MaxVelocity       float64 `json:"mv"`
	VelocityVariance  float64 `json:"vv"`
	PathEfficiency   float64 `json:"pe"`
	DirectionChanges  int     `json:"dc"`
	MicroCorrections  int     `json:"mc"`
	BacktrackCount    int     `json:"bc"`
	BacktrackDistance float64 `json:"bd"`
	PauseCount        int     `json:"pc"`
	TotalPauseDuration float64 `json:"pd"`
	Smoothness        float64 `json:"sm"`
	Jitter            float64 `json:"jt"`
	Entropy           float64 `json:"en"`
	HumanLikeness     float64 `json:"hl"`
}

type DeviceInfo struct {
	UserAgent   string `json:"ua"`
	Platform    string `json:"pl"`
	ScreenWidth int    `json:"sw"`
	ScreenHeight int   `json:"sh"`
	TouchSupport bool   `json:"ts"`
	PixelRatio  float64 `json:"pr"`
	Language    string `json:"la"`
	Timezone    string `json:"tz"`
}

type TrajectorySummary struct {
	PointCount int     `json:"pc"`
	Duration   int64   `json:"dr"`
	Distance   float64 `json:"ds"`
	IsValid    bool    `json:"iv"`
}

func NewTrajectoryEncryptionService() *TrajectoryEncryptionService {
	return &TrajectoryEncryptionService{
		key:       generateDefaultKey(),
		salt:      time.Now().Unix(),
		algorithm: "AES-256-GCM",
		version:   "3.0",
	}
}

func generateDefaultKey() []byte {
	seed := fmt.Sprintf("hjtpx-slider-trajectory-%d", time.Now().Unix()/3600)
	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}

func (s *TrajectoryEncryptionService) SetKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(key) < 32 {
		return fmt.Errorf("key must be at least 32 characters")
	}

	hash := sha256.Sum256([]byte(key))
	s.key = hash[:]
	return nil
}

func (s *TrajectoryEncryptionService) SetVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = version
}

func (s *TrajectoryEncryptionService) GetVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *TrajectoryEncryptionService) Encrypt(data []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.key) != 32 {
		return "", fmt.Errorf("invalid key size: expected 32 bytes")
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *TrajectoryEncryptionService) EncryptWithAAD(data []byte, additionalData []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.key) != 32 {
		return "", fmt.Errorf("invalid key size: expected 32 bytes")
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, additionalData)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *TrajectoryEncryptionService) Decrypt(encrypted string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *TrajectoryEncryptionService) DecryptWithAAD(encrypted string, additionalData []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *TrajectoryEncryptionService) EncryptTrajectory(data *TrajectoryFeatures) (*EncryptedTrajectoryData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trajectory data: %w", err)
	}

	encrypted, err := s.Encrypt(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt trajectory: %w", err)
	}

	checksum := s.CalculateChecksum(jsonData)

	return &EncryptedTrajectoryData{
		Version:   "2.0",
		Timestamp: time.Now().Unix(),
		Encrypted: encrypted,
		Checksum:  checksum,
		Features:  data,
	}, nil
}

func (s *TrajectoryEncryptionService) DecryptTrajectory(encrypted *EncryptedTrajectoryData) (*TrajectoryFeatures, error) {
	if encrypted.Version != "2.0" {
		return nil, fmt.Errorf("unsupported version: %s", encrypted.Version)
	}

	plaintext, err := s.Decrypt(encrypted.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt trajectory: %w", err)
	}

	checksum := s.CalculateChecksum(plaintext)
	if checksum != encrypted.Checksum {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", encrypted.Checksum, checksum)
	}

	var features TrajectoryFeatures
	if err := json.Unmarshal(plaintext, &features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trajectory features: %w", err)
	}

	return &features, nil
}

func (s *TrajectoryEncryptionService) CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:8])
}

func (s *TrajectoryEncryptionService) DecryptFromString(encryptedStr string) (*TrajectoryFeatures, error) {
	var encrypted EncryptedTrajectoryData
	if err := json.Unmarshal([]byte(encryptedStr), &encrypted); err != nil {
		if strings.Contains(encryptedStr, "encrypted") || strings.Contains(encryptedStr, "enc") {
			var legacyData map[string]interface{}
			if err := json.Unmarshal([]byte(encryptedStr), &legacyData); err == nil {
				if enc, ok := legacyData["encrypted"].(string); ok {
					encrypted.Encrypted = enc
					encrypted.Version = "2.0"
				}
			}
		} else {
			return nil, fmt.Errorf("failed to parse encrypted data: %w", err)
		}
	}

	if encrypted.Version == "" {
		encrypted.Version = "2.0"
	}

	return s.DecryptTrajectory(&encrypted)
}

func (s *TrajectoryEncryptionService) ValidateChecksum(data []byte, checksum string) bool {
	calculated := s.CalculateChecksum(data)
	return calculated == checksum
}

type TrajectoryIntegrityValidator struct {
	mu          sync.RWMutex
	maxAge      time.Duration
	minPoints   int
	maxPoints   int
	minDuration time.Duration
	maxDuration time.Duration
	minDistance float64
	maxDistance float64
}

func NewTrajectoryIntegrityValidator() *TrajectoryIntegrityValidator {
	return &TrajectoryIntegrityValidator{
		maxAge:      5 * time.Minute,
		minPoints:   10,
		maxPoints:   1000,
		minDuration: 300 * time.Millisecond,
		maxDuration: 30 * time.Second,
		minDistance: 50,
		maxDistance: 5000,
	}
}

func (v *TrajectoryIntegrityValidator) Validate(encrypted *EncryptedTrajectoryData) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if time.Now().Unix()-encrypted.Timestamp > int64(v.maxAge.Seconds()) {
		return fmt.Errorf("trajectory data expired")
	}

	if encrypted.Features == nil {
		return fmt.Errorf("missing trajectory features")
	}

	features := encrypted.Features

	if features.TotalDistance < v.minDistance || features.TotalDistance > v.maxDistance {
		return fmt.Errorf("invalid distance: %.2f", features.TotalDistance)
	}

	return nil
}

func (v *TrajectoryIntegrityValidator) SetConstraints(constraints map[string]interface{}) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if minPoints, ok := constraints["min_points"].(int); ok {
		v.minPoints = minPoints
	}

	if maxPoints, ok := constraints["max_points"].(int); ok {
		v.maxPoints = maxPoints
	}

	if minDuration, ok := constraints["min_duration"].(int64); ok {
		v.minDuration = time.Duration(minDuration) * time.Millisecond
	}

	if maxDuration, ok := constraints["max_duration"].(int64); ok {
		v.maxDuration = time.Duration(maxDuration) * time.Millisecond
	}

	if minDistance, ok := constraints["min_distance"].(float64); ok {
		v.minDistance = minDistance
	}

	if maxDistance, ok := constraints["max_distance"].(float64); ok {
		v.maxDistance = maxDistance
	}

	return nil
}

type SecureTrajectoryProcessor struct {
	encryptionService *TrajectoryEncryptionService
	validator         *TrajectoryIntegrityValidator
	analyzer          *SliderAnalysisV2
}

func NewSecureTrajectoryProcessor() *SecureTrajectoryProcessor {
	return &SecureTrajectoryProcessor{
		encryptionService: NewTrajectoryEncryptionService(),
		validator:         NewTrajectoryIntegrityValidator(),
		analyzer:          NewSliderAnalysisV2(),
	}
}

func (p *SecureTrajectoryProcessor) ProcessEncryptedTrajectory(encryptedStr string) (*TrajectoryFeatures, error) {
	encrypted, err := p.encryptionService.DecryptFromString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt trajectory: %w", err)
	}

	if err := p.validator.Validate(&EncryptedTrajectoryData{
		Timestamp: time.Now().Unix(),
		Features:  encrypted,
	}); err != nil {
		return nil, fmt.Errorf("trajectory validation failed: %w", err)
	}

	return encrypted, nil
}

func (p *SecureTrajectoryProcessor) CreateEncryptedTrajectory(features *TrajectoryFeatures) (*EncryptedTrajectoryData, error) {
	return p.encryptionService.EncryptTrajectory(features)
}

func (p *SecureTrajectoryProcessor) AnalyzeTrajectory(trajectory []SliderTrajectoryPoint) (*ExtendedAnalysisResult, error) {
	return p.analyzer.PerformExtendedAnalysis(trajectory), nil
}

func (p *SecureTrajectoryProcessor) ValidateIntegrity(data []byte, checksum string) bool {
	return p.encryptionService.ValidateChecksum(data, checksum)
}

func (p *SecureTrajectoryProcessor) GenerateChecksum(data []byte) string {
	return p.encryptionService.CalculateChecksum(data)
}

type AdvancedEncryptionService struct {
	mu              sync.RWMutex
	masterKey      []byte
	sessionKeys    map[string][]byte
	keyRotationTime time.Time
}

func NewAdvancedEncryptionService() *AdvancedEncryptionService {
	svc := &AdvancedEncryptionService{
		sessionKeys:    make(map[string][]byte),
		keyRotationTime: time.Now(),
	}
	svc.masterKey = generateDefaultKey()
	return svc
}

func (s *AdvancedEncryptionService) GenerateSessionKey(sessionID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	s.sessionKeys[sessionID] = key

	if time.Since(s.keyRotationTime) > 24*time.Hour {
		s.rotateMasterKey()
	}

	return key, nil
}

func (s *AdvancedEncryptionService) rotateMasterKey() {
	hash := sha256.Sum256([]byte(fmt.Sprintf("rotated-%d", time.Now().Unix())))
	s.masterKey = hash[:]
	s.keyRotationTime = time.Now()
}

func (s *AdvancedEncryptionService) EncryptWithSessionKey(sessionID string, data []byte) (string, error) {
	s.mu.RLock()
	key, exists := s.sessionKeys[sessionID]
	s.mu.RUnlock()

	if !exists {
		var err error
		key, err = s.GenerateSessionKey(sessionID)
		if err != nil {
			return "", err
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *AdvancedEncryptionService) DecryptWithSessionKey(sessionID string, encrypted string) ([]byte, error) {
	s.mu.RLock()
	key, exists := s.sessionKeys[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session key not found: %s", sessionID)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *AdvancedEncryptionService) CleanupSessionKey(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessionKeys, sessionID)
}

func (s *AdvancedEncryptionService) EncryptData(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonData)
	checksum := base64.StdEncoding.EncodeToString(hash[:8])

	encrypted, err := s.EncryptWithSessionKey(checksum, jsonData)
	if err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"encrypted": encrypted,
		"checksum":  checksum,
		"timestamp": time.Now().Unix(),
		"version":   "3.0",
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(resultJSON), nil
}

func (s *AdvancedEncryptionService) DecryptData(encryptedData string, target interface{}) error {
	decoded, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return err
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(decoded, &envelope); err != nil {
		return err
	}

	encryptedStr, ok := envelope["encrypted"].(string)
	if !ok {
		return fmt.Errorf("missing encrypted field")
	}

	checksum, ok := envelope["checksum"].(string)
	if !ok {
		return fmt.Errorf("missing checksum field")
	}

	decrypted, err := s.DecryptWithSessionKey(checksum, encryptedStr)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	if err := json.Unmarshal(decrypted, target); err != nil {
		return fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	return nil
}
