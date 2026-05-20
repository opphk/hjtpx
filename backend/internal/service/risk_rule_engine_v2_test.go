package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewRiskRuleEngineV2(t *testing.T) {
	engine := NewRiskRuleEngineV2(nil, nil, nil)
	assert.NotNil(t, engine)
}

func TestRiskRuleEngineV2_EvaluateRules_Success(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "rule1",
				RuleName: "Test Rule 1",
				Enabled:  true,
				Action:   "block",
				Score:    30,
			},
			{
				RuleID:   "rule2",
				RuleName: "Test Rule 2",
				Enabled:  true,
				Action:   "verify",
				Score:    20,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Triggered)
	assert.Greater(t, result.TotalScore, 0)
	assert.NotEmpty(t, result.TriggeredRules)
}

func TestRiskRuleEngineV2_EvaluateRules_NoRules(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Triggered)
	assert.Equal(t, 0, result.TotalScore)
	assert.Empty(t, result.TriggeredRules)
}

func TestRiskRuleEngineV2_EvaluateRules_DisabledRules(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "rule1",
				RuleName: "Disabled Rule",
				Enabled:  false,
				Action:   "block",
				Score:    100,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.False(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_IPBlacklist(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "ip_blacklist",
				RuleName:   "IP黑名单",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "ip_blacklist",
				Conditions: map[string]interface{}{"ip": "10.0.0.1"},
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "10.0.0.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
	assert.Equal(t, 100, result.TotalScore)
}

func TestRiskRuleEngineV2_EvaluateRules_TooManyFailures(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "login_failures",
				RuleName:   "登录失败过多",
				Enabled:    true,
				Action:     "verify",
				Score:      50,
				RuleType:   "login_failures",
				Conditions: map[string]interface{}{"threshold": 3},
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:           1,
		Username:         "testuser",
		IP:               "192.168.1.1",
		EventType:        "login",
		LoginFailures:    5,
		RequestTime:      time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_VelocityLimit(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "velocity_limit",
				RuleName:   "频率限制",
				Enabled:    true,
				Action:     "verify",
				Score:      40,
				RuleType:   "velocity",
				Conditions: map[string]interface{}{"threshold": 10, "window": "1h"},
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:        1,
		Username:      "testuser",
		IP:            "192.168.1.1",
		EventType:     "login",
		EventCount:    15,
		RequestTime:   time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_NewDevice(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "new_device",
				RuleName:   "新设备登录",
				Enabled:    true,
				Action:     "verify",
				Score:      25,
				RuleType:   "new_device",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:        1,
		Username:      "testuser",
		IP:            "192.168.1.1",
		EventType:     "login",
		IsNewDevice:   true,
		RequestTime:   time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_UnusualLocation(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "unusual_location",
				RuleName:   "异常地理位置",
				Enabled:    true,
				Action:     "verify",
				Score:      35,
				RuleType:   "location",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:       1,
		Username:     "testuser",
		IP:           "192.168.1.1",
		EventType:    "login",
		IsUnusualLocation: true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_MultipleRules(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "rule1",
				RuleName:   "Rule 1",
				Enabled:    true,
				Action:     "verify",
				Score:      20,
				RuleType:   "test",
			},
			{
				RuleID:     "rule2",
				RuleName:   "Rule 2",
				Enabled:    true,
				Action:     "block",
				Score:      30,
				RuleType:   "test",
			},
			{
				RuleID:     "rule3",
				RuleName:   "Rule 3",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "test",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, 3, len(result.TriggeredRules))
	assert.Equal(t, 65, result.TotalScore)
}

func TestRiskRuleEngineV2_EvaluateRules_HighRiskScore(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "high_risk",
				RuleName:   "高风险用户",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "risk_level",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "riskyuser",
		IP:          "10.0.0.1",
		EventType:   "login",
		RiskLevel:   "high",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_GeoSpeedAnomaly(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "geo_speed",
				RuleName:   "地理位置跳跃",
				Enabled:    true,
				Action:     "verify",
				Score:      45,
				RuleType:   "geo_speed",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:           1,
		Username:         "testuser",
		IP:               "192.168.1.1",
		EventType:        "login",
		IsGeoSpeedAnomaly: true,
		RequestTime:      time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_BotDetection(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "bot_detection",
				RuleName:   "机器人检测",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "bot",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsBot:       true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_ProxyDetection(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "proxy",
				RuleName:   "代理检测",
				Enabled:    true,
				Action:     "verify",
				Score:      30,
				RuleType:   "proxy",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:       1,
		Username:     "testuser",
		IP:           "192.168.1.1",
		EventType:    "login",
		IsProxy:      true,
		RequestTime:  time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_VPNDetection(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "vpn",
				RuleName:   "VPN检测",
				Enabled:    true,
				Action:     "verify",
				Score:      20,
				RuleType:   "vpn",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsVPN:       true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_TorNode(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "tor",
				RuleName:   "Tor节点",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "tor",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsTor:      true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_HighRiskCountry(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "high_risk_country",
				RuleName:   "高风险国家",
				Enabled:    true,
				Action:     "verify",
				Score:      35,
				RuleType:   "country",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsHighRiskCountry: true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_TimeAnomaly(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "time_anomaly",
				RuleName:   "异常时间登录",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "time",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:        1,
		Username:      "testuser",
		IP:            "192.168.1.1",
		EventType:     "login",
		IsTimeAnomaly: true,
		RequestTime:   time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_DataCenter(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "datacenter",
				RuleName:   "数据中心IP",
				Enabled:    true,
				Action:     "verify",
				Score:      25,
				RuleType:   "datacenter",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:       1,
		Username:     "testuser",
		IP:           "192.168.1.1",
		EventType:    "login",
		IsDataCenter: true,
		RequestTime:  time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_AccountTakeover(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "account_takeover",
				RuleName:   "账户盗用",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "account_takeover",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:              1,
		Username:            "testuser",
		IP:                  "192.168.1.1",
		EventType:           "login",
		IsAccountTakeover:   true,
		RequestTime:         time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_CredentialStuffing(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "credential_stuffing",
				RuleName:   "凭证填充",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "credential_stuffing",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:                1,
		Username:              "testuser",
		IP:                    "192.168.1.1",
		EventType:             "login",
		IsCredentialStuffing:  true,
		RequestTime:           time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_PasswordSpraying(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "password_spraying",
				RuleName:   "密码喷洒",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "password_spraying",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:              1,
		Username:            "testuser",
		IP:                  "192.168.1.1",
		EventType:           "login",
		IsPasswordSpraying:  true,
		RequestTime:         time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_MultipleIPLogin(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "multi_ip",
				RuleName:   "多IP登录",
				Enabled:    true,
				Action:     "verify",
				Score:      30,
				RuleType:   "multi_ip",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:            1,
		Username:          "testuser",
		IP:                "192.168.1.1",
		EventType:         "login",
		UniqueIPCount:     5,
		RequestTime:       time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_DeviceFingerprintMismatch(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "fp_mismatch",
				RuleName:   "设备指纹不匹配",
				Enabled:    true,
				Action:     "verify",
				Score:      40,
				RuleType:   "fingerprint",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:                 1,
		Username:               "testuser",
		IP:                     "192.168.1.1",
		EventType:              "login",
		IsFingerprintMismatch:  true,
		RequestTime:            time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_BruteForce(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "brute_force",
				RuleName:   "暴力破解",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "brute_force",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsBruteForce: true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_SuspiciousUserAgent(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "suspicious_ua",
				RuleName:   "可疑UserAgent",
				Enabled:    true,
				Action:     "verify",
				Score:      20,
				RuleType:   "user_agent",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:        1,
		Username:      "testuser",
		IP:            "192.168.1.1",
		EventType:     "login",
		IsSuspiciousUserAgent: true,
		RequestTime:   time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_NoCookies(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "no_cookies",
				RuleName:   "无Cookie",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "cookies",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:       1,
		Username:     "testuser",
		IP:           "192.168.1.1",
		EventType:    "login",
		HasCookies:   false,
		RequestTime:  time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_HighRequestFrequency(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "high_freq",
				RuleName:   "高频请求",
				Enabled:    true,
				Action:     "verify",
				Score:      35,
				RuleType:   "frequency",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:             1,
		Username:           "testuser",
		IP:                 "192.168.1.1",
		EventType:          "login",
		RequestFrequency:   100,
		RequestTime:        time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_AbnormalSessionDuration(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "session_duration",
				RuleName:   "异常会话时长",
				Enabled:    true,
				Action:     "verify",
				Score:      25,
				RuleType:   "session",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:                  1,
		Username:                "testuser",
		IP:                      "192.168.1.1",
		EventType:               "login",
		IsAbnormalSessionDuration: true,
		RequestTime:             time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_LowTrustDevice(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "low_trust_device",
				RuleName:   "低信任设备",
				Enabled:    true,
				Action:     "verify",
				Score:      20,
				RuleType:   "device_trust",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:           1,
		Username:         "testuser",
		IP:               "192.168.1.1",
		EventType:        "login",
		DeviceTrustLevel: "low",
		RequestTime:      time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_HighTrustDevice(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "test_rule",
				RuleName:   "Test Rule",
				Enabled:    true,
				Action:     "allow",
				Score:      0,
				RuleType:   "test",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:           1,
		Username:         "testuser",
		IP:               "192.168.1.1",
		EventType:        "login",
		DeviceTrustLevel: "high",
		RequestTime:      time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.False(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_IPRange(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "ip_range",
				RuleName:   "IP段限制",
				Enabled:    true,
				Action:     "verify",
				Score:      25,
				RuleType:   "ip_range",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "10.255.255.1",
		EventType:   "login",
		IsIPInRange: true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_CompromisedPassword(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "compromised_pwd",
				RuleName:   "泄露密码",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "password_leak",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:                  1,
		Username:                "testuser",
		IP:                      "192.168.1.1",
		EventType:               "login",
		IsCompromisedPassword:   true,
		RequestTime:             time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_DomainAge(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "domain_age",
				RuleName:   "域名年龄",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "domain",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		IsNewDomain: true,
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_EmailDisposable(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "disposable_email",
				RuleName:   "临时邮箱",
				Enabled:    true,
				Action:     "block",
				Score:      100,
				RuleType:   "email",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:               1,
		Username:             "testuser",
		Email:                "test@tempmail.com",
		IP:                   "192.168.1.1",
		EventType:            "register",
		IsDisposableEmail:    true,
		RequestTime:          time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_EmailDomainReputation(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "email_domain",
				RuleName:   "邮箱域名信誉",
				Enabled:    true,
				Action:     "verify",
				Score:      30,
				RuleType:   "email_domain",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:                 1,
		Username:               "testuser",
		Email:                  "test@bad-domain.com",
		IP:                     "192.168.1.1",
		EventType:              "register",
		EmailDomainReputation:   "low",
		RequestTime:            time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
}

func TestRiskRuleEngineV2_EvaluateRules_CombinedRisk(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:     "rule1",
				RuleName:   "Rule 1",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "test",
			},
			{
				RuleID:     "rule2",
				RuleName:   "Rule 2",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "test",
			},
			{
				RuleID:     "rule3",
				RuleName:   "Rule 3",
				Enabled:    true,
				Action:     "verify",
				Score:      15,
				RuleType:   "test",
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, 45, result.TotalScore)
	assert.Equal(t, "verify", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_BlockAction(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "block_rule",
				RuleName: "Block Rule",
				Enabled:  true,
				Action:   "block",
				Score:    50,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "block", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_AllowAction(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "allow_rule",
				RuleName: "Allow Rule",
				Enabled:  true,
				Action:   "allow",
				Score:    20,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "allow", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_VerifyAction(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "verify_rule",
				RuleName: "Verify Rule",
				Enabled:  true,
				Action:   "verify",
				Score:    30,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "verify", result.Action)
}

func TestRiskRuleEngineV2_EvaluateRules_LogAction(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{
				RuleID:   "log_rule",
				RuleName: "Log Rule",
				Enabled:  true,
				Action:   "log",
				Score:    10,
			},
		},
	}

	ctx := &model.RiskContext{
		UserID:      1,
		Username:    "testuser",
		IP:          "192.168.1.1",
		EventType:   "login",
		RequestTime: time.Now(),
	}

	result, err := engine.EvaluateRules(context.Background(), ctx)
	assert.NoError(t, err)
	assert.True(t, result.Triggered)
	assert.Equal(t, "log", result.Action)
}

func TestRiskRuleEngineV2_GetRules(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true},
			{RuleID: "rule2", RuleName: "Rule 2", Enabled: false},
		},
	}

	rules, err := engine.GetRules(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rules))
}

func TestRiskRuleEngineV2_GetEnabledRules(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true},
			{RuleID: "rule2", RuleName: "Rule 2", Enabled: false},
			{RuleID: "rule3", RuleName: "Rule 3", Enabled: true},
		},
	}

	enabledRules := engine.GetEnabledRules()
	assert.Equal(t, 2, len(enabledRules))
	for _, rule := range enabledRules {
		assert.True(t, rule.Enabled)
	}
}

func TestRiskRuleEngineV2_AddRule(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	rule := &model.RiskRule{
		RuleID:   "new_rule",
		RuleName: "New Rule",
		Enabled:  true,
		Action:   "verify",
		Score:    25,
	}

	err := engine.AddRule(context.Background(), rule)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(engine.rules))
	assert.Equal(t, "new_rule", engine.rules[0].RuleID)
}

func TestRiskRuleEngineV2_UpdateRule(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true, Score: 20},
		},
	}

	updatedRule := &model.RiskRule{
		RuleID:   "rule1",
		RuleName: "Updated Rule",
		Enabled:  false,
		Score:    30,
	}

	err := engine.UpdateRule(context.Background(), "rule1", updatedRule)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Rule", engine.rules[0].RuleName)
	assert.Equal(t, false, engine.rules[0].Enabled)
	assert.Equal(t, 30, engine.rules[0].Score)
}

func TestRiskRuleEngineV2_UpdateRule_NotFound(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	updatedRule := &model.RiskRule{
		RuleID:   "nonexistent",
		RuleName: "Updated Rule",
	}

	err := engine.UpdateRule(context.Background(), "nonexistent", updatedRule)
	assert.Error(t, err)
}

func TestRiskRuleEngineV2_DeleteRule(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1"},
			{RuleID: "rule2", RuleName: "Rule 2"},
		},
	}

	err := engine.DeleteRule(context.Background(), "rule1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(engine.rules))
	assert.Equal(t, "rule2", engine.rules[0].RuleID)
}

func TestRiskRuleEngineV2_DeleteRule_NotFound(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	err := engine.DeleteRule(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestRiskRuleEngineV2_GetRuleByID(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true},
		},
	}

	rule, err := engine.GetRuleByID("rule1")
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, "rule1", rule.RuleID)
}

func TestRiskRuleEngineV2_GetRuleByID_NotFound(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	rule, err := engine.GetRuleByID("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, rule)
}

func TestRiskRuleEngineV2_EnableRule(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: false},
		},
	}

	err := engine.EnableRule(context.Background(), "rule1", true)
	assert.NoError(t, err)
	assert.True(t, engine.rules[0].Enabled)
}

func TestRiskRuleEngineV2_DisableRule(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true},
		},
	}

	err := engine.EnableRule(context.Background(), "rule1", false)
	assert.NoError(t, err)
	assert.False(t, engine.rules[0].Enabled)
}

func TestRiskRuleEngineV2_EnableRule_NotFound(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{},
	}

	err := engine.EnableRule(context.Background(), "nonexistent", true)
	assert.Error(t, err)
}

func TestRiskRuleEngineV2_GetRiskStatistics(t *testing.T) {
	engine := &mockRiskEngineV2{
		rules: []*model.RiskRule{
			{RuleID: "rule1", RuleName: "Rule 1", Enabled: true},
			{RuleID: "rule2", RuleName: "Rule 2", Enabled: false},
		},
	}

	stats, err := engine.GetRiskStatistics(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.TotalRules)
	assert.Equal(t, 1, stats.EnabledRules)
}

func TestRiskContext_Structure(t *testing.T) {
	now := time.Now()
	ctx := &model.RiskContext{
		UserID:              1,
		Username:            "testuser",
		Email:               "test@example.com",
		IP:                  "192.168.1.1",
		UserAgent:           "Mozilla/5.0",
		EventType:           "login",
		RequestTime:         now,
		LoginFailures:       2,
		EventCount:          10,
		IsNewDevice:         true,
		IsUnusualLocation:   false,
		IsGeoSpeedAnomaly:   false,
		IsBot:               false,
		IsProxy:             false,
		IsVPN:               false,
		IsTor:               false,
		IsHighRiskCountry:   false,
		IsTimeAnomaly:       false,
		IsDataCenter:        false,
		IsAccountTakeover:   false,
		IsCredentialStuffing: false,
		IsPasswordSpraying:  false,
		UniqueIPCount:       3,
		IsFingerprintMismatch: false,
		IsBruteForce:        false,
		IsSuspiciousUserAgent: false,
		HasCookies:          true,
		RequestFrequency:    5,
		IsAbnormalSessionDuration: false,
		DeviceTrustLevel:    "medium",
		IsIPInRange:         false,
		IsCompromisedPassword: false,
		IsNewDomain:         false,
		IsDisposableEmail:   false,
		EmailDomainReputation: "medium",
		RiskLevel:           "low",
	}

	assert.Equal(t, uint(1), ctx.UserID)
	assert.Equal(t, "testuser", ctx.Username)
	assert.Equal(t, "test@example.com", ctx.Email)
	assert.Equal(t, "192.168.1.1", ctx.IP)
	assert.Equal(t, "Mozilla/5.0", ctx.UserAgent)
	assert.Equal(t, "login", ctx.EventType)
	assert.Equal(t, now, ctx.RequestTime)
	assert.Equal(t, 2, ctx.LoginFailures)
	assert.Equal(t, 10, ctx.EventCount)
	assert.True(t, ctx.IsNewDevice)
	assert.False(t, ctx.IsUnusualLocation)
}

func TestRiskResult_Structure(t *testing.T) {
	result := &model.RiskResult{
		Triggered:      true,
		TotalScore:     65,
		Action:         "verify",
		TriggeredRules: []string{"rule1", "rule2"},
		Message:        "Verification required",
		Timestamp:      time.Now(),
	}

	assert.True(t, result.Triggered)
	assert.Equal(t, 65, result.TotalScore)
	assert.Equal(t, "verify", result.Action)
	assert.Equal(t, 2, len(result.TriggeredRules))
	assert.Equal(t, "rule1", result.TriggeredRules[0])
	assert.Equal(t, "rule2", result.TriggeredRules[1])
	assert.Equal(t, "Verification required", result.Message)
	assert.False(t, result.Timestamp.IsZero())
}

func TestRiskRule_Structure(t *testing.T) {
	rule := &model.RiskRule{
		RuleID:     "test_rule",
		RuleName:   "Test Rule",
		Enabled:    true,
		Action:     "verify",
		Score:      30,
		RuleType:   "test",
		Conditions: map[string]interface{}{"threshold": 5},
	}

	assert.Equal(t, "test_rule", rule.RuleID)
	assert.Equal(t, "Test Rule", rule.RuleName)
	assert.True(t, rule.Enabled)
	assert.Equal(t, "verify", rule.Action)
	assert.Equal(t, 30, rule.Score)
	assert.Equal(t, "test", rule.RuleType)
	assert.Equal(t, 5, rule.Conditions["threshold"])
}

type mockRiskEngineV2 struct {
	rules []*model.RiskRule
}

func (m *mockRiskEngineV2) EvaluateRules(ctx context.Context, riskCtx *model.RiskContext) (*model.RiskResult, error) {
	var triggeredRules []string
	totalScore := 0
	maxAction := "allow"

	for _, rule := range m.rules {
		if !rule.Enabled {
			continue
		}

		triggered := m.checkRuleCondition(rule, riskCtx)
		if triggered {
			triggeredRules = append(triggeredRules, rule.RuleID)
			totalScore += rule.Score

			if rule.Action == "block" {
				maxAction = "block"
			} else if rule.Action == "verify" && maxAction != "block" {
				maxAction = "verify"
			} else if rule.Action == "log" && maxAction == "allow" {
				maxAction = "log"
			}
		}
	}

	return &model.RiskResult{
		Triggered:      len(triggeredRules) > 0,
		TotalScore:     totalScore,
		Action:         maxAction,
		TriggeredRules: triggeredRules,
		Message:       "Risk evaluation completed",
		Timestamp:     time.Now(),
	}, nil
}

func (m *mockRiskEngineV2) checkRuleCondition(rule *model.RiskRule, ctx *model.RiskContext) bool {
	switch rule.RuleType {
	case "ip_blacklist":
		if ctx.IP == "10.0.0.1" {
			return true
		}
	case "login_failures":
		return ctx.LoginFailures >= 3
	case "velocity":
		return ctx.EventCount > 10
	case "new_device":
		return ctx.IsNewDevice
	case "location":
		return ctx.IsUnusualLocation
	case "geo_speed":
		return ctx.IsGeoSpeedAnomaly
	case "bot":
		return ctx.IsBot
	case "proxy":
		return ctx.IsProxy
	case "vpn":
		return ctx.IsVPN
	case "tor":
		return ctx.IsTor
	case "country":
		return ctx.IsHighRiskCountry
	case "time":
		return ctx.IsTimeAnomaly
	case "datacenter":
		return ctx.IsDataCenter
	case "account_takeover":
		return ctx.IsAccountTakeover
	case "credential_stuffing":
		return ctx.IsCredentialStuffing
	case "password_spraying":
		return ctx.IsPasswordSpraying
	case "multi_ip":
		return ctx.UniqueIPCount > 3
	case "fingerprint":
		return ctx.IsFingerprintMismatch
	case "brute_force":
		return ctx.IsBruteForce
	case "user_agent":
		return ctx.IsSuspiciousUserAgent
	case "cookies":
		return !ctx.HasCookies
	case "frequency":
		return ctx.RequestFrequency > 50
	case "session":
		return ctx.IsAbnormalSessionDuration
	case "device_trust":
		return ctx.DeviceTrustLevel == "low"
	case "ip_range":
		return ctx.IsIPInRange
	case "password_leak":
		return ctx.IsCompromisedPassword
	case "domain":
		return ctx.IsNewDomain
	case "email":
		return ctx.IsDisposableEmail
	case "email_domain":
		return ctx.EmailDomainReputation == "low"
	case "risk_level":
		return ctx.RiskLevel == "high"
	default:
		if rule.Score > 0 {
			return true
		}
	}
	return false
}

func (m *mockRiskEngineV2) GetRules(ctx context.Context) ([]*model.RiskRule, error) {
	return m.rules, nil
}

func (m *mockRiskEngineV2) GetEnabledRules() []*model.RiskRule {
	var enabled []*model.RiskRule
	for _, rule := range m.rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

func (m *mockRiskEngineV2) AddRule(ctx context.Context, rule *model.RiskRule) error {
	m.rules = append(m.rules, rule)
	return nil
}

func (m *mockRiskEngineV2) UpdateRule(ctx context.Context, ruleID string, rule *model.RiskRule) error {
	for i, r := range m.rules {
		if r.RuleID == ruleID {
			m.rules[i] = rule
			return nil
		}
	}
	return errors.New("rule not found")
}

func (m *mockRiskEngineV2) DeleteRule(ctx context.Context, ruleID string) error {
	for i, r := range m.rules {
		if r.RuleID == ruleID {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return errors.New("rule not found")
}

func (m *mockRiskEngineV2) GetRuleByID(ruleID string) (*model.RiskRule, error) {
	for _, r := range m.rules {
		if r.RuleID == ruleID {
			return r, nil
		}
	}
	return nil, errors.New("rule not found")
}

func (m *mockRiskEngineV2) EnableRule(ctx context.Context, ruleID string, enabled bool) error {
	for _, r := range m.rules {
		if r.RuleID == ruleID {
			r.Enabled = enabled
			return nil
		}
	}
	return errors.New("rule not found")
}

func (m *mockRiskEngineV2) GetRiskStatistics(ctx context.Context) (*model.RiskStatistics, error) {
	enabledCount := 0
	for _, r := range m.rules {
		if r.Enabled {
			enabledCount++
		}
	}
	return &model.RiskStatistics{
		TotalRules:  len(m.rules),
		EnabledRules: enabledCount,
	}, nil
}
