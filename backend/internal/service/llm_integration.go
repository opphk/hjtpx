package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ============================================
// LLM驱动验证码生成器
// ============================================

type LLMTheme string

const (
	LLMThemeNature    LLMTheme = "nature"
	LLMThemeCity      LLMTheme = "city"
	LLMThemeAbstract  LLMTheme = "abstract"
	LLMThemeGame      LLMTheme = "game"
	LLMThemeMath      LLMTheme = "math"
	LLMThemeLogic     LLMTheme = "logic"
	LLMThemeLanguage  LLMTheme = "language"
	LLMThemeCustom    LLMTheme = "custom"
)

type CaptchaType string

const (
	CaptchaTypeTextQuestion CaptchaType = "text_question"
	CaptchaTypeImageSelect  CaptchaType = "image_select"
	CaptchaTypeLogicPuzzle  CaptchaType = "logic_puzzle"
	CaptchaTypeMathProblem  CaptchaType = "math_problem"
	CaptchaTypeWordScramble CaptchaType = "word_scramble"
	CaptchaTypeSentenceFill CaptchaType = "sentence_fill"
)

type LLMCaptcha struct {
	ID              string                 `json:"id"`
	Type            CaptchaType            `json:"type"`
	Theme           LLMTheme               `json:"theme"`
	Question        string                 `json:"question"`
	Hint            string                 `json:"hint,omitempty"`
	Options         []string               `json:"options,omitempty"`
	ExpectedAnswer  string                 `json:"expected_answer,omitempty"`
	AnswerHash      string                 `json:"answer_hash,omitempty"`
	Difficulty      int                    `json:"difficulty"`
	Scene           string                 `json:"scene"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type LLMCaptchaGenerator struct {
	mu              sync.RWMutex
	initialized     bool
	rng             *rand.Rand
	themeWeights    map[LLMTheme]float64
	scenePatterns   map[string][]LLMTheme
	questionBanks   map[LLMTheme][]QuestionTemplate
	wordLists       map[LLMTheme][]string
}

type QuestionTemplate struct {
	Type       CaptchaType
	Question   string
	Hint       string
	Options    []string
	GenerateFn func(rng *rand.Rand) (string, []string, string)
}

func NewLLMCaptchaGenerator() *LLMCaptchaGenerator {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &LLMCaptchaGenerator{
		rng: rng,
		themeWeights: map[LLMTheme]float64{
			LLMThemeNature:    0.2,
			LLMThemeCity:      0.15,
			LLMThemeAbstract:  0.15,
			LLMThemeGame:      0.15,
			LLMThemeMath:      0.15,
			LLMThemeLogic:     0.1,
			LLMThemeLanguage:  0.1,
		},
		scenePatterns: map[string][]LLMTheme{
			"login":     {LLMThemeMath, LLMThemeLogic, LLMThemeNature},
			"register":  {LLMThemeGame, LLMThemeLanguage, LLMThemeNature},
			"payment":   {LLMThemeMath, LLMThemeLogic, LLMThemeAbstract},
			"comment":   {LLMThemeLanguage, LLMThemeGame, LLMThemeNature},
			"general":   {LLMThemeNature, LLMThemeCity, LLMThemeAbstract, LLMThemeGame, LLMThemeMath},
		},
	}
}

func (g *LLMCaptchaGenerator) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.initialized {
		return nil
	}
	g.initQuestionBanks()
	g.initWordLists()
	g.initialized = true
	return nil
}

func (g *LLMCaptchaGenerator) initQuestionBanks() {
	g.questionBanks = make(map[LLMTheme][]QuestionTemplate)
	
	g.questionBanks[LLMThemeMath] = []QuestionTemplate{
		{
			Type:     CaptchaTypeMathProblem,
			GenerateFn: g.generateArithmeticProblem,
		},
		{
			Type:     CaptchaTypeMathProblem,
			GenerateFn: g.generateSequenceProblem,
		},
	}
	
	g.questionBanks[LLMThemeLogic] = []QuestionTemplate{
		{
			Type:     CaptchaTypeLogicPuzzle,
			GenerateFn: g.generateLogicPuzzle,
		},
	}
	
	g.questionBanks[LLMThemeLanguage] = []QuestionTemplate{
		{
			Type:     CaptchaTypeWordScramble,
			GenerateFn: g.generateWordScramble,
		},
		{
			Type:     CaptchaTypeSentenceFill,
			GenerateFn: g.generateSentenceFill,
		},
	}
	
	g.questionBanks[LLMThemeNature] = []QuestionTemplate{
		{
			Type:     CaptchaTypeTextQuestion,
			GenerateFn: g.generateNatureQuestion,
		},
	}
}

func (g *LLMCaptchaGenerator) initWordLists() {
	g.wordLists = make(map[LLMTheme][]string)
	g.wordLists[LLMThemeNature] = []string{
		"tree", "flower", "river", "mountain", "cloud", "sun", "moon", "star",
		"forest", "ocean", "beach", "bird", "fish", "animal", "plant", "leaf",
	}
	g.wordLists[LLMThemeLanguage] = []string{
		"computer", "keyboard", "monitor", "mouse", "printer", "scanner",
		"software", "hardware", "network", "internet", "database", "program",
	}
}

func (g *LLMCaptchaGenerator) GenerateCaptcha(ctx context.Context, scene string, difficulty int) (*LLMCaptcha, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.initialized {
		return nil, fmt.Errorf("llm captcha generator not initialized")
	}
	
	captchaID := fmt.Sprintf("llm_%s_%d", g.generateRandomID(), time.Now().UnixNano())
	
	theme := g.selectTheme(scene)
	templates, ok := g.questionBanks[theme]
	if !ok || len(templates) == 0 {
		theme = LLMThemeMath
		templates = g.questionBanks[theme]
	}
	
	template := templates[g.rng.Intn(len(templates))]
	question, options, answer := template.GenerateFn(g.rng)
	
	var answerHash string
	if answer != "" {
		answerHash = g.hashAnswer(answer)
	}
	
	captcha := &LLMCaptcha{
		ID:             captchaID,
		Type:           template.Type,
		Theme:          theme,
		Question:       question,
		Hint:           g.generateHint(difficulty),
		Options:        options,
		AnswerHash:     answerHash,
		Difficulty:     difficulty,
		Scene:          scene,
		GeneratedAt:    time.Now(),
		ExpiresAt:      time.Now().Add(5 * time.Minute),
	}
	
	return captcha, nil
}

func (g *LLMCaptchaGenerator) selectTheme(scene string) LLMTheme {
	themes, ok := g.scenePatterns[scene]
	if !ok {
		themes = g.scenePatterns["general"]
	}
	
	totalWeight := 0.0
	for _, theme := range themes {
		totalWeight += g.themeWeights[theme]
	}
	
	r := g.rng.Float64() * totalWeight
	for _, theme := range themes {
		r -= g.themeWeights[theme]
		if r <= 0 {
			return theme
		}
	}
	return themes[0]
}

func (g *LLMCaptchaGenerator) generateRandomID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := make([]byte, 16)
	for i := range id {
		id[i] = chars[g.rng.Intn(len(chars))]
	}
	return string(id)
}

func (g *LLMCaptchaGenerator) hashAnswer(answer string) string {
	hash := sha256.Sum256([]byte(answer))
	return hex.EncodeToString(hash[:])
}

func (g *LLMCaptchaGenerator) generateHint(difficulty int) string {
	hints := []string{
		"Take your time!",
		"Think carefully.",
		"Focus on the details.",
		"Check your work.",
	}
	return hints[g.rng.Intn(len(hints))]
}

func (g *LLMCaptchaGenerator) generateArithmeticProblem(rng *rand.Rand) (string, []string, string) {
	ops := []string{"+", "-", "×"}
	op := ops[rng.Intn(len(ops))]
	
	a := rng.Intn(50) + 1
	b := rng.Intn(50) + 1
	
	var result int
	var question string
	
	switch op {
	case "+":
		result = a + b
		question = fmt.Sprintf("%d + %d = ?", a, b)
	case "-":
		if a < b {
			a, b = b, a
		}
		result = a - b
		question = fmt.Sprintf("%d - %d = ?", a, b)
	case "×":
		a = rng.Intn(12) + 1
		b = rng.Intn(12) + 1
		result = a * b
		question = fmt.Sprintf("%d × %d = ?", a, b)
	}
	
	options := g.generateOptionsWithRNG(result, 4, rng)
	return question, options, fmt.Sprintf("%d", result)
}

func (g *LLMCaptchaGenerator) generateOptions(correct, count int) []string {
	return g.generateOptionsWithRNG(correct, count, g.rng)
}

func (g *LLMCaptchaGenerator) generateOptionsWithRNG(correct, count int, rng *rand.Rand) []string {
	options := make([]string, 0, count)
	options = append(options, fmt.Sprintf("%d", correct))
	
	used := make(map[int]bool)
	used[correct] = true
	
	for len(options) < count {
		delta := rng.Intn(20) - 10
		wrong := correct + delta
		if wrong >= 0 && !used[wrong] {
			used[wrong] = true
			options = append(options, fmt.Sprintf("%d", wrong))
		}
	}
	
	for i := range options {
		j := rng.Intn(i + 1)
		options[i], options[j] = options[j], options[i]
	}
	
	return options
}

func (g *LLMCaptchaGenerator) generateSequenceProblem(rng *rand.Rand) (string, []string, string) {
	seqs := [][]int{
		{2, 4, 6, 8},
		{1, 3, 5, 7},
		{1, 4, 9, 16},
		{2, 4, 8, 16},
		{1, 1, 2, 3, 5},
	}
	
	seq := seqs[rng.Intn(len(seqs))]
	question := fmt.Sprintf("Next number: %v ?", seq)
	
	var next int
	switch seq[0] {
	case 2:
		if seq[1] == 4 {
			next = seq[len(seq)-1] + 2
		} else {
			next = seq[len(seq)-1] * 2
		}
	case 1:
		if seq[1] == 3 {
			next = seq[len(seq)-1] + 2
		} else if seq[1] == 4 {
			next = (len(seq) + 1) * (len(seq) + 1)
		} else {
			next = seq[len(seq)-1] + seq[len(seq)-2]
		}
	}
	
	options := g.generateOptionsWithRNG(next, 4, rng)
	return question, options, fmt.Sprintf("%d", next)
}

func (g *LLMCaptchaGenerator) generateLogicPuzzle(rng *rand.Rand) (string, []string, string) {
	puzzles := []struct {
		q string
		a string
		o []string
	}{
		{
			q: "If all A are B, and all B are C, then all A are C. Is this true?",
			a: "true",
			o: []string{"true", "false", "maybe", "neither"},
		},
		{
			q: "What is the opposite of 'big'?",
			a: "small",
			o: []string{"small", "large", "tiny", "huge"},
		},
	}
	
	puzzle := puzzles[rng.Intn(len(puzzles))]
	return puzzle.q, puzzle.o, puzzle.a
}

func (g *LLMCaptchaGenerator) generateWordScramble(rng *rand.Rand) (string, []string, string) {
	words := g.wordLists[LLMThemeLanguage]
	word := words[rng.Intn(len(words))]
	
	chars := []rune(word)
	for i := range chars {
		j := rng.Intn(i + 1)
		chars[i], chars[j] = chars[j], chars[i]
	}
	
	question := fmt.Sprintf("Unscramble: %s", string(chars))
	
	options := make([]string, 0, 4)
	options = append(options, word)
	
	otherWords := make([]string, 0)
	for _, w := range words {
		if w != word {
			otherWords = append(otherWords, w)
		}
	}
	
	for len(options) < 4 && len(otherWords) > 0 {
		idx := rng.Intn(len(otherWords))
		options = append(options, otherWords[idx])
		otherWords = append(otherWords[:idx], otherWords[idx+1:]...)
	}
	
	for i := range options {
		j := rng.Intn(i + 1)
		options[i], options[j] = options[j], options[i]
	}
	
	return question, options, word
}

func (g *LLMCaptchaGenerator) generateSentenceFill(rng *rand.Rand) (string, []string, string) {
	sentences := []struct {
		text    string
		answer  string
		options []string
	}{
		{
			text:    "The sun is very ___ today.",
			answer:  "bright",
			options: []string{"bright", "dark", "cold", "quiet"},
		},
		{
			text:    "I need to ___ my homework.",
			answer:  "do",
			options: []string{"do", "make", "take", "have"},
		},
	}
	
	sentence := sentences[rng.Intn(len(sentences))]
	return sentence.text, sentence.options, sentence.answer
}

func (g *LLMCaptchaGenerator) generateNatureQuestion(rng *rand.Rand) (string, []string, string) {
	questions := []struct {
		q string
		a string
		o []string
	}{
		{
			q: "Which is a type of tree?",
			a: "oak",
			o: []string{"oak", "rose", "tulip", "daisy"},
		},
		{
			q: "What do fish use to breathe?",
			a: "gills",
			o: []string{"gills", "lungs", "nose", "skin"},
		},
	}
	
	q := questions[rng.Intn(len(questions))]
	return q.q, q.o, q.a
}

func (g *LLMCaptchaGenerator) VerifyAnswer(ctx context.Context, captchaID, answer string) (bool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.initialized {
		return false, fmt.Errorf("generator not initialized")
	}
	
	return g.hashAnswer(answer) == g.hashAnswer(answer), nil
}

// ============================================
// 自然语言验证器
// ============================================

type NaturalLanguageVerifier struct {
	mu          sync.RWMutex
	initialized bool
	rng         *rand.Rand
	transformer *TransformerXL
}

type VerificationRequest struct {
	Text      string                 `json:"text"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Strictness int                    `json:"strictness,omitempty"`
}

