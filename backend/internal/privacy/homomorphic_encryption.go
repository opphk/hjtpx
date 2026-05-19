package privacy

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"sync"
)

type HomomorphicEncryptionScheme int

const (
	PaillierScheme HomomorphicEncryptionScheme = iota
	ElGamalScheme
	BFVScheme
	RNGScheme
)

type HEKeyPair struct {
	PublicKey  *HEPublicKey
	PrivateKey *HEPrivateKey
}

type HEPublicKey struct {
	N       *big.Int
	G       *big.Int
	Lambda  *big.Int
	Mu      *big.Int
	Scheme  HomomorphicEncryptionScheme
	BitSize int
}

type HEPrivateKey struct {
	Lambda *big.Int
	Mu     *big.Int
	Phi    *big.Int
}

type HECiphertext struct {
	C1 *big.Int
	C2 *big.Int
}

type HEPlaintext struct {
	Value *big.Int
}

type HomomorphicEncryption struct {
	scheme  HomomorphicEncryptionScheme
	keyPair *HEKeyPair
	mu      sync.RWMutex
}

func NewHomomorphicEncryption(scheme HomomorphicEncryptionScheme, bitSize int) (*HomomorphicEncryption, error) {
	he := &HomomorphicEncryption{
		scheme: scheme,
	}

	var err error
	switch scheme {
	case PaillierScheme:
		he.keyPair, err = he.generatePaillierKeys(bitSize)
	case ElGamalScheme:
		he.keyPair, err = he.generateElGamalKeys(bitSize)
	default:
		he.keyPair, err = he.generatePaillierKeys(bitSize)
	}

	if err != nil {
		return nil, err
	}

	return he, nil
}

func (he *HomomorphicEncryption) generatePaillierKeys(bitSize int) (*HEKeyPair, error) {
	p, err := rand.Prime(rand.Reader, bitSize/2)
	if err != nil {
		return nil, err
	}
	q, err := rand.Prime(rand.Reader, bitSize/2)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).Mul(p, q)

	phi := new(big.Int).Mul(new(big.Int).Sub(p, big.NewInt(1)), new(big.Int).Sub(q, big.NewInt(1)))

	one := big.NewInt(1)
	g := new(big.Int).Add(n, one)

	nSquared := new(big.Int).Mul(n, n)

	lVal := new(big.Int)
	lVal.Exp(g, phi, nSquared)
	lVal.Sub(lVal, one)
	lVal.Div(lVal, n)

	mu := new(big.Int)
	mu.ModInverse(lVal, n)

	return &HEKeyPair{
		PublicKey: &HEPublicKey{
			N:       n,
			G:       g,
			Lambda:  phi,
			Mu:      mu,
			Scheme:  PaillierScheme,
			BitSize: bitSize,
		},
		PrivateKey: &HEPrivateKey{
			Lambda: phi,
			Mu:     mu,
			Phi:    phi,
		},
	}, nil
}

func (he *HomomorphicEncryption) generateElGamalKeys(bitSize int) (*HEKeyPair, error) {
	p, err := rand.Prime(rand.Reader, bitSize/2)
	if err != nil {
		return nil, err
	}

	g := big.NewInt(2)

	privateKey := new(big.Int)
	privateKeyBytes := make([]byte, bitSize/8)
	rand.Read(privateKeyBytes)
	privateKey.SetBytes(privateKeyBytes)
	privateKey.Mod(privateKey, p)

	y := new(big.Int)
	y.Exp(g, privateKey, p)

	return &HEKeyPair{
		PublicKey: &HEPublicKey{
			N:       p,
			G:       g,
			Scheme:  ElGamalScheme,
			BitSize: bitSize,
		},
		PrivateKey: &HEPrivateKey{
			Lambda: privateKey,
		},
	}, nil
}

func (he *HomomorphicEncryption) Encrypt(plaintext []byte) (*HECiphertext, error) {
	he.mu.RLock()
	defer he.mu.RUnlock()

	switch he.scheme {
	case PaillierScheme:
		return he.paillierEncrypt(plaintext)
	default:
		return he.paillierEncrypt(plaintext)
	}
}

