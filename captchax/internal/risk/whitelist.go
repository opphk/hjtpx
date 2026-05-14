package risk

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Whitelist struct {
	client       *redis.Client
	keyPrefix    string
	memStore     *MemoryWhitelist
	useRedis     bool
}

type WhitelistEntry struct {
	Type      WhitelistType
	Value     string
	ExpiresAt time.Time
	Reason    string
}

type WhitelistType string

const (
	WhitelistTypeUser    WhitelistType = "user"
	WhitelistTypeIP      WhitelistType = "ip"
	WhitelistTypeDomain  WhitelistType = "domain"
)

type WhitelistConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	KeyPrefix     string
	MemoryOnly    bool
}

func NewWhitelist(cfg *WhitelistConfig) (*Whitelist, error) {
	w := &Whitelist{
		keyPrefix: cfg.KeyPrefix,
		memStore:  NewMemoryWhitelist(),
	}

	if !cfg.MemoryOnly && cfg.RedisAddr != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err == nil {
			w.client = client
			w.useRedis = true
		}
	}

	if w.keyPrefix == "" {
		w.keyPrefix = "captchax:whitelist:"
	}

	return w, nil
}

func (w *Whitelist) IsWhiteListed(ctx context.Context, ip string, domain string, userID string) bool {
	if w.memStore.IsWhiteListed(ip, domain, userID) {
		return true
	}

	if w.useRedis {
		return w.checkRedisWhitelist(ctx, ip, domain, userID)
	}

	return false
}

func (w *Whitelist) checkRedisWhitelist(ctx context.Context, ip string, domain string, userID string) bool {
	if ip != "" {
		key := w.keyPrefix + "ip:" + ip
		exists, _ := w.client.Exists(ctx, key).Result()
		if exists > 0 {
			return true
		}
	}

	if domain != "" {
		key := w.keyPrefix + "domain:" + domain
		exists, _ := w.client.Exists(ctx, key).Result()
		if exists > 0 {
			return true
		}
	}

	if userID != "" {
		key := w.keyPrefix + "user:" + userID
		exists, _ := w.client.Exists(ctx, key).Result()
		if exists > 0 {
			return true
		}
	}

	return false
}

func (w *Whitelist) AddToWhitelist(ctx context.Context, entry *WhitelistEntry) error {
	w.memStore.Add(entry)

	if w.useRedis {
		key := w.keyPrefix + string(entry.Type) + ":" + entry.Value
		if entry.ExpiresAt.IsZero() {
			return w.client.Set(ctx, key, entry.Reason, 0).Err()
		}
		ttl := time.Until(entry.ExpiresAt)
		if ttl > 0 {
			return w.client.Set(ctx, key, entry.Reason, ttl).Err()
		}
	}

	return nil
}

func (w *Whitelist) RemoveFromWhitelist(ctx context.Context, wType WhitelistType, value string) error {
	w.memStore.Remove(wType, value)

	if w.useRedis {
		key := w.keyPrefix + string(wType) + ":" + value
		return w.client.Del(ctx, key).Err()
	}

	return nil
}

func (w *Whitelist) IsIPWhiteListed(ctx context.Context, ip string) bool {
	if w.memStore.IsIPWhiteListed(ip) {
		return true
	}

	if w.useRedis {
		key := w.keyPrefix + "ip:" + ip
		exists, _ := w.client.Exists(ctx, key).Result()
		return exists > 0
	}

	return false
}

func (w *Whitelist) IsDomainWhiteListed(ctx context.Context, domain string) bool {
	if w.memStore.IsDomainWhiteListed(domain) {
		return true
	}

	if w.useRedis {
		key := w.keyPrefix + "domain:" + domain
		exists, _ := w.client.Exists(ctx, key).Result()
		if exists > 0 {
			return true
		}

		domainParts := strings.Split(domain, ".")
		for i := 1; i < len(domainParts); i++ {
			wildcardDomain := strings.Join(domainParts[i:], ".")
			wildcardKey := w.keyPrefix + "domain:*." + wildcardDomain
			exists, _ := w.client.Exists(ctx, wildcardKey).Result()
			if exists > 0 {
				return true
			}
		}
	}

	return false
}

func (w *Whitelist) IsUserWhiteListed(ctx context.Context, userID string) bool {
	if w.memStore.IsUserWhiteListed(userID) {
		return true
	}

	if w.useRedis {
		key := w.keyPrefix + "user:" + userID
		exists, _ := w.client.Exists(ctx, key).Result()
		return exists > 0
	}

	return false
}

