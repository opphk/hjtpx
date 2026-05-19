package service

import (
	"context"
	"errors"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DeveloperEcosystemService interface {
	CreatePlayground(ctx context.Context, playground *Playground) error
	GetPlayground(ctx context.Context, id string) (*Playground, error)
	ListPlaygrounds(ctx context.Context, category string, limit, offset int) ([]*Playground, error)
	ExecutePlayground(ctx context.Context, id string, code string) (*ExecutionResult, error)
	SaveLab(ctx context.Context, lab *CodeLab) error
	GetLab(ctx context.Context, id string) (*CodeLab, error)
	ListLabs(ctx context.Context, userID string, limit, offset int) ([]*CodeLab, error)
	CreateBlogPost(ctx context.Context, post *BlogPost) error
	GetBlogPost(ctx context.Context, id string) (*BlogPost, error)
	ListBlogPosts(ctx context.Context, category string, limit, offset int) ([]*BlogPost, error)
	UpdateBlogPost(ctx context.Context, post *BlogPost) error
	CreateComment(ctx context.Context, comment *Comment) error
	GetComments(ctx context.Context, postID string) ([]*Comment, error)
	CreateQuestion(ctx context.Context, question *Question) error
	GetQuestion(ctx context.Context, id string) (*Question, error)
	ListQuestions(ctx context.Context, tags []string, limit, offset int) ([]*Question, error)
	AddAnswer(ctx context.Context, answer *Answer) error
	VoteQuestion(ctx context.Context, questionID, userID string, voteType string) error
	VoteAnswer(ctx context.Context, answerID, userID string, voteType string) error
}

type Playground struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Code        string    `json:"code"`
	Language    string    `json:"language"`
	Version     string    `json:"version"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ForkedFrom  string    `json:"forked_from,omitempty"`
	Tags        []string  `json:"tags"`
	Stars       int       `json:"stars"`
	Forks       int       `json:"forks"`
	Views       int       `json:"views"`
}

type ExecutionResult struct {
	ExecutionID string     `json:"execution_id"`
	PlaygroundID string   `json:"playground_id"`
	Status      string     `json:"status"`
	Output      string     `json:"output"`
	Error       string     `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time  `json:"timestamp"`
	Memory      int64      `json:"memory_usage"`
}

type CodeLab struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	UserID      string        `json:"user_id"`
	Files       []LabFile     `json:"files"`
	Exercises   []LabExercise `json:"exercises"`
	Solutions   map[string]string `json:"solutions,omitempty"`
	CompletedBy []string      `json:"completed_by"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Difficulty  string        `json:"difficulty"`
	Duration    int           `json:"estimated_duration_minutes"`
	Tags        []string      `json:"tags"`
	Rating      float64       `json:"rating"`
}

type LabFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Lang    string `json:"language"`
}

type LabExercise struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Instructions string  `json:"instructions"`
	TestCode    string   `json:"test_code"`
	Hints       []string `json:"hints"`
	Points      int      `json:"points"`
	Order       int      `json:"order"`
}

type BlogPost struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Summary   string    `json:"summary"`
	AuthorID  string    `json:"author_id"`
	AuthorName string   `json:"author_name"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Views     int       `json:"views"`
	Likes     int       `json:"likes"`
	Comments   int      `json:"comments_count"`
	Featured  bool      `json:"featured"`
}

type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Likes     int       `json:"likes"`
}

type Question struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	AuthorID    string    `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	Tags        []string  `json:"tags"`
	Upvotes     int       `json:"upvotes"`
	Downvotes   int       `json:"downvotes"`
	Views       int       `json:"views"`
	AnswersCount int      `json:"answers_count"`
	AcceptedAnswerID string `json:"accepted_answer_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
}

