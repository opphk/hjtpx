package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrHEInvalidParams     = errors.New("invalid homomorphic encryption parameters")
	ErrHEEncryptionFailed = errors.New("homomorphic encryption failed")
	ErrHEDecryptionFailed = errors.New("homomorphic encryption decryption failed")
	ErrHEOperationFailed   = errors.New("homomorphic operation failed")
	ErrMPCInvalidInput    = errors.New("invalid MPC input")
	ErrMPCInvalidShares   = errors.New("invalid MPC shares")
	ErrMPCReconstruction  = errors.New("MPC reconstruction failed")
	ErrFLInvalidModel     = errors.New("invalid federated learning model")
	ErrFLInvalidUpdate    = errors.New("invalid federated learning update")
	ErrPDSInvalidData     = errors.New("invalid private data sharing data")
)

type HEType string

const (
	HETypePaillier    HEType = "paillier"
	HETypeBGV         HEType = "bgv"
	HETypeCKKS        HEType = "ckks"
	HETypeBFV         HEType = "bfv"
)

type HEParams struct {
	BitSize    int
	Lambda     int
	PlaintextModulus int
	CiphertextModulus int
	NoiseBound int
	Depth      int
}

type HEPublicKey struct {
	N       *big.Int
	G       *big.Int
	NSquare *big.Int
}

type HEPrivateKey struct {
	Lambda *big.Int
	P      *big.Int
	Q      *big.Int
}

type HECiphertext struct {
	C1      *big.Int
	C2      *big.Int
	Type    HEType
	Depth   int
}

type HEPlaintext struct {
	M       *big.Int
	PlaintextModulus int
}

type HESecretKey struct {
	SK []byte
	PK []byte
}

type HEEncryptionResult struct {
	Ciphertext *HECiphertext
	PublicKey  *HEPublicKey
	Noise      float64
}

type HEOperationResult struct {
	Result     *HECiphertext
	Operation  string
	Duration   time.Duration
}

type MPCProtocol string

const (
	MPCProtocolGMW  MPCProtocol = "gmw"
	MPCProtocolBGW  MPCProtocol = "bgw"
	MPCProtocolYao   MPCProtocol = "yao"
)

type MPCParty struct {
	PartyID      string
	Input        []byte
	Shares       []byte
	IsHonest     bool
	Contribution float64
}

type MPCShare struct {
	ShareID   string
	PartyID   string
	ShareData []byte
	Index     int
}

type MPCComputation struct {
	Protocol     MPCProtocol
	Circuit      *MPCCircuit
	Parties      []*MPCParty
	Result       interface{}
	Duration     time.Duration
}

type MPCCircuit struct {
	InputGates  []*MPCGate
	OutputGates []*MPCGate
	ANDGates    []*MPCGate
	NOTGates    []*MPCGate
	Depth       int
}

type MPCGate struct {
	GateID     string
	Type       string
	InputWires []int
	OutputWire int
	TruthTable []int
}

type MPCShareResult struct {
	Shares     []*MPCShare
	Threshold  int
	ReconstructRequired int
}

type FLNode struct {
	NodeID       string
	DatasetSize  int
	ModelVersion string
	Status       string
	Reputation   float64
	LastUpdate   time.Time
}

type FLModel struct {
	ModelID       string
	Weights       []float64
	Gradient      []float64
	Version       string
	Accuracy      float64
	Participants  int
	AggregatedAt  time.Time
}

type FLAggregationRequest struct {
	NodeID     string
	ModelID    string
	Weights    []float64
	Gradient   []float64
	DatasetSize int
}

type FLAggregationResult struct {
	Success      bool
	AggregatedWeights []float64
	NewVersion   string
	Accuracy     float64
	Participants int
}

type FLPrivacyBudget struct {
	Epsilon      float64
	Delta        float64
	NoiseScale   float64
	MaxGradient  float64
}

type FLSecureAggregation struct {
	EncryptedShares map[string]*HECiphertext
	CombinedResult  *HECiphertext
	VerifyProof     []byte
}

type PrivacyDataShare struct {
	DataID       string
	DataType     string
	EncryptedData *HECiphertext
	AccessPolicy string
	OwnerID      string
	CreatedAt    time.Time
}

type DataAccessRequest struct {
	DataID       string
	RequesterID  string
	Purpose      string
	AccessLevel  string
	ZKProof      []byte
}

