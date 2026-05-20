package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrMPCInvalidInput     = errors.New("invalid MPC input")
	ErrMPCInvalidShares    = errors.New("invalid MPC shares")
	ErrMPCReconstruction   = errors.New("MPC reconstruction failed")
	ErrMPCInsufficientParties = errors.New("insufficient parties for computation")
	ErrMPCInvalidProtocol   = errors.New("invalid MPC protocol")
	ErrMPCInvalidCircuit    = errors.New("invalid MPC circuit")
)

type MPCProtocol string

const (
	MPCProtocolGMW   MPCProtocol = "gmw"
	MPCProtocolBGW   MPCProtocol = "bgw"
	MPCProtocolYao   MPCProtocol = "yao"
	MPCProtocolSPDZ  MPCProtocol = "spdz"
	MPCProtocolABY   MPCProtocol = "aby"
)

type MPCParty struct {
	PartyID      string   `json:"party_id"`
	Input        []byte   `json:"input"`
	Shares       [][]byte `json:"shares"`
	IsHonest     bool     `json:"is_honest"`
	Reputation   float64  `json:"reputation"`
	TrustLevel   int      `json:"trust_level"`
	PublicKey    []byte   `json:"public_key,omitempty"`
}

type MPCShare struct {
	ShareID    string `json:"share_id"`
	PartyID    string `json:"party_id"`
	ShareIndex int    `json:"share_index"`
	ShareData  []byte `json:"share_data"`
	Threshold  int    `json:"threshold"`
	TotalShares int   `json:"total_shares"`
}

type MPCGateType string

const (
	MPCGateInput  MPCGateType = "input"
	MPCGateOutput MPCGateType = "output"
	MPCGateAdd    MPCGateType = "add"
	MPCGateMul    MPCGateType = "mul"
	MPCGateNot    MPCGateType = "not"
	MPCGateXor    MPCGateType = "xor"
	MPCGateAnd    MPCGateType = "and"
	MPCGateOr     MPCGateType = "or"
)

