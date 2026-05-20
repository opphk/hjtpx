package apidocs

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type InteractiveAPIDocs struct {
	swagger *SwaggerSpec
}

type SwaggerSpec struct {
	OpenAPI string `json:"openapi"`
	Info    Info   `json:"info"`
	Paths   map[string]PathItem `json:"paths"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type PathItem struct {
	Get  *Operation `json:"get,omitempty"`
	Post *Operation `json:"post,omitempty"`
}

type Operation struct {
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	OperationID string            `json:"operationId"`
	Responses   map[string]Response `json:"responses"`
}

type Response struct {
	Description string `json:"description"`
}

type APIEndpoint struct {
	Path    string
	Method  string
	Summary string
}

type DocsHandler struct {
	docs *InteractiveAPIDocs
}

func NewInteractiveAPIDocs() *InteractiveAPIDocs {
	docs := &InteractiveAPIDocs{
		swagger: &SwaggerSpec{
			OpenAPI: "3.1.0",
			Info: Info{
				Title:       "HJTPX 行为验证系统 API",
				Description: "提供完整的行为验证 API 接口文档",
				Version:     "4.0",
			},
			Paths: map[string]PathItem{
				"/captcha/slider/generate": {
					Post: &Operation{
						Summary:     "生成滑块验证码",
						Description: "生成滑块验证码，返回背景图和拼图图片URL",
						OperationID: "generateSliderCaptcha",
						Responses: map[string]Response{
							"200": {Description: "成功"},
							"400": {Description: "参数错误"},
						},
					},
				},
				"/captcha/slider/verify": {
					Post: &Operation{
						Summary:     "验证滑块验证码",
						Description: "验证用户滑块操作轨迹",
						OperationID: "verifySliderCaptcha",
						Responses: map[string]Response{
							"200": {Description: "验证结果"},
						},
					},
				},
			},
		},
	}

	return docs
}

func (h *DocsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/docs", h.handleDocs)
	mux.HandleFunc("/docs/api", h.handleAPIDocs)
	mux.HandleFunc("/docs/spec.json", h.handleSwaggerSpec)
}

func (h *DocsHandler) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<!DOCTYPE html><html><head><title>HJTPX API 文档</title>")
	fmt.Fprintf(w, "<link rel='stylesheet' href='https://cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5.3.0/css/bootstrap.min.css'>")
	fmt.Fprintf(w, "</head><body><div class='container mt-5'>")
	fmt.Fprintf(w, "<h1>HJTPX API 文档中心</h1>")
	fmt.Fprintf(w, "<p class='lead'>探索、测试 HJTPX API 的完整功能</p>")
	fmt.Fprintf(w, "<h3>端点列表</h3><ul>")

	for path, item := range h.docs.swagger.Paths {
		if item.Post != nil {
			fmt.Fprintf(w, "<li><strong>POST</strong> %s - %s</li>", path, item.Post.Summary)
		}
		if item.Get != nil {
			fmt.Fprintf(w, "<li><strong>GET</strong> %s - %s</li>", path, item.Get.Summary)
		}
	}

	fmt.Fprintf(w, "</ul></div></body></html>")
}

func (h *DocsHandler) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	endpoints := h.getAllEndpoints()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"endpoints": endpoints,
	})
}

func (h *DocsHandler) handleSwaggerSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.docs.swagger)
}

func (h *DocsHandler) getAllEndpoints() []APIEndpoint {
	var endpoints []APIEndpoint

	for path, pathItem := range h.docs.swagger.Paths {
		if pathItem.Get != nil {
			endpoints = append(endpoints, APIEndpoint{
				Path:    path,
				Method:  "GET",
				Summary: pathItem.Get.Summary,
			})
		}
		if pathItem.Post != nil {
			endpoints = append(endpoints, APIEndpoint{
				Path:    path,
				Method:  "POST",
				Summary: pathItem.Post.Summary,
			})
		}
	}

	return endpoints
}
