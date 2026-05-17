package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

type CryptoService struct {
	secretKey []byte
	nonceUsed map[string]bool
}

func NewCryptoService(key ...[]byte) *CryptoService {
	cs := &CryptoService{
		nonceUsed: make(map[string]bool),
	}
	if len(key) > 0 && len(key[0]) > 0 {
		cs.secretKey = key[0]
	} else {
		cs.secretKey = []byte("hjtpx-crypto-key-2024")
	}
	return cs
}

func (s *CryptoService) SetSecretKey(key []byte) {
	s.secretKey = key
}

func (s *CryptoService) GetSecretKey() []byte {
	return s.secretKey
}

func (s *CryptoService) aesEncrypt(plaintext []byte) ([]byte, error) {
	if len(s.secretKey) == 0 {
		return nil, errors.New("secret key not set")
	}

	keyHash := sha256.Sum256(s.secretKey)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *CryptoService) aesDecrypt(ciphertext []byte) ([]byte, error) {
	if len(s.secretKey) == 0 {
		return nil, errors.New("secret key not set")
	}

	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext is empty")
	}

	keyHash := sha256.Sum256(s.secretKey)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *CryptoService) EncryptString(plaintext string) (string, error) {
	encrypted, err := s.aesEncrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (s *CryptoService) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := s.aesDecrypt(data)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

func (s *CryptoService) EncryptParams(params map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal params: %w", err)
	}

	encrypted, err := s.aesEncrypt(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt params: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (s *CryptoService) DecryptParams(encrypted string) (map[string]interface{}, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := s.aesDecrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt params: %w", err)
	}

	var params map[string]interface{}
	err = json.Unmarshal(decrypted, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return params, nil
}

func (s *CryptoService) isNonceUsed(nonce string) bool {
	return s.nonceUsed[nonce]
}

func (s *CryptoService) markNonceUsed(nonce string) {
	s.nonceUsed[nonce] = true
}

func (s *CryptoService) IsNonceUsed(nonce string) bool {
	return s.isNonceUsed(nonce)
}

func (s *CryptoService) MarkNonceUsed(nonce string) {
	s.markNonceUsed(nonce)
}

type Protector struct {
	obfuscator *Obfuscator
	crypto     *CryptoService
}

func NewProtector(key ...[]byte) *Protector {
	return &Protector{
		obfuscator: NewObfuscator(),
		crypto:     NewCryptoService(key...),
	}
}

func (p *Protector) SetSecretKey(key []byte) {
	p.crypto.SetSecretKey(key)
}

func (p *Protector) Protect(code string) (string, error) {
	obfuscated, err := p.obfuscator.Obfuscate(code)
	if err != nil {
		return code, err
	}

	protected := p.addAntiDebug(obfuscated)

	protected = "(function(){" + protected + "})();"

	return protected, nil
}

func (p *Protector) ProtectWithLevel(code string, level int) (string, error) {
	config := ObfuscatorConfig{
		EnableVariableObfuscation:  true,
		EnableStringEncryption:     level >= 2,
		EnableCodeCompression:      true,
		EnableControlFlowFlattening: level >= 2,
		EnableDeadCodeInjection:    level >= 3,
		EnableFunctionWrapping:     true,
		RemoveComments:            true,
		StringEncryptionKey:        p.crypto.GetSecretKey(),
	}

	obfuscator := NewObfuscator(config)
	obfuscated, err := obfuscator.Obfuscate(code)
	if err != nil {
		return code, err
	}

	protected := p.addAntiDebug(obfuscated)

	protected = "(function(){" + protected + "})();"

	return protected, nil
}

func (p *Protector) addAntiDebug(code string) string {
	antiDebug := `
!function(){
var t=window.outerWidth-window.innerWidth,o=window.outerHeight-window.innerHeight;
if(t>160||o>160){document.documentElement.style.display='none';document.body.innerHTML='<h1>Developer Tools Detected</h1>';}
var e=Object.defineProperty({},'toString',{get:function(){this.t=1}});
setInterval(function(){e.t&&(document.documentElement.style.display='none');},1e3);
}();
`
	return antiDebug + code
}

func (p *Protector) EncryptAndProtect(code string) (string, error) {
	encrypted, err := p.crypto.EncryptString(code)
	if err != nil {
		return "", err
	}

	loader := fmt.Sprintf(`
(function(){
var _0xc='%s';
var _0xd=document.createElement('script');
_0xd.type='text/javascript';
var _0xe=atob('%s');
try{
var _0xf=document.createElement('script');
_0xf.textContent=atob(_0xc);
document.head.appendChild(_0xf);
}catch(_0xg){
console.error('Failed to load protected code');
}
})();
`, encrypted, base64.StdEncoding.EncodeToString([]byte("console.error('Protected code failed');")))

	return loader, nil
}

func (p *Protector) GenerateIntegrityCheck(code string) string {
	hash := sha256.Sum256([]byte(code))
	hashStr := base64.StdEncoding.EncodeToString(hash[:])

	return fmt.Sprintf(`
(function(){
var _0xh='%s';
var _0xi='';
document.addEventListener('DOMContentLoaded',function(){
if(typeof window.__h==='undefined'){window.__h=_0xh;}
});
})();
`, hashStr)
}

func (p *Protector) VerifyIntegrity(code, expectedHash string) bool {
	hash := sha256.Sum256([]byte(code))
	hashStr := base64.StdEncoding.EncodeToString(hash[:])
	return hashStr == expectedHash
}

type ParameterProtector struct {
	crypto    *CryptoService
	keyPrefix string
}

func NewParameterProtector(key ...[]byte) *ParameterProtector {
	return &ParameterProtector{
		crypto:    NewCryptoService(key...),
		keyPrefix: "X-Enc-",
	}
}

func (p *ParameterProtector) EncryptRequestParams(params map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)

	combined := make(map[string]interface{})
	combined["params"] = params

	jsonData, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	encrypted, err := p.crypto.aesEncrypt(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt params: %w", err)
	}

	result["data"] = base64.StdEncoding.EncodeToString(encrypted)

	return result, nil
}

func (p *ParameterProtector) DecryptRequestParams(encryptedData string) (map[string]interface{}, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := p.crypto.aesDecrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt params: %w", err)
	}

	var combined map[string]interface{}
	err = json.Unmarshal(decrypted, &combined)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if params, ok := combined["params"].(map[string]interface{}); ok {
		return params, nil
	}

	return combined, nil
}