func (he *HomomorphicEncryption) paillierEncrypt(plaintext []byte) (*HECiphertext, error) {
	m := new(big.Int)
	m.SetBytes(plaintext)

	n := he.keyPair.PublicKey.N
	nSquared := new(big.Int).Mul(n, n)

	r := new(big.Int)
	rBytes := make([]byte, n.BitLen()/8+1)
	rand.Read(rBytes)
	r.SetBytes(rBytes)
	r.Mod(r, n)
	if r.Sign() <= 0 {
		r.Add(r, big.NewInt(1))
	}

	g := he.keyPair.PublicKey.G

	gm := new(big.Int)
	gm.Exp(g, m, nSquared)

	rn := new(big.Int)
	rn.Exp(r, n, nSquared)

	c := new(big.Int)
	c.Mul(gm, rn)
	c.Mod(c, nSquared)

	return &HECiphertext{
		C1: c,
		C2: r,
	}, nil
}

func (he *HomomorphicEncryption) Decrypt(ciphertext *HECiphertext) ([]byte, error) {
	he.mu.RLock()
	defer he.mu.RUnlock()

	switch he.scheme {
	case PaillierScheme:
		return he.paillierDecrypt(ciphertext)
	default:
		return he.paillierDecrypt(ciphertext)
	}
}

func (he *HomomorphicEncryption) paillierDecrypt(ciphertext *HECiphertext) ([]byte, error) {
	n := he.keyPair.PublicKey.N
	lambda := he.keyPair.PrivateKey.Lambda
	mu := he.keyPair.PrivateKey.Mu

	nSquared := new(big.Int).Mul(n, n)

	one := big.NewInt(1)

	l := new(big.Int)
	l.Exp(ciphertext.C1, lambda, nSquared)
	l.Sub(l, one)
	l.Div(l, n)

	m := new(big.Int)
	m.Mul(l, mu)
	m.Mod(m, n)

	return m.Bytes(), nil
}

func (he *HomomorphicEncryption) HomomorphicAdd(c1, c2 *HECiphertext) *HECiphertext {
	n := he.keyPair.PublicKey.N
	nSquared := new(big.Int).Mul(n, n)

	result := new(big.Int)
	result.Mul(c1.C1, c2.C1)
	result.Mod(result, nSquared)

	return &HECiphertext{
		C1: result,
		C2: new(big.Int),
	}
}

func (he *HomomorphicEncryption) HomomorphicAddScalar(c *HECiphertext, scalar int64) *HECiphertext {
	n := he.keyPair.PublicKey.N
	nSquared := new(big.Int).Mul(n, n)

	gScalar := new(big.Int)
	gScalar.Exp(big.NewInt(scalar), big.NewInt(1), nSquared)

	result := new(big.Int)
	result.Mul(c.C1, gScalar)
	result.Mod(result, nSquared)

	return &HECiphertext{
		C1: result,
		C2: c.C2,
	}
}

func (he *HomomorphicEncryption) HomomorphicMultiply(c *HECiphertext, scalar int64) *HECiphertext {
	nSquared := new(big.Int).Mul(he.keyPair.PublicKey.N, he.keyPair.PublicKey.N)

	result := new(big.Int)
	result.Exp(c.C1, big.NewInt(scalar), nSquared)

	return &HECiphertext{
		C1: result,
		C2: c.C2,
	}
}

func (he *HomomorphicEncryption) GetPublicKey() *HEPublicKey {
	he.mu.RLock()
	defer he.mu.RUnlock()
	return he.keyPair.PublicKey
}

func (he *HomomorphicEncryption) GetScheme() HomomorphicEncryptionScheme {
	he.mu.RLock()
	defer he.mu.RUnlock()
	return he.scheme
}

type PartialHomomorphicEncryption struct {
	keyPair *HEKeyPair
	mu      sync.RWMutex
}

func NewPartialHE(bitSize int) (*PartialHomomorphicEncryption, error) {
	p, err := rand.Prime(rand.Reader, bitSize/2)
	if err != nil {
		return nil, err
	}
	q, err := rand.Prime(rand.Reader, bitSize/2)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).Mul(p, q)
	g := new(big.Int).Add(n, big.NewInt(1))

	return &PartialHomomorphicEncryption{
		keyPair: &HEKeyPair{
			PublicKey: &HEPublicKey{
				N:       n,
				G:       g,
				Scheme:  PaillierScheme,
				BitSize: bitSize,
			},
		},
	}, nil
}