type DataAccessResponse struct {
	Success     bool
	Data        interface{}
	ErrorMessage string
}

type PrivacyComputingService struct {
	mu              sync.RWMutex
	heEngine        *HEEngine
	mpcEngine       *MPCEngine
	flEngine        *FLEngine
	pdsEngine       *PDSEngine
	privacyBudget   map[string]*FLPrivacyBudget
}

type HEEngine struct {
	mu        sync.RWMutex
	params    *HEParams
	heType    HEType
	publicKey *HEPublicKey
	privateKey *HEPrivateKey
}

type MPCEngine struct {
	mu        sync.RWMutex
	protocol  MPCProtocol
	parties   map[string]*MPCParty
	circuit   *MPCCircuit
}

type FLEngine struct {
	mu          sync.RWMutex
	nodes       map[string]*FLNode
	models      map[string]*FLModel
	globalModel *FLModel
	privacyBudget *FLPrivacyBudget
	secureAgg   *FLSecureAggregation
}

type PDSEngine struct {
	mu       sync.RWMutex
	data     map[string]*PrivacyDataShare
	policies map[string]*DataAccessPolicy
}

type DataAccessPolicy struct {
	PolicyID     string
	DataID       string
	RequiredZKProof bool
	MinTrustLevel   int
	AllowedPurposes []string
}

func NewPrivacyComputingService() *PrivacyComputingService {
	return &PrivacyComputingService{
		heEngine:      NewHEEngine(),
		mpcEngine:     NewMPCEngine(),
		flEngine:      NewFLEngine(),
		pdsEngine:     NewPDSEngine(),
		privacyBudget: make(map[string]*FLPrivacyBudget),
	}
}

func NewHEEngine() *HEEngine {
	return &HEEngine{
		params: &HEParams{
			BitSize:           2048,
			Lambda:            256,
			PlaintextModulus:  65537,
			CiphertextModulus: 0,
			NoiseBound:        100,
			Depth:             10,
		},
		heType: HETypePaillier,
	}
}

func (he *HEEngine) GenerateKeyPair() (*HEPublicKey, *HEPrivateKey, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	p, err := rand.Prime(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}

	q, err := rand.Prime(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}

	n := new(big.Int).Mul(p, q)
	g := new(big.Int).Add(n, big.NewInt(1))
	lambda := new(big.Int).Mul(new(big.Int).Sub(p, big.NewInt(1)), new(big.Int).Sub(q, big.NewInt(1)))

	nSquare := new(big.Int).Mul(n, n)

	return &HEPublicKey{
			N:       n,
			G:       g,
			NSquare: nSquare,
		}, &HEPrivateKey{
			Lambda: lambda,
			P:      p,
			Q:      q,
		}, nil
}

func (he *HEEngine) Encrypt(plaintext *big.Int, publicKey *HEPublicKey) (*HECiphertext, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	n := publicKey.N
	g := publicKey.G
	nSquare := publicKey.NSquare

	r := new(big.Int)
	for r.BitLen() < n.BitLen()/2 {
		r,_ = rand.Int(rand.Reader, n)
	}

	c1 := new(big.Int).Exp(g, plaintext, nSquare)
	rN := new(big.Int).Exp(r, n, nSquare)
	c1 = new(big.Int).Mul(c1, rN)
	c1 = new(big.Int).Mod(c1, nSquare)

	c2 := new(big.Int).Exp(g, big.NewInt(1), nSquare)

	return &HECiphertext{
		C1:    c1,
		C2:    c2,
		Type:  he.heType,
		Depth: 1,
	}, nil
}

func (he *HEEngine) Decrypt(ciphertext *HECiphertext, privateKey *HEPrivateKey) (*big.Int, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	n := privateKey.P
	n.Mul(privateKey.P, privateKey.Q)

	mu := new(big.Int).ModInverse(privateKey.P, n)

	power := new(big.Int).Sub(ciphertext.C1, big.NewInt(1))
	power.Div(power, n)

	m := new(big.Int).Mod(power, n)
	m.Mul(m, mu)
	m.Mod(m, n)

	return m, nil
}

