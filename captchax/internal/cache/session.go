package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionConfig struct {
	TTL            time.Duration
	CookieName     string
	CookieDomain   string
	CookiePath     string
	Secure         bool
	HttpOnly       bool
	SameSite       string
	MaxSessions    int
	RefreshEnabled bool
	RefreshBefore  time.Duration
}

type Session struct {
	ID        string
	UserID    string
	Data      map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	IP        string
	UserAgent string
}

type SessionStore interface {
	Create(ctx context.Context, data map[string]interface{}) (*Session, error)
	Get(ctx context.Context, id string) (*Session, bool, error)
	Update(ctx context.Context, id string, data map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	Refresh(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]*Session, error)
	DeleteByUser(ctx context.Context, userID string) error
	Exists(ctx context.Context, id string) (bool, error)
}

type RedisSessionStoreInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	SRem(ctx context.Context, key string, members ...interface{}) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
}

type RedisSessionStore struct {
	redis    RedisSessionStoreInterface
	config   *SessionConfig
	prefix   string
	locker   *SessionLocker
}

type SessionLocker struct {
	redis RedisSessionStoreInterface
}

type DistributedSessionStore struct {
	redis       RedisSessionStoreInterface
	config      *SessionConfig
	prefix      string
	userIndex   map[string]map[string]bool
	mu          sync.RWMutex
}

func NewRedisSessionStore(redis RedisSessionStoreInterface, cfg *SessionConfig) *RedisSessionStore {
	if cfg == nil {
		cfg = &SessionConfig{
			TTL:            24 * time.Hour,
			CookieName:     "captchax_session",
			CookiePath:     "/",
			Secure:         true,
			HttpOnly:       true,
			SameSite:       "Strict",
			MaxSessions:    10,
			RefreshEnabled: true,
			RefreshBefore:  5 * time.Minute,
		}
	}

	return &RedisSessionStore{
		redis:   redis,
		config:  cfg,
		prefix:  "captchax:session:",
		locker: &SessionLocker{redis: redis},
	}
}

func (s *RedisSessionStore) Create(ctx context.Context, data map[string]interface{}) (*Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &Session{
		ID:        sessionID,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(s.config.TTL),
	}

	if userID, ok := data["user_id"].(string); ok {
		session.UserID = userID
	}

	key := s.prefix + sessionID
	jsonData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.redis.Set(ctx, key, string(jsonData), s.config.TTL); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	if session.UserID != "" {
		userKey := s.prefix + "user:" + session.UserID
		s.redis.SAdd(ctx, userKey, sessionID)
		s.redis.Expire(ctx, userKey, s.config.TTL)
	}

	return session, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, id string) (*Session, bool, error) {
	key := s.prefix + id

	data, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, false, nil
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if s.config.RefreshEnabled && time.Until(session.ExpiresAt) < s.config.RefreshBefore {
		session.ExpiresAt = time.Now().Add(s.config.TTL)
		jsonData, _ := json.Marshal(session)
		s.redis.Set(ctx, key, string(jsonData), s.config.TTL)
	}

	return &session, true, nil
}

func (s *RedisSessionStore) Update(ctx context.Context, id string, data map[string]interface{}) error {
	session, exists, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session not found")
	}

	for k, v := range data {
		session.Data[k] = v
	}
	session.UpdatedAt = time.Now()

	jsonData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := s.prefix + id
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = s.config.TTL
	}

	return s.redis.Set(ctx, key, string(jsonData), ttl)
}

func (s *RedisSessionStore) Delete(ctx context.Context, id string) error {
	session, exists, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	key := s.prefix + id
	if err := s.redis.Del(ctx, key); err != nil {
		return err
	}

	if exists && session.UserID != "" {
		userKey := s.prefix + "user:" + session.UserID
		s.redis.SRem(ctx, userKey, id)
	}

	return nil
}