func (phe *PartialHomomorphicEncryption) Encrypt(m int64) (*HECiphertext, error) {
	phe.mu.RLock()
	defer phe.mu.RUnlock()

	n := phe.keyPair.PublicKey.N
	nSquared := new(big.Int).Mul(n, n)

	r := new(big.Int)
	rBytes := make([]byte, n.BitLen()/8+1)
	rand.Read(rBytes)
	r.SetBytes(rBytes)
	r.Mod(r, n)
	if r.Sign() <= 0 {
		r.Add(r, big.NewInt(1))
	}

	g := phe.keyPair.PublicKey.G

	gm := new(big.Int)
	gm.Exp(g, big.NewInt(m), nSquared)

	rn := new(big.Int)
	rn.Exp(r, n, nSquared)

	c := new(big.Int)
	c.Mul(gm, rn)
	c.Mod(c, nSquared)

	return &HECiphertext{
		C1: c,
		C2: r,
	}, nil
}

func (phe *PartialHomomorphicEncryption) Add(ct1, ct2 *HECiphertext) *HECiphertext {
	phe.mu.RLock()
	defer phe.mu.RUnlock()

	n := phe.keyPair.PublicKey.N
	nSquared := new(big.Int).Mul(n, n)

	result := new(big.Int)
	result.Mul(ct1.C1, ct2.C1)
	result.Mod(result, nSquared)

	return &HECiphertext{
		C1: result,
	}
}

func (phe *PartialHomomorphicEncryption) Multiply(ct *HECiphertext, scalar int64) *HECiphertext {
	phe.mu.RLock()
	defer phe.mu.RUnlock()

	nSquared := new(big.Int).Mul(phe.keyPair.PublicKey.N, phe.keyPair.PublicKey.N)

	result := new(big.Int)
	result.Exp(ct.C1, big.NewInt(scalar), nSquared)

	return &HECiphertext{
		C1: result,
	}
}

func (phe *PartialHomomorphicEncryption) Decrypt(ct *HECiphertext) (int64, error) {
	phe.mu.RLock()
	defer phe.mu.RUnlock()

	n := phe.keyPair.PublicKey.N

	m := new(big.Int)
	m.Mod(ct.C1, n)

	return m.Int64(), nil
}

func (phe *PartialHomomorphicEncryption) EncryptVector(values []int64) ([]*HECiphertext, error) {
	ciphertexts := make([]*HECiphertext, len(values))
	for i, v := range values {
		ct, err := phe.Encrypt(v)
		if err != nil {
			return nil, err
		}
		ciphertexts[i] = ct
	}
	return ciphertexts, nil
}

func (phe *PartialHomomorphicEncryption) SumVector(ciphertexts []*HECiphertext) *HECiphertext {
	if len(ciphertexts) == 0 {
		return nil
	}

	result := ciphertexts[0]
	for i := 1; i < len(ciphertexts); i++ {
		result = phe.Add(result, ciphertexts[i])
	}
	return result
}

func (phe *PartialHomomorphicEncryption) InnerProduct(vec1, vec2 []*HECiphertext) (*HECiphertext, error) {
	if len(vec1) != len(vec2) {
		return nil, fmt.Errorf("vectors must have the same length")
	}

	if len(vec1) == 0 {
		return nil, fmt.Errorf("vectors cannot be empty")
	}

	products := make([]*HECiphertext, len(vec1))
	for i := range vec1 {
		ct1 := vec1[i].C1.Int64()
		ct2 := vec2[i]

		product := phe.Multiply(ct2, ct1)
		products[i] = product
	}

	return phe.SumVector(products), nil
}

func (he *HomomorphicEncryption) DemonstrateUsage() error {
	fmt.Println("=== 同态加密演示 ===")

	m1 := int64(10)
	m2 := int64(20)

	ct1, err := he.Encrypt([]byte(fmt.Sprintf("%d", m1)))
	if err != nil {
		return err
	}

	ct2, err := he.Encrypt([]byte(fmt.Sprintf("%d", m2)))
	if err != nil {
		return err
	}

	ctSum := he.HomomorphicAdd(ct1, ct2)

	result, err := he.Decrypt(ctSum)
	if err != nil {
		return err
	}

	fmt.Printf("输入: %d + %d\n", m1, m2)
	fmt.Printf("加密后相加，再解密结果: %s\n", string(result))

	return nil
}

