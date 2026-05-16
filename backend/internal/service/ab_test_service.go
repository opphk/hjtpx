package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrABTestNotFound    = errors.New("ab test not found")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrInvalidTestStatus = errors.New("invalid test status")
	ErrInvalidTraffic    = errors.New("traffic percentage must sum to 100")
)

type ABTestService struct{}

func NewABTestService() *ABTestService {
	return &ABTestService{}
}

type CreateABTestInput struct {
	Name        string                 `json:"name" binding:"required,min=1,max=255"`
	Description string                 `json:"description"`
	ApplicationID uint               `json:"application_id" binding:"required"`
	Variants    []CreateVariantInput  `json:"variants" binding:"required,min=2"`
	Config      map[string]interface{} `json:"config"`
}

type CreateVariantInput struct {
	Name           string                 `json:"name" binding:"required"`
	IsControl      bool                   `json:"is_control"`
	TrafficPercent int                    `json:"traffic_percent" binding:"required,min=0,max=100"`
	Config         map[string]interface{} `json:"config"`
	Description    string                 `json:"description"`
}

type UpdateABTestInput struct {
	Name        *string                `json:"name" binding:"omitempty,max=255"`
	Description *string                `json:"description"`
	Variants    *[]CreateVariantInput  `json:"variants"`
	Config      *map[string]interface{} `json:"config"`
}

type ListABTestsFilter struct {
	Page         int
	PageSize     int
	Keyword      string
	ApplicationID uint
	Status       string
	SortField    string
	SortOrder    string
}

type ABTestSummary struct {
	Total    int64 `json:"total"`
	Running  int64 `json:"running"`
	Stop     int64 `json:"stopped"`
	Draft    int64 `json:"draft"`
}

type VariantStats struct {
	VariantID      uint    `json:"variant_id"`
	VariantName    string  `json:"variant_name"`
	IsControl      bool    `json:"is_control"`
	Visitors       int64   `json:"visitors"`
	Conversions    int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	Improvement    float64 `json:"improvement"`
	Confidence     float64 `json:"confidence"`
	TrafficPercent int     `json:"traffic_percent"`
}

type TestReport struct {
	TestID         uint           `json:"test_id"`
	TestName       string         `json:"test_name"`
	Status         string         `json:"status"`
	StartDate      *time.Time     `json:"start_date"`
	EndDate        *time.Time     `json:"end_date"`
	TotalVisitors  int64          `json:"total_visitors"`
	WinningVariant *uint          `json:"winning_variant,omitempty"`
	Variants       []VariantStats `json:"variants"`
	Recommendations []string      `json:"recommendations"`
}

type AssignVariantRequest struct {
	TestID        uint   `json:"test_id" binding:"required"`
	SessionID     string `json:"session_id" binding:"required"`
	UserID        *uint  `json:"user_id"`
	DeviceID      string `json:"device_id"`
}