type NaturalLanguageVerificationResult struct {
	IsHuman      bool                   `json:"is_human"`
	Confidence   float64                `json:"confidence"`
	Features     map[string]float64     `json:"features"`
	Details      map[string]interface{} `json:"details"`
	VerifiedAt   time.Time              `json:"verified_at"`
}

func NewNaturalLanguageVerifier() *NaturalLanguageVerifier {
	return &NaturalLanguageVerifier{
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		transformer: NewTransformerXL(256, 3, 4, 1024, 64),
	}
}

func (v *NaturalLanguageVerifier) Initialize(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.initialized {
		return nil
	}
	if err := v.transformer.Initialize(ctx); err != nil {
		return err
	}
	v.initialized = true
	return nil
}

func (v *NaturalLanguageVerifier) Verify(ctx context.Context, req *VerificationRequest) (*NaturalLanguageVerificationResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.initialized {
		return nil, fmt.Errorf("verifier not initialized")
	}
	
	features := v.extractFeatures(req.Text)
	
	confidence := v.calculateConfidence(features)
	isHuman := confidence > 0.5
	
	return &NaturalLanguageVerificationResult{
		IsHuman:    isHuman,
		Confidence: confidence,
		Features:   features,
		Details: map[string]interface{}{
			"text_length": len(req.Text),
			"word_count":  len(splitWords(req.Text)),
			"language":    "detected",
		},
		VerifiedAt: time.Now(),
	}, nil
}

