package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TwoFactorType string

const (
	TwoFactorCode    TwoFactorType = "code"
	TwoFactorConfirm TwoFactorType = "confirm"
	TwoFactorPassword TwoFactorType = "password"
)

type TwoFactorChallenge struct {
	ID          string
	Type        TwoFactorType
	Action      string
	ResourceID  string
	UserID      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Verified    bool
	Attempts    int
	MaxAttempts int
	Code        string
	Context     map[string]interface{}
}

type TwoFactorConfig struct {
	Enabled           bool
	CodeLength        int
	CodeExpiry        time.Duration
	MaxAttempts       int
	RequireForDeletes bool
	RequireForConfig  bool
	RequireForAdmin   bool
	ProtectedPaths    []string
}

var defaultTwoFactorConfig = &TwoFactorConfig{
	Enabled:           true,
	CodeLength:        6,
	CodeExpiry:        5 * time.Minute,
	MaxAttempts:       3,
	RequireForDeletes: true,
	RequireForConfig:  true,
	RequireForAdmin:   true,
	ProtectedPaths: []string{
		"/admin/api/config",
		"/admin/api/blacklist",
		"/admin/api/whitelist",
	},
}

type TwoFactorService struct {
	config   *TwoFactorConfig
	pending  map[string]*TwoFactorChallenge
	mu       sync.RWMutex
	verified map[string]bool
}

func NewTwoFactorService(config *TwoFactorConfig) *TwoFactorService {
	if config == nil {
		config = defaultTwoFactorConfig
	}
	return &TwoFactorService{
		config:   config,
		pending:  make(map[string]*TwoFactorChallenge),
		verified: make(map[string]bool),
	}
}

func (s *TwoFactorService) IsEnabled() bool {
	return s.config.Enabled
}

func (s *TwoFactorService) IsProtected(action string, path string) bool {
	if !s.config.Enabled {
		return false
	}

	switch action {
	case "delete":
		return s.config.RequireForDeletes
	case "update", "create":
		if strings.Contains(path, "/config") {
			return s.config.RequireForConfig
		}
	case "admin":
		return s.config.RequireForAdmin
	}

	for _, p := range s.config.ProtectedPaths {
		if strings.Contains(path, p) {
			return true
		}
	}

	return false
}

func (s *TwoFactorService) CreateChallenge(ctx context.Context, userID, action, resourceID string, challengeType TwoFactorType) (*TwoFactorChallenge, error) {
	if !s.IsProtected(action, "") {
		return nil, nil
	}

	challengeID, err := generateChallengeID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge ID: %w", err)
	}

	challenge := &TwoFactorChallenge{
		ID:          challengeID,
		Type:        challengeType,
		Action:      action,
		ResourceID:  resourceID,
		UserID:      userID,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(s.config.CodeExpiry),
		Verified:    false,
		Attempts:    0,
		MaxAttempts: s.config.MaxAttempts,
		Context:     make(map[string]interface{}),
	}

	if challengeType == TwoFactorCode {
		code, err := s.generateCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		challenge.Code = code
	}

	s.mu.Lock()
	s.pending[challengeID] = challenge
	s.mu.Unlock()

	return challenge, nil
}

func (s *TwoFactorService) generateCode() (string, error) {
	bytes := make([]byte, s.config.CodeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := ""
	for _, b := range bytes[:s.config.CodeLength] {
		code += fmt.Sprintf("%d", b%10)
	}
	return code, nil
}

func (s *TwoFactorService) GetChallenge(challengeID string) (*TwoFactorChallenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	challenge, exists := s.pending[challengeID]
	if !exists {
		return nil, errors.New("challenge not found")
	}

	if time.Now().After(challenge.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}

	return challenge, nil
}

func (s *TwoFactorService) VerifyChallenge(challengeID, codeOrConfirm string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, exists := s.pending[challengeID]
	if !exists {
		return false, errors.New("challenge not found")
	}

	if time.Now().After(challenge.ExpiresAt) {
		delete(s.pending, challengeID)
		return false, errors.New("challenge expired")
	}

	if challenge.Verified {
		return true, nil
	}

	challenge.Attempts++
	if challenge.Attempts > challenge.MaxAttempts {
		delete(s.pending, challengeID)
		return false, errors.New("too many attempts")
	}

	switch challenge.Type {
	case TwoFactorCode:
		if challenge.Code != codeOrConfirm {
			return false, errors.New("invalid code")
		}
	case TwoFactorConfirm:
		if !strings.EqualFold(codeOrConfirm, "confirm") && codeOrConfirm != "true" {
			return false, errors.New("confirmation required")
		}
	case TwoFactorPassword:
		h := sha256.New()
		h.Write([]byte(codeOrConfirm))
		hash := hex.EncodeToString(h.Sum(nil))
		if hash != challenge.Code {
			return false, errors.New("invalid password")
		}
	}

	challenge.Verified = true
	s.verified[challengeID] = true

	go func() {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		delete(s.verified, challengeID)
		delete(s.pending, challengeID)
		s.mu.Unlock()
	}()

	return true, nil
}

func (s *TwoFactorService) IsVerified(challengeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.verified[challengeID]
}

func (s *TwoFactorService) InvalidateChallenge(challengeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, challengeID)
	delete(s.verified, challengeID)
}