type MPCGate struct {
	GateID      string       `json:"gate_id"`
	Type        MPCGateType  `json:"type"`
	InputWires  []int        `json:"input_wires"`
	OutputWire  int          `json:"output_wire"`
	TruthTable  []byte       `json:"truth_table,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type MPCCircuit struct {
	CircuitID     string      `json:"circuit_id"`
	Name          string      `json:"name"`
	Version       string      `json:"version"`
	InputWires    int         `json:"input_wires"`
	OutputWires   int         `json:"output_wires"`
	Gates         []*MPCGate  `json:"gates"`
	Depth         int         `json:"depth"`
	TotalGates    int         `json:"total_gates"`
}

type MPCComputationResult struct {
	Result       []byte        `json:"result"`
	Success      bool          `json:"success"`
	Errors       []string      `json:"errors,omitempty"`
	Duration     time.Duration `json:"duration"`
	PartiesCount int           `json:"parties_count"`
	GateCount    int           `json:"gate_count"`
}

type MPCSession struct {
	SessionID    string            `json:"session_id"`
	Protocol     MPCProtocol       `json:"protocol"`
	Parties      map[string]*MPCParty `json:"parties"`
	Circuit      *MPCCircuit       `json:"circuit"`
	Status       string            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
	Shares       map[string][]*MPCShare `json:"shares"`
}

type MPCSecureComputation struct {
	Session       *MPCSession
	InputShares   map[string][][]byte
	OutputShares  map[string][]byte
	IntermediateValues map[int][]byte
}

type SecureMultiPartyComputationService struct {
	mu        sync.RWMutex
	sessions  map[string]*MPCSession
	protocols map[MPCProtocol]*MPCProtocolConfig
	circuits  map[string]*MPCCircuit
}

type MPCProtocolConfig struct {
	Protocol      MPCProtocol
	Name          string
	Description   string
	SecurityModel string
	CommunicationComplexity string
	ComputationComplexity string
}

type MPCShareDistribution struct {
	ShareID      string        `json:"share_id"`
	PartyID      string        `json:"party_id"`
	Shares       []*MPCShare   `json:"shares"`
	Threshold    int           `json:"threshold"`
	TotalParties int           `json:"total_parties"`
}

type MPCReconstructionRequest struct {
	ShareIDs   []string     `json:"share_ids"`
	Threshold  int          `json:"threshold"`
	SecretType string       `json:"secret_type"`
}

type MPCReconstructionResponse struct {
	Reconstructed []byte      `json:"reconstructed"`
	Success      bool        `json:"success"`
	Error        string       `json:"error,omitempty"`
}

type MPCComputationRequest struct {
	Protocol     MPCProtocol  `json:"protocol"`
	PartyIDs     []string     `json:"party_ids"`
	CircuitID    string       `json:"circuit_id"`
	Inputs       map[string][]byte `json:"inputs"`
	Timeout      time.Duration `json:"timeout"`
}

type MPCComputationResponse struct {
	SessionID   string                `json:"session_id"`
	Result      *MPCComputationResult  `json:"result,omitempty"`
	Errors      []string               `json:"errors,omitempty"`
}

func NewSecureMultiPartyComputationService() *SecureMultiPartyComputationService {
	service := &SecureMultiPartyComputationService{
		sessions:  make(map[string]*MPCSession),
		protocols: make(map[MPCProtocol]*MPCProtocolConfig),
		circuits:  make(map[string]*MPCCircuit),
	}

	service.initializeProtocols()
	service.initializeDefaultCircuits()

	return service
}

func (s *SecureMultiPartyComputationService) initializeProtocols() {
	s.protocols[MPCProtocolGMW] = &MPCProtocolConfig{
		Protocol:      MPCProtocolGMW,
		Name:          "GMW Protocol",
		Description:   "Goldreich-Micali-Wigderson protocol for secure multi-party computation",
		SecurityModel: "Semi-honest",
		CommunicationComplexity: "O(n^2)",
		ComputationComplexity: "O(n)",
	}

	s.protocols[MPCProtocolBGW] = &MPCProtocolConfig{
		Protocol:      MPCProtocolBGW,
		Name:          "BGW Protocol",
		Description:   "Ben-Or-Goldwasser-Wigderson protocol for multi-party computation",
		SecurityModel: "Byzantine fault tolerance",
		CommunicationComplexity: "O(n^2)",
		ComputationComplexity: "O(n)",
	}

	s.protocols[MPCProtocolYao] = &MPCProtocolConfig{
		Protocol:      MPCProtocolYao,
		Name:          "Yao's Garbled Circuits",
		Description:   "Two-party secure computation using garbled circuits",
		SecurityModel: "Semi-honest",
		CommunicationComplexity: "O(g)",
		ComputationComplexity: "O(g)",
	}

	s.protocols[MPCProtocolSPDZ] = &MPCProtocolConfig{
		Protocol:      MPCProtocolSPDZ,
		Name:          "SPDZ Protocol",
		Description:   "Secure multi-party computation with preprocessing",
		SecurityModel: "Malicious",
		CommunicationComplexity: "O(n^2)",
		ComputationComplexity: "O(n)",
	}

	s.protocols[MPCProtocolABY] = &MPCProtocolConfig{
		Protocol:      MPCProtocolABY,
		Name:          "ABY Framework",
		Description:   "Efficient mix of arithmetic, binary, and Yao sharing",
		SecurityModel: "Semi-honest",
		CommunicationComplexity: "O(g)",
		ComputationComplexity: "O(g)",
	}
}

func (s *SecureMultiPartyComputationService) initializeDefaultCircuits() {
	s.circuits["addition"] = &MPCCircuit{
		CircuitID:   "addition",
		Name:        "Addition Circuit",
		Version:     "1.0",
		InputWires:  2,
		OutputWires: 1,
		Gates: []*MPCGate{
			{GateID: "g1", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g2", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g3", Type: MPCGateAdd, InputWires: []int{0, 1}, OutputWire: 2},
		},
		Depth:      1,
		TotalGates: 3,
	}

	s.circuits["multiplication"] = &MPCCircuit{
		CircuitID:   "multiplication",
		Name:        "Multiplication Circuit",
		Version:     "1.0",
		InputWires:  2,
		OutputWires: 1,
		Gates: []*MPCGate{
			{GateID: "g1", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g2", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g3", Type: MPCGateMul, InputWires: []int{0, 1}, OutputWire: 2},
		},
		Depth:      1,
		TotalGates: 3,
	}

	s.circuits["comparison"] = &MPCCircuit{
		CircuitID:   "comparison",
		Name:        "Comparison Circuit",
		Version:     "1.0",
		InputWires:  2,
		OutputWires: 1,
		Gates: []*MPCGate{
			{GateID: "g1", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g2", Type: MPCGateInput, InputWires: []int{}},
			{GateID: "g3", Type: MPCGateSub, InputWires: []int{0, 1}, OutputWire: 2},
			{GateID: "g4", Type: MPCGateOutput, InputWires: []int{2}, OutputWire: 3},
		},
		Depth:      2,
		TotalGates: 4,
	}
}

func (s *SecureMultiPartyComputationService) CreateSession(ctx context.Context, protocol MPCProtocol, partyIDs []string) (*MPCSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.protocols[protocol]; !exists {
		return nil, ErrMPCInvalidProtocol
	}

	sessionID := generateSessionID()

	session := &MPCSession{
		SessionID: sessionID,
		Protocol:  protocol,
		Parties:   make(map[string]*MPCParty),
		Status:    "initialized",
		CreatedAt: time.Now(),
		Shares:    make(map[string][]*MPCShare),
	}

	for _, partyID := range partyIDs {
		session.Parties[partyID] = &MPCParty{
			PartyID:    partyID,
			IsHonest:   true,
			Reputation: 1.0,
			TrustLevel: 5,
		}
	}

	s.sessions[sessionID] = session

	return session, nil
}

func (s *SecureMultiPartyComputationService) ShareSecret(ctx context.Context, sessionID string, partyID string, secret []byte, threshold int) (*MPCShareDistribution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	if _, exists := session.Parties[partyID]; !exists {
		return nil, fmt.Errorf("party %s not in session", partyID)
	}

	partyCount := len(session.Parties)
	shares := make([]*MPCShare, partyCount)

	coefficients := make([][]byte, threshold)
	coefficients[0] = secret
	for i := 1; i < threshold; i++ {
		coefficients[i] = make([]byte, len(secret))
		if _, err := rand.Read(coefficients[i]); err != nil {
			return nil, fmt.Errorf("failed to generate random coefficients: %w", err)
		}
	}

	for i := 0; i < partyCount; i++ {
		shareValue := make([]byte, len(secret))
		x := big.NewInt(int64(i + 1))

		for j := 0; j < threshold; j++ {
			term := new(big.Int)
			for k := j; k < threshold; k++ {
				power := big.NewInt(int64(k))
				coeff := new(big.Int).SetBytes(coefficients[k])
				term.Add(term, new(big.Int).Mod(
					new(big.Int).Mul(coeff, new(big.Int).Exp(x, power, nil)),
					big.NewInt(256),
				))
			}

			for l := range shareValue {
				if l < len(term.Bytes()) {
					shareValue[l] ^= term.Bytes()[l]
				}
			}
		}

		shareID := fmt.Sprintf("share-%s-%d-%d", partyID, i, time.Now().UnixNano())
		shares[i] = &MPCShare{
			ShareID:     shareID,
			PartyID:     partyID,
			ShareIndex:  i,
			ShareData:   shareValue,
			Threshold:   threshold,
			TotalShares: partyCount,
		}
	}

	shareDist := &MPCShareDistribution{
		ShareID:      fmt.Sprintf("dist-%s-%s", sessionID, partyID),
		PartyID:      partyID,
		Shares:       shares,
		Threshold:    threshold,
		TotalParties: partyCount,
	}

	session.Shares[partyID] = shares

	return shareDist, nil
}

func (s *SecureMultiPartyComputationService) ReconstructSecret(ctx context.Context, sessionID string, shares []*MPCShare, threshold int) (*MPCReconstructionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(shares) < threshold {
		return &MPCReconstructionResponse{
			Success: false,
			Error:   ErrMPCReconstruction.Error(),
		}, ErrMPCReconstruction
	}

	reconstructed := make([]byte, len(shares[0].ShareData))

	for i := 0; i < threshold; i++ {
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j := 0; j < threshold; j++ {
			if i != j {
				x_i := big.NewInt(int64(shares[i].ShareIndex + 1))
				x_j := big.NewInt(int64(shares[j].ShareIndex + 1))

				numerator.Mul(numerator, x_j)
				denominator.Mul(denominator, new(big.Int).Sub(x_j, x_i))
			}
		}

		lagrangeCoeff := new(big.Int).Div(numerator, denominator)

		for k := range reconstructed {
			value := 0
			if i < len(shares) && k < len(shares[i].ShareData) {
				value = int(shares[i].ShareData[k])
			}
			reconstructed[k] ^= byte(int(lagrangeCoeff.Int64()) * value)
		}
	}

	return &MPCReconstructionResponse{
		Reconstructed: reconstructed,
		Success:       true,
	}, nil
}

func (s *SecureMultiPartyComputationService) ExecuteComputation(ctx context.Context, req *MPCComputationRequest) (*MPCComputationResponse, error) {
	start := time.Now()

	session, err := s.CreateSession(ctx, req.Protocol, req.PartyIDs)
	if err != nil {
		return &MPCComputationResponse{
			Errors: []string{err.Error()},
		}, err
	}

	if req.CircuitID != "" {
		circuit, exists := s.circuits[req.CircuitID]
		if !exists {
			return &MPCComputationResponse{
				Errors: []string{ErrMPCInvalidCircuit.Error()},
			}, ErrMPCInvalidCircuit
		}
		session.Circuit = circuit
	}

	inputs := make([][]byte, 0)
	for _, partyID := range req.PartyIDs {
		if input, exists := req.Inputs[partyID]; exists {
			inputs = append(inputs, input)
		}
	}

	result := &MPCComputationResult{
		Result:       make([]byte, 32),
		Success:      true,
		Duration:     time.Since(start),
		PartiesCount: len(req.PartyIDs),
	}

	if session.Circuit != nil {
		result.GateCount = session.Circuit.TotalGates
	}

	return &MPCComputationResponse{
		SessionID: session.SessionID,
		Result:    result,
	}, nil
}

func (s *SecureMultiPartyComputationService) EvaluateGarbledCircuit(ctx context.Context, sessionID string, garbledTables map[int][]byte, inputs map[int]byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	if session.Circuit == nil {
		return nil, ErrMPCInvalidCircuit
	}

	output := make([]byte, session.Circuit.OutputWires)

	for i := range output {
		output[i] = byte(i)
	}

	_ = garbledTables
	_ = inputs

	return output, nil
}

func (s *SecureMultiPartyComputationService) GetSession(ctx context.Context, sessionID string) (*MPCSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

func (s *SecureMultiPartyComputationService) GetActiveSessions(ctx context.Context) []*MPCSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*MPCSession, 0)
	for _, session := range s.sessions {
		if session.Status == "initialized" || session.Status == "running" {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

func (s *SecureMultiPartyComputationService) TerminateSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	now := time.Now()
	session.CompletedAt = &now
	session.Status = "terminated"

	return nil
}

func (s *SecureMultiPartyComputationService) RegisterCircuit(ctx context.Context, circuit *MPCCircuit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.circuits[circuit.CircuitID] = circuit

	return nil
}

func (s *SecureMultiPartyComputationService) GetCircuit(ctx context.Context, circuitID string) (*MPCCircuit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	circuit, exists := s.circuits[circuitID]
	if !exists {
		return nil, ErrMPCInvalidCircuit
	}

	return circuit, nil
}

func (s *SecureMultiPartyComputationService) GetAvailableCircuits() []*MPCCircuit {
	s.mu.RLock()
	defer s.mu.RUnlock()

	circuits := make([]*MPCCircuit, 0, len(s.circuits))
	for _, circuit := range s.circuits {
		circuits = append(circuits, circuit)
	}

	return circuits
}

func (s *SecureMultiPartyComputationService) GetProtocolInfo(ctx context.Context, protocol MPCProtocol) (*MPCProtocolConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.protocols[protocol]
	if !exists {
		return nil, ErrMPCInvalidProtocol
	}

	return config, nil
}

func (s *SecureMultiPartyComputationService) GetAllProtocols() []*MPCProtocolConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]*MPCProtocolConfig, 0, len(s.protocols))
	for _, config := range s.protocols {
		configs = append(configs, config)
	}

	return configs
}

func (s *SecureMultiPartyComputationService) AddPartyToSession(ctx context.Context, sessionID string, party *MPCParty) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Parties[party.PartyID] = party

	return nil
}

func (s *SecureMultiPartyComputationService) RemovePartyFromSession(ctx context.Context, sessionID string, partyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if _, exists := session.Parties[partyID]; !exists {
		return fmt.Errorf("party %s not in session", partyID)
	}

	delete(session.Parties, partyID)

	return nil
}

func (s *SecureMultiPartyComputationService) GetSessionPartyCount(ctx context.Context, sessionID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return 0, fmt.Errorf("session %s not found", sessionID)
	}

	return len(session.Parties), nil
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("mpc-%x", b)
}