type SignatureValidator struct {
	secretKey       []byte
	timestampTTL    int64
	nonceCache      map[string]int64
	maxNonceCache   int
}

func NewSignatureValidator(secretKey []byte) *SignatureValidator {
	return &SignatureValidator{
		secretKey:     secretKey,
		timestampTTL:  300,
		nonceCache:    make(map[string]int64),
		maxNonceCache: 10000,
	}
}

func (v *SignatureValidator) SetTimestampTTL(ttl int64) {
	v.timestampTTL = ttl
}

func (v *SignatureValidator) calculateSignature(method, path, timestamp, nonce string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(path))
	h.Write([]byte(timestamp))
	h.Write([]byte(nonce))
	if len(body) > 0 {
		h.Write(body)
	}
	h.Write(v.secretKey)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (v *SignatureValidator) ValidateRequest(method, path, signature, timestamp, nonce string, body []byte) error {
	if signature == "" || timestamp == "" || nonce == "" {
		return errors.New("missing signature headers")
	}

	ts, err := parseTimestamp(timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	if !v.isTimestampValid(ts) {
		return errors.New("request expired")
	}

	if v.isNonceUsed(nonce) {
		return errors.New("nonce reused")
	}
	v.markNonceUsed(nonce)

	expectedSig := v.calculateSignature(method, path, timestamp, nonce, body)
	if signature != expectedSig {
		return errors.New("invalid signature")
	}

	return nil
}

func (v *SignatureValidator) GenerateSignature(method, path string, body []byte) (signature, timestamp, nonce string, err error) {
	timestamp = fmt.Sprintf("%d", currentTime.Unix())

	nonceBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", "", "", err
	}
	nonce = base64.StdEncoding.EncodeToString(nonceBytes)

	signature = v.calculateSignature(method, path, timestamp, nonce, body)
	return signature, timestamp, nonce, nil
}

type timeProvider interface {
	Unix() int64
}

type realTime struct{}

func (realTime) Unix() int64 {
	return time.Now().Unix()
}

var currentTime timeProvider = realTime{}

func setTimeProvider(t timeProvider) {
	currentTime = t
}

func parseTimestamp(ts string) (int64, error) {
	var t int64
	_, err := fmt.Sscanf(ts, "%d", &t)
	if err != nil {
		return 0, err
	}
	return t, nil
}

func (v *SignatureValidator) isTimestampValid(ts int64) bool {
	now := currentTime.Unix()
	return (now-ts) <= v.timestampTTL && (ts-now) <= v.timestampTTL
}

func (v *SignatureValidator) isNonceUsed(nonce string) bool {
	_, exists := v.nonceCache[nonce]
	return exists
}

func (v *SignatureValidator) markNonceUsed(nonce string) {
	if len(v.nonceCache) >= v.maxNonceCache {
		v.cleanupNonceCache()
	}
	v.nonceCache[nonce] = currentTime.Unix()
}

func (v *SignatureValidator) cleanupNonceCache() {
	now := currentTime.Unix()
	for nonce, ts := range v.nonceCache {
		if (now-ts) > v.timestampTTL*2 {
			delete(v.nonceCache, nonce)
		}
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateRandomString(n int) string {
	bytes, _ := GenerateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(bytes)[:n]
}

func MaskSensitiveData(data string, visibleChars int) string {
	if len(data) <= visibleChars {
		return strings.Repeat("*", len(data))
	}
	return data[:visibleChars] + strings.Repeat("*", len(data)-visibleChars)
}

func SanitizeLogOutput(data string) string {
	sensitivePatterns := []string{
		`password["\s]*[:=]["\s]*\S+`,
		`token["\s]*[:=]["\s]*\S+`,
		`secret["\s]*[:=]["\s]*\S+`,
		`key["\s]*[:=]["\s]*\S+`,
		`auth["\s]*[:=]["\s]*\S+`,
	}

	result := data
	for _, pattern := range sensitivePatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) >= 2 {
				return parts[1] + ": [REDACTED]"
			}
			return "[REDACTED]"
		})
	}

	return result
}