func (he *HEEngine) Add(ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	result := new(big.Int).Mul(ct1.C1, ct2.C1)
	result.Mod(result, ct1.C1)

	return &HECiphertext{
		C1:    result,
		C2:    ct1.C2,
		Type:  he.heType,
		Depth: max(ct1.Depth, ct2.Depth),
	}, nil
}

func (he *HEEngine) Multiply(ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	result := new(big.Int).Exp(ct1.C1, ct2.C1, ct1.C1)

	return &HECiphertext{
		C1:    result,
		C2:    ct1.C2,
		Type:  he.heType,
		Depth: ct1.Depth + ct2.Depth,
	}, nil
}

func (he *HEEngine) ScalarMultiply(ct *HECiphertext, scalar *big.Int) (*HECiphertext, error) {
	he.mu.Lock()
	defer he.mu.Unlock()

	result := new(big.Int).Exp(ct.C1, scalar, ct.C1)

	return &HECiphertext{
		C1:    result,
		C2:    ct.C2,
		Type:  he.heType,
		Depth: ct.Depth,
	}, nil
}

func NewMPCEngine() *MPCEngine {
	return &MPCEngine{
		protocol: MPCProtocolGMW,
		parties:  make(map[string]*MPCParty),
	}
}

func (mpc *MPCEngine) SetupProtocol(protocol MPCProtocol, parties []string) error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	mpc.protocol = protocol

	for _, partyID := range parties {
		mpc.parties[partyID] = &MPCParty{
			PartyID:      partyID,
			Input:        []byte{},
			Shares:       []byte{},
			IsHonest:     true,
			Contribution: 1.0,
		}
	}

	mpc.circuit = &MPCCircuit{
		InputGates:  make([]*MPCGate, 0),
		OutputGates: make([]*MPCGate, 0),
		ANDGates:    make([]*MPCGate, 0),
		NOTGates:    make([]*MPCGate, 0),
		Depth:       0,
	}

	return nil
}

func (mpc *MPCEngine) ShareSecret(partyID string, secret []byte, threshold int) ([]*MPCShare, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	shares := make([]*MPCShare, 0, threshold)

	shareCount := threshold * 2

	for i := 0; i < shareCount; i++ {
		shareData := make([]byte, len(secret))
		_, err := rand.Read(shareData)
		if err != nil {
			return nil, err
		}

		share := &MPCShare{
			ShareID:   fmt.Sprintf("share-%s-%d", partyID, i),
			PartyID:   partyID,
			ShareData: shareData,
			Index:     i,
		}
		shares = append(shares, share)
	}

	return shares, nil
}

func (mpc *MPCEngine) ReconstructSecret(shares []*MPCShare, threshold int) ([]byte, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if len(shares) < threshold {
		return nil, ErrMPCReconstruction
	}

	result := make([]byte, 32)
	for _, share := range shares[:threshold] {
		for i := range result {
			result[i] ^= share.ShareData[i]
		}
	}

	return result, nil
}

func (mpc *MPCEngine) ComputeMPC(parties []string, computationType string) (*MPCComputation, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	result := &MPCComputation{
		Protocol: mpc.protocol,
		Circuit:  mpc.circuit,
		Parties:  make([]*MPCParty, 0),
		Duration: time.Duration(0),
	}

	for _, partyID := range parties {
		if party, exists := mpc.parties[partyID]; exists {
			result.Parties = append(result.Parties, party)
		}
	}

	return result, nil
}

func (mpc *MPCEngine) EvaluateGarbledCircuit(input []byte) ([]byte, error) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	output := make([]byte, len(input))
	for i := range output {
		output[i] = input[i] ^ 0xFF
	}

	return output, nil
}

func NewFLEngine() *FLEngine {
	return &FLEngine{
		nodes:       make(map[string]*FLNode),
		models:      make(map[string]*FLModel),
		privacyBudget: &FLPrivacyBudget{
			Epsilon:     1.0,
			Delta:       1e-5,
			NoiseScale:  1.0,
			MaxGradient: 5.0,
		},
		secureAgg: &FLSecureAggregation{
			EncryptedShares: make(map[string]*HECiphertext),
		},
	}
}

func (fl *FLEngine) RegisterNode(node *FLNode) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	node.LastUpdate = time.Now()
	fl.nodes[node.NodeID] = node

	return nil
}

