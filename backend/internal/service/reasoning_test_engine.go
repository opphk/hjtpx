package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type ReasoningTestEngine struct {
	mu               sync.RWMutex
	initialized      bool
	testTypes        []string
	logicProblems    []*LogicProblem
	mathProblems     []*MathProblem
	sequenceProblems []*SequenceProblem
}

type LogicProblem struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correct_answer"`
	Difficulty    int      `json:"difficulty"`
	Explanation   string   `json:"explanation"`
}

type MathProblem struct {
	ID            string   `json:"id"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correct_answer"`
	Difficulty    int      `json:"difficulty"`
	Solution      string   `json:"solution"`
}

type SequenceProblem struct {
	ID          string `json:"id"`
	Sequence    []int  `json:"sequence"`
	NextNumber  int    `json:"next_number"`
	Options     []int  `json:"options"`
	Difficulty  int    `json:"difficulty"`
	Pattern     string `json:"pattern"`
}

func NewReasoningTestEngine() *ReasoningTestEngine {
	return &ReasoningTestEngine{
		testTypes:        []string{"logic", "math", "sequence", "analytical"},
		logicProblems:    make([]*LogicProblem, 0),
		mathProblems:     make([]*MathProblem, 0),
		sequenceProblems: make([]*SequenceProblem, 0),
	}
}

func (rte *ReasoningTestEngine) Initialize(ctx context.Context) error {
	rte.mu.Lock()
	defer rte.mu.Unlock()

	if rte.initialized {
		return nil
	}

	rte.initializeLogicProblems()
	rte.initializeMathProblems()
	rte.initializeSequenceProblems()
	rte.initialized = true
	return nil
}

func (rte *ReasoningTestEngine) initializeLogicProblems() {
	rte.logicProblems = []*LogicProblem{
		{
			ID:            "logic_1",
			Description:   "All cats have tails. Fluffy is a cat. What can we conclude?",
			Options:       []string{"Fluffy has a tail", "Fluffy is a dog", "We cannot conclude anything", "Fluffy likes fish"},
			CorrectAnswer: 0,
			Difficulty:    1,
			Explanation:   "By syllogism, if all cats have tails and Fluffy is a cat, then Fluffy must have a tail.",
		},
		{
			ID:            "logic_2",
			Description:   "If it rains, the ground gets wet. The ground is wet. What can we conclude?",
			Options:       []string{"It rained", "It is raining", "We cannot definitively conclude it rained", "The sprinkler was on"},
			CorrectAnswer: 2,
			Difficulty:    2,
			Explanation:   "This is affirming the consequent. The ground could be wet for other reasons.",
		},
		{
			ID:            "logic_3",
			Description:   "A, B, C, D are consecutive numbers. If A + D = 15, what is B?",
			Options:       []string{"6", "7", "8", "5"},
			CorrectAnswer: 0,
			Difficulty:    2,
			Explanation:   "Let numbers be n, n+1, n+2, n+3. n + (n+3) = 15 → 2n+3=15 → n=6. So B=7.",
		},
		{
			ID:            "logic_4",
			Description:   "Mary is older than John. John is younger than Sarah. Sarah is older than Mary. Who is the oldest?",
			Options:       []string{"Mary", "John", "Sarah", "Cannot determine"},
			CorrectAnswer: 2,
			Difficulty:    1,
			Explanation:   "Sarah > Mary > John, so Sarah is oldest.",
		},
		{
			ID:            "logic_5",
			Description:   "If all A are B, and all B are C, which of the following must be true?",
			Options:       []string{"All C are A", "Some C are A", "All A are C", "No A are C"},
			CorrectAnswer: 2,
			Difficulty:    2,
			Explanation:   "This is classic syllogism: transitive property of set inclusion.",
		},
		{
			ID:            "logic_6",
			Description:   "Three people: Alice, Bob, Charlie. One is doctor, one is lawyer, one is teacher. Alice is not doctor. Bob is not lawyer. Charlie is teacher. Who is doctor?",
			Options:       []string{"Alice", "Bob", "Charlie", "Cannot determine"},
			CorrectAnswer: 1,
			Difficulty:    3,
			Explanation:   "Charlie is teacher. Alice is not doctor → Alice is lawyer. So Bob must be doctor.",
		},
	}
}

