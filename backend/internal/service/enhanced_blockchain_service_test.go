package service

import (
	"context"
	"testing"
	"time"
)

func TestEnhancedBlockchainService_DeploySmartContract(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	contract := &SmartContract{
		ContractID:   "test-contract-001",
		ContractType: "verification_record",
		ChainID:      "ethereum",
		SourceCode:   "pragma solidity ^0.8.0; contract VerificationRecord { }",
		ABI:          `[{"inputs":[],"name":"record","outputs":[],"type":"constructor"}]`,
		Bytecode:     "0x608060405234801561001057600080fd5b50",
		Params: map[string]string{
			"appID": "test-app",
		},
		GasLimit: 500000,
	}

	proof, err := service.DeploySmartContract(ctx, contract)
	if err != nil {
		t.Fatalf("DeploySmartContract failed: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof should not be nil")
	}

	if proof.ContractID != contract.ContractID {
		t.Errorf("ContractID mismatch: expected %s, got %s", contract.ContractID, proof.ContractID)
	}

	if proof.ChainID != "ethereum" {
		t.Errorf("ChainID mismatch: expected ethereum, got %s", proof.ChainID)
	}

	if proof.Status != "deployed" {
		t.Errorf("Status mismatch: expected deployed, got %s", proof.Status)
	}

	if len(proof.Events) == 0 {
		t.Error("Events should not be empty")
	}
}

func TestEnhancedBlockchainService_GenerateZKProof(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	request := &ZKProofRequest{
		ProofType: "verification_proof",
		PublicInputs: map[string]string{
			"appID":     "test-app",
			"timestamp": "1234567890",
		},
		PrivateInputs: map[string]string{
			"secret": "very-secret-key",
		},
		CircuitID: "verification_circuit_v1",
	}

	response, err := service.GenerateZKProof(ctx, request)
	if err != nil {
		t.Fatalf("GenerateZKProof failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.ProofID == "" {
		t.Error("ProofID should not be empty")
	}

	if response.Status != "generated" {
		t.Errorf("Status mismatch: expected generated, got %s", response.Status)
	}
}

func TestEnhancedBlockchainService_InitiateCrossChainTransfer(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	transfer := &CrossChainTransfer{
		TransferID:  "transfer-001",
		SourceChain: "ethereum",
		TargetChain: "polygon",
		AssetType:   "verification_record",
		Amount:      "1",
		Recipient:   "0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
		Sender:      "0x1234567890ABCDEF1234567890ABCDEF12345678",
		RelayerFee:  "0.01",
		Slippage:    "0.5",
	}

	bridge, err := service.InitiateCrossChainTransfer(ctx, transfer)
	if err != nil {
		t.Fatalf("InitiateCrossChainTransfer failed: %v", err)
	}

	if bridge == nil {
		t.Fatal("Bridge should not be nil")
	}

	if bridge.SourceChain != "ethereum" {
		t.Errorf("SourceChain mismatch: expected ethereum, got %s", bridge.SourceChain)
	}
}

func TestEnhancedBlockchainService_GetBlockchainAuditTrail(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	report, err := service.GetBlockchainAuditTrail(ctx, "test-app", start, end)
	if err != nil {
		t.Fatalf("GetBlockchainAuditTrail failed: %v", err)
	}

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.AppID != "test-app" {
		t.Errorf("AppID mismatch: expected test-app, got %s", report.AppID)
	}

	if report.TotalRecords == 0 {
		t.Error("TotalRecords should be greater than 0")
	}

	if report.MerkleRoot == "" {
		t.Error("MerkleRoot should not be empty")
	}
}

func TestEnhancedBlockchainService_VerifyAuditIntegrity(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	report, err := service.GetBlockchainAuditTrail(ctx, "test-app", start, end)
	if err != nil {
		t.Fatalf("GetBlockchainAuditTrail failed: %v", err)
	}

	verification, err := service.VerifyAuditIntegrity(ctx, report.ReportID)
	if err != nil {
		t.Fatalf("VerifyAuditIntegrity failed: %v", err)
	}

	if verification == nil {
		t.Fatal("Verification should not be nil")
	}

	if !verification.IsValid {
		t.Error("Audit trail should be valid")
	}
}

func TestEnhancedBlockchainService_InvalidInputs(t *testing.T) {
	service := NewEnhancedBlockchainService()
	ctx := context.Background()

	_, err := service.DeploySmartContract(ctx, nil)
	if err == nil {
		t.Error("DeploySmartContract should fail with nil contract")
	}

	_, err = service.GenerateZKProof(ctx, nil)
	if err == nil {
		t.Error("GenerateZKProof should fail with nil request")
	}

	_, err = service.InitiateCrossChainTransfer(ctx, nil)
	if err == nil {
		t.Error("InitiateCrossChainTransfer should fail with nil transfer")
	}

	_, err = service.GetCrossChainStatus(ctx, "non-existent")
	if err == nil {
		t.Error("GetCrossChainStatus should fail with non-existent bridge")
	}
}
