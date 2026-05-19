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
	Name          string                 `json:"name" binding:"required,min=1,max=255"`
	Description   string                 `json:"description"`
	ApplicationID uint                   `json:"application_id" binding:"required"`
	Variants      []CreateVariantInput   `json:"variants" binding:"required,min=2"`
	Config        map[string]interface{} `json:"config"`
}

type CreateVariantInput struct {
	Name           string                 `json:"name" binding:"required"`
	IsControl      bool                   `json:"is_control"`
	TrafficPercent int                    `json:"traffic_percent" binding:"required,min=0,max=100"`
	Config         map[string]interface{} `json:"config"`
	Description    string                 `json:"description"`
}

type UpdateABTestInput struct {
	Name        *string                 `json:"name" binding:"omitempty,max=255"`
	Description *string                 `json:"description"`
	Variants    *[]CreateVariantInput   `json:"variants"`
	Config      *map[string]interface{} `json:"config"`
}

type ListABTestsFilter struct {
	Page          int
	PageSize      int
	Keyword       string
	ApplicationID uint
	Status        string
	SortField     string
	SortOrder     string
}

type ABTestSummary struct {
	Total   int64 `json:"total"`
	Running int64 `json:"running"`
	Stop    int64 `json:"stopped"`
	Draft   int64 `json:"draft"`
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
	TestID          uint           `json:"test_id"`
	TestName        string         `json:"test_name"`
	Status          string         `json:"status"`
	StartDate       *time.Time     `json:"start_date"`
	EndDate         *time.Time     `json:"end_date"`
	TotalVisitors   int64          `json:"total_visitors"`
	WinningVariant  *uint          `json:"winning_variant,omitempty"`
	Variants        []VariantStats `json:"variants"`
	Recommendations []string       `json:"recommendations"`
}

