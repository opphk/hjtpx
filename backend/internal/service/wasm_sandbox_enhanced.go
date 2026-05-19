package service

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WasmSecurityLevel 安全级别
type WasmSecurityLevel int

const (
	WasmSecurityLevelLow WasmSecurityLevel = iota
	WasmSecurityLevelMedium
	WasmSecurityLevelHigh
	WasmSecurityLevelCritical
)

// WasmResourceType 资源类型
type WasmResourceType string

const (
	WasmResourceCPU    WasmResourceType = "cpu"
	WasmResourceMemory WasmResourceType = "memory"
	WasmResourceNetwork WasmResourceType = "network"
	WasmResourceFile   WasmResourceType = "file"
	WasmResourceAPI   WasmResourceType = "api"
)

// WASMSandboxEnhanced 增强的 WASM 沙箱
type WASMSandboxEnhanced struct {
	mu                sync.RWMutex
	modules           map[string]*WasmSandboxModule
	policies          map[string]*WasmSecurityPolicy
	resourceManager   *WasmResourceManager
	auditLog          *WasmAuditLog
	securityLevel     WasmSecurityLevel
	activeInstances  atomic.Int32
	maxInstances     int
	whitelist        map[string]bool
	blacklist        map[string]bool
}

// WasmSandboxModule 沙箱中的模块
type WasmSandboxModule struct {
	ID             string
	Name           string
	Hash           string
	LoadedAt       time.Time
	LastUsed       time.Time
	UseCount       int64
	ResourceLimits *WasmResourceLimits
	State          WasmModuleState
	AllowedAPIs    []string
	AllowedHosts   []string
	Metadata       map[string]string
}

// WasmModuleState 模块状态
type WasmModuleState string

const (
	WasmModuleStateLoaded    WasmModuleState = "loaded"
	WasmModuleStateRunning   WasmModuleState = "running"
	WasmModuleStatePaused    WasmModuleState = "paused"
	WasmModuleStateTerminated WasmModuleState = "terminated"
)

// WasmSecurityPolicy 安全策略
type WasmSecurityPolicy struct {
	ID              string
	Name            string
	Level           WasmSecurityLevel
	ResourceLimits *WasmResourceLimits
	APIWhitelist    []string
	APIBlacklist    []string
	HostWhitelist   []string
	HostBlacklist   []string
	ExecutionTime   time.Duration
	IdleTimeout     time.Duration
	MemoryLimit     int64
	CPULimit        float64
	NetworkAllowed  bool
	FileAccessAllowed bool
	CreatedAt       time.Time
}

// WasmResourceLimits 资源限制
type WasmResourceLimits struct {
	MaxMemory      int64         // 最大内存（字节）
	MaxCPUPercent  float64       // 最大 CPU 百分比
	MaxExecution   time.Duration // 最大执行时间
	MaxIdleTime    time.Duration // 最大空闲时间
	MaxAPIRequests int           // 最大 API 请求数
	MaxNetworkIO   int64         // 最大网络 IO
	MaxFileIO      int64         // 最大文件 IO
}

// WasmResourceManager 资源管理器
type WasmResourceManager struct {
	mu              sync.Mutex
	resourceUsage   map[string]*WasmResourceUsage
	quota         *WasmResourceLimits
	monitorInterval time.Duration
	stopChan      chan struct{}
}

// WasmResourceUsage 资源使用情况
type WasmResourceUsage struct {
	MemoryUsed    int64
	CPUUsed     float64
	ExecutionTime time.Duration
	APIRequests int
	NetworkIO   int64
	FileIO      int64
	StartTime    time.Time
	LastAccess  time.Time
}

// WasmAuditLog 审计日志
type WasmAuditLog struct {
	mu       sync.Mutex
	entries  *list.List
	maxSize  int
}

// WasmAuditEntry 审计条目
type WasmAuditEntry struct {
	Timestamp  time.Time
	ModuleID   string
	EventType  string
	Resource   string
	Action     string
	Success     bool
	Error      string
	Details    map[string]interface{}
}