func (s *RedisSessionStore) Refresh(ctx context.Context, id string) error {
	session, exists, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = time.Now().Add(s.config.TTL)
	session.UpdatedAt = time.Now()

	jsonData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := s.prefix + id
	return s.redis.Set(ctx, key, string(jsonData), s.config.TTL)
}

func (s *RedisSessionStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	userKey := s.prefix + "user:" + userID

	sessionIDs, err := s.redis.SMembers(ctx, userKey)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, exists, err := s.Get(ctx, id)
		if err != nil {
			continue
		}
		if exists {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *RedisSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	sessions, err := s.ListByUser(ctx, userID)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if err := s.Delete(ctx, session.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *RedisSessionStore) Exists(ctx context.Context, id string) (bool, error) {
	key := s.prefix + id
	count, err := s.redis.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *RedisSessionStore) Lock(ctx context.Context, id string, timeout time.Duration) (bool, error) {
	lockKey := s.prefix + "lock:" + id
	return s.redis.SetNX(ctx, lockKey, "1", timeout)
}

func (s *RedisSessionStore) Unlock(ctx context.Context, id string) error {
	lockKey := s.prefix + "lock:" + id
	return s.redis.Del(ctx, lockKey)
}

func (sl *SessionLocker) Lock(ctx context.Context, key string, timeout time.Duration) (bool, error) {
	return sl.redis.SetNX(ctx, "lock:"+key, "1", timeout)
}

func (sl *SessionLocker) Unlock(ctx context.Context, key string) error {
	return sl.redis.Del(ctx, "lock:"+key)
}

func (s *RedisSessionStore) GetConfig() *SessionConfig {
	return s.config
}

type MultiNodeSessionManager struct {
	localStore  *LocalSessionStore
	redisStore  *RedisSessionStore
	config      *SessionConfig
}

type LocalSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	index    map[string][]string
	config   *SessionConfig
}

func NewLocalSessionStore(cfg *SessionConfig) *LocalSessionStore {
	if cfg == nil {
		cfg = &SessionConfig{
			TTL:       30 * time.Minute,
			MaxSessions: 1000,
		}
	}

	return &LocalSessionStore{
		sessions: make(map[string]*Session),
		index:    make(map[string][]string),
		config:   cfg,
	}
}

func (l *LocalSessionStore) Create(ctx context.Context, data map[string]interface{}) (*Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()

	session := &Session{
		ID:        sessionID,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(l.config.TTL),
	}

	if userID, ok := data["user_id"].(string); ok {
		session.UserID = userID
		l.addToIndex(userID, sessionID)
	}

	l.mu.Lock()
	l.sessions[sessionID] = session
	l.mu.Unlock()

	return session, nil
}

func (l *LocalSessionStore) Get(ctx context.Context, id string) (*Session, bool, error) {
	l.mu.RLock()
	session, exists := l.sessions[id]
	l.mu.RUnlock()

	if !exists {
		return nil, false, nil
	}

	if time.Now().After(session.ExpiresAt) {
		l.Delete(ctx, id)
		return nil, false, nil
	}

	return session, true, nil
}

func (l *LocalSessionStore) Update(ctx context.Context, id string, data map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, exists := l.sessions[id]
	if !exists {
		return fmt.Errorf("session not found")
	}

	for k, v := range data {
		session.Data[k] = v
	}
	session.UpdatedAt = time.Now()

	return nil
}

func (l *LocalSessionStore) Delete(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if session, exists := l.sessions[id]; exists && session.UserID != "" {
		l.removeFromIndex(session.UserID, id)
	}

	delete(l.sessions, id)
	return nil
}

func (l *LocalSessionStore) Refresh(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, exists := l.sessions[id]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = time.Now().Add(l.config.TTL)
	session.UpdatedAt = time.Now()
	return nil
}

func (l *LocalSessionStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	l.mu.RLock()
	ids := l.index[userID]
	l.mu.RUnlock()

	sessions := make([]*Session, 0, len(ids))
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, id := range ids {
		if session, exists := l.sessions[id]; exists {
			if time.Now().Before(session.ExpiresAt) {
				sessions = append(sessions, session)
			}
		}
	}

	return sessions, nil
}

func (l *LocalSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	ids := l.ListIDsByUser(userID)

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, id := range ids {
		delete(l.sessions, id)
	}
	delete(l.index, userID)

	return nil
}

func (l *LocalSessionStore) Exists(ctx context.Context, id string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	session, exists := l.sessions[id]
	if !exists {
		return false, nil
	}

	return time.Now().Before(session.ExpiresAt), nil
}

func (l *LocalSessionStore) ListIDsByUser(userID string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	ids := make([]string, len(l.index[userID]))
	copy(ids, l.index[userID])
	return ids
}

func (l *LocalSessionStore) addToIndex(userID, sessionID string) {
	if _, exists := l.index[userID]; !exists {
		l.index[userID] = make([]string, 0)
	}
	l.index[userID] = append(l.index[userID], sessionID)
}

func (l *LocalSessionStore) removeFromIndex(userID, sessionID string) {
	if ids, exists := l.index[userID]; exists {
		newIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			if id != sessionID {
				newIDs = append(newIDs, id)
			}
		}
		l.index[userID] = newIDs
	}
}

