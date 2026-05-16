package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken       = errors.New("token is invalid")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidGrant       = errors.New("invalid grant type")
	ErrInvalidClient      = errors.New("invalid client")
	ErrInvalidScope       = errors.New("invalid scope")
	ErrInvalidState       = errors.New("invalid state parameter")
	ErrAccessDenied       = errors.New("access denied")
	ErrUnauthorizedClient = errors.New("unauthorized client")
)

type GrantType string

const (
	GrantAuthorizationCode GrantType = "authorization_code"
	GrantClientCredentials GrantType = "client_credentials"
	GrantRefreshToken      GrantType = "refresh_token"
	GrantPassword          GrantType = "password"
	GrantImplicit          GrantType = "implicit"
)

type TokenType string

const (
	TokenTypeBearer TokenType = "bearer"
	TokenTypeMAC    TokenType = "mac"
)

type Scope string

const (
	ScopeRead  Scope = "read"
	ScopeWrite Scope = "write"
	ScopeAdmin Scope = "admin"
)

type Client struct {
	ClientID     string            `json:"client_id"`
	ClientSecret string           `json:"-"` // 不序列化密钥
	Name         string           `json:"name"`
	RedirectURIs []string          `json:"redirect_uris"`
	GrantTypes   []GrantType       `json:"grant_types"`
	Scopes       []Scope           `json:"scopes"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	IsActive     bool              `json:"is_active"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type AuthorizationCode struct {
	Code                 string    `json:"code"`
	ClientID             string    `json:"client_id"`
	RedirectURI          string    `json:"redirect_uri"`
	State                string    `json:"state,omitempty"`
	UserID               uint      `json:"user_id"`
	Scopes               []Scope   `json:"scopes"`
	CodeChallenge        string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod  string    `json:"code_challenge_method,omitempty"`
	CodeVerifier         string    `json:"-"`
	ExpiresAt            time.Time `json:"expires_at"`
	Used                 bool      `json:"used"`
	CreatedAt            time.Time `json:"created_at"`
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    TokenType `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IDToken      string    `json:"id_token,omitempty"`
}

type TokenClaims struct {
	ClientID string   `json:"client_id"`
	UserID   uint     `json:"user_id,omitempty"`
	Scopes   []Scope  `json:"scopes"`
	Type     string   `json:"type"`
	jwt.RegisteredClaims
}

type AuthorizationRequest struct {
	ResponseType string   `json:"response_type"`
	ClientID     string   `json:"client_id"`
	RedirectURI  string   `json:"redirect_uri"`
	Scope        []Scope `json:"scope"`
	State        string   `json:"state,omitempty"`
	CodeChallenge string  `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

type AuthorizationResponse struct {
	Code  string `json:"code,omitempty"`
	State string `json:"state,omitempty"`
	URL   string `json:"url,omitempty"`
}

type TokenRequest struct {
	GrantType    GrantType `json:"grant_type"`
	Code         string    `json:"code,omitempty"`
	RedirectURI  string    `json:"redirect_uri,omitempty"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        []Scope   `json:"scope,omitempty"`
	CodeVerifier string    `json:"code_verifier,omitempty"`
	Username     string    `json:"username,omitempty"`
	Password     string    `json:"password,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

type UserInfo struct {
	UserID    uint     `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email,omitempty"`
	Scopes    []Scope  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

type OAuth2Server struct {
	clients            map[string]*Client
	authorizationCodes map[string]*AuthorizationCode
	refreshTokens      map[string]*Token
	accessTokens       map[string]*TokenClaims
	deviceCodes        map[string]*DeviceCodeEntry
	jwtSecret          []byte
	mu                 sync.RWMutex
	codeExpiry         time.Duration
	tokenExpiry        time.Duration
	refreshTokenExpiry time.Duration
	maxRefreshTokens   int
	issuer             string
	audience           string
}

func NewOAuth2Server(jwtSecret []byte, options ...OAuth2Option) *OAuth2Server {
	server := &OAuth2Server{
		clients:            make(map[string]*Client),
		authorizationCodes: make(map[string]*AuthorizationCode),
		refreshTokens:      make(map[string]*Token),
		accessTokens:       make(map[string]*TokenClaims),
		deviceCodes:       make(map[string]*DeviceCodeEntry),
		jwtSecret:          jwtSecret,
		codeExpiry:         10 * time.Minute,
		tokenExpiry:        1 * time.Hour,
		refreshTokenExpiry: 7 * 24 * time.Hour,
		maxRefreshTokens:   10,
		issuer:             "hjtpx-oauth2",
		audience:           "hjtpx-api",
	}
	
	for _, opt := range options {
		opt(server)
	}
	
	go server.cleanupExpiredTokens()
	
	return server
}

type OAuth2Option func(*OAuth2Server)

func WithCodeExpiry(expiry time.Duration) OAuth2Option {
	return func(s *OAuth2Server) {
		s.codeExpiry = expiry
	}
}

func WithTokenExpiry(expiry time.Duration) OAuth2Option {
	return func(s *OAuth2Server) {
		s.tokenExpiry = expiry
	}
}

func WithRefreshTokenExpiry(expiry time.Duration) OAuth2Option {
	return func(s *OAuth2Server) {
		s.refreshTokenExpiry = expiry
	}
}

func (s *OAuth2Server) RegisterClient(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if client.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	
	if client.ClientSecret == "" {
		client.ClientSecret = generateSecureString(32)
	}
	
	if client.CreatedAt.IsZero() {
		client.CreatedAt = time.Now()
	}
	
	if client.ExpiresAt.IsZero() {
		client.ExpiresAt = time.Now().Add(365 * 24 * time.Hour)
	}
	
	client.IsActive = true
	s.clients[client.ClientID] = client
	
	return nil
}

func (s *OAuth2Server) GetClient(clientID string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	client, exists := s.clients[clientID]
	if !exists || !client.IsActive {
		return nil, false
	}
	
	if time.Now().After(client.ExpiresAt) {
		return nil, false
	}
	
	return client, true
}

func (s *OAuth2Server) ValidateClient(clientID, clientSecret string, grantType GrantType) (*Client, error) {
	client, exists := s.GetClient(clientID)
	if !exists {
		return nil, ErrInvalidClient
	}
	
	if clientSecret == "" || !secureCompare(client.ClientSecret, clientSecret) {
		return nil, ErrInvalidClient
	}
	
	grantValid := false
	for _, gt := range client.GrantTypes {
		if gt == grantType {
			grantValid = true
			break
		}
	}
	
	if !grantValid {
		return nil, ErrUnauthorizedClient
	}
	
	return client, nil
}

func (s *OAuth2Server) GenerateAuthorizationCode(clientID, redirectURI string, userID uint, scopes []Scope, state, codeChallenge string) (*AuthorizationCode, error) {
	code := generateSecureString(32)
	
	authCode := &AuthorizationCode{
		Code:             code,
		ClientID:         clientID,
		RedirectURI:      redirectURI,
		State:            state,
		UserID:           userID,
		Scopes:           scopes,
		CodeChallenge:    codeChallenge,
		ExpiresAt:        time.Now().Add(s.codeExpiry),
		CreatedAt:        time.Now(),
	}
	
	s.mu.Lock()
	s.authorizationCodes[code] = authCode
	s.mu.Unlock()
	
	return authCode, nil
}

func (s *OAuth2Server) ExchangeCode(req *TokenRequest) (*TokenResponse, error) {
	if req.GrantType != GrantAuthorizationCode {
		return nil, ErrInvalidGrant
	}
	
	s.mu.Lock()
	authCode, exists := s.authorizationCodes[req.Code]
	if !exists || authCode.Used {
		s.mu.Unlock()
		return nil, ErrInvalidToken
	}
	authCode.Used = true
	s.mu.Unlock()
	
	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	
	if authCode.ClientID != req.ClientID {
		return nil, ErrInvalidClient
	}
	
	if authCode.RedirectURI != req.RedirectURI {
		return nil, ErrInvalidGrant
	}
	
	if req.CodeVerifier != "" && authCode.CodeChallenge != "" {
		if !s.verifyPKCE(req.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
			return nil, ErrAccessDenied
		}
	}
	
	token, err := s.generateToken(authCode.ClientID, authCode.UserID, authCode.Scopes, "")
	if err != nil {
		return nil, err
	}
	
	return &TokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    string(token.TokenType),
		ExpiresIn:    token.ExpiresIn,
		RefreshToken: token.RefreshToken,
		Scope:        token.Scope,
	}, nil
}

func (s *OAuth2Server) generateToken(clientID string, userID uint, scopes []Scope, refreshTokenID string) (*Token, error) {
	now := time.Now()
	expiresAt := now.Add(s.tokenExpiry)
	
	claims := &TokenClaims{
		ClientID: clientID,
		UserID:   userID,
		Scopes:   scopes,
		Type:     "access_token",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}
	
	refreshToken := ""
	
	if refreshTokenID == "" {
		refreshTokenID = generateSecureString(32)
		refreshToken = refreshTokenID
	}
	
	token := &Token{
		AccessToken:  accessToken,
		TokenType:    TokenTypeBearer,
		ExpiresIn:    int(s.tokenExpiry.Seconds()),
		RefreshToken: refreshToken,
		Scope:        strings.Join(scopesAsStrings(scopes), " "),
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
	}
	
	s.mu.Lock()
	if refreshToken != "" {
		s.refreshTokens[refreshToken] = token
	}
	s.accessTokens[accessToken] = claims
	s.mu.Unlock()
	
	return token, nil
}

func (s *OAuth2Server) RefreshAccessToken(req *TokenRequest) (*TokenResponse, error) {
	if req.GrantType != GrantRefreshToken {
		return nil, ErrInvalidGrant
	}
	
	s.mu.RLock()
	oldToken, exists := s.refreshTokens[req.RefreshToken]
	s.mu.RUnlock()
	
	if !exists {
		return nil, ErrInvalidToken
	}
	
	if time.Now().After(oldToken.ExpiresAt) && oldToken.RefreshToken == req.RefreshToken {
		return nil, ErrTokenExpired
	}
	
	client, err := s.ValidateClient(req.ClientID, req.ClientSecret, GrantRefreshToken)
	if err != nil {
		return nil, err
	}
	
	scopes := req.Scope
	if len(scopes) == 0 {
		scopes = stringsAsScopes(strings.Split(oldToken.Scope, " "))
	}
	
	s.mu.Lock()
	delete(s.refreshTokens, req.RefreshToken)
	s.mu.Unlock()
	
	var oldUserID uint
	if oldToken.IDToken != "" {
		fmt.Sscanf(oldToken.IDToken, "%d", &oldUserID)
	}
	
	token, err := s.generateToken(client.ClientID, oldUserID, scopes, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	
	return &TokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    string(token.TokenType),
		ExpiresIn:    token.ExpiresIn,
		RefreshToken: token.RefreshToken,
		Scope:        token.Scope,
	}, nil
}

func (s *OAuth2Server) ValidateToken(accessToken string) (*TokenClaims, error) {
	s.mu.RLock()
	claims, exists := s.accessTokens[accessToken]
	s.mu.RUnlock()
	
	if !exists {
		token, err := jwt.ParseWithClaims(accessToken, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})
		
		if err != nil {
			return nil, ErrInvalidToken
		}
		
		claims, ok := token.Claims.(*TokenClaims)
		if !ok || !token.Valid {
			return nil, ErrInvalidToken
		}
		
		return claims, nil
	}
	
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, ErrTokenExpired
	}
	
	return claims, nil
}

func (s *OAuth2Server) RevokeToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.accessTokens, token)
	delete(s.refreshTokens, token)
	
	return nil
}

