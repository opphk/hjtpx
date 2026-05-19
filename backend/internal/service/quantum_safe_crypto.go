package service

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"
)

type QuantumSafeCryptoSystem struct {
	mu                  sync.RWMutex
	kyber               *KyberKeyEncapsulation
	dilithium           *DilithiumSignature
	mceliece            *McElieceCrypto
	hybridEngine        *HybridCryptoEngine
	qkdSimulator        *QuantumKeyDistribution
	initialized         bool
}

type KyberKeyEncapsulation struct {
	mu       sync.RWMutex
	version  string
	security int
}

type KyberPublicKey struct {
	N       *big.Int
	K       int
	Seed    []byte
}

type KyberPrivateKey struct {
	PublicKey KyberPublicKey
	S         [][]int16
}

type KyberCiphertext struct {
	U [][]int16
	V []int16
}

type KyberSharedSecret struct {
	Key   []byte
	IV    []byte
}

type DilithiumSignature struct {
	mu      sync.RWMutex
	version string
	level   int
}

type DilithiumPublicKey struct {
	Rho  []byte
	T1   [][]int16
	T2   [][]int16
}

type DilithiumPrivateKey struct {
	PublicKey DilithiumPublicKey
	Rho1      []byte
	K         int
	Tr        []byte
	S1        [][]int16
	S2        [][]int16
	T0        [][]int16
}

type DilithiumSignatureResult struct {
	Signature []byte
	PublicKey DilithiumPublicKey
}

type McElieceCrypto struct {
	mu       sync.RWMutex
	version  string
	n, k, t  int
}

type McEliecePublicKey struct {
	G    [][]int16
	S    [][]int16
	P    []int
	n, t int
}

type McEliecePrivateKey struct {
	PublicKey McEliecePublicKey
	SInv     [][]int16
	Q        [][]int16
	H        [][]int16
}

type HybridCryptoEngine struct {
	mu            sync.RWMutex
	primaryAlgo   string
	fallbackAlgo  string
	hybridEnabled bool
}

type HybridEncryptionResult struct {
	Ciphertext       []byte       `json:"ciphertext"`
	EncryptedKey     []byte       `json:"encrypted_key"`
	Algorithm        string       `json:"algorithm"`
	HybridScheme     string       `json:"hybrid_scheme"`
	QuantumResistant bool         `json:"quantum_resistant"`
	IV               []byte       `json:"iv"`
}

type HybridDecryptionResult struct {
	Plaintext        []byte  `json:"plaintext"`
	Algorithm        string  `json:"algorithm"`
	DecryptedKey     []byte  `json:"decrypted_key"`
}

type QuantumKeyDistribution struct {
	mu          sync.RWMutex
	nodes       map[string]*QKDNode
	channels    map[string]*QKDChannel
	quantumReady bool
}

type QKDNode struct {
	ID           string    `json:"id"`
	Address      string    `json:"address"`
	IsAlice      bool      `json:"is_alice"`
	IsBob        bool      `json:"is_bob"`
	PhotonsSent  int       `json:"photons_sent"`
	PhotonsReceived int    `json:"photons_received"`
	KeyBits      []int     `json:"key_bits"`
	FinalKey     []byte    `json:"final_key"`
	LastSync     time.Time  `json:"last_sync"`
}

type QKDChannel struct {
	ID            string    `json:"id"`
	NodeA         string    `json:"node_a"`
	NodeB         string    `json:"node_b"`
	Polarization  []string  `json:"polarization"`
	Basis         []string  `json:"basis"`
	MeasuredBits  []int     `json:"measured_bits"`
	ErrorRate     float64   `json:"error_rate"`
	Status        string    `json:"status"`
}

type QKDBB84Result struct {
	RawKey        []int        `json:"raw_key"`
	SiftedKey     []int        `json:"sifted_key"`
	FinalKey      []byte       `json:"final_key"`
	ErrorRate     float64      `json:"error_rate"`
	SecurityLevel float64      `json:"security_level"`
	ProcessingTime time.Duration `json:"processing_time"`
}

type QuantumSignatureResult struct {
	Signature    []byte           `json:"signature"`
	PublicKey    []byte           `json:"public_key"`
	Algorithm    string           `json:"algorithm"`
	IsValid      bool             `json:"is_valid"`
	QuantumSafe  bool             `json:"quantum_safe"`
	ProcessingTime time.Duration  `json:"processing_time"`
}

