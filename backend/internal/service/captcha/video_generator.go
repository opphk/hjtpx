package captcha

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type VideoCaptchaType string

const (
	VideoTypeContent  VideoCaptchaType = "content"
	VideoTypeAction   VideoCaptchaType = "action"
	VideoTypeSequence VideoCaptchaType = "sequence"
)

type VideoGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type VideoCaptchaRequest struct {
	Type        VideoCaptchaType `json:"type"`
	Duration    int              `json:"duration"`     // 视频时长（秒）
	Language    string           `json:"language"`     // "zh-CN" or "en-US"
	ClientIP    string           `json:"client_ip"`
	UserAgent   string           `json:"user_agent"`
	Fingerprint string           `json:"fingerprint"`
}

type VideoCaptchaResponse struct {
	SessionID     string         `json:"session_id"`
	VideoData     string         `json:"video_data"`       // base64 encoded video
	VideoType     VideoCaptchaType `json:"video_type"`       // 验证码类型
	Question      string         `json:"question"`         // 视频内容问题
	Options       []string       `json:"options"`          // 选项（用于内容理解）
	ActionHint    string         `json:"action_hint"`      // 动作提示（用于动作识别）
	SequenceCount int            `json:"sequence_count"`   // 序列数量（用于序列验证）
	ExpiresIn     int64          `json:"expires_in"`
	ExpiresAt     int64          `json:"expires_at"`
	Language      string         `json:"language"`
}

type VideoChallenge struct {
	Type          VideoCaptchaType `json:"type"`
	Question      string           `json:"question"`
	CorrectAnswer string           `json:"correct_answer"`
	Options       []string         `json:"options"`
	ActionPattern []int            `json:"action_pattern"` // 动作模式序列
	SequenceData  []string         `json:"sequence_data"`  // 序列数据
}

func NewVideoGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *VideoGeneratorService {
	return &VideoGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *VideoGeneratorService) Generate(ctx context.Context, req *VideoCaptchaRequest) (*VideoCaptchaResponse, error) {
	if req.Type == "" {
		req.Type = VideoTypeContent
	}
	if req.Duration <= 0 {
		req.Duration = 3
	}
	if req.Language == "" {
		req.Language = "zh-CN"
	}

	challenge := s.generateChallenge(req.Type, req.Language)
	videoData := s.generateVideoData(challenge, req.Duration)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	response := &VideoCaptchaResponse{
		SessionID:     sessionID,
		VideoData:     base64.StdEncoding.EncodeToString(videoData),
		VideoType:     challenge.Type,
		Question:      challenge.Question,
		Options:       challenge.Options,
		ExpiresIn:     int64(5 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
		Language:      req.Language,
	}

	switch challenge.Type {
	case VideoTypeAction:
		response.ActionHint = s.getActionHint(challenge.ActionPattern, req.Language)
	case VideoTypeSequence:
		response.SequenceCount = len(challenge.SequenceData)
	}

	return response, nil
}

func (s *VideoGeneratorService) generateChallenge(videoType VideoCaptchaType, language string) *VideoChallenge {
	switch videoType {
	case VideoTypeContent:
		return s.generateContentChallenge(language)
	case VideoTypeAction:
		return s.generateActionChallenge(language)
	case VideoTypeSequence:
		return s.generateSequenceChallenge(language)
	default:
		return s.generateContentChallenge(language)
	}
}

func (s *VideoGeneratorService) generateContentChallenge(language string) *VideoChallenge {
	contentChallenges := []struct {
		Question string
		Answer   string
		Options  []string
	}{
		{
			Question: "视频中出现的动物是什么？",
			Answer:   "猫",
			Options:  []string{"狗", "猫", "鸟", "鱼"},
		},
		{
			Question: "视频中的天气是？",
			Answer:   "晴天",
			Options:  []string{"晴天", "雨天", "阴天", "雪天"},
		},
		{
			Question: "视频中物体的颜色是？",
			Answer:   "红色",
			Options:  []string{"红色", "蓝色", "绿色", "黄色"},
		},
		{
			Question: "视频中的交通工具是？",
			Answer:   "汽车",
			Options:  []string{"汽车", "飞机", "火车", "轮船"},
		},
		{
			Question: "视频中人物的动作是？",
			Answer:   "跑步",
			Options:  []string{"跑步", "走路", "跳跃", "站立"},
		},
	}

	if language == "en-US" {
		contentChallenges = []struct {
			Question string
			Answer   string
			Options  []string
		}{
			{
				Question: "What animal appears in the video?",
				Answer:   "Cat",
				Options:  []string{"Dog", "Cat", "Bird", "Fish"},
			},
			{
				Question: "What is the weather in the video?",
				Answer:   "Sunny",
				Options:  []string{"Sunny", "Rainy", "Cloudy", "Snowy"},
			},
			{
				Question: "What is the color of the object?",
				Answer:   "Red",
				Options:  []string{"Red", "Blue", "Green", "Yellow"},
			},
			{
				Question: "What vehicle is in the video?",
				Answer:   "Car",
				Options:  []string{"Car", "Plane", "Train", "Ship"},
			},
			{
				Question: "What action is the person doing?",
				Answer:   "Running",
				Options:  []string{"Running", "Walking", "Jumping", "Standing"},
			},
		}
	}

	challenge := contentChallenges[rand.Intn(len(contentChallenges))]

	shuffledOptions := shuffleStringSlice(challenge.Options)

	return &VideoChallenge{
		Type:          VideoTypeContent,
		Question:      challenge.Question,
		CorrectAnswer: challenge.Answer,
		Options:       shuffledOptions,
	}
}

func (s *VideoGeneratorService) generateActionChallenge(language string) *VideoChallenge {
	actionPatterns := [][]int{
		{1, 2, 1}, // 左、右、左
		{2, 1, 2}, // 右、左、右
		{1, 1, 2}, // 左、左、右
		{2, 2, 1}, // 右、右、左
		{1, 2, 2}, // 左、右、右
		{1, 2, 3}, // 左、右、上
		{3, 2, 1}, // 上、右、左
		{2, 3, 1}, // 右、上、左
	}

	pattern := actionPatterns[rand.Intn(len(actionPatterns))]

	return &VideoChallenge{
		Type:          VideoTypeAction,
		Question:      "请模仿视频中的手势动作",
		CorrectAnswer: fmt.Sprintf("%v", pattern),
		ActionPattern: pattern,
	}
}

func (s *VideoGeneratorService) generateSequenceChallenge(language string) *VideoChallenge {
	sequenceColors := []string{"红色", "蓝色", "绿色", "黄色", "紫色", "橙色"}
	if language == "en-US" {
		sequenceColors = []string{"Red", "Blue", "Green", "Yellow", "Purple", "Orange"}
	}

	sequenceLength := 3 + rand.Intn(3)
	sequence := make([]string, sequenceLength)

	for i := 0; i < sequenceLength; i++ {
		sequence[i] = sequenceColors[rand.Intn(len(sequenceColors))]
	}

	question := "请按顺序点击视频中出现的颜色"
	if language == "en-US" {
		question = "Click the colors in the order they appear in the video"
	}

	return &VideoChallenge{
		Type:          VideoTypeSequence,
		Question:      question,
		CorrectAnswer: fmt.Sprintf("%v", sequence),
		SequenceData:  sequence,
	}
}

func (s *VideoGeneratorService) generateVideoData(challenge *VideoChallenge, duration int) []byte {
	frameCount := duration * 10

	var videoData []byte
	videoData = append(videoData, []byte("VIDEO_MAGIC_HEADER")...)
	videoData = append(videoData, byte(challenge.Type[0]))

	for i := 0; i < frameCount; i++ {
		frame := generateVideoFrame(challenge, i, frameCount)
		videoData = append(videoData, frame...)
	}

	videoData = append(videoData, []byte("VIDEO_MAGIC_FOOTER")...)

	return videoData
}

func generateVideoFrame(challenge *VideoChallenge, frameIndex, totalFrames int) []byte {
	frame := make([]byte, 256)

	progress := float64(frameIndex) / float64(totalFrames)

	switch challenge.Type {
	case VideoTypeContent:
		frame[0] = byte('C')
		copy(frame[1:], []byte(challenge.CorrectAnswer)[:min(len(challenge.CorrectAnswer), 32)])
	case VideoTypeAction:
		frame[0] = byte('A')
		for i, action := range challenge.ActionPattern {
			if i < 32 {
				frame[i+1] = byte(action)
			}
		}
	case VideoTypeSequence:
		frame[0] = byte('S')
		if len(challenge.SequenceData) > 0 {
			currentSegment := int(progress * float64(len(challenge.SequenceData)))
			if currentSegment < len(challenge.SequenceData) {
				copy(frame[1:], []byte(challenge.SequenceData[currentSegment])[:min(len(challenge.SequenceData[currentSegment]), 32)])
			}
		}
	}

	for i := 33; i < 256; i++ {
		frame[i] = byte(rand.Intn(256))
	}

	return frame
}

func (s *VideoGeneratorService) getActionHint(actionPattern []int, language string) string {
	actionNames := map[int]string{
		1: "左",
		2: "右",
		3: "上",
		4: "下",
	}
	if language == "en-US" {
		actionNames = map[int]string{
			1: "Left",
			2: "Right",
			3: "Up",
			4: "Down",
		}
	}

	hint := ""
	for i, action := range actionPattern {
		if i > 0 {
			hint += " → "
		}
		hint += actionNames[action]
	}

	return hint
}

func shuffleStringSlice(slice []string) []string {
	result := make([]string, len(slice))
	perm := rand.Perm(len(slice))
	for i, p := range perm {
		result[i] = slice[p]
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}