package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrInvalidCircuit          = errors.New("invalid circuit")
	ErrProofGenerationFailure = errors.New("proof generation failed")
	ErrSetupPhaseFailure      = errors.New("setup phase failed")
	ErrInvalidWitnessCircuit  = errors.New("invalid witness for circuit")
	ErrConstraintNotSatisfied = errors.New("constraint not satisfied")
)

type CircuitInput struct {
	Public []string `json:"public"`
	Secret []string `json:"secret"`
}

type Constraint struct {
	A []string `json:"a"`
	B []string `json:"b"`
	C []string `json:"c"`
}

type ArithmeticCircuit struct {
	NumInputs    int         `json:"num_inputs"`
	NumOutputs   int         `json:"num_outputs"`
	NumConstraints int       `json:"num_constraints"`
	Constraints  []Constraint `json:"constraints"`
	WitnessOrder []string    `json:"witness_order"`
}

type SetupParams struct {
	Alpha        *big.Int    `json:"alpha"`
	Beta         *big.Int    `json:"beta"`
	Gamma        *big.Int    `json:"gamma"`
	Delta        *big.Int    `json:"delta"`
	IC           []*Point    `json:"ic"`
	VC           []*Point    `json:"vc"`
	SetupTime    int64       `json:"setup_time"`
	IsToxicWaste bool        `json:"is_toxic_waste"`
}

type ProvingKey struct {
	IC        []*Point    `json:"ic"`
	A         []*Point    `json:"a"`
	B         []*Point    `json:"b"`
	C         []*Point    `json:"c"`
	H         []*Point    `json:"h"`
	K         []*Point    `json:"k"`
	Protocol  string      `json:"protocol"`
	CurveType CurveType   `json:"curve_type"`
	CreatedAt int64       `json:"created_at"`
}

type VerificationKey struct {
	IC        []*Point    `json:"ic"`
	VC        []*Point    `json:"vc"`
	Alpha     *Point      `json:"alpha"`
	Beta      *Point      `json:"beta"`
	Gamma     *Point      `json:"gamma"`
	Delta     *Point      `json:"delta"`
	Protocol  string      `json:"protocol"`
	CurveType CurveType   `json:"curve_type"`
	CreatedAt int64       `json:"created_at"`
}

type SNARKProof struct {
	A        *Point `json:"a"`
	B        *Point `json:"b"`
	C        *Point `json:"c"`
	PublicInputs []string `json:"public_inputs"`
	Protocol     string   `json:"protocol"`
	CreatedAt    int64    `json:"created_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ZKSNARKService struct {
	mu          sync.RWMutex
	circuit     *ArithmeticCircuit
	setupParams *SetupParams
	provingKey  *ProvingKey
	verKey      *VerificationKey
	curveType   CurveType
}

type SNARKProofRequest struct {
	Witness     map[string]interface{} `json:"witness"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	CircuitID   string                 `json:"circuit_id,omitempty"`
	Protocol    string                 `json:"protocol"`
}

type SNARKProofResponse struct {
	Proof      *SNARKProof `json:"proof"`
	PublicHash string      `json:"public_hash"`
	CreatedAt  int64       `json:"created_at"`
	ExpiresAt  int64       `json:"expires_at"`
}

type SNARKVerificationRequest struct {
	Proof       *SNARKProof `json:"proof"`
	PublicInputs []string   `json:"public_inputs"`
	Protocol    string      `json:"protocol"`
}

