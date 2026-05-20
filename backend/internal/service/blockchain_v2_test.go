package service

import (
	"context"
	"testing"
	"time"
)

func TestBlockchainV2Service_RecordVerificationV2(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	record := &VerificationRecordV2{
		AppID:        "test_app",
		SessionID:    "test_session",
		EventType:    "login",
		EventData:    map[string]interface{}{"user": "test_user", "action": "login"},
		RiskLevel:    "low",
		RiskScore:    15.5,
		UserAgent:    "Mozilla/5.0",
		IPAddress:    "192.168.1.100",
		Country:      "US",
		DeviceFinger: "abc123",
	}

	proof, err := service.RecordVerificationV2(ctx, record)
	if err != nil {
		t.Fatalf("RecordVerificationV2 failed: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof should not be nil")
	}

	if proof.ProofID == "" {
		t.Error("ProofID should not be empty")
	}

	if proof.RecordID != record.RecordID {
		t.Errorf("RecordID mismatch: got %s, want %s", proof.RecordID, record.RecordID)
	}

	if proof.ChainID != "ethereum" {
		t.Errorf("ChainID should be ethereum, got %s", proof.ChainID)
	}

	if proof.Status != "confirmed" {
		t.Errorf("Status should be confirmed, got %s", proof.Status)
	}

	if proof.ZKProof == "" {
		t.Error("ZKProof should not be empty")
	}

	if proof.GasUsed == 0 {
		t.Error("GasUsed should not be zero")
	}
}

func TestBlockchainV2Service_VerifyProofV2(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	record := &VerificationRecordV2{
		AppID:        "test_app",
		SessionID:    "test_session",
		EventType:    "verification",
		EventData:    map[string]interface{}{"status": "success"},
		RiskLevel:    "low",
		RiskScore:    10.0,
		UserAgent:    "Mozilla/5.0",
		IPAddress:    "192.168.1.101",
		DeviceFinger: "def456",
	}

	proof, err := service.RecordVerificationV2(ctx, record)
	if err != nil {
		t.Fatalf("RecordVerificationV2 failed: %v", err)
	}

	verifiedProof, err := service.VerifyProofV2(ctx, proof.ProofID)
	if err != nil {
		t.Fatalf("VerifyProofV2 failed: %v", err)
	}

	if verifiedProof == nil {
		t.Fatal("VerifiedProof should not be nil")
	}

	if verifiedProof.Status != "verified" {
		t.Errorf("Status should be verified, got %s", verifiedProof.Status)
	}

	_, err = service.VerifyProofV2(ctx, "nonexistent_id")
	if err != ErrProofNotFoundV2 {
		t.Errorf("Should return ErrProofNotFoundV2 for nonexistent proof, got %v", err)
	}
}

func TestBlockchainV2Service_CreateSmartContract(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	contract := &SmartContract{
		Name:     "VerificationContract",
		Version:  "1.0.0",
		ChainID:  "ethereum",
		ABI:      `[{"name":"recordVerification","type":"function"}]`,
		Bytecode: "0x608060405234801561001057600080fd5b50",
		GasLimit: 100000,
	}

	contractID, err := service.CreateSmartContract(ctx, contract)
	if err != nil {
		t.Fatalf("CreateSmartContract failed: %v", err)
	}

	if contractID == "" {
		t.Error("ContractID should not be empty")
	}

	if contract.ContractID != contractID {
		t.Errorf("ContractID mismatch: got %s, want %s", contract.ContractID, contractID)
	}

	if !contract.IsVerified {
		t.Error("Contract should be verified after deployment")
	}

	if contract.Deployer == "" {
		t.Error("Deployer should be set after deployment")
	}
}

