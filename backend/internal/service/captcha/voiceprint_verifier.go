package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type VoiceprintVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VoiceprintVerifyRequest struct {
	SessionID     string    `json:"session_id" binding:"required"`
	VoiceData     string    `json:"voice_data"`
	Features      *VoiceFeatures `json:"features"`
}

type VoiceprintVerifyResult struct {
	Success        bool    `json:"success"`
	Message        string  `json:"message"`
	SimilarityScore float64 `json:"similarity_score"`
	MatchLevel     string  `json:"match_level"`
}

type VoiceFeatures struct {
	MFCC           []float64 `json:"mfcc"`
	SpectralFlux   []float64 `json:"spectral_flux"`
	Formants       []float64 `json:"formants"`
	FundamentalFreq float64  `json:"fundamental_freq"`
	Energy         float64   `json:"energy"`
}

const (
	VoiceprintMatchLevelHigh   = "high"
	VoiceprintMatchLevelMedium = "medium"
	VoiceprintMatchLevelLow    = "low"
)

func NewVoiceprintVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VoiceprintVerifierService {
	return &VoiceprintVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *VoiceprintVerifierService) Verify(ctx context.Context, req *VoiceprintVerifyRequest) (*VoiceprintVerifyResult, error) {
	session, err := v.getSession(ctx, req.SessionID)
	if err != nil {
		return &VoiceprintVerifyResult{
			Success: false,
			Message: "Session not found",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VoiceprintVerifyResult{
			Success: false,
			Message: "验证码已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VoiceprintVerifyResult{
			Success: false,
			Message: "验证次数已用完",
		}, nil
	}

	v.incrementVerifyCount(ctx, session.SessionID)

	if session.Status == "verified" {
		return &VoiceprintVerifyResult{
			Success:         true,
			Message:         "验证码已验证通过",
			SimilarityScore: session.SimilarityScore,
			MatchLevel:      v.getMatchLevel(session.SimilarityScore),
		}, nil
	}

	var pattern VoiceprintPattern
	if err := json.Unmarshal([]byte(session.Pattern), &pattern); err != nil {
		return &VoiceprintVerifyResult{
			Success: false,
			Message: "Pattern parsing failed",
		}, nil
	}

	similarityScore := v.calculateSimilarity(req.Features, &pattern)
	matchLevel := v.getMatchLevel(similarityScore)

	v.updateSimilarityScore(ctx, session.SessionID, similarityScore)

	if similarityScore >= 0.7 {
		v.markAsVerified(ctx, session.SessionID)
		return &VoiceprintVerifyResult{
			Success:         true,
			Message:         "声纹验证成功",
			SimilarityScore: similarityScore,
			MatchLevel:      matchLevel,
		}, nil
	}

	return &VoiceprintVerifyResult{
		Success:         false,
		Message:         fmt.Sprintf("声纹不匹配，相似度 %.0f%%", similarityScore*100),
		SimilarityScore: similarityScore,
		MatchLevel:      matchLevel,
	}, nil
}

func (v *VoiceprintVerifierService) getSession(ctx context.Context, sessionID string) (*models.VoiceprintCaptchaSession, error) {
	if v.sessionCache != nil {
		if session, err := v.sessionCache.GetVoiceprint(ctx, sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		if session, err := v.captchaRepo.GetVoiceprintSession(sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (v *VoiceprintVerifierService) incrementVerifyCount(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.IncrementVoiceprintVerifyCount(ctx, sessionID); err != nil {
			log.Printf("增加声纹验证计数失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.IncrementVoiceprintVerifyCount(sessionID); err != nil {
			log.Printf("数据库增加声纹验证计数失败: %v", err)
		}
	}
}

func (v *VoiceprintVerifierService) updateSimilarityScore(ctx context.Context, sessionID string, score float64) {
	if v.sessionCache != nil {
		if err := v.sessionCache.UpdateVoiceprintSimilarity(ctx, sessionID, score); err != nil {
			log.Printf("更新声纹相似度失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.UpdateVoiceprintSimilarity(sessionID, score); err != nil {
			log.Printf("数据库更新声纹相似度失败: %v", err)
		}
	}
}

func (v *VoiceprintVerifierService) markAsVerified(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.MarkVoiceprintAsVerified(ctx, sessionID); err != nil {
			log.Printf("缓存标记声纹验证失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.MarkVoiceprintAsVerified(sessionID); err != nil {
			log.Printf("数据库标记声纹验证失败: %v", err)
		}
	}
}

func (v *VoiceprintVerifierService) calculateSimilarity(reqFeatures *VoiceFeatures, pattern *VoiceprintPattern) float64 {
	if reqFeatures == nil {
		return 0
	}

	var score float64 = 0
	var count int = 0

	if len(reqFeatures.MFCC) > 0 && len(pattern.Frequencies) > 0 {
		freqSim := v.cosineSimilarity(reqFeatures.MFCC, pattern.Frequencies)
		score += freqSim
		count++
	}

	if len(reqFeatures.Formants) > 0 {
		formantSim := v.cosineSimilarity(reqFeatures.Formants, pattern.Frequencies[:len(reqFeatures.Formants)])
		score += formantSim
		count++
	}

	if reqFeatures.FundamentalFreq > 0 && len(pattern.Frequencies) > 0 {
		targetFreq := pattern.Frequencies[0]
		freqDiff := math.Abs(reqFeatures.FundamentalFreq - targetFreq)
		freqSim := math.Max(0, 1-freqDiff/targetFreq)
		score += freqSim
		count++
	}

	if reqFeatures.Energy > 0 && len(pattern.Amplitudes) > 0 {
		avgAmp := 0.0
		for _, a := range pattern.Amplitudes {
			avgAmp += a
		}
		avgAmp /= float64(len(pattern.Amplitudes))
		energySim := math.Max(0, 1-math.Abs(reqFeatures.Energy-avgAmp)/avgAmp)
		score += energySim
		count++
	}

	if count == 0 {
		return 0.5
	}

	finalScore := score / float64(count)
	if finalScore > 1 {
		finalScore = 1
	}
	if finalScore < 0 {
		finalScore = 0
	}

	return finalScore
}

func (v *VoiceprintVerifierService) cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (v *VoiceprintVerifierService) getMatchLevel(score float64) string {
	if score >= 0.85 {
		return VoiceprintMatchLevelHigh
	} else if score >= 0.7 {
		return VoiceprintMatchLevelMedium
	}
	return VoiceprintMatchLevelLow
}

func (v *VoiceprintVerifierService) ExtractFeatures(audioData []byte) *VoiceFeatures {
	if len(audioData) == 0 {
		return nil
	}

	mfcc := v.extractMFCC(audioData)
	spectralFlux := v.extractSpectralFlux(audioData)
	formants := v.extractFormants(audioData)
	fundamentalFreq := v.extractFundamentalFreq(audioData)
	energy := v.calculateEnergy(audioData)

	return &VoiceFeatures{
		MFCC:            mfcc,
		SpectralFlux:    spectralFlux,
		Formants:        formants,
		FundamentalFreq: fundamentalFreq,
		Energy:          energy,
	}
}

func (v *VoiceprintVerifierService) extractMFCC(audioData []byte) []float64 {
	frameSize := 512
	numFrames := len(audioData) / 2 / frameSize
	if numFrames < 1 {
		numFrames = 1
	}

	mfcc := make([]float64, 13)
	for i := 0; i < 13; i++ {
		baseFreq := 100.0 + float64(i)*50.0
		var sum float64
		for j := 0; j < numFrames; j++ {
			offset := j * frameSize * 2
			if offset+frameSize*2 > len(audioData) {
				break
			}
			var frameSum float64
			for k := 0; k < frameSize && offset+k*2+1 < len(audioData); k++ {
				sample := int16(audioData[offset+k*2]) | (int16(audioData[offset+k*2+1]) << 8)
				frameSum += math.Abs(float64(sample))
			}
			sum += frameSum * mathSin(2*3.14159*baseFreq*float64(j)/float64(numFrames))
		}
		mfcc[i] = sum / float64(numFrames)
	}

	return mfcc
}

func (v *VoiceprintVerifierService) extractSpectralFlux(audioData []byte) []float64 {
	frameSize := 256
	numFrames := len(audioData) / 2 / frameSize
	if numFrames < 2 {
		numFrames = 2
	}

	spectralFlux := make([]float64, numFrames-1)
	for i := 0; i < numFrames-1; i++ {
		offset := i * frameSize * 2
		var energy1, energy2 float64
		for j := 0; j < frameSize && offset+j*2+1 < len(audioData); j++ {
			sample := int16(audioData[offset+j*2]) | (int16(audioData[offset+j*2+1]) << 8)
			energy1 += float64(sample) * float64(sample)
		}
		offset2 := (i + 1) * frameSize * 2
		for j := 0; j < frameSize && offset2+j*2+1 < len(audioData); j++ {
			sample := int16(audioData[offset2+j*2]) | (int16(audioData[offset2+j*2+1]) << 8)
			energy2 += float64(sample) * float64(sample)
		}
		spectralFlux[i] = math.Sqrt(energy2) - math.Sqrt(energy1)
		if spectralFlux[i] < 0 {
			spectralFlux[i] = -spectralFlux[i]
		}
	}

	return spectralFlux
}

func (v *VoiceprintVerifierService) extractFormants(audioData []byte) []float64 {
	formants := make([]float64, 5)
	formantFreqs := []float64{500, 1500, 2500, 3500, 4500}

	sampleCount := len(audioData) / 2
	if sampleCount == 0 {
		return formants
	}

	for i, freq := range formantFreqs {
		var sum float64
		count := 0
		for j := 0; j < sampleCount && j < 1000; j++ {
			offset := j * 2
			if offset+1 >= len(audioData) {
				break
			}
			sample := int16(audioData[offset]) | (int16(audioData[offset+1]) << 8)
			sum += float64(sample) * mathSin(2*3.14159*freq*float64(j)/float64(sampleCount))
			count++
		}
		if count > 0 {
			formants[i] = math.Abs(sum) / float64(count)
		}
	}

	return formants
}

func (v *VoiceprintVerifierService) extractFundamentalFreq(audioData []byte) float64 {
	sampleCount := len(audioData) / 2
	if sampleCount < 256 {
		return 0
	}

	var maxCorrelation float64 = 0
	var fundamentalFreq float64 = 0

	minPeriod := sampleCount / 1000
	maxPeriod := sampleCount / 80
	if minPeriod < 1 {
		minPeriod = 1
	}
	if maxPeriod > sampleCount/2 {
		maxPeriod = sampleCount / 2
	}

	for period := minPeriod; period < maxPeriod; period++ {
		var correlation float64 = 0
		var norm1, norm2 float64

		for i := 0; i < sampleCount-period && i < 1000; i++ {
			offset := i * 2
			if offset+1 >= len(audioData) || (i+period)*2+1 >= len(audioData) {
				break
			}
			s1 := int16(audioData[offset]) | (int16(audioData[offset+1]) << 8)
			s2 := int16(audioData[(i+period)*2]) | (int16(audioData[(i+period)*2+1]) << 8)
			correlation += float64(s1) * float64(s2)
			norm1 += float64(s1) * float64(s1)
			norm2 += float64(s2) * float64(s2)
		}

		if norm1 > 0 && norm2 > 0 {
			normalizedCorr := correlation / (math.Sqrt(norm1) * math.Sqrt(norm2))
			if normalizedCorr > maxCorrelation {
				maxCorrelation = normalizedCorr
				fundamentalFreq = 44100.0 / float64(period)
			}
		}
	}

	return fundamentalFreq
}

func (v *VoiceprintVerifierService) calculateEnergy(audioData []byte) float64 {
	sampleCount := len(audioData) / 2
	if sampleCount == 0 {
		return 0
	}

	var energy float64
	for i := 0; i < sampleCount && i < 10000; i++ {
		offset := i * 2
		if offset+1 >= len(audioData) {
			break
		}
		sample := int16(audioData[offset]) | (int16(audioData[offset+1]) << 8)
		energy += float64(sample) * float64(sample)
	}

	return math.Sqrt(energy / float64(sampleCount))
}

func (v *VoiceprintVerifierService) GetSessionForStatus(ctx context.Context, sessionID string) (*models.VoiceprintCaptchaSession, error) {
	return v.getSession(ctx, sessionID)
}