func (fl *FLEngine) InitializeModel(modelID string, initialWeights []float64) (*FLModel, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	model := &FLModel{
		ModelID:      modelID,
		Weights:      initialWeights,
		Gradient:     make([]float64, len(initialWeights)),
		Version:      "1.0.0",
		Accuracy:     0.0,
		Participants: 0,
		AggregatedAt: time.Now(),
	}

	fl.models[modelID] = model
	fl.globalModel = model

	return model, nil
}

func (fl *FLEngine) AggregateUpdates(request *FLAggregationRequest) (*FLAggregationResult, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	model, exists := fl.models[request.ModelID]
	if !exists {
		return nil, ErrFLInvalidModel
	}

	if len(request.Weights) != len(model.Weights) {
		return nil, ErrFLInvalidUpdate
	}

	weightSum := make([]float64, len(model.Weights))
	for i := range model.Weights {
		weightSum[i] = model.Weights[i] + request.Weights[i]*float64(request.DatasetSize)
	}

	totalWeight := float64(request.DatasetSize) + 1.0
	avgWeights := make([]float64, len(model.Weights))
	for i := range avgWeights {
		avgWeights[i] = weightSum[i] / totalWeight
	}

	noise := make([]float64, len(avgWeights))
	noiseScale := fl.privacyBudget.NoiseScale
	for i := range noise {
		for j := 0; j < 8; j++ {
			randomByte := make([]byte, 1)
			rand.Read(randomByte)
			noise[i] += float64(randomByte[0]) / 255.0
		}
		noise[i] = (noise[i]/8.0 - 0.5) * noiseScale * 2.0
		avgWeights[i] += noise[i]
	}

	model.Weights = avgWeights
	model.Gradient = request.Gradient
	model.Participants++
	model.AggregatedAt = time.Now()

	return &FLAggregationResult{
		Success:           true,
		AggregatedWeights: avgWeights,
		NewVersion:        model.Version,
		Accuracy:          model.Accuracy,
		Participants:      model.Participants,
	}, nil
}

func (fl *FLEngine) ApplyDifferentialPrivacy(gradients []float64, epsilon, delta float64) []float64 {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	noisedGradients := make([]float64, len(gradients))

	for i := range gradients {
		sensitivity := 1.0
		scale := sensitivity / epsilon

		randomValue := 0.0
		for j := 0; j < 8; j++ {
			randomByte := make([]byte, 1)
			rand.Read(randomByte)
			randomValue += float64(randomByte[0]) / 255.0
		}
		randomValue = (randomValue/8.0 - 0.5) * 2.0 * scale

		noisedGradients[i] = gradients[i] + randomValue
	}

	return noisedGradients
}

func (fl *FLEngine) SecureAggregate(encryptedShares map[string]*HECiphertext) (*FLSecureAggregation, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	result := &FLSecureAggregation{
		EncryptedShares: encryptedShares,
	}

	return result, nil
}

func (fl *FLEngine) VerifyUpdate(nodeID string, weights []float64) (bool, error) {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	node, exists := fl.nodes[nodeID]
	if !exists {
		return false, ErrFLInvalidModel
	}

	for _, w := range weights {
		if w > fl.privacyBudget.MaxGradient || w < -fl.privacyBudget.MaxGradient {
			return false, nil
		}
	}

	node.LastUpdate = time.Now()

	return true, nil
}

func NewPDSEngine() *PDSEngine {
	return &PDSEngine{
		data:     make(map[string]*PrivacyDataShare),
		policies: make(map[string]*DataAccessPolicy),
	}
}

func (pds *PDSEngine) RegisterData(dataID, dataType, ownerID string, encryptedData *HECiphertext) error {
	pds.mu.Lock()
	defer pds.mu.Unlock()

	data := &PrivacyDataShare{
		DataID:        dataID,
		DataType:      dataType,
		EncryptedData: encryptedData,
		AccessPolicy:  "default",
		OwnerID:       ownerID,
		CreatedAt:     time.Now(),
	}

	pds.data[dataID] = data

	return nil
}

func (pds *PDSEngine) SetAccessPolicy(dataID string, policy *DataAccessPolicy) error {
	pds.mu.Lock()
	defer pds.mu.Unlock()

	if _, exists := pds.data[dataID]; !exists {
		return ErrPDSInvalidData
	}

	policy.PolicyID = fmt.Sprintf("policy-%s-%d", dataID, time.Now().UnixNano())
	policy.DataID = dataID
	pds.policies[dataID] = policy

	return nil
}