func TestBlockchainV2Service_ExecuteSmartContract(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	contract := &SmartContract{
		Name:     "TestContract",
		Version:  "1.0.0",
		ChainID:  "ethereum",
		ABI:      `[]`,
		Bytecode: "0x",
		GasLimit: 100000,
	}

	contractID, err := service.CreateSmartContract(ctx, contract)
	if err != nil {
		t.Fatalf("CreateSmartContract failed: %v", err)
	}

	params := map[string]interface{}{
		"user_id": "user123",
	}

	result, err := service.ExecuteSmartContract(ctx, contractID, "recordVerification", params)
	if err != nil {
		t.Fatalf("ExecuteSmartContract failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.Success {
		t.Error("Execution should be successful")
	}

	if result.Method != "recordVerification" {
		t.Errorf("Method should be recordVerification, got %s", result.Method)
	}

	if result.GasUsed == 0 {
		t.Error("GasUsed should not be zero")
	}

	if len(result.Events) == 0 {
		t.Error("Should have at least one event")
	}

	_, err = service.ExecuteSmartContract(ctx, "nonexistent", "method", params)
	if err != ErrContractNotFound {
		t.Errorf("Should return ErrContractNotFound, got %v", err)
	}
}

func TestBlockchainV2Service_VerifyCrossChainV2(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	request := &CrossChainRequestV2{
		SourceChain:      "ethereum",
		TargetChain:      "polkadot",
		Identity:         "user_identity_123",
		VerificationType: "trust_verification",
		Proof:            "proof_data",
		ZKProof:          "zk_proof_data",
		Metadata:         map[string]interface{}{"verification_level": "high"},
	}

	response, err := service.VerifyCrossChainV2(ctx, request)
	if err != nil {
		t.Fatalf("VerifyCrossChainV2 failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if !response.Valid {
		t.Error("Response should be valid")
	}

	if response.TrustScore == 0 {
		t.Error("TrustScore should not be zero")
	}

	if len(response.Proofs) != 2 {
		t.Errorf("Should have 2 proofs (source and target), got %d", len(response.Proofs))
	}

	for _, proof := range response.Proofs {
		if !proof.IsVerified {
			t.Error("Proof should be verified")
		}
	}
}

func TestBlockchainV2Service_SetupPolkadotBridge(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	config := &BridgeConfig{
		SourceChain: "ethereum",
		TargetChain: "polkadot",
		BridgeType: "parachain",
		Endpoint:   "wss://polkadot-rpc.example.com",
		APIKey:     "test_api_key",
	}

	err := service.SetupPolkadotBridge(ctx, config)
	if err != nil {
		t.Fatalf("SetupPolkadotBridge failed: %v", err)
	}

	if config.BridgeID == "" {
		t.Error("BridgeID should be set")
	}

	if config.ChainBridge != "polkadot" {
		t.Errorf("ChainBridge should be polkadot, got %s", config.ChainBridge)
	}

	if !config.IsActive {
		t.Error("Bridge should be active after setup")
	}
}

func TestBlockchainV2Service_SetupCosmosBridge(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	config := &BridgeConfig{
		SourceChain: "ethereum",
		TargetChain: "cosmos",
		BridgeType: "ibc",
		Endpoint:   "https://cosmos-rpc.example.com",
		APIKey:     "cosmos_api_key",
	}

	err := service.SetupCosmosBridge(ctx, config)
	if err != nil {
		t.Fatalf("SetupCosmosBridge failed: %v", err)
	}

	if config.ChainBridge != "cosmos" {
		t.Errorf("ChainBridge should be cosmos, got %s", config.ChainBridge)
	}
}

func TestBlockchainV2Service_TransferCrossChain(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	bridgeConfig := &BridgeConfig{
		SourceChain: "ethereum",
		TargetChain: "polkadot",
		BridgeType: "parachain",
	}
	err := service.SetupPolkadotBridge(ctx, bridgeConfig)
	if err != nil {
		t.Fatalf("SetupPolkadotBridge failed: %v", err)
	}

	transfer := &CrossChainTransfer{
		SourceChain: "ethereum",
		TargetChain: "polkadot",
		AssetType:   "ETH",
		Amount:      "1.5",
		Sender:      "0xsender123",
		Recipient:   "polkadot_recipient",
		RelayerFee:  "0.01",
	}

	result, err := service.TransferCrossChain(ctx, transfer)
	if err != nil {
		t.Fatalf("TransferCrossChain failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.Success {
		t.Error("Transfer should be successful")
	}

	if result.SourceTxHash == "" {
		t.Error("SourceTxHash should not be empty")
	}

	if result.TargetTxHash == "" {
		t.Error("TargetTxHash should not be empty")
	}

	if transfer.Status != "completed" {
		t.Errorf("Transfer status should be completed, got %s", transfer.Status)
	}
}

func TestBlockchainV2Service_GenerateZKProof(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	witness := &ZKWitness{
		PublicInputs: map[string]interface{}{
			"user_id":   "user123",
			"timestamp": 1234567890,
		},
		PrivateInputs: map[string]interface{}{
			"secret_key": "secret123",
		},
		Statement: "user_verification",
	}

	proof, err := service.GenerateZKProof(ctx, witness)
	if err != nil {
		t.Fatalf("GenerateZKProof failed: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof should not be nil")
	}

	if proof.ProofID == "" {
		t.Error("ProofID should not be empty")
	}

	if proof.WitnessID != witness.WitnessID {
		t.Errorf("WitnessID mismatch: got %s, want %s", proof.WitnessID, witness.WitnessID)
	}

	if proof.ProofData == "" {
		t.Error("ProofData should not be empty")
	}

	if proof.PublicHash == "" {
		t.Error("PublicHash should not be empty")
	}

	if proof.CircuitType != "verification_circuit" {
		t.Errorf("CircuitType should be verification_circuit, got %s", proof.CircuitType)
	}

	if proof.IsVerified {
		t.Error("Newly generated proof should not be verified")
	}
}

func TestBlockchainV2Service_VerifyZKProof(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	witness := &ZKWitness{
		PublicInputs: map[string]interface{}{
			"user_id": "user456",
		},
		PrivateInputs: map[string]interface{}{
			"password": "pass123",
		},
		Statement: "identity_verification",
	}

	generatedProof, err := service.GenerateZKProof(ctx, witness)
	if err != nil {
		t.Fatalf("GenerateZKProof failed: %v", err)
	}

	valid, err := service.VerifyZKProof(ctx, generatedProof)
	if err != nil {
		t.Fatalf("VerifyZKProof failed: %v", err)
	}

	if !valid {
		t.Error("Proof should be valid")
	}

	invalidProof := &ZKProof{
		ProofID:    generatedProof.ProofID,
		PublicHash: "invalid_hash",
		ProofData:  "invalid_data",
	}

	valid, err = service.VerifyZKProof(ctx, invalidProof)
	if err != nil {
		t.Fatalf("VerifyZKProof should not return error for invalid proof, got %v", err)
	}

	if valid {
		t.Error("Invalid proof should not be valid")
	}

	_, err = service.VerifyZKProof(ctx, &ZKProof{ProofID: "nonexistent"})
	if err != ErrProofNotFoundV2 {
		t.Errorf("Should return ErrProofNotFoundV2, got %v", err)
	}
}

func TestBlockchainV2Service_GenerateZKProofBatch(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	witnesses := []*ZKWitness{
		{
			PublicInputs:  map[string]interface{}{"user_id": "user1"},
			PrivateInputs: map[string]interface{}{"secret": "secret1"},
			Statement:     "test1",
		},
		{
			PublicInputs:  map[string]interface{}{"user_id": "user2"},
			PrivateInputs: map[string]interface{}{"secret": "secret2"},
			Statement:     "test2",
		},
		{
			PublicInputs:  map[string]interface{}{"user_id": "user3"},
			PrivateInputs: map[string]interface{}{"secret": "secret3"},
			Statement:     "test3",
		},
	}

	proofs, err := service.GenerateZKProofBatch(ctx, witnesses)
	if err != nil {
		t.Fatalf("GenerateZKProofBatch failed: %v", err)
	}

	if len(proofs) != 3 {
		t.Errorf("Should generate 3 proofs, got %d", len(proofs))
	}

	for i, proof := range proofs {
		if proof == nil {
			t.Errorf("Proof at index %d should not be nil", i)
		}
	}
}

func TestBlockchainV2Service_CreateAuditTrail(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	event := &AuditEvent{
		TrailID:   "test_trail",
		EventType: "verification",
		ActorID:   "user123",
		Action:    "verify_identity",
		Resource:  "session_456",
		Details:   map[string]interface{}{"method": "biometric"},
	}

	err := service.CreateAuditTrail(ctx, event)
	if err != nil {
		t.Fatalf("CreateAuditTrail failed: %v", err)
	}

	if event.EventID == "" {
		t.Error("EventID should be set")
	}

	if event.Hash == "" {
		t.Error("Hash should be computed")
	}

	if event.PreviousHash != "" {
		t.Error("First event should have empty PreviousHash")
	}

	event2 := &AuditEvent{
		TrailID:   "test_trail",
		EventType: "verification",
		ActorID:   "user456",
		Action:    "verify_session",
		Resource:  "session_789",
	}

	err = service.CreateAuditTrail(ctx, event2)
	if err != nil {
		t.Fatalf("CreateAuditTrail failed for second event: %v", err)
	}

	if event2.PreviousHash != event.Hash {
		t.Errorf("Second event PreviousHash should link to first event Hash")
	}
}

func TestBlockchainV2Service_QueryAuditTrail(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	trailID := "query_test_trail"

	events := []*AuditEvent{
		{TrailID: trailID, EventType: "login", ActorID: "user1", Action: "login", Resource: "session1"},
		{TrailID: trailID, EventType: "verify", ActorID: "user1", Action: "verify", Resource: "session1"},
		{TrailID: trailID, EventType: "login", ActorID: "user2", Action: "login", Resource: "session2"},
		{TrailID: trailID, EventType: "logout", ActorID: "user1", Action: "logout", Resource: "session1"},
	}

	for _, event := range events {
		err := service.CreateAuditTrail(ctx, event)
		if err != nil {
			t.Fatalf("CreateAuditTrail failed: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	query := &AuditQuery{
		TrailID: trailID,
		Limit:   10,
		Offset:  0,
	}

	results, err := service.QueryAuditTrail(ctx, query)
	if err != nil {
		t.Fatalf("QueryAuditTrail failed: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Should return 4 events, got %d", len(results))
	}

	actorQuery := &AuditQuery{
		TrailID: trailID,
		ActorID: "user1",
		Limit:   10,
	}

	results, err = service.QueryAuditTrail(ctx, actorQuery)
	if err != nil {
		t.Fatalf("QueryAuditTrail failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Should return 3 events for user1, got %d", len(results))
	}

	typeQuery := &AuditQuery{
		TrailID:   trailID,
		EventType: "login",
		Limit:     10,
	}

	results, err = service.QueryAuditTrail(ctx, typeQuery)
	if err != nil {
		t.Fatalf("QueryAuditTrail failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Should return 2 login events, got %d", len(results))
	}

	paginationQuery := &AuditQuery{
		TrailID: trailID,
		Limit:   2,
		Offset:  1,
	}

	results, err = service.QueryAuditTrail(ctx, paginationQuery)
	if err != nil {
		t.Fatalf("QueryAuditTrail failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Should return 2 events with pagination, got %d", len(results))
	}
}

func TestBlockchainV2Service_VerifyAuditIntegrity(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	trailID := "integrity_test_trail"

	events := []*AuditEvent{
		{TrailID: trailID, EventType: "create", ActorID: "user1", Action: "create"},
		{TrailID: trailID, EventType: "update", ActorID: "user1", Action: "update"},
		{TrailID: trailID, EventType: "delete", ActorID: "user2", Action: "delete"},
	}

	for _, event := range events {
		err := service.CreateAuditTrail(ctx, event)
		if err != nil {
			t.Fatalf("CreateAuditTrail failed: %v", err)
		}
	}

	result, err := service.VerifyAuditIntegrity(ctx, trailID)
	if err != nil {
		t.Fatalf("VerifyAuditIntegrity failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.IsValid {
		t.Error("Audit trail should be valid")
	}

	if result.TotalEvents != 3 {
		t.Errorf("TotalEvents should be 3, got %d", result.TotalEvents)
	}

	if result.VerifiedHashes != 3 {
		t.Errorf("VerifiedHashes should be 3, got %d", result.VerifiedHashes)
	}

	if result.FailedHashes != 0 {
		t.Errorf("FailedHashes should be 0, got %d", result.FailedHashes)
	}

	_, err = service.VerifyAuditIntegrity(ctx, "nonexistent_trail")
	if err != ErrAuditTrailNotFound {
		t.Errorf("Should return ErrAuditTrailNotFound, got %v", err)
	}
}

func TestBlockchainV2Service_TrackTransaction(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	txHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	status, err := service.TrackTransaction(ctx, txHash)
	if err != nil {
		t.Fatalf("TrackTransaction failed: %v", err)
	}

	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.TxHash != txHash {
		t.Errorf("TxHash mismatch: got %s, want %s", status.TxHash, txHash)
	}

	if status.ChainID != "ethereum" {
		t.Errorf("ChainID should be ethereum, got %s", status.ChainID)
	}

	if status.Status != "confirmed" {
		t.Errorf("Status should be confirmed, got %s", status.Status)
	}

	if status.BlockNumber == 0 {
		t.Error("BlockNumber should not be zero")
	}

	if status.GasUsed == 0 {
		t.Error("GasUsed should not be zero")
	}
}

func TestBlockchainV2Service_GetAuditMetrics(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	trailID := "metrics_test_trail"

	events := []*AuditEvent{
		{TrailID: trailID, EventType: "login", ActorID: "user1", Action: "login", ChainID: "ethereum"},
		{TrailID: trailID, EventType: "verify", ActorID: "user1", Action: "verify", ChainID: "ethereum"},
		{TrailID: trailID, EventType: "transfer", ActorID: "user2", Action: "transfer", ChainID: "polkadot"},
	}

	for _, event := range events {
		err := service.CreateAuditTrail(ctx, event)
		if err != nil {
			t.Fatalf("CreateAuditTrail failed: %v", err)
		}
	}

	metrics, err := service.GetAuditMetrics(ctx, time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("GetAuditMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.TotalEvents < 3 {
		t.Errorf("TotalEvents should be at least 3, got %d", metrics.TotalEvents)
	}

	if metrics.EventsByType == nil {
		t.Error("EventsByType should not be nil")
	}

	if metrics.EventsByChain == nil {
		t.Error("EventsByChain should not be nil")
	}
}

func TestBlockchainV2Service_NilRecord(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	_, err := service.RecordVerificationV2(ctx, nil)
	if err == nil {
		t.Error("Should return error for nil record")
	}
}

func TestBlockchainV2Service_CrossChainWithoutBridge(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	transfer := &CrossChainTransfer{
		SourceChain: "ethereum",
		TargetChain: "polkadot",
		AssetType:   "ETH",
		Amount:      "1.0",
		Sender:      "0xsender",
		Recipient:   "recipient",
	}

	result, err := service.TransferCrossChain(ctx, transfer)
	if err != nil {
		t.Fatalf("TransferCrossChain should not return error: %v", err)
	}

	if result.Success {
		t.Error("Transfer should fail without configured bridge")
	}
}

func TestBlockchainV2Service_InvalidZKProof(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	_, err := service.VerifyZKProof(ctx, nil)
	if err != ErrInvalidZKProof {
		t.Errorf("Should return ErrInvalidZKProof for nil proof, got %v", err)
	}
}

func TestBlockchainV2Service_InvalidWitness(t *testing.T) {
	service := NewBlockchainV2Service()
	ctx := context.Background()

	_, err := service.GenerateZKProof(ctx, nil)
	if err != ErrInvalidWitness {
		t.Errorf("Should return ErrInvalidWitness for nil witness, got %v", err)
	}
}
