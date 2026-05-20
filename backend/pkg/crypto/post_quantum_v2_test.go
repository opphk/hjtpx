package crypto

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostQuantumV2(t *testing.T) {
	pq := NewPostQuantumV2()
	assert.NotNil(t, pq)
	assert.NotNil(t, pq.kyberEngine)
	assert.NotNil(t, pq.dilithiumEngine)
	assert.NotNil(t, pq.hybridEngine)
	assert.NotNil(t, pq.keyStore)
	assert.NotNil(t, pq.keyManager)
	assert.NotNil(t, pq.protocolManager)
	assert.NotNil(t, pq.auditLogger)
}

func TestPostQuantumV2Initialize(t *testing.T) {
	pq := NewPostQuantumV2()
	ctx := context.Background()

	err := pq.Initialize(ctx)
	require.NoError(t, err)
	assert.True(t, pq.initialized)
}

func TestPostQuantumV2GenerateKyberKeyPair(t *testing.T) {
	pq := NewPostQuantumV2()

	testCases := []struct {
		name      string
		algorithm PQV2Algorithm
		wantErr   bool
	}{
		{
			name:      "Kyber512",
			algorithm: PQV2Kyber512,
			wantErr:   false,
		},
		{
			name:      "Kyber768",
			algorithm: PQV2Kyber768,
			wantErr:   false,
		},
		{
			name:      "Kyber1024",
			algorithm: PQV2Kyber1024,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyPair, err := pq.GenerateKyberKeyPairV2(tc.algorithm)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, keyPair)
			assert.NotEmpty(t, keyPair.PublicKey)
			assert.NotEmpty(t, keyPair.PrivateKey)
			assert.Equal(t, tc.algorithm, keyPair.Algorithm)
			assert.NotEmpty(t, keyPair.KeyID)
			assert.False(t, keyPair.CreatedAt.IsZero())
			assert.False(t, keyPair.ExpiresAt.IsZero())
		})
	}
}

func TestPostQuantumV2GenerateDilithiumKeyPair(t *testing.T) {
	pq := NewPostQuantumV2()

	testCases := []struct {
		name      string
		algorithm PQV2Algorithm
		wantErr   bool
	}{
		{
			name:      "Dilithium2",
			algorithm: PQV2Dilithium2,
			wantErr:   false,
		},
		{
			name:      "Dilithium3",
			algorithm: PQV2Dilithium3,
			wantErr:   false,
		},
		{
			name:      "Dilithium5",
			algorithm: PQV2Dilithium5,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyPair, err := pq.GenerateDilithiumKeyPairV2(tc.algorithm)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, keyPair)
			assert.NotEmpty(t, keyPair.PublicKey)
			assert.NotEmpty(t, keyPair.PrivateKey)
			assert.Equal(t, tc.algorithm, keyPair.Algorithm)
			assert.NotEmpty(t, keyPair.KeyID)
		})
	}
}

func TestPostQuantumV2KyberEncapsulateDecapsulate(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	ciphertext, sharedSecret, err := pq.KyberEncapsulateV2(keyPair.PublicKey, PQV2Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, ciphertext)
	assert.NotNil(t, sharedSecret)
	assert.NotEmpty(t, ciphertext.Data)
	assert.NotEmpty(t, sharedSecret.Data)

	decryptedSecret, err := pq.KyberDecapsulateV2(ciphertext.Data, keyPair.PrivateKey, PQV2Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, decryptedSecret)
}

func TestPostQuantumV2DilithiumSignVerify(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateDilithiumKeyPairV2(PQV2Dilithium3)
	require.NoError(t, err)

	message := []byte("test message for post-quantum v2 signature")

	signature, err := pq.DilithiumSignV2(message, keyPair.PrivateKey, PQV2Dilithium3)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, signature.Data)
	assert.Equal(t, PQV2Dilithium3, signature.Algorithm)

	valid, err := pq.DilithiumVerifyV2(message, signature, keyPair.PublicKey, PQV2Dilithium3)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPostQuantumV2HybridEncryptDecrypt(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	plaintext := []byte("test message for hybrid encryption")

	encryptedData, err := pq.HybridEncryptV2(plaintext, keyPair.PublicKey, PQV2Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, encryptedData)
	assert.NotEmpty(t, encryptedData.Ciphertext)
	assert.NotEmpty(t, encryptedData.EncryptedKey)
	assert.NotEmpty(t, encryptedData.IV)
	assert.Equal(t, PQV2Kyber768, encryptedData.Algorithm)

	decrypted, err := pq.HybridDecryptV2(encryptedData, keyPair.PrivateKey, PQV2Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, decrypted)
}

