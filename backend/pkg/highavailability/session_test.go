package highavailability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemorySessionStore_Create(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:       "test-session-1",
		UserID:   "user-1",
		Data:     map[string]interface{}{"key": "value"},
		IsActive: true,
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "test-session-1")
	require.NoError(t, err)
	assert.Equal(t, "test-session-1", retrieved.ID)
	assert.Equal(t, "user-1", retrieved.UserID)
}

func TestInMemorySessionStore_Get(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:     "test-session-1",
		UserID: "user-1",
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "test-session-1")
	require.NoError(t, err)
	assert.Equal(t, "test-session-1", retrieved.ID)
}

func TestInMemorySessionStore_GetExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:        "test-session-1",
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	_, err = store.Get(ctx, "test-session-1")
	assert.Error(t, err)
}

func TestInMemorySessionStore_Update(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:     "test-session-1",
		UserID: "user-1",
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	session.Data = map[string]interface{}{"updated": "value"}
	err = store.Update(ctx, session)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "test-session-1")
	require.NoError(t, err)
	assert.Equal(t, "value", retrieved.Data["updated"])
}

func TestInMemorySessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:     "test-session-1",
		UserID: "user-1",
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	err = store.Delete(ctx, "test-session-1")
	require.NoError(t, err)

	_, err = store.Get(ctx, "test-session-1")
	assert.Error(t, err)
}

func TestInMemorySessionStore_Exists(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:     "test-session-1",
		UserID: "user-1",
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "test-session-1")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Exists(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInMemorySessionStore_Refresh(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	session := &Session{
		ID:        "test-session-1",
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := store.Create(ctx, session)
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	time.Sleep(10 * time.Millisecond)

	err = store.Refresh(ctx, "test-session-1")
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "test-session-1")
	require.NoError(t, err)
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry))
}

func TestInMemorySessionStore_GetByUserID(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	sessions := []*Session{
		{ID: "session-1", UserID: "user-1"},
		{ID: "session-2", UserID: "user-1"},
		{ID: "session-3", UserID: "user-2"},
	}

	for _, session := range sessions {
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	user1Sessions, err := store.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, user1Sessions, 2)

	user2Sessions, err := store.GetByUserID(ctx, "user-2")
	require.NoError(t, err)
	assert.Len(t, user2Sessions, 1)
}

func TestInMemorySessionStore_DeleteExpired(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	sessions := []*Session{
		{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(1 * time.Hour)},
		{ID: "session-2", UserID: "user-1", ExpiresAt: time.Now().Add(-1 * time.Hour)},
		{ID: "session-3", UserID: "user-2", ExpiresAt: time.Now().Add(-1 * time.Hour)},
	}

	for _, session := range sessions {
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	count, err := store.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	exists1, _ := store.Exists(ctx, "session-1")
	assert.True(t, exists1)

	exists2, _ := store.Exists(ctx, "session-2")
	assert.False(t, exists2)
}

func TestInMemorySessionStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			session := &Session{
				ID:     "session-" + string(rune('0'+id)),
				UserID: "user-1",
			}
			_ = store.Create(ctx, session)
		}(i)
	}
	wg.Wait()

	sessions, err := store.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 10)
}

func TestSessionManager_CreateSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
		DefaultTTL: 24 * time.Hour,
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", map[string]interface{}{"key": "value"}, 0)
	require.NoError(t, err)

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user-1", session.UserID)
	assert.Equal(t, "instance-1", session.InstanceID)
	assert.True(t, session.IsActive)
}

func TestSessionManager_GetSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	created, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	retrieved, err := manager.GetSession(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestSessionManager_UpdateSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	session.Data = map[string]interface{}{"updated": "data"}
	err = manager.UpdateSession(ctx, session)
	require.NoError(t, err)

	retrieved, err := manager.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "data", retrieved.Data["updated"])
}

func TestSessionManager_DeleteSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	err = manager.DeleteSession(ctx, session.ID)
	require.NoError(t, err)

	_, err = manager.GetSession(ctx, session.ID)
	assert.Error(t, err)
}

