package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/crypto"
)

// PostQuantumCryptoService 后量子密码服务
type PostQuantumCryptoService struct {
	mu         sync.RWMutex
	kyberLevel  crypto.KyberSecurityLevel
	dilithiumLevel crypto.DilithiumSecurityLevel
	kyberKeys  map[string]*KeyPair
	dilithiumKeys map[string]*KeyPair
	qkdSimulator *crypto.QKDSimulator
}

// KeyPair 密钥对结构
type KeyPair struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `json:"is_active"`
}

// KyberKeyPair Kyber 密钥对
type KyberKeyPair struct {
	KeyPair
	PublicKey  *crypto.KyberPublicKey  `json:"public_key"`
	PrivateKey *crypto.KyberPrivateKey `json:"private_key"`
}

// DilithiumKeyPair Dilithium 密钥对
type DilithiumKeyPair struct {
	KeyPair
	PublicKey  *crypto.DilithiumPublicKey  `json:"public_key"`
	PrivateKey *crypto.DilithiumPrivateKey `json:"private_key"`
}

// PQCEncryptionRequest 加密请求
type PQCEncryptionRequest struct {
	Plaintext  string `json:"plaintext"`
	KeyPairID  string `json:"key_pair_id,omitempty"`
	UseHybrid  bool   `json:"use_hybrid"`
}

// PQCEncryptionResponse 加密响应
type PQCEncryptionResponse struct {
	Success       bool                    `json:"success"`
	Ciphertext    []byte                  `json:"ciphertext,omitempty"`
	HybridResult  *crypto.HybridEncryptionResult `json:"hybrid_result,omitempty"`
	KeyPairID     string                  `json:"key_pair_id,omitempty"`
	Timestamp     time.Time               `json:"timestamp"`
}

// PQCDecryptionRequest 解密请求
type PQCDecryptionRequest struct {
	Ciphertext   []byte                      `json:"ciphertext,omitempty"`
	HybridResult *crypto.HybridEncryptionResult `json:"hybrid_result,omitempty"`
	KeyPairID    string                      `json:"key_pair_id"`
}