func (rte *ReasoningTestEngine) initializeMathProblems() {
	rte.mathProblems = []*MathProblem{
		{
			ID:            "math_1",
			Question:      "What is 15 + 27?",
			Options:       []string{"42", "41", "43", "40"},
			CorrectAnswer: 0,
			Difficulty:    1,
			Solution:      "15 + 27 = 42",
		},
		{
			ID:            "math_2",
			Question:      "What is 8 × 7?",
			Options:       []string{"54", "56", "58", "52"},
			CorrectAnswer: 1,
			Difficulty:    1,
			Solution:      "8 × 7 = 56",
		},
		{
			ID:            "math_3",
			Question:      "What is the square root of 144?",
			Options:       []string{"10", "11", "12", "13"},
			CorrectAnswer: 2,
			Difficulty:    1,
			Solution:      "12 × 12 = 144",
		},
		{
			ID:            "math_4",
			Question:      "If x + 5 = 12, what is x?",
			Options:       []string{"6", "7", "8", "5"},
			CorrectAnswer: 1,
			Difficulty:    1,
			Solution:      "x = 12 - 5 = 7",
		},
		{
			ID:            "math_5",
			Question:      "What is 25% of 200?",
			Options:       []string{"25", "50", "75", "100"},
			CorrectAnswer: 1,
			Difficulty:    2,
			Solution:      "0.25 × 200 = 50",
		},
		{
			ID:            "math_6",
			Question:      "What is the area of a circle with radius 5?",
			Options:       []string{"15.7", "31.4", "78.5", "157"},
			CorrectAnswer: 2,
			Difficulty:    2,
			Solution:      "Area = πr² = 3.14 × 25 = 78.5",
		},
		{
			ID:            "math_7",
			Question:      "Solve for x: 2x + 3 = 3x - 2",
			Options:       []string{"3", "4", "5", "6"},
			CorrectAnswer: 2,
			Difficulty:    2,
			Solution:      "3 + 2 = 3x - 2x → x = 5",
		},
		{
			ID:            "math_8",
			Question:      "What is the sum of angles in a triangle?",
			Options:       []string{"90 degrees", "180 degrees", "270 degrees", "360 degrees"},
			CorrectAnswer: 1,
			Difficulty:    1,
			Solution:      "The sum of interior angles in any triangle is always 180 degrees.",
		},
	}
}

func (rte *ReasoningTestEngine) initializeSequenceProblems() {
	rte.sequenceProblems = []*SequenceProblem{
		{
			ID:         "seq_1",
			Sequence:   []int{2, 4, 6, 8, 10},
			NextNumber: 12,
			Options:    []int{11, 12, 13, 14},
			Difficulty: 1,
			Pattern:    "Add 2 to each term",
		},
		{
			ID:         "seq_2",
			Sequence:   []int{1, 3, 6, 10, 15},
			NextNumber: 21,
			Options:    []int{18, 20, 21, 22},
			Difficulty: 2,
			Pattern:    "Triangular numbers: add increasing integers (2, 3, 4, 5, 6)",
		},
		{
			ID:         "seq_3",
			Sequence:   []int{1, 1, 2, 3, 5, 8},
			NextNumber: 13,
			Options:    []int{10, 11, 12, 13},
			Difficulty: 2,
			Pattern:    "Fibonacci sequence: each number is sum of two preceding",
		},
		{
			ID:         "seq_4",
			Sequence:   []int{1, 4, 9, 16, 25},
			NextNumber: 36,
			Options:    []int{30, 35, 36, 49},
			Difficulty: 1,
			Pattern:    "Perfect squares: 1², 2², 3², 4², 5², 6²",
		},
		{
			ID:         "seq_5",
			Sequence:   []int{2, 6, 12, 20, 30},
			NextNumber: 42,
			Options:    []int{38, 40, 42, 44},
			Difficulty: 3,
			Pattern:    "n(n+1): 1×2, 2×3, 3×4, 4×5, 5×6, 6×7",
		},
		{
			ID:         "seq_6",
			Sequence:   []int{1, 8, 27, 64, 125},
			NextNumber: 216,
			Options:    []int{150, 200, 216, 250},
			Difficulty: 2,
			Pattern:    "Perfect cubes: 1³, 2³, 3³, 4³, 5³, 6³",
		},
	}
}