type TrackEventRequest struct {
	TestID      uint                   `json:"test_id" binding:"required"`
	VariantID   uint                   `json:"variant_id" binding:"required"`
	SessionID   string                 `json:"session_id" binding:"required"`
	EventName   string                 `json:"event_name" binding:"required"`
	EventType   string                 `json:"event_type"`
	IsConversion bool                   `json:"is_conversion"`
	Value       float64                `json:"value"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (s *ABTestService) CreateABTest(input *CreateABTestInput) (*models.ABTest, error) {
	if input.Name == "" {
		return nil, ErrInvalidInput
	}

	totalTraffic := 0
	for _, v := range input.Variants {
		totalTraffic += v.TrafficPercent
	}
	if totalTraffic != 100 {
		return nil, ErrInvalidTraffic
	}

	var hasControl bool
	for _, v := range input.Variants {
		if v.IsControl {
			hasControl = true
			break
		}
	}
	if !hasControl && len(input.Variants) > 0 {
		input.Variants[0].IsControl = true
	}

	var configJSON string
	if input.Config != nil {
		configBytes, err := json.Marshal(input.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		configJSON = string(configBytes)
	}

	test := &models.ABTest{
		Name:          input.Name,
		Description:   input.Description,
		ApplicationID: input.ApplicationID,
		Status:        "draft",
		Config:        configJSON,
	}

	if err := database.DB.Create(test).Error; err != nil {
		return nil, fmt.Errorf("failed to create ab test: %w", err)
	}

	for _, v := range input.Variants {
		var variantConfigJSON string
		if v.Config != nil {
			variantConfigBytes, err := json.Marshal(v.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal variant config: %w", err)
			}
			variantConfigJSON = string(variantConfigBytes)
		}

		variant := &models.ABTestVariant{
			ABTestID:      test.ID,
			Name:          v.Name,
			IsControl:     v.IsControl,
			TrafficPercent: v.TrafficPercent,
			Config:        variantConfigJSON,
			Description:   v.Description,
		}
		if err := database.DB.Create(variant).Error; err != nil {
			return nil, fmt.Errorf("failed to create variant: %w", err)
		}
	}

	database.DB.Preload("Variants").Preload("Application").First(test, test.ID)
	return test, nil
}

func (s *ABTestService) GetABTestByID(id uint) (*models.ABTest, error) {
	var test models.ABTest
	if err := database.DB.Preload("Variants").Preload("Application").First(&test, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrABTestNotFound
		}
		return nil, fmt.Errorf("failed to get ab test: %w", err)
	}
	return &test, nil
}

func (s *ABTestService) ListABTests(filter *ListABTestsFilter) (*PaginatedResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	db := database.DB.Model(&models.ABTest{})

	if filter.Keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}
	if filter.ApplicationID > 0 {
		db = db.Where("application_id = ?", filter.ApplicationID)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}

	sortField := "created_at"
	sortOrder := "DESC"
	if filter.SortField != "" {
		sortField = filter.SortField
	}
	if filter.SortOrder == "ASC" {
		sortOrder = "ASC"
	}
	db = db.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count ab tests: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	var tests []models.ABTest
	if err := db.Preload("Variants").Preload("Application").Offset(offset).Limit(filter.PageSize).Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("failed to list ab tests: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       tests,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ABTestService) UpdateABTest(id uint, input *UpdateABTestInput) (*models.ABTest, error) {
	test, err := s.GetABTestByID(id)
	if err != nil {
		return nil, err
	}

	if test.Status != "draft" {
		return nil, ErrInvalidTestStatus
	}

	updates := make(map[string]interface{})
	if input.Name != nil && *input.Name != "" {
		updates["name"] = *input.Name
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Config != nil {
		configBytes, err := json.Marshal(*input.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		updates["config"] = string(configBytes)
	}

	if len(updates) > 0 {
		if err := database.DB.Model(test).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update ab test: %w", err)
		}
	}

	if input.Variants != nil {
		totalTraffic := 0
		for _, v := range *input.Variants {
			totalTraffic += v.TrafficPercent
		}
		if totalTraffic != 100 {
			return nil, ErrInvalidTraffic
		}

		database.DB.Where("ab_test_id = ?", id).Delete(&models.ABTestVariant{})

		for _, v := range *input.Variants {
			var variantConfigJSON string
			if v.Config != nil {
				variantConfigBytes, err := json.Marshal(v.Config)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal variant config: %w", err)
				}
				variantConfigJSON = string(variantConfigBytes)
			}

			variant := &models.ABTestVariant{
				ABTestID:       id,
				Name:           v.Name,
				IsControl:      v.IsControl,
				TrafficPercent: v.TrafficPercent,
				Config:         variantConfigJSON,
				Description:    v.Description,
			}
			if err := database.DB.Create(variant).Error; err != nil {
				return nil, fmt.Errorf("failed to create variant: %w", err)
			}
		}
	}

	database.DB.Preload("Variants").Preload("Application").First(test, id)
	return test, nil
}

func (s *ABTestService) DeleteABTest(id uint) error {
	test, err := s.GetABTestByID(id)
	if err != nil {
		return err
	}

	database.DB.Where("ab_test_id = ?", id).Delete(&models.ABTestEvent{})
	database.DB.Where("ab_test_id = ?", id).Delete(&models.ABTestAssignment{})
	database.DB.Where("ab_test_id = ?", id).Delete(&models.ABTestVariant{})

	if err := database.DB.Delete(test).Error; err != nil {
		return fmt.Errorf("failed to delete ab test: %w", err)
	}

	return nil
}

func (s *ABTestService) StartABTest(id uint) (*models.ABTest, error) {
	test, err := s.GetABTestByID(id)
	if err != nil {
		return nil, err
	}

	if test.Status != "draft" {
		return nil, ErrInvalidTestStatus
	}

	now := time.Now()
	if err := database.DB.Model(test).Updates(map[string]interface{}{
		"status":     "running",
		"start_date": &now,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to start ab test: %w", err)
	}

	database.DB.Preload("Variants").Preload("Application").First(test, id)
	return test, nil
}

func (s *ABTestService) StopABTest(id uint) (*models.ABTest, error) {
	test, err := s.GetABTestByID(id)
	if err != nil {
		return nil, err
	}

	if test.Status != "running" {
		return nil, ErrInvalidTestStatus
	}

	now := time.Now()
	if err := database.DB.Model(test).Updates(map[string]interface{}{
		"status":   "stopped",
		"end_date": &now,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to stop ab test: %w", err)
	}

	database.DB.Preload("Variants").Preload("Application").First(test, id)
	return test, nil
}

func (s *ABTestService) GetActiveTests(applicationID uint) ([]models.ABTest, error) {
	var tests []models.ABTest
	query := database.DB.Where("status = ?", "running")
	if applicationID > 0 {
		query = query.Where("application_id = ?", applicationID)
	}
	if err := query.Preload("Variants").Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("failed to get active tests: %w", err)
	}
	return tests, nil
}

func (s *ABTestService) GetABTestSummary(applicationID uint) (*ABTestSummary, error) {
	var summary ABTestSummary

	db := database.DB.Model(&models.ABTest{})
	if applicationID > 0 {
		db = db.Where("application_id = ?", applicationID)
	}
	db.Count(&summary.Total)

	db.Where("status = ?", "running").Count(&summary.Running)
	db.Where("status = ?", "stopped").Count(&summary.Stop)
	db.Where("status = ?", "draft").Count(&summary.Draft)

	return &summary, nil
}

func (s *ABTestService) AssignVariant(req *AssignVariantRequest) (*models.ABTestVariant, error) {
	test, err := s.GetABTestByID(req.TestID)
	if err != nil {
		return nil, err
	}

	if test.Status != "running" {
		return nil, ErrInvalidTestStatus
	}

	var existingAssignment models.ABTestAssignment
	result := database.DB.Where("ab_test_id = ? AND session_id = ?", req.TestID, req.SessionID).First(&existingAssignment)
	if result.Error == nil {
		var variant models.ABTestVariant
		if err := database.DB.First(&variant, existingAssignment.VariantID).Error; err != nil {
			return nil, ErrVariantNotFound
		}
		return &variant, nil
	}

	selectedVariant := s.selectVariantByHash(req.SessionID, test.Variants)

	assignment := &models.ABTestAssignment{
		ABTestID:   req.TestID,
		VariantID:  selectedVariant.ID,
		SessionID:  req.SessionID,
		UserID:     req.UserID,
		DeviceID:   req.DeviceID,
		AssignedAt: time.Now(),
	}
	if err := database.DB.Create(assignment).Error; err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return selectedVariant, nil
}

func (s *ABTestService) selectVariantByHash(sessionID string, variants []models.ABTestVariant) *models.ABTestVariant {
	hash := sha256.Sum256([]byte(sessionID))
	hashHex := hex.EncodeToString(hash[:])

	hashNum := 0
	for i := 0; i < 8; i++ {
		hashNum = hashNum << 4
		val, _ := strconv.ParseInt(string(hashHex[i]), 16, 64)
		hashNum += int(val)
	}
	hashMod := hashNum % 100

	cumulative := 0
	for _, variant := range variants {
		cumulative += variant.TrafficPercent
		if hashMod < cumulative {
			return &variant
		}
	}

	return &variants[0]
}

func (s *ABTestService) TrackEvent(req *TrackEventRequest) error {
	var metadataJSON string
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	event := &models.ABTestEvent{
		ABTestID:     req.TestID,
		VariantID:    req.VariantID,
		SessionID:    req.SessionID,
		EventName:    req.EventName,
		EventType:    req.EventType,
		IsConversion: req.IsConversion,
		Value:        req.Value,
		Metadata:     metadataJSON,
		Timestamp:    time.Now(),
	}

	if err := database.DB.Create(event).Error; err != nil {
		return fmt.Errorf("failed to track event: %w", err)
	}

	return nil
}

func (s *ABTestService) GetTestReport(testID uint) (*TestReport, error) {
	test, err := s.GetABTestByID(testID)
	if err != nil {
		return nil, err
	}

	var totalVisitors int64
	database.DB.Model(&models.ABTestAssignment{}).
		Where("ab_test_id = ?", testID).
		Count(&totalVisitors)

	variantsStats := make([]VariantStats, 0, len(test.Variants))
	var controlConversionRate float64
	var controlVisitors, controlConversions int64

	for _, variant := range test.Variants {
		var visitors int64
		database.DB.Model(&models.ABTestAssignment{}).
			Where("ab_test_id = ? AND variant_id = ?", testID, variant.ID).
			Count(&visitors)

		var conversions int64
		database.DB.Model(&models.ABTestEvent{}).
			Where("ab_test_id = ? AND variant_id = ? AND is_conversion = ?", testID, variant.ID, true).
			Count(&conversions)

		var conversionRate float64
		if visitors > 0 {
			conversionRate = float64(conversions) / float64(visitors) * 100
		}

		if variant.IsControl {
			controlConversionRate = conversionRate
			controlVisitors = visitors
			controlConversions = conversions
		}

		variantsStats = append(variantsStats, VariantStats{
			VariantID:      variant.ID,
			VariantName:    variant.Name,
			IsControl:      variant.IsControl,
			Visitors:       visitors,
			Conversions:    conversions,
			ConversionRate: conversionRate,
			TrafficPercent: variant.TrafficPercent,
		})
	}

	for i := range variantsStats {
		if !variantsStats[i].IsControl && controlConversionRate > 0 {
			variantsStats[i].Improvement = ((variantsStats[i].ConversionRate - controlConversionRate) / controlConversionRate) * 100
		}
		if controlVisitors > 0 {
			variantsStats[i].Confidence = s.calculateConfidence(
				controlVisitors, controlConversions,
				variantsStats[i].Visitors, variantsStats[i].Conversions,
			)
		}
	}

	var winningVariant *uint
	if test.Status == "stopped" || test.Status == "running" {
		var maxConfidence float64
		for _, vs := range variantsStats {
			if !vs.IsControl && vs.Confidence > 95 && vs.ConversionRate > controlConversionRate {
				if vs.Confidence > maxConfidence {
					maxConfidence = vs.Confidence
					winnerID := vs.VariantID
					winningVariant = &winnerID
				}
			}
		}
	}

	recommendations := s.generateRecommendations(variantsStats, test.Status)

	return &TestReport{
		TestID:          test.ID,
		TestName:        test.Name,
		Status:          test.Status,
		StartDate:       test.StartDate,
		EndDate:         test.EndDate,
		TotalVisitors:   totalVisitors,
		WinningVariant:  winningVariant,
		Variants:        variantsStats,
		Recommendations: recommendations,
	}, nil
}

func (s *ABTestService) calculateConfidence(cVisitors, cConversions, vVisitors, vConversions int64) float64 {
	if cVisitors == 0 || vVisitors == 0 {
		return 0
	}

	cRate := float64(cConversions) / float64(cVisitors)
	vRate := float64(vConversions) / float64(vVisitors)

	pPool := float64(cConversions + vConversions) / float64(cVisitors + vVisitors)
	sePool := math.Sqrt(pPool * (1 - pPool) * (1/float64(cVisitors) + 1/float64(vVisitors)))

	if sePool == 0 {
		return 0
	}

	zScore := (vRate - cRate) / sePool

	confidence := s.normalCDF(math.Abs(zScore)) * 100
	return confidence
}

func (s *ABTestService) normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func (s *ABTestService) generateRecommendations(variants []VariantStats, status string) []string {
	recommendations := make([]string, 0)

	var minVisitors int64 = math.MaxInt64
	var maxVisitors int64 = 0
	for _, v := range variants {
		if v.Visitors < minVisitors {
			minVisitors = v.Visitors
		}
		if v.Visitors > maxVisitors {
			maxVisitors = v.Visitors
		}
	}

	if minVisitors < 100 {
		recommendations = append(recommendations, "测试数据量较少，建议继续收集更多访客数据以获得可靠结论")
	}

	if status == "running" {
		recommendations = append(recommendations, "测试正在进行中，请勿过早结束，建议每个变体至少收集1000个访客")
	}

	hasWinner := false
	for _, v := range variants {
		if !v.IsControl && v.Confidence > 95 && v.ConversionRate > 0 {
			hasWinner = true
			recommendations = append(recommendations,
				fmt.Sprintf("变体 '%s' 表现优于对照组 (提升 %.2f%%, 置信度 %.1f%%)，考虑结束测试",
					v.VariantName, v.Improvement, v.Confidence))
		}
	}

	if !hasWinner && minVisitors >= 1000 && status == "running" {
		recommendations = append(recommendations, "尚未发现具有统计显著性的变体，考虑调整变体设计或延长测试时间")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "继续收集数据以获得更可靠的测试结果")
	}

	return recommendations
}