func (s *TwoFactorService) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, challenge := range s.pending {
		if now.After(challenge.ExpiresAt) {
			delete(s.pending, id)
			delete(s.verified, id)
		}
	}
}

type ConfirmationToken struct {
	Token     string
	Action    string
	ResourceID string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

type ConfirmationService struct {
	config      *TwoFactorConfig
	tokens      map[string]*ConfirmationToken
	mu          sync.RWMutex
	tokenExpiry time.Duration
}

func NewConfirmationService(config *TwoFactorConfig) *ConfirmationService {
	if config == nil {
		config = defaultTwoFactorConfig
	}
	return &ConfirmationService{
		config:      config,
		tokens:      make(map[string]*ConfirmationToken),
		tokenExpiry: 10 * time.Minute,
	}
}

func (s *ConfirmationService) GenerateToken(userID, action, resourceID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token] = &ConfirmationToken{
		Token:      token,
		Action:     action,
		ResourceID: resourceID,
		UserID:     userID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(s.tokenExpiry),
		Used:       false,
	}

	return token, nil
}

func (s *ConfirmationService) ValidateToken(token string) (*ConfirmationToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tokens[token]
	if !exists {
		return nil, errors.New("token not found")
	}

	if t.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().After(t.ExpiresAt) {
		delete(s.tokens, token)
		return nil, errors.New("token expired")
	}

	return t, nil
}

func (s *ConfirmationService) ConsumeToken(token string) (*ConfirmationToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tokens[token]
	if !exists {
		return nil, errors.New("token not found")
	}

	if t.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().After(t.ExpiresAt) {
		delete(s.tokens, token)
		return nil, errors.New("token expired")
	}

	t.Used = true
	delete(s.tokens, token)

	return t, nil
}

type TwoFactorMiddleware struct {
	service *TwoFactorService
	paths   map[string]TwoFactorType
}

func NewTwoFactorMiddleware(config *TwoFactorConfig) *TwoFactorMiddleware {
	if config == nil {
		config = defaultTwoFactorConfig
	}
	return &TwoFactorMiddleware{
		service: NewTwoFactorService(config),
		paths:   make(map[string]TwoFactorType),
	}
}

func (m *TwoFactorMiddleware) AddProtectedPath(path string, action TwoFactorType) {
	m.paths[path] = action
}

func (m *TwoFactorMiddleware) RequireConfirmation() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		if !m.service.IsProtected(method, path) {
			c.Next()
			return
		}

		challengeID := c.GetHeader("X-Confirmation-ID")
		if challengeID == "" {
			challengeID = c.GetHeader("X-Two-Factor-ID")
		}

		if challengeID == "" {
			c.AbortWithStatusJSON(403, gin.H{
				"error":       "confirmation_required",
				"message":     "This action requires confirmation",
				"challenge_id": "",
			})
			return
		}

		if !m.service.IsVerified(challengeID) {
			c.AbortWithStatusJSON(403, gin.H{
				"error":       "confirmation_required",
				"message":     "Confirmation required or expired",
				"challenge_id": challengeID,
			})
			return
		}

		c.Next()
	}
}

func generateChallengeID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *TwoFactorService) GetCodeForTesting(challengeID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	challenge, exists := s.pending[challengeID]
	if !exists {
		return "", errors.New("challenge not found")
	}

	return challenge.Code, nil
}

type SensitiveOperation struct {
	Action    string
	Path      string
	Method    string
	RequiresConfirm bool
	AuditAction AuditAction
	AuditLevel AuditLevel
}

