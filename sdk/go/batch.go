package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type BatchRequest struct {
	SessionID string
	Type      string
	Data      interface{}
}

type BatchResponse struct {
	Index   int
	Success bool
	Data    interface{}
	Error   error
}

type BatchResult struct {
	Total      int
	Successful int
	Failed     int
	Results    []BatchResponse
}

type BatchClient struct {
	client     *Client
	workerPool int
	maxRetries int
}

func NewBatchClient(client *Client, workers int, retries int) *BatchClient {
	if workers <= 0 {
		workers = 5
	}
	if retries < 0 {
		retries = 0
	}
	return &BatchClient{
		client:     client,
		workerPool: workers,
		maxRetries: retries,
	}
}

func (bc *BatchClient) BatchVerify(ctx context.Context, requests []BatchRequest) *BatchResult {
	result := &BatchResult{
		Total:   len(requests),
		Results: make([]BatchResponse, len(requests)),
	}

	if len(requests) == 0 {
		return result
	}

	jobs := make(chan BatchRequest, len(requests))
	results := make(chan BatchResponse, len(requests))

	var wg sync.WaitGroup

	for i := 0; i < bc.workerPool; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range jobs {
				select {
				case <-ctx.Done():
					results <- BatchResponse{
						Index:   -1,
						Success: false,
						Error:   ctx.Err(),
					}
					continue
				default:
				}

				resp, err := bc.verifyWithRetry(req)
				results <- BatchResponse{
					Index:   findRequestIndex(requests, req),
					Success: err == nil && resp.Success,
					Data:    resp,
					Error:   err,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, req := range requests {
			jobs <- req
		}
		close(jobs)
	}()

	for resp := range results {
		if resp.Index >= 0 && resp.Index < len(result.Results) {
			result.Results[resp.Index] = resp
			if resp.Success {
				result.Successful++
			} else {
				result.Failed++
			}
		}
	}

	return result
}

func (bc *BatchClient) verifyWithRetry(req BatchRequest) (*VerifyCaptchaResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= bc.maxRetries; attempt++ {
		if attempt > 0 {
			delay := RetryStrategy(attempt, 100*time.Millisecond)
			time.Sleep(delay)
		}

		verifyReq := &VerifyCaptchaRequest{
			SessionID: req.SessionID,
		}

		switch req.Type {
		case "slider":
			if data, ok := req.Data.(map[string]interface{}); ok {
				if x, ok := data["x"].(int); ok {
					verifyReq.X = x
				}
				if y, ok := data["y"].(int); ok {
					verifyReq.Y = y
				}
				if trajectory, ok := data["trajectory"].([]TrajectoryPoint); ok {
					verifyReq.Trajectory = trajectory
				}
			}
		}

		resp, err := bc.client.VerifyCaptcha(verifyReq)
		if err == nil {
			return resp, nil
		}

		if !IsRetryableError(err) {
			return resp, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func findRequestIndex(requests []BatchRequest, target BatchRequest) int {
	for i, req := range requests {
		if req.SessionID == target.SessionID && req.Type == target.Type {
			return i
		}
	}
	return -1
}

type BulkCaptchaRequest struct {
	Type     string
	Width    int
	Height   int
	Tolerance int
	MaxPoints int
	Mode     string
}

type BulkCaptchaResponse struct {
	Index     int
	SessionID string
	ImageURL  string
	Error     error
}

func (c *Client) BulkGetCaptcha(ctx context.Context, requests []BulkCaptchaRequest) []BulkCaptchaResponse {
	responses := make([]BulkCaptchaResponse, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	concurrency := 10
	if len(requests) < concurrency {
		concurrency = len(requests)
	}

	sem := make(chan struct{}, concurrency)

	for i, req := range requests {
		wg.Add(1)
		sem <- struct{}{}

		go func(index int, r BulkCaptchaRequest) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				mu.Lock()
				responses[index] = BulkCaptchaResponse{
					Index: index,
					Error: ctx.Err(),
				}
				mu.Unlock()
				return
			default:
			}

			var resp interface{}
			var err error

			switch r.Type {
			case "slider":
				sliderResp, sliderErr := c.GetSliderCaptcha(r.Width, r.Height, r.Tolerance)
				if sliderErr != nil {
					err = sliderErr
				} else {
					resp = sliderResp
				}
			case "click":
				clickResp, clickErr := c.GetClickCaptcha(r.MaxPoints, r.Mode)
				if clickErr != nil {
					err = clickErr
				} else {
					resp = clickResp
				}
			default:
				err = fmt.Errorf("unsupported captcha type: %s", r.Type)
			}

			mu.Lock()
			if err != nil {
				responses[index] = BulkCaptchaResponse{
					Index: index,
					Error: err,
				}
			} else {
				switch r := resp.(type) {
				case *SliderCaptchaResponse:
					responses[index] = BulkCaptchaResponse{
						Index:     index,
						SessionID: r.SessionID,
						ImageURL:  r.ImageURL,
					}
				case *ClickCaptchaResponse:
					responses[index] = BulkCaptchaResponse{
						Index:     index,
						SessionID: r.SessionID,
						ImageURL:  r.ImageURL,
					}
				}
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return responses
}

func (c *Client) GetClickCaptcha(maxPoints int, mode string) (*ClickCaptchaResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/click", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if maxPoints > 0 {
		q.Add("points", fmt.Sprintf("%d", maxPoints))
	}
	if mode != "" {
		q.Add("mode", mode)
	}
	req.URL.RawQuery = q.Encode()

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    ClickCaptchaResponse   `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

type ClickCaptchaResponse struct {
	SessionID    string   `json:"session_id"`
	ImageURL     string   `json:"image_url"`
	Hint         string   `json:"hint"`
	HintOrder    []int    `json:"hint_order"`
	MaxPoints    int      `json:"max_points"`
	Mode         string   `json:"mode"`
	AllowShuffle bool     `json:"allow_shuffle"`
	Points       [][]int  `json:"points,omitempty"`
}

type BatchVerifyRequest struct {
	Items []BatchVerifyItem `json:"items"`
}

type BatchVerifyItem struct {
	SessionID   string             `json:"session_id"`
	Type        string             `json:"type"`
	X           int                `json:"x,omitempty"`
	Y           int                `json:"y,omitempty"`
	Points      [][]int            `json:"points,omitempty"`
	Trajectory  []TrajectoryPoint  `json:"trajectory,omitempty"`
}

type BatchVerifyResponse struct {
	Results []BatchVerifyResultItem `json:"results"`
}

type BatchVerifyResultItem struct {
	Index     int    `json:"index"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Remaining int    `json:"remaining_attempts,omitempty"`
}

func (c *Client) BatchVerifyCaptcha(req *BatchVerifyRequest) (*BatchVerifyResponse, error) {
	url := fmt.Sprintf("%s/api/v1/captcha/batch-verify", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    BatchVerifyResponse  `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

func NewSDKErrorFromResponse(code int, message string) *SDKError {
	return &SDKError{
		Code:    code,
		Message: message,
		Err:     nil,
	}
}

func (bc *BatchClient) GetWorkerPoolSize() int {
	return bc.workerPool
}

func (bc *BatchClient) SetWorkerPoolSize(size int) {
	if size > 0 {
		bc.workerPool = size
	}
}

func (bc *BatchClient) GetMaxRetries() int {
	return bc.maxRetries
}

func (bc *BatchClient) SetMaxRetries(retries int) {
	if retries >= 0 {
		bc.maxRetries = retries
	}
}