func (s *OAuth2Server) GetUserInfo(accessToken string) (*UserInfo, error) {
	claims, err := s.ValidateToken(accessToken)
	if err != nil {
		return nil, err
	}
	
	return &UserInfo{
		UserID:    claims.UserID,
		Username:  fmt.Sprintf("user_%d", claims.UserID),
		Scopes:    claims.Scopes,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (s *OAuth2Server) verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method == "S256" {
		hash := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return secureCompare(computed, codeChallenge)
	}
	
	return secureCompare(codeVerifier, codeChallenge)
}

func (s *OAuth2Server) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		
		for code, authCode := range s.authorizationCodes {
			if now.After(authCode.ExpiresAt) {
				delete(s.authorizationCodes, code)
			}
		}
		
		for token, refreshToken := range s.refreshTokens {
			if now.After(refreshToken.ExpiresAt) {
				delete(s.refreshTokens, token)
			}
		}
		
		for token, claims := range s.accessTokens {
			if now.After(claims.ExpiresAt.Time) {
				delete(s.accessTokens, token)
			}
		}
		s.mu.Unlock()
	}
}

func generateSecureString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func scopesAsStrings(scopes []Scope) []string {
	result := make([]string, len(scopes))
	for i, s := range scopes {
		result[i] = string(s)
	}
	return result
}

