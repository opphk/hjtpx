package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

type PasswordStrength struct {
	Score      int
	Strength   string
	Violations []string
}

type EnhancedPasswordPolicy struct {
	MinLength           int
	RequireUppercase    bool
	RequireLowercase    bool
	RequireNumbers      bool
	RequireSpecialChars bool
	MaxLength           int
	ProhibitCommon      bool
	ProhibitUsername    bool
	ProhibitKeyboard    bool
	ProhibitRepeating   bool
}

var DefaultPasswordPolicy = EnhancedPasswordPolicy{
	MinLength:           8,
	RequireUppercase:    true,
	RequireLowercase:    true,
	RequireNumbers:      true,
	RequireSpecialChars: true,
	MaxLength:           128,
	ProhibitCommon:      true,
	ProhibitUsername:    true,
	ProhibitKeyboard:    true,
	ProhibitRepeating:   true,
}

var commonPasswords = []string{
	"password", "123456", "12345678", "qwerty", "abc123",
	"monkey", "1234567", "letmein", "trustno1", "dragon",
	"baseball", "iloveyou", "master", "sunshine", "ashley",
	"football", "password1", "shadow", "123123", "654321",
}

var keyboardPatterns = []string{
	"qwerty", "asdfgh", "zxcvbn", "qwertyuiop", "asdfghjkl",
	"1234567890", "0987654321", "qazwsx", "edcrfv", "tgbyhn",
}

func NewEnhancedPasswordPolicy(minLength int) *EnhancedPasswordPolicy {
	policy := DefaultPasswordPolicy
	if minLength > 0 {
		policy.MinLength = minLength
	}
	return &policy
}

func (p *EnhancedPasswordPolicy) ValidatePassword(password, username string) PasswordStrength {
	violations := make([]string, 0)
	score := 100

	if len(password) < p.MinLength {
		violations = append(violations, "密码长度至少需要8个字符")
		score -= 30
	}

	if len(password) > p.MaxLength {
		violations = append(violations, "密码长度不能超过128个字符")
		score -= 20
	}

	if p.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		violations = append(violations, "密码必须包含至少一个大写字母")
		score -= 15
	}

	if p.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		violations = append(violations, "密码必须包含至少一个小写字母")
		score -= 15
	}

	if p.RequireNumbers && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		violations = append(violations, "密码必须包含至少一个数字")
		score -= 15
	}

	if p.RequireSpecialChars && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;':\",./<>?]`).MatchString(password) {
		violations = append(violations, "密码必须包含至少一个特殊字符")
		score -= 15
	}

	if p.ProhibitCommon {
		lower := strings.ToLower(password)
		for _, common := range commonPasswords {
			if lower == common || strings.Contains(lower, common) {
				violations = append(violations, "密码包含常见词汇，不够安全")
				score -= 40
				break
			}
		}
	}

	if p.ProhibitUsername && username != "" {
		if strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
			violations = append(violations, "密码不能包含用户名")
			score -= 30
		}
	}

	if p.ProhibitKeyboard {
		lower := strings.ToLower(password)
		for _, pattern := range keyboardPatterns {
			if strings.Contains(lower, pattern) {
				violations = append(violations, "密码不能包含键盘连续字符")
				score -= 25
				break
			}
		}
	}

	if p.ProhibitRepeating {
		if hasRepeatingChars(password, 3) {
			violations = append(violations, "密码不能包含连续3个以上相同字符")
			score -= 20
		}
	}

	uniqueChars := countUniqueChars(password)
	uniqueRatio := float64(uniqueChars) / float64(len(password))
	if uniqueRatio < 0.5 && len(password) > 5 {
		violations = append(violations, "密码字符多样性不足")
		score -= 15
	}

	if score < 0 {
		score = 0
	}

	strength := "弱"
	if score >= 80 {
		strength = "强"
	} else if score >= 60 {
		strength = "中等"
	}

	return PasswordStrength{
		Score:      score,
		Strength:   strength,
		Violations: violations,
	}
}