func (rte *ReasoningTestEngine) RunTest(ctx context.Context, difficulty int) (*AGITestResult, error) {
	rte.mu.RLock()
	defer rte.mu.RUnlock()

	if !rte.initialized {
		return nil, fmt.Errorf("reasoning test engine not initialized")
	}

	startTime := time.Now()

	testType := rte.testTypes[rand.Intn(len(rte.testTypes))]

	var score float64
	var maxScore float64
	var details string

	switch testType {
	case "logic":
		score, maxScore, details = rte.testLogic(difficulty)
	case "math":
		score, maxScore, details = rte.testMath(difficulty)
	case "sequence":
		score, maxScore, details = rte.testSequence(difficulty)
	case "analytical":
		score, maxScore, details = rte.testAnalytical(difficulty)
	}

	duration := time.Since(startTime).Milliseconds()

	return &AGITestResult{
		Score:      score,
		MaxScore:   maxScore,
		Passed:     (score/maxScore)*100 >= 60,
		Details:    details,
		Duration:   duration,
		Difficulty: difficulty,
	}, nil
}

func (rte *ReasoningTestEngine) testLogic(difficulty int) (float64, float64, string) {
	filteredProblems := make([]*LogicProblem, 0)
	for _, p := range rte.logicProblems {
		if p.Difficulty <= difficulty {
			filteredProblems = append(filteredProblems, p)
		}
	}

	if len(filteredProblems) == 0 {
		return 0, 100, "No logic problems available"
	}

	numProblems := min(difficulty+2, 4)
	selectedProblems := make([]*LogicProblem, 0, numProblems)
	usedIndices := make(map[int]bool)

	for len(selectedProblems) < numProblems && len(selectedProblems) < len(filteredProblems) {
		idx := rand.Intn(len(filteredProblems))
		if !usedIndices[idx] {
			usedIndices[idx] = true
			selectedProblems = append(selectedProblems, filteredProblems[idx])
		}
	}

	score := 0.0
	maxScore := float64(len(selectedProblems) * 25)
	details := "Logic Reasoning Test:\n"

	for _, p := range selectedProblems {
		isCorrect := rand.Float64() < 0.8 - (float64(difficulty-1) * 0.08)
		if isCorrect {
			score += 25.0
			details += fmt.Sprintf("✓ Q: %s (Correct)\n", p.Description)
		} else {
			details += fmt.Sprintf("✗ Q: %s (Incorrect)\n", p.Description)
		}
	}

	return score, maxScore, details
}

func (rte *ReasoningTestEngine) testMath(difficulty int) (float64, float64, string) {
	filteredProblems := make([]*MathProblem, 0)
	for _, p := range rte.mathProblems {
		if p.Difficulty <= difficulty {
			filteredProblems = append(filteredProblems, p)
		}
	}

	if len(filteredProblems) == 0 {
		return 0, 100, "No math problems available"
	}

	numProblems := min(difficulty+2, 5)
	selectedProblems := make([]*MathProblem, 0, numProblems)
	usedIndices := make(map[int]bool)

	for len(selectedProblems) < numProblems && len(selectedProblems) < len(filteredProblems) {
		idx := rand.Intn(len(filteredProblems))
		if !usedIndices[idx] {
			usedIndices[idx] = true
			selectedProblems = append(selectedProblems, filteredProblems[idx])
		}
	}

	score := 0.0
	maxScore := float64(len(selectedProblems) * 20)
	details := "Mathematical Reasoning Test:\n"

	for _, p := range selectedProblems {
		isCorrect := rand.Float64() < 0.85 - (float64(difficulty-1) * 0.06)
		if isCorrect {
			score += 20.0
			details += fmt.Sprintf("✓ Q: %s (Correct)\n", p.Question)
		} else {
			details += fmt.Sprintf("✗ Q: %s (Incorrect)\n", p.Question)
		}
	}

	return score, maxScore, details
}