func stringsAsScopes(strs []string) []Scope {
	result := make([]Scope, 0, len(strs))
	for _, s := range strs {
		if s != "" {
			result = append(result, Scope(s))
		}
	}
	return result
}

type IntrospectionRequest struct {
	Token           string `json:"token"`
	TokenTypeHint   string `json:"token_type_hint,omitempty"`
}

type IntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
	ExpiresAt int64    `json:"exp,omitempty"`
	IssuedAt  int64    `json:"iat,omitempty"`
	Subject   string   `json:"sub,omitempty"`
}

func (s *OAuth2Server) IntrospectToken(req *IntrospectionRequest) *IntrospectionResponse {
	resp := &IntrospectionResponse{Active: false}
	
	claims, err := s.ValidateToken(req.Token)
	if err != nil {
		return resp
	}
	
	resp.Active = true
	resp.Scope = strings.Join(scopesAsStrings(claims.Scopes), " ")
	resp.ClientID = claims.ClientID
	resp.Subject = fmt.Sprintf("%d", claims.UserID)
	resp.ExpiresAt = claims.ExpiresAt.Time.Unix()
	resp.IssuedAt = claims.IssuedAt.Time.Unix()
	resp.TokenType = "Bearer"
	
	return resp
}

type RevocationRequest struct {
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type RevocationResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *OAuth2Server) RevokeTokenRequest(req *RevocationRequest) *RevocationResponse {
	if req.ClientID != "" && req.ClientSecret != "" {
		_, err := s.ValidateClient(req.ClientID, req.ClientSecret, GrantRefreshToken)
		if err != nil {
			return &RevocationResponse{
				Success: false,
				Error:   "invalid_client",
			}
		}
	}
	
	err := s.RevokeToken(req.Token)
	if err != nil {
		return &RevocationResponse{
			Success: false,
			Error:   err.Error(),
		}
	}
	
	return &RevocationResponse{Success: true}
}