func TestPostQuantumV2Handshake(t *testing.T) {
	pq := NewPostQuantumV2()

	clientKP, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	result, err := pq.PerformHandshakeV2(clientKP.PublicKey, PQV2Kyber768)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SessionKey)
	assert.NotNil(t, result.SharedSecret)
	assert.NotNil(t, result.Ciphertext)
	assert.NotEmpty(t, result.PublicKey)
	assert.Equal(t, 2, result.ProtocolVersion.Major)
}

func TestPostQuantumV2DeriveKey(t *testing.T) {
	pq := NewPostQuantumV2()

	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	derivedKey, err := pq.DeriveKeyV2(secret, "test-purpose", 32)
	require.NoError(t, err)
	assert.NotNil(t, derivedKey)
	assert.Len(t, derivedKey, 32)

	derivedKey2, err := pq.DeriveKeyV2(secret, "test-purpose", 32)
	require.NoError(t, err)

	assert.Equal(t, derivedKey, derivedKey2)

	derivedKey3, err := pq.DeriveKeyV2(secret, "different-purpose", 32)
	require.NoError(t, err)
	assert.NotEqual(t, derivedKey, derivedKey3)
}

func TestPostQuantumV2SerializeDeserialize(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	serialized, err := pq.SerializeKeyPairV2(keyPair)
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	deserialized, err := pq.DeserializeKeyPairV2(serialized)
	require.NoError(t, err)
	assert.NotNil(t, deserialized)
	assert.Equal(t, keyPair.Algorithm, deserialized.Algorithm)
	assert.Equal(t, keyPair.KeyID, deserialized.KeyID)
	assert.Equal(t, keyPair.PublicKey, deserialized.PublicKey)
	assert.Equal(t, keyPair.PrivateKey, deserialized.PrivateKey)
}

func TestPostQuantumV2KeyStore(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	metadata, err := pq.GetKeyInfoV2(context.Background(), keyPair.KeyID)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, keyPair.KeyID, metadata.KeyID)
	assert.Equal(t, PQV2Kyber768, metadata.Algorithm)
	assert.Equal(t, "active", metadata.Status)
}

func TestPostQuantumV2RotateKey(t *testing.T) {
	pq := NewPostQuantumV2()

	store := pq.keyManager.CreateKeyStore("test-namespace")

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	store.StoreKey(keyPair.KeyID, keyPair.PrivateKey, &PQV2KeyMetadata{
		KeyID:         keyPair.KeyID,
		Algorithm:     keyPair.Algorithm,
		CreatedAt:     keyPair.CreatedAt,
		ExpiresAt:     keyPair.ExpiresAt,
		UsageCount:    0,
		MaxUsageCount: 100,
		Status:        "active",
	})

	rotation, err := pq.keyManager.RotateKey("test-namespace", keyPair.KeyID)
	require.NoError(t, err)
	assert.NotNil(t, rotation)
	assert.Equal(t, keyPair.KeyID, rotation.OldKeyID)
	assert.NotEmpty(t, rotation.NewKeyID)
}

func TestPostQuantumV2ProtocolVersion(t *testing.T) {
	pq := NewPostQuantumV2()

	version := pq.GetVersionV2()
	assert.Equal(t, 2, version.Major)
	assert.Equal(t, 0, version.Minor)
	assert.Equal(t, 1, version.Patch)
	assert.Equal(t, "v2.0.1", version.String())
}

func TestPostQuantumV2SecurityLevel(t *testing.T) {
	pq := NewPostQuantumV2()

	level := pq.GetSecurityLevelV2()
	assert.Equal(t, PQV2Security192, level)
}

