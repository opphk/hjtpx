package risk

import (
	"math"
	"time"
)

type Engine struct {
	rules       []Rule
	ipCache     *IPCache
	deviceCache *DeviceCache
}

type Rule struct {
	Name      string           `json:"name"`
	Condition func(*Context) bool `json:"-"`
	Score     float64          `json:"score"`
	Action    string           `json:"action"`
	Priority  int              `json:"priority"`
}

type Context struct {
	IPAddress     string
	Fingerprint   string
	UserAgent     string
	SessionCount  int
	AttemptCount  int
	FailCount     int
	SuccessCount  int
	RequestCount  int
	RiskScore     float64
	IsNewDevice   bool
	IsNewIP       bool
	Country       string
	ASN           string
	IsVPN         bool
	IsProxy       bool
	IsTor         bool
	IsDatacenter  bool
	BlockedCount  int
	LastAttemptAt *time.Time
	LastSuccessAt *time.Time
	FirstSeenAt  *time.Time
}

type Decision struct {
	Allow       bool     `json:"allow"`
	RiskScore   float64  `json:"risk_score"`
	RiskLevel   string   `json:"risk_level"`
	Action      string   `json:"action"`
	Reasons     []string `json:"reasons"`
	CaptchaType string   `json:"captcha_type"`
}

func NewEngine() *Engine {
	engine := &Engine{
		rules:       make([]Rule, 0),
		ipCache:     NewIPCache(),
		deviceCache: NewDeviceCache(),
	}

	engine.initDefaultRules()

	return engine
}

func (e *Engine) initDefaultRules() {
	e.rules = append(e.rules, Rule{
		Name: "high_fail_rate",
		Condition: func(ctx *Context) bool {
			if ctx.AttemptCount < 3 {
				return false
			}
			failRate := float64(ctx.FailCount) / float64(ctx.AttemptCount)
			return failRate > 0.5
		},
		Score:    0.4,
		Action:   "captcha",
		Priority: 10,
	})

	e.rules = append(e.rules, Rule{
		Name: "too_many_attempts",
		Condition: func(ctx *Context) bool {
			return ctx.AttemptCount > 10
		},
		Score:    0.3,
		Action:   "captcha",
		Priority: 20,
	})

	e.rules = append(e.rules, Rule{
		Name: "new_device_ip",
		Condition: func(ctx *Context) bool {
			return ctx.IsNewDevice && ctx.IsNewIP
		},
		Score:    0.2,
		Action:   "captcha",
		Priority: 30,
	})

	e.rules = append(e.rules, Rule{
		Name: "vpn_proxy_tor",
		Condition: func(ctx *Context) bool {
			return ctx.IsVPN || ctx.IsProxy || ctx.IsTor
		},
		Score:    0.5,
		Action:   "captcha",
		Priority: 5,
	})

	e.rules = append(e.rules, Rule{
		Name: "datacenter_ip",
		Condition: func(ctx *Context) bool {
			return ctx.IsDatacenter
		},
		Score:    0.3,
		Action:   "captcha",
		Priority: 15,
	})

	e.rules = append(e.rules, Rule{
		Name: "suspicious_user_agent",
		Condition: func(ctx *Context) bool {
			ua := ctx.UserAgent
			return ua == "" || len(ua) < 20 || ua == "Mozilla/4.0"
		},
		Score:    0.4,
		Action:   "captcha",
		Priority: 25,
	})

	e.rules = append(e.rules, Rule{
		Name: "rapid_requests",
		Condition: func(ctx *Context) bool {
			return ctx.RequestCount > 100
		},
		Score:    0.6,
		Action:   "block",
		Priority: 1,
	})

	e.rules = append(e.rules, Rule{
		Name: "blocked_before",
		Condition: func(ctx *Context) bool {
			return ctx.BlockedCount > 0
		},
		Score:    0.2,
		Action:   "captcha",
		Priority: 35,
	})
}

func (e *Engine) Evaluate(ctx *Context) *Decision {
	decision := &Decision{
		RiskScore: 0,
		Reasons:   make([]string, 0),
	}

	for _, rule := range e.rules {
		if rule.Condition(ctx) {
			decision.RiskScore += rule.Score
			decision.Reasons = append(decision.Reasons, rule.Name)
		}
	}

	decision.RiskScore = math.Min(decision.RiskScore, 1.0)

	if decision.RiskScore < 0.3 {
		decision.RiskLevel = "low"
		decision.Allow = true
		decision.Action = "allow"
		decision.CaptchaType = ""
	} else if decision.RiskScore < 0.6 {
		decision.RiskLevel = "medium"
		decision.Allow = true
		decision.Action = "captcha"
		decision.CaptchaType = "simple"
	} else if decision.RiskScore < 0.8 {
		decision.RiskLevel = "high"
		decision.Allow = true
		decision.Action = "captcha"
		decision.CaptchaType = "complex"
	} else {
		decision.RiskLevel = "critical"
		decision.Allow = false
		decision.Action = "block"
		decision.CaptchaType = ""
	}

	return decision
}

func (e *Engine) UpdateContext(ctx *Context, isSuccess bool) {
	if isSuccess {
		ctx.SuccessCount++
		now := time.Now()
		ctx.LastSuccessAt = &now
	} else {
		ctx.FailCount++
		now := time.Now()
		ctx.LastAttemptAt = &now
	}

	e.ipCache.Record(ctx.IPAddress, isSuccess)
	e.deviceCache.Record(ctx.Fingerprint, isSuccess)
}

type IPCache struct {
	records map[string]*IPRecord
}

type IPRecord struct {
	RequestCount int
	FailCount    int
	SuccessCount int
	LastSeen     time.Time
}

func NewIPCache() *IPCache {
	return &IPCache{
		records: make(map[string]*IPRecord),
	}
}

func (c *IPCache) Record(ip string, success bool) {
	record, exists := c.records[ip]
	if !exists {
		record = &IPRecord{}
		c.records[ip] = record
	}

	record.RequestCount++
	record.LastSeen = time.Now()

	if success {
		record.SuccessCount++
	} else {
		record.FailCount++
	}
}

func (c *IPCache) Get(ip string) *IPRecord {
	return c.records[ip]
}

type DeviceCache struct {
	records map[string]*DeviceRecord
}

type DeviceRecord struct {
	RequestCount int
	FailCount    int
	SuccessCount int
	FirstSeen    time.Time
	LastSeen     time.Time
	KnownIPs     map[string]bool
}

func NewDeviceCache() *DeviceCache {
	return &DeviceCache{
		records: make(map[string]*DeviceRecord),
	}
}

func (c *DeviceCache) Record(fingerprint string, success bool) {
	record, exists := c.records[fingerprint]
	if !exists {
		record = &DeviceRecord{
			FirstSeen: time.Now(),
			KnownIPs:  make(map[string]bool),
		}
		c.records[fingerprint] = record
	}

	record.RequestCount++
	record.LastSeen = time.Now()

	if success {
		record.SuccessCount++
	} else {
		record.FailCount++
	}
}

func (c *DeviceCache) Get(fingerprint string) *DeviceRecord {
	return c.records[fingerprint]
}

func (c *DeviceCache) AddIP(fingerprint, ip string) {
	if record, exists := c.records[fingerprint]; exists {
		record.KnownIPs[ip] = true
	}
}
