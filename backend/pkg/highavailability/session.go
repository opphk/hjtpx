package highavailability

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Session struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id,omitempty"`
	Data         map[string]interface{} `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	InstanceID   string                 `json:"instance_id"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, sessionID string) error
	Exists(ctx context.Context, sessionID string) (bool, error)
	Refresh(ctx context.Context, sessionID string) error
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

type RedisSessionStore struct {
	client      *goredis.Client
	keyPrefix   string
	defaultTTL  time.Duration
	mu          sync.RWMutex
	serializer  SessionSerializer
}

type SessionSerializer interface {
	Serialize(session *Session) ([]byte, error)
	Deserialize(data []byte) (*Session, error)
}

type JSONSessionSerializer struct{}

func (s *JSONSessionSerializer) Serialize(session *Session) ([]byte, error) {
	return json.Marshal(session)
}

func (s *JSONSessionSerializer) Deserialize(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

type SessionStoreConfig struct {
	KeyPrefix  string
	DefaultTTL time.Duration
	MaxTTL     time.Duration
}

var defaultSessionStoreConfig = &SessionStoreConfig{
	KeyPrefix:  "session:",
	DefaultTTL: 24 * time.Hour,
	MaxTTL:     7 * 24 * time.Hour,
}

func NewRedisSessionStore(client *goredis.Client, config *SessionStoreConfig) (*RedisSessionStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	if config == nil {
		config = defaultSessionStoreConfig
	}

	return &RedisSessionStore{
		client:     client,
		keyPrefix:  config.KeyPrefix,
		defaultTTL: config.DefaultTTL,
		serializer: &JSONSessionSerializer{},
	}, nil
}

func (s *RedisSessionStore) sessionKey(sessionID string) string {
	return s.keyPrefix + sessionID
}

func (s *RedisSessionStore) userSessionsKey(userID string) string {
	return s.keyPrefix + "user:" + userID
}

func (s *RedisSessionStore) Create(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = now.Add(s.defaultTTL)
	}

	if session.Data == nil {
		session.Data = make(map[string]interface{})
	}

	data, err := s.serializer.Serialize(session)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = s.defaultTTL
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, s.sessionKey(session.ID), data, ttl)

	if session.UserID != "" {
		pipe.SAdd(ctx, s.userSessionsKey(session.UserID), session.ID)
		pipe.Expire(ctx, s.userSessionsKey(session.UserID), s.defaultTTL)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	data, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session, err := s.serializer.Deserialize(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}

	return session, nil
}

