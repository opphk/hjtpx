package service

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// DilithiumLevel 定义 Dilithium 安全级别
type DilithiumLevel int

const (
	Dilithium2 DilithiumLevel = iota // 128位安全级别
	Dilithium3                       // 192位安全级别
	Dilithium5                       // 256位安全级别
)

// DilithiumConfig Dilithium 配置
type DilithiumConfig struct {
	Level DilithiumLevel
}

// DilithiumPublicKeyV2 Dilithium 公钥
type DilithiumPublicKeyV2 struct {
	Rho []byte   // 随机种子
	T1  [][]int16 // 多项式矩阵
	T2  [][]int16 // 多项式矩阵
}

// DilithiumPrivateKeyV2 Dilithium 私钥
type DilithiumPrivateKeyV2 struct {
	PublicKey DilithiumPublicKeyV2
	Rho1      []byte   // 第二个随机种子
	K         int      // 维度参数
	Tr        []byte   // 哈希种子
	S1        [][]int16 // 小范数多项式向量
	S2        [][]int16 // 小范数多项式向量
	T0        [][]int16 // 多项式向量
}

// DilithiumSignatureV2 Dilithium 签名
type DilithiumSignatureV2 struct {
	C  []byte     // 挑战值
	Z  [][]int16 // 响应向量
	H  []byte     // 提示向量的编码
}

// DilithiumService CRYSTALS-Dilithium 签名服务
type DilithiumService struct {
	mu     sync.RWMutex
	config *DilithiumConfig
}

// NewDilithiumService 创建新的 Dilithium 服务
func NewDilithiumService(config *DilithiumConfig) *DilithiumService {
	if config == nil {
		config = &DilithiumConfig{Level: Dilithium2}
	}
	return &DilithiumService{
		config: config,
	}
}

// GenerateKeyPair 生成 Dilithium 密钥对
func (ds *DilithiumService) GenerateKeyPair() (*DilithiumPublicKeyV2, *DilithiumPrivateKeyV2, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	var k, l, n int
	switch ds.config.Level {
	case Dilithium2:
		k = 4
		l = 4
		n = 256
	case Dilithium3:
		k = 6
		l = 5
		n = 256
	case Dilithium5:
		k = 8
		l = 7
		n = 256
	}

	// 生成随机种子
	rho := make([]byte, 32)
	rho1 := make([]byte, 32)
	tr := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, rho); err != nil {
		return nil, nil, err
	}
	if _, err := io.ReadFull(rand.Reader, rho1); err != nil {
		return nil, nil, err
	}
	if _, err := io.ReadFull(rand.Reader, tr); err != nil {
		return nil, nil, err
	}

	// 生成密钥矩阵（模拟）
	s1 := make([][]int16, l)
	s2 := make([][]int16, k)
	t1 := make([][]int16, k)
	t2 := make([][]int16, k)
	t0 := make([][]int16, k)

	for i := 0; i < l; i++ {
		s1[i] = make([]int16, n)
		for j := 0; j < n; j++ {
			s1[i][j] = int16(randInt(32) - 16) // 小范数
		}
	}

	for i := 0; i < k; i++ {
		s2[i] = make([]int16, n)
		t1[i] = make([]int16, n)
		t2[i] = make([]int16, n)
		t0[i] = make([]int16, n)
		for j := 0; j < n; j++ {
			s2[i][j] = int16(randInt(32) - 16)
			t1[i][j] = int16(randInt(16))
			t2[i][j] = int16(randInt(16) - 8)
			t0[i][j] = int16(randInt(8))
		}
	}

	publicKey := &DilithiumPublicKeyV2{
		Rho: rho,
		T1:  t1,
		T2:  t2,
	}

	privateKey := &DilithiumPrivateKeyV2{
		PublicKey: *publicKey,
		Rho1:      rho1,
		K:         k,
		Tr:        tr,
		S1:        s1,
		S2:        s2,
		T0:        t0,
	}

	return publicKey, privateKey, nil
}

// Sign 使用私钥对消息进行签名
func (ds *DilithiumService) Sign(privateKey *DilithiumPrivateKeyV2, message []byte) (*DilithiumSignatureV2, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}

	// 模拟签名过程
	msgHash := sha512.Sum512(message)
	
	c := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, c); err != nil {
		return nil, err
	}
	
	// 组合消息哈希和随机挑战作为签名
	sigC := make([]byte, 64)
	copy(sigC[:32], msgHash[:32])
	copy(sigC[32:], c)
	
	k := privateKey.K
	z := make([][]int16, k)
	for i := 0; i < k; i++ {
		z[i] = make([]int16, 256)
		for j := 0; j < 256; j++ {
			z[i][j] = int16(randInt(64) - 32)
		}
	}
	
	h := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, h); err != nil {
		return nil, err
	}

	return &DilithiumSignatureV2{
		C: sigC,
		Z: z,
		H: h,
	}, nil
}

// Verify 使用公钥验证签名
func (ds *DilithiumService) Verify(publicKey *DilithiumPublicKeyV2, message []byte, signature *DilithiumSignatureV2) (bool, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if publicKey == nil || signature == nil {
		return false, errors.New("public key or signature is nil")
	}

	if len(signature.C) < 32 {
		return false, nil
	}

	// 模拟验证过程
	msgHash := sha512.Sum512(message)
	for i := 0; i < 32; i++ {
		if signature.C[i] != msgHash[i] {
			return false, nil
		}
	}

	return true, nil
}

// SerializePublicKey 序列化公钥
func (ds *DilithiumService) SerializePublicKey(publicKey *DilithiumPublicKey) ([]byte, error) {
	return json.Marshal(publicKey)
}

// DeserializePublicKey 反序列化公钥
func (ds *DilithiumService) DeserializePublicKey(data []byte) (*DilithiumPublicKeyV2, error) {
	var pubKey DilithiumPublicKeyV2
	if err := json.Unmarshal(data, &pubKey); err != nil {
		return nil, err
	}
	return &pubKey, nil
}

// SerializeSignature 序列化签名
func (ds *DilithiumService) SerializeSignature(signature *DilithiumSignatureV2) ([]byte, error) {
	return json.Marshal(signature)
}

// DeserializeSignature 反序列化签名
func (ds *DilithiumService) DeserializeSignature(data []byte) (*DilithiumSignatureV2, error) {
	var sig DilithiumSignatureV2
	if err := json.Unmarshal(data, &sig); err != nil {
		return nil, err
	}
	return &sig, nil
}

// BatchVerify 批量验证签名
func (ds *DilithiumService) BatchVerify(publicKeys []*DilithiumPublicKeyV2, messages [][]byte, signatures []*DilithiumSignatureV2) (bool, error) {
	if len(publicKeys) != len(messages) || len(messages) != len(signatures) {
		return false, errors.New("invalid batch sizes")
	}

	for i := 0; i < len(publicKeys); i++ {
		valid, err := ds.Verify(publicKeys[i], messages[i], signatures[i])
		if err != nil {
			return false, err
		}
		if !valid {
			return false, nil
		}
	}

	return true, nil
}
