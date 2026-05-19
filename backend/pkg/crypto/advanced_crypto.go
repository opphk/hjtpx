package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
)

var (
	ErrChaCha20KeyLength = errors.New("ChaCha20 key must be 32 bytes")
	ErrInvalidNonceSize  = errors.New("invalid nonce size")
	ErrBlake2KeyLength   = errors.New("BLAKE2 key must be 0-64 bytes")
)

func ChaCha20Poly1305Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrChaCha20KeyLength
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func ChaCha20Poly1305Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrChaCha20KeyLength
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidNonceSize
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

func ChaCha20Poly1305EncryptWithAAD(plaintext, key, additionalData []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrChaCha20KeyLength
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, additionalData)
	return ciphertext, nil
}

func ChaCha20Poly1305DecryptWithAAD(ciphertext, key, additionalData []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrChaCha20KeyLength
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 AEAD: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidNonceSize
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var defaultArgon2Params = Argon2Params{
	Memory:      65536,
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

func Argon2Hash(password, salt []byte, params *Argon2Params) ([]byte, error) {
	if params == nil {
		params = &defaultArgon2Params
	}

	if len(salt) == 0 {
		var err error
		salt, err = GenerateRandomBytes(int(params.SaltLength))
		if err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	return argon2.IDKey(password, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength), nil
}

func Argon2HashString(password string, params *Argon2Params) (string, error) {
	if params == nil {
		params = &defaultArgon2Params
	}

	salt, err := GenerateRandomBytes(int(params.SaltLength))
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	return encoded, nil
}

func ScryptDeriveKey(password, salt []byte, params *ScryptParams) ([]byte, error) {
	if params == nil {
		params = &ScryptParams{
			N:       16384,
			R:       8,
			P:       1,
			KeyLen:  32,
			SaltLen: 16,
		}
	}

	if len(salt) == 0 {
		var err error
		salt, err = GenerateRandomBytes(params.SaltLen)
		if err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	return scrypt.Key(password, salt, params.N, params.R, params.P, params.KeyLen)
}

type ScryptParams struct {
	N       int
	R       int
	P       int
	KeyLen  int
	SaltLen int
}

func Blake2bHash(data []byte, key []byte) ([]byte, error) {
	if len(key) > 64 {
		return nil, ErrBlake2KeyLength
	}

	var h hash.Hash
	var err error

	if len(key) > 0 {
		h, err = blake2b.New256(key)
	} else {
		h, err = blake2b.New256(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create BLAKE2b hash: %w", err)
	}

	h.Write(data)
	return h.Sum(nil), nil
}

func Blake2bHashString(data string, key []byte) (string, error) {
	hash, err := Blake2bHash([]byte(data), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func Blake2b512Hash(data []byte, key []byte) ([]byte, error) {
	if len(key) > 64 {
		return nil, ErrBlake2KeyLength
	}

	var h hash.Hash
	var err error

	if len(key) > 0 {
		h, err = blake2b.New512(key)
	} else {
		h, err = blake2b.New512(nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create BLAKE2b-512 hash: %w", err)
	}

	h.Write(data)
	return h.Sum(nil), nil
}

func Blake2b512HashString(data string, key []byte) (string, error) {
	hash, err := Blake2b512Hash([]byte(data), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func AESWrapKey(wrappingKey, keyToWrap []byte) ([]byte, error) {
	if len(wrappingKey) != 16 && len(wrappingKey) != 24 && len(wrappingKey) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(wrappingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return wrapKey(block, keyToWrap)
}

func AESUnwrapKey(wrappingKey, wrappedKey []byte) ([]byte, error) {
	if len(wrappingKey) != 16 && len(wrappingKey) != 24 && len(wrappingKey) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(wrappingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return unwrapKey(block, wrappedKey)
}

func wrapKey(block cipher.Block, key []byte) ([]byte, error) {
	if len(key)%8 != 0 {
		return nil, errors.New("key must be a multiple of 8 bytes")
	}

	n := len(key) / 8
	if n < 2 {
		return nil, errors.New("key must be at least 16 bytes")
	}

	r := make([][]byte, n)
	for i := 0; i < n; i++ {
		r[i] = make([]byte, 8)
		copy(r[i], key[i*8:(i+1)*8])
	}

	a := make([]byte, 8)
	copy(a, []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6})

	for j := 0; j <= 5; j++ {
		for i := 1; i <= n; i++ {
			b := make([]byte, 16)
			copy(b, a)
			copy(b[8:], r[i-1])
			block.Encrypt(b, b)
			t := uint64((n*j + i) & 0xFFFFFFFF)
			for k := 7; k >= 0; k-- {
				b[k] ^= byte(t)
				t >>= 8
			}
			copy(a, b[:8])
			copy(r[i-1], b[8:])
		}
	}

	result := make([]byte, 8+n*8)
	copy(result, a)
	for i := 0; i < n; i++ {
		copy(result[8+i*8:], r[i])
	}

	return result, nil
}

func unwrapKey(block cipher.Block, wrapped []byte) ([]byte, error) {
	if len(wrapped)%8 != 0 || len(wrapped) < 24 {
		return nil, errors.New("invalid wrapped key length")
	}

	n := (len(wrapped) - 8) / 8

	a := make([]byte, 8)
	copy(a, wrapped[:8])

	r := make([][]byte, n)
	for i := 0; i < n; i++ {
		r[i] = make([]byte, 8)
		copy(r[i], wrapped[8+i*8:])
	}

	for j := 5; j >= 0; j-- {
		for i := n; i >= 1; i-- {
			t := uint64((n*j + i) & 0xFFFFFFFF)
			b := make([]byte, 16)
			copy(b, a)
			copy(b[8:], r[i-1])
			for k := 7; k >= 0; k-- {
				b[k] ^= byte(t)
				t >>= 8
			}
			block.Decrypt(b, b)
			copy(a, b[:8])
			copy(r[i-1], b[8:])
		}
	}

	expectedA := []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}
	for i := 0; i < 8; i++ {
		if a[i] != expectedA[i] {
			return nil, errors.New("invalid wrapped key")
		}
	}

	result := make([]byte, n*8)
	for i := 0; i < n; i++ {
		copy(result[i*8:], r[i])
	}

	return result, nil
}

type HKDFParams struct {
	Hash      HashAlgorithmType
	Salt      []byte
	Info      []byte
	KeyLength int
}

func HKDFExtract(salt, ikm []byte, hashType HashAlgorithmType) ([]byte, error) {
	var h func() hash.Hash
	switch hashType {
	case AlgoSHA256:
		h = sha256.New
	case AlgoSHA512:
		h = sha512.New
	default:
		h = sha256.New
	}

	if len(salt) == 0 {
		salt = make([]byte, h().Size())
	}

	mac := hmac.New(h, salt)
	mac.Write(ikm)
	return mac.Sum(nil), nil
}

func HKDFExpand(prk, info []byte, length int, hashType HashAlgorithmType) ([]byte, error) {
	var h func() hash.Hash
	switch hashType {
	case AlgoSHA256:
		h = sha256.New
	case AlgoSHA512:
		h = sha512.New
	default:
		h = sha256.New
	}

	hashLen := h().Size()
	blocks := (length + hashLen - 1) / hashLen

	if blocks > 255 {
		return nil, errors.New("HKDF expand length too large")
	}

	result := make([]byte, length)
	current := make([]byte, 0, hashLen)

	for i := 1; i <= blocks; i++ {
		current = append(current, info...)
		current = append(current, byte(i))
		mac := hmac.New(h, prk)
		mac.Write(current)
		block := mac.Sum(nil)
		copy(result[(i-1)*hashLen:min(i*hashLen, length)], block)
		current = block
	}

	return result, nil
}

func HKDF(ikm []byte, params *HKDFParams) ([]byte, error) {
	if params == nil {
		params = &HKDFParams{
			Hash:      AlgoSHA256,
			KeyLength: 32,
		}
	}

	prk, err := HKDFExtract(params.Salt, ikm, params.Hash)
	if err != nil {
		return nil, err
	}

	return HKDFExpand(prk, params.Info, params.KeyLength, params.Hash)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func EncryptThenMAC(plaintext, encryptionKey, macKey []byte, algorithm string) ([]byte, []byte, error) {
	var ciphertext []byte
	var err error

	switch algorithm {
	case "AES-GCM":
		ciphertext, err = AESEncrypt(plaintext, encryptionKey)
	case "ChaCha20-Poly1305":
		ciphertext, err = ChaCha20Poly1305Encrypt(plaintext, encryptionKey)
	default:
		ciphertext, err = AESEncrypt(plaintext, encryptionKey)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	mac, err := ComputeHMAC(macKey, ciphertext, AlgoSHA256)
	if err != nil {
		return nil, nil, fmt.Errorf("MAC computation failed: %w", err)
	}

	return ciphertext, mac, nil
}

func VerifyThenDecrypt(ciphertext, mac, encryptionKey, macKey []byte, algorithm string) ([]byte, error) {
	expectedMAC, err := ComputeHMAC(macKey, ciphertext, AlgoSHA256)
	if err != nil {
		return nil, fmt.Errorf("MAC computation failed: %w", err)
	}

	if !ConstantTimeCompareBytes(mac, expectedMAC) {
		return nil, errors.New("MAC verification failed")
	}

	var plaintext []byte
	switch algorithm {
	case "AES-GCM":
		plaintext, err = AESDecrypt(ciphertext, encryptionKey)
	case "ChaCha20-Poly1305":
		plaintext, err = ChaCha20Poly1305Decrypt(ciphertext, encryptionKey)
	default:
		plaintext, err = AESDecrypt(ciphertext, encryptionKey)
	}

	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

func GenerateSecureKeyPair() (string, string, error) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	privatePEM, err := ExportEd25519PrivateKeyToPEM(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to export private key: %w", err)
	}

	publicPEM, err := ExportEd25519PublicKeyToPEM(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to export public key: %w", err)
	}

	return privatePEM, publicPEM, nil
}

func EncryptWithMultipleKeys(plaintext []byte, keys [][]byte, algorithms []string) ([]byte, error) {
	if len(keys) != len(algorithms) {
		return nil, errors.New("number of keys must match number of algorithms")
	}

	current := plaintext
	var err error

	for i := 0; i < len(keys); i++ {
		switch algorithms[i] {
		case "AES-GCM":
			current, err = AESEncrypt(current, keys[i])
		case "ChaCha20-Poly1305":
			current, err = ChaCha20Poly1305Encrypt(current, keys[i])
		default:
			current, err = AESEncrypt(current, keys[i])
		}
		if err != nil {
			return nil, fmt.Errorf("layer %d encryption failed: %w", i+1, err)
		}
	}

	return current, nil
}

func DecryptWithMultipleKeys(ciphertext []byte, keys [][]byte, algorithms []string) ([]byte, error) {
	if len(keys) != len(algorithms) {
		return nil, errors.New("number of keys must match number of algorithms")
	}

	current := ciphertext
	var err error

	for i := len(keys) - 1; i >= 0; i-- {
		switch algorithms[i] {
		case "AES-GCM":
			current, err = AESDecrypt(current, keys[i])
		case "ChaCha20-Poly1305":
			current, err = ChaCha20Poly1305Decrypt(current, keys[i])
		default:
			current, err = AESDecrypt(current, keys[i])
		}
		if err != nil {
			return nil, fmt.Errorf("layer %d decryption failed: %w", i+1, err)
		}
	}

	return current, nil
}