func (pds *PDSEngine) RequestAccess(request *DataAccessRequest) (*DataAccessResponse, error) {
	pds.mu.RLock()
	defer pds.mu.RUnlock()

	data, exists := pds.data[request.DataID]
	if !exists {
		return &DataAccessResponse{
			Success:     false,
			ErrorMessage: "data not found",
		}, ErrPDSInvalidData
	}

	if request.RequesterID == data.OwnerID {
		return &DataAccessResponse{
			Success: true,
			Data:    data.EncryptedData,
		}, nil
	}

	policy, exists := pds.policies[request.DataID]
	if exists && policy.RequiredZKProof {
		if len(request.ZKProof) == 0 {
			return &DataAccessResponse{
				Success:     false,
				ErrorMessage: "ZK proof required",
			}, nil
		}
	}

	return &DataAccessResponse{
		Success: true,
		Data:    data.EncryptedData,
	}, nil
}

func (pds *PDSEngine) GetData(dataID string) (*PrivacyDataShare, error) {
	pds.mu.RLock()
	defer pds.mu.RUnlock()

	data, exists := pds.data[dataID]
	if !exists {
		return nil, ErrPDSInvalidData
	}

	return data, nil
}

func (s *PrivacyComputingService) HEGenerateKeyPair() (*HEPublicKey, *HEPrivateKey, error) {
	return s.heEngine.GenerateKeyPair()
}

func (s *PrivacyComputingService) HEEncrypt(plaintext *big.Int, publicKey *HEPublicKey) (*HECiphertext, error) {
	return s.heEngine.Encrypt(plaintext, publicKey)
}

func (s *PrivacyComputingService) HEDecrypt(ciphertext *HECiphertext, privateKey *HEPrivateKey) (*big.Int, error) {
	return s.heEngine.Decrypt(ciphertext, privateKey)
}

func (s *PrivacyComputingService) HEAdd(ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	return s.heEngine.Add(ct1, ct2)
}

func (s *PrivacyComputingService) HEMultiply(ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	return s.heEngine.Multiply(ct1, ct2)
}

func (s *PrivacyComputingService) HEScalarMultiply(ct *HECiphertext, scalar *big.Int) (*HECiphertext, error) {
	return s.heEngine.ScalarMultiply(ct, scalar)
}

func (s *PrivacyComputingService) MPCSetupProtocol(protocol MPCProtocol, parties []string) error {
	return s.mpcEngine.SetupProtocol(protocol, parties)
}

func (s *PrivacyComputingService) MPCShareSecret(partyID string, secret []byte, threshold int) ([]*MPCShare, error) {
	return s.mpcEngine.ShareSecret(partyID, secret, threshold)
}

func (s *PrivacyComputingService) MPCReconstructSecret(shares []*MPCShare, threshold int) ([]byte, error) {
	return s.mpcEngine.ReconstructSecret(shares, threshold)
}

func (s *PrivacyComputingService) MPCCompute(parties []string, computationType string) (*MPCComputation, error) {
	return s.mpcEngine.ComputeMPC(parties, computationType)
}

func (s *PrivacyComputingService) FLRegisterNode(node *FLNode) error {
	return s.flEngine.RegisterNode(node)
}

func (s *PrivacyComputingService) FLInitializeModel(modelID string, initialWeights []float64) (*FLModel, error) {
	return s.flEngine.InitializeModel(modelID, initialWeights)
}

func (s *PrivacyComputingService) FLAggregateUpdates(request *FLAggregationRequest) (*FLAggregationResult, error) {
	return s.flEngine.AggregateUpdates(request)
}

func (s *PrivacyComputingService) FLApplyDifferentialPrivacy(gradients []float64, epsilon, delta float64) []float64 {
	return s.flEngine.ApplyDifferentialPrivacy(gradients, epsilon, delta)
}

func (s *PrivacyComputingService) PDSRegisterData(dataID, dataType, ownerID string, encryptedData *HECiphertext) error {
	return s.pdsEngine.RegisterData(dataID, dataType, ownerID, encryptedData)
}

func (s *PrivacyComputingService) PDSSetAccessPolicy(dataID string, policy *DataAccessPolicy) error {
	return s.pdsEngine.SetAccessPolicy(dataID, policy)
}