// NewWASMSandboxEnhanced 创建增强的沙箱
func NewWASMSandboxEnhanced() *WASMSandboxEnhanced {
	sandbox := &WASMSandboxEnhanced{
		modules:          make(map[string]*WasmSandboxModule),
		policies:         make(map[string]*WasmSecurityPolicy),
		resourceManager:  NewWasmResourceManager(),
		auditLog:         NewWasmAuditLog(10000),
		securityLevel:    WasmSecurityLevelHigh,
		maxInstances:     100,
		whitelist:        make(map[string]bool),
		blacklist:        make(map[string]bool),
	}

	// 创建默认策略
	sandbox.createDefaultPolicies()

	return sandbox
}

// NewWasmResourceManager 创建资源管理器
func NewWasmResourceManager() *WasmResourceManager {
	rm := &WasmResourceManager{
		resourceUsage:  make(map[string]*WasmResourceUsage),
		quota: &WasmResourceLimits{
			MaxMemory:      1024 * 1024 * 1024, // 1GB
			MaxCPUPercent:  80.0, // 80%
			MaxExecution:   30 * time.Minute,
			MaxIdleTime:    5 * time.Minute,
			MaxAPIRequests: 10000,
			MaxNetworkIO:   100 * 1024 * 1024, // 100MB
			MaxFileIO:      50 * 1024 * 1024, // 50MB
		},
		monitorInterval: 5 * time.Second,
		stopChan:        make(chan struct{}),
	}

	go rm.startMonitoring()

	return rm
}

// NewWasmAuditLog 创建审计日志
func NewWasmAuditLog(maxSize int) *WasmAuditLog {
	return &WasmAuditLog{
		entries: list.New(),
		maxSize: maxSize,
	}
}

// createDefaultPolicies 创建默认策略
func (s *WASMSandboxEnhanced) createDefaultPolicies() {
	defaultPolicy := &WasmSecurityPolicy{
		ID:              "default",
		Name:            "Default Security Policy",
		Level:           WasmSecurityLevelHigh,
		ResourceLimits: &WasmResourceLimits{
			MaxMemory:      256 * 1024 * 1024, // 256MB
			MaxCPUPercent:  50.0, // 50%
			MaxExecution:   10 * time.Minute,
			MaxIdleTime:    2 * time.Minute,
			MaxAPIRequests: 1000,
			MaxNetworkIO:   10 * 1024 * 1024, // 10MB
			MaxFileIO:      5 * 1024 * 1024, // 5MB
		},
		APIWhitelist:    []string{"math", "string", "crypto"},
		APIBlacklist:    []string{"eval", "unsafe", "process"},
		HostWhitelist:   []string{"localhost", "127.0.0.1"},
		HostBlacklist:   []string{"malicious.com"},
		ExecutionTime:   10 * time.Minute,
		IdleTimeout:     2 * time.Minute,
		MemoryLimit:     256 * 1024 * 1024,
		CPULimit:        50.0,
		NetworkAllowed:  false,
		FileAccessAllowed: false,
		CreatedAt:       time.Now(),
	}

	s.policies["default"] = defaultPolicy
}

// LoadModule 加载模块到沙箱
func (s *WASMSandboxEnhanced) LoadModule(id string, name string, data []byte, policyID string) (*WasmSandboxModule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeInstances.Load() >= int32(s.maxInstances) {
		return nil, errors.New("max instances reached")
	}

	// 检查黑名单
	if s.blacklist[id] {
		return nil, errors.New("module is blacklisted")
	}

	// 计算模块哈希
	hash := s.computeHash(data)

	// 获取策略
	policy, exists := s.policies[policyID]
	if !exists {
		policy = s.policies["default"]
	}

	module := &WasmSandboxModule{
		ID:             id,
		Name:           name,
		Hash:           hash,
		LoadedAt:       time.Now(),
		LastUsed:       time.Now(),
		ResourceLimits: policy.ResourceLimits,
		State:          WasmModuleStateLoaded,
		AllowedAPIs:    policy.APIWhitelist,
		AllowedHosts:   policy.HostWhitelist,
		Metadata:       make(map[string]string),
	}

	s.modules[id] = module
	s.activeInstances.Add(1)

	// 初始化资源使用
	s.resourceManager.initUsage(id)

	// 记录审计日志
	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  id,
		EventType: "load",
		Resource:  "module",
		Action:    "load",
		Success:   true,
	})

	return module, nil
}

