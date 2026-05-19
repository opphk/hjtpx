package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ============================================
// GPT驱动的智能验证码生成服务
// ============================================

type GPTCaptchaService struct {
	aiPromptGenerator *AIPromptGenerator
	captchaGenerator  *SmartCaptchaGenerator
}

type AICaptchaRequest struct {
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id"`
	Difficulty  string `json:"difficulty"` // easy, medium, hard, expert
	ContentType string `json:"content_type"` // text, math, image, audio, semantic
	RiskLevel   float64 `json:"risk_level"`
}

type AICaptchaResponse struct {
	SessionID        string                 `json:"session_id"`
	Type             string                 `json:"type"`
	Challenge        string                 `json:"challenge"`
	Options          []AICaptchaOption      `json:"options,omitempty"`
	ImageBase64      string                 `json:"image_base64,omitempty"`
	AudioBase64      string                 `json:"audio_base64,omitempty"`
	ExpiresAt        int64                  `json:"expires_at"`
	Difficulty       string                 `json:"difficulty"`
	ExpectedAnswer   string                 `json:"-"`
	ValidationConfig AICaptchaValidation    `json:"-"`
}

type AICaptchaOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	IsCorrect bool `json:"-"`
}

type AICaptchaValidation struct {
	AllowTolerance bool    `json:"allow_tolerance"`
	Tolerance      float64 `json:"tolerance"`
	MaxAttempts    int     `json:"max_attempts"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

func NewGPTCaptchaService() *GPTCaptchaService {
	return &GPTCaptchaService{
		aiPromptGenerator: NewAIPromptGenerator(),
		captchaGenerator:  NewSmartCaptchaGenerator(),
	}
}

func (s *GPTCaptchaService) GenerateCaptcha(ctx context.Context, req *AICaptchaRequest) (*AICaptchaResponse, error) {
	switch req.ContentType {
	case "text":
		return s.generateTextCaptcha(req)
	case "math":
		return s.generateMathCaptcha(req)
	case "image":
		return s.generateImageCaptcha(req)
	case "audio":
		return s.generateAudioCaptcha(req)
	case "semantic":
		return s.generateSemanticCaptcha(req)
	default:
		return s.generateSmartCaptcha(ctx, req)
	}
}

func (s *GPTCaptchaService) generateTextCaptcha(req *AICaptchaRequest) (*AICaptchaResponse, error) {
	prompt := s.aiPromptGenerator.GenerateTextChallenge(req.Difficulty, req.RiskLevel)
	challenge := prompt.Challenge
	answer := prompt.ExpectedAnswer

	return &AICaptchaResponse{
		SessionID:      req.SessionID,
		Type:           "text",
		Challenge:      challenge,
		ExpectedAnswer: answer,
		ExpiresAt:      time.Now().Add(120 * time.Second).Unix(),
		Difficulty:     req.Difficulty,
		ValidationConfig: AICaptchaValidation{
			AllowTolerance: false,
			Tolerance:      0,
			MaxAttempts:    3,
			TimeoutSeconds: 120,
		},
	}, nil
}

func (s *GPTCaptchaService) generateMathCaptcha(req *AICaptchaRequest) (*AICaptchaResponse, error) {
	mathProblem := s.aiPromptGenerator.GenerateMathChallenge(req.Difficulty)

	return &AICaptchaResponse{
		SessionID:      req.SessionID,
		Type:           "math",
		Challenge:      mathProblem.Question,
		ExpectedAnswer: mathProblem.Answer,
		ExpiresAt:      time.Now().Add(120 * time.Second).Unix(),
		Difficulty:     req.Difficulty,
		ValidationConfig: AICaptchaValidation{
			AllowTolerance: true,
			Tolerance:      0.01,
			MaxAttempts:    3,
			TimeoutSeconds: 120,
		},
	}, nil
}

func (s *GPTCaptchaService) generateImageCaptcha(req *AICaptchaRequest) (*AICaptchaResponse, error) {
	imageChallenge := s.captchaGenerator.GenerateImageChallenge(req.Difficulty, fmt.Sprintf("%.2f", req.RiskLevel))

	return &AICaptchaResponse{
		SessionID:      req.SessionID,
		Type:           "image",
		Challenge:      imageChallenge.Instruction,
		Options:        imageChallenge.Options,
		ImageBase64:    imageChallenge.ImageBase64,
		ExpectedAnswer: imageChallenge.CorrectAnswer,
		ExpiresAt:      time.Now().Add(120 * time.Second).Unix(),
		Difficulty:     req.Difficulty,
		ValidationConfig: AICaptchaValidation{
			AllowTolerance: true,
			Tolerance:      0.1,
			MaxAttempts:    3,
			TimeoutSeconds: 120,
		},
	}, nil
}

func (s *GPTCaptchaService) generateAudioCaptcha(req *AICaptchaRequest) (*AICaptchaResponse, error) {
	audioChallenge := s.captchaGenerator.GenerateAudioChallenge(req.Difficulty)

	return &AICaptchaResponse{
		SessionID:      req.SessionID,
		Type:           "audio",
		Challenge:      "请听音频并输入听到的数字",
		AudioBase64:    audioChallenge.AudioBase64,
		ExpectedAnswer: audioChallenge.Code,
		ExpiresAt:      time.Now().Add(120 * time.Second).Unix(),
		Difficulty:     req.Difficulty,
		ValidationConfig: AICaptchaValidation{
			AllowTolerance: true,
			Tolerance:      0.15,
			MaxAttempts:    5,
			TimeoutSeconds: 180,
		},
	}, nil
}

func (s *GPTCaptchaService) generateSemanticCaptcha(req *AICaptchaRequest) (*AICaptchaResponse, error) {
	semanticChallenge := s.aiPromptGenerator.GenerateSemanticChallenge(req.Difficulty)

	return &AICaptchaResponse{
		SessionID:      req.SessionID,
		Type:           "semantic",
		Challenge:      semanticChallenge.Question,
		Options:        semanticChallenge.Options,
		ExpectedAnswer: semanticChallenge.CorrectAnswer,
		ExpiresAt:      time.Now().Add(180 * time.Second).Unix(),
		Difficulty:     req.Difficulty,
		ValidationConfig: AICaptchaValidation{
			AllowTolerance: false,
			Tolerance:      0,
			MaxAttempts:    3,
			TimeoutSeconds: 180,
		},
	}, nil
}

func (s *GPTCaptchaService) generateSmartCaptcha(ctx context.Context, req *AICaptchaRequest) (*AICaptchaResponse, error) {
	captchaType := s.determineCaptchaType(fmt.Sprintf("%.2f", req.RiskLevel), req.Difficulty)
	req.ContentType = captchaType
	return s.GenerateCaptcha(ctx, req)
}

func (s *GPTCaptchaService) determineCaptchaType(riskLevel, difficulty string) string {
	risk := parseRiskLevel(riskLevel)
	
	switch {
	case risk >= 0.8:
		return "semantic"
	case risk >= 0.6:
		return "image"
	case risk >= 0.4:
		return "math"
	default:
		if difficulty == "hard" || difficulty == "expert" {
			return "semantic"
		}
		return "text"
	}
}

func (s *GPTCaptchaService) ValidateCaptcha(ctx context.Context, sessionID, userAnswer string, expectedAnswer string) (bool, error) {
	return strings.EqualFold(userAnswer, expectedAnswer), nil
}

// ============================================
// AI提示生成器
// ============================================

type AIPromptGenerator struct {
	textPatterns    map[string][]string
	mathOperators   map[string][]string
	semanticTopics  []string
}

type GeneratedPrompt struct {
	Challenge      string
	ExpectedAnswer string
}

type MathProblem struct {
	Question string
	Answer   string
}

type SemanticQuestion struct {
	Question      string
	Options       []AICaptchaOption
	CorrectAnswer string
}

func NewAIPromptGenerator() *AIPromptGenerator {
	return &AIPromptGenerator{
		textPatterns: map[string][]string{
			"easy": {
				"请输入以下单词: %s",
				"验证码: %s",
				"请输入验证码: %s",
			},
			"medium": {
				"请输入图片中的文字: %s",
				"识别图片中的字符: %s",
				"请输入下图中的验证码: %s",
			},
			"hard": {
				"请识别并输入图片中的扭曲文字: %s",
				"解析图片中的验证码字符: %s",
				"输入图片中显示的验证码: %s",
			},
			"expert": {
				"请仔细识别图片中的变形验证码: %s",
				"识别复杂背景中的验证码: %s",
			},
		},
		mathOperators: map[string][]string{
			"easy":   {"+", "-"},
			"medium": {"+", "-", "*"},
			"hard":   {"+", "-", "*", "/"},
			"expert": {"+", "-", "*", "/", "^"},
		},
		semanticTopics: []string{
			"常识问答",
			"逻辑推理",
			"图像描述",
			"语义理解",
			"上下文推理",
		},
	}
}

func (g *AIPromptGenerator) GenerateTextChallenge(difficulty string, riskLevel float64) *GeneratedPrompt {
	patterns := g.textPatterns[difficulty]
	if patterns == nil {
		patterns = g.textPatterns["medium"]
	}

	code := g.generateVerificationCode(difficulty, riskLevel)
	pattern := patterns[rand.Intn(len(patterns))]

	return &GeneratedPrompt{
		Challenge:      fmt.Sprintf(pattern, code),
		ExpectedAnswer: code,
	}
}

func (g *AIPromptGenerator) generateVerificationCode(difficulty string, riskLevel float64) string {
	length := 4
	switch difficulty {
	case "easy":
		length = 4
	case "medium":
		length = 5
	case "hard":
		length = 6
	case "expert":
		length = 7
	}

	chars := "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	if riskLevel >= 0.7 {
		chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func (g *AIPromptGenerator) GenerateMathChallenge(difficulty string) *MathProblem {
	operators := g.mathOperators[difficulty]
	if operators == nil {
		operators = g.mathOperators["medium"]
	}

	operator := operators[rand.Intn(len(operators))]
	var a, b int

	switch difficulty {
	case "easy":
		a = rand.Intn(50) + 1
		b = rand.Intn(50) + 1
	case "medium":
		a = rand.Intn(100) + 10
		b = rand.Intn(50) + 1
	case "hard":
		a = rand.Intn(200) + 50
		b = rand.Intn(100) + 10
	case "expert":
		a = rand.Intn(500) + 100
		b = rand.Intn(200) + 50
	default:
		a = rand.Intn(100) + 1
		b = rand.Intn(50) + 1
	}

	var answer int
	var question string

	switch operator {
	case "+":
		answer = a + b
		question = fmt.Sprintf("%d + %d = ?", a, b)
	case "-":
		if a < b {
			a, b = b, a
		}
		answer = a - b
		question = fmt.Sprintf("%d - %d = ?", a, b)
	case "*":
		a = rand.Intn(20) + 2
		b = rand.Intn(10) + 2
		answer = a * b
		question = fmt.Sprintf("%d × %d = ?", a, b)
	case "/":
		b = rand.Intn(10) + 2
		answer = rand.Intn(20) + 1
		a = b * answer
		question = fmt.Sprintf("%d ÷ %d = ?", a, b)
	case "^":
		base := rand.Intn(5) + 2
		exp := rand.Intn(3) + 2
		a = base
		answer = 1
		for i := 0; i < exp; i++ {
			answer *= base
		}
		question = fmt.Sprintf("%d^%d = ?", a, exp)
	}

	return &MathProblem{
		Question: question,
		Answer:   fmt.Sprintf("%d", answer),
	}
}

func (g *AIPromptGenerator) GenerateSemanticChallenge(difficulty string) *SemanticQuestion {
	questions := g.getSemanticQuestions(difficulty)
	question := questions[rand.Intn(len(questions))]

	return &SemanticQuestion{
		Question:      question.question,
		Options:       question.options,
		CorrectAnswer: question.correctAnswer,
	}
}

func (g *AIPromptGenerator) getSemanticQuestions(difficulty string) []struct {
	question      string
	options       []AICaptchaOption
	correctAnswer string
} {
	switch difficulty {
	case "easy":
		return []struct {
			question      string
			options       []AICaptchaOption
			correctAnswer string
		}{
			{
				question: "以下哪个是水果？",
				options: []AICaptchaOption{
					{ID: "a", Label: "苹果"},
					{ID: "b", Label: "桌子"},
					{ID: "c", Label: "椅子"},
					{ID: "d", Label: "书本"},
				},
				correctAnswer: "a",
			},
			{
				question: "以下哪个是动物？",
				options: []AICaptchaOption{
					{ID: "a", Label: "汽车"},
					{ID: "b", Label: "猫"},
					{ID: "c", Label: "房子"},
					{ID: "d", Label: "电脑"},
				},
				correctAnswer: "b",
			},
		}
	case "medium":
		return []struct {
			question      string
			options       []AICaptchaOption
			correctAnswer string
		}{
			{
				question: "如果所有的鸟都会飞，企鹅是鸟，那么企鹅会飞吗？",
				options: []AICaptchaOption{
					{ID: "a", Label: "会"},
					{ID: "b", Label: "不会"},
					{ID: "c", Label: "不确定"},
					{ID: "d", Label: "取决于种类"},
				},
				correctAnswer: "a",
			},
			{
				question: "小明有5个苹果，给了小红2个，又买了3个，现在有几个？",
				options: []AICaptchaOption{
					{ID: "a", Label: "4个"},
					{ID: "b", Label: "5个"},
					{ID: "c", Label: "6个"},
					{ID: "d", Label: "8个"},
				},
				correctAnswer: "c",
			},
		}
	case "hard":
		return []struct {
			question      string
			options       []AICaptchaOption
			correctAnswer string
		}{
			{
				question: "一个房间里有3个人，每个人都有2只手，每只手有5个手指，请问房间里总共有多少个手指？",
				options: []AICaptchaOption{
					{ID: "a", Label: "15"},
					{ID: "b", Label: "30"},
					{ID: "c", Label: "6"},
					{ID: "d", Label: "10"},
				},
				correctAnswer: "b",
			},
			{
				question: "以下哪个选项描述的是因果关系？",
				options: []AICaptchaOption{
					{ID: "a", Label: "今天下雨了，地面湿了"},
					{ID: "b", Label: "天空是蓝色的，草是绿色的"},
					{ID: "c", Label: "猫有尾巴，狗也有尾巴"},
					{ID: "d", Label: "太阳东升西落"},
				},
				correctAnswer: "a",
			},
		}
	default:
		return []struct {
			question      string
			options       []AICaptchaOption
			correctAnswer string
		}{
			{
				question: "以下哪个是水果？",
				options: []AICaptchaOption{
					{ID: "a", Label: "苹果"},
					{ID: "b", Label: "桌子"},
					{ID: "c", Label: "椅子"},
					{ID: "d", Label: "书本"},
				},
				correctAnswer: "a",
			},
		}
	}
}

// ============================================
// 智能验证码生成器
// ============================================

type SmartCaptchaGenerator struct{}

type ImageChallenge struct {
	Instruction    string
	ImageBase64    string
	Options        []AICaptchaOption
	CorrectAnswer  string
}

type AudioChallenge struct {
	AudioBase64 string
	Code        string
}

func NewSmartCaptchaGenerator() *SmartCaptchaGenerator {
	return &SmartCaptchaGenerator{}
}

func (g *SmartCaptchaGenerator) GenerateImageChallenge(difficulty, riskLevel string) *ImageChallenge {
	challenges := []struct {
		instruction string
		options     []AICaptchaOption
		correct     string
	}{
		{
			instruction: "请选择包含汽车的图片",
			options: []AICaptchaOption{
				{ID: "1", Label: "图片1"},
				{ID: "2", Label: "图片2"},
				{ID: "3", Label: "图片3"},
				{ID: "4", Label: "图片4"},
			},
			correct: "2",
		},
		{
			instruction: "请选择包含动物的图片",
			options: []AICaptchaOption{
				{ID: "1", Label: "图片1"},
				{ID: "2", Label: "图片2"},
				{ID: "3", Label: "图片3"},
				{ID: "4", Label: "图片4"},
			},
			correct: "3",
		},
		{
			instruction: "请选择包含植物的图片",
			options: []AICaptchaOption{
				{ID: "1", Label: "图片1"},
				{ID: "2", Label: "图片2"},
				{ID: "3", Label: "图片3"},
				{ID: "4", Label: "图片4"},
			},
			correct: "1",
		},
	}

	challenge := challenges[rand.Intn(len(challenges))]

	for i := range challenge.options {
		challenge.options[i].IsCorrect = challenge.options[i].ID == challenge.correct
	}

	return &ImageChallenge{
		Instruction:   challenge.instruction,
		Options:       challenge.options,
		CorrectAnswer: challenge.correct,
		ImageBase64:   g.generateDummyImage(),
	}
}

func (g *SmartCaptchaGenerator) GenerateAudioChallenge(difficulty string) *AudioChallenge {
	length := 4
	switch difficulty {
	case "hard", "expert":
		length = 6
	case "medium":
		length = 5
	}

	code := make([]byte, length)
	for i := range code {
		code[i] = byte('0' + rand.Intn(10))
	}

	return &AudioChallenge{
		AudioBase64: g.generateDummyAudio(),
		Code:        string(code),
	}
}

func (g *SmartCaptchaGenerator) generateDummyImage() string {
	return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAARklEQVR4Ae3UMQ6AIAwF0d/z/0b0J5J+AKWATWAS0ATWAvWALWA7WA3WALWALWALWA9WAMWAMWAMWA9WAMWAMWAMWA9WAMWAMWAMWA9WAMWAMWAMWA9WAMWAMWAMf8B8wBXQ4xwYg4gAAAABJRU5ErkJggg=="
}

func (g *SmartCaptchaGenerator) generateDummyAudio() string {
	return "data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2teleloqQkM9a2teleloqQkM9a2teleloqQkM9a2teleloqQkM="
}

func parseRiskLevel(risk string) float64 {
	switch risk {
	case "high":
		return 0.8
	case "medium":
		return 0.5
	case "low":
		return 0.2
	default:
		return 0.5
	}
}

func (s *GPTCaptchaService) ToJSON(response *AICaptchaResponse) ([]byte, error) {
	return json.Marshal(response)
}

func (s *GPTCaptchaService) FromJSON(data []byte) (*AICaptchaResponse, error) {
	var response AICaptchaResponse
	err := json.Unmarshal(data, &response)
	return &response, err
}