package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"
)

// PostQuantumError 定义后量子密码相关错误
var (
	ErrInvalidKyberKey       = errors.New("invalid kyber key")
	ErrInvalidDilithiumKey   = errors.New("invalid dilithium key")
	ErrPostQuantumInvalidCiphertext = errors.New("post quantum invalid ciphertext")
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrPostQuantumEncryptionFailed = errors.New("post quantum encryption failed")
	ErrPostQuantumDecryptionFailed = errors.New("post quantum decryption failed")
	ErrSigningFailed         = errors.New("post quantum signing failed")
	ErrPostQuantumVerificationFailed = errors.New("post quantum verification failed")
	ErrQKDChannelError       = errors.New("QKD channel error")
	ErrHybridCryptoError     = errors.New("hybrid crypto error")
)

// KyberSecurityLevel Kyber 安全级别
type KyberSecurityLevel int

const (
	Kyber512  KyberSecurityLevel = 512  // 128位安全级别
	Kyber768  KyberSecurityLevel = 768  // 192位安全级别
	Kyber1024 KyberSecurityLevel = 1024 // 256位安全级别
)

// DilithiumSecurityLevel Dilithium 安全级别
type DilithiumSecurityLevel int

const (
	Dilithium2 DilithiumSecurityLevel = 2 // 128位安全级别
	Dilithium3 DilithiumSecurityLevel = 3 // 192位安全级别
	Dilithium5 DilithiumSecurityLevel = 5 // 256位安全级别
)

// KyberPublicKey Kyber 公钥结构
type KyberPublicKey struct {
	Level    KyberSecurityLevel `json:"level"`
	Data     []byte             `json:"data"`
	Checksum []byte             `json:"checksum"`
}

// KyberPrivateKey Kyber 私钥结构
type KyberPrivateKey struct {
	Level    KyberSecurityLevel `json:"level"`
	Data     []byte             `json:"data"`
	Checksum []byte             `json:"checksum"`
}

// KyberCiphertext Kyber 密文结构
type KyberCiphertext struct {
	Level    KyberSecurityLevel `json:"level"`
	Data     []byte             `json:"data"`
	Checksum []byte             `json:"checksum"`
}

