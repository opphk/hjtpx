package captcha

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"time"
)

type GeneratorType string

const (
	GeneratorTypeSlider   GeneratorType = "slider"
	GeneratorTypeImage   GeneratorType = "image"
	GeneratorTypeMath    GeneratorType = "math"
)

var (
	ErrInvalidGeneratorType = errors.New("invalid generator type")
	ErrGenerationFailed     = errors.New("captcha generation failed")
)

type Generator interface {
	Generate() (interface{}, error)
	GetType() GeneratorType
}

type GeneratorFactory struct {
	generators map[GeneratorType]Generator
}

func NewGeneratorFactory() *GeneratorFactory {
	factory := &GeneratorFactory{
		generators: make(map[GeneratorType]Generator),
	}

	factory.registerGenerators()

	return factory
}

func (f *GeneratorFactory) registerGenerators() {
	f.generators[GeneratorTypeSlider] = NewSliderGeneratorAdapter()
	f.generators[GeneratorTypeImage] = NewSliderGeneratorAdapter()
	f.generators[GeneratorTypeMath] = NewMathGenerator()
}

func (f *GeneratorFactory) GetGenerator(genType GeneratorType) (Generator, error) {
	generator, ok := f.generators[genType]
	if !ok {
		return nil, ErrInvalidGeneratorType
	}
	return generator, nil
}

func (f *GeneratorFactory) RegisterGenerator(genType GeneratorType, generator Generator) {
	f.generators[genType] = generator
}

type CaptchaChallenge struct {
	Type        GeneratorType   `json:"type"`
	Token       string          `json:"token"`
	Background  string          `json:"background"`
	SliderImage string          `json:"slider_image,omitempty"`
	Question    string          `json:"question,omitempty"`
	Answer      string          `json:"-"`
	X           int             `json:"x,omitempty"`
	Y           int             `json:"y,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (c *CaptchaChallenge) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *CaptchaChallenge) GetAnswer() string {
	return c.Answer
}

func (c *CaptchaChallenge) ValidateAnswer(answer string) bool {
	return c.Answer == answer
}

type SliderGeneratorAdapter struct {
	sliderGen *SliderGenerator
}

func NewSliderGeneratorAdapter() *SliderGeneratorAdapter {
	return &SliderGeneratorAdapter{
		sliderGen: NewSliderGenerator(),
	}
}

func (a *SliderGeneratorAdapter) Generate() (interface{}, error) {
	result, err := a.sliderGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGenerationFailed, err)
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	return &CaptchaChallenge{
		Type:        GeneratorTypeSlider,
		Token:       result.Token,
		Background:  result.BackgroundImage,
		SliderImage: result.SliderImage,
		X:           result.X,
		Y:           result.Y,
		Answer:      fmt.Sprintf("%d", result.X),
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}, nil
}

func (a *SliderGeneratorAdapter) GetType() GeneratorType {
	return GeneratorTypeSlider
}

type MathGenerator struct{}

func NewMathGenerator() *MathGenerator {
	return &MathGenerator{}
}

type MathChallenge struct {
	Question string `json:"question"`
	Answer   int    `json:"-"`
}

func (m *MathGenerator) Generate() (interface{}, error) {
	a := secureRandInt(1, 9)
	b := secureRandInt(1, 9)
	answer := a + b
	question := fmt.Sprintf("%d + %d = ?", a, b)

	tokenBytes := make([]byte, 16)
	cryptorand.Read(tokenBytes)

	expiresAt := time.Now().Add(5 * time.Minute)

	return &CaptchaChallenge{
		Type:      GeneratorTypeMath,
		Token:     fmt.Sprintf("%x", tokenBytes),
		Question:  question,
		Answer:    fmt.Sprintf("%d", answer),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MathGenerator) GetType() GeneratorType {
	return GeneratorTypeMath
}

type ImageGenerator struct {
	sliderGen *SliderGenerator
}

func NewImageGenerator() *ImageGenerator {
	return &ImageGenerator{
		sliderGen: NewSliderGenerator(),
	}
}

func (g *ImageGenerator) Generate() (interface{}, error) {
	result, err := g.sliderGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGenerationFailed, err)
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	return &CaptchaChallenge{
		Type:        GeneratorTypeImage,
		Token:       result.Token,
		Background:  result.BackgroundImage,
		SliderImage: result.SliderImage,
		X:           result.X,
		Y:           result.Y,
		Answer:      fmt.Sprintf("%d", result.X),
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}, nil
}

func (g *ImageGenerator) GetType() GeneratorType {
	return GeneratorTypeImage
}

func secureRandInt(min, max int) int {
	return min + int(secureRandom()%uint64(max-min+1))
}

func secureRandom() uint64 {
	bytes := make([]byte, 8)
	cryptorand.Read(bytes)
	var result uint64
	for _, b := range bytes {
		result = result<<8 | uint64(b)
	}
	return result
}

func randBytes(bytes []byte) {
	if _, err := cryptorand.Read(bytes); err != nil {
		for i := range bytes {
			bytes[i] = byte(i % 256)
		}
	}
}
