package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type ImageCategory string

const (
	CategoryAnimal     ImageCategory = "animal"
	CategoryVehicle    ImageCategory = "vehicle"
	CategoryFood       ImageCategory = "food"
	CategoryBuilding   ImageCategory = "building"
	CategoryNature     ImageCategory = "nature"
	CategoryObject     ImageCategory = "object"
	CategoryPerson     ImageCategory = "person"
	CategoryScenery    ImageCategory = "scenery"
)

type SemanticImage struct {
	ID         string        `json:"id"`
	Category   ImageCategory `json:"category"`
	URL        string        `json:"url"`
	Base64Data string        `json:"base64_data"`
	Labels     []string      `json:"labels"`
	Keywords   []string      `json:"keywords"`
	Difficulty string        `json:"difficulty"`
}

type SemanticPuzzle struct {
	Images          []SemanticImage    `json:"images"`
	Question        string            `json:"question"`
	CorrectAnswer   string            `json:"correct_answer"`
	Options         []string          `json:"options"`
	Category       ImageCategory     `json:"category"`
	Difficulty      string            `json:"difficulty"`
	AnalysisType    string            `json:"analysis_type"`
	TimeLimit       int               `json:"time_limit"`
}

type CreateSemanticRequest struct {
	Difficulty  string `json:"difficulty"`
	Category    string `json:"category"`
	AnalysisType string `json:"analysis_type"`
	ImageCount  int    `json:"image_count"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateSemanticResponse struct {
	SessionID     string         `json:"session_id"`
	Puzzle        *SemanticPuzzle `json:"puzzle"`
	ExpiresIn     int64          `json:"expires_in"`
	ExpiresAt     int64          `json:"expires_at"`
}

type VerifySemanticRequest struct {
	SessionID      string   `json:"session_id" binding:"required"`
	Answer         string   `json:"answer" binding:"required"`
	AnswerIndex    int      `json:"answer_index"`
	ConfidenceScore float64  `json:"confidence_score"`
	ResponseTime   int64    `json:"response_time"`
	AnalysisMethod string   `json:"analysis_method"`
	Keywords       []string `json:"keywords"`
	RiskScore      float64  `json:"risk_score"`
}

type VerifySemanticResult struct {
	Success          bool    `json:"success"`
	Message          string  `json:"message"`
	Score            float64 `json:"score"`
	CorrectAnswer    string  `json:"correct_answer"`
	ConfidenceScore  float64 `json:"confidence_score"`
	SemanticScore    float64 `json:"semantic_score"`
	TimeBonus        float64 `json:"time_bonus"`
	AnalysisFeedback string  `json:"analysis_feedback"`
}

type SemanticGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type SemanticVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var imageDatabase = map[ImageCategory][]SemanticImage{
	CategoryAnimal: {
		{ID: "cat", Category: CategoryAnimal, Labels: []string{"猫", "动物", "哺乳动物"}, Keywords: []string{"宠物", "小型", "毛茸茸"}},
		{ID: "dog", Category: CategoryAnimal, Labels: []string{"狗", "动物", "哺乳动物"}, Keywords: []string{"忠诚", "宠物", "家养"}},
		{ID: "bird", Category: CategoryAnimal, Labels: []string{"鸟", "动物", "飞禽"}, Keywords: []string{"飞行", "羽毛", "鸣叫"}},
		{ID: "fish", Category: CategoryAnimal, Labels: []string{"鱼", "动物", "水生"}, Keywords: []string{"游泳", "海洋", "鳞片"}},
		{ID: "horse", Category: CategoryAnimal, Labels: []string{"马", "动物", "哺乳动物"}, Keywords: []string{"奔跑", "骑乘", "草食"}},
		{ID: "elephant", Category: CategoryAnimal, Labels: []string{"大象", "动物", "哺乳动物"}, Keywords: []string{"庞大", "长鼻子", "非洲"}},
		{ID: "lion", Category: CategoryAnimal, Labels: []string{"狮子", "动物", "猫科"}, Keywords: []string{"凶猛", "森林之王", "鬃毛"}},
		{ID: "rabbit", Category: CategoryAnimal, Labels: []string{"兔子", "动物", "小型"}, Keywords: []string{"可爱", "长耳朵", "跳跃"}},
	},
	CategoryVehicle: {
		{ID: "car", Category: CategoryVehicle, Labels: []string{"汽车", "交通工具", "四轮"}, Keywords: []string{"出行", "代步", "汽油"}},
		{ID: "bus", Category: CategoryVehicle, Labels: []string{"公交车", "交通工具", "公共"}, Keywords: []string{"乘客", "城市", "公共交通"}},
		{ID: "bicycle", Category: CategoryVehicle, Labels: []string{"自行车", "交通工具", "两轮"}, Keywords: []string{"环保", "健身", "人力"}},
		{ID: "airplane", Category: CategoryVehicle, Labels: []string{"飞机", "交通工具", "航空"}, Keywords: []string{"飞行", "天空", "旅行"}},
		{ID: "train", Category: CategoryVehicle, Labels: []string{"火车", "交通工具", "铁路"}, Keywords: []string{"轨道", "高速", "运输"}},
		{ID: "ship", Category: CategoryVehicle, Labels: []string{"轮船", "交通工具", "水上"}, Keywords: []string{"航海", "海洋", "漂浮"}},
	},
	CategoryFood: {
		{ID: "pizza", Category: CategoryFood, Labels: []string{"披萨", "食物", "意大利"}, Keywords: []string{"圆形", "奶酪", "烤制"}},
		{ID: "hamburger", Category: CategoryFood, Labels: []string{"汉堡", "食物", "快餐"}, Keywords: []string{"面包", "肉饼", "夹心"}},
		{ID: "sushi", Category: CategoryFood, Labels: []string{"寿司", "食物", "日本"}, Keywords: []string{"米饭", "生鱼片", "海苔"}},
		{ID: "noodles", Category: CategoryFood, Labels: []string{"面条", "食物", "主食"}, Keywords: []string{"面条", "汤", "中式"}},
		{ID: "icecream", Category: CategoryFood, Labels: []string{"冰淇淋", "甜点", "冷饮"}, Keywords: []string{"甜", "冷冻", "奶油"}},
		{ID: "cake", Category: CategoryFood, Labels: []string{"蛋糕", "甜点", "烘焙"}, Keywords: []string{"生日", "奶油", "甜食"}},
	},
	CategoryNature: {
		{ID: "mountain", Category: CategoryNature, Labels: []string{"山", "自然", "地形"}, Keywords: []string{"高耸", "岩石", "攀登"}},
		{ID: "ocean", Category: CategoryNature, Labels: []string{"海洋", "自然", "水域"}, Keywords: []string{"蓝色", "广阔", "波浪"}},
		{ID: "forest", Category: CategoryNature, Labels: []string{"森林", "自然", "植被"}, Keywords: []string{"树木", "绿色", "生态"}},
		{ID: "desert", Category: CategoryNature, Labels: []string{"沙漠", "自然", "地形"}, Keywords: []string{"干旱", "沙丘", "仙人掌"}},
		{ID: "waterfall", Category: CategoryNature, Labels: []string{"瀑布", "自然", "水体"}, Keywords: []string{"水流", "落差", "轰鸣"}},
		{ID: "sunset", Category: CategoryNature, Labels: []string{"日落", "自然", "天文"}, Keywords: []string{"橙色", "黄昏", "天空"}},
	},
	CategoryBuilding: {
		{ID: "skyscraper", Category: CategoryBuilding, Labels: []string{"摩天大楼", "建筑", "高层"}, Keywords: []string{"高耸", "现代", "玻璃"}},
		{ID: "house", Category: CategoryBuilding, Labels: []string{"房子", "建筑", "住宅"}, Keywords: []string{"居住", "家庭", "温馨"}},
		{ID: "temple", Category: CategoryBuilding, Labels: []string{"寺庙", "建筑", "宗教"}, Keywords: []string{"宗教", "历史", "祈祷"}},
		{ID: "bridge", Category: CategoryBuilding, Labels: []string{"桥梁", "建筑", "交通"}, Keywords: []string{"跨越", "结构", "河流"}},
		{ID: "castle", Category: CategoryBuilding, Labels: []string{"城堡", "建筑", "历史"}, Keywords: []string{"欧洲", "防御", "王子"}},
	},
}

var questionTemplates = map[string][]string{
	"category": {
		"这张图片中的主要对象属于哪个类别？",
		"图片中的物体属于什么类型？",
		"以下哪项最准确地描述了图片中的内容？",
	},
	"description": {
		"请用一句话描述这张图片",
		"这张图片展示的是什么场景？",
		"图片中的主要内容是什么？",
	},
	"counting": {
		"图片中出现了几个同类对象？",
		"图片中总共有多少个%s？",
		"你能数出图片中有多少%s吗？",
	},
	"color": {
		"图片中的主要物体是什么颜色？",
		"这张图片的主色调是什么？",
		"图片中哪个颜色最突出？",
	},
	"action": {
		"图片中的主体在做什么？",
		"图中的人物/动物正在做什么？",
		"这个场景展示的是什么动作？",
	},
}

func NewSemanticGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *SemanticGeneratorService {
	return &SemanticGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewSemanticVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *SemanticVerifierService {
	return &SemanticVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *SemanticGeneratorService) Create(ctx context.Context, req *CreateSemanticRequest) (*CreateSemanticResponse, error) {
	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	category := ImageCategory(req.Category)
	if category == "" {
		categories := []ImageCategory{CategoryAnimal, CategoryVehicle, CategoryFood, CategoryNature, CategoryBuilding}
		category = categories[rand.Intn(len(categories))]
	}

	analysisType := req.AnalysisType
	if analysisType == "" {
		types := []string{"category", "description", "counting", "color", "action"}
		analysisType = types[rand.Intn(len(types))]
	}

	imageCount := req.ImageCount
	if imageCount <= 0 {
		imageCount = 4
	}
	if imageCount > 9 {
		imageCount = 9
	}

	puzzle := s.generateSemanticPuzzle(category, analysisType, difficulty, imageCount)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	puzzleData, err := json.Marshal(puzzle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal puzzle: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(puzzleData),
		SliderURL:     string(puzzleData),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	return &CreateSemanticResponse{
		SessionID: sessionID,
		Puzzle:    puzzle,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *SemanticGeneratorService) generateSemanticPuzzle(category ImageCategory, analysisType, difficulty string, imageCount int) *SemanticPuzzle {
	rand.Seed(time.Now().UnixNano())

	categoryImages, ok := imageDatabase[category]
	if !ok || len(categoryImages) == 0 {
		categoryImages = imageDatabase[CategoryAnimal]
	}

	selectedImages := make([]SemanticImage, 0, imageCount)
	usedIDs := make(map[string]bool)

	for len(selectedImages) < imageCount && len(selectedImages) < len(categoryImages) {
		img := categoryImages[rand.Intn(len(categoryImages))]
		if !usedIDs[img.ID] {
			usedIDs[img.ID] = true
			selectedImages = append(selectedImages, SemanticImage{
				ID:         img.ID,
				Category:   img.Category,
				Labels:     img.Labels,
				Keywords:   img.Keywords,
				Difficulty: difficulty,
			})
		}
	}

	correctAnswer := s.getCorrectAnswer(selectedImages, analysisType)
	options := s.generateOptions(analysisType, correctAnswer, category)

	question := s.generateQuestion(analysisType, category)

	timeLimit := s.getTimeLimit(difficulty)

	return &SemanticPuzzle{
		Images:        selectedImages,
		Question:      question,
		CorrectAnswer: correctAnswer,
		Options:       options,
		Category:      category,
		Difficulty:    difficulty,
		AnalysisType:  analysisType,
		TimeLimit:     timeLimit,
	}
}

func (s *SemanticGeneratorService) getCorrectAnswer(images []SemanticImage, analysisType string) string {
	if len(images) == 0 {
		return ""
	}

	img := images[rand.Intn(len(images))]

	switch analysisType {
	case "category":
		return string(img.Category)
	case "description":
		if len(img.Labels) > 0 {
			return img.Labels[0]
		}
		return img.ID
	case "counting":
		count := len(images)
		return fmt.Sprintf("%d", count)
	case "color":
		if len(img.Keywords) > 0 {
			for _, kw := range img.Keywords {
				if isColorKeyword(kw) {
					return kw
				}
			}
		}
		return "自然色"
	case "action":
		if len(img.Keywords) > 0 {
			for _, kw := range img.Keywords {
				if isActionKeyword(kw) {
					return kw
				}
			}
		}
		return "静止"
	default:
		if len(img.Labels) > 0 {
			return img.Labels[0]
		}
		return img.ID
	}
}

func (s *SemanticGeneratorService) generateOptions(analysisType, correctAnswer string, category ImageCategory) []string {
	options := make([]string, 0, 4)
	options = append(options, correctAnswer)

	categoryImages, _ := imageDatabase[category]
	otherLabels := make([]string, 0)

	for _, img := range categoryImages {
		for _, label := range img.Labels {
			if label != correctAnswer && !containsString(options, label) {
				otherLabels = append(otherLabels, label)
			}
		}
		for _, kw := range img.Keywords {
			if kw != correctAnswer && !containsString(options, kw) && !containsString(otherLabels, kw) {
				otherLabels = append(otherLabels, kw)
			}
		}
	}

	rand.Shuffle(len(otherLabels), func(i, j int) {
		otherLabels[i], otherLabels[j] = otherLabels[j], otherLabels[i]
	})

	for _, label := range otherLabels {
		if len(options) >= 4 {
			break
		}
		options = append(options, label)
	}

	for len(options) < 4 {
		options = append(options, fmt.Sprintf("选项%d", len(options)+1))
	}

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	return options
}

func (s *SemanticGeneratorService) generateQuestion(analysisType string, category ImageCategory) string {
	templates, ok := questionTemplates[analysisType]
	if !ok {
		templates = questionTemplates["category"]
	}

	question := templates[rand.Intn(len(templates))]

	categoryName := getCategoryChineseName(category)
	question = strings.ReplaceAll(question, "%s", categoryName)

	return question
}

func (s *SemanticGeneratorService) getTimeLimit(difficulty string) int {
	switch difficulty {
	case "easy":
		return 60
	case "medium":
		return 45
	case "hard":
		return 30
	case "expert":
		return 20
	default:
		return 45
	}
}

func (s *SemanticVerifierService) Verify(ctx context.Context, req *VerifySemanticRequest) (*VerifySemanticResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifySemanticResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifySemanticResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifySemanticResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var puzzle SemanticPuzzle
	if err := json.Unmarshal([]byte(session.BackgroundURL), &puzzle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle: %w", err)
	}

	if session.Status == "verified" {
		return &VerifySemanticResult{
			Success:         true,
			Message:         "验证码已验证通过",
			Score:           100,
			CorrectAnswer:   puzzle.CorrectAnswer,
			ConfidenceScore: 100,
			SemanticScore:   100,
			TimeBonus:       0,
		}, nil
	}

	isCorrect := s.checkAnswer(&puzzle, req.Answer, req.AnswerIndex)

	confidenceScore := s.calculateConfidenceScore(&puzzle, req)

	semanticScore := s.calculateSemanticScore(&puzzle, req)

	timeBonus := s.calculateTimeBonus(&puzzle, req.ResponseTime)

	totalScore := 0.0
	if isCorrect {
		totalScore = 70 + semanticScore*20 + timeBonus
	} else {
		totalScore = semanticScore * 50
	}
	totalScore = math.Min(100, totalScore)

	isSuccess := isCorrect && confidenceScore >= 0.5 && semanticScore >= 0.3

	if isSuccess {
		session.Status = "verified"
		if s.sessionCache != nil {
			_ = s.sessionCache.UpdateStatus(ctx, req.SessionID, "verified")
		}
		if s.captchaRepo != nil {
			_ = s.captchaRepo.UpdateStatus(req.SessionID, "verified")
		}
	}

	return &VerifySemanticResult{
		Success:         isSuccess,
		Message:         func() string {
			if isSuccess {
				return "语义验证成功"
			}
			return fmt.Sprintf("验证失败，正确答案是：%s", puzzle.CorrectAnswer)
		}(),
		Score:            totalScore,
		CorrectAnswer:    puzzle.CorrectAnswer,
		ConfidenceScore:  confidenceScore * 100,
		SemanticScore:    semanticScore * 100,
		TimeBonus:        timeBonus,
		AnalysisFeedback: s.generateFeedback(&puzzle, req),
	}, nil
}

func (s *SemanticVerifierService) checkAnswer(puzzle *SemanticPuzzle, answer string, answerIndex int) bool {
	if strings.TrimSpace(strings.ToLower(answer)) == strings.TrimSpace(strings.ToLower(puzzle.CorrectAnswer)) {
		return true
	}

	if answerIndex >= 0 && answerIndex < len(puzzle.Options) {
		if strings.TrimSpace(strings.ToLower(puzzle.Options[answerIndex])) == strings.TrimSpace(strings.ToLower(puzzle.CorrectAnswer)) {
			return true
		}
	}

	return false
}

func (s *SemanticVerifierService) calculateConfidenceScore(puzzle *SemanticPuzzle, req *VerifySemanticRequest) float64 {
	confidence := req.ConfidenceScore

	if confidence == 0 {
		confidence = 0.5
	}

	if len(req.Keywords) > 0 {
		matchCount := 0
		for _, userKw := range req.Keywords {
			for _, label := range puzzle.Images[0].Labels {
				if containsString(puzzle.Images[0].Keywords, userKw) || userKw == label {
					matchCount++
					break
				}
			}
		}
		keywordMatchRatio := float64(matchCount) / float64(len(req.Keywords))
		confidence = (confidence + keywordMatchRatio) / 2
	}

	return math.Min(1, math.Max(0, confidence))
}

func (s *SemanticVerifierService) calculateSemanticScore(puzzle *SemanticPuzzle, req *VerifySemanticRequest) float64 {
	if len(req.Keywords) == 0 {
		return 0.5
	}

	imageLabels := make([]string, 0)
	for _, img := range puzzle.Images {
		imageLabels = append(imageLabels, img.Labels...)
		imageLabels = append(imageLabels, img.Keywords...)
	}

	matchCount := 0
	for _, userKw := range req.Keywords {
		for _, label := range imageLabels {
			if strings.Contains(strings.ToLower(label), strings.ToLower(userKw)) ||
				strings.Contains(strings.ToLower(userKw), strings.ToLower(label)) {
				matchCount++
				break
			}
		}
	}

	return float64(matchCount) / float64(len(req.Keywords))
}

func (s *SemanticVerifierService) calculateTimeBonus(puzzle *SemanticPuzzle, responseTime int64) float64 {
	if responseTime <= 0 {
		return 0
	}

	timeLimitMs := int64(puzzle.TimeLimit * 1000)

	if responseTime < timeLimitMs/3 {
		return 10
	} else if responseTime < timeLimitMs/2 {
		return 7
	} else if responseTime < timeLimitMs {
		return 5
	} else if responseTime < timeLimitMs*2 {
		return 2
	}

	return 0
}

func (s *SemanticVerifierService) generateFeedback(puzzle *SemanticPuzzle, req *VerifySemanticRequest) string {
	if len(puzzle.Images) == 0 {
		return "无法分析，请重试"
	}

	img := puzzle.Images[0]

	switch puzzle.AnalysisType {
	case "category":
		return fmt.Sprintf("提示：图片中的主要对象是%s，属于%s类别", img.Labels[0], string(img.Category))
	case "description":
		return fmt.Sprintf("提示：图片展示的是%s", strings.Join(img.Labels, "、"))
	case "counting":
		return fmt.Sprintf("提示：图片中有%d个相关对象", len(puzzle.Images))
	case "color":
		return fmt.Sprintf("提示：图片的主要颜色是%s", strings.Join(img.Keywords, "或"))
	case "action":
		return fmt.Sprintf("提示：图片中%s", img.Keywords[0])
	default:
		return fmt.Sprintf("提示：关键词包括：%s", strings.Join(img.Labels, "、"))
	}
}

func (s *SemanticVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *SemanticVerifierService) incrementVerifyCount(sessionID string) {
	if s.sessionCache != nil {
		_ = s.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if s.captchaRepo != nil {
		_ = s.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (s *SemanticVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return s.getSession(sessionID)
}

func (s *SemanticVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isColorKeyword(word string) bool {
	colors := []string{"红色", "蓝色", "绿色", "黄色", "橙色", "紫色", "粉色", "黑色", "白色", "灰色", "棕色", "金色", "银色"}
	for _, c := range colors {
		if strings.Contains(word, c) {
			return true
		}
	}
	return false
}

func isActionKeyword(word string) bool {
	actions := []string{"站立", "行走", "奔跑", "飞行", "游泳", "跳跃", "进食", "睡眠", "玩耍", "工作"}
	for _, a := range actions {
		if strings.Contains(word, a) {
			return true
		}
	}
	return false
}

func getCategoryChineseName(category ImageCategory) string {
	names := map[ImageCategory]string{
		CategoryAnimal:   "动物",
		CategoryVehicle:  "交通工具",
		CategoryFood:     "食物",
		CategoryBuilding: "建筑物",
		CategoryNature:   "自然风景",
		CategoryObject:   "物品",
		CategoryPerson:   "人物",
		CategoryScenery:  "场景",
	}
	if name, ok := names[category]; ok {
		return name
	}
	return "物体"
}
