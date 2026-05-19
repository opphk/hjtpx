package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeveloperEcosystemService(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	assert.NotNil(t, svc)
}

func TestCreatePlayground(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground := &Playground{
		Name:        "Test Playground",
		Description: "Test description",
		Category:    "captcha",
		Code:        "console.log('Hello');",
		Language:    "javascript",
		CreatedBy:   "user123",
		Tags:        []string{"captcha", "demo"},
	}

	err := svc.CreatePlayground(ctx, playground)

	require.NoError(t, err)
	assert.NotEmpty(t, playground.ID)
	assert.Equal(t, "18.0.0", playground.Version)
}

func TestGetPlayground(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground := &Playground{
		Name:      "Get Test",
		Language:  "javascript",
		CreatedBy: "user123",
	}

	err := svc.CreatePlayground(ctx, playground)
	require.NoError(t, err)

	retrieved, err := svc.GetPlayground(ctx, playground.ID)

	require.NoError(t, err)
	assert.Equal(t, playground.ID, retrieved.ID)
	assert.Equal(t, playground.Name, retrieved.Name)
}

func TestGetPlayground_NotFound(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground, err := svc.GetPlayground(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, playground)
	assert.Equal(t, ErrPlaygroundNotFound, err)
}

func TestGetPlayground_IncrementsViews(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground := &Playground{
		Name:      "Views Test",
		Language:  "javascript",
		CreatedBy: "user123",
		Views:     5,
	}

	err := svc.CreatePlayground(ctx, playground)
	require.NoError(t, err)

	_, err = svc.GetPlayground(ctx, playground.ID)
	require.NoError(t, err)

	assert.Equal(t, 6, playground.Views)
}

func TestListPlaygrounds(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		playground := &Playground{
			Name:      "List Test " + string(rune('0'+i)),
			Category:  "captcha",
			Language:  "javascript",
			CreatedBy: "user123",
		}
		err := svc.CreatePlayground(ctx, playground)
		require.NoError(t, err)
	}

	playgrounds, err := svc.ListPlaygrounds(ctx, "", 10, 0)

	require.NoError(t, err)
	assert.Len(t, playgrounds, 5)
}

func TestListPlaygrounds_FilterByCategory(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground1 := &Playground{Name: "Captcha PG", Category: "captcha", Language: "javascript"}
	playground2 := &Playground{Name: "IoT PG", Category: "iot", Language: "python"}

	err := svc.CreatePlayground(ctx, playground1)
	require.NoError(t, err)

	err = svc.CreatePlayground(ctx, playground2)
	require.NoError(t, err)

	captchaPGs, err := svc.ListPlaygrounds(ctx, "captcha", 10, 0)
	require.NoError(t, err)
	assert.Len(t, captchaPGs, 1)
	assert.Equal(t, "Captcha PG", captchaPGs[0].Name)
}

func TestExecutePlayground(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground := &Playground{
		Name:     "Execute Test",
		Code:     "console.log('Hello World');",
		Language: "javascript",
	}

	err := svc.CreatePlayground(ctx, playground)
	require.NoError(t, err)

	result, err := svc.ExecutePlayground(ctx, playground.ID, "")

	require.NoError(t, err)
	assert.NotEmpty(t, result.ExecutionID)
	assert.Equal(t, playground.ID, result.PlaygroundID)
	assert.Equal(t, "success", result.Status)
	assert.Contains(t, result.Output, "Hello World")
}

func TestExecutePlayground_CustomCode(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	playground := &Playground{
		Name:     "Custom Code Test",
		Code:     "original code",
		Language: "javascript",
	}

	err := svc.CreatePlayground(ctx, playground)
	require.NoError(t, err)

	result, err := svc.ExecutePlayground(ctx, playground.ID, "console.log('Custom');")

	require.NoError(t, err)
	assert.Contains(t, result.Output, "Custom")
}

