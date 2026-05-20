package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	ErrWatermarkFailed       = errors.New("watermark operation failed")
	ErrInvalidWatermark      = errors.New("invalid watermark data")
	ErrWatermarkNotFound     = errors.New("watermark not found")
	ErrModelNotWatermarked   = errors.New("model is not watermarked")
	ErrVerificationFailed    = errors.New("watermark verification failed")
)

type WatermarkType string

const (
	WatermarkTypeRobust     WatermarkType = "robust"
	WatermarkTypeFragile    WatermarkType = "fragile"
	WatermarkTypeSemiFragile WatermarkType = "semi_fragile"
)

type EmbeddingMethod string

const (
	EmbeddingWeight perturbation  = "weight_perturbation"
	EmbeddingActivation EmbeddingMethod = "activation"
	EmbeddingOutput    EmbeddingMethod = "output"
	EmbeddingBackdoor  EmbeddingMethod = "backdoor"
)

type WatermarkConfig struct {
	WatermarkID     string
	Type            WatermarkType
	EmbeddingMethod EmbeddingMethod
	Message         []byte
	Strength        float64
	Position        []int
	SecretKey       []byte
}

type Watermark struct {
	WatermarkID     string           `json:"watermark_id"`
	Type            WatermarkType    `json:"type"`
	EmbeddingMethod EmbeddingMethod   `json:"embedding_method"`
	Message         []byte           `json:"message"`
	Hash            string           `json:"hash"`
	Strength        float64          `json:"strength"`
	Position        []int            `json:"position"`
	LayerIndex      int              `json:"layer_index"`
	NeuronIndex     int              `json:"neuron_index"`
	CreatedAt       time.Time        `json:"created_at"`
	CreatorID       string           `json:"creator_id"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type WatermarkVerificationResult struct {
	IsValid         bool               `json:"is_valid"`
	Confidence      float64            `json:"confidence"`
	WatermarkData   []byte             `json:"watermark_data,omitempty"`
	ExtractedHash   string             `json:"extracted_hash"`
	ExpectedHash    string             `json:"expected_hash"`
	MatchScore      float64            `json:"match_score"`
	MethodUsed      string             `json:"method_used"`
	ProcessingTime  time.Duration      `json:"processing_time"`
}

type WatermarkRemovalResult struct {
	Success        bool     `json:"success"`
	RemovedLayers  int      `json:"removed_layers"`
	RemainingWatermarks int `json:"remaining_watermarks"`
	IntegrityCheck bool     `json:"integrity_check"`
}

type AIModelWatermarkingService struct {
	mu          sync.RWMutex
	watermarks  map[string]*Watermark
	models      map[string]*WatermarkedModel
	keys        map[string]*WatermarkKey
	verificationHistory map[string][]*WatermarkVerificationResult
}

type WatermarkedModel struct {
	ModelID       string
	WatermarkIDs  []string
	OriginalHash  string
	WatermarkedAt time.Time
	Metadata      map[string]interface{}
}

type WatermarkKey struct {
	KeyID     string
	PublicKey []byte
	SecretKey []byte
	Algorithm string
	CreatedAt time.Time
}

func NewAIModelWatermarkingService() *AIModelWatermarkingService {
	return &AIModelWatermarkingService{
		watermarks:  make(map[string]*Watermark),
		models:     make(map[string]*WatermarkedModel),
		keys:       make(map[string]*WatermarkKey),
		verificationHistory: make(map[string][]*WatermarkVerificationResult),
	}
}

func (s *AIModelWatermarkingService) GenerateWatermarkKey(ctx context.Context, algorithm string) (*WatermarkKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyID := generateKeyID()
	publicKey := make([]byte, 32)
	secretKey := make([]byte, 32)

	if _, err := rand.Read(publicKey); err != nil {
		return nil, fmt.Errorf("%w: failed to generate public key: %v", ErrWatermarkFailed, err)
	}

	if _, err := rand.Read(secretKey); err != nil {
		return nil, fmt.Errorf("%w: failed to generate secret key: %v", ErrWatermarkFailed, err)
	}

	key := &WatermarkKey{
		KeyID:     keyID,
		PublicKey: publicKey,
		SecretKey: secretKey,
		Algorithm: algorithm,
		CreatedAt: time.Now(),
	}

	s.keys[keyID] = key

	return key, nil
}

func (s *AIModelWatermarkingService) EmbedWatermark(ctx context.Context, modelID string, config *WatermarkConfig) (*Watermark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(config.Message) == 0 {
		return nil, ErrInvalidWatermark
	}

	watermarkID := generateWatermarkID()

	messageHash := s.computeHash(config.Message)

	watermark := &Watermark{
		WatermarkID:     watermarkID,
		Type:            config.Type,
		EmbeddingMethod: config.EmbeddingMethod,
		Message:         config.Message,
		Hash:            messageHash,
		Strength:        config.Strength,
		Position:        config.Position,
		LayerIndex:      0,
		NeuronIndex:     0,
		CreatedAt:       time.Now(),
		CreatorID:       "system",
	}

	if len(config.Position) >= 2 {
		watermark.LayerIndex = config.Position[0]
		watermark.NeuronIndex = config.Position[1]
	}

	s.watermarks[watermarkID] = watermark

	watermarkedModel, exists := s.models[modelID]
	if !exists {
		watermarkedModel = &WatermarkedModel{
			ModelID:       modelID,
			WatermarkIDs:  make([]string, 0),
			WatermarkedAt: time.Now(),
			Metadata:      make(map[string]interface{}),
		}
		s.models[modelID] = watermarkedModel
	}

	watermarkedModel.WatermarkIDs = append(watermarkedModel.WatermarkIDs, watermarkID)

	return watermark, nil
}

func (s *AIModelWatermarkingService) VerifyWatermark(ctx context.Context, modelID string, expectedMessage []byte) (*WatermarkVerificationResult, error) {
	start := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	model, exists := s.models[modelID]
	if !exists {
		return &WatermarkVerificationResult{
			IsValid:        false,
			Confidence:     0.0,
			MethodUsed:     "none",
			ProcessingTime: time.Since(start),
		}, ErrModelNotWatermarked
	}

	if len(model.WatermarkIDs) == 0 {
		return &WatermarkVerificationResult{
			IsValid:        false,
			Confidence:     0.0,
			MethodUsed:     "none",
			ProcessingTime: time.Since(start),
		}, ErrWatermarkNotFound
	}

	expectedHash := s.computeHash(expectedMessage)

	bestMatch := 0.0
	var bestWatermark *Watermark

	for _, watermarkID := range model.WatermarkIDs {
		watermark, exists := s.watermarks[watermarkID]
		if !exists {
			continue
		}

		matchScore := s.computeMatchScore(watermark.Hash, expectedHash)

		if matchScore > bestMatch {
			bestMatch = matchScore
			bestWatermark = watermark
		}
	}

	result := &WatermarkVerificationResult{
		IsValid:       bestMatch > 0.8,
		Confidence:    bestMatch,
		ExtractedHash: "",
		ExpectedHash:  expectedHash,
		MatchScore:    bestMatch,
		MethodUsed:    "hash_comparison",
		ProcessingTime: time.Since(start),
	}

	if bestWatermark != nil {
		result.ExtractedHash = bestWatermark.Hash
		result.WatermarkData = bestWatermark.Message
	}

	history := s.verificationHistory[modelID]
	history = append(history, result)
	s.verificationHistory[modelID] = history

	return result, nil
}

func (s *AIModelWatermarkingService) ExtractWatermark(ctx context.Context, modelID string) (*Watermark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotWatermarked
	}

	if len(model.WatermarkIDs) == 0 {
		return nil, ErrWatermarkNotFound
	}

	watermarkID := model.WatermarkIDs[0]
	watermark, exists := s.watermarks[watermarkID]
	if !exists {
		return nil, ErrWatermarkNotFound
	}

	return watermark, nil
}

func (s *AIModelWatermarkingService) RemoveWatermark(ctx context.Context, modelID string) (*WatermarkRemovalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotWatermarked
	}

	removedLayers := len(model.WatermarkIDs)

	for _, watermarkID := range model.WatermarkIDs {
		delete(s.watermarks, watermarkID)
	}

	model.WatermarkIDs = make([]string, 0)

	return &WatermarkRemovalResult{
		Success:           true,
		RemovedLayers:     removedLayers,
		RemainingWatermarks: 0,
		IntegrityCheck:    true,
	}, nil
}

func (s *AIModelWatermarkingService) EmbedRobustWatermark(ctx context.Context, modelID string, message []byte, strength float64) (*Watermark, error) {
	config := &WatermarkConfig{
		Type:            WatermarkTypeRobust,
		EmbeddingMethod: EmbeddingWeight,
		Message:         message,
		Strength:        strength,
		Position:        s.generateRandomPosition(),
	}

	return s.EmbedWatermark(ctx, modelID, config)
}

func (s *AIModelWatermarkingService) EmbedFragileWatermark(ctx context.Context, modelID string, message []byte) (*Watermark, error) {
	config := &WatermarkConfig{
		Type:            WatermarkTypeFragile,
		EmbeddingMethod: EmbeddingOutput,
		Message:         message,
		Strength:        1.0,
		Position:        s.generateRandomPosition(),
	}

	return s.EmbedWatermark(ctx, modelID, config)
}

func (s *AIModelWatermarkingService) EmbedSemiFragileWatermark(ctx context.Context, modelID string, message []byte, tolerance float64) (*Watermark, error) {
	config := &WatermarkConfig{
		Type:            WatermarkTypeSemiFragile,
		EmbeddingMethod: EmbeddingActivation,
		Message:         message,
		Strength:        tolerance,
		Position:        s.generateRandomPosition(),
	}

	return s.EmbedWatermark(ctx, modelID, config)
}

func (s *AIModelWatermarkingService) DetectWatermark(ctx context.Context, modelID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return false, nil
	}

	return len(model.WatermarkIDs) > 0, nil
}

func (s *AIModelWatermarkingService) ListWatermarks(ctx context.Context, modelID string) []*Watermark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return []*Watermark{}
	}

	watermarks := make([]*Watermark, 0, len(model.WatermarkIDs))
	for _, watermarkID := range model.WatermarkIDs {
		if watermark, exists := s.watermarks[watermarkID]; exists {
			watermarks = append(watermarks, watermark)
		}
	}

	return watermarks
}

func (s *AIModelWatermarkingService) GetWatermark(ctx context.Context, watermarkID string) (*Watermark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	watermark, exists := s.watermarks[watermarkID]
	if !exists {
		return nil, ErrWatermarkNotFound
	}

	return watermark, nil
}

func (s *AIModelWatermarkingService) GetModelWatermarks(ctx context.Context, modelID string) []*Watermark {
	return s.ListWatermarks(ctx, modelID)
}

func (s *AIModelWatermarkingService) UpdateWatermarkMetadata(ctx context.Context, watermarkID string, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	watermark, exists := s.watermarks[watermarkID]
	if !exists {
		return ErrWatermarkNotFound
	}

	watermark.Metadata = metadata

	return nil
}

func (s *AIModelWatermarkingService) GetVerificationHistory(ctx context.Context, modelID string) []*WatermarkVerificationResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.verificationHistory[modelID]
	if !exists {
		return []*WatermarkVerificationResult{}
	}

	return history
}

func (s *AIModelWatermarkingService) GetServiceStats(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	stats["total_watermarks"] = len(s.watermarks)
	stats["total_models"] = len(s.models)
	stats["total_keys"] = len(s.keys)

	watermarkedModels := 0
	for _, model := range s.models {
		if len(model.WatermarkIDs) > 0 {
			watermarkedModels++
		}
	}
	stats["watermarked_models"] = watermarkedModels

	verificationCount := 0
	for _, history := range s.verificationHistory {
		verificationCount += len(history)
	}
	stats["total_verifications"] = verificationCount

	watermarkTypes := make(map[string]int)
	for _, watermark := range s.watermarks {
		watermarkTypes[string(watermark.Type)]++
	}
	stats["watermark_types"] = watermarkTypes

	return stats
}

func (s *AIModelWatermarkingService) computeHash(data []byte) string {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = data[i%len(data)]
	}

	hashString := base64.StdEncoding.EncodeToString(hash)
	return hashString
}

func (s *AIModelWatermarkingService) computeMatchScore(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 1.0
	}

	if len(hash1) != len(hash2) {
		return 0.0
	}

	matchCount := 0
	for i := 0; i < len(hash1) && i < len(hash2); i++ {
		if hash1[i] == hash2[i] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(hash1))
}

func (s *AIModelWatermarkingService) generateRandomPosition() []int {
	position := make([]int, 2)
	
	b := make([]byte, 1)
	rand.Read(b)
	position[0] = int(b[0]) % 10

	rand.Read(b)
	position[1] = int(b[0]) % 100

	return position
}

func (s *AIModelWatermarkingService) EmbedWatermarkWithKey(ctx context.Context, modelID string, message []byte, keyID string) (*Watermark, error) {
	s.mu.RLock()
	key, keyExists := s.keys[keyID]
	s.mu.RUnlock()

	if !keyExists {
		return nil, fmt.Errorf("key %s not found", keyID)
	}

	encryptedMessage := make([]byte, len(message))
	for i, b := range message {
		encryptedMessage[i] = b ^ key.SecretKey[i%len(key.SecretKey)]
	}

	config := &WatermarkConfig{
		Type:            WatermarkTypeRobust,
		EmbeddingMethod: EmbeddingWeight,
		Message:         encryptedMessage,
		Strength:        0.5,
		Position:        s.generateRandomPosition(),
	}

	return s.EmbedWatermark(ctx, modelID, config)
}

func (s *AIModelWatermarkingService) VerifyWatermarkWithKey(ctx context.Context, modelID string, expectedMessage []byte, keyID string) (*WatermarkVerificationResult, error) {
	s.mu.RLock()
	key, keyExists := s.keys[keyID]
	s.mu.RUnlock()

	if !keyExists {
		return nil, fmt.Errorf("key %s not found", keyID)
	}

	result, err := s.VerifyWatermark(ctx, modelID, expectedMessage)
	if err != nil {
		return result, err
	}

	if result.WatermarkData != nil {
		decryptedMessage := make([]byte, len(result.WatermarkData))
		for i, b := range result.WatermarkData {
			decryptedMessage[i] = b ^ key.SecretKey[i%len(key.SecretKey)]
		}
		result.WatermarkData = decryptedMessage
	}

	return result, nil
}

func (s *AIModelWatermarkingService) RegisterModel(ctx context.Context, modelID string, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.models[modelID] = &WatermarkedModel{
		ModelID:       modelID,
		WatermarkIDs:  make([]string, 0),
		OriginalHash:  s.computeHash([]byte(modelID)),
		WatermarkedAt: time.Now(),
		Metadata:      metadata,
	}

	return nil
}

func (s *AIModelWatermarkingService) UnregisterModel(ctx context.Context, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.models, modelID)
	delete(s.verificationHistory, modelID)

	return nil
}

func (s *AIModelWatermarkingService) GetModelInfo(ctx context.Context, modelID string) (*WatermarkedModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotWatermarked
	}

	return model, nil
}

type WatermarkExportResult struct {
	Watermarks  []*Watermark `json:"watermarks"`
	ExportTime  time.Time    `json:"export_time"`
	Format      string       `json:"format"`
}

func (s *AIModelWatermarkingService) ExportWatermarks(ctx context.Context, modelID string) (*WatermarkExportResult, error) {
	watermarks := s.ListWatermarks(ctx, modelID)

	return &WatermarkExportResult{
		Watermarks: watermarks,
		ExportTime: time.Now(),
		Format:    "json",
	}, nil
}

func generateWatermarkID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "wm-" + base64.URLEncoding.EncodeToString(b)
}

func generateKeyID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "key-" + base64.URLEncoding.EncodeToString(b)
}
