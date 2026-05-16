package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLFilter(t *testing.T) {
	filter := NewSQLFilter()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"正常输入", "Hello World", false},
		{"包含UNION", "SELECT * FROM users UNION SELECT * FROM admins", true},
		{"包含OR 1=1", "admin' OR '1'='1", true},
		{"包含DROP", "DROP TABLE users", true},
		{"包含注释", "SELECT * FROM users -- comment", true},
		{"包含SQL关键字", "INSERT INTO users VALUES ('test')", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ContainsSQLKeywords(tt.input)
			assert.Equal(t, tt.expected, result, "SQL关键词检测失败")
		})
	}
}

func TestFilterSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空字符串", "", ""},
		{"正常字符串", "Hello", "Hello"},
		{"单引号转义", "user'name", "user''name"},
		{"移除注释", "SELECT * FROM users -- comment", "SELECT * FROM users  "},
		{"移除块注释", "SELECT /* comment */ * FROM users", "SELECT  * FROM users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterSQL(tt.input)
			assert.Contains(t, result, tt.expected[:min(len(tt.expected), len(result))])
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空字符串", "", ""},
		{"正常标识符", "users", "users"},
		{"大写转小写", "Users", "users"},
		{"带数字", "users123", "users123"},
		{"带下划线", "my_table", "my_table"},
		{"危险关键词", "DROP", ""},
		{"危险关键词小写", "drop", ""},
		{"非法字符", "user-name", ""},
		{"非法开头", "123table", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeQueryBuilder(t *testing.T) {
	t.Run("基本查询构建", func(t *testing.T) {
		qb := NewSafeQueryBuilder("users")
		query, args := qb.WhereEqual("id", 1).BuildSelect()

		assert.Contains(t, query, "SELECT * FROM users")
		assert.Contains(t, query, "WHERE")
		assert.Equal(t, 1, len(args))
	})

	t.Run("带排序和分页", func(t *testing.T) {
		qb := NewSafeQueryBuilder("users")
		query, args := qb.WhereEqual("status", "active").
			OrderBy("created_at", "DESC").
			Limit(10).
			Offset(20).
			BuildSelect()

		assert.Contains(t, query, "ORDER BY")
		assert.Contains(t, query, "LIMIT")
		assert.Contains(t, query, "OFFSET")
		assert.Equal(t, 2, len(args))
	})

	t.Run("空表名", func(t *testing.T) {
		qb := NewSafeQueryBuilder("")
		query, _ := qb.WhereEqual("id", 1).BuildSelect()
		assert.Equal(t, "", query)
	})

	t.Run("非法列名", func(t *testing.T) {
		qb := NewSafeQueryBuilder("users")
		query, _ := qb.WhereEqual("DROP", 1).BuildSelect()
		assert.NotContains(t, query, "DROP")
	})
}

func TestDetectSQLInjection(t *testing.T) {
	detector := NewSQLInjectionDetector()

	tests := []struct {
		name     string
		input    string
		expected bool
		minScore int
	}{
		{"正常输入", "Hello World", false, 0},
		{"UNION注入", "SELECT * FROM users UNION SELECT * FROM admins", true, 10},
		{"DROP注入", "DROP TABLE users", true, 10},
		{"多次注入", "SELECT * FROM users; DELETE FROM admins; DROP TABLE users", true, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected, score := detector.Detect(tt.input)
			assert.Equal(t, tt.expected, detected)
			if tt.expected {
				assert.GreaterOrEqual(t, score, tt.minScore)
			}
		})
	}
}

func TestEncryptor(t *testing.T) {
	t.Run("加密和解密", func(t *testing.T) {
		key := []byte("1234567890123456")
		encryptor, err := NewEncryptor(key)
		require.NoError(t, err)

		plaintext := []byte("Hello, World!")
		ciphertext, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)

		decrypted, err := encryptor.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("字符串加密解密", func(t *testing.T) {
		key := []byte("1234567890123456")
		encryptor, err := NewEncryptor(key)
		require.NoError(t, err)

		plaintext := "敏感数据"
		ciphertext, err := encryptor.EncryptString(plaintext)
		require.NoError(t, err)

		decrypted, err := encryptor.DecryptString(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("无效密钥长度", func(t *testing.T) {
		key := []byte("short")
		_, err := NewEncryptor(key)
		assert.Error(t, err)
	})
}

func TestHashFunctions(t *testing.T) {
	t.Run("SHA256哈希", func(t *testing.T) {
		data := []byte("test data")
		hash1 := HashSHA256(data)
		hash2 := HashSHA256(data)

		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64)
	})

	t.Run("HMAC", func(t *testing.T) {
		data := []byte("test data")
		key := []byte("secret key")
		hmac1 := HashHMAC(data, key)
		hmac2 := HashHMAC(data, key)

		assert.Equal(t, hmac1, hmac2)
		assert.NotEmpty(t, hmac1)
	})
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		dataType string
		expected string
	}{
		{"手机号", "13812345678", "phone", "138****5678"},
		{"邮箱", "user@example.com", "email", "us****@example.com"},
		{"身份证", "110101199001011234", "id_card", "1101**********1234"},
		{"银行卡", "6222021234567890123", "bank_card", "6222************0123"},
		{"密码", "secretpassword", "password", "******"},
		{"API密钥", "sk_abcdefghijklmnopqrstuvwxyz123456", "api_key", "sk_a************************3456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.data, tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSensitiveDataMasker(t *testing.T) {
	masker := NewSensitiveDataMasker()

	t.Run("JSON数据脱敏", func(t *testing.T) {
		jsonData := `{"phone": "13812345678", "email": "user@example.com"}`
		result := masker.MaskAll(jsonData)

		assert.Contains(t, result, "138****5678")
		assert.Contains(t, result, "us****@example.com")
	})

	t.Run("Map数据脱敏", func(t *testing.T) {
		data := map[string]interface{}{
			"username": "john",
			"password": "secret123",
			"email":    "john@example.com",
		}

		result := masker.MaskInMap(data, []string{"password"})

		assert.Equal(t, "john", result["username"])
		assert.Equal(t, "******", result["password"])
	})
}

func TestGenerateSecureToken(t *testing.T) {
	t.Run("生成令牌", func(t *testing.T) {
		token1, err := GenerateSecureToken(32)
		require.NoError(t, err)
		assert.Len(t, token1, 44)

		token2, err := GenerateSecureToken(32)
		require.NoError(t, err)
		assert.NotEqual(t, token1, token2)
	})
}

func TestIPManager(t *testing.T) {
	manager := NewIPManager()

	t.Run("基本操作", func(t *testing.T) {
		manager.AddToWhitelist("192.168.1.1")
		manager.AddToBlacklist("10.0.0.1")

		assert.True(t, manager.IsAllowed("192.168.1.1"))
		assert.True(t, manager.IsBlocked("10.0.0.1"))
		assert.False(t, manager.IsAllowed("10.0.0.1"))
	})

	t.Run("IP范围", func(t *testing.T) {
		err := manager.AddCIDR("192.168.1.0/24")
		require.NoError(t, err)

		assert.True(t, manager.IsAllowed("192.168.1.50"))
		assert.True(t, manager.IsAllowed("192.168.1.100"))
	})

	t.Run("白名单优先级", func(t *testing.T) {
		manager.AddToWhitelist("10.0.0.1")
		manager.AddToBlacklist("10.0.0.1")

		assert.True(t, manager.IsAllowed("10.0.0.1"))
	})
}

func TestLoginProtector(t *testing.T) {
	protector := NewLoginProtector(5, 15*time.Minute)

	t.Run("记录失败尝试", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			protector.RecordAttempt("user1")
		}

		assert.False(t, protector.IsLocked("user1"))
		assert.Equal(t, 1, protector.GetRemainingAttempts("user1"))
	})

	t.Run("锁定账户", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			protector.RecordAttempt("user2")
		}

		assert.True(t, protector.IsLocked("user2"))
	})

	t.Run("重置尝试", func(t *testing.T) {
		protector.RecordAttempt("user3")
		protector.ResetAttempts("user3")

		assert.False(t, protector.IsLocked("user3"))
		assert.Equal(t, 5, protector.GetRemainingAttempts("user3"))
	})
}

func TestBruteForceProtector(t *testing.T) {
	config := &BruteForceConfig{
		MaxAttempts: 3,
		LockoutTime: 5 * time.Minute,
	}
	protector := NewBruteForceProtector(config)

	t.Run("记录失败并锁定", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			isLocked, _ := protector.RecordFailure("user1")
			if i < 2 {
				assert.False(t, isLocked)
			}
		}

		assert.True(t, protector.IsLocked("user1"))
	})

	t.Run("记录成功后解锁", func(t *testing.T) {
		protector.RecordSuccess("user2")
		assert.False(t, protector.IsLocked("user2"))
	})

	t.Run("手动解锁", func(t *testing.T) {
		protector.RecordFailure("user3")
		protector.RecordFailure("user3")
		protector.RecordFailure("user3")

		assert.True(t, protector.IsLocked("user3"))

		protector.Unlock("user3")
		assert.False(t, protector.IsLocked("user3"))
	})
}

