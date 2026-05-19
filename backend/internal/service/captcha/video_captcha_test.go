package captcha

import (
	"context"
	"fmt"
	"testing"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestVideoGeneratorService_Generate(t *testing.T) {
	generator := NewVideoGeneratorService(nil, nil)

	tests := []struct {
		name     string
		req      *VideoCaptchaRequest
		wantType VideoCaptchaType
	}{
		{
			name: "Generate content video captcha",
			req: &VideoCaptchaRequest{
				Type:     VideoTypeContent,
				Duration: 3,
				Language: "zh-CN",
			},
			wantType: VideoTypeContent,
		},
		{
			name: "Generate action video captcha",
			req: &VideoCaptchaRequest{
				Type:     VideoTypeAction,
				Duration: 3,
				Language: "zh-CN",
			},
			wantType: VideoTypeAction,
		},
		{
			name: "Generate sequence video captcha",
			req: &VideoCaptchaRequest{
				Type:     VideoTypeSequence,
				Duration: 3,
				Language: "zh-CN",
			},
			wantType: VideoTypeSequence,
		},
		{
			name: "Generate default content captcha when type empty",
			req: &VideoCaptchaRequest{
				Type:     "",
				Duration: 3,
				Language: "en-US",
			},
			wantType: VideoTypeContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := generator.Generate(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if resp.SessionID == "" {
				t.Error("Generate() returned empty session ID")
			}

			if resp.VideoType != tt.wantType {
				t.Errorf("Generate() VideoType = %v, want %v", resp.VideoType, tt.wantType)
			}

			if resp.VideoData == "" {
				t.Error("Generate() returned empty video data")
			}

			if resp.ExpiresIn <= 0 {
				t.Error("Generate() returned invalid expires_in")
			}

			if tt.req.Type == VideoTypeContent && resp.Question == "" {
				t.Error("Generate() returned empty question for content type")
			}

			if tt.req.Type == VideoTypeContent && len(resp.Options) == 0 {
				t.Error("Generate() returned empty options for content type")
			}

			if tt.req.Type == VideoTypeAction && resp.ActionHint == "" {
				t.Error("Generate() returned empty action hint for action type")
			}

			if tt.req.Type == VideoTypeSequence && resp.SequenceCount == 0 {
				t.Error("Generate() returned zero sequence count for sequence type")
			}
		})
	}
}

func TestVideoGeneratorService_generateContentChallenge(t *testing.T) {
	generator := NewVideoGeneratorService(nil, nil)

	tests := []struct {
		name     string
		language string
	}{
		{
			name:     "Chinese content challenge",
			language: "zh-CN",
		},
		{
			name:     "English content challenge",
			language: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := generator.generateContentChallenge(tt.language)

			if challenge.Type != VideoTypeContent {
				t.Errorf("generateContentChallenge() Type = %v, want %v", challenge.Type, VideoTypeContent)
			}

			if challenge.Question == "" {
				t.Error("generateContentChallenge() returned empty question")
			}

			if challenge.CorrectAnswer == "" {
				t.Error("generateContentChallenge() returned empty correct answer")
			}

			if len(challenge.Options) != 4 {
				t.Errorf("generateContentChallenge() returned %d options, want 4", len(challenge.Options))
			}

			found := false
			for _, opt := range challenge.Options {
				if opt == challenge.CorrectAnswer {
					found = true
					break
				}
			}
			if !found {
				t.Error("generateContentChallenge() correct answer not in options")
			}
		})
	}
}

func TestVideoGeneratorService_generateActionChallenge(t *testing.T) {
	generator := NewVideoGeneratorService(nil, nil)

	challenge := generator.generateActionChallenge("zh-CN")

	if challenge.Type != VideoTypeAction {
		t.Errorf("generateActionChallenge() Type = %v, want %v", challenge.Type, VideoTypeAction)
	}

	if len(challenge.ActionPattern) == 0 {
		t.Error("generateActionChallenge() returned empty action pattern")
	}

	for _, action := range challenge.ActionPattern {
		if action < 1 || action > 4 {
			t.Errorf("generateActionChallenge() invalid action value: %d", action)
		}
	}
}

func TestVideoGeneratorService_generateSequenceChallenge(t *testing.T) {
	generator := NewVideoGeneratorService(nil, nil)

	tests := []struct {
		name     string
		language string
	}{
		{
			name:     "Chinese sequence challenge",
			language: "zh-CN",
		},
		{
			name:     "English sequence challenge",
			language: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := generator.generateSequenceChallenge(tt.language)

			if challenge.Type != VideoTypeSequence {
				t.Errorf("generateSequenceChallenge() Type = %v, want %v", challenge.Type, VideoTypeSequence)
			}

			if len(challenge.SequenceData) < 3 || len(challenge.SequenceData) > 5 {
				t.Errorf("generateSequenceChallenge() sequence length = %d, want 3-5", len(challenge.SequenceData))
			}

			for _, color := range challenge.SequenceData {
				if color == "" {
					t.Error("generateSequenceChallenge() returned empty color in sequence")
				}
			}
		})
	}
}