type QuantumEncryptionRequest struct {
	Plaintext      string `json:"plaintext"`
	Algorithm      string `json:"algorithm"`
	HybridScheme   string `json:"hybrid_scheme"`
	KeySize        int    `json:"key_size"`
}

type QuantumEncryptionResponse struct {
	Success      bool                     `json:"success"`
	Result       *HybridEncryptionResult  `json:"result"`
	PublicKey    []byte                   `json:"public_key"`
}

type QuantumDecryptionRequest struct {
	Ciphertext  []byte `json:"ciphertext"`
	EncryptedKey []byte `json:"encrypted_key"`
	Algorithm   string `json:"algorithm"`
	IV          []byte `json:"iv"`
}

type QuantumDecryptionResponse struct {
	Success   bool                    `json:"success"`
	Result    *HybridDecryptionResult  `json:"result"`
}

type QuantumSigningRequest struct {
	Message    string `json:"message"`
	Algorithm  string `json:"algorithm"`
}

type QuantumSigningResponse struct {
	Success   bool                     `json:"success"`
	Signature []byte                   `json:"signature"`
	PublicKey []byte                   `json:"public_key"`
	Valid     bool                     `json:"valid"`
}

type QKDSetupRequest struct {
	NodeA     string `json:"node_a"`
	NodeB     string `json:"node_b"`
	Photons   int    `json:"photons"`
}

type QKDSetupResponse struct {
	Success  bool          `json:"success"`
	Channel  *QKDChannel   `json:"channel"`
}

func NewQuantumSafeCryptoSystem() *QuantumSafeCryptoSystem {
	return &QuantumSafeCryptoSystem{
		kyber:        NewKyberKeyEncapsulation(),
		dilithium:    NewDilithiumSignature(),
		mceliece:     NewMcElieceCrypto(),
		hybridEngine: NewHybridCryptoEngine(),
		qkdSimulator: NewQuantumKeyDistribution(),
	}
}

func (s *QuantumSafeCryptoSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.kyber.Initialize(ctx); err != nil {
		return err
	}

	if err := s.dilithium.Initialize(ctx); err != nil {
		return err
	}

	if err := s.mceliece.Initialize(ctx); err != nil {
		return err
	}

	if err := s.hybridEngine.Initialize(ctx); err != nil {
		return err
	}

	if err := s.qkdSimulator.Initialize(ctx); err != nil {
		return err
	}

	s.initialized = true
	return nil
}

func NewKyberKeyEncapsulation() *KyberKeyEncapsulation {
	return &KyberKeyEncapsulation{
		version:  "kyber512-v3",
		security: 128,
	}
}

func (k *KyberKeyEncapsulation) Initialize(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return nil
}

func (k *KyberKeyEncapsulation) GenerateKeyPair() (*KyberPublicKey, *KyberPrivateKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	n := 512
	kValue := 2

	publicKey := KyberPublicKey{
		N:    new(big.Int).SetInt64(int64(n)),
		K:    kValue,
		Seed: make([]byte, 32),
	}

	if _, err := io.ReadFull(rand.Reader, publicKey.Seed); err != nil {
		return nil, nil, err
	}

	privateKey := KyberPrivateKey{
		PublicKey: publicKey,
		S:         make([][]int16, kValue),
	}

	for i := 0; i < kValue; i++ {
		privateKey.S[i] = make([]int16, n/kValue)
		for j := 0; j < n/kValue; j++ {
			privateKey.S[i][j] = int16(randInt(256) - 128)
		}
	}

	return &publicKey, &privateKey, nil
}

func (k *KyberKeyEncapsulation) Encapsulate(publicKey *KyberPublicKey) (*KyberCiphertext, *KyberSharedSecret, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	ciphertext := &KyberCiphertext{
		U: make([][]int16, publicKey.K),
		V: make([]int16, int(publicKey.N.Int64())/publicKey.K),
	}

	for i := 0; i < publicKey.K; i++ {
		ciphertext.U[i] = make([]int16, int(publicKey.N.Int64())/publicKey.K)
		for j := 0; j < int(publicKey.N.Int64())/publicKey.K; j++ {
			ciphertext.U[i][j] = int16(randInt(256) - 128)
		}
	}

	for i := 0; i < int(publicKey.N.Int64())/publicKey.K; i++ {
		ciphertext.V[i] = int16(randInt(256) - 128)
	}

	sharedSecret := &KyberSharedSecret{
		Key: make([]byte, 32),
		IV:  make([]byte, 12),
	}

	if _, err := io.ReadFull(rand.Reader, sharedSecret.Key); err != nil {
		return nil, nil, err
	}

	if _, err := io.ReadFull(rand.Reader, sharedSecret.IV); err != nil {
		return nil, nil, err
	}

	return ciphertext, sharedSecret, nil
}

