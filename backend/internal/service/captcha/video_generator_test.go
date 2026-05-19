package captcha

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVideoGeneratorService_Create(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	req := &VideoCaptchaRequest{
		Width:      640,
		Height:     360,
		Difficulty: 2,
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}

	result, err := generator.Create(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SessionID)
	assert.Contains(t, result.SessionID, "video_")
	assert.NotEmpty(t, result.Question)
	assert.NotEmpty(t, result.Options)
	assert.Len(t, result.Options, 4)
	assert.NotEmpty(t, result.TargetAction)
	assert.Greater(t, result.Duration, 0)
	assert.Greater(t, result.ExpiresIn, int64(0))
}

func TestVideoGeneratorService_CreateDefaultValues(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	req := &VideoCaptchaRequest{}

	result, err := generator.Create(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SessionID)
	assert.Contains(t, result.SessionID, "video_")
}

func TestVideoGeneratorService_CreateDifferentDifficulties(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	difficulties := []int{1, 2, 3}
	for _, diff := range difficulties {
		req := &VideoCaptchaRequest{
			Difficulty: diff,
		}

		result, err := generator.Create(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, diff, result.Difficulty)
	}
}

func TestVideoGeneratorService_selectTargetAction(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	action1 := generator.selectTargetAction(1)
	simpleActions := []string{"举手", "挥手", "点头", "摇头"}
	found := false
	for _, a := range simpleActions {
		if action1 == a {
			found = true
			break
		}
	}
	assert.True(t, found, "Difficulty 1 should select from simple actions")

	action2 := generator.selectTargetAction(2)
	assert.NotEmpty(t, action2)

	action3 := generator.selectTargetAction(3)
	assert.NotEmpty(t, action3)
}

func TestVideoGeneratorService_generateQuestion(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	action := "举手"
	question := generator.generateQuestion(action)

	assert.NotEmpty(t, question)
	assert.Contains(t, question, action)
}

func TestVideoGeneratorService_generateOptions(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	correctAction := "举手"
	options := generator.generateOptions(correctAction)

	assert.Len(t, options, 4)
	assert.Contains(t, options, correctAction)

	uniqueOptions := make(map[string]bool)
	for _, opt := range options {
		uniqueOptions[opt] = true
	}
	assert.Len(t, uniqueOptions, 4)
}

func TestVideoGeneratorService_calculateDuration(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	tests := []struct {
		difficulty int
		expected   int
	}{
		{1, 5},
		{2, 8},
		{3, 12},
		{0, 8},
	}

	for _, tt := range tests {
		duration := generator.calculateDuration(tt.difficulty)
		assert.Equal(t, tt.expected, duration)
	}
}

func TestVideoGeneratorService_GetSession_NotFound(t *testing.T) {
	generator := NewVideoGeneratorServiceSimple()

	session, err := generator.GetSession(context.Background(), "nonexistent_session")
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestGenerateVideoSessionID(t *testing.T) {
	id1 := generateVideoSessionID()
	id2 := generateVideoSessionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.Contains(t, id1, "video_")
	assert.Contains(t, id2, "video_")
	assert.NotEqual(t, id1, id2)
}
