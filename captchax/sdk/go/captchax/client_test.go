package captchax

import (
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(NewConfig("https://captchax.example.com"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
}

func TestNewClientWithoutBaseURL(t *testing.T) {
	_, err := NewClient(NewConfig(""))
	if err == nil {
		t.Fatal("expected error when baseURL is empty")
	}
}

func TestConfigWithOptions(t *testing.T) {
	width := 300
	height := 150

	config := NewConfig("https://captchax.example.com").
		WithAppID("test-app").
		WithTimeout(5 * time.Second).
		WithRetryTimes(2).
		WithAPIVersion(APIVersionV2)

	if config.AppID != "test-app" {
		t.Errorf("expected AppID to be 'test-app', got '%s'", config.AppID)
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("expected Timeout to be 5s, got %v", config.Timeout)
	}

	if config.RetryTimes != 2 {
		t.Errorf("expected RetryTimes to be 2, got %d", config.RetryTimes)
	}

	if config.APIVersion != APIVersionV2 {
		t.Errorf("expected APIVersion to be V2, got %s", config.APIVersion)
	}

	_ = width
	_ = height
}

func TestClientSetAppID(t *testing.T) {
	client, _ := NewClient(NewConfig("https://captchax.example.com"))
	client.SetAppID("new-app-id")
}

func TestClientSetAPIVersion(t *testing.T) {
	client, _ := NewClient(NewConfig("https://captchax.example.com"))
	if client.GetAPIVersion() != APIVersionV1 {
		t.Errorf("expected default APIVersion to be V1, got %s", client.GetAPIVersion())
	}

	client.SetAPIVersion(APIVersionV2)
	if client.GetAPIVersion() != APIVersionV2 {
		t.Errorf("expected APIVersion to be V2, got %s", client.GetAPIVersion())
	}
}

func TestCreateClientInfo(t *testing.T) {
	client, _ := NewClient(NewConfig("https://captchax.example.com"))
	info := client.CreateClientInfo(map[string]interface{}{"key": "value"})

	if info == "" {
		t.Fatal("expected non-empty client info")
	}
}

func TestError(t *testing.T) {
	err := NewError("test error")
	if err.Error() != "CaptchaXError(500): test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	errWithCode := NewErrorWithCode("test", 400, 400)
	if errWithCode.Code != 400 {
		t.Errorf("expected code 400, got %d", errWithCode.Code)
	}
}

func TestNewClientWithDefault(t *testing.T) {
	client, err := NewClientWithDefault("https://captchax.example.com")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