type Answer struct {
	ID         string    `json:"id"`
	QuestionID  string    `json:"question_id"`
	Body       string    `json:"body"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Upvotes    int       `json:"upvotes"`
	Downvotes  int       `json:"downvotes"`
	IsAccepted bool      `json:"is_accepted"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type developerEcosystemService struct {
	playgrounds map[string]*Playground
	labs       map[string]*CodeLab
	blogPosts  map[string]*BlogPost
	comments   map[string][]*Comment
	questions  map[string]*Question
	answers    map[string][]*Answer
	votes      map[string]map[string]int
}

var (
	ErrPlaygroundNotFound = errors.New("playground not found")
	ErrLabNotFound        = errors.New("lab not found")
	ErrPostNotFound       = errors.New("blog post not found")
	ErrQuestionNotFound   = errors.New("question not found")
)

func NewDeveloperEcosystemService() DeveloperEcosystemService {
	return &developerEcosystemService{
		playgrounds: make(map[string]*Playground),
		labs:       make(map[string]*CodeLab),
		blogPosts:  make(map[string]*BlogPost),
		comments:   make(map[string][]*Comment),
		questions:  make(map[string]*Question),
		answers:    make(map[string][]*Answer),
		votes:      make(map[string]map[string]int),
	}
}

func (s *developerEcosystemService) CreatePlayground(ctx context.Context, playground *Playground) error {
	if playground.ID == "" {
		playground.ID = uuid.New().String()
	}
	if playground.CreatedAt.IsZero() {
		playground.CreatedAt = time.Now()
	}
	playground.UpdatedAt = playground.CreatedAt
	if playground.Version == "" {
		playground.Version = "18.0.0"
	}

	s.playgrounds[playground.ID] = playground
	return nil
}

func (s *developerEcosystemService) GetPlayground(ctx context.Context, id string) (*Playground, error) {
	p, exists := s.playgrounds[id]
	if !exists {
		return nil, ErrPlaygroundNotFound
	}
	p.Views++
	return p, nil
}

func (s *developerEcosystemService) ListPlaygrounds(ctx context.Context, category string, limit, offset int) ([]*Playground, error) {
	var result []*Playground
	for _, p := range s.playgrounds {
		if category == "" || p.Category == category {
			result = append(result, p)
		}
	}
	if offset >= len(result) {
		return []*Playground{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (s *developerEcosystemService) ExecutePlayground(ctx context.Context, id string, code string) (*ExecutionResult, error) {
	playground, exists := s.playgrounds[id]
	if !exists {
		return nil, ErrPlaygroundNotFound
	}

	startTime := time.Now()
	result := &ExecutionResult{
		ExecutionID:  uuid.New().String(),
		PlaygroundID: id,
		Timestamp:    startTime,
	}

	if code == "" {
		code = playground.Code
	}

	switch strings.ToLower(playground.Language) {
	case "javascript", "js":
		result.Output = s.executeJavaScript(code)
	case "python", "py":
		result.Output = "# Python simulation\nprint('Hello from Python!')\n# Note: This is a simulation, not actual execution"
	case "go":
		result.Output = "// Go simulation\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello from Go!\")\n}\n// Note: This is a simulation, not actual execution"
	default:
		result.Output = "// Output simulation\nconsole.log('Code executed successfully!');"
	}

	result.Duration = time.Since(startTime)
	result.Status = "success"
	result.Memory = 1024 * 1024

	return result, nil
}

func (s *developerEcosystemService) executeJavaScript(code string) string {
	if strings.Contains(strings.ToLower(code), "error") {
		return "Error: Syntax error in code"
	}

	if strings.Contains(code, "console.log") {
		lines := strings.Split(code, "\n")
		var output []string
		for _, line := range lines {
			if strings.Contains(line, "console.log") {
				start := strings.Index(line, "console.log") + 11
				end := len(line)
				if idx := strings.Index(line[start:], ")"); idx != -1 {
					end = start + idx
				}
				content := strings.Trim(line[start:end], "('\" )")
				output = append(output, content)
			}
		}
		if len(output) > 0 {
			return strings.Join(output, "\n")
		}
	}

	return "Code executed successfully (simulation)"
}

func (s *developerEcosystemService) SaveLab(ctx context.Context, lab *CodeLab) error {
	if lab.ID == "" {
		lab.ID = uuid.New().String()
	}
	if lab.CreatedAt.IsZero() {
		lab.CreatedAt = time.Now()
	}
	lab.UpdatedAt = time.Now()

	s.labs[lab.ID] = lab
	return nil
}

func (s *developerEcosystemService) GetLab(ctx context.Context, id string) (*CodeLab, error) {
	lab, exists := s.labs[id]
	if !exists {
		return nil, ErrLabNotFound
	}
	return lab, nil
}

func (s *developerEcosystemService) ListLabs(ctx context.Context, userID string, limit, offset int) ([]*CodeLab, error) {
	var result []*CodeLab
	for _, lab := range s.labs {
		if userID == "" || lab.UserID == userID {
			result = append(result, lab)
		}
	}
	if offset >= len(result) {
		return []*CodeLab{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (s *developerEcosystemService) CreateBlogPost(ctx context.Context, post *BlogPost) error {
	if post.ID == "" {
		post.ID = uuid.New().String()
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}
	post.UpdatedAt = post.CreatedAt
	if post.Status == "" {
		post.Status = "draft"
	}

	s.blogPosts[post.ID] = post
	return nil
}

func (s *developerEcosystemService) GetBlogPost(ctx context.Context, id string) (*BlogPost, error) {
	post, exists := s.blogPosts[id]
	if !exists {
		return nil, ErrPostNotFound
	}
	post.Views++
	return post, nil
}

func (s *developerEcosystemService) ListBlogPosts(ctx context.Context, category string, limit, offset int) ([]*BlogPost, error) {
	var result []*BlogPost
	for _, post := range s.blogPosts {
		if category == "" || post.Category == category {
			result = append(result, post)
		}
	}
	if offset >= len(result) {
		return []*BlogPost{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (s *developerEcosystemService) UpdateBlogPost(ctx context.Context, post *BlogPost) error {
	if _, exists := s.blogPosts[post.ID]; !exists {
		return ErrPostNotFound
	}
	post.UpdatedAt = time.Now()
	s.blogPosts[post.ID] = post
	return nil
}

func (s *developerEcosystemService) CreateComment(ctx context.Context, comment *Comment) error {
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}

	s.comments[comment.PostID] = append(s.comments[comment.PostID], comment)

	if post, exists := s.blogPosts[comment.PostID]; exists {
		post.Comments++
	}

	return nil
}

func (s *developerEcosystemService) GetComments(ctx context.Context, postID string) ([]*Comment, error) {
	return s.comments[postID], nil
}

func (s *developerEcosystemService) CreateQuestion(ctx context.Context, question *Question) error {
	if question.ID == "" {
		question.ID = uuid.New().String()
	}
	if question.CreatedAt.IsZero() {
		question.CreatedAt = time.Now()
	}
	question.UpdatedAt = question.CreatedAt
	if question.Status == "" {
		question.Status = "open"
	}

	s.questions[question.ID] = question
	return nil
}

func (s *developerEcosystemService) GetQuestion(ctx context.Context, id string) (*Question, error) {
	q, exists := s.questions[id]
	if !exists {
		return nil, ErrQuestionNotFound
	}
	q.Views++
	return q, nil
}

func (s *developerEcosystemService) ListQuestions(ctx context.Context, tags []string, limit, offset int) ([]*Question, error) {
	var result []*Question
	for _, q := range s.questions {
		if len(tags) == 0 {
			result = append(result, q)
		} else {
			for _, tag := range tags {
				if containsTag(q.Tags, tag) {
					result = append(result, q)
					break
				}
			}
		}
	}
	if offset >= len(result) {
		return []*Question{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func (s *developerEcosystemService) AddAnswer(ctx context.Context, answer *Answer) error {
	if answer.ID == "" {
		answer.ID = uuid.New().String()
	}
	if answer.CreatedAt.IsZero() {
		answer.CreatedAt = time.Now()
	}
	answer.UpdatedAt = answer.CreatedAt

	s.answers[answer.QuestionID] = append(s.answers[answer.QuestionID], answer)

	if q, exists := s.questions[answer.QuestionID]; exists {
		q.AnswersCount++
	}

	return nil
}

func (s *developerEcosystemService) VoteQuestion(ctx context.Context, questionID, userID string, voteType string) error {
	key := "question:" + questionID
	if s.votes[key] == nil {
		s.votes[key] = make(map[string]int)
	}

	if existing, ok := s.votes[key][userID]; ok {
		if q, exists := s.questions[questionID]; exists {
			if existing == 1 {
				q.Upvotes--
			} else {
				q.Downvotes--
			}
		}
	}

	vote := 0
	if voteType == "up" {
		vote = 1
		if q, exists := s.questions[questionID]; exists {
			q.Upvotes++
		}
	} else if voteType == "down" {
		vote = -1
		if q, exists := s.questions[questionID]; exists {
			q.Downvotes++
		}
	}

	s.votes[key][userID] = vote
	return nil
}

func (s *developerEcosystemService) VoteAnswer(ctx context.Context, answerID, userID string, voteType string) error {
	key := "answer:" + answerID
	if s.votes[key] == nil {
		s.votes[key] = make(map[string]int)
	}

	for _, answers := range s.answers {
		for _, a := range answers {
			if a.ID == answerID {
				if existing, ok := s.votes[key][userID]; ok {
					if existing == 1 {
						a.Upvotes--
					} else {
						a.Downvotes--
					}
				}

				if voteType == "up" {
					a.Upvotes++
					s.votes[key][userID] = 1
				} else if voteType == "down" {
					a.Downvotes++
					s.votes[key][userID] = -1
				}
				return nil
			}
		}
	}

	return errors.New("answer not found")
}

func (s *developerEcosystemService) RenderBlogContent(content string) string {
	tmpl, err := template.New("blog").Parse(content)
	if err != nil {
		return content
	}

	var buf strings.Builder
	tmpl.Execute(&buf, nil)
	return buf.String()
}

func (s *developerEcosystemService) SearchContent(query string, searchType string) []interface{} {
	var results []interface{}

	switch searchType {
	case "playground":
		for _, p := range s.playgrounds {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(p.Description), strings.ToLower(query)) {
				results = append(results, p)
			}
		}
	case "lab":
		for _, lab := range s.labs {
			if strings.Contains(strings.ToLower(lab.Title), strings.ToLower(query)) {
				results = append(results, lab)
			}
		}
	case "blog":
		for _, post := range s.blogPosts {
			if strings.Contains(strings.ToLower(post.Title), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(post.Content), strings.ToLower(query)) {
				results = append(results, post)
			}
		}
	case "question":
		for _, q := range s.questions {
			if strings.Contains(strings.ToLower(q.Title), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(q.Body), strings.ToLower(query)) {
				results = append(results, q)
			}
		}
	default:
	}

	return results
}