// UnloadModule 卸载模块
func (s *WASMSandboxEnhanced) UnloadModule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	module, exists := s.modules[id]
	if !exists {
		return errors.New("module not found")
	}

	module.State = WasmModuleStateTerminated
	delete(s.modules, id)
	s.activeInstances.Add(-1)

	// 清理资源使用
	s.resourceManager.cleanupUsage(id)

	// 记录审计日志
	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  id,
		EventType: "unload",
		Resource:  "module",
		Action:    "unload",
		Success:   true,
	})

	return nil
}

// ExecuteModule 执行模块
func (s *WASMSandboxEnhanced) ExecuteModule(id string, input []byte) ([]byte, error) {
	s.mu.RLock()
	module, exists := s.modules[id]
	s.mu.RUnlock()

	if !exists {
		return nil, errors.New("module not found")
	}

	if module.State == WasmModuleStateTerminated {
		return nil, errors.New("module is terminated")
	}

	// 检查资源限制
	if !s.resourceManager.checkLimits(id) {
		s.auditLog.AddEntry(WasmAuditEntry{
			Timestamp: time.Now(),
			ModuleID:  id,
			EventType: "execute",
			Resource:  "resource",
			Action:    "check",
			Success:   false,
			Error:     "resource limits exceeded",
		})
		return nil, errors.New("resource limits exceeded")
	}

	// 更新模块状态
	s.mu.Lock()
	module.State = WasmModuleStateRunning
	module.LastUsed = time.Now()
	module.UseCount++
	s.mu.Unlock()

	// 开始执行
	start := time.Now()
	s.resourceManager.startExecution(id)

	// 这里是模拟执行，实际应该调用真正的 WASM 运行时
	// 为了安全，我们不会实际执行未经验证的代码
	result := s.simulateExecution(input)

	// 结束执行
	s.resourceManager.endExecution(id, start)

	// 更新状态
	s.mu.Lock()
	module.State = WasmModuleStateLoaded
	s.mu.Unlock()

	// 记录审计日志
	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  id,
		EventType: "execute",
		Resource:  "module",
		Action:    "execute",
		Success:   true,
	})

	return result, nil
}

// simulateExecution 模拟执行
func (s *WASMSandboxEnhanced) simulateExecution(input []byte) []byte {
	// 这里只是一个模拟，实际上应该集成真正的 WASM 运行时
	// 例如使用 wazero 或其他 Go 的 WASM 运行时库
	return input
}

// PauseModule 暂停模块
func (s *WASMSandboxEnhanced) PauseModule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	module, exists := s.modules[id]
	if !exists {
		return errors.New("module not found")
	}

	if module.State != WasmModuleStateRunning {
		return errors.New("module is not running")
	}

	module.State = WasmModuleStatePaused

	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  id,
		EventType: "pause",
		Resource:  "module",
		Action:    "pause",
		Success:   true,
	})

	return nil
}

// ResumeModule 恢复模块
func (s *WASMSandboxEnhanced) ResumeModule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	module, exists := s.modules[id]
	if !exists {
		return errors.New("module not found")
	}

	if module.State != WasmModuleStatePaused {
		return errors.New("module is not paused")
	}

	module.State = WasmModuleStateRunning
	module.LastUsed = time.Now()

	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  id,
		EventType: "resume",
		Resource:  "module",
		Action:    "resume",
		Success:   true,
	})

	return nil
}

// AddPolicy 添加安全策略
func (s *WASMSandboxEnhanced) AddPolicy(policy *WasmSecurityPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.policies[policy.ID] = policy
}

// GetPolicy 获取策略
func (s *WASMSandboxEnhanced) GetPolicy(id string) (*WasmSecurityPolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, exists := s.policies[id]
	return policy, exists
}

// SetSecurityLevel 设置安全级别
func (s *WASMSandboxEnhanced) SetSecurityLevel(level WasmSecurityLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.securityLevel = level
}

// AddToWhitelist 添加到白名单
func (s *WASMSandboxEnhanced) AddToWhitelist(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.whitelist[id] = true
	delete(s.blacklist, id)
}