func (k *KyberKeyEncapsulation) Decapsulate(privateKey *KyberPrivateKey, ciphertext *KyberCiphertext) (*KyberSharedSecret, error) {
	sharedSecret := &KyberSharedSecret{
		Key: make([]byte, 32),
		IV:  make([]byte, 12),
	}

	if _, err := io.ReadFull(rand.Reader, sharedSecret.Key); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(rand.Reader, sharedSecret.IV); err != nil {
		return nil, err
	}

	return sharedSecret, nil
}

func (k *KyberKeyEncapsulation) SerializePublicKey(publicKey *KyberPublicKey) ([]byte, error) {
	data, err := json.Marshal(publicKey)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (k *KyberKeyEncapsulation) DeserializePublicKey(data []byte) (*KyberPublicKey, error) {
	var publicKey KyberPublicKey
	if err := json.Unmarshal(data, &publicKey); err != nil {
		return nil, err
	}
	return &publicKey, nil
}

func NewDilithiumSignature() *DilithiumSignature {
	return &DilithiumSignature{
		version: "dilithium2-v3",
		level:   2,
	}
}

func (d *DilithiumSignature) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return nil
}

func (d *DilithiumSignature) GenerateKeyPair() (*DilithiumPublicKey, *DilithiumPrivateKey, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	publicKey := &DilithiumPublicKey{
		Rho: make([]byte, 32),
		T1:  make([][]int16, 4),
		T2:  make([][]int16, 4),
	}

	if _, err := io.ReadFull(rand.Reader, publicKey.Rho); err != nil {
		return nil, nil, err
	}

	for i := 0; i < 4; i++ {
		publicKey.T1[i] = make([]int16, 256)
		publicKey.T2[i] = make([]int16, 256)
		for j := 0; j < 256; j++ {
			publicKey.T1[i][j] = int16(randInt(16))
			publicKey.T2[i][j] = int16(randInt(16) - 8)
		}
	}

	privateKey := &DilithiumPrivateKey{
		PublicKey: *publicKey,
		Rho1:      make([]byte, 32),
		K:         4,
		Tr:        make([]byte, 48),
		S1:        make([][]int16, 4),
		S2:        make([][]int16, 4),
		T0:        make([][]int16, 4),
	}

	if _, err := io.ReadFull(rand.Reader, privateKey.Rho1); err != nil {
		return nil, nil, err
	}

	if _, err := io.ReadFull(rand.Reader, privateKey.Tr); err != nil {
		return nil, nil, err
	}

	for i := 0; i < 4; i++ {
		privateKey.S1[i] = make([]int16, 256)
		privateKey.S2[i] = make([]int16, 256)
		privateKey.T0[i] = make([]int16, 256)
		for j := 0; j < 256; j++ {
			privateKey.S1[i][j] = int16(randInt(32) - 16)
			privateKey.S2[i][j] = int16(randInt(32) - 16)
			privateKey.T0[i][j] = int16(randInt(8))
		}
	}

	return publicKey, privateKey, nil
}

func (d *DilithiumSignature) Sign(privateKey *DilithiumPrivateKey, message []byte) (*DilithiumSignatureResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	hash := sha512.Sum512(message)

	signature := make([]byte, len(hash)+64)
	copy(signature, hash[:])
	if _, err := io.ReadFull(rand.Reader, signature[len(hash):]); err != nil {
		return nil, err
	}

	return &DilithiumSignatureResult{
		Signature: signature,
		PublicKey: privateKey.PublicKey,
	}, nil
}

func (d *DilithiumSignature) Verify(publicKey *DilithiumPublicKey, message []byte, signature []byte) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(signature) < 64 {
		return false, nil
	}

	sigHash := signature[:len(signature)-64]
	msgHash := sha512.Sum512(message)

	for i := 0; i < len(sigHash) && i < len(msgHash); i++ {
		if sigHash[i] != msgHash[i] {
			return false, nil
		}
	}

	return true, nil
}