func (l *LocalSessionStore) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.sessions)
}

func (l *LocalSessionStore) Cleanup() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	count := 0

	for id, session := range l.sessions {
		if now.After(session.ExpiresAt) {
			if session.UserID != "" {
				l.removeFromIndex(session.UserID, id)
			}
			delete(l.sessions, id)
			count++
		}
	}

	return count
}

func NewMultiNodeSessionManager(redis RedisSessionStoreInterface, cfg *SessionConfig) *MultiNodeSessionManager {
	return &MultiNodeSessionManager{
		localStore: NewLocalSessionStore(cfg),
		redisStore: NewRedisSessionStore(redis, cfg),
		config:     cfg,
	}
}

func (m *MultiNodeSessionManager) Create(ctx context.Context, data map[string]interface{}) (*Session, error) {
	session, err := m.redisStore.Create(ctx, data)
	if err != nil {
		return nil, err
	}

	localData := make(map[string]interface{})
	for k, v := range data {
		localData[k] = v
	}
	m.localStore.Create(ctx, localData)

	return session, nil
}

func (m *MultiNodeSessionManager) Get(ctx context.Context, id string) (*Session, bool, error) {
	session, exists, err := m.localStore.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if exists {
		return session, true, nil
	}

	session, exists, err = m.redisStore.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if exists {
		m.localStore.Create(ctx, session.Data)
		return session, true, nil
	}

	return nil, false, nil
}

func (m *MultiNodeSessionManager) Update(ctx context.Context, id string, data map[string]interface{}) error {
	if err := m.localStore.Update(ctx, id, data); err != nil {
		return err
	}
	return m.redisStore.Update(ctx, id, data)
}

func (m *MultiNodeSessionManager) Delete(ctx context.Context, id string) error {
	if err := m.localStore.Delete(ctx, id); err != nil {
		return err
	}
	return m.redisStore.Delete(ctx, id)
}

func (m *MultiNodeSessionManager) Refresh(ctx context.Context, id string) error {
	if err := m.localStore.Refresh(ctx, id); err != nil {
		return err
	}
	return m.redisStore.Refresh(ctx, id)
}

func (m *MultiNodeSessionManager) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	return m.redisStore.ListByUser(ctx, userID)
}

func (m *MultiNodeSessionManager) DeleteByUser(ctx context.Context, userID string) error {
	if err := m.localStore.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	return m.redisStore.DeleteByUser(ctx, userID)
}

func (m *MultiNodeSessionManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"local_sessions":  m.localStore.Count(),
		"config":          m.config,
	}
}