func TestExecutePlayground_NotFound(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	result, err := svc.ExecutePlayground(ctx, "non-existent", "")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSaveLab(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	lab := &CodeLab{
		Title:       "Test Lab",
		Description: "Test description",
		UserID:      "user123",
		Files: []LabFile{
			{Path: "main.js", Content: "console.log('test');", Lang: "javascript"},
		},
		Difficulty: "beginner",
		Duration:  30,
		Tags:       []string{"captcha", "javascript"},
	}

	err := svc.SaveLab(ctx, lab)

	require.NoError(t, err)
	assert.NotEmpty(t, lab.ID)
}

func TestGetLab(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	lab := &CodeLab{
		Title:  "Get Lab Test",
		UserID: "user123",
	}

	err := svc.SaveLab(ctx, lab)
	require.NoError(t, err)

	retrieved, err := svc.GetLab(ctx, lab.ID)

	require.NoError(t, err)
	assert.Equal(t, lab.ID, retrieved.ID)
}

func TestGetLab_NotFound(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	lab, err := svc.GetLab(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, lab)
	assert.Equal(t, ErrLabNotFound, err)
}

func TestListLabs(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		lab := &CodeLab{
			Title:  "Lab " + string(rune('0'+i)),
			UserID: "user123",
		}
		err := svc.SaveLab(ctx, lab)
		require.NoError(t, err)
	}

	labs, err := svc.ListLabs(ctx, "user123", 10, 0)

	require.NoError(t, err)
	assert.Len(t, labs, 3)
}

func TestCreateBlogPost(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post := &BlogPost{
		Title:      "Test Post",
		Content:    "Test content",
		AuthorID:   "user123",
		AuthorName: "Test User",
		Category:   "tutorial",
		Tags:       []string{"captcha", "security"},
	}

	err := svc.CreateBlogPost(ctx, post)

	require.NoError(t, err)
	assert.NotEmpty(t, post.ID)
	assert.Equal(t, "draft", post.Status)
}

func TestGetBlogPost(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post := &BlogPost{
		Title:    "Blog Test",
		AuthorID: "user123",
	}

	err := svc.CreateBlogPost(ctx, post)
	require.NoError(t, err)

	retrieved, err := svc.GetBlogPost(ctx, post.ID)

	require.NoError(t, err)
	assert.Equal(t, post.ID, retrieved.ID)
}

func TestGetBlogPost_NotFound(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post, err := svc.GetBlogPost(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, post)
	assert.Equal(t, ErrPostNotFound, err)
}

func TestUpdateBlogPost(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post := &BlogPost{
		Title:    "Original Title",
		AuthorID: "user123",
		Status:   "draft",
	}

	err := svc.CreateBlogPost(ctx, post)
	require.NoError(t, err)

	post.Title = "Updated Title"
	post.Status = "published"

	err = svc.UpdateBlogPost(ctx, post)
	require.NoError(t, err)

	retrieved, err := svc.GetBlogPost(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, "published", retrieved.Status)
}

func TestListBlogPosts(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		post := &BlogPost{
			Title:    "Post " + string(rune('0'+i)),
			Category: "tutorial",
			AuthorID: "user123",
		}
		err := svc.CreateBlogPost(ctx, post)
		require.NoError(t, err)
	}

	posts, err := svc.ListBlogPosts(ctx, "", 10, 0)

	require.NoError(t, err)
	assert.Len(t, posts, 5)
}

func TestCreateComment(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post := &BlogPost{
		Title:    "Comment Test Post",
		AuthorID: "user123",
	}
	err := svc.CreateBlogPost(ctx, post)
	require.NoError(t, err)

	comment := &Comment{
		PostID:   post.ID,
		UserID:   "user456",
		UserName: "Commenter",
		Content:  "Great post!",
	}

	err = svc.CreateComment(ctx, comment)

	require.NoError(t, err)
	assert.NotEmpty(t, comment.ID)

	retrieved, err := svc.GetBlogPost(ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.Comments)
}

func TestGetComments(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	post := &BlogPost{Title: "Comments Test", AuthorID: "user123"}
	err := svc.CreateBlogPost(ctx, post)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		comment := &Comment{
			PostID:   post.ID,
			UserID:   "user" + string(rune('0'+i)),
			UserName: "User " + string(rune('0'+i)),
			Content:  "Comment " + string(rune('0'+i)),
		}
		err = svc.CreateComment(ctx, comment)
		require.NoError(t, err)
	}

	comments, err := svc.GetComments(ctx, post.ID)

	require.NoError(t, err)
	assert.Len(t, comments, 3)
}