func TestSessionManager_ValidateSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	valid, err := manager.ValidateSession(ctx, session.ID)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSessionManager_RefreshSession(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	time.Sleep(10 * time.Millisecond)

	err = manager.RefreshSession(ctx, session.ID)
	require.NoError(t, err)

	retrieved, err := manager.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry))
}

func TestSessionManager_GetUserSessions(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	_, err = manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)
	_, err = manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)
	_, err = manager.CreateSession(ctx, "user-2", nil, 0)
	require.NoError(t, err)

	sessions, err := manager.GetUserSessions(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSessionManager_DeleteUserSessions(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	session1, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)
	_, err = manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	err = manager.DeleteUserSessions(ctx, "user-1")
	require.NoError(t, err)

	exists, _ := store.Exists(ctx, session1.ID)
	assert.False(t, exists)
}

func TestSessionManager_SetSessionData(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	err = manager.SetSessionData(ctx, session.ID, "key", "value")
	require.NoError(t, err)

	data, ok := manager.GetSessionData(ctx, session.ID, "key")
	assert.True(t, ok)
	assert.Equal(t, "value", data)
}

func TestSessionManager_GetSessionData(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", map[string]interface{}{"existing": "data"}, 0)
	require.NoError(t, err)

	data, ok := manager.GetSessionData(ctx, session.ID, "existing")
	assert.True(t, ok)
	assert.Equal(t, "data", data)

	_, ok = manager.GetSessionData(ctx, session.ID, "non-existent")
	assert.False(t, ok)
}

func TestSessionManager_DeleteSessionData(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	session, err := manager.CreateSession(ctx, "user-1", map[string]interface{}{"key": "value"}, 0)
	require.NoError(t, err)

	err = manager.DeleteSessionData(ctx, session.ID, "key")
	require.NoError(t, err)

	_, ok := manager.GetSessionData(ctx, session.ID, "key")
	assert.False(t, ok)
}

func TestSessionManager_CleanupExpiredSessions(t *testing.T) {
	store := NewInMemorySessionStore()
	manager, err := NewSessionManager(&SessionManagerConfig{
		Store:      store,
		InstanceID: "instance-1",
	})
	require.NoError(t, err)

	ctx := context.Background()

	_, err = manager.CreateSession(ctx, "user-1", nil, 0)
	require.NoError(t, err)

	count, err := manager.CleanupExpiredSessions(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestJSONSessionSerializer_SerializeDeserialize(t *testing.T) {
	serializer := &JSONSessionSerializer{}

	session := &Session{
		ID:         "test-session",
		UserID:     "user-1",
		Data:       map[string]interface{}{"key": "value"},
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	data, err := serializer.Serialize(session)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	deserialized, err := serializer.Deserialize(data)
	require.NoError(t, err)
	assert.Equal(t, session.ID, deserialized.ID)
	assert.Equal(t, session.UserID, deserialized.UserID)
	assert.Equal(t, session.Data["key"], deserialized.Data["key"])
}

func TestJSONSessionSerializer_DeserializeError(t *testing.T) {
	serializer := &JSONSessionSerializer{}

	_, err := serializer.Deserialize([]byte("invalid json"))
	assert.Error(t, err)
}

func TestSessionConsistencyManager_StartStopSync(t *testing.T) {
	scm := NewSessionConsistencyManager(nil, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	go scm.StartSync(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	scm.StopSync()
}

func TestSessionConsistencyManager_SetLocal(t *testing.T) {
	scm := NewSessionConsistencyManager(nil, 10*time.Second)
	ctx := context.Background()

	session := &Session{
		ID:     "test-session",
		UserID: "user-1",
	}

	err := scm.SetLocal(ctx, session)
	require.NoError(t, err)

	retrieved, err := scm.GetLocal(ctx, "test-session")
	require.NoError(t, err)
	assert.Equal(t, "test-session", retrieved.ID)
}

func TestSessionConsistencyManager_GetLocal(t *testing.T) {
	scm := NewSessionConsistencyManager(nil, 10*time.Second)
	ctx := context.Background()

	_, err := scm.GetLocal(ctx, "non-existent")
	assert.Error(t, err)
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}