type AssignVariantRequest struct {
	TestID    uint   `json:"test_id" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
	UserID    *uint  `json:"user_id"`
	DeviceID  string `json:"device_id"`
}

type TrackEventRequest struct {
	TestID       uint                   `json:"test_id" binding:"required"`
	VariantID    uint                   `json:"variant_id" binding:"required"`
	SessionID    string                 `json:"session_id" binding:"required"`
	EventName    string                 `json:"event_name" binding:"required"`
	EventType    string                 `json:"event_type"`
	IsConversion bool                   `json:"is_conversion"`
	Value        float64                `json:"value"`
	Metadata     map[string]interface{} `json:"metadata"`
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
			ABTestID:       test.ID,
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

	pPool := float64(cConversions+vConversions) / float64(cVisitors+vVisitors)
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

type VariantComparison struct {
	Variant1       VariantStats `json:"variant1"`
	Variant2       VariantStats `json:"variant2"`
	RelativeDiff   float64      `json:"relative_diff"`
	AbsoluteDiff   float64      `json:"absolute_diff"`
	StatisticalSig bool         `json:"statistical_significance"`
	Conclusion     string       `json:"conclusion"`
}

func (s *ABTestService) CompareVariants(testID uint) ([]VariantComparison, error) {
	test, err := s.GetABTestByID(testID)
	if err != nil {
		return nil, err
	}

	report, err := s.GetTestReport(testID)
	if err != nil {
		return nil, err
	}

	comparisons := make([]VariantComparison, 0)
	for i := 0; i < len(report.Variants); i++ {
		for j := i + 1; j < len(report.Variants); j++ {
			v1 := report.Variants[i]
			v2 := report.Variants[j]

			relativeDiff := 0.0
			if v1.ConversionRate > 0 {
				relativeDiff = ((v2.ConversionRate - v1.ConversionRate) / v1.ConversionRate) * 100
			}

			absoluteDiff := v2.ConversionRate - v1.ConversionRate

			statSig := false
			if v1.Confidence > 95 {
				statSig = true
			}

			conclusion := s.generateComparisonConclusion(v1, v2, relativeDiff, statSig)

			comparisons = append(comparisons, VariantComparison{
				Variant1:       v1,
				Variant2:       v2,
				RelativeDiff:   relativeDiff,
				AbsoluteDiff:   absoluteDiff,
				StatisticalSig: statSig,
				Conclusion:     conclusion,
			})
		}
	}

	_ = test

	return comparisons, nil
}

func (s *ABTestService) generateComparisonConclusion(v1, v2 VariantStats, relativeDiff float64, statSig bool) string {
	if !statSig {
		return fmt.Sprintf("'%s' 和 '%s' 之间暂无统计显著差异，需更多数据", v1.VariantName, v2.VariantName)
	}

	if relativeDiff > 5 {
		return fmt.Sprintf("'%s' 显著优于 '%s' (提升 %.2f%%)", v2.VariantName, v1.VariantName, relativeDiff)
	} else if relativeDiff < -5 {
		return fmt.Sprintf("'%s' 显著劣于 '%s' (降低 %.2f%%)", v2.VariantName, v1.VariantName, -relativeDiff)
	} else {
		return fmt.Sprintf("'%s' 和 '%s' 表现相近 (差异 %.2f%%)", v1.VariantName, v2.VariantName, relativeDiff)
	}
}

type VariantAnalytics struct {
	VariantID     uint        `json:"variant_id"`
	VariantName   string      `json:"variant_name"`
	Period        string      `json:"period"`
	DailyData     []DailyStat `json:"daily_data"`
	HourlyData    []HourlyStat `json:"hourly_data"`
	Summary       AnalyticsSummary `json:"summary"`
}

type DailyStat struct {
	Date         string  `json:"date"`
	Visitors     int64   `json:"visitors"`
	Conversions  int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
}

type HourlyStat struct {
	Hour          int     `json:"hour"`
	Visitors      int64   `json:"visitors"`
	Conversions   int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
}

type AnalyticsSummary struct {
	AvgVisitors      float64 `json:"avg_visitors"`
	AvgConversions   float64 `json:"avg_conversions"`
	AvgConversionRate float64 `json:"avg_conversion_rate"`
	Trend            string  `json:"trend"`
	TrendPercent     float64 `json:"trend_percent"`
}

func (s *ABTestService) GetVariantAnalytics(testID, variantID uint, period string) (*VariantAnalytics, error) {
	test, err := s.GetABTestByID(testID)
	if err != nil {
		return nil, err
	}

	var variant *models.ABTestVariant
	for _, v := range test.Variants {
		if v.ID == variantID {
			variant = &v
			break
		}
	}

	if variant == nil {
		return nil, ErrVariantNotFound
	}

	days := s.parsePeriod(period)
	startDate := time.Now().AddDate(0, 0, -days)

	dailyData := make([]DailyStat, 0)
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		var visitors int64
		database.DB.Model(&models.ABTestAssignment{}).
			Where("ab_test_id = ? AND variant_id = ? AND DATE(assigned_at) = ?", testID, variantID, dateStr).
			Count(&visitors)

		var conversions int64
		database.DB.Model(&models.ABTestEvent{}).
			Where("ab_test_id = ? AND variant_id = ? AND is_conversion = ? AND DATE(timestamp) = ?", testID, variantID, true, dateStr).
			Count(&conversions)

		conversionRate := 0.0
		if visitors > 0 {
			conversionRate = float64(conversions) / float64(visitors) * 100
		}

		dailyData = append(dailyData, DailyStat{
			Date:          dateStr,
			Visitors:      visitors,
			Conversions:   conversions,
			ConversionRate: conversionRate,
		})
	}

	hourlyData := make([]HourlyStat, 0)
	for hour := 0; hour < 24; hour++ {
		var visitors int64
		database.DB.Model(&models.ABTestAssignment{}).
			Where("ab_test_id = ? AND variant_id = ? AND EXTRACT(HOUR FROM assigned_at) = ?", testID, variantID, hour).
			Count(&visitors)

		var conversions int64
		database.DB.Model(&models.ABTestEvent{}).
			Where("ab_test_id = ? AND variant_id = ? AND is_conversion = ? AND EXTRACT(HOUR FROM timestamp) = ?", testID, variantID, true, hour).
			Count(&conversions)

		conversionRate := 0.0
		if visitors > 0 {
			conversionRate = float64(conversions) / float64(visitors) * 100
		}

		hourlyData = append(hourlyData, HourlyStat{
			Hour:           hour,
			Visitors:       visitors,
			Conversions:    conversions,
			ConversionRate: conversionRate,
		})
	}

	summary := s.calculateAnalyticsSummary(dailyData)

	return &VariantAnalytics{
		VariantID:   variantID,
		VariantName: variant.Name,
		Period:      period,
		DailyData:   dailyData,
		HourlyData:  hourlyData,
		Summary:     summary,
	}, nil
}

func (s *ABTestService) parsePeriod(period string) int {
	switch period {
	case "7d":
		return 7
	case "14d":
		return 14
	case "30d":
		return 30
	default:
		return 7
	}
}

func (s *ABTestService) calculateAnalyticsSummary(dailyData []DailyStat) AnalyticsSummary {
	if len(dailyData) == 0 {
		return AnalyticsSummary{}
	}

	var totalVisitors, totalConversions int64
	var totalRate float64

	for _, d := range dailyData {
		totalVisitors += d.Visitors
		totalConversions += d.Conversions
		totalRate += d.ConversionRate
	}

	avgVisitors := float64(totalVisitors) / float64(len(dailyData))
	avgConversions := float64(totalConversions) / float64(len(dailyData))
	avgConversionRate := totalRate / float64(len(dailyData))

	trend := "stable"
	trendPercent := 0.0

	if len(dailyData) >= 2 {
		firstHalf := dailyData[:len(dailyData)/2]
		secondHalf := dailyData[len(dailyData)/2:]

		var firstAvg, secondAvg float64
		for _, d := range firstHalf {
			firstAvg += d.ConversionRate
		}
		firstAvg /= float64(len(firstHalf))

		for _, d := range secondHalf {
			secondAvg += d.ConversionRate
		}
		secondAvg /= float64(len(secondHalf))

		if firstAvg > 0 {
			trendPercent = ((secondAvg - firstAvg) / firstAvg) * 100
		}

		if trendPercent > 5 {
			trend = "improving"
		} else if trendPercent < -5 {
			trend = "declining"
		}
	}

	return AnalyticsSummary{
		AvgVisitors:       avgVisitors,
		AvgConversions:    avgConversions,
		AvgConversionRate: avgConversionRate,
		Trend:             trend,
		TrendPercent:      trendPercent,
	}
}

type TestRecommendation struct {
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Priority int    `json:"priority"`
	Impact  string  `json:"impact"`
}

func (s *ABTestService) GetTestRecommendations(testID uint) ([]TestRecommendation, error) {
	test, err := s.GetABTestByID(testID)
	if err != nil {
		return nil, err
	}

	report, err := s.GetTestReport(testID)
	if err != nil {
		return nil, err
	}

	recommendations := make([]TestRecommendation, 0)

	recommendations = append(recommendations, TestRecommendation{
		Type:    "data_quality",
		Title:   "数据质量检查",
		Content: s.generateDataQualityRecommendation(report),
		Priority: 1,
		Impact:  "high",
	})

	if test.Status == "running" {
		recommendations = append(recommendations, TestRecommendation{
			Type:    "timing",
			Title:   "测试时长建议",
			Content: s.generateTimingRecommendation(report),
			Priority: 2,
			Impact:  "medium",
		})
	}

	recommendations = append(recommendations, TestRecommendation{
		Type:    "statistical",
		Title:   "统计分析建议",
		Content: s.generateStatisticalRecommendation(report),
		Priority: 3,
		Impact:  "high",
	})

	if test.Status == "stopped" {
		recommendations = append(recommendations, TestRecommendation{
			Type:    "next_steps",
			Title:   "后续步骤",
			Content: s.generateNextStepsRecommendation(report),
			Priority: 4,
			Impact:  "medium",
		})
	}

	return recommendations, nil
}

func (s *ABTestService) generateDataQualityRecommendation(report *TestReport) string {
	var totalVisitors int64
	for _, v := range report.Variants {
		totalVisitors += v.Visitors
	}

	if totalVisitors < 1000 {
		return fmt.Sprintf("当前总访客数 %d 较少，建议继续收集数据至少达到1000个访客以确保结果可靠性", totalVisitors)
	}

	return fmt.Sprintf("数据质量良好，当前已收集 %d 个访客，数据量充足", totalVisitors)
}

func (s *ABTestService) generateTimingRecommendation(report *TestReport) string {
	var minVisitors int64 = math.MaxInt64
	for _, v := range report.Variants {
		if v.Visitors < minVisitors {
			minVisitors = v.Visitors
		}
	}

	if minVisitors < 1000 {
		return fmt.Sprintf("建议继续运行测试至每个变体至少1000个访客，当前最少变体访客数为 %d", minVisitors)
	}

	hasWinner := false
	for _, v := range report.Variants {
		if !v.IsControl && v.Confidence > 95 {
			hasWinner = true
			break
		}
	}

	if hasWinner {
		return "已检测到统计显著的优胜者，建议考虑结束测试或按预期时长继续运行"
	}

	return "测试正在进行中，建议保持当前状态继续收集数据"
}

func (s *ABTestService) generateStatisticalRecommendation(report *TestReport) string {
	var highConfidenceVariants int
	for _, v := range report.Variants {
		if v.Confidence > 95 {
			highConfidenceVariants++
		}
	}

	if highConfidenceVariants > 0 {
		return fmt.Sprintf("有 %d 个变体达到95%%置信度，统计结果可靠", highConfidenceVariants)
	}

	var maxConfidence float64
	for _, v := range report.Variants {
		if v.Confidence > maxConfidence {
			maxConfidence = v.Confidence
		}
	}

	return fmt.Sprintf("当前最高置信度为 %.1f%%，建议继续收集数据以达到95%%的统计显著性阈值", maxConfidence)
}

func (s *ABTestService) generateNextStepsRecommendation(report *TestReport) string {
	if report.WinningVariant != nil {
		for _, v := range report.Variants {
			if v.VariantID == *report.WinningVariant {
				return fmt.Sprintf("建议将 '%s' 作为正式方案部署，并考虑设计后续优化测试", v.VariantName)
			}
		}
	}

	return "建议分析所有变体数据，识别潜在优化方向，设计下一轮A/B测试"
}