func TestCreateQuestion(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{
		Title:      "How to implement captcha?",
		Body:       "I need help implementing captcha...",
		AuthorID:   "user123",
		AuthorName: "Asker",
		Tags:       []string{"captcha", "implementation"},
	}

	err := svc.CreateQuestion(ctx, question)

	require.NoError(t, err)
	assert.NotEmpty(t, question.ID)
	assert.Equal(t, "open", question.Status)
}

func TestGetQuestion(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{
		Title:    "Question Test",
		Body:     "Test body",
		AuthorID: "user123",
	}

	err := svc.CreateQuestion(ctx, question)
	require.NoError(t, err)

	retrieved, err := svc.GetQuestion(ctx, question.ID)

	require.NoError(t, err)
	assert.Equal(t, question.ID, retrieved.ID)
}

func TestGetQuestion_NotFound(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question, err := svc.GetQuestion(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, question)
	assert.Equal(t, ErrQuestionNotFound, err)
}

func TestListQuestions(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		question := &Question{
			Title:    "Question " + string(rune('0'+i)),
			Body:     "Body",
			AuthorID: "user123",
			Tags:     []string{"captcha"},
		}
		err := svc.CreateQuestion(ctx, question)
		require.NoError(t, err)
	}

	questions, err := svc.ListQuestions(ctx, nil, 10, 0)

	require.NoError(t, err)
	assert.Len(t, questions, 5)
}

func TestListQuestions_FilterByTags(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	q1 := &Question{Title: "Captcha Q", Tags: []string{"captcha"}}
	q2 := &Question{Title: "IoT Q", Tags: []string{"iot"}}
	q3 := &Question{Title: "Both Q", Tags: []string{"captcha", "iot"}}

	err := svc.CreateQuestion(ctx, q1)
	require.NoError(t, err)
	err = svc.CreateQuestion(ctx, q2)
	require.NoError(t, err)
	err = svc.CreateQuestion(ctx, q3)
	require.NoError(t, err)

	captchaQs, err := svc.ListQuestions(ctx, []string{"captcha"}, 10, 0)
	require.NoError(t, err)
	assert.Len(t, captchaQs, 2)
}

func TestAddAnswer(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{
		Title:    "Answer Test Q",
		Body:     "Test body",
		AuthorID: "user123",
	}

	err := svc.CreateQuestion(ctx, question)
	require.NoError(t, err)

	answer := &Answer{
		QuestionID: question.ID,
		Body:       "This is the answer...",
		AuthorID:   "user456",
		AuthorName: "Answerer",
	}

	err = svc.AddAnswer(ctx, answer)

	require.NoError(t, err)
	assert.NotEmpty(t, answer.ID)

	retrieved, err := svc.GetQuestion(ctx, question.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.AnswersCount)
}

func TestVoteQuestion(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{
		Title:    "Vote Test Q",
		Body:     "Test body",
		AuthorID: "user123",
		Upvotes:  0,
	}

	err := svc.CreateQuestion(ctx, question)
	require.NoError(t, err)

	err = svc.VoteQuestion(ctx, question.ID, "voter1", "up")

	require.NoError(t, err)
	assert.Equal(t, 1, question.Upvotes)
}

func TestVoteQuestion_ChangeVote(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{
		Title:    "Change Vote Q",
		Body:     "Test body",
		AuthorID: "user123",
	}

	err := svc.CreateQuestion(ctx, question)
	require.NoError(t, err)

	err = svc.VoteQuestion(ctx, question.ID, "voter1", "up")
	require.NoError(t, err)
	assert.Equal(t, 1, question.Upvotes)

	err = svc.VoteQuestion(ctx, question.ID, "voter1", "down")
	require.NoError(t, err)
	assert.Equal(t, 0, question.Upvotes)
	assert.Equal(t, 1, question.Downvotes)
}

func TestVoteAnswer(t *testing.T) {
	svc := NewDeveloperEcosystemService()
	ctx := context.Background()

	question := &Question{Title: "Answer Vote Q", Body: "Body", AuthorID: "user123"}
	err := svc.CreateQuestion(ctx, question)
	require.NoError(t, err)

	answer := &Answer{
		QuestionID: question.ID,
		Body:       "Answer body",
		AuthorID:   "user456",
	}

	err = svc.AddAnswer(ctx, answer)
	require.NoError(t, err)

	err = svc.VoteAnswer(ctx, answer.ID, "voter1", "up")

	require.NoError(t, err)
	assert.Equal(t, 1, answer.Upvotes)
}