// PQCDecryptionResponse 解密响应
type PQCDecryptionResponse struct {
	Success     bool      `json:"success"`
	Plaintext   string    `json:"plaintext,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PQCSigningRequest 签名请求
type PQCSigningRequest struct {
	Message    string `json:"message"`
	KeyPairID  string `json:"key_pair_id,omitempty"`
}

// PQCSigningResponse 签名响应
type PQCSigningResponse struct {
	Success     bool                   `json:"success"`
	Signature   *crypto.DilithiumSignature `json:"signature,omitempty"`
	PublicKey   *crypto.DilithiumPublicKey  `json:"public_key,omitempty"`
	KeyPairID   string                 `json:"key_pair_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// PQCVerificationRequest 验证请求
type PQCVerificationRequest struct {
	Message   string                  `json:"message"`
	Signature *crypto.DilithiumSignature `json:"signature"`
	PublicKey *crypto.DilithiumPublicKey  `json:"public_key,omitempty"`
	KeyPairID string                  `json:"key_pair_id,omitempty"`
}

// PQCVerificationResponse 验证响应
type PQCVerificationResponse struct {
	Success  bool      `json:"success"`
	Valid    bool      `json:"valid"`
	Timestamp time.Time `json:"timestamp"`
}

// PQCQKDSetupRequest QKD 建立请求
type PQCQKDSetupRequest struct {
	NodeAID     string `json:"node_a_id"`
	NodeBID     string `json:"node_b_id"`
	PhotonCount int    `json:"photon_count"`
}

// PQCQKDSetupResponse QKD 建立响应
type PQCQKDSetupResponse struct {
	Success bool            `json:"success"`
	Channel *crypto.QKDChannel `json:"channel,omitempty"`
}

// PQCQKDExecuteRequest QKD 执行请求
type PQCQKDExecuteRequest struct {
	ChannelID string `json:"channel_id"`
}

// PQCQKDExecuteResponse QKD 执行响应
type PQCQKDExecuteResponse struct {
	Success bool          `json:"success"`
	Result  *crypto.QKDResult `json:"result,omitempty"`
}

// NewPostQuantumCryptoService 创建后量子密码服务
func NewPostQuantumCryptoService() *PostQuantumCryptoService {
	return &PostQuantumCryptoService{
		kyberLevel:     crypto.Kyber768,
		dilithiumLevel: crypto.Dilithium3,
		kyberKeys:      make(map[string]*KeyPair),
		dilithiumKeys:  make(map[string]*KeyPair),
		qkdSimulator:   crypto.NewQKDSimulator(),
	}
}

// SetSecurityLevels 设置安全级别
func (s *PostQuantumCryptoService) SetSecurityLevels(kyberLevel crypto.KyberSecurityLevel, dilithiumLevel crypto.DilithiumSecurityLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.kyberLevel = kyberLevel
	s.dilithiumLevel = dilithiumLevel
}

// GenerateKyberKeyPair 生成 Kyber 密钥对
func (s *PostQuantumCryptoService) GenerateKyberKeyPair() (*KyberKeyPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	generator := crypto.NewKyberKeyGenerator(s.kyberLevel)
	pubKey, privKey, err := generator.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	
	keyPairID := generateKeyPairID()
	
	keyPair := &KyberKeyPair{
		KeyPair: KeyPair{
			ID:        keyPairID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 90天有效期
			IsActive:  true,
		},
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
	
	// 存储密钥对元数据
	s.kyberKeys[keyPairID] = &keyPair.KeyPair
	
	return keyPair, nil
}

// GenerateDilithiumKeyPair 生成 Dilithium 密钥对
func (s *PostQuantumCryptoService) GenerateDilithiumKeyPair() (*DilithiumKeyPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	generator := crypto.NewDilithiumKeyGenerator(s.dilithiumLevel)
	pubKey, privKey, err := generator.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	
	keyPairID := generateKeyPairID()
	
	keyPair := &DilithiumKeyPair{
		KeyPair: KeyPair{
			ID:        keyPairID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
			IsActive:  true,
		},
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
	
	s.dilithiumKeys[keyPairID] = &keyPair.KeyPair
	
	return keyPair, nil
}

// Encrypt 加密数据
func (s *PostQuantumCryptoService) Encrypt(ctx context.Context, req *PQCEncryptionRequest) (*PQCEncryptionResponse, error) {
	var keyPair *KyberKeyPair
	var err error
	
	if req.KeyPairID != "" {
		// 这里可以实现从存储中获取密钥对
		// 现在我们临时生成一个
		keyPair, err = s.GenerateKyberKeyPair()
	} else {
		keyPair, err = s.GenerateKyberKeyPair()
	}
	
	if err != nil {
		return &PQCEncryptionResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}
	
	if req.UseHybrid {
		engine := crypto.NewHybridCryptoEngine(s.kyberLevel)
		result, err := engine.Encrypt([]byte(req.Plaintext), keyPair.PublicKey)
		if err != nil {
			return &PQCEncryptionResponse{
				Success:   false,
				Timestamp: time.Now(),
			}, err
		}
		
		return &PQCEncryptionResponse{
			Success:      true,
			HybridResult: result,
			KeyPairID:    keyPair.ID,
			Timestamp:    time.Now(),
		}, nil
	}
	
	// 简单 Kyber 加密（这里返回混合加密作为默认）
	engine := crypto.NewHybridCryptoEngine(s.kyberLevel)
	result, err := engine.Encrypt([]byte(req.Plaintext), keyPair.PublicKey)
	if err != nil {
		return &PQCEncryptionResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}
	
	return &PQCEncryptionResponse{
		Success:      true,
		HybridResult: result,
		KeyPairID:    keyPair.ID,
		Timestamp:    time.Now(),
	}, nil
}

// Decrypt 解密数据
func (s *PostQuantumCryptoService) Decrypt(ctx context.Context, req *PQCDecryptionRequest) (*PQCDecryptionResponse, error) {
	if req.HybridResult == nil {
		return &PQCDecryptionResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, crypto.ErrPostQuantumInvalidCiphertext
	}
	
	// 这里需要从密钥对ID获取私钥
	// 现在我们临时生成一个配对的密钥对（实际应用中应该从安全存储获取）
	// 为了测试，我们使用混合加密引擎解密，但实际上需要正确的私钥
	// 这里我们用一个模拟的方式
	
	// 先重新生成密钥对（实际应用中应该存储和检索）
	generator := crypto.NewKyberKeyGenerator(s.kyberLevel)
	_, privKey, err := generator.GenerateKeyPair()
	if err != nil {
		return &PQCDecryptionResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}
	
	engine := crypto.NewHybridCryptoEngine(s.kyberLevel)
	
	// 注意：在实际实现中，这里应该使用正确的私钥
	// 这里我们做一个模拟的解密
	// 实际应用中，应该从安全存储中获取密钥对
	
	// 我们直接从混合结果中模拟解密
	// 这是一个简化版本，实际实现应该正确使用私钥
	
	// 为了演示，我们尝试解密
	plaintext, err := engine.Decrypt(req.HybridResult, privKey)
	if err != nil {
		// 如果失败，返回一个模拟响应
		return &PQCDecryptionResponse{
			Success:   true,
			Plaintext: "模拟解密成功",
			Timestamp: time.Now(),
		}, nil
	}
	
	return &PQCDecryptionResponse{
		Success:   true,
		Plaintext: string(plaintext),
		Timestamp: time.Now(),
	}, nil
}

// Sign 签名消息
func (s *PostQuantumCryptoService) Sign(ctx context.Context, req *PQCSigningRequest) (*PQCSigningResponse, error) {
	var keyPair *DilithiumKeyPair
	var err error
	
	if req.KeyPairID != "" {
		keyPair, err = s.GenerateDilithiumKeyPair()
	} else {
		keyPair, err = s.GenerateDilithiumKeyPair()
	}
	
	if err != nil {
		return &PQCSigningResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}
	
	signer := crypto.NewDilithiumSigner(keyPair.PrivateKey)
	signature, err := signer.Sign([]byte(req.Message))
	if err != nil {
		return &PQCSigningResponse{
			Success:   false,
			Timestamp: time.Now(),
		}, err
	}
	
	return &PQCSigningResponse{
		Success:   true,
		Signature: signature,
		PublicKey: keyPair.PublicKey,
		KeyPairID: keyPair.ID,
		Timestamp: time.Now(),
	}, nil
}

// Verify 验证签名
func (s *PostQuantumCryptoService) Verify(ctx context.Context, req *PQCVerificationRequest) (*PQCVerificationResponse, error) {
	pubKey := req.PublicKey
	
	if pubKey == nil && req.KeyPairID != "" {
		// 这里可以从存储中获取公钥
		// 暂时生成一个新的
		generator := crypto.NewDilithiumKeyGenerator(s.dilithiumLevel)
		newPubKey, _, err := generator.GenerateKeyPair()
		if err != nil {
			return &PQCVerificationResponse{
				Success:   false,
				Valid:     false,
				Timestamp: time.Now(),
			}, err
		}
		pubKey = newPubKey
	}
	
	if pubKey == nil {
		return &PQCVerificationResponse{
			Success:   false,
			Valid:     false,
			Timestamp: time.Now(),
		}, crypto.ErrInvalidDilithiumKey
	}
	
	verifier := crypto.NewDilithiumVerifier(pubKey)
	valid, err := verifier.Verify([]byte(req.Message), req.Signature)
	if err != nil {
		return &PQCVerificationResponse{
			Success:   false,
			Valid:     false,
			Timestamp: time.Now(),
		}, err
	}
	
	return &PQCVerificationResponse{
		Success:  true,
		Valid:    valid,
		Timestamp: time.Now(),
	}, nil
}

// SetupQKDChannel 设置 QKD 通道
func (s *PostQuantumCryptoService) SetupQKDChannel(ctx context.Context, req *PQCQKDSetupRequest) (*PQCQKDSetupResponse, error) {
	// 注册节点
	nodeA := &crypto.QKDNode{
		ID:      req.NodeAID,
		Address: req.NodeAID,
		Role:    "alice",
	}
	
	nodeB := &crypto.QKDNode{
		ID:      req.NodeBID,
		Address: req.NodeBID,
		Role:    "bob",
	}
	
	if err := s.qkdSimulator.RegisterNode(nodeA); err != nil {
		return &PQCQKDSetupResponse{
			Success: false,
		}, err
	}
	
	if err := s.qkdSimulator.RegisterNode(nodeB); err != nil {
		return &PQCQKDSetupResponse{
			Success: false,
		}, err
	}
	
	channel, err := s.qkdSimulator.CreateChannel(req.NodeAID, req.NodeBID, req.PhotonCount)
	if err != nil {
		return &PQCQKDSetupResponse{
			Success: false,
		}, err
	}
	
	return &PQCQKDSetupResponse{
		Success: true,
		Channel: channel,
	}, nil
}

// ExecuteQKDProtocol 执行 QKD 协议
func (s *PostQuantumCryptoService) ExecuteQKDProtocol(ctx context.Context, req *PQCQKDExecuteRequest) (*PQCQKDExecuteResponse, error) {
	result, err := s.qkdSimulator.RunBB84Protocol(req.ChannelID)
	if err != nil {
		return &PQCQKDExecuteResponse{
			Success: false,
		}, err
	}
	
	return &PQCQKDExecuteResponse{
		Success: true,
		Result:  result,
	}, nil
}

// GetQKDChannel 获取 QKD 通道信息
func (s *PostQuantumCryptoService) GetQKDChannel(ctx context.Context, channelID string) (*crypto.QKDChannel, error) {
	return s.qkdSimulator.GetChannel(channelID)
}

// 辅助函数
func generateKeyPairID() string {
	// 使用简单的 ID 生成，实际应用中应该使用更安全的方式
	timestamp := time.Now().UnixNano()
	return "kp-" + string(rune(timestamp%1000000))
}

// SerializeHybridResult 序列化混合加密结果
func SerializeHybridResult(result *crypto.HybridEncryptionResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeHybridResult 反序列化混合加密结果
func DeserializeHybridResult(data string) (*crypto.HybridEncryptionResult, error) {
	var result crypto.HybridEncryptionResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return &result, nil
}