func TestVideoVerifierService_verifyContent(t *testing.T) {
	verifier := NewVideoVerifierService()

	tests := []struct {
		name          string
		correctAnswer string
		answer        string
		wantSuccess   bool
		wantScore     float64
	}{
		{
			name:          "Correct answer",
			correctAnswer: "猫",
			answer:        "猫",
			wantSuccess:   true,
			wantScore:     100,
		},
		{
			name:          "Incorrect answer",
			correctAnswer: "猫",
			answer:        "狗",
			wantSuccess:   false,
			wantScore:     0,
		},
		{
			name:          "Empty answer",
			correctAnswer: "猫",
			answer:        "",
			wantSuccess:   false,
			wantScore:     0,
		},
		{
			name:          "Case insensitive match",
			correctAnswer: "Cat",
			answer:        "cat",
			wantSuccess:   true,
			wantScore:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &models.VideoCaptchaSession{
				Type:          string(VideoTypeContent),
				CorrectAnswer: tt.correctAnswer,
			}

			success, score, _ := verifier.verifyContent(session, tt.answer)

			if success != tt.wantSuccess {
				t.Errorf("verifyContent() success = %v, want %v", success, tt.wantSuccess)
			}

			if score != tt.wantScore {
				t.Errorf("verifyContent() score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

func TestVideoVerifierService_verifyAction(t *testing.T) {
	verifier := NewVideoVerifierService()

	tests := []struct {
		name            string
		expectedPattern []int
		actionResult    []int
		wantSuccess     bool
	}{
		{
			name:            "Exact match",
			expectedPattern: []int{1, 2, 1},
			actionResult:    []int{1, 2, 1},
			wantSuccess:     true,
		},
		{
			name:            "Partial match",
			expectedPattern: []int{1, 2, 1},
			actionResult:    []int{1, 1, 1},
			wantSuccess:     false,
		},
		{
			name:            "Wrong length",
			expectedPattern: []int{1, 2, 1},
			actionResult:    []int{1, 2},
			wantSuccess:     false,
		},
		{
			name:            "Empty result",
			expectedPattern: []int{1, 2, 1},
			actionResult:    []int{},
			wantSuccess:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actionPatternJSON := fmt.Sprintf("[%d,%d,%d]", tt.expectedPattern[0], tt.expectedPattern[1], tt.expectedPattern[2])
			session := &models.VideoCaptchaSession{
				Type:          string(VideoTypeAction),
				ActionPattern: actionPatternJSON,
			}

			success, _, _ := verifier.verifyAction(session, tt.actionResult)

			if success != tt.wantSuccess {
				t.Errorf("verifyAction() success = %v, want %v", success, tt.wantSuccess)
			}
		})
	}
}

func TestVideoVerifierService_verifySequence(t *testing.T) {
	verifier := NewVideoVerifierService()

	tests := []struct {
		name             string
		expectedSequence []string
		sequence         []string
		wantSuccess      bool
	}{
		{
			name:             "Exact match",
			expectedSequence: []string{"红色", "蓝色", "绿色"},
			sequence:         []string{"红色", "蓝色", "绿色"},
			wantSuccess:      true,
		},
		{
			name:             "Wrong order",
			expectedSequence: []string{"红色", "蓝色", "绿色"},
			sequence:         []string{"红色", "绿色", "蓝色"},
			wantSuccess:      false,
		},
		{
			name:             "Wrong length",
			expectedSequence: []string{"红色", "蓝色", "绿色"},
			sequence:         []string{"红色", "蓝色"},
			wantSuccess:      false,
		},
		{
			name:             "Case insensitive",
			expectedSequence: []string{"Red", "Blue", "Green"},
			sequence:         []string{"red", "blue", "green"},
			wantSuccess:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sequenceJSON := fmt.Sprintf(`["%s","%s","%s"]`, tt.expectedSequence[0], tt.expectedSequence[1], tt.expectedSequence[2])
			session := &models.VideoCaptchaSession{
				Type:         string(VideoTypeSequence),
				SequenceData: sequenceJSON,
			}

			success, _, _ := verifier.verifySequence(session, tt.sequence)

			if success != tt.wantSuccess {
				t.Errorf("verifySequence() success = %v, want %v", success, tt.wantSuccess)
			}
		})
	}
}