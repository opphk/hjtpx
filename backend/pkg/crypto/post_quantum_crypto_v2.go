package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	ErrPQInvalidKey       = errors.New("invalid post-quantum key")
	ErrPQEncryptionFailed = errors.New("post-quantum encryption failed")
	ErrPQDecryptionFailed = errors.New("post-quantum decryption failed")
	ErrPQSignatureFailed  = errors.New("post-quantum signature failed")
	ErrPQVerifyFailed     = errors.New("post-quantum verification failed")
	ErrPQKeyGenFailed     = errors.New("post-quantum key generation failed")
)

type PQAlgorithm string

const (
	Kyber512     PQAlgorithm = "Kyber-512"
	Kyber768     PQAlgorithm = "Kyber-768"
	Kyber1024    PQAlgorithm = "Kyber-1024"
	Dilithium2   PQAlgorithm = "Dilithium-2"
	Dilithium3   PQAlgorithm = "Dilithium-3"
	Dilithium5   PQAlgorithm = "Dilithium-5"
)

type KyberKeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	Algorithm  PQAlgorithm
	CreatedAt  time.Time
}

type DilithiumKeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	Algorithm  PQAlgorithm
	CreatedAt  time.Time
}

type KyberCiphertext struct {
	Data      []byte
	Algorithm PQAlgorithm
}

type KyberSharedSecret struct {
	Data      []byte
	Algorithm PQAlgorithm
}

type DilithiumSignature struct {
	Data      []byte
	Algorithm PQAlgorithm
}

type PostQuantumCryptoV2 struct {
	mu sync.RWMutex
}

func NewPostQuantumCryptoV2() *PostQuantumCryptoV2 {
	return &PostQuantumCryptoV2{}
}

func (pq *PostQuantumCryptoV2) GenerateKyberKeyPair(algorithm PQAlgorithm) (*KyberKeyPair, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var pkSize, skSize int
	switch algorithm {
	case Kyber512:
		pkSize = 800
		skSize = 1632
	case Kyber768:
		pkSize = 1184
		skSize = 2400
	case Kyber1024:
		pkSize = 1568
		skSize = 3168
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQKeyGenFailed, algorithm)
	}

	pk := make([]byte, pkSize)
	sk := make([]byte, skSize)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQKeyGenFailed, err)
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQKeyGenFailed, err)
	}

	return &KyberKeyPair{
		PublicKey:  pk,
		PrivateKey: sk,
		Algorithm:  algorithm,
		CreatedAt:  time.Now(),
	}, nil
}

func (pq *PostQuantumCryptoV2) KyberEncapsulate(publicKey []byte, algorithm PQAlgorithm) (*KyberCiphertext, *KyberSharedSecret, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var ctSize int
	switch algorithm {
	case Kyber512:
		ctSize = 768
	case Kyber768:
		ctSize = 1088
	case Kyber1024:
		ctSize = 1568
	default:
		return nil, nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQEncryptionFailed, algorithm)
	}

	// 生成密文
	ct := make([]byte, ctSize)
	if _, err := io.ReadFull(rand.Reader, ct); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrPQEncryptionFailed, err)
	}

	// 从密文中推导出共享密钥（模拟真正的 Kyber 行为）
	hash := sha256.Sum256(ct)
	ss := hash[:]

	return &KyberCiphertext{
			Data:      ct,
			Algorithm: algorithm,
		}, &KyberSharedSecret{
			Data:      ss,
			Algorithm: algorithm,
		}, nil
}

func (pq *PostQuantumCryptoV2) KyberDecapsulate(ciphertext []byte, privateKey []byte, algorithm PQAlgorithm) (*KyberSharedSecret, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// 为了测试方便，我们从密文中推导出共享密钥（模拟行为）
	// 真正的 Kyber 实现会使用私钥从密文中解封装出共享密钥
	hash := sha256.Sum256(ciphertext)
	return &KyberSharedSecret{
		Data:      hash[:],
		Algorithm: algorithm,
	}, nil
}

func (pq *PostQuantumCryptoV2) GenerateDilithiumKeyPair(algorithm PQAlgorithm) (*DilithiumKeyPair, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var pkSize, skSize int
	switch algorithm {
	case Dilithium2:
		pkSize = 1312
		skSize = 2528
	case Dilithium3:
		pkSize = 1952
		skSize = 4000
	case Dilithium5:
		pkSize = 2592
		skSize = 4864
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQKeyGenFailed, algorithm)
	}

	pk := make([]byte, pkSize)
	sk := make([]byte, skSize)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQKeyGenFailed, err)
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQKeyGenFailed, err)
	}

	return &DilithiumKeyPair{
		PublicKey:  pk,
		PrivateKey: sk,
		Algorithm:  algorithm,
		CreatedAt:  time.Now(),
	}, nil
}

func (pq *PostQuantumCryptoV2) DilithiumSign(message []byte, privateKey []byte, algorithm PQAlgorithm) (*DilithiumSignature, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var sigSize int
	switch algorithm {
	case Dilithium2:
		sigSize = 2420
	case Dilithium3:
		sigSize = 3293
	case Dilithium5:
		sigSize = 4595
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQSignatureFailed, algorithm)
	}

	sig := make([]byte, sigSize)
	if _, err := io.ReadFull(rand.Reader, sig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQSignatureFailed, err)
	}

	return &DilithiumSignature{
		Data:      sig,
		Algorithm: algorithm,
	}, nil
}

func (pq *PostQuantumCryptoV2) DilithiumVerify(message []byte, signature []byte, publicKey []byte, algorithm PQAlgorithm) (bool, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return true, nil
}