type SNARKVerificationResponse struct {
	Valid       bool   `json:"valid"`
	PublicHash  string `json:"public_hash"`
	VerifiedAt  int64  `json:"verified_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func NewZKSNARKService(curveType CurveType) *ZKSNARKService {
	return &ZKSNARKService{
		curveType: curveType,
	}
}

func (s *ZKSNARKService) Setup(circuit *ArithmeticCircuit) error {
	if circuit == nil || circuit.NumConstraints <= 0 {
		return ErrInvalidCircuit
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.circuit = circuit

	alpha := generateRandomScalar()
	beta := generateRandomScalar()
	gamma := generateRandomScalar()
	delta := generateRandomScalar()

	s.setupParams = &SetupParams{
		Alpha:        alpha,
		Beta:         beta,
		Gamma:        gamma,
		Delta:        delta,
		IC:           make([]*Point, 0),
		VC:           make([]*Point, 0),
		SetupTime:    time.Now().Unix(),
		IsToxicWaste: true,
	}

	ic := make([]*Point, circuit.NumInputs+1)
	for i := 0; i <= circuit.NumInputs; i++ {
		ic[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
	}
	s.setupParams.IC = ic

	vc := make([]*Point, circuit.NumOutputs+1)
	for i := 0; i <= circuit.NumOutputs; i++ {
		vc[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
	}
	s.setupParams.VC = vc

	s.provingKey = &ProvingKey{
		IC:        ic,
		A:         make([]*Point, circuit.NumConstraints+1),
		B:         make([]*Point, circuit.NumConstraints+1),
		C:         make([]*Point, circuit.NumConstraints+1),
		H:         make([]*Point, circuit.NumConstraints),
		K:         make([]*Point, circuit.NumOutputs+1),
		Protocol:  "G16",
		CurveType: s.curveType,
		CreatedAt: time.Now().Unix(),
	}

	for i := 0; i <= circuit.NumConstraints; i++ {
		s.provingKey.A[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
		s.provingKey.B[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
		s.provingKey.C[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
	}

	for i := 0; i < circuit.NumConstraints; i++ {
		s.provingKey.H[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
	}

	for i := 0; i <= circuit.NumOutputs; i++ {
		s.provingKey.K[i] = &Point{
			X: new(big.Int).SetBytes(generateRandomBytes(32)),
			Y: new(big.Int).SetBytes(generateRandomBytes(32)),
		}
	}

	s.verKey = &VerificationKey{
		IC:        ic,
		VC:        vc,
		Alpha:     &Point{X: alpha, Y: new(big.Int).SetBytes(generateRandomBytes(32))},
		Beta:      &Point{X: beta, Y: new(big.Int).SetBytes(generateRandomBytes(32))},
		Gamma:    &Point{X: gamma, Y: new(big.Int).SetBytes(generateRandomBytes(32))},
		Delta:    &Point{X: delta, Y: new(big.Int).SetBytes(generateRandomBytes(32))},
		Protocol: "G16",
		CurveType: s.curveType,
		CreatedAt: time.Now().Unix(),
	}

	return nil
}

func (s *ZKSNARKService) GenerateProof(request *SNARKProofRequest) (*SNARKProofResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.provingKey == nil {
		return nil, fmt.Errorf("proving key not initialized")
	}

	witnessValues := make([]*big.Int, 0)
	for _, name := range s.circuit.WitnessOrder {
		if val, ok := request.Witness[name]; ok {
			switch v := val.(type) {
			case float64:
				witnessValues = append(witnessValues, big.NewInt(int64(v)))
			case int:
				witnessValues = append(witnessValues, big.NewInt(int64(v)))
			case string:
				numVal := new(big.Int)
				numVal.SetString(v, 10)
				witnessValues = append(witnessValues, numVal)
			default:
				witnessValues = append(witnessValues, big.NewInt(0))
			}
		} else {
			witnessValues = append(witnessValues, big.NewInt(0))
		}
	}

	if len(witnessValues) != len(s.circuit.WitnessOrder) {
		return nil, ErrInvalidWitness
	}

	aX := new(big.Int)
	for i := 0; i < len(witnessValues) && i < len(s.provingKey.A); i++ {
		aX.Add(aX, new(big.Int).Mul(witnessValues[i], s.provingKey.A[i].X))
	}

	bX := new(big.Int)
	for i := 0; i < len(witnessValues) && i < len(s.provingKey.B); i++ {
		bX.Add(bX, new(big.Int).Mul(witnessValues[i], s.provingKey.B[i].X))
	}

	cX := new(big.Int)
	for i := 0; i < len(witnessValues) && i < len(s.provingKey.C); i++ {
		cX.Add(cX, new(big.Int).Mul(witnessValues[i], s.provingKey.C[i].X))
	}

	proof := &SNARKProof{
		A:        s.provingKey.A[0],
		B:        s.provingKey.B[0],
		C:        s.provingKey.C[0],
		Protocol: "G16",
		CreatedAt: time.Now().Unix(),
		Metadata: make(map[string]string),
	}

	publicInputs := make([]string, 0)
	for _, input := range request.PublicInputs {
		publicInputs = append(publicInputs, fmt.Sprintf("%v", input))
	}
	proof.PublicInputs = publicInputs

	publicHashData := []byte{}
	for _, input := range publicInputs {
		publicHashData = append(publicHashData, []byte(input)...)
	}
	hash := sha256.Sum256(publicHashData)
	publicHash := base64.StdEncoding.EncodeToString(hash[:])

	return &SNARKProofResponse{
		Proof:      proof,
		PublicHash: publicHash,
		CreatedAt:  time.Now().Unix(),
		ExpiresAt:  time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

func (s *ZKSNARKService) VerifyProof(request *SNARKVerificationRequest) (*SNARKVerificationResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.verKey == nil {
		return nil, fmt.Errorf("verification key not initialized")
	}

	if request.Proof == nil {
		return &SNARKVerificationResponse{
			Valid:      false,
			VerifiedAt: time.Now().Unix(),
		}, nil
	}

	if request.Proof.A == nil || request.Proof.B == nil || request.Proof.C == nil {
		return &SNARKVerificationResponse{
			Valid:      false,
			VerifiedAt: time.Now().Unix(),
		}, nil
	}

	valid := s.performVerification(request.Proof, request.PublicInputs)

	publicHashData := []byte{}
	for _, input := range request.PublicInputs {
		publicHashData = append(publicHashData, []byte(input)...)
	}
	hash := sha256.Sum256(publicHashData)
	publicHash := base64.StdEncoding.EncodeToString(hash[:])

	return &SNARKVerificationResponse{
		Valid:      valid,
		PublicHash: publicHash,
		VerifiedAt: time.Now().Unix(),
		Metadata: map[string]string{
			"protocol": request.Proof.Protocol,
		},
	}, nil
}

func (s *ZKSNARKService) performVerification(proof *SNARKProof, publicInputs []string) bool {
	if len(publicInputs) == 0 {
		return true
	}

	if len(s.verKey.IC) < len(publicInputs)+1 {
		return false
	}

	sumX := new(big.Int)
	for i := 0; i < len(publicInputs); i++ {
		if i+1 < len(s.verKey.IC) {
			sumX.Add(sumX, s.verKey.IC[i+1].X)
		}
	}

	proofAValid := proof.A != nil && proof.A.X != nil
	proofBValid := proof.B != nil && proof.B.X != nil
	proofCValid := proof.C != nil && proof.C.X != nil

	return proofAValid && proofBValid && proofCValid
}

func (s *ZKSNARKService) GetProvingKey() (*ProvingKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.provingKey == nil {
		return nil, fmt.Errorf("proving key not initialized")
	}

	return s.provingKey, nil
}

func (s *ZKSNARKService) GetVerificationKey() (*VerificationKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.verKey == nil {
		return nil, fmt.Errorf("verification key not initialized")
	}

	return s.verKey, nil
}

func (s *ZKSNARKService) CreateRangeProofCircuit(lower, upper int64) *ArithmeticCircuit {
	return &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: int(upper - lower + 1),
		Constraints: []Constraint{
			{A: []string{"x", fmt.Sprintf("-%d", lower)}, B: []string{"1"}, C: []string{"out_lower"}},
			{A: []string{"x", fmt.Sprintf("-%d", upper)}, B: []string{"1"}, C: []string{"out_upper"}},
		},
		WitnessOrder: []string{"x", "out_lower", "out_upper"},
	}
}

func (s *ZKSNARKService) CreateMembershipProofCircuit(setSize int) *ArithmeticCircuit {
	constraints := make([]Constraint, 0)
	for i := 0; i < setSize; i++ {
		constraints = append(constraints, Constraint{
			A: []string{fmt.Sprintf("b%d", i)},
			B: []string{"1"},
			C: []string{"sum"},
		})
	}

	return &ArithmeticCircuit{
		NumInputs:     setSize + 1,
		NumOutputs:    1,
		NumConstraints: setSize,
		Constraints:   constraints,
		WitnessOrder:  generateWitnessOrder(setSize),
	}
}

func generateWitnessOrder(setSize int) []string {
	order := make([]string, 0, setSize+2)
	order = append(order, "x")
	for i := 0; i < setSize; i++ {
		order = append(order, fmt.Sprintf("b%d", i))
	}
	order = append(order, "sum")
	return order
}

func (pk *ProvingKey) ToJSON() (string, error) {
	data, err := json.Marshal(pk)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proving key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ParseProvingKeyFromJSON(jsonStr string) (*ProvingKey, error) {
	data, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var pk ProvingKey
	if err := json.Unmarshal(data, &pk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proving key: %w", err)
	}

	return &pk, nil
}

func (vk *VerificationKey) ToJSON() (string, error) {
	data, err := json.Marshal(vk)
	if err != nil {
		return "", fmt.Errorf("failed to marshal verification key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ParseVerificationKeyFromJSON(jsonStr string) (*VerificationKey, error) {
	data, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var vk VerificationKey
	if err := json.Unmarshal(data, &vk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verification key: %w", err)
	}

	return &vk, nil
}

func (p *SNARKProof) ToJSON() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SNARK proof: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ParseSNARKProofFromJSON(jsonStr string) (*SNARKProof, error) {
	data, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var proof SNARKProof
	if err := json.Unmarshal(data, &proof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SNARK proof: %w", err)
	}

	return &proof, nil
}

func generateRandomScalar() *big.Int {
	bytes := generateRandomBytes(32)
	return new(big.Int).SetBytes(bytes)
}

func generateRandomBytes(n int) []byte {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return bytes
}

func (s *ZKSNARKService) ExportProvingKey() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.provingKey == nil {
		return nil, fmt.Errorf("proving key not initialized")
	}

	data, err := json.Marshal(s.provingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proving key: %w", err)
	}

	return data, nil
}

func (s *ZKSNARKService) ExportVerificationKey() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.verKey == nil {
		return nil, fmt.Errorf("verification key not initialized")
	}

	data, err := json.Marshal(s.verKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification key: %w", err)
	}

	return data, nil
}

func (s *ZKSNARKService) ImportProvingKey(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pk ProvingKey
	if err := json.Unmarshal(data, &pk); err != nil {
		return fmt.Errorf("failed to unmarshal proving key: %w", err)
	}

	s.provingKey = &pk
	return nil
}

func (s *ZKSNARKService) ImportVerificationKey(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var vk VerificationKey
	if err := json.Unmarshal(data, &vk); err != nil {
		return fmt.Errorf("failed to unmarshal verification key: %w", err)
	}

	s.verKey = &vk
	return nil
}

func (s *ZKSNARKService) CreateKnowledgeProofCircuit(predicate string) *ArithmeticCircuit {
	return &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x", predicate}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}
}

func (s *ZKSNARKService) CreateEqualityProofCircuit() *ArithmeticCircuit {
	return &ArithmeticCircuit{
		NumInputs:     2,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x", "y", "-1"}, B: []string{"1"}, C: []string{"diff"}},
		},
		WitnessOrder: []string{"x", "y", "diff"},
	}
}

func (s *ZKSNARKService) ValidateWitness(witness map[string]interface{}, circuit *ArithmeticCircuit) error {
	for _, name := range circuit.WitnessOrder {
		if _, ok := witness[name]; !ok {
			return fmt.Errorf("%w: missing witness %s", ErrInvalidWitness, name)
		}
	}
	return nil
}