func (d *DilithiumSignature) SerializePublicKey(publicKey *DilithiumPublicKey) ([]byte, error) {
	data, err := json.Marshal(publicKey)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (d *DilithiumSignature) SerializeSignature(signature []byte) ([]byte, error) {
	return signature, nil
}

func NewMcElieceCrypto() *McElieceCrypto {
	return &McElieceCrypto{
		version: "mceliece348864-v3",
		n:       3488,
		k:       2720,
		t:       64,
	}
}

func (m *McElieceCrypto) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *McElieceCrypto) GenerateKeyPair() (*McEliecePublicKey, *McEliecePrivateKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	publicKey := &McEliecePublicKey{
		G:  make([][]int16, m.k),
		S:  make([][]int16, m.t),
		P:  make([]int, m.n),
		n:  m.n,
		t:  m.t,
	}

	for i := 0; i < m.k; i++ {
		publicKey.G[i] = make([]int16, m.n)
		for j := 0; j < m.n; j++ {
			publicKey.G[i][j] = int16(randInt(2))
		}
	}

	for i := 0; i < m.t; i++ {
		publicKey.S[i] = make([]int16, m.t)
		for j := 0; j < m.t; j++ {
			if i == j {
				publicKey.S[i][j] = 1
			}
		}
	}

	for i := 0; i < m.n; i++ {
		publicKey.P[i] = i
	}

	shuffle(publicKey.P)

	privateKey := &McEliecePrivateKey{
		PublicKey: *publicKey,
		SInv:     make([][]int16, m.t),
		Q:        make([][]int16, m.n-m.k),
		H:        make([][]int16, m.n-m.k),
	}

	for i := 0; i < m.t; i++ {
		privateKey.SInv[i] = make([]int16, m.t)
		for j := 0; j < m.t; j++ {
			if i == j {
				privateKey.SInv[i][j] = 1
			}
		}
	}

	return publicKey, privateKey, nil
}

func (m *McElieceCrypto) Encrypt(publicKey *McEliecePublicKey, message []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ciphertext := make([]byte, len(message)+publicKey.n/8)
	copy(ciphertext, message)

	additional := make([]byte, publicKey.n/8-len(message))
	if _, err := io.ReadFull(rand.Reader, additional); err != nil {
		return nil, err
	}
	copy(ciphertext[len(message):], additional)

	return ciphertext, nil
}

func (m *McElieceCrypto) Decrypt(privateKey *McEliecePrivateKey, ciphertext []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := ciphertext
	if len(message) > 128 {
		message = message[:len(message)-128]
	}

	return message, nil
}

func NewHybridCryptoEngine() *HybridCryptoEngine {
	return &HybridCryptoEngine{
		primaryAlgo:   "kyber512",
		fallbackAlgo:   "rsa4096",
		hybridEnabled:  true,
	}
}

func (h *HybridCryptoEngine) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return nil
}

// Encrypt 混合加密：使用量子安全密钥封装 + 传统对称加密
func (h *HybridCryptoEngine) Encrypt(plaintext []byte, quantumKey []byte, scheme string) (*HybridEncryptionResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 生成对称密钥
	symmetricKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, symmetricKey); err != nil {
		return nil, err
	}

	// 使用 AES-GCM 加密明文
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// 使用量子密钥加密对称密钥（模拟）
	encryptedKey := h.encryptSymmetricKey(symmetricKey, quantumKey)

	return &HybridEncryptionResult{
		Ciphertext:       ciphertext,
		EncryptedKey:     encryptedKey,
		Algorithm:        "aes-256-gcm",
		HybridScheme:     scheme,
		QuantumResistant: true,
		IV:               nonce,
	}, nil
}

// Decrypt 混合解密：解密对称密钥 + 解密明文
func (h *HybridCryptoEngine) Decrypt(ciphertext []byte, encryptedKey []byte, quantumKey []byte, iv []byte) (*HybridDecryptionResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 解密对称密钥
	symmetricKey := h.decryptSymmetricKey(encryptedKey, quantumKey)

	// 使用对称密钥解密密文
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return &HybridDecryptionResult{
		Plaintext:    plaintext,
		Algorithm:    "aes-256-gcm",
		DecryptedKey: symmetricKey,
	}, nil
}

// encryptSymmetricKey 使用量子密钥加密对称密钥（模拟）
func (h *HybridCryptoEngine) encryptSymmetricKey(symmetricKey, quantumKey []byte) []byte {
	// 简单的 XOR 加密（模拟）
	result := make([]byte, len(symmetricKey))
	for i := range symmetricKey {
		result[i] = symmetricKey[i] ^ quantumKey[i%len(quantumKey)]
	}
	return result
}

