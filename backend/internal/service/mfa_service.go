package service

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/pquerna/otp/totp"
)

type MFAService struct {
}

func NewMFAService() *MFAService {
	return &MFAService{}
}

type TOTPConfig struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
	URL    string `json:"url"`
}

func (s *MFAService) GenerateTOTPSecret(accountName, issuer string) (*TOTPConfig, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return nil, err
	}

	return &TOTPConfig{
		Secret: key.Secret(),
		URL:    key.URL(),
	}, nil
}

func (s *MFAService) VerifyTOTP(secret, code string) (bool, error) {
	valid := totp.Validate(code, secret)
	return valid, nil
}

func (s *MFAService) EnableTOTP(userID uint, secret string) error {
	mfa := &models.UserMFA{
		UserID:    userID,
		MFAType:   "totp",
		Secret:    secret,
		IsEnabled: true,
	}

	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return err
	}

	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return err
	}
	mfa.BackupCodes = string(backupCodesJSON)

	return nil
}

func (s *MFAService) SendSMSCode(userID uint, phone string) (string, *models.MFACode, error) {
	code, err := s.generateRandomCode()
	if err != nil {
		return "", nil, err
	}

	mfaCode := &models.MFACode{
		TargetType:  "user",
		TargetID:    userID,
		MFAType:     "sms",
		Code:        code,
		Destination: phone,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	fmt.Printf("模拟发送 SMS 验证码到 %s: %s\n", phone, code)

	return code, mfaCode, nil
}

func (s *MFAService) SendEmailCode(userID uint, email string) (string, *models.MFACode, error) {
	code, err := s.generateRandomCode()
	if err != nil {
		return "", nil, err
	}

	mfaCode := &models.MFACode{
		TargetType:  "user",
		TargetID:    userID,
		MFAType:     "email",
		Code:        code,
		Destination: email,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	fmt.Printf("模拟发送 Email 验证码到 %s: %s\n", email, code)

	return code, mfaCode, nil
}

func (s *MFAService) VerifyCode(code string, expectedCode *models.MFACode) (bool, error) {
	if expectedCode == nil {
		return false, fmt.Errorf("验证码无效")
	}

	if expectedCode.IsUsed {
		return false, fmt.Errorf("验证码已使用")
	}

	if time.Now().After(expectedCode.ExpiresAt) {
		return false, fmt.Errorf("验证码已过期")
	}

	if expectedCode.Code != code {
		return false, fmt.Errorf("验证码错误")
	}

	return true, nil
}

func (s *MFAService) EnableMFA(userID uint, mfaType, phone, email string) error {
	var mfa *models.UserMFA

	switch mfaType {
	case "totp":
		mfa = &models.UserMFA{
			UserID:    userID,
			MFAType:   "totp",
			IsEnabled: true,
		}
	case "sms":
		mfa = &models.UserMFA{
			UserID:    userID,
			MFAType:   "sms",
			Phone:     phone,
			IsEnabled: true,
		}
	case "email":
		mfa = &models.UserMFA{
			UserID:    userID,
			MFAType:   "email",
			Email:     email,
			IsEnabled: true,
		}
	default:
		return fmt.Errorf("无效的 MFA 类型")
	}

	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return err
	}

	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return err
	}
	mfa.BackupCodes = string(backupCodesJSON)

	return nil
}

func (s *MFAService) DisableMFA(userID uint) error {
	return nil
}

func (s *MFAService) GetMFAStatus(userID uint) (*models.UserMFA, error) {
	return nil, nil
}

func (s *MFAService) GenerateBackupCodes() ([]string, error) {
	return s.generateBackupCodes()
}

func (s *MFAService) generateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := range codes {
		code := make([]byte, 16)
		_, err := rand.Read(code)
		if err != nil {
			return nil, err
		}
		codes[i] = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code)[:8]
	}
	return codes, nil
}

func (s *MFAService) VerifyBackupCode(code string, backupCodesStr string) (bool, int, error) {
	var backupCodes []string
	err := json.Unmarshal([]byte(backupCodesStr), &backupCodes)
	if err != nil {
		return false, -1, err
	}

	for i, bc := range backupCodes {
		if bc == code {
			return true, i, nil
		}
	}

	return false, -1, nil
}

func (s *MFAService) generateRandomCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}
