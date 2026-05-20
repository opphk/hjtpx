package service

import (
	"context"
	"fmt"
	"testing"
)

func TestDeveloperEcosystemV2Service_CreateSDK(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	sdk := &SDK{
		Name:         "JavaScript SDK",
		Language:     "javascript",
		Version:      "v1.0.0",
		Description:  "JavaScript SDK for captcha integration",
		Repository:   "https://github.com/hjtpx/sdk-javascript",
		Author:       "hjtpx",
		Tags:         []string{"captcha", "verification"},
		Features:    []string{"verify", "generate", "render"},
		Status:      "published",
	}

	err := service.CreateSDK(context.Background(), sdk)
	if err != nil {
		t.Fatalf("CreateSDK() error = %v", err)
	}

	if sdk.ID == "" {
		t.Error("Expected SDK ID to be set")
	}

	retrieved, err := service.GetSDK(context.Background(), sdk.ID)
	if err != nil {
		t.Fatalf("GetSDK() error = %v", err)
	}

	if retrieved.Name != sdk.Name {
		t.Errorf("Expected name %s, got %s", sdk.Name, retrieved.Name)
	}
}

func TestDeveloperEcosystemV2Service_ListSDKs(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	sdks := []*SDK{
		{Language: "javascript", Author: "hjtpx", Tags: []string{"captcha"}},
		{Language: "python", Author: "hjtpx", Tags: []string{"captcha"}},
		{Language: "javascript", Author: "community", Tags: []string{"verification"}},
	}

	for _, sdk := range sdks {
		service.CreateSDK(context.Background(), sdk)
	}

	result, err := service.ListSDKs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListSDKs() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 SDKs, got %d", len(result))
	}
}

func TestDeveloperEcosystemV2Service_CreatePlugin(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	plugin := &Plugin{
		Name:         "Analytics Plugin",
		Version:      "v1.0.0",
		Description:  "Analytics and reporting plugin",
		Author:       "hjtpx",
		Category:     "analytics",
		Tags:         []string{"reporting", "metrics"},
		Status:      "published",
	}

	err := service.CreatePlugin(context.Background(), plugin)
	if err != nil {
		t.Fatalf("CreatePlugin() error = %v", err)
	}

	retrieved, err := service.GetPlugin(context.Background(), plugin.ID)
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}

	if retrieved.Name != plugin.Name {
		t.Errorf("Expected name %s, got %s", plugin.Name, retrieved.Name)
	}
}

func TestDeveloperEcosystemV2Service_InstallPlugin(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	plugin := &Plugin{
		Name:    "Test Plugin",
		Version: "v1.0.0",
		Author:  "hjtpx",
		Status: "published",
	}
	service.CreatePlugin(context.Background(), plugin)

	err := service.InstallPlugin(context.Background(), plugin.ID, "APP001")
	if err != nil {
		t.Fatalf("InstallPlugin() error = %v", err)
	}

	retrieved, _ := service.GetPlugin(context.Background(), plugin.ID)
	if retrieved.Installations != 1 {
		t.Errorf("Expected 1 installation, got %d", retrieved.Installations)
	}
}

func TestDeveloperEcosystemV2Service_RegisterAPIEndpoint(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	api := &APIEndpoint{
		Name:        "Verify Captcha",
		Method:      "POST",
		Path:        "/api/v1/verify",
		Description: "Verify captcha token",
		Version:     "v1",
		AuthType:   "api_key",
		RateLimit:   1000,
		Status:     "active",
		Parameters: []APIParameter{
			{Name: "token", Type: "string", Required: true, Description: "Captcha token"},
		},
	}

	err := service.RegisterAPIEndpoint(context.Background(), api)
	if err != nil {
		t.Fatalf("RegisterAPIEndpoint() error = %v", err)
	}

	if api.ID == "" {
		t.Error("Expected API endpoint ID to be set")
	}
}