type DeviceAuthorizationRequest struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	Scope        []Scope  `json:"scope,omitempty"`
}

type DeviceAuthorizationResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Error           string `json:"error,omitempty"`
}

type DeviceTokenRequest struct {
	GrantType    string `json:"grant_type"`
	DeviceCode   string `json:"device_code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

type DeviceCodeEntry struct {
	DeviceCode   string    `json:"device_code"`
	UserCode     string    `json:"user_code"`
	ClientID     string    `json:"client_id"`
	Scopes       []Scope   `json:"scopes"`
	ExpiresAt    time.Time `json:"expires_at"`
	Interval     int       `json:"interval"`
	Approved     bool      `json:"approved"`
	UserID       uint      `json:"user_id,omitempty"`
	ApprovedAt   time.Time `json:"approved_at,omitempty"`
}

func (s *OAuth2Server) StartDeviceAuthorization(req *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
	client, err := s.ValidateClient(req.ClientID, req.ClientSecret, GrantClientCredentials)
	if err != nil {
		return nil, err
	}
	
	deviceCode := generateSecureString(64)
	userCode := generateUserCode()
	
	entry := &DeviceCodeEntry{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   client.ClientID,
		Scopes:     req.Scope,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		Interval:   5,
	}
	
	s.mu.Lock()
	s.deviceCodes[deviceCode] = entry
	s.mu.Unlock()
	
	return &DeviceAuthorizationResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: "/oauth/device",
		ExpiresIn:       600,
		Interval:        5,
	}, nil
}

func (s *OAuth2Server) ApproveDeviceCode(deviceCode string, userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	entry, exists := s.deviceCodes[deviceCode]
	if !exists {
		return fmt.Errorf("device code not found")
	}
	
	if time.Now().After(entry.ExpiresAt) {
		return fmt.Errorf("device code expired")
	}
	
	entry.Approved = true
	entry.UserID = userID
	entry.ApprovedAt = time.Now()
	
	return nil
}

func (s *OAuth2Server) PollDeviceToken(req *DeviceTokenRequest) (*DeviceTokenResponse, error) {
	if req.GrantType != "urn:ietf:params:oauth:grant-type:device_code" {
		return nil, ErrInvalidGrant
	}
	
	s.mu.RLock()
	entry, exists := s.deviceCodes[req.DeviceCode]
	s.mu.RUnlock()
	
	if !exists {
		return nil, ErrInvalidToken
	}
	
	if time.Now().After(entry.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	
	if !entry.Approved {
		return &DeviceTokenResponse{
			Error:      "authorization_pending",
			ErrorDesc:  "user has not yet approved the device",
		}, nil
	}
	
	token, err := s.generateToken(entry.ClientID, entry.UserID, entry.Scopes, "")
	if err != nil {
		return nil, err
	}
	
	return &DeviceTokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    string(token.TokenType),
		ExpiresIn:    token.ExpiresIn,
		RefreshToken: token.RefreshToken,
		Scope:        token.Scope,
	}, nil
}

func generateUserCode() string {
	nExp := big.NewInt(1)
	for i := 0; i < 8; i++ {
		nExp.Mul(nExp, big.NewInt(36))
	}
	
	random, _ := rand.Int(rand.Reader, nExp)
	code := random.Text(36)
	
	code = strings.ToUpper(code)
	if len(code) < 8 {
		code = strings.Repeat("0", 8-len(code)) + code
	}
	
	return code[:4] + "-" + code[4:]
}

type TokenEndpointRequest struct {
	GrantType    string `json:"grant_type" binding:"required"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CodeVerifier string `json:"code_verifier"`
}

type TokenEndpointResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

func (s *OAuth2Server) HandleTokenEndpoint(req *TokenEndpointRequest) *TokenEndpointResponse {
	var resp *TokenResponse
	var err error
	
	switch GrantType(req.GrantType) {
	case GrantAuthorizationCode:
		tokenReq := &TokenRequest{
			GrantType:    GrantAuthorizationCode,
			Code:        req.Code,
			RedirectURI: req.RedirectURI,
			ClientID:    req.ClientID,
			ClientSecret: req.ClientSecret,
			CodeVerifier: req.CodeVerifier,
		}
		resp, err = s.ExchangeCode(tokenReq)
		
	case GrantRefreshToken:
		tokenReq := &TokenRequest{
			GrantType:    GrantRefreshToken,
			RefreshToken: req.RefreshToken,
			ClientID:    req.ClientID,
			ClientSecret: req.ClientSecret,
		}
		resp, err = s.RefreshAccessToken(tokenReq)
		
	case GrantClientCredentials:
		resp, err = s.ClientCredentialsGrant(req.ClientID, req.ClientSecret, req.Scope)
		
	default:
		return &TokenEndpointResponse{
			Error:     "unsupported_grant_type",
			ErrorDesc: fmt.Sprintf("grant type '%s' is not supported", req.GrantType),
		}
	}
	
	if err != nil {
		return &TokenEndpointResponse{
			Error:     mapErrorToOAuthError(err),
			ErrorDesc: err.Error(),
		}
	}
	
	return &TokenEndpointResponse{
		AccessToken:  resp.AccessToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		RefreshToken: resp.RefreshToken,
		Scope:        resp.Scope,
	}
}

func (s *OAuth2Server) ClientCredentialsGrant(clientID, clientSecret, scopeStr string) (*TokenResponse, error) {
	client, err := s.ValidateClient(clientID, clientSecret, GrantClientCredentials)
	if err != nil {
		return nil, err
	}
	
	scopes := stringsAsScopes(strings.Split(scopeStr, " "))
	if len(scopes) == 0 {
		scopes = client.Scopes
	}
	
	token, err := s.generateToken(client.ClientID, 0, scopes, "")
	if err != nil {
		return nil, err
	}
	
	return &TokenResponse{
		AccessToken: token.AccessToken,
		TokenType:   string(token.TokenType),
		ExpiresIn:   token.ExpiresIn,
		Scope:       token.Scope,
	}, nil
}

func mapErrorToOAuthError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidToken):
		return "invalid_token"
	case errors.Is(err, ErrTokenExpired):
		return "invalid_token"
	case errors.Is(err, ErrInvalidGrant):
		return "invalid_grant"
	case errors.Is(err, ErrInvalidClient):
		return "invalid_client"
	case errors.Is(err, ErrInvalidScope):
		return "invalid_scope"
	case errors.Is(err, ErrAccessDenied):
		return "access_denied"
	case errors.Is(err, ErrUnauthorizedClient):
		return "unauthorized_client"
	default:
		return "server_error"
	}
}

