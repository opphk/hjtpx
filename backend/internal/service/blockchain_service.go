package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BlockchainService interface {
	RecordVerification(ctx context.Context, record *VerificationRecord) (*BlockchainProof, error)
	VerifyProof(ctx context.Context, proofID string) (*BlockchainProof, error)
	GetVerificationHistory(ctx context.Context, appID string, limit, offset int) ([]*VerificationRecord, error)
	GetCrossChainIdentity(ctx context.Context, identity string) (*CrossChainIdentity, error)
	RegisterCrossChainIdentity(ctx context.Context, identity *CrossChainIdentity) error
	VerifyCrossChain(ctx context.Context, request *CrossChainVerifyRequest) (*CrossChainVerifyResponse, error)
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, appID string, start, end time.Time) ([]*AuditLog, error)
	VerifyAuditLog(ctx context.Context, logID string) (bool, error)
	ExportAuditTrail(ctx context.Context, appID string) ([]byte, error)
}

type VerificationRecord struct {
	RecordID      string    `json:"record_id"`
	AppID         string    `json:"app_id"`
	SessionID     string    `json:"session_id"`
	EventType     string    `json:"event_type"`
	EventData     string    `json:"event_data"`
	Hash          string    `json:"hash"`
	Timestamp     time.Time `json:"timestamp"`
	RiskLevel     string    `json:"risk_level"`
	RiskScore     float64   `json:"risk_score"`
	UserAgent     string    `json:"user_agent"`
	IPAddress     string    `json:"ip_address"`
	Country       string    `json:"country"`
	DeviceFinger  string    `json:"device_fingerprint"`
	ChainTxHash   string    `json:"chain_tx_hash"`
	BlockNumber   uint64    `json:"block_number"`
	Confirmations int       `json:"confirmations"`
}

type BlockchainProof struct {
	ProofID      string    `json:"proof_id"`
	RecordID     string    `json:"record_id"`
	ChainID      string    `json:"chain_id"`
	TxHash       string    `json:"tx_hash"`
	BlockHash    string    `json:"block_hash"`
	BlockNumber  uint64    `json:"block_number"`
	Timestamp    time.Time `json:"timestamp"`
	PreviousHash string    `json:"previous_hash"`
	MerkleRoot   string    `json:"merkle_root"`
	Signature    string    `json:"signature"`
	Status       string    `json:"status"`
	Confirmations int      `json:"confirmations"`
}

type CrossChainIdentity struct {
	IdentityID   string    `json:"identity_id"`
	Identity     string    `json:"identity"`
	ChainType    string    `json:"chain_type"`
	PublicKey    string    `json:"public_key"`
	TrustScore   float64   `json:"trust_score"`
	LinkedIDs    []string  `json:"linked_ids"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Status       string    `json:"status"`
	Verified     bool      `json:"verified"`
}

type CrossChainVerifyRequest struct {
	SourceChain      string `json:"source_chain"`
	TargetChain      string `json:"target_chain"`
	Identity         string `json:"identity"`
	VerificationType string `json:"verification_type"`
	Proof            string `json:"proof"`
}

type CrossChainVerifyResponse struct {
	Valid      bool      `json:"valid"`
	Identity   string    `json:"identity"`
	TrustScore float64   `json:"trust_score"`
	VerifiedAt time.Time `json:"verified_at"`
	Message    string    `json:"message"`
}

type AuditLog struct {
	LogID      string    `json:"log_id"`
	AppID      string    `json:"app_id"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
	Hash       string    `json:"hash"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Result     string    `json:"result"`
}

type blockchainService struct {
	records    map[string]*VerificationRecord
	proofs     map[string]*BlockchainProof
	identities map[string]*CrossChainIdentity
	auditLogs  map[string]*AuditLog
	merkleTree *MerkleTree
	chainState map[string]*ChainState
}

type ChainState struct {
	ChainID      string    `json:"chain_id"`
	LatestBlock  uint64    `json:"latest_block"`
	LatestHash   string    `json:"latest_hash"`
	TotalRecords uint64    `json:"total_records"`
	LastUpdated  time.Time `json:"last_updated"`
}

type MerkleTree struct {
	Root      string    `json:"root"`
	Leaves    []string  `json:"leaves"`
	Tree      []string  `json:"tree"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	ErrRecordNotFound    = errors.New("verification record not found")
	ErrProofNotFound     = errors.New("proof not found")
	ErrInvalidHash       = errors.New("invalid hash")
	ErrChainNotSupported = errors.New("chain not supported")
	ErrIdentityExists    = errors.New("identity already exists")
	ErrInvalidProof      = errors.New("invalid proof")
)