func (s *PrivacyComputingService) PDSRequestAccess(request *DataAccessRequest) (*DataAccessResponse, error) {
	return s.pdsEngine.RequestAccess(request)
}

func (s *PrivacyComputingService) SetPrivacyBudget(userID string, budget *FLPrivacyBudget) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.privacyBudget[userID] = budget
}

func (s *PrivacyComputingService) GetPrivacyBudget(userID string) *FLPrivacyBudget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.privacyBudget[userID]
}

func (s *PrivacyComputingService) PerformPrivacyCheck(data []float64, budget *FLPrivacyBudget) bool {
	for _, v := range data {
		if abs(v) > budget.MaxGradient {
			return false
		}
	}
	return true
}

type HEOperationRequest struct {
	Operation string
	Operand1  interface{}
	Operand2  interface{}
}

type HEOperationResponse struct {
	Success      bool
	Result       *HECiphertext
	ErrorMessage string
}

func (s *PrivacyComputingService) PerformHEOperation(ctx context.Context, req *HEOperationRequest) (*HEOperationResponse, error) {
	_ = time.Now()

	switch req.Operation {
	case "add":
		ct1, ok1 := req.Operand1.(*HECiphertext)
		ct2, ok2 := req.Operand2.(*HECiphertext)
		if !ok1 || !ok2 {
			return &HEOperationResponse{
				Success:      false,
				ErrorMessage: "invalid operands for add",
			}, ErrHEInvalidParams
		}

		result, err := s.HEAdd(ct1, ct2)
		if err != nil {
			return &HEOperationResponse{
				Success:      false,
				ErrorMessage: err.Error(),
			}, err
		}

		return &HEOperationResponse{
			Success: true,
			Result:  result,
		}, nil

	case "multiply":
		ct1, ok1 := req.Operand1.(*HECiphertext)
		ct2, ok2 := req.Operand2.(*HECiphertext)
		if !ok1 || !ok2 {
			return &HEOperationResponse{
				Success:      false,
				ErrorMessage: "invalid operands for multiply",
			}, ErrHEInvalidParams
		}

		result, err := s.HEMultiply(ct1, ct2)
		if err != nil {
			return &HEOperationResponse{
				Success:      false,
				ErrorMessage: err.Error(),
			}, err
		}

		return &HEOperationResponse{
			Success: true,
			Result:  result,
		}, nil

	default:
		return &HEOperationResponse{
			Success:      false,
			ErrorMessage: "unsupported operation",
		}, ErrHEOperationFailed
	}
}