func (w *Whitelist) AddIPToWhitelist(ctx context.Context, ip string, reason string, expiresAt time.Time) error {
	return w.AddToWhitelist(ctx, &WhitelistEntry{
		Type:      WhitelistTypeIP,
		Value:     ip,
		ExpiresAt: expiresAt,
		Reason:    reason,
	})
}

func (w *Whitelist) AddDomainToWhitelist(ctx context.Context, domain string, reason string, expiresAt time.Time) error {
	return w.AddToWhitelist(ctx, &WhitelistEntry{
		Type:      WhitelistTypeDomain,
		Value:     domain,
		ExpiresAt: expiresAt,
		Reason:    reason,
	})
}

func (w *Whitelist) AddUserToWhitelist(ctx context.Context, userID string, reason string, expiresAt time.Time) error {
	return w.AddToWhitelist(ctx, &WhitelistEntry{
		Type:      WhitelistTypeUser,
		Value:     userID,
		ExpiresAt: expiresAt,
		Reason:    reason,
	})
}

func (w *Whitelist) GetWhitelistEntries(ctx context.Context) ([]WhitelistEntry, error) {
	var entries []WhitelistEntry

	entries = append(entries, w.memStore.GetAll()...)

	if w.useRedis {
		pattern := w.keyPrefix + "*"
		keys, err := w.client.Keys(ctx, pattern).Result()
		if err == nil {
			for _, key := range keys {
				parts := strings.Split(key, ":")
				if len(parts) >= 3 {
					entry := WhitelistEntry{
						Type:  WhitelistType(parts[len(parts)-2]),
						Value: parts[len(parts)-1],
					}
					entries = append(entries, entry)
				}
			}
		}
	}

	return entries, nil
}

func (w *Whitelist) Close() error {
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}

type MemoryWhitelist struct {
	mu      sync.RWMutex
	entries map[WhitelistType]map[string]*WhitelistEntry
}

func NewMemoryWhitelist() *MemoryWhitelist {
	return &MemoryWhitelist{
		entries: map[WhitelistType]map[string]*WhitelistEntry{
			WhitelistTypeUser:   {},
			WhitelistTypeIP:     {},
			WhitelistTypeDomain: {},
		},
	}
}

func (m *MemoryWhitelist) IsWhiteListed(ip string, domain string, userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if userID != "" {
		if entry, ok := m.entries[WhitelistTypeUser][userID]; ok {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
				return true
			}
		}
	}

	if ip != "" {
		if entry, ok := m.entries[WhitelistTypeIP][ip]; ok {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
				return true
			}
		}
	}

	if domain != "" {
		if entry, ok := m.entries[WhitelistTypeDomain][domain]; ok {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
				return true
			}
		}
	}

	return false
}

func (m *MemoryWhitelist) IsIPWhiteListed(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[WhitelistTypeIP][ip]; ok {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
			return true
		}
	}
	return false
}

func (m *MemoryWhitelist) IsDomainWhiteListed(domain string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[WhitelistTypeDomain][domain]; ok {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
			return true
		}
	}

	domainParts := strings.Split(domain, ".")
	for i := 1; i < len(domainParts); i++ {
		wildcardDomain := "*." + strings.Join(domainParts[i:], ".")
		if entry, ok := m.entries[WhitelistTypeDomain][wildcardDomain]; ok {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
				return true
			}
		}
	}

	return false
}

func (m *MemoryWhitelist) IsUserWhiteListed(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[WhitelistTypeUser][userID]; ok {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(time.Now()) {
			return true
		}
	}
	return false
}

func (m *MemoryWhitelist) Add(entry *WhitelistEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[entry.Type][entry.Value] = entry
}

func (m *MemoryWhitelist) Remove(wType WhitelistType, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries[wType], value)
}

func (m *MemoryWhitelist) GetAll() []WhitelistEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []WhitelistEntry
	now := time.Now()

	for wType, typeEntries := range m.entries {
		for _, entry := range typeEntries {
			if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(now) {
				entries = append(entries, WhitelistEntry{
					Type:      wType,
					Value:     entry.Value,
					ExpiresAt: entry.ExpiresAt,
					Reason:    entry.Reason,
				})
			}
		}
	}

	return entries
}