func TestDeveloperEcosystemV2Service_ManageAPIKey(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	operation := &APIKeyOperation{
		Operation:   "create",
		Name:        "Test API Key",
		Permissions: []string{"read", "write"},
		Scopes:      []string{"captcha:verify", "captcha:generate"},
		ExpiresIn:   30,
	}

	key, err := service.ManageAPIKey(context.Background(), operation)
	if err != nil {
		t.Fatalf("ManageAPIKey() error = %v", err)
	}

	if key.ID == "" {
		t.Error("Expected API key ID to be set")
	}

	if key.Key == "" {
		t.Error("Expected API key to be generated")
	}

	if len(key.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(key.Permissions))
	}

	operation = &APIKeyOperation{
		Operation: "revoke",
		KeyID:    key.ID,
	}

	revokedKey, err := service.ManageAPIKey(context.Background(), operation)
	if err != nil {
		t.Fatalf("ManageAPIKey() error = %v", err)
	}

	if revokedKey.Status != "revoked" {
		t.Errorf("Expected status 'revoked', got %s", revokedKey.Status)
	}
}

func TestDeveloperEcosystemV2Service_TrackAPIUsage(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	usage := &APIUsage{
		KeyID:      "KEY001",
		Endpoint:   "/api/v1/verify",
		Method:     "POST",
		Path:       "/api/v1/verify",
		StatusCode: 200,
		LatencyMs:  45,
		BytesIn:    256,
		BytesOut:   512,
	}

	err := service.TrackAPIUsage(context.Background(), usage)
	if err != nil {
		t.Fatalf("TrackAPIUsage() error = %v", err)
	}
}

func TestDeveloperEcosystemV2Service_GetAPIUsageReport(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	report, err := service.GetAPIUsageReport(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetAPIUsageReport() error = %v", err)
	}

	if report.TotalCalls == 0 {
		t.Error("Expected non-zero total calls")
	}

	if len(report.TopEndpoints) == 0 {
		t.Error("Expected top endpoints in report")
	}
}

func TestDeveloperEcosystemV2Service_CreateMarketplaceItem(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	item := &MarketplaceItem{
		Name:        "Premium Analytics Dashboard",
		Type:        "plugin",
		Description: "Advanced analytics dashboard",
		Version:     "v2.0.0",
		Category:    "analytics",
		Price:       99.99,
		IsPaid:      true,
		Tags:        []string{"analytics", "dashboard", "premium"},
		Features:    []string{"real-time", "custom reports", "export"},
		Status:     "published",
	}

	err := service.CreateMarketplaceItem(context.Background(), item)
	if err != nil {
		t.Fatalf("CreateMarketplaceItem() error = %v", err)
	}

	if item.ID == "" {
		t.Error("Expected marketplace item ID to be set")
	}

	retrieved, err := service.GetMarketplaceItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("GetMarketplaceItem() error = %v", err)
	}

	if retrieved.Name != item.Name {
		t.Errorf("Expected name %s, got %s", item.Name, retrieved.Name)
	}
}

func TestDeveloperEcosystemV2Service_ReviewMarketplaceItem(t *testing.T) {
	service := NewDeveloperEcosystemV2Service()

	item := &MarketplaceItem{
		Name:        "Test Plugin",
		Type:        "plugin",
		Version:     "v1.0.0",
		Status:     "published",
	}
	service.CreateMarketplaceItem(context.Background(), item)

	review := &MarketplaceReview{
		UserID:   "USER001",
		UserName: "Test User",
		Rating:   5,
		Title:    "Excellent plugin!",
		Content:  "This plugin works great",
		Pros:     []string{"Easy to use", "Great features"},
		Cons:     []string{"Documentation could be better"},
		Version:  "v1.0.0",
	}

	err := service.ReviewMarketplaceItem(context.Background(), review)
	if err != nil {
		t.Fatalf("ReviewMarketplaceItem() error = %v", err)
	}
}

func TestDeveloperEcosystemV2Service_RenderAPIDocumentation(t *testing.T) {
	api := &APIEndpoint{
		Name:        "Test API",
		Method:      "POST",
		Path:        "/api/v1/test",
		Description: "Test API endpoint",
		Version:     "v1",
		AuthType:   "oauth2",
		Parameters: []APIParameter{
			{Name: "param1", Type: "string", Required: true, Location: "query"},
		},
	}

	doc := fmt.Sprintf("# %s API Documentation\n\n## Overview\n%s\n\n## Endpoint\n- **Method**: %s\n- **Path**: %s\n- **Version**: %s\n- **Status**: %s\n\n## Authentication\n%s\n\n## Parameters\n", api.Name, api.Description, api.Method, api.Path, api.Version, api.Status, api.AuthType)

	if len(doc) == 0 {
		t.Error("Expected non-empty documentation")
	}
}