// AddToBlacklist 添加到黑名单
func (s *WASMSandboxEnhanced) AddToBlacklist(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blacklist[id] = true
	delete(s.whitelist, id)
}

// IsWhitelisted 检查是否在白名单
func (s *WASMSandboxEnhanced) IsWhitelisted(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.whitelist[id]
}

// IsBlacklisted 检查是否在黑名单
func (s *WASMSandboxEnhanced) IsBlacklisted(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.blacklist[id]
}

// GetModel 获取模块
func (s *WASMSandboxEnhanced) GetModel(id string) (*WasmSandboxModule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	module, exists := s.modules[id]
	return module, exists
}

// ListModels 列出所有模块
func (s *WASMSandboxEnhanced) ListModels() []*WasmSandboxModule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	modules := make([]*WasmSandboxModule, 0, len(s.modules))
	for _, module := range s.modules {
		modules = append(modules, module)
	}
	return modules
}

// GetAuditLog 获取审计日志
func (s *WASMSandboxEnhanced) GetAuditLog(limit int) []*WasmAuditEntry {
	return s.auditLog.GetEntries(limit)
}

// GetResourceUsage 获取资源使用情况
func (s *WASMSandboxEnhanced) GetResourceUsage(id string) (*WasmResourceUsage, bool) {
	return s.resourceManager.getUsage(id)
}

// computeHash 计算哈希
func (s *WASMSandboxEnhanced) computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// startMonitoring 开始监控
func (rm *WasmResourceManager) startMonitoring() {
	ticker := time.NewTicker(rm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.checkResources()
		case <-rm.stopChan:
			return
		}
	}
}

// checkResources 检查资源
func (rm *WasmResourceManager) checkResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	for _, usage := range rm.resourceUsage {
		// 检查空闲时间
		if now.Sub(usage.LastAccess) > rm.quota.MaxIdleTime {
			// 可以考虑自动卸载空闲模块
		}
	}
}

// initUsage 初始化使用情况
func (rm *WasmResourceManager) initUsage(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.resourceUsage[id] = &WasmResourceUsage{
		StartTime:  time.Now(),
		LastAccess: time.Now(),
	}
}

// cleanupUsage 清理使用情况
func (rm *WasmResourceManager) cleanupUsage(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.resourceUsage, id)
}

// getUsage 获取使用情况
func (rm *WasmResourceManager) getUsage(id string) (*WasmResourceUsage, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	usage, exists := rm.resourceUsage[id]
	return usage, exists
}

// checkLimits 检查限制
func (rm *WasmResourceManager) checkLimits(id string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 这里应该检查实际的资源使用情况
	// 简化版本，总是返回 true
	return true
}

// startExecution 开始执行
func (rm *WasmResourceManager) startExecution(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if usage, exists := rm.resourceUsage[id]; exists {
		usage.LastAccess = time.Now()
	}
}

// endExecution 结束执行
func (rm *WasmResourceManager) endExecution(id string, start time.Time) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if usage, exists := rm.resourceUsage[id]; exists {
		usage.ExecutionTime += time.Since(start)
		usage.LastAccess = time.Now()
	}
}

// AddEntry 添加条目
func (al *WasmAuditLog) AddEntry(entry WasmAuditEntry) {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.entries.PushBack(entry)

	// 限制大小
	for al.entries.Len() > al.maxSize {
		al.entries.Remove(al.entries.Front())
	}
}

// GetEntries 获取条目
func (al *WasmAuditLog) GetEntries(limit int) []*WasmAuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	if limit <= 0 || limit > al.entries.Len() {
		limit = al.entries.Len()
	}

	entries := make([]*WasmAuditEntry, 0, limit)
	count := 0

	// 从后往前获取最新的条目
	for e := al.entries.Back(); e != nil && count < limit; e = e.Prev() {
		entry := e.Value.(WasmAuditEntry)
		entries = append(entries, &entry)
		count++
	}

	return entries
}

// Close 关闭沙箱
func (s *WASMSandboxEnhanced) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 终止所有模块
	for id := range s.modules {
		s.modules[id].State = WasmModuleStateTerminated
	}

	// 停止资源监控
	close(s.resourceManager.stopChan)
}