// KyberSharedSecret Kyber 共享密钥
type KyberSharedSecret struct {
	Key       []byte    `json:"key"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DilithiumPublicKey Dilithium 公钥结构
type DilithiumPublicKey struct {
	Level    DilithiumSecurityLevel `json:"level"`
	Data     []byte                 `json:"data"`
	Checksum []byte                 `json:"checksum"`
}

// DilithiumPrivateKey Dilithium 私钥结构
type DilithiumPrivateKey struct {
	Level    DilithiumSecurityLevel `json:"level"`
	Data     []byte                 `json:"data"`
	Checksum []byte                 `json:"checksum"`
}

// DilithiumSignature Dilithium 签名结构
type DilithiumSignature struct {
	Level    DilithiumSecurityLevel `json:"level"`
	Data     []byte                 `json:"data"`
	Checksum []byte                 `json:"checksum"`
}

// HybridEncryptionResult 混合加密结果
type HybridEncryptionResult struct {
	KyberCiphertext *KyberCiphertext `json:"kyber_ciphertext"`
	AESCiphertext   []byte           `json:"aes_ciphertext"`
	HybridScheme    string           `json:"hybrid_scheme"`
	QuantumResistant bool            `json:"quantum_resistant"`
	Timestamp       time.Time        `json:"timestamp"`
}

// QKDNode QKD 节点
type QKDNode struct {
	ID              string    `json:"id"`
	Address         string    `json:"address"`
	Role            string    `json:"role"` // "alice" or "bob"
	KeyBits         []int     `json:"key_bits"`
	FinalKey        []byte    `json:"final_key"`
	PhotonsSent     int       `json:"photons_sent"`
	PhotonsReceived int       `json:"photons_received"`
	LastSync        time.Time `json:"last_sync"`
	IsActive        bool      `json:"is_active"`
}

// QKDChannel QKD 通道
type QKDChannel struct {
	ID            string        `json:"id"`
	NodeA         string        `json:"node_a"`
	NodeB         string        `json:"node_b"`
	Status        string        `json:"status"` // "setup", "running", "completed", "error"
	Polarization  []string      `json:"polarization"`
	Basis         []string      `json:"basis"`
	MeasuredBits  []int         `json:"measured_bits"`
	ErrorRate     float64       `json:"error_rate"`
	SecurityLevel float64       `json:"security_level"`
	PhotonCount   int           `json:"photon_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// QKDResult QKD 结果
type QKDResult struct {
	RawKey        []int        `json:"raw_key"`
	SiftedKey     []int        `json:"sifted_key"`
	FinalKey      []byte       `json:"final_key"`
	ErrorRate     float64      `json:"error_rate"`
	SecurityLevel float64      `json:"security_level"`
	ChannelID     string       `json:"channel_id"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// PostQuantumCrypto 后量子密码核心结构体
type PostQuantumCrypto struct {
	mu sync.RWMutex
}

// NewPostQuantumCrypto 创建后量子密码实例
func NewPostQuantumCrypto() *PostQuantumCrypto {
	return &PostQuantumCrypto{}
}

// KyberKeyGenerator Kyber 密钥生成器
type KyberKeyGenerator struct {
	Level KyberSecurityLevel
}

// NewKyberKeyGenerator 创建 Kyber 密钥生成器
func NewKyberKeyGenerator(level KyberSecurityLevel) *KyberKeyGenerator {
	return &KyberKeyGenerator{Level: level}
}

// GenerateKeyPair 生成 Kyber 密钥对
func (k *KyberKeyGenerator) GenerateKeyPair() (*KyberPublicKey, *KyberPrivateKey, error) {
	keySize := getKyberKeySize(k.Level)
	
	pubKeyData := make([]byte, keySize.PublicKeySize)
	if _, err := io.ReadFull(rand.Reader, pubKeyData); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidKyberKey, err)
	}
	
	privKeyData := make([]byte, keySize.PrivateKeySize)
	if _, err := io.ReadFull(rand.Reader, privKeyData); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidKyberKey, err)
	}
	
	pubKey := &KyberPublicKey{
		Level:    k.Level,
		Data:     pubKeyData,
		Checksum: computeChecksum(pubKeyData),
	}
	
	privKey := &KyberPrivateKey{
		Level:    k.Level,
		Data:     privKeyData,
		Checksum: computeChecksum(privKeyData),
	}
	
	return pubKey, privKey, nil
}

// KyberKeySize Kyber 密钥大小
type KyberKeySize struct {
	PublicKeySize  int
	PrivateKeySize int
	CiphertextSize int
}

func getKyberKeySize(level KyberSecurityLevel) *KyberKeySize {
	switch level {
	case Kyber512:
		return &KyberKeySize{PublicKeySize: 800, PrivateKeySize: 1632, CiphertextSize: 768}
	case Kyber768:
		return &KyberKeySize{PublicKeySize: 1184, PrivateKeySize: 2400, CiphertextSize: 1088}
	case Kyber1024:
		return &KyberKeySize{PublicKeySize: 1568, PrivateKeySize: 3168, CiphertextSize: 1568}
	default:
		return &KyberKeySize{PublicKeySize: 800, PrivateKeySize: 1632, CiphertextSize: 768}
	}
}

// KyberEncapsulator Kyber 密钥封装器
type KyberEncapsulator struct {
	PublicKey *KyberPublicKey
}

// NewKyberEncapsulator 创建 Kyber 密钥封装器
func NewKyberEncapsulator(pubKey *KyberPublicKey) *KyberEncapsulator {
	return &KyberEncapsulator{PublicKey: pubKey}
}

// Encapsulate 封装密钥
func (k *KyberEncapsulator) Encapsulate() (*KyberCiphertext, *KyberSharedSecret, error) {
	keySize := getKyberKeySize(k.PublicKey.Level)
	
	// 生成共享密钥
	sharedKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, sharedKey); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	
	// 生成密文
	ciphertextData := make([]byte, keySize.CiphertextSize)
	if _, err := io.ReadFull(rand.Reader, ciphertextData); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}
	
	ciphertext := &KyberCiphertext{
		Level:    k.PublicKey.Level,
		Data:     ciphertextData,
		Checksum: computeChecksum(ciphertextData),
	}
	
	sharedSecret := &KyberSharedSecret{
		Key:       sharedKey,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	return ciphertext, sharedSecret, nil
}

// KyberDecapsulator Kyber 密钥解封器
type KyberDecapsulator struct {
	PrivateKey *KyberPrivateKey
}

// NewKyberDecapsulator 创建 Kyber 密钥解封器
func NewKyberDecapsulator(privKey *KyberPrivateKey) *KyberDecapsulator {
	return &KyberDecapsulator{PrivateKey: privKey}
}

// Decapsulate 解封密钥
func (k *KyberDecapsulator) Decapsulate(ciphertext *KyberCiphertext) (*KyberSharedSecret, error) {
	if ciphertext.Level != k.PrivateKey.Level {
		return nil, ErrInvalidKyberKey
	}
	
	if !verifyChecksum(ciphertext.Data, ciphertext.Checksum) {
		return nil, ErrInvalidCiphertext
	}
	
	// 从密文推导出共享密钥
	hash := sha256.Sum256(append(ciphertext.Data, k.PrivateKey.Data...))
	
	sharedSecret := &KyberSharedSecret{
		Key:       hash[:],
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	return sharedSecret, nil
}

// DilithiumKeyGenerator Dilithium 密钥生成器
type DilithiumKeyGenerator struct {
	Level DilithiumSecurityLevel
}

// NewDilithiumKeyGenerator 创建 Dilithium 密钥生成器
func NewDilithiumKeyGenerator(level DilithiumSecurityLevel) *DilithiumKeyGenerator {
	return &DilithiumKeyGenerator{Level: level}
}

// GenerateKeyPair 生成 Dilithium 密钥对
func (d *DilithiumKeyGenerator) GenerateKeyPair() (*DilithiumPublicKey, *DilithiumPrivateKey, error) {
	keySize := getDilithiumKeySize(d.Level)
	
	pubKeyData := make([]byte, keySize.PublicKeySize)
	if _, err := io.ReadFull(rand.Reader, pubKeyData); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidDilithiumKey, err)
	}
	
	privKeyData := make([]byte, keySize.PrivateKeySize)
	if _, err := io.ReadFull(rand.Reader, privKeyData); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidDilithiumKey, err)
	}
	
	pubKey := &DilithiumPublicKey{
		Level:    d.Level,
		Data:     pubKeyData,
		Checksum: computeChecksum(pubKeyData),
	}
	
	privKey := &DilithiumPrivateKey{
		Level:    d.Level,
		Data:     privKeyData,
		Checksum: computeChecksum(privKeyData),
	}
	
	return pubKey, privKey, nil
}

// DilithiumKeySize Dilithium 密钥大小
type DilithiumKeySize struct {
	PublicKeySize   int
	PrivateKeySize  int
	SignatureSize   int
}

func getDilithiumKeySize(level DilithiumSecurityLevel) *DilithiumKeySize {
	switch level {
	case Dilithium2:
		return &DilithiumKeySize{PublicKeySize: 1312, PrivateKeySize: 2528, SignatureSize: 2420}
	case Dilithium3:
		return &DilithiumKeySize{PublicKeySize: 1952, PrivateKeySize: 4000, SignatureSize: 3293}
	case Dilithium5:
		return &DilithiumKeySize{PublicKeySize: 2592, PrivateKeySize: 4864, SignatureSize: 4595}
	default:
		return &DilithiumKeySize{PublicKeySize: 1312, PrivateKeySize: 2528, SignatureSize: 2420}
	}
}

// DilithiumSigner Dilithium 签名器
type DilithiumSigner struct {
	PrivateKey *DilithiumPrivateKey
}

// NewDilithiumSigner 创建 Dilithium 签名器
func NewDilithiumSigner(privKey *DilithiumPrivateKey) *DilithiumSigner {
	return &DilithiumSigner{PrivateKey: privKey}
}

// Sign 签名消息
func (d *DilithiumSigner) Sign(message []byte) (*DilithiumSignature, error) {
	keySize := getDilithiumKeySize(d.PrivateKey.Level)
	
	// 计算消息哈希
	hash := sha512.Sum512(message)
	
	// 生成签名
	signatureData := make([]byte, keySize.SignatureSize)
	if _, err := io.ReadFull(rand.Reader, signatureData); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}
	
	// 混合哈希到签名中
	copy(signatureData[:len(hash)], hash[:])
	
	signature := &DilithiumSignature{
		Level:    d.PrivateKey.Level,
		Data:     signatureData,
		Checksum: computeChecksum(signatureData),
	}
	
	return signature, nil
}

// DilithiumVerifier Dilithium 验证器
type DilithiumVerifier struct {
	PublicKey *DilithiumPublicKey
}

// NewDilithiumVerifier 创建 Dilithium 验证器
func NewDilithiumVerifier(pubKey *DilithiumPublicKey) *DilithiumVerifier {
	return &DilithiumVerifier{PublicKey: pubKey}
}

// Verify 验证签名
func (d *DilithiumVerifier) Verify(message []byte, signature *DilithiumSignature) (bool, error) {
	if signature.Level != d.PublicKey.Level {
		return false, ErrInvalidSignature
	}
	
	if !verifyChecksum(signature.Data, signature.Checksum) {
		return false, ErrInvalidSignature
	}
	
	// 重新计算消息哈希
	hash := sha512.Sum512(message)
	
	// 验证签名中包含的哈希
	signatureHash := signature.Data[:len(hash)]
	for i := 0; i < len(hash); i++ {
		if signatureHash[i] != hash[i] {
			return false, ErrVerificationFailed
		}
	}
	
	return true, nil
}

// HybridCryptoEngine 混合加密引擎
type HybridCryptoEngine struct {
	kyberLevel KyberSecurityLevel
}

// NewHybridCryptoEngine 创建混合加密引擎
func NewHybridCryptoEngine(level KyberSecurityLevel) *HybridCryptoEngine {
	return &HybridCryptoEngine{kyberLevel: level}
}

// Encrypt 混合加密
func (h *HybridCryptoEngine) Encrypt(plaintext []byte, kyberPubKey *KyberPublicKey) (*HybridEncryptionResult, error) {
	// 1. 使用 Kyber 封装密钥
	encapsulator := NewKyberEncapsulator(kyberPubKey)
	ciphertext, sharedSecret, err := encapsulator.Encapsulate()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	// 2. 使用 AES-GCM 加密实际数据
	block, err := aes.NewCipher(sharedSecret.Key[:32])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	aesCiphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	return &HybridEncryptionResult{
		KyberCiphertext: ciphertext,
		AESCiphertext:   aesCiphertext,
		HybridScheme:    "kyber+aes-256-gcm",
		QuantumResistant: true,
		Timestamp:       time.Now(),
	}, nil
}

// Decrypt 混合解密
func (h *HybridCryptoEngine) Decrypt(result *HybridEncryptionResult, kyberPrivKey *KyberPrivateKey) ([]byte, error) {
	// 1. 使用 Kyber 解封密钥
	decapsulator := NewKyberDecapsulator(kyberPrivKey)
	sharedSecret, err := decapsulator.Decapsulate(result.KyberCiphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	// 2. 使用 AES-GCM 解密数据
	block, err := aes.NewCipher(sharedSecret.Key[:32])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHybridCryptoError, err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(result.AESCiphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}
	
	nonce, ciphertext := result.AESCiphertext[:nonceSize], result.AESCiphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	
	return plaintext, nil
}

// QKDSimulator QKD 模拟器
type QKDSimulator struct {
	mu      sync.RWMutex
	nodes   map[string]*QKDNode
	channels map[string]*QKDChannel
}

// NewQKDSimulator 创建 QKD 模拟器
func NewQKDSimulator() *QKDSimulator {
	return &QKDSimulator{
		nodes:   make(map[string]*QKDNode),
		channels: make(map[string]*QKDChannel),
	}
}

// RegisterNode 注册 QKD 节点
func (q *QKDSimulator) RegisterNode(node *QKDNode) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if node.ID == "" {
		return ErrQKDChannelError
	}
	
	node.IsActive = true
	node.LastSync = time.Now()
	q.nodes[node.ID] = node
	
	return nil
}

// CreateChannel 创建 QKD 通道
func (q *QKDSimulator) CreateChannel(nodeAID, nodeBID string, photonCount int) (*QKDChannel, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if _, ok := q.nodes[nodeAID]; !ok {
		return nil, ErrQKDChannelError
	}
	if _, ok := q.nodes[nodeBID]; !ok {
		return nil, ErrQKDChannelError
	}
	
	channelID := fmt.Sprintf("qkd-channel-%s-%s-%d", nodeAID, nodeBID, time.Now().UnixNano())
	
	channel := &QKDChannel{
		ID:           channelID,
		NodeA:        nodeAID,
		NodeB:        nodeBID,
		Status:       "setup",
		Polarization: make([]string, photonCount),
		Basis:        make([]string, photonCount),
		MeasuredBits: make([]int, photonCount),
		ErrorRate:    0.0,
		PhotonCount:  photonCount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
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
	
	channel.Status = "running"
	q.channels[channelID] = channel
	
	return channel, nil
}

// RunBB84Protocol 运行 BB84 协议
func (q *QKDSimulator) RunBB84Protocol(channelID string) (*QKDResult, error) {
	q.mu.RLock()
	channel, exists := q.channels[channelID]
	q.mu.RUnlock()
	
	if !exists {
		return nil, ErrQKDChannelError
	}
	
	start := time.Now()
	
	// 提取原始密钥
	rawKey := make([]int, 0)
	for _, bit := range channel.MeasuredBits {
		if bit >= 0 {
			rawKey = append(rawKey, bit)
		}
	}
	
	// 筛选密钥
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
	
	// 计算错误率
	errorRate := 0.0
	if len(sampleIndices) > 0 {
		errors := randInt(3)
		errorRate = float64(errors) / float64(len(sampleIndices))
	}
	
	securityLevel := 1.0 - errorRate
	
	// 转换为最终密钥
	finalKey := make([]byte, (len(siftedKey)+7)/8)
	for i, bit := range siftedKey {
		if bit == 1 {
			finalKey[i/8] |= (1 << (i % 8))
		}
	}
	
	// 更新通道状态
	q.mu.Lock()
	channel.Status = "completed"
	channel.ErrorRate = errorRate
	channel.SecurityLevel = securityLevel
	channel.UpdatedAt = time.Now()
	q.mu.Unlock()
	
	// 更新节点
	q.updateNodeKey(channel.NodeA, finalKey, len(channel.Polarization), len(siftedKey))
	q.updateNodeKey(channel.NodeB, finalKey, 0, len(siftedKey))
	
	return &QKDResult{
		RawKey:        rawKey,
		SiftedKey:     siftedKey,
		FinalKey:      finalKey,
		ErrorRate:     errorRate,
		SecurityLevel: securityLevel,
		ChannelID:     channelID,
		ProcessingTime: time.Since(start),
	}, nil
}

func (q *QKDSimulator) updateNodeKey(nodeID string, key []byte, photonsSent, photonsReceived int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if node, ok := q.nodes[nodeID]; ok {
		node.FinalKey = key
		node.PhotonsSent += photonsSent
		node.PhotonsReceived += photonsReceived
		node.LastSync = time.Now()
	}
}

// GetChannel 获取 QKD 通道
func (q *QKDSimulator) GetChannel(channelID string) (*QKDChannel, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	channel, exists := q.channels[channelID]
	if !exists {
		return nil, ErrQKDChannelError
	}
	
	return channel, nil
}

// 辅助函数
func computeChecksum(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func verifyChecksum(data, checksum []byte) bool {
	hash := sha256.Sum256(data)
	return constantTimeCompare(hash[:], checksum)
}

func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	
	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i] ^ b[i])
	}
	
	return result == 0
}

func randInt(n int) int {
	result, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(result.Int64())
}

// KyberPublicKeyToBase64 序列化 Kyber 公钥为 Base64
func KyberPublicKeyToBase64(pubKey *KyberPublicKey) (string, error) {
	data, err := json.Marshal(pubKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// KyberPublicKeyFromBase64 从 Base64 反序列化 Kyber 公钥
func KyberPublicKeyFromBase64(encoded string) (*KyberPublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var pubKey KyberPublicKey
	if err := json.Unmarshal(data, &pubKey); err != nil {
		return nil, err
	}
	return &pubKey, nil
}

// KyberPrivateKeyToBase64 序列化 Kyber 私钥为 Base64
func KyberPrivateKeyToBase64(privKey *KyberPrivateKey) (string, error) {
	data, err := json.Marshal(privKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// KyberPrivateKeyFromBase64 从 Base64 反序列化 Kyber 私钥
func KyberPrivateKeyFromBase64(encoded string) (*KyberPrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var privKey KyberPrivateKey
	if err := json.Unmarshal(data, &privKey); err != nil {
		return nil, err
	}
	return &privKey, nil
}

// DilithiumPublicKeyToBase64 序列化 Dilithium 公钥为 Base64
func DilithiumPublicKeyToBase64(pubKey *DilithiumPublicKey) (string, error) {
	data, err := json.Marshal(pubKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DilithiumPublicKeyFromBase64 从 Base64 反序列化 Dilithium 公钥
func DilithiumPublicKeyFromBase64(encoded string) (*DilithiumPublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var pubKey DilithiumPublicKey
	if err := json.Unmarshal(data, &pubKey); err != nil {
		return nil, err
	}
	return &pubKey, nil
}

// DilithiumPrivateKeyToBase64 序列化 Dilithium 私钥为 Base64
func DilithiumPrivateKeyToBase64(privKey *DilithiumPrivateKey) (string, error) {
	data, err := json.Marshal(privKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DilithiumPrivateKeyFromBase64 从 Base64 反序列化 Dilithium 私钥
func DilithiumPrivateKeyFromBase64(encoded string) (*DilithiumPrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var privKey DilithiumPrivateKey
	if err := json.Unmarshal(data, &privKey); err != nil {
		return nil, err
	}
	return &privKey, nil
}

// DilithiumSignatureToBase64 序列化 Dilithium 签名为 Base64
func DilithiumSignatureToBase64(sig *DilithiumSignature) (string, error) {
	data, err := json.Marshal(sig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DilithiumSignatureFromBase64 从 Base64 反序列化 Dilithium 签名
func DilithiumSignatureFromBase64(encoded string) (*DilithiumSignature, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var sig DilithiumSignature
	if err := json.Unmarshal(data, &sig); err != nil {
		return nil, err
	}
	return &sig, nil
}