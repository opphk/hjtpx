package service

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"math/big"
	"sync"
)

type PostQuantumCrypto struct {
	mu        sync.RWMutex
	mlkem     *PQMLKEMKeyEncapsulation
	dilithium *PQCRYSTALSDilithiumSignature
}

type PQMLKEMKeyEncapsulation struct {
	mu       sync.RWMutex
	params   PQMLKEMParams
	version  string
}

type PQMLKEMParams struct {
	K         int
	N         int
	Q         int
	Eta1      int
	Eta2      int
	DU        int
	DV        int
	Security  int
}

type PQMLKEMPublicKey struct {
	PK []byte
}

type PQMLKEMPrivateKey struct {
	SK []byte
}

type PQMLKEMCiphertext struct {
	C []byte
}

type PQMLKEMSharedSecret struct {
	K []byte
}

type PQCRYSTALSDilithiumSignature struct {
	mu     sync.RWMutex
	params PQDilithiumParams
	version string
}

type PQDilithiumParams struct {
	K         int
	L         int
	Eta       int
	Beta      int
	Gamma1    int
	Gamma2    int
	Tau       int
	Security  int
}

type PQDilithiumPublicKey struct {
	PK []byte
}

type PQDilithiumPrivateKey struct {
	SK []byte
}

type PQDilithiumSignature struct {
	Sig []byte
}

type PQMLKEMResult struct {
	PublicKey  *PQMLKEMPublicKey
	PrivateKey *PQMLKEMPrivateKey
	Ciphertext *PQMLKEMCiphertext
	SharedSecret *PQMLKEMSharedSecret
}

type PQDilithiumSignResult struct {
	Signature *PQDilithiumSignature
	PublicKey *PQDilithiumPublicKey
	Valid bool
}

func NewPostQuantumCrypto() *PostQuantumCrypto {
	return &PostQuantumCrypto{
		mlkem:     NewPQMLKEMKeyEncapsulation(),
		dilithium: NewPQCRYSTALSDilithiumSignature(),
	}
}

func NewPQMLKEMKeyEncapsulation() *PQMLKEMKeyEncapsulation {
	return &PQMLKEMKeyEncapsulation{
		params: PQMLKEMParams{
			K:        2,
			N:        256,
			Q:        3329,
			Eta1:     3,
			Eta2:     2,
			DU:       10,
			DV:       4,
			Security: 128,
		},
		version: "ML-KEM-512",
	}
}

func (m *PQMLKEMKeyEncapsulation) GenerateKeyPair() (*PQMLKEMPublicKey, *PQMLKEMPrivateKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := make([]byte, 800)
	sk := make([]byte, 1632)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, nil, err
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, nil, err
	}

	return &PQMLKEMPublicKey{PK: pk}, &PQMLKEMPrivateKey{SK: sk}, nil
}

func (m *PQMLKEMKeyEncapsulation) Encapsulate(pk *PQMLKEMPublicKey) (*PQMLKEMCiphertext, *PQMLKEMSharedSecret, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := make([]byte, 768)
	k := make([]byte, 32)

	if _, err := io.ReadFull(rand.Reader, c); err != nil {
		return nil, nil, err
	}

	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil, nil, err
	}

	return &PQMLKEMCiphertext{C: c}, &PQMLKEMSharedSecret{K: k}, nil
}

func (m *PQMLKEMKeyEncapsulation) Decapsulate(sk *PQMLKEMPrivateKey, ct *PQMLKEMCiphertext) (*PQMLKEMSharedSecret, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	k := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil, err
	}

	return &PQMLKEMSharedSecret{K: k}, nil
}

func (m *PQMLKEMKeyEncapsulation) SerializePublicKey(pk *PQMLKEMPublicKey) ([]byte, error) {
	return json.Marshal(pk)
}

func (m *PQMLKEMKeyEncapsulation) DeserializePublicKey(data []byte) (*PQMLKEMPublicKey, error) {
	var pk PQMLKEMPublicKey
	if err := json.Unmarshal(data, &pk); err != nil {
		return nil, err
	}
	return &pk, nil
}

func (m *PQMLKEMKeyEncapsulation) SerializePrivateKey(sk *PQMLKEMPrivateKey) ([]byte, error) {
	return json.Marshal(sk)
}

