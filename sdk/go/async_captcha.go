package captcha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AsyncClient 异步验证码客户端
type AsyncClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewAsyncClient 创建新的异步验证码客户端
func NewAsyncClient(baseURL string, options ...Option) *AsyncClient {
	c := &AsyncClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range options {
		ac := (*Client)(c)
		opt(ac)
	}

	return c
}

// AsyncSliderCaptchaResponse 异步滑块验证码响应
type AsyncSliderCaptchaResponse struct {
	SessionID    string `json:"session_id"`
	ImageURL    string `json:"image_url"`
	PuzzleURL   string `json:"puzzle_url"`
	HintURL     string `json:"hint_url"`
	Shape       int    `json:"shape"`
	SecretY     int    `json:"secret_y"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
}

// AsyncVerifyCaptchaResponse 异步验证码验证响应
type AsyncVerifyCaptchaResponse struct {
	Success         bool                `json:"success"`
	Message         string              `json:"message"`
	RemainingAttempts int               `json:"remaining_attempts"`
	TrajectoryResult *TrajectoryResult  `json:"trajectory_result,omitempty"`
}

// GetSliderCaptchaAsync 异步获取滑块验证码
func (c *AsyncClient) GetSliderCaptchaAsync(width, height, tolerance int) <-chan *AsyncResult[AsyncSliderCaptchaResponse] {
	result := make(chan *AsyncResult[AsyncSliderCaptchaResponse], 1)

	go func() {
		url := fmt.Sprintf("%s/api/v1/captcha/slider", c.baseURL)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			result <- &AsyncResult[AsyncSliderCaptchaResponse]{Error: err}
			return
		}

		q := req.URL.Query()
		if width > 0 {
			q.Add("width", fmt.Sprintf("%d", width))
		}
		if height > 0 {
			q.Add("height", fmt.Sprintf("%d", height))
		}
		if tolerance > 0 {
			q.Add("tolerance", fmt.Sprintf("%d", tolerance))
		}
		req.URL.RawQuery = q.Encode()

		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			result <- &AsyncResult[AsyncSliderCaptchaResponse]{Error: err}
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result <- &AsyncResult[AsyncSliderCaptchaResponse]{Error: err}
			return
		}

		var apiResult struct {
			Code    int                        `json:"code"`
			Message string                     `json:"message"`
			Data    AsyncSliderCaptchaResponse `json:"data"`
		}

		if err := json.Unmarshal(body, &apiResult); err != nil {
			result <- &AsyncResult[AsyncSliderCaptchaResponse]{Error: err}
			return
		}

		if apiResult.Code != 0 {
			result <- &AsyncResult[AsyncSliderCaptchaResponse]{
				Error: fmt.Errorf("API error: %s", apiResult.Message),
			}
			return
		}

		result <- &AsyncResult[AsyncSliderCaptchaResponse]{Data: apiResult.Data}
	}()

	return result
}

// VerifyCaptchaAsync 异步验证验证码
func (c *AsyncClient) VerifyCaptchaAsync(req *VerifyCaptchaRequest) <-chan *AsyncResult[AsyncVerifyCaptchaResponse] {
	result := make(chan *AsyncResult[AsyncVerifyCaptchaResponse], 1)

	go func() {
		url := fmt.Sprintf("%s/api/v1/captcha/verify", c.baseURL)

		reqBody, err := json.Marshal(req)
		if err != nil {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Error: err}
			return
		}

		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Error: err}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("X-API-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Error: err}
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Error: err}
			return
		}

		var apiResult struct {
			Code    int                          `json:"code"`
			Message string                       `json:"message"`
			Data    AsyncVerifyCaptchaResponse   `json:"data"`
		}

		if err := json.Unmarshal(body, &apiResult); err != nil {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Error: err}
			return
		}

		if apiResult.Code != 0 {
			result <- &AsyncResult[AsyncVerifyCaptchaResponse]{
				Error: fmt.Errorf("API error: %s", apiResult.Message),
			}
			return
		}

		result <- &AsyncResult[AsyncVerifyCaptchaResponse]{Data: apiResult.Data}
	}()

	return result
}

// AsyncResult 异步操作结果
type AsyncResult[T any] struct {
	Data  T
	Error error
}

// BatchGetSliderCaptcha 批量异步获取滑块验证码
func (c *AsyncClient) BatchGetSliderCaptcha(count int, width, height, tolerance int) []<-chan *AsyncResult[AsyncSliderCaptchaResponse] {
	channels := make([]<-chan *AsyncResult[AsyncSliderCaptchaResponse], count)
	for i := 0; i < count; i++ {
		channels[i] = c.GetSliderCaptchaAsync(width, height, tolerance)
	}
	return channels
}

// WaitAll 等待所有异步操作完成
func WaitAll[T any](channels []<-chan *AsyncResult[T]) ([]*AsyncResult[T], error) {
	results := make([]*AsyncResult[T], len(channels))
	errs := make([]error, 0)

	for i, ch := range channels {
		result := <-ch
		results[i] = result
		if result.Error != nil {
			errs = append(errs, result.Error)
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("multiple errors occurred: %v", errs)
	}

	return results, nil
}