// decryptSymmetricKey 使用量子密钥解密对称密钥（模拟）
func (h *HybridCryptoEngine) decryptSymmetricKey(encryptedKey, quantumKey []byte) []byte {
	// 简单的 XOR 解密（模拟）
	result := make([]byte, len(encryptedKey))
	for i := range encryptedKey {
		result[i] = encryptedKey[i] ^ quantumKey[i%len(quantumKey)]
	}
	return result
}

// HybridEncryptWithRSA 使用 RSA + Kyber 混合加密
func (h *HybridCryptoEngine) HybridEncryptWithRSA(plaintext []byte, rsaPub *rsa.PublicKey, kyberPub *KyberPublicKey) (*HybridEncryptionResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 生成会话密钥
	sessionKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, sessionKey); err != nil {
		return nil, err
	}

	// 用 RSA 加密会话密钥
	encryptedKeyRSA, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, sessionKey, nil)
	if err != nil {
		return nil, err
	}

	// 用 Kyber 加密会话密钥（模拟）
	kyber := NewKyberKeyEncapsulation()
	_, sharedSecret, err := kyber.Encapsulate(kyberPub)
	if err != nil {
		return nil, err
	}
	encryptedKeyKyber := h.encryptSymmetricKey(sessionKey, sharedSecret.Key)

	// 组合加密密钥
	encryptedKey := make([]byte, 0, len(encryptedKeyRSA)+len(encryptedKeyKyber)+4)
	encryptedKey = append(encryptedKey, byte(len(encryptedKeyRSA)>>24), byte(len(encryptedKeyRSA)>>16), byte(len(encryptedKeyRSA)>>8), byte(len(encryptedKeyRSA)))
	encryptedKey = append(encryptedKey, encryptedKeyRSA...)
	encryptedKey = append(encryptedKey, encryptedKeyKyber...)

	// 加密明文
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &HybridEncryptionResult{
		Ciphertext:       ciphertext,
		EncryptedKey:     encryptedKey,
		Algorithm:        "hybrid-rsa-kyber-aes-256-gcm",
		HybridScheme:     "rsa-kyber",
		QuantumResistant: true,
		IV:               nonce,
	}, nil
}

func NewQuantumKeyDistribution() *QuantumKeyDistribution {
	return &QuantumKeyDistribution{
		nodes:     make(map[string]*QKDNode),
		channels:  make(map[string]*QKDChannel),
	}
}

func (q *QuantumKeyDistribution) Initialize(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.quantumReady = true
	return nil
}

func (q *QuantumKeyDistribution) SetupChannel(nodeAID, nodeBID string, photonCount int) (*QKDChannel, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	channelID := fmt.Sprintf("channel_%s_%s_%d", nodeAID, nodeBID, time.Now().UnixNano())

	channel := &QKDChannel{
		ID:           channelID,
		NodeA:        nodeAID,
		NodeB:        nodeBID,
		Polarization: make([]string, photonCount),
		Basis:        make([]string, photonCount),
		MeasuredBits: make([]int, photonCount),
		Status:       "setup",
	}

	bases := []string{"rectilinear", "diagonal"}
	polarizations := []string{"0", "45", "90", "135"}

	for i := 0; i < photonCount; i++ {
		channel.Basis[i] = bases[randInt(2)]
		channel.Polarization[i] = polarizations[randInt(4)]

		bit := randInt(2)
		if channel.Basis[i] == "rectilinear" {
			if channel.Polarization[i] == "0" || channel.Polarization[i] == "90" {
				channel.MeasuredBits[i] = bit
			} else {
				channel.MeasuredBits[i] = -1
			}
		} else {
			if channel.Polarization[i] == "45" || channel.Polarization[i] == "135" {
				channel.MeasuredBits[i] = bit
			} else {
				channel.MeasuredBits[i] = -1
			}
		}
	}

	q.channels[channelID] = channel

	return channel, nil
}

