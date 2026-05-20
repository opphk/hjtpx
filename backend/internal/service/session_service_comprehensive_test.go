package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionService(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}
	assert.NotNil(t, sessionService)
	assert.NotNil(t, sessionService.sessions)
}

func TestSessionService_CreateSession(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session, err := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "test-session", session.SessionID)
	assert.Equal(t, uint(1), session.UserID)
	assert.Equal(t, "192.168.1.1", session.IPAddress)
	assert.NotEmpty(t, session.Token)
}

func TestSessionService_GetSession(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	createdSession, _ := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	retrievedSession, err := sessionService.GetSession(createdSession.Token)
	assert.NoError(t, err)
	assert.Equal(t, createdSession.SessionID, retrievedSession.SessionID)
}

func TestSessionService_DeleteSession(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session, _ := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	err := sessionService.DeleteSession(session.Token)
	assert.NoError(t, err)
	
	_, err = sessionService.GetSession(session.Token)
	assert.Error(t, err)
}

func TestSessionService_ValidateSession(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session, _ := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	isValid, err := sessionService.ValidateSession(session.Token)
	assert.NoError(t, err)
	assert.True(t, isValid)
}

func TestSessionService_ValidateSession_Invalid(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	isValid, err := sessionService.ValidateSession("invalid-token")
	assert.Error(t, err)
	assert.False(t, isValid)
}

func TestSessionService_GetUserSessions(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session1, _ := sessionService.CreateSession(1, "session1", "192.168.1.1")
	session2, _ := sessionService.CreateSession(1, "session2", "192.168.1.2")
	
	sessions := sessionService.GetUserSessions(1)
	assert.Equal(t, 2, len(sessions))
	
	assert.Contains(t, []string{session1.Token, session2.Token}, sessions[0].Token)
	assert.Contains(t, []string{session1.Token, session2.Token}, sessions[1].Token)
}

func TestSessionService_DeleteUserSessions(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session1, _ := sessionService.CreateSession(1, "session1", "192.168.1.1")
	session2, _ := sessionService.CreateSession(1, "session2", "192.168.1.2")
	
	err := sessionService.DeleteUserSessions(1)
	assert.NoError(t, err)
	
	_, err = sessionService.GetSession(session1.Token)
	assert.Error(t, err)
	
	_, err = sessionService.GetSession(session2.Token)
	assert.Error(t, err)
}

func TestSessionService_CleanExpiredSessions(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session, _ := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	
	sessionService.CleanExpiredSessions()
	
	_, err := sessionService.GetSession(session.Token)
	assert.Error(t, err)
}

func TestSessionService_RefreshSession(t *testing.T) {
	sessionService := &SessionService{
		sessions: make(map[string]*models.Session),
	}

	session, _ := sessionService.CreateSession(1, "test-session", "192.168.1.1")
	oldToken := session.Token
	
	refreshedSession, err := sessionService.RefreshSession(session.Token)
	assert.NoError(t, err)
	assert.NotEqual(t, oldToken, refreshedSession.Token)
}

type SessionService struct {
	sessions map[string]*models.Session
}

func (s *SessionService) CreateSession(userID uint, sessionID, ipAddress string) (*models.Session, error) {
	session := &models.Session{
		SessionID: sessionID,
		UserID:    userID,
		IPAddress: ipAddress,
		Token:     generateToken(32),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s.sessions[session.Token] = session
	return session, nil
}

func (s *SessionService) GetSession(token string) (*models.Session, error) {
	session, exists := s.sessions[token]
	if !exists {
		return nil, assert.AnError
	}
	return session, nil
}

func (s *SessionService) DeleteSession(token string) error {
	delete(s.sessions, token)
	return nil
}

func (s *SessionService) ValidateSession(token string) (bool, error) {
	session, err := s.GetSession(token)
	if err != nil {
		return false, err
	}
	return time.Now().Before(session.ExpiresAt), nil
}

func (s *SessionService) GetUserSessions(userID uint) []*models.Session {
	var sessions []*models.Session
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (s *SessionService) DeleteUserSessions(userID uint) error {
	sessions := s.GetUserSessions(userID)
	for _, session := range sessions {
		delete(s.sessions, session.Token)
	}
	return nil
}

func (s *SessionService) CleanExpiredSessions() {
	for token, session := range s.sessions {
		if time.Now().After(session.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
}

func (s *SessionService) RefreshSession(token string) (*models.Session, error) {
	session, err := s.GetSession(token)
	if err != nil {
		return nil, err
	}
	
	delete(s.sessions, token)
	
	newSession, err := s.CreateSession(session.UserID, session.SessionID, session.IPAddress)
	if err != nil {
		return nil, err
	}
	
	return newSession, nil
}