type HETest struct {
	he *HomomorphicEncryption
}

func NewHETest() *HETest {
	he, _ := NewHomomorphicEncryption(PaillierScheme, 2048)
	return &HETest{he: he}
}

func (t *HETest) TestAddition() error {
	m1 := int64(100)
	m2 := int64(200)

	ct1, err := t.he.Encrypt([]byte(fmt.Sprintf("%d", m1)))
	if err != nil {
		return err
	}

	ct2, err := t.he.Encrypt([]byte(fmt.Sprintf("%d", m2)))
	if err != nil {
		return err
	}

	ctSum := t.he.HomomorphicAdd(ct1, ct2)

	result, err := t.he.Decrypt(ctSum)
	if err != nil {
		return err
	}

	var resultVal int64
	fmt.Sscanf(string(result), "%d", &resultVal)

	if resultVal != m1+m2 {
		return fmt.Errorf("addition test failed: expected %d, got %d", m1+m2, resultVal)
	}

	return nil
}

func (t *HETest) TestScalarMultiplication() error {
	m := int64(50)
	scalar := int64(3)

	ct, err := t.he.Encrypt([]byte(fmt.Sprintf("%d", m)))
	if err != nil {
		return err
	}

	ctMult := t.he.HomomorphicMultiply(ct, scalar)

	result, err := t.he.Decrypt(ctMult)
	if err != nil {
		return err
	}

	var resultVal int64
	fmt.Sscanf(string(result), "%d", &resultVal)

	if resultVal != m*scalar {
		return fmt.Errorf("scalar multiplication test failed: expected %d, got %d", m*scalar, resultVal)
	}

	return nil
}

func (t *HETest) RunAllTests() []error {
	errors := make([]error, 0)

	if err := t.TestAddition(); err != nil {
		errors = append(errors, err)
	}

	if err := t.TestScalarMultiplication(); err != nil {
		errors = append(errors, err)
	}

	return errors
}

type SEALLikeScheme struct {
	polyModulusDegree int
	coeffModBits      []int
	plainModulus      int64
	parms             SEALSparams
}

type SEALSparams struct {
	PolyModulusDegree int
	CoeffModBits      []int
	PlainModulus      int64
}

func NewSEALLikeScheme(polyModDegree int, plainMod int64) *SEALLikeScheme {
	return &SEALLikeScheme{
		polyModulusDegree: polyModDegree,
		plainModulus:      plainMod,
		coeffModBits:      []int{60, 40, 40, 60},
		parms: SEALSparams{
			PolyModulusDegree: polyModDegree,
			CoeffModBits:      []int{60, 40, 40, 60},
			PlainModulus:      plainMod,
		},
	}
}

func (s *SEALLikeScheme) Encode(values []int64) []int64 {
	encoded := make([]int64, s.polyModulusDegree*2)
	for i, v := range values {
		if i >= len(encoded) {
			break
		}
		encoded[i] = v % s.plainModulus
	}
	return encoded
}

func (s *SEALLikeScheme) Decode(plaintext []int64) []int64 {
	return plaintext[:len(plaintext)/2]
}

func (s *SEALLikeScheme) AddPlain(encrypted []int64, plaintext []int64) []int64 {
	result := make([]int64, len(encrypted))
	for i := range result {
		result[i] = (encrypted[i] + plaintext[i]) % s.plainModulus
	}
	return result
}

func (s *SEALLikeScheme) MultiplyPlain(encrypted []int64, plaintext []int64) []int64 {
	result := make([]int64, len(encrypted)*2)
	for i := range encrypted {
		for j := range plaintext {
			result[i+j] = (result[i+j] + encrypted[i]*plaintext[j]) % s.plainModulus
		}
	}
	return result[:len(encrypted)]
}

func VectorCommitmentHash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func VectorCommitmentVerify(data []byte, commitment []byte) bool {
	hash := VectorCommitmentHash(data)
	if len(hash) != len(commitment) {
		return false
	}
	for i := range hash {
		if hash[i] != commitment[i] {
			return false
		}
	}
	return true
}
