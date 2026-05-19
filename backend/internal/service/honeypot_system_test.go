package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHoneypotService_EvaluateRequest(t *testing.T) {
	service := NewHoneypotService()
	ctx := context.Background()

	tests := []struct {
		name           string
		path           string
		wantHoneypot   bool
	}{
		{
			name:         "Normal path",
			path:         "/api/users",
			wantHoneypot: false,
		},
		{
			name:         "Admin honeypot",
			path:         "/admin",
			wantHoneypot: true,
		},
		{
			name:         "wp-admin honeypot",
			path:         "/wp-admin",
			wantHoneypot: true,
		},
		{
			name:         "phpmyadmin honeypot",
			path:         "/phpmyadmin",
			wantHoneypot: true,
		},
		{
			name:         ".env honeypot",
			path:         "/.env",
			wantHoneypot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path, nil)
			result, err := service.EvaluateRequest(ctx, req)
			if err != nil {
				t.Errorf("EvaluateRequest() error = %v", err)
				return
			}

			if result.IsHoneypot != tt.wantHoneypot {
				t.Errorf("EvaluateRequest() IsHoneypot = %v, want %v", result.IsHoneypot, tt.wantHoneypot)
			}
		})
	}
}

func TestHoneypotService_CheckCredential(t *testing.T) {
	service := NewHoneypotService()

	tests := []struct {
		name     string
		username string
		password string
		wantHit  bool
	}{
		{
			name:     "Valid decoy credential",
			username: "admin",
			password: "admin123",
			wantHit:  true,
		},
		{
			name:     "Another valid decoy",
			username: "root",
			password: "toor",
			wantHit:  true,
		},
		{
			name:     "Invalid credential",
			username: "admin",
			password: "wrongpassword",
			wantHit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, decoy := service.CheckCredential(tt.username, tt.password)
			if hit != tt.wantHit {
				t.Errorf("CheckCredential() hit = %v, want %v", hit, tt.wantHit)
			}
			if hit && decoy == nil {
				t.Error("CheckCredential() returned nil decoy when hit is true")
			}
		})
	}
}

func TestHoneypotService_GetHoneypots(t *testing.T) {
	service := NewHoneypotService()

	honeypots := service.GetAllHoneypots()
	if len(honeypots) == 0 {
		t.Error("GetAllHoneypots() returned empty list")
	}

	for _, hp := range honeypots {
		if hp.ID == "" {
			t.Error("GetAllHoneypots() returned honeypot with empty ID")
		}
	}
}

func TestHoneypotService_GetHoneypotByID(t *testing.T) {
	service := NewHoneypotService()

	honeypots := service.GetAllHoneypots()
	if len(honeypots) == 0 {
		t.Skip("No honeypots available")
	}

	honeypot, err := service.GetHoneypotByID(honeypots[0].ID)
	if err != nil {
		t.Errorf("GetHoneypotByID() error = %v", err)
	}

	if honeypot == nil {
		t.Error("GetHoneypotByID() returned nil")
	}
}

func TestHoneypotService_UpdateHoneypot(t *testing.T) {
	service := NewHoneypotService()

	honeypots := service.GetAllHoneypots()
	if len(honeypots) == 0 {
		t.Skip("No honeypots available")
	}

	updates := &Honeypot{
		Name:     "Updated Honeypot Name",
		IsActive: false,
	}

	err := service.UpdateHoneypot(honeypots[0].ID, updates)
	if err != nil {
		t.Errorf("UpdateHoneypot() error = %v", err)
	}
}

func TestHoneypotService_GetHoneypotStatistics(t *testing.T) {
	service := NewHoneypotService()

	stats := service.GetHoneypotStatistics()

	if stats == nil {
		t.Error("GetHoneypotStatistics() returned nil")
	}

	if stats["total_honeypots"] == nil {
		t.Error("GetHoneypotStatistics() missing total_honeypots")
	}

	if stats["decoy_credentials"] == nil {
		t.Error("GetHoneypotStatistics() missing decoy_credentials")
	}
}

func TestHoneypotService_GetDecoyCredentials(t *testing.T) {
	service := NewHoneypotService()

	creds := service.GetDecoyCredentials()
	if len(creds) == 0 {
		t.Error("GetDecoyCredentials() returned empty list")
	}
}

func TestHoneypotService_GetDecoyFiles(t *testing.T) {
	service := NewHoneypotService()

	files := service.GetDecoyFiles()
	if len(files) == 0 {
		t.Error("GetDecoyFiles() returned empty list")
	}
}

func TestHoneypotService_AddDecoyCredential(t *testing.T) {
	service := NewHoneypotService()

	initialCount := len(service.decoyCredentials)

	cred := &DecoyCredential{
		Username:    "testuser",
		Password:    "testpass",
		TargetSystem: "test",
	}

	err := service.AddDecoyCredential(cred)
	if err != nil {
		t.Errorf("AddDecoyCredential() error = %v", err)
	}

	if len(service.decoyCredentials) != initialCount+1 {
		t.Errorf("AddDecoyCredential() did not add credential")
	}
}

func TestHoneypotService_AddDecoyFile(t *testing.T) {
	service := NewHoneypotService()

	file := &DecoyFile{
		Path:    "/test/path",
		Content: "test content",
	}

	err := service.AddDecoyFile(file)
	if err != nil {
		t.Errorf("AddDecoyFile() error = %v", err)
	}
}

func TestHoneypotService_EnableDisable(t *testing.T) {
	service := NewHoneypotService()

	service.Disable()
	if service.IsEnabled() {
		t.Error("Disable() did not disable service")
	}

	service.Enable()
	if !service.IsEnabled() {
		t.Error("Enable() did not enable service")
	}
}

func TestHoneypotService_SetTrapRate(t *testing.T) {
	service := NewHoneypotService()

	service.SetTrapRate(0.5)
	if service.trapRate != 0.5 {
		t.Errorf("SetTrapRate() trapRate = %v, want 0.5", service.trapRate)
	}

	service.SetTrapRate(1.5)
	if service.trapRate != 1.0 {
		t.Errorf("SetTrapRate() should cap at 1.0, got %v", service.trapRate)
	}

	service.SetTrapRate(-0.5)
	if service.trapRate != 0.0 {
		t.Errorf("SetTrapRate() should cap at 0.0, got %v", service.trapRate)
	}
}

func TestHoneypotService_GetActiveThreatActors(t *testing.T) {
	service := NewHoneypotService()

	actors := service.GetActiveThreatActors()
	if actors == nil {
		t.Error("GetActiveThreatActors() returned nil")
	}
}

func TestHoneypotService_GenerateReport(t *testing.T) {
	service := NewHoneypotService()

	report := service.GenerateReport()

	if report == nil {
		t.Error("GenerateReport() returned nil")
	}

	if report.Summary == nil {
		t.Error("GenerateReport() returned nil Summary")
	}
}

func TestHoneypotService_Export(t *testing.T) {
	service := NewHoneypotService()

	data, err := service.ExportHoneypotData()
	if err != nil {
		t.Errorf("ExportHoneypotData() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportHoneypotData() returned empty data")
	}
}

func TestHoneypotService_SimulateAttack(t *testing.T) {
	service := NewHoneypotService()

	result, err := service.SimulateAttack("sql_injection", "/api/test")
	if err != nil {
		t.Errorf("SimulateAttack() error = %v", err)
	}

	if result == nil {
		t.Error("SimulateAttack() returned nil")
	}
}