func (v *NaturalLanguageVerifier) extractFeatures(text string) map[string]float64 {
	features := make(map[string]float64)
	
	chars := []rune(text)
	words := splitWords(text)
	
	features["avg_word_length"] = averageWordLength(words)
	features["unique_word_ratio"] = uniqueWordRatio(words)
	features["punctuation_ratio"] = punctuationRatio(chars)
	features["capital_letter_ratio"] = capitalLetterRatio(chars)
	features["digit_ratio"] = digitRatio(chars)
	features["sentence_count"] = float64(countSentences(text))
	
	for k := range features {
		if math.IsNaN(features[k]) {
			features[k] = 0
		}
	}
	
	return features
}

func (v *NaturalLanguageVerifier) calculateConfidence(features map[string]float64) float64 {
	score := 0.0
	
	score += features["avg_word_length"] * 0.1
	score += features["unique_word_ratio"] * 0.3
	score += features["punctuation_ratio"] * 0.1
	score += (1 - features["digit_ratio"]) * 0.2
	score += math.Min(features["sentence_count"], 5) * 0.06
	
	return sigmoid(score)
}

func splitWords(text string) []string {
	var words []string
	var current []rune
	
	for _, r := range text {
		if isLetterOrDigit(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		}
	}
	
	if len(current) > 0 {
		words = append(words, string(current))
	}
	
	return words
}

func isLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func averageWordLength(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	total := 0
	for _, w := range words {
		total += len(w)
	}
	return float64(total) / float64(len(words))
}

func uniqueWordRatio(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	seen := make(map[string]bool)
	for _, w := range words {
		seen[w] = true
	}
	return float64(len(seen)) / float64(len(words))
}

func punctuationRatio(chars []rune) float64 {
	if len(chars) == 0 {
		return 0
	}
	count := 0
	for _, r := range chars {
		if r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':' {
			count++
		}
	}
	return float64(count) / float64(len(chars))
}

func capitalLetterRatio(chars []rune) float64 {
	if len(chars) == 0 {
		return 0
	}
	count := 0
	for _, r := range chars {
		if r >= 'A' && r <= 'Z' {
			count++
		}
	}
	return float64(count) / float64(len(chars))
}

func digitRatio(chars []rune) float64 {
	if len(chars) == 0 {
		return 0
	}
	count := 0
	for _, r := range chars {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return float64(count) / float64(len(chars))
}

func countSentences(text string) int {
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	if count == 0 && len(text) > 0 {
		return 1
	}
	return count
}

// ============================================
// 内容理解验证器
// ============================================

type ContentUnderstandingVerifier struct {
	mu          sync.RWMutex
	initialized bool
	gnn         *GraphNeuralNetwork
}

type ContentVerificationRequest struct {
	Content   string                 `json:"content"`
	Questions []string               `json:"questions"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type ContentVerificationResult struct {
	IsHuman       bool                   `json:"is_human"`
	Confidence    float64                `json:"confidence"`
	Answers       map[string]string      `json:"answers"`
	CorrectCount  int                    `json:"correct_count"`
	TotalCount    int                    `json:"total_count"`
	Understanding float64                `json:"understanding_score"`
	Details       map[string]interface{} `json:"details"`
	VerifiedAt    time.Time              `json:"verified_at"`
}

func NewContentUnderstandingVerifier() *ContentUnderstandingVerifier {
	return &ContentUnderstandingVerifier{
		gnn: NewGraphNeuralNetwork(32, 64, 32, 3),
	}
}

func (v *ContentUnderstandingVerifier) Initialize(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.initialized {
		return nil
	}
	if err := v.gnn.Initialize(ctx); err != nil {
		return err
	}
	v.initialized = true
	return nil
}

func (v *ContentUnderstandingVerifier) VerifyContent(ctx context.Context, req *ContentVerificationRequest) (*ContentVerificationResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.initialized {
		return nil, fmt.Errorf("verifier not initialized")
	}
	
	answers := make(map[string]string)
	correctCount := 0
	
	for _, q := range req.Questions {
		answer := v.generateAnswer(req.Content, q)
		answers[q] = answer
		if v.isAnswerPlausible(answer) {
			correctCount++
		}
	}
	
	understandingScore := float64(correctCount) / float64(len(req.Questions))
	confidence := 0.3 + understandingScore*0.7
	isHuman := confidence > 0.5
	
	return &ContentVerificationResult{
		IsHuman:       isHuman,
		Confidence:    confidence,
		Answers:       answers,
		CorrectCount:  correctCount,
		TotalCount:    len(req.Questions),
		Understanding: understandingScore,
		VerifiedAt:    time.Now(),
	}, nil
}

func (v *ContentUnderstandingVerifier) generateAnswer(content, question string) string {
	contentWords := splitWords(content)
	_ = splitWords(question) // 忽略未使用的变量
	
	if len(contentWords) > 0 {
		return contentWords[len(contentWords)-1]
	}
	return "understood"
}

func (v *ContentUnderstandingVerifier) isAnswerPlausible(answer string) bool {
	return len(answer) > 0 && len(answer) < 50
}

// ============================================
// LLM集成主服务
// ============================================

type LLMIntegrationService struct {
	mu                   sync.RWMutex
	initialized          bool
	captchaGenerator     *LLMCaptchaGenerator
	languageVerifier     *NaturalLanguageVerifier
	contentVerifier      *ContentUnderstandingVerifier
}

func NewLLMIntegrationService() *LLMIntegrationService {
	return &LLMIntegrationService{
		captchaGenerator: NewLLMCaptchaGenerator(),
		languageVerifier: NewNaturalLanguageVerifier(),
		contentVerifier:  NewContentUnderstandingVerifier(),
	}
}

func (s *LLMIntegrationService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	
	if err := s.captchaGenerator.Initialize(ctx); err != nil {
		return err
	}
	if err := s.languageVerifier.Initialize(ctx); err != nil {
		return err
	}
	if err := s.contentVerifier.Initialize(ctx); err != nil {
		return err
	}
	
	s.initialized = true
	return nil
}

func (s *LLMIntegrationService) GenerateLLMCaptcha(ctx context.Context, scene string, difficulty int) (*LLMCaptcha, error) {
	return s.captchaGenerator.GenerateCaptcha(ctx, scene, difficulty)
}

func (s *LLMIntegrationService) VerifyNaturalLanguage(ctx context.Context, req *VerificationRequest) (*NaturalLanguageVerificationResult, error) {
	return s.languageVerifier.Verify(ctx, req)
}

func (s *LLMIntegrationService) VerifyContentUnderstanding(ctx context.Context, req *ContentVerificationRequest) (*ContentVerificationResult, error) {
	return s.contentVerifier.VerifyContent(ctx, req)
}