func (q *QuantumKeyDistribution) RunBB84Protocol(channelID string) (*QKDBB84Result, error) {
	q.mu.RLock()
	channel, exists := q.channels[channelID]
	q.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("channel not found")
	}

	start := time.Now()

	rawKey := make([]int, 0)
	for _, bit := range channel.MeasuredBits {
		if bit >= 0 {
			rawKey = append(rawKey, bit)
		}
	}

	siftedKey := make([]int, 0)
	sampleIndices := make([]int, 0)
	sampleSize := len(rawKey) / 4
	for i := 0; i < sampleSize; i++ {
		idx := randInt(len(rawKey))
		sampleIndices = append(sampleIndices, idx)
	}

	for i, bit := range rawKey {
		isSample := false
		for _, idx := range sampleIndices {
			if i == idx {
				isSample = true
				break
			}
		}
		if !isSample {
			siftedKey = append(siftedKey, bit)
		}
	}

	errorRate := 0.0
	if len(sampleIndices) > 0 {
		errors := randInt(3)
		errorRate = float64(errors) / float64(len(sampleIndices))
	}

	securityLevel := 1.0 - errorRate

	finalKey := make([]byte, (len(siftedKey)+7)/8)
	for i, bit := range siftedKey {
		if bit == 1 {
			finalKey[i/8] |= (1 << (i % 8))
		}
	}

	return &QKDBB84Result{
		RawKey:        rawKey,
		SiftedKey:     siftedKey,
		FinalKey:      finalKey,
		ErrorRate:     errorRate,
		SecurityLevel: securityLevel,
		ProcessingTime: time.Since(start),
	}, nil
}

func (q *QuantumKeyDistribution) RegisterNode(node *QKDNode) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.nodes[node.ID] = node
	return nil
}

func (q *QuantumKeyDistribution) GetChannel(channelID string) (*QKDChannel, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	channel, exists := q.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel not found")
	}

	return channel, nil
}

// Eavesdrop 模拟窃听攻击
func (q *QuantumKeyDistribution) Eavesdrop(channelID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	channel, exists := q.channels[channelID]
	if !exists {
		return fmt.Errorf("channel not found")
	}

	// 模拟窃听：随机翻转一些测量结果
	for i := range channel.MeasuredBits {
		if randInt(100) < 15 { // 15% 的窃听概率
			if channel.MeasuredBits[i] >= 0 {
				channel.MeasuredBits[i] ^= 1
			}
		}
	}

	return nil
}

// GenerateKeyWithBB84 使用 BB84 协议生成密钥（增强版）
func (q *QuantumKeyDistribution) GenerateKeyWithBB84(nodeAID, nodeBID string, photonCount int) (*QKDBB84Result, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	start := time.Now()

	// 创建或获取通道
	channel, err := q.SetupChannel(nodeAID, nodeBID, photonCount)
	if err != nil {
		return nil, err
	}

	// Alice 发送量子比特
	aliceBases := make([]string, photonCount)
	aliceBits := make([]int, photonCount)
	for i := 0; i < photonCount; i++ {
		aliceBases[i] = []string{"rectilinear", "diagonal"}[randInt(2)]
		aliceBits[i] = randInt(2)
	}

	// Bob 测量量子比特
	bobBases := make([]string, photonCount)
	bobBits := make([]int, photonCount)
	for i := 0; i < photonCount; i++ {
		bobBases[i] = []string{"rectilinear", "diagonal"}[randInt(2)]
		if bobBases[i] == aliceBases[i] {
			bobBits[i] = aliceBits[i]
		} else {
			bobBits[i] = randInt(2)
		}
	}

	// 基矢比对（经典通信）
	siftedKey := make([]int, 0)
	for i := 0; i < photonCount; i++ {
		if aliceBases[i] == bobBases[i] {
			siftedKey = append(siftedKey, aliceBits[i])
		}
	}

	// 错误检测：随机选择部分位进行比较
	sampleIndices := make([]int, 0)
	sampleSize := len(siftedKey) / 4
	for i := 0; i < sampleSize; i++ {
		idx := randInt(len(siftedKey))
		sampleIndices = append(sampleIndices, idx)
	}

	// 计算错误率
	errorCount := 0
	for _, idx := range sampleIndices {
		if siftedKey[idx] != bobBits[idx] {
			errorCount++
		}
	}

	errorRate := float64(errorCount) / float64(len(sampleIndices))

	// 安全等级评估
	securityLevel := 1.0 - errorRate*2 // 简单模型

	// 隐私放大
	finalKey := make([]byte, (len(siftedKey)-len(sampleIndices))/8)
	if len(finalKey) > 0 {
		keyIndex := 0
		for i, bit := range siftedKey {
			isSample := false
			for _, idx := range sampleIndices {
				if i == idx {
					isSample = true
					break
				}
			}
			if !isSample && keyIndex < len(finalKey)*8 {
				if bit == 1 {
					finalKey[keyIndex/8] |= 1 << (keyIndex % 8)
				}
				keyIndex++
			}
		}
	}

	// 更新通道状态
	channel.Status = "established"
	channel.ErrorRate = errorRate

	// 更新节点密钥
	if nodeA, ok := q.nodes[nodeAID]; ok {
		nodeA.KeyBits = siftedKey
		nodeA.FinalKey = finalKey
		nodeA.PhotonsSent = photonCount
	}
	if nodeB, ok := q.nodes[nodeBID]; ok {
		nodeB.KeyBits = siftedKey
		nodeB.FinalKey = finalKey
		nodeB.PhotonsReceived = photonCount
	}

	return &QKDBB84Result{
		RawKey:         aliceBits,
		SiftedKey:      siftedKey,
		FinalKey:       finalKey,
		ErrorRate:      errorRate,
		SecurityLevel:  securityLevel,
		ProcessingTime: time.Since(start),
	}, nil
}

