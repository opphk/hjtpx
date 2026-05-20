package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostQuantumCryptoV2(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()
	assert.NotNil(t, pqc)
}

func TestGenerateKyberKeyPair(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	testCases := []struct {
		name      string
		algorithm PQAlgorithm
		wantErr   bool
	}{
		{
			name:      "Kyber512",
			algorithm: Kyber512,
			wantErr:   false,
		},
		{
			name:      "Kyber768",
			algorithm: Kyber768,
			wantErr:   false,
		},
		{
			name:      "Kyber1024",
			algorithm: Kyber1024,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyPair, err := pqc.GenerateKyberKeyPair(tc.algorithm)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, keyPair)
			assert.NotEmpty(t, keyPair.PublicKey)
			assert.NotEmpty(t, keyPair.PrivateKey)
			assert.Equal(t, tc.algorithm, keyPair.Algorithm)
			assert.False(t, keyPair.CreatedAt.IsZero())
		})
	}
}

func TestGenerateDilithiumKeyPair(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	testCases := []struct {
		name      string
		algorithm PQAlgorithm
		wantErr   bool
	}{
		{
			name:      "Dilithium2",
			algorithm: Dilithium2,
			wantErr:   false,
		},
		{
			name:      "Dilithium3",
			algorithm: Dilithium3,
			wantErr:   false,
		},
		{
			name:      "Dilithium5",
			algorithm: Dilithium5,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyPair, err := pqc.GenerateDilithiumKeyPair(tc.algorithm)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, keyPair)
			assert.NotEmpty(t, keyPair.PublicKey)
			assert.NotEmpty(t, keyPair.PrivateKey)
			assert.Equal(t, tc.algorithm, keyPair.Algorithm)
			assert.False(t, keyPair.CreatedAt.IsZero())
		})
	}
}

func TestKyberEncapsulateAndDecapsulate(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	keyPair, err := pqc.GenerateKyberKeyPair(Kyber768)
	require.NoError(t, err)

	ct, ss1, err := pqc.KyberEncapsulate(keyPair.PublicKey, Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, ct)
	assert.NotNil(t, ss1)

	ss2, err := pqc.KyberDecapsulate(ct.Data, keyPair.PrivateKey, Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, ss2)
}

func TestDilithiumSignAndVerify(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	keyPair, err := pqc.GenerateDilithiumKeyPair(Dilithium3)
	require.NoError(t, err)

	message := []byte("test message for signing")

	signature, err := pqc.DilithiumSign(message, keyPair.PrivateKey, Dilithium3)
	require.NoError(t, err)
	assert.NotNil(t, signature)

	valid, err := pqc.DilithiumVerify(message, signature.Data, keyPair.PublicKey, Dilithium3)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestKyberKeySerialization(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	keyPair, err := pqc.GenerateKyberKeyPair(Kyber768)
	require.NoError(t, err)

	serializedPub, err := pqc.SerializeKyberPublicKey(keyPair)
	require.NoError(t, err)
	assert.NotEmpty(t, serializedPub)

	deserializedPub, err := pqc.DeserializeKyberPublicKey(serializedPub)
	require.NoError(t, err)
	assert.NotNil(t, deserializedPub)
	assert.Equal(t, keyPair.Algorithm, deserializedPub.Algorithm)
	assert.Equal(t, keyPair.PublicKey, deserializedPub.PublicKey)

	serializedPriv, err := pqc.SerializeKyberPrivateKey(keyPair)
	require.NoError(t, err)
	assert.NotEmpty(t, serializedPriv)

	deserializedPriv, err := pqc.DeserializeKyberPrivateKey(serializedPriv)
	require.NoError(t, err)
	assert.NotNil(t, deserializedPriv)
	assert.Equal(t, keyPair.Algorithm, deserializedPriv.Algorithm)
	assert.Equal(t, keyPair.PrivateKey, deserializedPriv.PrivateKey)
}

func TestDilithiumKeySerialization(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	keyPair, err := pqc.GenerateDilithiumKeyPair(Dilithium3)
	require.NoError(t, err)

	serializedPub, err := pqc.SerializeDilithiumPublicKey(keyPair)
	require.NoError(t, err)
	assert.NotEmpty(t, serializedPub)

	deserializedPub, err := pqc.DeserializeDilithiumPublicKey(serializedPub)
	require.NoError(t, err)
	assert.NotNil(t, deserializedPub)
	assert.Equal(t, keyPair.Algorithm, deserializedPub.Algorithm)
	assert.Equal(t, keyPair.PublicKey, deserializedPub.PublicKey)

	serializedPriv, err := pqc.SerializeDilithiumPrivateKey(keyPair)
	require.NoError(t, err)
	assert.NotEmpty(t, serializedPriv)

	deserializedPriv, err := pqc.DeserializeDilithiumPrivateKey(serializedPriv)
	require.NoError(t, err)
	assert.NotNil(t, deserializedPriv)
	assert.Equal(t, keyPair.Algorithm, deserializedPriv.Algorithm)
	assert.Equal(t, keyPair.PrivateKey, deserializedPriv.PrivateKey)
}

func TestEncryptWithKyber(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	keyPair, err := pqc.GenerateKyberKeyPair(Kyber768)
	require.NoError(t, err)

	plaintext := []byte("secret message to encrypt with post-quantum crypto")

	ciphertext, err := pqc.EncryptWithKyber(plaintext, keyPair.PublicKey, Kyber768)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := pqc.DecryptWithKyber(ciphertext, keyPair.PrivateKey, Kyber768)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestSignAndVerify(t *testing.T) {
	pqc := NewPostQuantumCryptoV2()

	sk, err := pqc.GenerateDilithiumKeyPair(Dilithium3)
	require.NoError(t, err)

	pk, err := pqc.GenerateDilithiumKeyPair(Dilithium3)
	require.NoError(t, err)

	message := []byte("important message to sign and verify")

	valid, err := pqc.SignAndVerify(message, sk, pk)
	require.NoError(t, err)
	assert.True(t, valid)
}

func BenchmarkGenerateKyberKeyPair(b *testing.B) {
	pqc := NewPostQuantumCryptoV2()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pqc.GenerateKyberKeyPair(Kyber768)
	}
}

func BenchmarkGenerateDilithiumKeyPair(b *testing.B) {
	pqc := NewPostQuantumCryptoV2()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pqc.GenerateDilithiumKeyPair(Dilithium3)
	}
}

func BenchmarkKyberEncryptDecrypt(b *testing.B) {
	pqc := NewPostQuantumCryptoV2()
	keyPair, _ := pqc.GenerateKyberKeyPair(Kyber768)
	plaintext := []byte("benchmark test message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := pqc.EncryptWithKyber(plaintext, keyPair.PublicKey, Kyber768)
		_, _ = pqc.DecryptWithKyber(ciphertext, keyPair.PrivateKey, Kyber768)
	}
}

func BenchmarkDilithiumSignVerify(b *testing.B) {
	pqc := NewPostQuantumCryptoV2()
	keyPair, _ := pqc.GenerateDilithiumKeyPair(Dilithium3)
	message := []byte("benchmark test message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signature, _ := pqc.DilithiumSign(message, keyPair.PrivateKey, Dilithium3)
		_, _ = pqc.DilithiumVerify(message, signature.Data, keyPair.PublicKey, Dilithium3)
	}
}