func TestPostQuantumV2EncryptDecryptAPI(t *testing.T) {
	pq := NewPostQuantumV2()
	ctx := context.Background()

	keyPair, err := pq.GenerateKyberKeyPairV2(PQV2Kyber768)
	require.NoError(t, err)

	plaintext := []byte("test message for API encryption")

	encReq := &PQV2EncryptionRequest{
		Plaintext:    plaintext,
		PublicKey:    keyPair.PublicKey,
		Algorithm:    PQV2Kyber768,
		HybridScheme: "kyber-classic",
	}

	encResp, err := pq.EncryptV2(ctx, encReq)
	require.NoError(t, err)
	assert.True(t, encResp.Success)
	assert.NotNil(t, encResp.EncryptedData)
}

func TestPostQuantumV2AuditLogger(t *testing.T) {
	pq := NewPostQuantumV2()

	pq.auditLogger.Log("encrypt", "test-key-id", "client-1", "127.0.0.1", true, "")

	logs := pq.auditLogger.GetLogs("test-key-id")
	assert.Len(t, logs, 1)
	assert.Equal(t, "encrypt", logs[0].Operation)
	assert.True(t, logs[0].Success)
}

func TestPQV2KeyStoreStoreAndGet(t *testing.T) {
	store := NewPQV2KeyStore()

	metadata := &PQV2KeyMetadata{
		KeyID:         "test-key",
		Algorithm:     PQV2Kyber768,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		UsageCount:    0,
		MaxUsageCount: 100,
		Status:        "active",
	}

	keyData := []byte("test-key-data")

	err := store.StoreKey("test-key", keyData, metadata)
	require.NoError(t, err)

	retrievedData, retrievedMetadata, err := store.GetKey("test-key")
	require.NoError(t, err)
	assert.Equal(t, keyData, retrievedData)
	assert.Equal(t, "test-key", retrievedMetadata.KeyID)
}

func TestPQV2KeyStoreExpired(t *testing.T) {
	store := NewPQV2KeyStore()

	metadata := &PQV2KeyMetadata{
		KeyID:         "expired-key",
		Algorithm:     PQV2Kyber768,
		CreatedAt:     time.Now().Add(-48 * time.Hour),
		ExpiresAt:     time.Now().Add(-24 * time.Hour),
		UsageCount:    0,
		MaxUsageCount: 100,
		Status:        "active",
	}

	store.StoreKey("expired-key", []byte("test"), metadata)

	_, _, err := store.GetKey("expired-key")
	assert.Error(t, err)
	assert.Equal(t, ErrPQV2KeyExpired, err)
}

func TestPQV2Session(t *testing.T) {
	pm := NewPQV2ProtocolManager()

	session := pm.CreateSession("test-session", []byte("shared-secret"))
	assert.NotNil(t, session)
	assert.Equal(t, "test-session", session.SessionID)
	assert.NotEmpty(t, session.SharedSecret)

	retrieved, err := pm.GetSession("test-session")
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, retrieved.SessionID)
}

func TestPQV2SignatureVerifyFail(t *testing.T) {
	pq := NewPostQuantumV2()

	keyPair, err := pq.GenerateDilithiumKeyPairV2(PQV2Dilithium2)
	require.NoError(t, err)

	message := []byte("test message")
	signature := &PQV2Signature{
		Data:        []byte{},
		Algorithm:   PQV2Dilithium2,
		PublicKey:   keyPair.PublicKey,
		SigningTime: time.Now(),
	}

	valid, err := pq.DilithiumVerifyV2(message, signature, keyPair.PublicKey, PQV2Dilithium2)
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestPQV2GenerateUnsupportedAlgorithm(t *testing.T) {
	pq := NewPostQuantumV2()

	_, err := pq.GenerateKyberKeyPairV2(PQV2Algorithm("unsupported"))
	assert.Error(t, err)

	_, err = pq.GenerateDilithiumKeyPairV2(PQV2Algorithm("unsupported"))
	assert.Error(t, err)
}

func TestPQV2EncryptedDataSerialization(t *testing.T) {
	encryptedData := &PQV2EncryptedData{
		Ciphertext:   []byte("ciphertext"),
		EncryptedKey: []byte("encrypted-key"),
		IV:           []byte("iv"),
		AuthTag:      []byte("auth-tag"),
		Algorithm:    PQV2Kyber768,
		HybridScheme: "kyber-classic",
		KeyVersion:   2,
	}

	data, err := json.Marshal(encryptedData)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded PQV2EncryptedData
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, encryptedData.Algorithm, decoded.Algorithm)
	assert.Equal(t, encryptedData.HybridScheme, decoded.HybridScheme)
}