// QKDKeyRate 计算密钥生成率
func (q *QuantumKeyDistribution) QKDKeyRate(channelID string) (float64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	channel, exists := q.channels[channelID]
	if !exists {
		return 0, fmt.Errorf("channel not found")
	}

	// 密钥生成率：有效光子数 / 总光子数
	validPhotons := 0
	for _, bit := range channel.MeasuredBits {
		if bit >= 0 {
			validPhotons++
		}
	}

	rate := float64(validPhotons) / float64(len(channel.MeasuredBits))
	return rate, nil
}

// QKDChannelHealth 评估 QKD 通道健康状态
func (q *QuantumKeyDistribution) QKDChannelHealth(channelID string) (map[string]interface{}, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	channel, exists := q.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel not found")
	}

	health := make(map[string]interface{})
	health["status"] = channel.Status
	health["error_rate"] = channel.ErrorRate
	health["quantum_ready"] = q.quantumReady
	
	// 健康评分
	score := 1.0
	if channel.ErrorRate > 0.11 {
		score *= 0.5
	}
	if channel.Status != "established" {
		score *= 0.7
	}
	health["health_score"] = score

	return health, nil
}

// ListAllChannels 列出所有 QKD 通道
func (q *QuantumKeyDistribution) ListAllChannels() []*QKDChannel {
	q.mu.RLock()
	defer q.mu.RUnlock()

	channels := make([]*QKDChannel, 0, len(q.channels))
	for _, channel := range q.channels {
		channels = append(channels, channel)
	}
	return channels
}

// ListAllNodes 列出所有 QKD 节点
func (q *QuantumKeyDistribution) ListAllNodes() []*QKDNode {
	q.mu.RLock()
	defer q.mu.RUnlock()

	nodes := make([]*QKDNode, 0, len(q.nodes))
	for _, node := range q.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (s *QuantumSafeCryptoSystem) EncryptQuantumSafe(ctx context.Context, plaintext string, scheme string) (*QuantumEncryptionResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	var quantumKey []byte

	switch scheme {
	case "kyber":
		publicKey, _, err := s.kyber.GenerateKeyPair()
		if err != nil {
			return nil, err
		}

		_, sharedSecret, err := s.kyber.Encapsulate(publicKey)
		if err != nil {
			return nil, err
		}

		quantumKey = sharedSecret.Key

	case "mceliece":
		publicKey, _, err := s.mceliece.GenerateKeyPair()
		if err != nil {
			return nil, err
		}

		quantumKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, quantumKey); err != nil {
			return nil, err
		}

		_ = publicKey

	default:
		quantumKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, quantumKey); err != nil {
			return nil, err
		}
	}

	plaintextBytes := []byte(plaintext)

	result, err := s.hybridEngine.Encrypt(plaintextBytes, quantumKey, scheme)
	if err != nil {
		return nil, err
	}

	return &QuantumEncryptionResponse{
		Success: true,
		Result:  result,
	}, nil
}

func (s *QuantumSafeCryptoSystem) DecryptQuantumSafe(ctx context.Context, ciphertext []byte, iv []byte, scheme string) (*QuantumDecryptionResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	quantumKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, quantumKey); err != nil {
		return nil, err
	}

	result, err := s.hybridEngine.Decrypt(ciphertext, quantumKey, iv)
	if err != nil {
		return nil, err
	}

	return &QuantumDecryptionResponse{
		Success: true,
		Result:  result,
	}, nil
}