// ValidateModule 验证模块
func (s *WASMSandboxEnhanced) ValidateModule(data []byte) (bool, error) {
	// 这里可以添加模块验证逻辑
	// 例如检查 WASM 魔术数字、验证签名等
	if len(data) < 8 {
		return false, errors.New("invalid module size")
	}

	// 检查 WASM 魔术数字
	magic := []byte{0x00, 0x61, 0x73, 0x6d} // \0asm
	for i := 0; i < 4; i++ {
		if data[i] != magic[i] {
			return false, errors.New("invalid WASM magic number")
		}
	}

	return true, nil
}

// SetModuleMetadata 设置模块元数据
func (s *WASMSandboxEnhanced) SetModuleMetadata(id string, key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	module, exists := s.modules[id]
	if !exists {
		return errors.New("module not found")
	}

	module.Metadata[key] = value
	return nil
}

// GetModuleMetadata 获取模块元数据
func (s *WASMSandboxEnhanced) GetModuleMetadata(id string, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	module, exists := s.modules[id]
	if !exists {
		return "", false
	}

	value, ok := module.Metadata[key]
	return value, ok
}

// GetStats 获取沙箱统计信息
func (s *WASMSandboxEnhanced) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_modules": len(s.modules),
		"max_instances":   s.maxInstances,
		"security_level":   GetWasmSecurityLevelName(s.securityLevel),
		"policies_count":  len(s.policies),
		"whitelist_size": len(s.whitelist),
		"blacklist_size": len(s.blacklist),
		"audit_log_size": s.auditLog.entries.Len(),
	}
}

// CheckAPIAccess 检查 API 访问权限
func (s *WASMSandboxEnhanced) CheckAPIAccess(moduleID string, apiName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	module, exists := s.modules[moduleID]
	if !exists {
		return false
	}

	// 检查黑名单
	for _, blacklisted := range s.policies["default"].APIBlacklist {
		if apiName == blacklisted {
			s.auditLog.AddEntry(WasmAuditEntry{
				Timestamp: time.Now(),
				ModuleID:  moduleID,
				EventType: "api_access",
				Resource:  apiName,
				Action:    "deny",
				Success:   false,
				Error:     "API is blacklisted",
			})
			return false
		}
	}

	// 检查白名单
	for _, allowed := range module.AllowedAPIs {
		if apiName == allowed {
			return true
		}
	}

	// 默认拒绝
	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  moduleID,
		EventType: "api_access",
		Resource:  apiName,
		Action:    "deny",
		Success:   false,
		Error:     "API not in whitelist",
	})
	return false
}

// CheckHostAccess 检查主机访问权限
func (s *WASMSandboxEnhanced) CheckHostAccess(moduleID string, host string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	module, exists := s.modules[moduleID]
	if !exists {
		return false
	}

	// 检查黑名单
	for _, blacklisted := range s.policies["default"].HostBlacklist {
		if host == blacklisted {
			s.auditLog.AddEntry(WasmAuditEntry{
				Timestamp: time.Now(),
				ModuleID:  moduleID,
				EventType: "host_access",
				Resource:  host,
				Action:    "deny",
				Success:   false,
				Error:     "Host is blacklisted",
			})
			return false
		}
	}

	// 检查白名单
	for _, allowed := range module.AllowedHosts {
		if host == allowed {
			return true
		}
	}

	// 默认拒绝
	s.auditLog.AddEntry(WasmAuditEntry{
		Timestamp: time.Now(),
		ModuleID:  moduleID,
		EventType: "host_access",
		Resource:  host,
		Action:    "deny",
		Success:   false,
		Error:     "Host not in whitelist",
	})
	return false
}

// GetWasmSecurityLevelName 获取安全级别名称
func GetWasmSecurityLevelName(level WasmSecurityLevel) string {
	switch level {
	case WasmSecurityLevelLow:
		return "Low"
	case WasmSecurityLevelMedium:
		return "Medium"
	case WasmSecurityLevelHigh:
		return "High"
	case WasmSecurityLevelCritical:
		return "Critical"
	default:
		return fmt.Sprintf("Unknown (%d)", level)
	}
}