func hasRepeatingChars(s string, maxRepeats int) bool {
	if len(s) < maxRepeats {
		return false
	}

	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count > maxRepeats {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

func countUniqueChars(s string) int {
	seen := make(map[rune]bool)
	for _, c := range s {
		seen[c] = true
	}
	return len(seen)
}

func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = 16
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	_, err := rand.Read(password)
	if err != nil {
		return "", err
	}

	for i := range password {
		password[i] = charset[int(password[i])%len(charset)]
	}

	return string(password), nil
}

func HashPasswordWithSalt(password string) (hash string, salt string, err error) {
	saltBytes := make([]byte, 32)
	_, err = rand.Read(saltBytes)
	if err != nil {
		return "", "", err
	}
	salt = hex.EncodeToString(saltBytes)

	saltedPassword := password + salt
	hashBytes := sha256.Sum256([]byte(saltedPassword))
	hash = hex.EncodeToString(hashBytes[:])

	return hash, salt, nil
}

func VerifyPasswordWithSalt(password, hash, salt string) bool {
	saltedPassword := password + salt
	hashBytes := sha256.Sum256([]byte(saltedPassword))
	computedHash := hex.EncodeToString(hashBytes[:])
	return computedHash == hash
}

type SessionConfig struct {
	SessionTimeout    time.Duration
	AbsoluteTimeout   time.Duration
	IdleTimeout       time.Duration
	MaxConcurrent     int
	RequireReAuth     bool
	SecureCookie      bool
	HttpOnlyCookie    bool
	SameSiteCookie    string
	CookieName        string
	CookiePrefix      string
}

var DefaultSessionConfig = SessionConfig{
	SessionTimeout:  24 * time.Hour,
	AbsoluteTimeout:  7 * 24 * time.Hour,
	IdleTimeout:      30 * time.Minute,
	MaxConcurrent:    3,
	RequireReAuth:    true,
	SecureCookie:     true,
	HttpOnlyCookie:   true,
	SameSiteCookie:   "strict",
	CookieName:       "hjtpx_session",
	CookiePrefix:     "hjtpx_",
}

type SecureSessionData struct {
	SessionID    string
	UserID       uint
	Username     string
	CreatedAt    time.Time
	LastActivity time.Time
	IPAddress    string
	UserAgent    string
	DeviceFingerprint string
	IsValid      bool
}

func NewSessionManager(config *SessionConfig) *SessionManager {
	if config == nil {
		config = &DefaultSessionConfig
	}
	return &SessionManager{
		config: config,
		sessions: make(map[string]*SecureSessionData),
	}
}

type SessionManager struct {
	config   *SessionConfig
	sessions map[string]*SecureSessionData
}

func (sm *SessionManager) CreateSession(userID uint, username, ip, userAgent string) (*SecureSessionData, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &SecureSessionData{
		SessionID:    sessionID,
		UserID:       userID,
		Username:     username,
		CreatedAt:    now,
		LastActivity: now,
		IPAddress:    ip,
		UserAgent:    userAgent,
		IsValid:      true,
	}

	sm.sessions[sessionID] = session
	return session, nil
}

func (sm *SessionManager) ValidateSession(sessionID, ip, userAgent string) bool {
	session, exists := sm.sessions[sessionID]
	if !exists || !session.IsValid {
		return false
	}

	if time.Since(session.LastActivity) > sm.config.SessionTimeout {
		session.IsValid = false
		return false
	}

	if time.Since(session.CreatedAt) > sm.config.AbsoluteTimeout {
		session.IsValid = false
		return false
	}

	if sm.config.IdleTimeout > 0 && time.Since(session.LastActivity) > sm.config.IdleTimeout {
		session.IsValid = false
		return false
	}

	if ip != "" && session.IPAddress != ip {
		return false
	}

	if userAgent != "" && session.UserAgent != userAgent {
		return false
	}

	session.LastActivity = time.Now()
	return true
}

func (sm *SessionManager) InvalidateSession(sessionID string) {
	if session, exists := sm.sessions[sessionID]; exists {
		session.IsValid = false
	}
}

func (sm *SessionManager) InvalidateAllUserSessions(userID uint) {
	for _, session := range sm.sessions {
		if session.UserID == userID {
			session.IsValid = false
		}
	}
}

func (sm *SessionManager) GetActiveSessions(userID uint) []*SecureSessionData {
	sessions := make([]*SecureSessionData, 0)
	for _, session := range sm.sessions {
		if session.UserID == userID && session.IsValid {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (sm *SessionManager) CleanupExpiredSessions() int {
	count := 0
	now := time.Now()
	for sessionID, session := range sm.sessions {
		if !session.IsValid || 
		   now.Sub(session.LastActivity) > sm.config.SessionTimeout ||
		   now.Sub(session.CreatedAt) > sm.config.AbsoluteTimeout {
			delete(sm.sessions, sessionID)
			count++
		}
	}
	return count
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type EncryptionConfig struct {
	Algorithm      string
	KeySize        int
	Mode           string
	EnableCBC      bool
	EnableGCM      bool
	EnableHMAC     bool
	HMACAlgorithm  string
	RotateKeys     bool
	RotationPeriod time.Duration
}

var DefaultEncryptionConfig = EncryptionConfig{
	Algorithm:     "AES-256-GCM",
	KeySize:       256,
	Mode:          "GCM",
	EnableCBC:     false,
	EnableGCM:     true,
	EnableHMAC:    true,
	HMACAlgorithm: "SHA-256",
	RotateKeys:    true,
	RotationPeriod: 90 * 24 * time.Hour,
}

func NewEnhancedEncryptionService(config *EncryptionConfig) *EncryptionService {
	if config == nil {
		config = &DefaultEncryptionConfig
	}
	return &EncryptionService{
		config: config,
	}
}

type EncryptionService struct {
	config *EncryptionConfig
}

func (es *EncryptionService) GenerateKey() ([]byte, error) {
	key := make([]byte, es.config.KeySize/8)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (es *EncryptionService) Encrypt(plaintext, key []byte) ([]byte, error) {
	return encryptAES(plaintext, key)
}

func (es *EncryptionService) Decrypt(ciphertext, key []byte) ([]byte, error) {
	return decryptAES(ciphertext, key)
}

func (es *EncryptionService) GenerateSignature(data, key []byte) ([]byte, error) {
	return generateHMAC(data, key)
}

func (es *EncryptionService) VerifySignature(data, signature, key []byte) bool {
	expected, err := generateHMAC(data, key)
	if err != nil {
		return false
	}
	return constantTimeCompare(signature, expected)
}

func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

func encryptAES(plaintext, key []byte) ([]byte, error) {
	block, err := newCipher(key)
	if err != nil {
		return nil, err
	}
	
	padding := block.BlockSize() - len(plaintext)%block.BlockSize()
	padtext := make([]byte, len(plaintext)+padding)
	copy(padtext, plaintext)
	for i := len(plaintext); i < len(padtext); i++ {
		padtext[i] = byte(padding)
	}
	
	ciphertext := make([]byte, block.BlockSize()+len(padtext))
	block.Encrypt(ciphertext, padtext)
	return ciphertext, nil
}

func decryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := newCipher(key)
	if err != nil {
		return nil, err
	}
	
	if len(ciphertext) < block.BlockSize() {
		return nil, errors.New("ciphertext too short")
	}
	
	plaintext := make([]byte, len(ciphertext)-block.BlockSize())
	block.Decrypt(plaintext, ciphertext)
	
	padding := int(plaintext[len(plaintext)-1])
	if padding > len(plaintext) {
		return nil, errors.New("invalid padding")
	}
	return plaintext[:len(plaintext)-padding], nil
}

func generateHMAC(data, key []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil), nil
}

func newCipher(key []byte) (cipher.Block, error) {
	switch len(key) {
	case 16:
		return aes.NewCipher(key)
	case 24:
		return aes.NewCipher(key)
	case 32:
		return aes.NewCipher(key)
	default:
		return nil, errors.New("invalid key size")
	}
}