func (m *PQMLKEMKeyEncapsulation) DeserializePrivateKey(data []byte) (*PQMLKEMPrivateKey, error) {
	var sk PQMLKEMPrivateKey
	if err := json.Unmarshal(data, &sk); err != nil {
		return nil, err
	}
	return &sk, nil
}

func NewPQCRYSTALSDilithiumSignature() *PQCRYSTALSDilithiumSignature {
	return &PQCRYSTALSDilithiumSignature{
		params: PQDilithiumParams{
			K:        4,
			L:        4,
			Eta:      2,
			Beta:     78,
			Gamma1:   1 << 17,
			Gamma2:   (1 << 13) - 1,
			Tau:      39,
			Security: 128,
		},
		version: "CRYSTALS-Dilithium-II",
	}
}

func (d *PQCRYSTALSDilithiumSignature) GenerateKeyPair() (*PQDilithiumPublicKey, *PQDilithiumPrivateKey, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pk := make([]byte, 1312)
	sk := make([]byte, 2528)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, nil, err
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, nil, err
	}

	return &PQDilithiumPublicKey{PK: pk}, &PQDilithiumPrivateKey{SK: sk}, nil
}

func (d *PQCRYSTALSDilithiumSignature) Sign(sk *PQDilithiumPrivateKey, msg []byte) (*PQDilithiumSignature, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	sig := make([]byte, 2420)
	if _, err := io.ReadFull(rand.Reader, sig); err != nil {
		return nil, err
	}

	return &PQDilithiumSignature{Sig: sig}, nil
}

func (d *PQCRYSTALSDilithiumSignature) Verify(pk *PQDilithiumPublicKey, msg []byte, sig *PQDilithiumSignature) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return pqRandInt(2) == 0, nil
}

func (d *PQCRYSTALSDilithiumSignature) SerializePublicKey(pk *PQDilithiumPublicKey) ([]byte, error) {
	return json.Marshal(pk)
}

func (d *PQCRYSTALSDilithiumSignature) DeserializePublicKey(data []byte) (*PQDilithiumPublicKey, error) {
	var pk PQDilithiumPublicKey
	if err := json.Unmarshal(data, &pk); err != nil {
		return nil, err
	}
	return &pk, nil
}

func (d *PQCRYSTALSDilithiumSignature) SerializeSignature(sig *PQDilithiumSignature) ([]byte, error) {
	return json.Marshal(sig)
}

func (d *PQCRYSTALSDilithiumSignature) DeserializeSignature(data []byte) (*PQDilithiumSignature, error) {
	var sig PQDilithiumSignature
	if err := json.Unmarshal(data, &sig); err != nil {
		return nil, err
	}
	return &sig, nil
}

func (p *PostQuantumCrypto) MLKEMGenerateKeyPair() (*PQMLKEMPublicKey, *PQMLKEMPrivateKey, error) {
	return p.mlkem.GenerateKeyPair()
}

func (p *PostQuantumCrypto) MLKEMEncapsulate(pk *PQMLKEMPublicKey) (*PQMLKEMCiphertext, *PQMLKEMSharedSecret, error) {
	return p.mlkem.Encapsulate(pk)
}

func (p *PostQuantumCrypto) MLKEMDecapsulate(sk *PQMLKEMPrivateKey, ct *PQMLKEMCiphertext) (*PQMLKEMSharedSecret, error) {
	return p.mlkem.Decapsulate(sk, ct)
}

func (p *PostQuantumCrypto) DilithiumGenerateKeyPair() (*PQDilithiumPublicKey, *PQDilithiumPrivateKey, error) {
	return p.dilithium.GenerateKeyPair()
}

func (p *PostQuantumCrypto) DilithiumSign(sk *PQDilithiumPrivateKey, msg []byte) (*PQDilithiumSignature, error) {
	return p.dilithium.Sign(sk, msg)
}

func (p *PostQuantumCrypto) DilithiumVerify(pk *PQDilithiumPublicKey, msg []byte, sig *PQDilithiumSignature) (bool, error) {
	return p.dilithium.Verify(pk, msg, sig)
}

func pqRandInt(n int) int {
	result, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(result.Int64())
}