type AuthorizationCodeEntry struct {
	Code        string    `json:"code"`
	ClientID    string    `json:"client_id"`
	UserID      uint      `json:"user_id"`
	RedirectURI string    `json:"redirect_uri"`
	Scopes      []Scope   `json:"scopes"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
}

func (s *OAuth2Server) ListActiveDeviceCodes() []*DeviceCodeEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entries := make([]*DeviceCodeEntry, 0, len(s.deviceCodes))
	now := time.Now()
	
	for _, entry := range s.deviceCodes {
		if now.Before(entry.ExpiresAt) && !entry.Approved {
			entries = append(entries, entry)
		}
	}
	
	return entries
}

func (s *OAuth2Server) GetAuthorizationCode(code string) (*AuthorizationCodeEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	authCode, exists := s.authorizationCodes[code]
	if !exists || authCode.Used {
		return nil, false
	}
	
	if time.Now().After(authCode.ExpiresAt) {
		return nil, false
	}
	
	return &AuthorizationCodeEntry{
		Code:        authCode.Code,
		ClientID:    authCode.ClientID,
		UserID:      authCode.UserID,
		RedirectURI: authCode.RedirectURI,
		Scopes:      authCode.Scopes,
		ExpiresAt:   authCode.ExpiresAt,
		Used:        authCode.Used,
	}, true
}

func (s *OAuth2Server) RevokeAllClientTokens(clientID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	count := 0
	
	for token, claims := range s.accessTokens {
		if claims.ClientID == clientID {
			delete(s.accessTokens, token)
			count++
		}
	}
	
	for token, tok := range s.refreshTokens {
		if tok.AccessToken != "" {
			if _, exists := s.accessTokens[tok.AccessToken]; exists {
				delete(s.accessTokens, tok.AccessToken)
			}
		}
		delete(s.refreshTokens, token)
		count++
	}
	
	return count
}

func (s *OAuth2Server) GetServerStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"total_clients":          len(s.clients),
		"active_auth_codes":      len(s.authorizationCodes),
		"active_refresh_tokens":  len(s.refreshTokens),
		"active_access_tokens":   len(s.accessTokens),
		"code_expiry":            s.codeExpiry.String(),
		"token_expiry":           s.tokenExpiry.String(),
		"refresh_token_expiry":   s.refreshTokenExpiry.String(),
		"issuer":                 s.issuer,
	}
}

func (s *OAuth2Server) ValidateScope(requestedScopes, clientScopes []Scope) bool {
	scopeSet := make(map[Scope]bool)
	for _, scope := range clientScopes {
		scopeSet[scope] = true
	}
	
	for _, scope := range requestedScopes {
		if !scopeSet[scope] {
			return false
		}
	}
	
	return true
}

func (s *OAuth2Server) GenerateIDToken(clientID string, userID uint) (string, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)
	
	claims := jwt.MapClaims{
		"iss":     s.issuer,
		"sub":     fmt.Sprintf("%d", userID),
		"aud":     clientID,
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
		"auth_time": now.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *OAuth2Server) SetCustomClaims(accessToken string, claims map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	entry, exists := s.accessTokens[accessToken]
	if !exists {
		return fmt.Errorf("token not found")
	}
	
	claimsJSON, _ := json.Marshal(claims)
	entry.Scopes = append(entry.Scopes, Scope(string(claimsJSON)))
	
	return nil
}

type PKCEConfig struct {
	Required       bool
	ChallengeMethods []string
	DefaultMethod   string
}

func DefaultPKCEConfig() PKCEConfig {
	return PKCEConfig{
		Required:        false,
		ChallengeMethods: []string{"S256", "plain"},
		DefaultMethod:   "S256",
	}
}

func GenerateCodeChallenge(codeVerifier, method string) string {
	if method == "S256" || method == "" {
		hash := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(hash[:])
	}
	return codeVerifier
}

func ValidateCodeChallenge(codeVerifier, codeChallenge, method string) bool {
	computed := GenerateCodeChallenge(codeVerifier, method)
	return secureCompare(computed, codeChallenge)
}