func TestAuditLogger(t *testing.T) {
	logger := NewAuditLogger("", 100)

	t.Run("记录登录事件", func(t *testing.T) {
		logger.LogLogin(true, "user1", "192.168.1.1", 1)
		logger.LogLogin(false, "user1", "192.168.1.1", 0)

		logs := logger.GetRecentLogs(10)
		assert.GreaterOrEqual(t, len(logs), 2)
	})

	t.Run("记录安全违规", func(t *testing.T) {
		logger.LogSecurityViolation(
			"sql_injection_attempt",
			"192.168.1.100",
			"/api/login",
			"Mozilla/5.0",
			map[string]interface{}{"payload": "test"},
		)

		logs := logger.GetCriticalLogs(time.Hour)
		assert.Greater(t, len(logs), 0)
	})

	t.Run("获取统计信息", func(t *testing.T) {
		stats := logger.GetStats(time.Hour)

		assert.Contains(t, stats, "total_events")
		assert.Contains(t, stats, "critical_events")
		assert.Contains(t, stats, "unique_ips")
	})
}

func TestSecurityConfig(t *testing.T) {
	t.Run("加载默认配置", func(t *testing.T) {
		config := NewSecurityConfig()
		assert.NotNil(t, config)
		assert.True(t, config.RateLimit.Enabled)
		assert.True(t, config.CSRF.Enabled)
		assert.True(t, config.XSS.Enabled)
	})

	t.Run("验证配置", func(t *testing.T) {
		config := NewSecurityConfig()
		err := config.ValidateConfig()
		assert.NoError(t, err)
	})

	t.Run("更新配置", func(t *testing.T) {
		config := NewSecurityConfig()

		newRateLimit := RateLimitConfig{
			Enabled:      true,
			GlobalLimit:  2000,
			GlobalWindow: 2 * time.Minute,
		}
		config.UpdateRateLimitConfig(newRateLimit)

		retrieved := config.GetRateLimitConfig()
		assert.Equal(t, 2000, retrieved.GlobalLimit)
	})

	t.Run("克隆配置", func(t *testing.T) {
		config := NewSecurityConfig()
		clone := config.CloneConfig()

		assert.Equal(t, config.RateLimit.GlobalLimit, clone.RateLimit.GlobalLimit)
	})

	t.Run("导出导入配置", func(t *testing.T) {
		config := NewSecurityConfig()

		data, err := config.ExportConfig()
		require.NoError(t, err)

		newConfig := NewSecurityConfig()
		err = newConfig.ImportConfig(data)
		require.NoError(t, err)

		assert.Equal(t, config.RateLimit.GlobalLimit, newConfig.RateLimit.GlobalLimit)
	})
}

func BenchmarkSQLFilter(b *testing.B) {
	filter := NewSQLFilter()
	input := "SELECT * FROM users WHERE id = 1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.ContainsSQLKeywords(input)
	}
}

func BenchmarkEncryption(b *testing.B) {
	key := []byte("1234567890123456")
	encryptor, _ := NewEncryptor(key)
	plaintext := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := encryptor.Encrypt(plaintext)
		encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkMaskPhone(b *testing.B) {
	phone := "13812345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MaskSensitiveData(phone, "phone")
	}
}