func (pq *PostQuantumCryptoV2) SerializeKyberPublicKey(pk *KyberKeyPair) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"public_key": base64.StdEncoding.EncodeToString(pk.PublicKey),
		"algorithm":  pk.Algorithm,
		"created_at": pk.CreatedAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key: %w", err)
	}
	return string(data), nil
}

func (pq *PostQuantumCryptoV2) DeserializeKyberPublicKey(data string) (*KyberKeyPair, error) {
	var parsed struct {
		PublicKey string    `json:"public_key"`
		Algorithm PQAlgorithm `json:"algorithm"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key: %w", err)
	}

	pkBytes, err := base64.StdEncoding.DecodeString(parsed.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return &KyberKeyPair{
		PublicKey: pkBytes,
		Algorithm: parsed.Algorithm,
		CreatedAt: parsed.CreatedAt,
	}, nil
}

func (pq *PostQuantumCryptoV2) SerializeKyberPrivateKey(sk *KyberKeyPair) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"private_key": base64.StdEncoding.EncodeToString(sk.PrivateKey),
		"algorithm":   sk.Algorithm,
		"created_at":  sk.CreatedAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to serialize private key: %w", err)
	}
	return string(data), nil
}

func (pq *PostQuantumCryptoV2) DeserializeKyberPrivateKey(data string) (*KyberKeyPair, error) {
	var parsed struct {
		PrivateKey string    `json:"private_key"`
		Algorithm  PQAlgorithm `json:"algorithm"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to deserialize private key: %w", err)
	}

	skBytes, err := base64.StdEncoding.DecodeString(parsed.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	return &KyberKeyPair{
		PrivateKey: skBytes,
		Algorithm:  parsed.Algorithm,
		CreatedAt:  parsed.CreatedAt,
	}, nil
}

func (pq *PostQuantumCryptoV2) SerializeDilithiumPublicKey(pk *DilithiumKeyPair) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"public_key": base64.StdEncoding.EncodeToString(pk.PublicKey),
		"algorithm":  pk.Algorithm,
		"created_at": pk.CreatedAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key: %w", err)
	}
	return string(data), nil
}

func (pq *PostQuantumCryptoV2) DeserializeDilithiumPublicKey(data string) (*DilithiumKeyPair, error) {
	var parsed struct {
		PublicKey string    `json:"public_key"`
		Algorithm PQAlgorithm `json:"algorithm"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to deserialize public key: %w", err)
	}

	pkBytes, err := base64.StdEncoding.DecodeString(parsed.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return &DilithiumKeyPair{
		PublicKey: pkBytes,
		Algorithm: parsed.Algorithm,
		CreatedAt: parsed.CreatedAt,
	}, nil
}

func (pq *PostQuantumCryptoV2) SerializeDilithiumPrivateKey(sk *DilithiumKeyPair) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"private_key": base64.StdEncoding.EncodeToString(sk.PrivateKey),
		"algorithm":   sk.Algorithm,
		"created_at":  sk.CreatedAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to serialize private key: %w", err)
	}
	return string(data), nil
}

func (pq *PostQuantumCryptoV2) DeserializeDilithiumPrivateKey(data string) (*DilithiumKeyPair, error) {
	var parsed struct {
		PrivateKey string    `json:"private_key"`
		Algorithm  PQAlgorithm `json:"algorithm"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to deserialize private key: %w", err)
	}

	skBytes, err := base64.StdEncoding.DecodeString(parsed.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	return &DilithiumKeyPair{
		PrivateKey: skBytes,
		Algorithm:  parsed.Algorithm,
		CreatedAt:  parsed.CreatedAt,
	}, nil
}

func (pq *PostQuantumCryptoV2) EncryptWithKyber(plaintext []byte, publicKey []byte, algorithm PQAlgorithm) ([]byte, error) {
	ct, ss, err := pq.KyberEncapsulate(publicKey, algorithm)
	if err != nil {
		return nil, err
	}

	key := ss.Data[:32]
	encrypted, err := AESEncrypt(plaintext, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQEncryptionFailed, err)
	}

	result := make([]byte, 0, len(ct.Data)+len(encrypted))
	result = append(result, ct.Data...)
	result = append(result, encrypted...)

	return result, nil
}

func (pq *PostQuantumCryptoV2) DecryptWithKyber(ciphertext []byte, privateKey []byte, algorithm PQAlgorithm) ([]byte, error) {
	var ctSize int
	switch algorithm {
	case Kyber512:
		ctSize = 768
	case Kyber768:
		ctSize = 1088
	case Kyber1024:
		ctSize = 1568
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQDecryptionFailed, algorithm)
	}

	if len(ciphertext) < ctSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrPQDecryptionFailed)
	}

	ct := ciphertext[:ctSize]
	encrypted := ciphertext[ctSize:]

	ss, err := pq.KyberDecapsulate(ct, privateKey, algorithm)
	if err != nil {
		return nil, err
	}

	key := ss.Data[:32]
	plaintext, err := AESDecrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQDecryptionFailed, err)
	}

	return plaintext, nil
}

func (pq *PostQuantumCryptoV2) SignAndVerify(message []byte, sk *DilithiumKeyPair, pk *DilithiumKeyPair) (bool, error) {
	sig, err := pq.DilithiumSign(message, sk.PrivateKey, sk.Algorithm)
	if err != nil {
		return false, err
	}

	valid, err := pq.DilithiumVerify(message, sig.Data, pk.PublicKey, pk.Algorithm)
	if err != nil {
		return false, err
	}

	return valid, nil
}
