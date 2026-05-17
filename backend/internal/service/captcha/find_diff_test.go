package captcha

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFindDiffImage(t *testing.T) {
	image, err := generateFindDiffImage(400, 400, 5)
	assert.NoError(t, err)
	assert.NotNil(t, image)
	assert.Equal(t, 400, image.Width)
	assert.Equal(t, 400, image.Height)
	assert.Equal(t, 5, image.DiffCount)
	assert.Len(t, image.Differences, 5)
	assert.NotEmpty(t, image.Image1Data)
	assert.NotEmpty(t, image.Image2Data)
}

func TestFindDiffGeneratorService(t *testing.T) {
	generator := NewFindDiffGeneratorService(nil, nil)
	assert.NotNil(t, generator)

	req := &CreateFindDiffRequest{
		Width:       400,
		Height:      400,
		DiffCount:   5,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.Image)
	assert.NotEmpty(t, resp.Image.Image1Data)
	assert.NotEmpty(t, resp.Image.Image2Data)
}

func TestFindDiffVerifierService(t *testing.T) {
	generator := NewFindDiffGeneratorService(nil, nil)
	verifier := NewFindDiffVerifierService(nil, nil)

	req := &CreateFindDiffRequest{
		Width:     400,
		Height:    400,
		DiffCount: 5,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	image := resp.Image
	userDiffs := make([]FindDifference, 0)
	for _, diff := range image.Differences {
		userDiffs = append(userDiffs, FindDifference{
			X: diff.X,
			Y: diff.Y,
		})
	}

	isValid, score := verifier.validateDifferences(image, userDiffs)
	assert.True(t, isValid)
	assert.Equal(t, 100.0, score)
}

func TestFindDiffPartialMatch(t *testing.T) {
	generator := NewFindDiffGeneratorService(nil, nil)
	verifier := NewFindDiffVerifierService(nil, nil)

	req := &CreateFindDiffRequest{
		Width:     400,
		Height:    400,
		DiffCount: 5,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	image := resp.Image
	userDiffs := make([]FindDifference, 0)
	for i, diff := range image.Differences {
		if i < 3 {
			userDiffs = append(userDiffs, FindDifference{
				X: diff.X,
				Y: diff.Y,
			})
		}
	}

	isValid, score := verifier.validateDifferences(image, userDiffs)
	assert.False(t, isValid)
	assert.Equal(t, 60.0, score)
}

func TestFindDiffNoMatch(t *testing.T) {
	generator := NewFindDiffGeneratorService(nil, nil)
	verifier := NewFindDiffVerifierService(nil, nil)

	req := &CreateFindDiffRequest{
		Width:     400,
		Height:    400,
		DiffCount: 5,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	image := resp.Image
	userDiffs := make([]FindDifference, 0)

	isValid, score := verifier.validateDifferences(image, userDiffs)
	assert.False(t, isValid)
	assert.Equal(t, 0.0, score)
}

func TestFindDiffWrongPosition(t *testing.T) {
	generator := NewFindDiffGeneratorService(nil, nil)
	verifier := NewFindDiffVerifierService(nil, nil)

	req := &CreateFindDiffRequest{
		Width:     400,
		Height:    400,
		DiffCount: 5,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	image := resp.Image
	userDiffs := []FindDifference{
		{X: 10, Y: 10},
		{X: 20, Y: 20},
	}

	isValid, score := verifier.validateDifferences(image, userDiffs)
	assert.False(t, isValid)
	assert.Equal(t, 0.0, score)
}
