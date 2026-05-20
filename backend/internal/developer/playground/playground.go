package playground

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PlaygroundServer struct {
	port      int
	codeExecutor *CodeExecutor
	snippets  *SnippetStore
}

type CodeExecutor struct {
	sandbox Sandbox
	timeout time.Duration
}

type Sandbox struct {
	allowNetwork bool
	allowFiles   bool
	maxMemory    int64
	maxCPU       int
}

type SnippetStore struct {
	snippets map[string]*CodeSnippet
	mu       sync.RWMutex
}

type CodeSnippet struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Language  string    `json:"language"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags"`
}

func NewPlaygroundServer(port int) *PlaygroundServer {
	ps := &PlaygroundServer{
		port: port,
		snippets: NewSnippetStore(),
		codeExecutor: &CodeExecutor{
			sandbox: Sandbox{
				allowNetwork: false,
				allowFiles:   false,
				maxMemory:    256 * 1024 * 1024,
				maxCPU:       2,
			},
			timeout: 5 * time.Second,
		},
	}

	ps.initDefaultSnippets()

	return ps
}

func NewSnippetStore() *SnippetStore {
	return &SnippetStore{
		snippets: make(map[string]*CodeSnippet),
	}
}

func (s *SnippetStore) Create(snippet *CodeSnippet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if snippet.ID == "" {
		snippet.ID = uuid.New().String()
	}
	snippet.CreatedAt = time.Now()

	s.snippets[snippet.ID] = snippet
	return nil
}

func (s *SnippetStore) Get(id string) (*CodeSnippet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snippet, ok := s.snippets[id]
	if !ok {
		return nil, fmt.Errorf("snippet not found: %s", id)
	}
	return snippet, nil
}

func (s *SnippetStore) List() []*CodeSnippet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*CodeSnippet
	for _, snippet := range s.snippets {
		results = append(results, snippet)
	}
	return results
}

func (ps *PlaygroundServer) initDefaultSnippets() {
	defaultSnippets := []*CodeSnippet{
		{
			ID:       "hello-world",
			Title:    "Hello World - Go",
			Language: "go",
			Code: `package main

import "fmt"

func main() {
    fmt.Println("Hello, HJTPX Playground!")
}`,
			Tags: []string{"hello-world", "beginner"},
		},
		{
			ID:       "slider-captcha",
			Title:    "滑块验证码集成",
			Language: "go",
			Code: `package main

import (
    "context"
    "fmt"
)

func main() {
    // 初始化客户端
    fmt.Println("Initializing HJTPX client...")

    ctx := context.Background()
    _ = ctx

    fmt.Println("Generate slider captcha...")
}`,
			Tags: []string{"captcha", "slider"},
		},
	}

	for _, snippet := range defaultSnippets {
		ps.snippets.Create(snippet)
	}
}

type ExecutionRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type ExecutionResult struct {
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration"`
}

func (ps *PlaygroundServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/playground", ps.handlePlayground)
	mux.HandleFunc("/api/snippets", ps.handleSnippets)
	mux.HandleFunc("/api/snippets/", ps.handleSnippetDetail)
	mux.HandleFunc("/api/execute", ps.handleExecute)
}

func (ps *PlaygroundServer) handlePlayground(w http.ResponseWriter, r *http.Request) {
	snippets := ps.snippets.List()

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<!DOCTYPE html><html><head><title>HJTPX Playground</title>")
	fmt.Fprintf(w, "<link rel='stylesheet' href='https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.0/css/bootstrap.min.css'>")
	fmt.Fprintf(w, "</head><body><div class='container mt-5'>")
	fmt.Fprintf(w, "<h1>HJTPX Code Playground</h1>")

	for _, s := range snippets {
		fmt.Fprintf(w, "<div class='card mb-3'><div class='card-body'>")
		fmt.Fprintf(w, "<h5>%s</h5><p>%s</p>", s.Title, s.Language)
		fmt.Fprintf(w, "<code>%s</code>", s.Code)
		fmt.Fprintf(w, "</div></div>")
	}

	fmt.Fprintf(w, "</div></body></html>")
}

func (ps *PlaygroundServer) handleSnippets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		snippets := ps.snippets.List()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"snippets": snippets,
		})
	}
}

func (ps *PlaygroundServer) handleSnippetDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/snippets/"):]

	switch r.Method {
	case http.MethodGet:
		snippet, err := ps.snippets.Get(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(snippet)
	}
}

func (ps *PlaygroundServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	result := &ExecutionResult{
		Output:   "Code execution simulated\nNote: In production, use actual sandboxed execution",
		Duration: time.Since(start).String(),
	}

	json.NewEncoder(w).Encode(result)
}

func (ps *PlaygroundServer) Start() error {
	mux := http.NewServeMux()
	ps.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", ps.port)
	log.Printf("Playground server starting on %s", addr)
	return http.ListenAndServe(addr, mux)
}