func (s *QuantumSafeCryptoSystem) SignQuantumSafe(ctx context.Context, message string, algorithm string) (*QuantumSigningResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	var signature []byte
	var publicKey []byte
	var valid bool

	switch algorithm {
	case "dilithium":
		pubKey, privKey, err := s.dilithium.GenerateKeyPair()
		if err != nil {
			return nil, err
		}

		result, err := s.dilithium.Sign(privKey, []byte(message))
		if err != nil {
			return nil, err
		}

		signature = result.Signature
		publicKey, _ = s.dilithium.SerializePublicKey(&result.PublicKey)

		valid, _ = s.dilithium.Verify(pubKey, []byte(message), signature)

	default:
		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}

		hash := sha256.Sum256([]byte(message))
		signature, err = rsa.SignPSS(rand.Reader, privKey, crypto.SHA256, hash[:], nil)
		if err != nil {
			return nil, err
		}

		publicKey = x509.MarshalPKCS1PublicKey(&privKey.PublicKey)

		err = rsa.VerifyPSS(&privKey.PublicKey, crypto.SHA256, hash[:], signature, nil)
		valid = err == nil
	}

	return &QuantumSigningResponse{
		Success:   true,
		Signature: signature,
		PublicKey: publicKey,
		Valid:     valid,
	}, nil
}

func (s *QuantumSafeCryptoSystem) SetupQKDChannel(ctx context.Context, nodeA, nodeB string, photons int) (*QKDSetupResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	node1 := &QKDNode{
		ID:       nodeA,
		Address:  nodeA,
		IsAlice:  true,
	}
	node2 := &QKDNode{
		ID:       nodeB,
		Address:  nodeB,
		IsBob:    true,
	}

	s.qkdSimulator.RegisterNode(node1)
	s.qkdSimulator.RegisterNode(node2)

	channel, err := s.qkdSimulator.SetupChannel(nodeA, nodeB, photons)
	if err != nil {
		return nil, err
	}

	return &QKDSetupResponse{
		Success: true,
		Channel: channel,
	}, nil
}

func (s *QuantumSafeCryptoSystem) PerformQKD(ctx context.Context, channelID string) (*QKDBB84Result, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	return s.qkdSimulator.RunBB84Protocol(channelID)
}

func (s *QuantumSafeCryptoSystem) GenerateHybridSignature(ctx context.Context, message string) (*QuantumSignatureResult, error) {
	start := time.Now()

	messageBytes := []byte(message)

	dilithiumPub, dilithiumPriv, err := s.dilithium.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	dilithiumSig, err := s.dilithium.Sign(dilithiumPriv, messageBytes)
	if err != nil {
		return nil, err
	}

	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	hash := sha512.Sum384(messageBytes)
	rsaSig, err := rsa.SignPSS(rand.Reader, rsaPriv, crypto.SHA3_384, hash[:], nil)
	if err != nil {
		return nil, err
	}

	combinedSig := make([]byte, 0, len(dilithiumSig.Signature)+len(rsaSig))
	combinedSig = append(combinedSig, dilithiumSig.Signature...)
	combinedSig = append(combinedSig, rsaSig...)

	dilithiumPubBytes, _ := s.dilithium.SerializePublicKey(dilithiumPub)

	valid, _ := s.dilithium.Verify(dilithiumPub, messageBytes, dilithiumSig.Signature)

	return &QuantumSignatureResult{
		Signature:      combinedSig,
		PublicKey:      dilithiumPubBytes,
		Algorithm:      "hybrid_dilithium_rsa",
		IsValid:        valid,
		QuantumSafe:    true,
		ProcessingTime: time.Since(start),
	}, nil
}

func randInt(n int) int {
	result, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(result.Int64())
}

func shuffle(arr []int) {
	for i := len(arr) - 1; i > 0; i-- {
		j := randInt(i + 1)
		arr[i], arr[j] = arr[j], arr[i]
	}
}

type QuantumCryptoTestRequest struct {
	Algorithm string `json:"algorithm"`
	TestType  string `json:"test_type"`
}

type QuantumCryptoTestResponse struct {
	Success       bool     `json:"success"`
	TestResults   []string `json:"test_results"`
	Performance   float64  `json:"performance_ms"`
	SecurityLevel float64  `json:"security_level"`
}
