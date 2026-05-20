package community

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CommunityPortal struct {
	forums    *ForumStore
	tutorials *TutorialStore
}

type ForumStore struct {
	forums map[string]*Forum
	posts  map[string][]*ForumPost
	mu     sync.RWMutex
}

type Forum struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	PostCount int       `json:"post_count"`
}

type ForumPost struct {
	ID        string    `json:"id"`
	ForumID   string    `json:"forum_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	Views     int       `json:"views"`
	Likes     int       `json:"likes"`
}

type TutorialStore struct {
	tutorials map[string]*Tutorial
	mu        sync.RWMutex
}

type Tutorial struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Difficulty  string          `json:"difficulty"`
	Duration    int             `json:"duration_minutes"`
	Steps       []TutorialStep  `json:"steps"`
	Author      string          `json:"author"`
	Views       int             `json:"views"`
}

type TutorialStep struct {
	Order int    `json:"order"`
	Title string `json:"title"`
	Content string `json:"content"`
}

type PortalHandler struct {
	portal *CommunityPortal
}

func NewCommunityPortal() *CommunityPortal {
	portal := &CommunityPortal{
		forums:    NewForumStore(),
		tutorials: NewTutorialStore(),
	}

	portal.initSampleData()

	return portal
}

func NewForumStore() *ForumStore {
	return &ForumStore{
		forums: make(map[string]*Forum),
		posts:  make(map[string][]*ForumPost),
	}
}

func NewTutorialStore() *TutorialStore {
	return &TutorialStore{
		tutorials: make(map[string]*Tutorial),
	}
}

func (p *CommunityPortal) initSampleData() {
	p.forums.forums["general"] = &Forum{
		ID:        "general",
		Title:     "综合讨论",
		CreatedAt: time.Now().AddDate(0, -1, 0),
		PostCount: 156,
	}

	p.tutorials.tutorials["quickstart"] = &Tutorial{
		ID:          "quickstart",
		Title:       "快速开始：5分钟集成滑块验证",
		Description: "学习如何在5分钟内将滑块验证码集成到你的应用中",
		Category:    "Getting Started",
		Difficulty:  "Beginner",
		Duration:    5,
		Steps: []TutorialStep{
			{Order: 1, Title: "安装 SDK", Content: "首先安装 HJTPX Go SDK"},
			{Order: 2, Title: "初始化客户端", Content: "创建验证码客户端实例"},
			{Order: 3, Title: "生成验证码", Content: "在需要验证的页面生成验证码"},
		},
		Author: "HJTPX Team",
		Views:  1523,
	}
}

func (h *PortalHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/community", h.handleCommunity)
	mux.HandleFunc("/community/forums", h.handleForums)
	mux.HandleFunc("/community/tutorials", h.handleTutorials)
	mux.HandleFunc("/community/tutorials/", h.handleTutorialDetail)
}

func (h *PortalHandler) handleCommunity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<!DOCTYPE html><html><head><title>HJTPX 开发者社区</title>")
	fmt.Fprintf(w, "<link rel='stylesheet' href='https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.0/css/bootstrap.min.css'>")
	fmt.Fprintf(w, "</head><body>")
	fmt.Fprintf(w, "<nav class='navbar navbar-dark bg-dark'><div class='container'><a class='navbar-brand' href='/'>HJTPX</a></div></nav>")
	fmt.Fprintf(w, "<div class='container mt-5'><h1>HJTPX 开发者社区</h1>")
	fmt.Fprintf(w, "<p>与全球开发者一起成长</p>")

	tutorials := h.portal.tutorials.List()
	fmt.Fprintf(w, "<h3>热门教程</h3>")
	for _, t := range tutorials {
		fmt.Fprintf(w, "<div class='card mb-3'><div class='card-body'>")
		fmt.Fprintf(w, "<h5>%s</h5><p>%s</p>", t.Title, t.Description)
		fmt.Fprintf(w, "<small class='text-muted'>%s | %d分钟</small>", t.Difficulty, t.Duration)
		fmt.Fprintf(w, "</div></div>")
	}

	fmt.Fprintf(w, "</div></body></html>")
}

func (h *PortalHandler) handleForums(w http.ResponseWriter, r *http.Request) {
	forums := h.portal.forums.List()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"forums": forums,
	})
}

func (h *PortalHandler) handleTutorials(w http.ResponseWriter, r *http.Request) {
	tutorials := h.portal.tutorials.List()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tutorials": tutorials,
	})
}

func (h *PortalHandler) handleTutorialDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/community/tutorials/"):]
	tutorial := h.portal.tutorials.Get(id)

	if tutorial == nil {
		http.Error(w, "Tutorial not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tutorial)
}

func (s *ForumStore) List() []*Forum {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var forums []*Forum
	for _, f := range s.forums {
		forums = append(forums, f)
	}
	return forums
}

func (s *TutorialStore) List() []*Tutorial {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tutorials []*Tutorial
	for _, t := range s.tutorials {
		tutorials = append(tutorials, t)
	}
	return tutorials
}

func (s *TutorialStore) Get(id string) *Tutorial {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tutorials[id]
}
