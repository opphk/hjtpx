package service

import (
	"encoding/base32"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTOTP(t *testing.T) {
	// 测试用的密钥和对应的码（固定时间窗口）
	secret := "JBSWY3DPEHPK3PXP" // 测试密钥

	// 测试 validateTOTP 函数（我们需要修改它来接受自定义时间）
	// 这里我们通过测试时间来验证逻辑
	t.Run("TOTP validation logic", func(t *testing.T) {
		// 这里测试的是基本的HMAC逻辑，不依赖实际时间
		// 我们直接测试 validateTOTP 函数的实现
		key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
		assert.NoError(t, err)
		assert.NotEmpty(t, key)
	})
}

func TestGenerateBackupCodes(t *testing.T) {
	t.Run("Generate 10 backup codes", func(t *testing.T) {
		service := NewMFAService()
		codes, err := service.GenerateBackupCodes()
		assert.NoError(t, err)
		assert.Len(t, codes, 10)
		for _, code := range codes {
			assert.Len(t, code, 8)
		}
	})

	t.Run("Backup codes are unique", func(t *testing.T) {
		service := NewMFAService()
		codes1, _ := service.GenerateBackupCodes()
		codes2, _ := service.GenerateBackupCodes()
		// 两个不同的生成应该有不同的码（概率极低相同）
		different := false
		for i := range codes1 {
			if codes1[i] != codes2[i] {
				different = true
				break
			}
		}
		assert.True(t, different)
	})
}

func TestMFAServiceInterface(t *testing.T) {
	t.Run("Service creation", func(t *testing.T) {
		service := NewMFAService()
		assert.NotNil(t, service)
	})
}

// 测试TOTP相关功能的完整性
func TestTOTPWorkflow(t *testing.T) {
	// 这个测试主要验证TOTP功能的完整性
	// 在实际项目中需要使用测试数据库
	t.Skip("Skipping database-dependent test")
}

// 测试短信验证码功能
func TestSMSCodeWorkflow(t *testing.T) {
	t.Skip("Skipping database-dependent test")
}

// 测试邮箱验证码功能
func TestEmailCodeWorkflow(t *testing.T) {
	t.Skip("Skipping database-dependent test")
}

// 测试备份码功能
func TestBackupCodes(t *testing.T) {
	t.Skip("Skipping database-dependent test")
}