func (s *PrivacyComputingService) GetFLNodes() []*FLNode {
	s.flEngine.mu.RLock()
	defer s.flEngine.mu.RUnlock()

	nodes := make([]*FLNode, 0, len(s.flEngine.nodes))
	for _, node := range s.flEngine.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (s *PrivacyComputingService) GetFLModels() []*FLModel {
	s.flEngine.mu.RLock()
	defer s.flEngine.mu.RUnlock()

	models := make([]*FLModel, 0, len(s.flEngine.models))
	for _, model := range s.flEngine.models {
		models = append(models, model)
	}

	return models
}

func (s *PrivacyComputingService) GetRegisteredData() []*PrivacyDataShare {
	s.pdsEngine.mu.RLock()
	defer s.pdsEngine.mu.RUnlock()

	data := make([]*PrivacyDataShare, 0, len(s.pdsEngine.data))
	for _, d := range s.pdsEngine.data {
		data = append(data, d)
	}

	return data
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type MPCComputationRequest struct {
	Protocol      MPCProtocol
	PartyIDs      []string
	ComputationType string
	Inputs        map[string][]byte
}

type MPCComputationResponse struct {
	Success     bool
	Result      []byte
	ErrorMessage string
	Duration    time.Duration
}

func (s *PrivacyComputingService) ExecuteMPCComputation(ctx context.Context, req *MPCComputationRequest) (*MPCComputationResponse, error) {
	start := time.Now()

	err := s.mpcEngine.SetupProtocol(req.Protocol, req.PartyIDs)
	if err != nil {
		return &MPCComputationResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	_, err = s.mpcEngine.ComputeMPC(req.PartyIDs, req.ComputationType)
	if err != nil {
		return &MPCComputationResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	output := []byte(fmt.Sprintf("computed-%s", req.ComputationType))

	return &MPCComputationResponse{
		Success:     true,
		Result:      output,
		Duration:    time.Since(start),
	}, nil
}

type MPCShareDistribution struct {
	ShareID    string   `json:"shareId"`
	PartyID    string   `json:"partyId"`
	Shares     [][]byte `json:"shares"`
	Threshold  int      `json:"threshold"`
	TotalShares int     `json:"totalShares"`
}

func (s *PrivacyComputingService) DistributeMPCShares(ctx context.Context, partyID string, secret []byte, threshold, totalParties int) (*MPCShareDistribution, error) {
	shares, err := s.mpcEngine.ShareSecret(partyID, secret, threshold)
	if err != nil {
		return nil, err
	}

	shareData := make([][]byte, len(shares))
	for i, share := range shares {
		shareData[i] = share.ShareData
	}

	return &MPCShareDistribution{
		ShareID:     fmt.Sprintf("dist-%s-%d", partyID, time.Now().UnixNano()),
		PartyID:     partyID,
		Shares:      shareData,
		Threshold:   threshold,
		TotalShares: totalParties,
	}, nil
}

func (s *PrivacyComputingService) ReconstructMPCShares(ctx context.Context, shares []*MPCShare, threshold int) ([]byte, error) {
	return s.mpcEngine.ReconstructSecret(shares, threshold)
}

type PrivacyVerificationRequest struct {
	DataID      string
	ZKProof     []byte
	Purpose     string
}

type PrivacyVerificationResponse struct {
	Valid        bool
	Errors       []string
	Warnings     []string
}

func (s *PrivacyComputingService) VerifyPrivacyCompliance(ctx context.Context, req *PrivacyVerificationRequest) (*PrivacyVerificationResponse, error) {
	errors := make([]string, 0)
	warnings := make([]string, 0)

	if len(req.ZKProof) == 0 {
		errors = append(errors, "ZK proof required for privacy verification")
	}

	return &PrivacyVerificationResponse{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}, nil
}

func (s *PrivacyComputingService) ExportEncryptedData(ctx context.Context, dataID string, exportFormat string) ([]byte, error) {
	data, err := s.pdsEngine.GetData(dataID)
	if err != nil {
		return nil, err
	}

	exportData := map[string]interface{}{
		"data_id":      data.DataID,
		"data_type":    data.DataType,
		"export_format": exportFormat,
		"created_at":   data.CreatedAt,
	}

	jsonData, err := json.Marshal(exportData)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (s *PrivacyComputingService) GenerateMPCProof(ctx context.Context, computation *MPCComputation) ([]byte, error) {
	proof := make([]byte, 64)
	rand.Read(proof)

	return proof, nil
}

func (s *PrivacyComputingService) VerifyMPCProof(ctx context.Context, computation *MPCComputation, proof []byte) (bool, error) {
	return len(proof) > 0, nil
}

type HEVerificationRequest struct {
	Ciphertext   *HECiphertext
	PublicKey    *HEPublicKey
	Plaintext    *big.Int
}

type HEVerificationResponse struct {
	Valid         bool
	VerifiedValue *big.Int
	ErrorMessage  string
}

func (s *PrivacyComputingService) VerifyHECiphertext(ctx context.Context, req *HEVerificationRequest) (*HEVerificationResponse, error) {
	if req.Ciphertext == nil {
		return &HEVerificationResponse{
			Valid:        false,
			ErrorMessage: "ciphertext is nil",
		}, ErrHEInvalidParams
	}

	return &HEVerificationResponse{
		Valid:         true,
		VerifiedValue: big.NewInt(0),
	}, nil
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeFromBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func (s *PrivacyComputingService) GetMPCPartyCount() int {
	s.mpcEngine.mu.RLock()
	defer s.mpcEngine.mu.RUnlock()

	return len(s.mpcEngine.parties)
}

func (s *PrivacyComputingService) GetFLNodeCount() int {
	s.flEngine.mu.RLock()
	defer s.flEngine.mu.RUnlock()

	return len(s.flEngine.nodes)
}

func (s *PrivacyComputingService) GetPDSDataCount() int {
	s.pdsEngine.mu.RLock()
	defer s.pdsEngine.mu.RUnlock()

	return len(s.pdsEngine.data)
}
