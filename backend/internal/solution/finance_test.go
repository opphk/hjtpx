package solution

import (
	"context"
	"testing"
	"time"
)

func TestFinanceSecurityService_ValidateTransaction(t *testing.T) {
	service := NewFinanceSecurityService().(FinanceSecurityService)

	tests := []struct {
		name      string
		tx        *Transaction
		wantValid bool
		wantErr   bool
	}{
		{
			name: "valid transaction",
			tx: &Transaction{
				TransactionID: "TX001",
				Amount:       1000.00,
				Currency:     "USD",
				TransactionType: "transfer",
				CustomerID:   "CUST001",
				Timestamp:    time.Now(),
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "invalid amount",
			tx: &Transaction{
				TransactionID: "TX002",
				Amount:        -100.00,
				Currency:      "USD",
				TransactionType: "transfer",
				Timestamp:    time.Now(),
			},
			wantValid: false,
			wantErr:   false,
		},
		{
			name: "missing transaction type",
			tx: &Transaction{
				TransactionID: "TX003",
				Amount:       500.00,
				Currency:      "USD",
				Timestamp:    time.Now(),
			},
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateTransaction(context.Background(), tt.tx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateTransaction() valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestFinanceSecurityService_DetectFraud(t *testing.T) {
	service := NewFinanceSecurityService().(FinanceSecurityService)

	tx := &Transaction{
		TransactionID: "TX001",
		Amount:       150000.00,
		Currency:     "USD",
		TransactionType: "high_value",
		CustomerID:   "CUST001",
		Timestamp:    time.Now(),
		Location: &GeoLocation{
			Country: "KP",
		},
	}

	result, err := service.DetectFraud(context.Background(), tx)
	if err != nil {
		t.Fatalf("DetectFraud() error = %v", err)
	}

	if result.FraudScore < 0.5 {
		t.Errorf("Expected high fraud score for risky transaction, got %v", result.FraudScore)
	}
}

func TestFinanceSecurityService_CalculateRiskScore(t *testing.T) {
	service := NewFinanceSecurityService().(FinanceSecurityService)

	tx := &Transaction{
		TransactionID: "TX001",
		Amount:       150000.00,
		Currency:     "USD",
		TransactionType: "high_value",
		Timestamp:    time.Now().Add(-4 * time.Hour),
		Location: &GeoLocation{
			Country: "KP",
		},
		DeviceFingerprint: "",
	}

	result, err := service.CalculateRiskScore(context.Background(), tx)
	if err != nil {
		t.Fatalf("CalculateRiskScore() error = %v", err)
	}

	if result.Level != "critical" {
		t.Errorf("Expected critical risk level, got %v", result.Level)
	}

	if result.Score < 0.7 {
		t.Errorf("Expected score >= 0.7, got %v", result.Score)
	}
}

func TestFinanceSecurityService_CheckAML(t *testing.T) {
	service := NewFinanceSecurityService().(FinanceSecurityService)

	tx := &Transaction{
		TransactionID: "TX001",
		Amount:       15000.00,
		Currency:     "USD",
		TransactionType: "transfer",
		CustomerID:   "CUST001",
		Timestamp:    time.Now(),
		Location: &GeoLocation{
			Country: "KP",
		},
	}

	result, err := service.CheckAML(context.Background(), tx)
	if err != nil {
		t.Fatalf("CheckAML() error = %v", err)
	}

	if result.RiskLevel != "high" {
		t.Errorf("Expected high risk level, got %v", result.RiskLevel)
	}
}

func TestFinanceSecurityService_GenerateComplianceReport(t *testing.T) {
	service := NewFinanceSecurityService().(FinanceSecurityService)

	report, err := service.GenerateComplianceReport(context.Background(), "monthly")
	if err != nil {
		t.Fatalf("GenerateComplianceReport() error = %v", err)
	}

	if report.TotalTransactions == 0 {
		t.Error("Expected non-zero total transactions")
	}

	if report.ComplianceMetrics["aml_compliance_rate"] == 0 {
		t.Error("Expected AML compliance rate to be set")
	}
}
