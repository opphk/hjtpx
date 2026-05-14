package captchax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	baseURL    string
	timeout    time.Duration
	retryTimes int
	headers    map[string]string
	client     *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration, retryTimes int) *HTTPClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &HTTPClient{
		baseURL:    baseURL,
		timeout:    timeout,
		retryTimes: retryTimes,
		headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

func (c *HTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

func (c *HTTPClient) buildURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "http") {
		return endpoint
	}
	endpoint = strings.TrimPrefix(endpoint, "/")
	return fmt.Sprintf("%s/%s", c.baseURL, endpoint)
}

func (c *HTTPClient) request(ctx context.Context, method, endpoint string, body interface{}, deduplicationID string) ([]byte, *Error) {
	var lastErr error
	for attempt := 0; attempt <= c.retryTimes; attempt++ {
		req, err := c.buildRequest(ctx, method, endpoint, body, deduplicationID)
		if err != nil {
			return nil, NewError(err.Error())
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retryTimes {
				time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
				continue
			}
			return nil, NewError(fmt.Sprintf("request failed after %d attempts: %v", c.retryTimes+1, err))
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, NewError(fmt.Sprintf("failed to read response: %v", err))
		}

		if resp.StatusCode >= 500 && attempt < c.retryTimes {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
			continue
		}

		if resp.StatusCode >= 400 {
			var apiResp APIResponse
			if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.Message != "" {
				return nil, NewErrorWithCode(apiResp.Message, apiResp.Code, resp.StatusCode)
			}
			return nil, NewErrorWithCode(fmt.Sprintf("HTTP error: %d", resp.StatusCode), resp.StatusCode, resp.StatusCode)
		}

		return respBody, nil
	}

	return nil, NewError(fmt.Sprintf("request failed after %d attempts: %v", c.retryTimes+1, lastErr))
}

func (c *HTTPClient) buildRequest(ctx context.Context, method, endpoint string, body interface{}, deduplicationID string) (*http.Request, error) {
	urlStr := c.buildURL(endpoint)
	var reqBody io.Reader

	if body != nil && method != http.MethodGet {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return nil, err
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	if deduplicationID != "" {
		req.Header.Set("X-Deduplication-ID", deduplicationID)
	}

	return req, nil
}

func (c *HTTPClient) Get(ctx context.Context, endpoint string) ([]byte, *Error) {
	return c.request(ctx, http.MethodGet, endpoint, nil, "")
}

func (c *HTTPClient) Post(ctx context.Context, endpoint string, body interface{}, deduplicationID string) ([]byte, *Error) {
	return c.request(ctx, http.MethodPost, endpoint, body, deduplicationID)
}

func (c *HTTPClient) Put(ctx context.Context, endpoint string, body interface{}) ([]byte, *Error) {
	return c.request(ctx, http.MethodPut, endpoint, body, "")
}

func (c *HTTPClient) Delete(ctx context.Context, endpoint string) ([]byte, *Error) {
	return c.request(ctx, http.MethodDelete, endpoint, nil, "")
}

func (c *HTTPClient) parseResponse(body []byte, v interface{}) *Error {
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return NewError(fmt.Sprintf("failed to parse response: %v", err))
	}

	if apiResp.Code != 0 && apiResp.Code != 200 {
		return NewErrorWithCode(apiResp.Message, apiResp.Code, 400)
	}

	if apiResp.Data != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return NewError(fmt.Sprintf("failed to marshal data: %v", err))
		}
		if err := json.Unmarshal(dataBytes, v); err != nil {
			return NewError(fmt.Sprintf("failed to unmarshal data: %v", err))
		}
	}

	return nil
}
