package security

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// IPRange IP范围
type IPRange struct {
	Start net.IP
	End   net.IP
}

// IPManager IP管理器
type IPManager struct {
	whitelist map[string]bool
	blacklist map[string]bool
	ranges    []IPRange
	mu        sync.RWMutex
}

// NewIPManager 创建IP管理器
func NewIPManager() *IPManager {
	return &IPManager{
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
		ranges:    make([]IPRange, 0),
	}
}

// IsAllowed 检查IP是否允许
func (m *IPManager) IsAllowed(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.whitelist[ip] {
		return true
	}

	for _, ipRange := range m.ranges {
		if ipRange.Contains(ip) {
			return true
		}
	}

	return false
}

// IsBlocked 检查IP是否被阻止
func (m *IPManager) IsBlocked(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.blacklist[ip]
}

// AddToWhitelist 添加IP到白名单
func (m *IPManager) AddToWhitelist(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist[strings.TrimSpace(ip)] = true
}

// RemoveFromWhitelist 从白名单移除IP
func (m *IPManager) RemoveFromWhitelist(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.whitelist, strings.TrimSpace(ip))
}

// AddToBlacklist 添加IP到黑名单
func (m *IPManager) AddToBlacklist(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[strings.TrimSpace(ip)] = true
}

// RemoveFromBlacklist 从黑名单移除IP
func (m *IPManager) RemoveFromBlacklist(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.blacklist, strings.TrimSpace(ip))
}

// AddRange 添加IP范围
func (m *IPManager) AddRange(startIP, endIP string) error {
	start := net.ParseIP(startIP)
	if start == nil {
		return fmt.Errorf("无效的起始IP: %s", startIP)
	}

	end := net.ParseIP(endIP)
	if end == nil {
		return fmt.Errorf("无效的结束IP: %s", endIP)
	}

	ipRange := IPRange{
		Start: start,
		End:   end,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.ranges = append(m.ranges, ipRange)

	return nil
}

// AddCIDR 添加CIDR范围
func (m *IPManager) AddCIDR(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("无效的CIDR: %s", cidr)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	start := ipNet.IP
	broadcast := make(net.IP, len(start))
	copy(broadcast, start)

	for i := len(broadcast) - 1; i >= 0; i-- {
		if broadcast[i] < 255 {
			broadcast[i]++
			break
		}
	}

	ipRange := IPRange{
		Start: start,
		End:   broadcast,
	}

	m.ranges = append(m.ranges, ipRange)

	return nil
}

// Contains 检查IP是否在范围内
func (r *IPRange) Contains(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	return bytesCompare(parsedIP, r.Start) >= 0 && bytesCompare(parsedIP, r.End) <= 0
}

// bytesCompare 比较IP字节
func bytesCompare(a, b net.IP) int {
	a4 := a.To4()
	b4 := b.To4()

	if a4 != nil && b4 != nil {
		for i := 0; i < 4; i++ {
			if a4[i] < b4[i] {
				return -1
			}
			if a4[i] > b4[i] {
				return 1
			}
		}
		return 0
	}

	for i := 0; i < 16; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// LoadFromFile 从文件加载黑白名单
func (m *IPManager) LoadFromFile(filepath string, isWhitelist bool) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) > 0 {
			entry := strings.TrimSpace(record[0])
			if entry == "" || strings.HasPrefix(entry, "#") {
				continue
			}

			if strings.Contains(entry, "/") {
				if err := m.AddCIDR(entry); err != nil {
					continue
				}
			} else if isWhitelist {
				m.AddToWhitelist(entry)
			} else {
				m.AddToBlacklist(entry)
			}
		}
	}

	return nil
}

// GetWhitelist 获取白名单
func (m *IPManager) GetWhitelist() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.whitelist))
	for ip := range m.whitelist {
		result = append(result, ip)
	}

	return result
}

// GetBlacklist 获取黑名单
func (m *IPManager) GetBlacklist() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, 0, len(m.blacklist))
	for ip := range m.blacklist {
		result = append(result, ip)
	}

	return result
}

// ClearWhitelist 清空白名单
func (m *IPManager) ClearWhitelist() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist = make(map[string]bool)
}

// ClearBlacklist 清空黑名单
func (m *IPManager) ClearBlacklist() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist = make(map[string]bool)
}

// ClearRanges 清空IP范围
func (m *IPManager) ClearRanges() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ranges = make([]IPRange, 0)
}

// GetClientIP 从请求中获取客户端IP
func GetClientIP(remoteAddr string, xRealIP string, xForwardedFor string) string {
	if xRealIP != "" {
		return xRealIP
	}

	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if remoteAddr != "" {
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			return host
		}
		return remoteAddr
	}

	return ""
}

// IPReputation IP信誉评分
type IPReputation struct {
	IP           string
	Score        float64
	LastSeen     time.Time
	FailedLogins int
	SuccessLogins int
	Country      string
	IsVPN        bool
	IsProxy      bool
	IsTor        bool
}

// IPReputationManager IP信誉管理器
type IPReputationManager struct {
	reputations map[string]*IPReputation
	mu          sync.RWMutex
}

// NewIPReputationManager 创建IP信誉管理器
func NewIPReputationManager() *IPReputationManager {
	return &IPReputationManager{
		reputations: make(map[string]*IPReputation),
	}
}

// GetReputation 获取IP信誉
func (m *IPReputationManager) GetReputation(ip string) *IPReputation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if rep, exists := m.reputations[ip]; exists {
		return rep
	}

	return &IPReputation{
		IP:    ip,
		Score: 50,
	}
}

// RecordSuccess 记录成功
func (m *IPReputationManager) RecordSuccess(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, exists := m.reputations[ip]
	if !exists {
		rep = &IPReputation{
			IP:    ip,
			Score: 50,
		}
		m.reputations[ip] = rep
	}

	rep.SuccessLogins++
	rep.Score = min(100, rep.Score+5)
	rep.LastSeen = time.Now()
}

// RecordFailure 记录失败
func (m *IPReputationManager) RecordFailure(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, exists := m.reputations[ip]
	if !exists {
		rep = &IPReputation{
			IP:    ip,
			Score: 50,
		}
		m.reputations[ip] = rep
	}

	rep.FailedLogins++
	rep.Score = max(0, rep.Score-10)
	rep.LastSeen = time.Now()
}

// IsTrusted 判断IP是否可信
func (m *IPReputationManager) IsTrusted(ip string) bool {
	rep := m.GetReputation(ip)
	return rep.Score >= 70
}

// IsSuspicious 判断IP是否可疑
func (m *IPReputationManager) IsSuspicious(ip string) bool {
	rep := m.GetReputation(ip)
	return rep.Score < 30
}

// IsBlocked 判断IP是否应该被阻止
func (m *IPReputationManager) IsBlocked(ip string) bool {
	rep := m.GetReputation(ip)
	return rep.Score < 10 || rep.FailedLogins > 10
}

// ResetReputation 重置IP信誉
func (m *IPReputationManager) ResetReputation(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reputations, ip)
}

// CleanupOldReputations 清理旧的信誉记录
func (m *IPReputationManager) CleanupOldReputations(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for ip, rep := range m.reputations {
		if rep.LastSeen.Before(cutoff) {
			delete(m.reputations, ip)
		}
	}
}