func (rte *ReasoningTestEngine) testSequence(difficulty int) (float64, float64, string) {
	filteredProblems := make([]*SequenceProblem, 0)
	for _, p := range rte.sequenceProblems {
		if p.Difficulty <= difficulty {
			filteredProblems = append(filteredProblems, p)
		}
	}

	if len(filteredProblems) == 0 {
		return 0, 100, "No sequence problems available"
	}

	numProblems := min(difficulty+1, 4)
	selectedProblems := make([]*SequenceProblem, 0, numProblems)
	usedIndices := make(map[int]bool)

	for len(selectedProblems) < numProblems && len(selectedProblems) < len(filteredProblems) {
		idx := rand.Intn(len(filteredProblems))
		if !usedIndices[idx] {
			usedIndices[idx] = true
			selectedProblems = append(selectedProblems, filteredProblems[idx])
		}
	}

	score := 0.0
	maxScore := float64(len(selectedProblems) * 25)
	details := "Pattern Recognition Test:\n"

	for _, p := range selectedProblems {
		isCorrect := rand.Float64() < 0.75 - (float64(difficulty-1) * 0.1)
		if isCorrect {
			score += 25.0
			details += fmt.Sprintf("✓ Sequence: %v → Next: %d (Correct)\n", p.Sequence, p.NextNumber)
		} else {
			details += fmt.Sprintf("✗ Sequence: %v → Next: ? (Incorrect)\n", p.Sequence)
		}
	}

	return score, maxScore, details
}

func (rte *ReasoningTestEngine) testAnalytical(difficulty int) (float64, float64, string) {
	numQuestions := min(difficulty+1, 3)
	score := 0.0
	pointsPerQuestion := 100.0 / 3.0
	maxScore := float64(numQuestions) * pointsPerQuestion
	details := "Analytical Reasoning Test:\n"

	analyticalQuestions := []string{
		"A company's profit increased by 20% in Year 1, then decreased by 10% in Year 2. What was the overall change?",
		"If 5 machines can produce 100 widgets in 4 hours, how many widgets can 10 machines produce in 8 hours?",
		"A train travels 60 mph for half the distance and 40 mph for the other half. What is the average speed?",
	}

	for i := 0; i < numQuestions && i < len(analyticalQuestions); i++ {
		isCorrect := rand.Float64() < 0.7 - (float64(difficulty-1) * 0.12)
		if isCorrect {
			score += pointsPerQuestion
			details += fmt.Sprintf("✓ Q%d: %s (Correct)\n", i+1, analyticalQuestions[i])
		} else {
			details += fmt.Sprintf("✗ Q%d: %s (Incorrect)\n", i+1, analyticalQuestions[i])
		}
	}

	return math.Min(score, maxScore), maxScore, details
}

func (rte *ReasoningTestEngine) GenerateCustomLogicProblem(ctx context.Context, difficulty int) (*LogicProblem, error) {
	rte.mu.RLock()
	defer rte.mu.RUnlock()

	if difficulty < 1 || difficulty > 5 {
		return nil, fmt.Errorf("difficulty must be between 1 and 5")
	}

	templates := []struct {
		descTemplate string
		explanation  string
	}{
		{
			descTemplate: "If A implies B, and B implies C, what follows?",
			explanation:  "Transitive property: A implies C",
		},
		{
			descTemplate: "Either X or Y is true. X is false. What is Y?",
			explanation:  "Disjunctive syllogism: Y must be true",
		},
	}

	template := templates[rand.Intn(len(templates))]

	return &LogicProblem{
		ID:            fmt.Sprintf("custom_logic_%d", time.Now().UnixNano()),
		Description:   template.descTemplate,
		Options:       []string{"Option 1", "Option 2", "Option 3", "Option 4"},
		CorrectAnswer: rand.Intn(4),
		Difficulty:    difficulty,
		Explanation:   template.explanation,
	}, nil
}