var sensitiveOperations = []SensitiveOperation{
	{Action: "delete_config", Path: "/admin/api/config", Method: "POST", RequiresConfirm: true, AuditAction: ActionConfigChange, AuditLevel: AuditLevelWarning},
	{Action: "add_blacklist", Path: "/admin/api/blacklist", Method: "POST", RequiresConfirm: true, AuditAction: ActionBlacklistAdd, AuditLevel: AuditLevelWarning},
	{Action: "remove_blacklist", Path: "/admin/api/blacklist", Method: "DELETE", RequiresConfirm: true, AuditAction: ActionBlacklistRemove, AuditLevel: AuditLevelWarning},
	{Action: "add_whitelist", Path: "/admin/api/whitelist", Method: "POST", RequiresConfirm: true, AuditAction: ActionWhitelistAdd, AuditLevel: AuditLevelWarning},
	{Action: "remove_whitelist", Path: "/admin/api/whitelist", Method: "DELETE", RequiresConfirm: true, AuditAction: ActionWhitelistRemove, AuditLevel: AuditLevelWarning},
	{Action: "delete_captcha", Path: "/admin/api/captcha", Method: "DELETE", RequiresConfirm: true, AuditAction: ActionSensitiveDelete, AuditLevel: AuditLevelWarning},
	{Action: "create_admin", Path: "/admin/api/admin", Method: "POST", RequiresConfirm: true, AuditAction: ActionAdminCreate, AuditLevel: AuditLevelCritical},
	{Action: "delete_admin", Path: "/admin/api/admin", Method: "DELETE", RequiresConfirm: true, AuditAction: ActionAdminDelete, AuditLevel: AuditLevelCritical},
	{Action: "rotate_key", Path: "/admin/api/key/rotate", Method: "POST", RequiresConfirm: true, AuditAction: ActionKeyRotate, AuditLevel: AuditLevelCritical},
}

func MatchSensitiveOperation(method, path string) *SensitiveOperation {
	for i := range sensitiveOperations {
		op := &sensitiveOperations[i]
		if op.Method == method && pathContains(path, op.Path) {
			return op
		}
	}
	return nil
}

func pathContains(full, part string) bool {
	if part == "" {
		return false
	}
	return strings.HasPrefix(full, part) || strings.Contains(full, part)
}

type OperationConfirmHandler struct {
	twoFactor *TwoFactorService
	confirm   *ConfirmationService
	logger    *AuditLoggerV2
}

func NewOperationConfirmHandler(twoFactor *TwoFactorService, confirm *ConfirmationService, logger *AuditLoggerV2) *OperationConfirmHandler {
	return &OperationConfirmHandler{
		twoFactor: twoFactor,
		confirm:   confirm,
		logger:    logger,
	}
}

func (h *OperationConfirmHandler) InitiateConfirmation(c *gin.Context, action, resourceID string) (string, error) {
	userID := ""
	if v, exists := c.Get("user_id"); exists {
		userID = fmt.Sprintf("%v", v)
	}

	token, err := h.confirm.GenerateToken(userID, action, resourceID)
	if err != nil {
		return "", fmt.Errorf("failed to generate confirmation token: %w", err)
	}

	username := ""
	if v, exists := c.Get("username"); exists {
		username = fmt.Sprintf("%v", v)
	}

	LogConfigChange(h.logger, userID, username, GetClientIP(c), map[string]interface{}{
		"action":      action,
		"resource_id": resourceID,
		"token":       token,
	})

	return token, nil
}

func (h *OperationConfirmHandler) ValidateAndExecute(c *gin.Context, action func() error) error {
	token := c.GetHeader("X-Confirmation-Token")
	if token == "" {
		token = c.Query("confirmation_token")
	}

	if token == "" {
		return errors.New("confirmation token required")
	}

	t, err := h.confirm.ConsumeToken(token)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if err := action(); err != nil {
		return err
	}

	userID := ""
	if v, exists := c.Get("user_id"); exists {
		userID = fmt.Sprintf("%v", v)
	}
	username := ""
	if v, exists := c.Get("username"); exists {
		username = fmt.Sprintf("%v", v)
	}

	LogSecurityEvent(h.logger, AuditLevelInfo, ActionConfigChange, "sensitive operation completed", map[string]interface{}{
		"user_id":     userID,
		"username":    username,
		"ip_address":  GetClientIP(c),
		"action":      t.Action,
		"resource_id": t.ResourceID,
	})

	return nil
}