func NewBlockchainService() BlockchainService {
	return &blockchainService{
		records:    make(map[string]*VerificationRecord),
		proofs:     make(map[string]*BlockchainProof),
		identities: make(map[string]*CrossChainIdentity),
		auditLogs:  make(map[string]*AuditLog),
		merkleTree: newMerkleTree(),
		chainState: initChainStates(),
	}
}

func initChainStates() map[string]*ChainState {
	return map[string]*ChainState{
		"ethereum": {
			ChainID:      "ethereum",
			LatestBlock:  1,
			TotalRecords: 0,
			LastUpdated:  time.Now(),
		},
		"polygon": {
			ChainID:      "polygon",
			LatestBlock:  1,
			TotalRecords: 0,
			LastUpdated:  time.Now(),
		},
		"bsc": {
			ChainID:      "bsc",
			LatestBlock:  1,
			TotalRecords: 0,
			LastUpdated:  time.Now(),
		},
	}
}

func (s *blockchainService) RecordVerification(ctx context.Context, record *VerificationRecord) (*BlockchainProof, error) {
	if record == nil {
		return nil, errors.New("record cannot be nil")
	}

	if record.RecordID == "" {
		record.RecordID = uuid.New().String()
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	record.Hash = s.computeRecordHash(record)

	s.records[record.RecordID] = record

	s.merkleTree.addLeaf(record.Hash)

	chainState := s.chainState["ethereum"]
	blockNum := chainState.LatestBlock + 1
	blockHash := s.computeBlockHash(blockNum, chainState.LatestHash, record.Hash)
	chainState.LatestBlock = blockNum
	chainState.LatestHash = blockHash
	chainState.TotalRecords++
	chainState.LastUpdated = time.Now()

	proof := &BlockchainProof{
		ProofID:       uuid.New().String(),
		RecordID:      record.RecordID,
		ChainID:       "ethereum",
		TxHash:        s.computeTxHash(record),
		BlockHash:     blockHash,
		BlockNumber:   blockNum,
		Timestamp:     time.Now(),
		PreviousHash:  chainState.LatestHash,
		MerkleRoot:    s.merkleTree.Root,
		Signature:     s.signProof(record),
		Status:        "confirmed",
		Confirmations: 1,
	}

	s.proofs[proof.ProofID] = proof
	record.ChainTxHash = proof.TxHash
	record.BlockNumber = proof.BlockNumber

	return proof, nil
}

func (s *blockchainService) computeRecordHash(record *VerificationRecord) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d:%f:%s:%s:%s:%s",
		record.RecordID,
		record.AppID,
		record.SessionID,
		record.EventType,
		record.Timestamp.Unix(),
		record.RiskScore,
		record.UserAgent,
		record.IPAddress,
		record.DeviceFinger,
		record.EventData,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *blockchainService) computeBlockHash(blockNum uint64, prevHash, recordHash string) string {
	data := fmt.Sprintf("%d:%s:%s:%d", blockNum, prevHash, recordHash, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *blockchainService) computeTxHash(record *VerificationRecord) string {
	data := fmt.Sprintf("%s:%s:%s:%d", record.RecordID, record.Hash, record.Timestamp.Format(time.RFC3339), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

func (s *blockchainService) signProof(record *VerificationRecord) string {
	data := fmt.Sprintf("%s:%s:%s", record.RecordID, record.Hash, time.Now().Format(time.RFC3339))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *blockchainService) VerifyProof(ctx context.Context, proofID string) (*BlockchainProof, error) {
	proof, exists := s.proofs[proofID]
	if !exists {
		return nil, ErrProofNotFound
	}

	record, exists := s.records[proof.RecordID]
	if !exists {
		return nil, ErrRecordNotFound
	}

	expectedHash := s.computeRecordHash(record)
	if expectedHash != record.Hash {
		proof.Status = "invalid"
		return proof, nil
	}

	proof.Status = "verified"
	return proof, nil
}

func (s *blockchainService) GetVerificationHistory(ctx context.Context, appID string, limit, offset int) ([]*VerificationRecord, error) {
	var result []*VerificationRecord

	for _, record := range s.records {
		if record.AppID == appID {
			result = append(result, record)
		}
	}

	if offset >= len(result) {
		return []*VerificationRecord{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (s *blockchainService) GetCrossChainIdentity(ctx context.Context, identity string) (*CrossChainIdentity, error) {
	id, exists := s.identities[identity]
	if !exists {
		return nil, ErrProofNotFound
	}
	return id, nil
}

func (s *blockchainService) RegisterCrossChainIdentity(ctx context.Context, identity *CrossChainIdentity) error {
	if identity == nil {
		return errors.New("identity cannot be nil")
	}

	if identity.IdentityID == "" {
		identity.IdentityID = uuid.New().String()
	}
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Now()
	}
	identity.UpdatedAt = time.Now()

	if _, exists := s.identities[identity.Identity]; exists {
		return ErrIdentityExists
	}

	identity.TrustScore = s.calculateTrustScore(identity)

	s.identities[identity.Identity] = identity

	return nil
}

func (s *blockchainService) calculateTrustScore(identity *CrossChainIdentity) float64 {
	score := 50.0

	if identity.PublicKey != "" {
		score += 20.0
	}

	if len(identity.LinkedIDs) > 0 {
		score += float64(len(identity.LinkedIDs) * 5)
		if score > 90 {
			score = 90
		}
	}

	age := time.Since(identity.CreatedAt)
	if age > 365*24*time.Hour {
		score += 15.0
	} else if age > 180*24*time.Hour {
		score += 10.0
	} else if age > 30*24*time.Hour {
		score += 5.0
	}

	return score
}

func (s *blockchainService) VerifyCrossChain(ctx context.Context, request *CrossChainVerifyRequest) (*CrossChainVerifyResponse, error) {
	if request.SourceChain == "" || request.TargetChain == "" {
		return nil, errors.New("source and target chain required")
	}

	identity, exists := s.identities[request.Identity]
	if !exists {
		return &CrossChainVerifyResponse{
			Valid:      false,
			Identity:   request.Identity,
			TrustScore: 0,
			VerifiedAt: time.Now(),
			Message:    "identity not found",
		}, nil
	}

	if identity.ChainType != request.SourceChain {
		return &CrossChainVerifyResponse{
			Valid:      false,
			Identity:   request.Identity,
			TrustScore: identity.TrustScore,
			VerifiedAt: time.Now(),
			Message:    "chain type mismatch",
		}, nil
	}

	valid := identity.Verified && identity.TrustScore >= 50.0

	message := "verification successful"
	if !valid {
		message = "verification failed: insufficient trust score or unverified identity"
	}

	return &CrossChainVerifyResponse{
		Valid:      valid,
		Identity:   request.Identity,
		TrustScore: identity.TrustScore,
		VerifiedAt: time.Now(),
		Message:    message,
	}, nil
}

func (s *blockchainService) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	if s.auditLogs == nil {
		s.auditLogs = make(map[string]*AuditLog)
	}

	if log.LogID == "" {
		log.LogID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	log.Hash = s.computeAuditLogHash(log)
	s.auditLogs[log.LogID] = log

	return nil
}

func (s *blockchainService) GetAuditLogs(ctx context.Context, appID string, start, end time.Time) ([]*AuditLog, error) {
	var result []*AuditLog

	for _, log := range s.auditLogs {
		if log.AppID == appID && !log.Timestamp.Before(start) && !log.Timestamp.After(end) {
			result = append(result, log)
		}
	}

	return result, nil
}

func (s *blockchainService) VerifyAuditLog(ctx context.Context, logID string) (bool, error) {
	log, exists := s.auditLogs[logID]
	if !exists {
		return false, ErrRecordNotFound
	}

	expectedHash := s.computeAuditLogHash(log)
	return log.Hash == expectedHash, nil
}

func (s *blockchainService) computeAuditLogHash(log *AuditLog) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%s:%s:%s",
		log.LogID,
		log.AppID,
		log.UserID,
		log.Timestamp.Unix(),
		log.Action,
		log.Resource,
		log.Details,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func newMerkleTree() *MerkleTree {
	return &MerkleTree{
		Leaves:    []string{},
		Tree:      []string{},
		Root:      "",
		Timestamp: time.Now(),
	}
}

func (m *MerkleTree) addLeaf(hash string) {
	m.Leaves = append(m.Leaves, hash)
	m.rebuildTree()
}

func (m *MerkleTree) rebuildTree() {
	if len(m.Leaves) == 0 {
		m.Root = ""
		m.Tree = []string{}
		return
	}

	currentLevel := make([]string, len(m.Leaves))
	copy(currentLevel, m.Leaves)
	m.Tree = append(m.Tree, currentLevel...)

	for len(currentLevel) > 1 {
		nextLevel := []string{}
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined := currentLevel[i] + currentLevel[i+1]
				hash := sha256.Sum256([]byte(combined))
				nextLevel = append(nextLevel, hex.EncodeToString(hash[:]))
			} else {
				combined := currentLevel[i] + currentLevel[i]
				hash := sha256.Sum256([]byte(combined))
				nextLevel = append(nextLevel, hex.EncodeToString(hash[:]))
			}
		}
		m.Tree = append(m.Tree, nextLevel...)
		currentLevel = nextLevel
	}

	if len(currentLevel) > 0 {
		m.Root = currentLevel[0]
	}
}

func (s *blockchainService) ExportAuditTrail(ctx context.Context, appID string) ([]byte, error) {
	logs, err := s.GetAuditLogs(ctx, appID, time.Time{}, time.Now())
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return nil, err
	}

	return data, nil
}
