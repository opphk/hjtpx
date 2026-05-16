package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Encryptor 加密器
type Encryptor struct {
	key    []byte
	cipher cipher.Block
}

// NewEncryptor 创建加密器
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("密钥长度必须为16、24或32字节")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &Encryptor{
		key:    key,
		cipher: block,
	}, nil
}

// Encrypt 使用AES-GCM加密
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e.cipher == nil {
		return nil, errors.New("加密器未初始化")
	}

	gcm, err := cipher.NewGCM(e.cipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 使用AES-GCM解密
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e.cipher == nil {
		return nil, errors.New("加密器未初始化")
	}

	gcm, err := cipher.NewGCM(e.cipher)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("密文太短")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptString 加密字符串
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptString 解密字符串
func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	plaintext, err := e.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// HashSHA256 计算SHA256哈希
func HashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashSHA256String 计算字符串的SHA256哈希
func HashSHA256String(data string) string {
	return HashSHA256([]byte(data))
}

// HashHMAC 计算HMAC
func HashHMAC(data, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HashHMACString 计算字符串的HMAC
func HashHMACString(data, key string) string {
	return HashHMAC([]byte(data), []byte(key))
}

// VerifyHMAC 验证HMAC
func VerifyHMAC(data, key, expectedMAC []byte) bool {
	return hmac.Equal(ComputeHMAC(data, key), expectedMAC)
}

// ComputeHMAC 计算HMAC
func ComputeHMAC(message, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil)
}

// ConstantTimeCompare 恒定时间比较
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateRandomKey 生成随机密钥
func GenerateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// MaskSensitiveData 脱敏敏感数据
func MaskSensitiveData(data string, dataType string) string {
	switch dataType {
	case "phone":
		return maskPhone(data)
	case "email":
		return maskEmail(data)
	case "id_card":
		return maskIDCard(data)
	case "bank_card":
		return maskBankCard(data)
	case "password":
		return "******"
	case "api_key":
		return maskAPIKey(data)
	case "ip":
		return maskIP(data)
	default:
		return maskGeneric(data)
	}
}

// maskPhone 掩码手机号
func maskPhone(phone string) string {
	if len(phone) < 11 {
		return strings.Repeat("*", len(phone))
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// maskEmail 掩码邮箱
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return maskGeneric(email)
	}

	username := parts[0]
	if len(username) <= 2 {
		return "**@" + parts[1]
	}

	return username[:2] + strings.Repeat("*", len(username)-2) + "@" + parts[1]
}

// maskIDCard 掩码身份证号
func maskIDCard(idCard string) string {
	if len(idCard) < 8 {
		return strings.Repeat("*", len(idCard))
	}
	return idCard[:4] + "**********" + idCard[len(idCard)-4:]
}

// maskBankCard 掩码银行卡号
func maskBankCard(bankCard string) string {
	if len(bankCard) < 8 {
		return strings.Repeat("*", len(bankCard))
	}
	return bankCard[:4] + strings.Repeat("*", len(bankCard)-8) + bankCard[len(bankCard)-4:]
}

// maskAPIKey 掩码API密钥
func maskAPIKey(apiKey string) string {
	if len(apiKey) < 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

// maskIP 掩码IP地址
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return maskGeneric(ip)
	}
	return parts[0] + "." + parts[1] + ".*.*"
}

// maskGeneric 通用掩码
func maskGeneric(data string) string {
	if len(data) <= 4 {
		return strings.Repeat("*", len(data))
	}
	return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}

// SensitiveDataMasker 敏感数据脱敏器
type SensitiveDataMasker struct {
	patterns map[string]*regexp.Regexp
}

// NewSensitiveDataMasker 创建敏感数据脱敏器
func NewSensitiveDataMasker() *SensitiveDataMasker {
	return &SensitiveDataMasker{
		patterns: map[string]*regexp.Regexp{
			"phone":      regexp.MustCompile(`\b1[3-9]\d{9}\b`),
			"email":      regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`),
			"id_card":    regexp.MustCompile(`\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`),
			"bank_card":  regexp.MustCompile(`\b\d{16,19}\b`),
			"api_key":    regexp.MustCompile(`\b[a-zA-Z0-9]{20,}\b`),
			"password":   regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`),
			"ip":         regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
		},
	}
}

// MaskAll 脱敏所有类型的数据
func (m *SensitiveDataMasker) MaskAll(text string) string {
	result := text

	result = m.patterns["phone"].ReplaceAllStringFunc(result, func(match string) string {
		return maskPhone(match)
	})

	result = m.patterns["email"].ReplaceAllStringFunc(result, func(match string) string {
		return maskEmail(match)
	})

	result = m.patterns["id_card"].ReplaceAllStringFunc(result, func(match string) string {
		return maskIDCard(match)
	})

	result = m.patterns["bank_card"].ReplaceAllStringFunc(result, func(match string) string {
		return maskBankCard(match)
	})

	result = m.patterns["password"].ReplaceAllStringFunc(result, func(match string) string {
		return "password=******"
	})

	result = m.patterns["api_key"].ReplaceAllStringFunc(result, func(match string) string {
		return maskAPIKey(match)
	})

	result = m.patterns["ip"].ReplaceAllStringFunc(result, func(match string) string {
		return maskIP(match)
	})

	return result
}

// MaskInJSON JSON数据脱敏
func (m *SensitiveDataMasker) MaskInJSON(jsonStr string) string {
	return m.MaskAll(jsonStr)
}

// MaskInMap Map数据脱敏
func (m *SensitiveDataMasker) MaskInMap(data map[string]interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		shouldMask := false
		for _, field := range fields {
			if key == field {
				shouldMask = true
				break
			}
		}

		if shouldMask {
			if strVal, ok := value.(string); ok {
				result[key] = maskGeneric(strVal)
			} else {
				result[key] = "******"
			}
		} else {
			result[key] = value
		}
	}

	return result
}

// SecureRandom 生成安全随机数
func SecureRandom(min, max int64) (int64, error) {
	if min >= max {
		return 0, errors.New("min必须小于max")
	}

	diff := max - min
	bytes := make([]byte, 8)

	for {
		if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
			return 0, err
		}

		num := int64(0)
		for _, b := range bytes {
			num = num*256 + int64(b)
		}

		if num >= 0 && num < diff*256 {
			return min + (num % diff), nil
		}
	}
}

// GenerateSecureToken 生成安全令牌
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashPassword 密码哈希（使用bcrypt风格的简单实现）
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	hash := sha256.Sum256(append([]byte(password), salt...))
	return fmt.Sprintf("%x.%x", hash, salt), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	parts := strings.SplitN(hashedPassword, ".", 2)
	if len(parts) != 2 {
		return false
	}

	hashStr := parts[0]
	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	expectedHash := sha256.Sum256(append([]byte(password), salt...))
	return fmt.Sprintf("%x", expectedHash) == hashStr
}

// DeriveKey 派生密钥
func DeriveKey(password, salt []byte, keyLen int) ([]byte, error) {
	hash := sha256.New()

	hash.Write(password)
	hash.Write(salt)
	hash.Write([]byte("derive"))

	result := make([]byte, keyLen)
	copy(result, hash.Sum(nil))

	return result[:keyLen], nil
}

// SecureZero 清除敏感数据
func SecureZero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// GenerateAPIKey 生成API密钥
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return "sk_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateAPIKey 验证API密钥格式
func ValidateAPIKey(apiKey string) bool {
	if len(apiKey) < 10 {
		return false
	}
	return strings.HasPrefix(apiKey, "sk_")
}
