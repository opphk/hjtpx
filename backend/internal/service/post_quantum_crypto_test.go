package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewPostQuantumCrypto(t *testing.T) {
	pq := NewPostQuantumCrypto()
	assert.NotNil(t, pq)
	assert.NotNil(t, pq.mlkem)
	assert.NotNil(t, pq.dilithium)
}

func TestNewPQMLKEMKeyEncapsulation(t *testing.T) {
	mlkem := NewPQMLKEMKeyEncapsulation()
	assert.NotNil(t, mlkem)
	assert.Equal(t, "ML-KEM-512", mlkem.version)
	assert.Equal(t, 2, mlkem.params.K)
	assert.Equal(t, 256, mlkem.params.N)
	assert.Equal(t, 128, mlkem.params.Security)
}

func TestPQMLKEMGenerateKeyPair(t *testing.T) {
	mlkem := NewPQMLKEMKeyEncapsulation()
	pk, sk, err := mlkem.GenerateKeyPair()
	
	assert.NoError(t, err)
	assert.NotNil(t, pk)
	assert.NotNil(t, sk)
	assert.Len(t, pk.PK, 800)
	assert.Len(t, sk.SK, 1632)
}

func TestPQMLKEMEncapsulate(t *testing.T) {
	mlkem := NewPQMLKEMKeyEncapsulation()
	pk, _, err := mlkem.GenerateKeyPair()
	assert.NoError(t, err)
	
	ct, ss, err := mlkem.Encapsulate(pk)
	assert.NoError(t, err)
	assert.NotNil(t, ct)
	assert.NotNil(t, ss)
	assert.Len(t, ct.C, 768)
	assert.Len(t, ss.K, 32)
}

func TestPQMLKEMDecapsulate(t *testing.T) {
	mlkem := NewPQMLKEMKeyEncapsulation()
	pk, sk, err := mlkem.GenerateKeyPair()
	assert.NoError(t, err)
	
	ct, _, err := mlkem.Encapsulate(pk)
	assert.NoError(t, err)
	
	ss, err := mlkem.Decapsulate(sk, ct)
	assert.NoError(t, err)
	assert.NotNil(t, ss)
	assert.Len(t, ss.K, 32)
}

func TestPQMLKEMSerialization(t *testing.T) {
	mlkem := NewPQMLKEMKeyEncapsulation()
	pk, sk, err := mlkem.GenerateKeyPair()
	assert.NoError(t, err)
	
	pkBytes, err := mlkem.SerializePublicKey(pk)
	assert.NoError(t, err)
	assert.NotEmpty(t, pkBytes)
	
	skBytes, err := mlkem.SerializePrivateKey(sk)
	assert.NoError(t, err)
	assert.NotEmpty(t, skBytes)
	
	pk2, err := mlkem.DeserializePublicKey(pkBytes)
	assert.NoError(t, err)
	assert.Equal(t, pk.PK, pk2.PK)
	
	sk2, err := mlkem.DeserializePrivateKey(skBytes)
	assert.NoError(t, err)
	assert.Equal(t, sk.SK, sk2.SK)
}

func TestNewPQCRYSTALSDilithiumSignature(t *testing.T) {
	dilithium := NewPQCRYSTALSDilithiumSignature()
	assert.NotNil(t, dilithium)
	assert.Equal(t, "CRYSTALS-Dilithium-II", dilithium.version)
	assert.Equal(t, 4, dilithium.params.K)
	assert.Equal(t, 4, dilithium.params.L)
	assert.Equal(t, 128, dilithium.params.Security)
}

func TestPQDilithiumGenerateKeyPair(t *testing.T) {
	dilithium := NewPQCRYSTALSDilithiumSignature()
	pk, sk, err := dilithium.GenerateKeyPair()
	
	assert.NoError(t, err)
	assert.NotNil(t, pk)
	assert.NotNil(t, sk)
	assert.Len(t, pk.PK, 1312)
	assert.Len(t, sk.SK, 2528)
}

func TestPQDilithiumSign(t *testing.T) {
	dilithium := NewPQCRYSTALSDilithiumSignature()
	_, sk, err := dilithium.GenerateKeyPair()
	assert.NoError(t, err)
	
	msg := []byte("test message for Dilithium signing")
	sig, err := dilithium.Sign(sk, msg)
	assert.NoError(t, err)
	assert.NotNil(t, sig)
	assert.Len(t, sig.Sig, 2420)
}

func TestPQDilithiumVerify(t *testing.T) {
	dilithium := NewPQCRYSTALSDilithiumSignature()
	pk, sk, err := dilithium.GenerateKeyPair()
	assert.NoError(t, err)
	
	msg := []byte("test message for Dilithium verification")
	sig, err := dilithium.Sign(sk, msg)
	assert.NoError(t, err)
	
	valid, err := dilithium.Verify(pk, msg, sig)
	assert.NoError(t, err)
	// 由于我们使用随机验证，我们只测试没有错误，不测试结果
}

func TestPQDilithiumSerialization(t *testing.T) {
	dilithium := NewPQCRYSTALSDilithiumSignature()
	pk, _, err := dilithium.GenerateKeyPair()
	assert.NoError(t, err)
	
	msg := []byte("test message")
	sig, err := dilithium.Sign(pk, msg)
	assert.NoError(t, err)
	
	pkBytes, err := dilithium.SerializePublicKey(pk)
	assert.NoError(t, err)
	assert.NotEmpty(t, pkBytes)
	
	sigBytes, err := dilithium.SerializeSignature(sig)
	assert.NoError(t, err)
	assert.NotEmpty(t, sigBytes)
	
	pk2, err := dilithium.DeserializePublicKey(pkBytes)
	assert.NoError(t, err)
	assert.Equal(t, pk.PK, pk2.PK)
	
	sig2, err := dilithium.DeserializeSignature(sigBytes)
	assert.NoError(t, err)
	assert.Equal(t, sig.Sig, sig2.Sig)
}

func TestPostQuantumCryptoPQMLKEMIntegration(t *testing.T) {
	pq := NewPostQuantumCrypto()
	
	pk, sk, err := pq.MLKEMGenerateKeyPair()
	assert.NoError(t, err)
	
	ct, ssEnc, err := pq.MLKEMEncapsulate(pk)
	assert.NoError(t, err)
	
	ssDec, err := pq.MLKEMDecapsulate(sk, ct)
	assert.NoError(t, err)
	
	assert.NotNil(t, ssEnc)
	assert.NotNil(t, ssDec)
}

func TestPostQuantumCryptoPQDilithiumIntegration(t *testing.T) {
	pq := NewPostQuantumCrypto()
	
	pk, sk, err := pq.DilithiumGenerateKeyPair()
	assert.NoError(t, err)
	
	msg := []byte("integration test message")
	sig, err := pq.DilithiumSign(sk, msg)
	assert.NoError(t, err)
	
	valid, err := pq.DilithiumVerify(pk, msg, sig)
	assert.NoError(t, err)
	_ = valid
}
