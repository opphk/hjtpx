package captcha

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateMemoryCardsBoard(t *testing.T) {
	board, err := generateMemoryCardsBoard(4, 4, 8)
	assert.NoError(t, err)
	assert.NotNil(t, board)
	assert.Equal(t, 4, board.Width)
	assert.Equal(t, 4, board.Height)
	assert.Equal(t, 8, board.PairCount)
	assert.True(t, board.Shuffled)

	typeCount := make(map[int]int)
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			card := board.Cards[y][x]
			typeCount[card.Type]++
		}
	}

	for _, count := range typeCount {
		assert.True(t, count%2 == 0)
	}
}

func TestMemoryCardsGeneratorService(t *testing.T) {
	generator := NewMemoryCardsGeneratorService(nil, nil)
	assert.NotNil(t, generator)

	req := &CreateMemoryCardsRequest{
		Width:       4,
		Height:      4,
		CardTypes:   8,
		ShowTime:    5,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.Board)
	assert.NotEmpty(t, resp.CardIcons)
	assert.Equal(t, int64(5), resp.ShowTime)
}

func TestMemoryCardsVerifierService(t *testing.T) {
	generator := NewMemoryCardsGeneratorService(nil, nil)
	verifier := NewMemoryCardsVerifierService(nil, nil)

	req := &CreateMemoryCardsRequest{
		Width:     4,
		Height:    4,
		CardTypes: 8,
		ShowTime:  5,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	board := resp.Board
	matches := make([]MemoryCardsMatch, 0)

	typeIndices := make(map[int][]int)
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			card := board.Cards[y][x]
			typeIndices[card.Type] = append(typeIndices[card.Type], card.Index)
		}
	}

	for _, indices := range typeIndices {
		for i := 0; i < len(indices); i += 2 {
			if i+1 < len(indices) {
				card1 := findCardByIndex(board, indices[i])
				card2 := findCardByIndex(board, indices[i+1])
				if card1 != nil && card2 != nil {
					matches = append(matches, MemoryCardsMatch{
						Card1: card1,
						Card2: card2,
					})
				}
			}
		}
	}

	isValid, score := verifier.validateBoard(board, board, matches, 30)
	assert.True(t, isValid)
	assert.Equal(t, 100.0, score)
}

func TestMemoryCardsVerifierWithWrongMatch(t *testing.T) {
	generator := NewMemoryCardsGeneratorService(nil, nil)
	verifier := NewMemoryCardsVerifierService(nil, nil)

	req := &CreateMemoryCardsRequest{
		Width:     4,
		Height:    4,
		CardTypes: 8,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	board := resp.Board
	matches := make([]MemoryCardsMatch, 0)

	// Create a wrong match
	if board.Height >= 2 && board.Width >= 2 {
		card1 := &board.Cards[0][0]
		card2 := &board.Cards[0][1]
		matches = append(matches, MemoryCardsMatch{
			Card1: card1,
			Card2: card2,
		})
	}

	isValid, score := verifier.validateBoard(board, board, matches, 30)
	assert.False(t, isValid)
	assert.True(t, score < 100)
}

func TestMemoryCardsTimeBonus(t *testing.T) {
	generator := NewMemoryCardsGeneratorService(nil, nil)
	verifier := NewMemoryCardsVerifierService(nil, nil)

	req := &CreateMemoryCardsRequest{
		Width:     4,
		Height:    4,
		CardTypes: 8,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	board := resp.Board
	matches := make([]MemoryCardsMatch, 0)

	typeIndices := make(map[int][]int)
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			card := board.Cards[y][x]
			typeIndices[card.Type] = append(typeIndices[card.Type], card.Index)
		}
	}

	for _, indices := range typeIndices {
		for i := 0; i < len(indices); i += 2 {
			if i+1 < len(indices) {
				card1 := findCardByIndex(board, indices[i])
				card2 := findCardByIndex(board, indices[i+1])
				if card1 != nil && card2 != nil {
					matches = append(matches, MemoryCardsMatch{
						Card1: card1,
						Card2: card2,
					})
				}
			}
		}
	}

	// Faster time should give higher score (but capped at 100)
	_, fastScore := verifier.validateBoard(board, board, matches, 10)
	_, slowScore := verifier.validateBoard(board, board, matches, 60)

	assert.True(t, fastScore >= slowScore)
}

func findCardByIndex(board *MemoryCardsBoard, index int) *MemoryCard {
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if board.Cards[y][x].Index == index {
				return &board.Cards[y][x]
			}
		}
	}
	return nil
}