func (s *RedisSessionStore) Update(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	exists, err := s.Exists(ctx, session.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	session.UpdatedAt = time.Now()

	data, err := s.serializer.Serialize(session)
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = s.defaultTTL
	}

	err = s.client.Set(ctx, s.sessionKey(session.ID), data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		if err.Error() == fmt.Sprintf("session not found: %s", sessionID) {
			return nil
		}
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.sessionKey(sessionID))

	if session.UserID != "" {
		pipe.SRem(ctx, s.userSessionsKey(session.UserID), sessionID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("session ID is required")
	}

	count, err := s.client.Exists(ctx, s.sessionKey(sessionID)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

func (s *RedisSessionStore) Refresh(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(s.defaultTTL)

	return s.Update(ctx, session)
}

func (s *RedisSessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	sessionIDs, err := s.client.SMembers(ctx, s.userSessionsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user session IDs: %w", err)
	}

	sessions := make([]*Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (s *RedisSessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

type InMemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *InMemorySessionStore) Create(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = now.Add(24 * time.Hour)
	}

	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	return session, nil
}

func (s *InMemorySessionStore) Update(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.UpdatedAt = time.Now()
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *InMemorySessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return false, nil
	}

	if time.Now().After(session.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

func (s *InMemorySessionStore) Refresh(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	return nil
}

func (s *InMemorySessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range s.sessions {
		if session.UserID == userID && !time.Now().After(session.ExpiresAt) {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (s *InMemorySessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := int64(0)

	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
			count++
		}
	}

	return count, nil
}

type SessionManager struct {
	store          SessionStore
	instanceID     string
	defaultTTL     time.Duration
	maxTTL         time.Duration
	refreshEnabled bool
	mu             sync.RWMutex
}

type SessionManagerConfig struct {
	Store          SessionStore
	InstanceID     string
	DefaultTTL     time.Duration
	MaxTTL         time.Duration
	RefreshEnabled bool
}

func NewSessionManager(config *SessionManagerConfig) (*SessionManager, error) {
	if config.Store == nil {
		return nil, fmt.Errorf("session store is required")
	}

	defaultTTL := 24 * time.Hour
	if config.DefaultTTL > 0 {
		defaultTTL = config.DefaultTTL
	}

	maxTTL := 7 * 24 * time.Hour
	if config.MaxTTL > 0 {
		maxTTL = config.MaxTTL
	}

	return &SessionManager{
		store:          config.Store,
		instanceID:     config.InstanceID,
		defaultTTL:     defaultTTL,
		maxTTL:         maxTTL,
		refreshEnabled: config.RefreshEnabled,
	}, nil
}

func (sm *SessionManager) CreateSession(ctx context.Context, userID string, data map[string]interface{}, ttl time.Duration) (*Session, error) {
	session := &Session{
		ID:         generateSessionID(),
		UserID:     userID,
		Data:       data,
		InstanceID: sm.instanceID,
		IsActive:   true,
	}

	if ttl > 0 {
		if ttl > sm.maxTTL {
			ttl = sm.maxTTL
		}
		session.ExpiresAt = time.Now().Add(ttl)
	} else {
		session.ExpiresAt = time.Now().Add(sm.defaultTTL)
	}

	if err := sm.store.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if sm.refreshEnabled {
		if err := sm.store.Refresh(ctx, sessionID); err != nil {
			return session, nil
		}
	}

	return session, nil
}

func (sm *SessionManager) UpdateSession(ctx context.Context, session *Session) error {
	return sm.store.Update(ctx, session)
}

func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	return sm.store.Delete(ctx, sessionID)
}

func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string) (bool, error) {
	exists, err := sm.store.Exists(ctx, sessionID)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	session, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}

	return session.IsActive, nil
}

func (sm *SessionManager) RefreshSession(ctx context.Context, sessionID string) error {
	return sm.store.Refresh(ctx, sessionID)
}

func (sm *SessionManager) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	return sm.store.GetByUserID(ctx, userID)
}

func (sm *SessionManager) DeleteUserSessions(ctx context.Context, userID string) error {
	sessions, err := sm.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if err := sm.store.Delete(ctx, session.ID); err != nil {
			continue
		}
	}

	return nil
}

func (sm *SessionManager) SetSessionData(ctx context.Context, sessionID string, key string, value interface{}) error {
	session, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Data == nil {
		session.Data = make(map[string]interface{})
	}

	session.Data[key] = value
	return sm.store.Update(ctx, session)
}

func (sm *SessionManager) GetSessionData(ctx context.Context, sessionID string, key string) (interface{}, bool) {
	session, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return nil, false
	}

	value, ok := session.Data[key]
	return value, ok
}

func (sm *SessionManager) DeleteSessionData(ctx context.Context, sessionID string, key string) error {
	session, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	delete(session.Data, key)
	return sm.store.Update(ctx, session)
}

func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return sm.store.DeleteExpired(ctx)
}

func generateSessionID() string {
	return fmt.Sprintf("%s-%d", uuid.New().String(), time.Now().UnixNano())
}

type SessionConsistencyManager struct {
	localStore  *InMemorySessionStore
	remoteStore SessionStore
	syncInterval time.Duration
	stopCh      chan struct{}
	mu          sync.RWMutex
}

func NewSessionConsistencyManager(remoteStore SessionStore, syncInterval time.Duration) *SessionConsistencyManager {
	return &SessionConsistencyManager{
		localStore:  NewInMemorySessionStore(),
		remoteStore: remoteStore,
		syncInterval: syncInterval,
		stopCh:      make(chan struct{}),
	}
}

func (scm *SessionConsistencyManager) StartSync(ctx context.Context) {
	ticker := time.NewTicker(scm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-scm.stopCh:
			return
		case <-ticker.C:
			scm.syncToRemote()
		}
	}
}

func (scm *SessionConsistencyManager) StopSync() {
	close(scm.stopCh)
}

func (scm *SessionConsistencyManager) syncToRemote() {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	ctx := context.Background()
	sessions := scm.localStore.sessions

	for _, session := range sessions {
		if err := scm.remoteStore.Create(ctx, session); err != nil {
			continue
		}
	}
}

func (scm *SessionConsistencyManager) SetLocal(ctx context.Context, session *Session) error {
	return scm.localStore.Create(ctx, session)
}

func (scm *SessionConsistencyManager) GetLocal(ctx context.Context, sessionID string) (*Session, error) {
	return scm.localStore.Get(ctx, sessionID)
}
