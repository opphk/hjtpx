package sdk

import (
	"encoding/json"
)

const (
	AuthRegisterPath            = "/auth/register"
	AuthLoginPath               = "/auth/login"
	AuthLogoutPath              = "/auth/logout"
	AuthRefreshPath             = "/auth/refresh"
	AuthVerifyEmailPath         = "/auth/verify-email"
	AuthResendVerificationPath  = "/auth/resend-verification"
	AuthRequestPasswordResetPath = "/auth/request-password-reset"
	AuthResetPasswordPath      = "/auth/reset-password"
)

type RegisterRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	BehaviorData string `json:"behavior_data,omitempty"`
}

type RegisterResponse struct {
	UserID          uint   `json:"user_id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	VerificationLink string `json:"verification_link,omitempty"`
	Message        string `json:"message,omitempty"`
}

type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token,omitempty"`
	BehaviorData string `json:"behavior_data,omitempty"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type VerifyEmailResponse struct {
	Message string `json:"message"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type ResendVerificationResponse struct {
	VerificationLink string `json:"verification_link,omitempty"`
	Message        string `json:"message,omitempty"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

type RequestPasswordResetResponse struct {
	ResetLink string `json:"reset_link,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type AuthClient struct {
	*Client
}

func NewAuthClient(client *Client) *AuthClient {
	return &AuthClient{Client: client}
}

func (c *Client) Auth() *AuthClient {
	return NewAuthClient(c)
}

func (ac *AuthClient) Register(req *RegisterRequest) (*RegisterResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.Username == "" {
		return nil, NewSDKError(400, "username is required")
	}
	if req.Email == "" {
		return nil, NewSDKError(400, "email is required")
	}
	if req.Password == "" {
		return nil, NewSDKError(400, "password is required")
	}

	resp, err := ac.doRequestWithRetry("POST", AuthRegisterPath, req)
	if err != nil {
		return nil, err
	}

	var result RegisterResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) Login(req *LoginRequest) (*LoginResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.Username == "" {
		return nil, NewSDKError(400, "username is required")
	}
	if req.Password == "" {
		return nil, NewSDKError(400, "password is required")
	}

	resp, err := ac.doRequestWithRetry("POST", AuthLoginPath, req)
	if err != nil {
		return nil, err
	}

	var result LoginResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) Logout() error {
	_, err := ac.doRequestWithRetry("POST", AuthLogoutPath, nil)
	return err
}

func (ac *AuthClient) RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.RefreshToken == "" {
		return nil, NewSDKError(400, "refresh_token is required")
	}

	resp, err := ac.doRequestWithRetry("POST", AuthRefreshPath, req)
	if err != nil {
		return nil, err
	}

	var result RefreshTokenResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) VerifyEmail(token string) (*VerifyEmailResponse, error) {
	if token == "" {
		return nil, NewSDKError(400, "token is required")
	}

	resp, err := ac.doRequestWithRetry("GET", AuthVerifyEmailPath+"?token="+token, nil)
	if err != nil {
		return nil, err
	}

	var result VerifyEmailResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) ResendVerification(email string) (*ResendVerificationResponse, error) {
	if email == "" {
		return nil, NewSDKError(400, "email is required")
	}

	req := &ResendVerificationRequest{Email: email}
	resp, err := ac.doRequestWithRetry("POST", AuthResendVerificationPath, req)
	if err != nil {
		return nil, err
	}

	var result ResendVerificationResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) RequestPasswordReset(email string) (*RequestPasswordResetResponse, error) {
	if email == "" {
		return nil, NewSDKError(400, "email is required")
	}

	req := &RequestPasswordResetRequest{Email: email}
	resp, err := ac.doRequestWithRetry("POST", AuthRequestPasswordResetPath, req)
	if err != nil {
		return nil, err
	}

	var result RequestPasswordResetResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (ac *AuthClient) ResetPassword(req *ResetPasswordRequest) (*ResetPasswordResponse, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}
	if req.Token == "" {
		return nil, NewSDKError(400, "token is required")
	}
	if req.NewPassword == "" {
		return nil, NewSDKError(400, "new_password is required")
	}

	resp, err := ac.doRequestWithRetry("POST", AuthResetPasswordPath, req)
	if err != nil {
		return nil, err
	}

	var result ResetPasswordResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

const (
	UserProfilePath          = "/user/profile"
	UserChangePasswordPath  = "/user/change-password"
)

type UserProfile struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Nickname   string `json:"nickname,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Bio        string `json:"bio,omitempty"`
	IsVerified bool   `json:"is_verified,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Bio      string `json:"bio,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UserClient struct {
	*Client
}

func NewUserClient(client *Client) *UserClient {
	return &UserClient{Client: client}
}

func (c *Client) User() *UserClient {
	return NewUserClient(c)
}

func (uc *UserClient) GetProfile() (*UserProfile, error) {
	resp, err := uc.doRequestWithRetry("GET", UserProfilePath, nil)
	if err != nil {
		return nil, err
	}

	var result UserProfile
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (uc *UserClient) UpdateProfile(req *UpdateProfileRequest) (*UserProfile, error) {
	if req == nil {
		return nil, NewSDKError(400, "request cannot be nil")
	}

	resp, err := uc.doRequestWithRetry("PUT", UserProfilePath, req)
	if err != nil {
		return nil, err
	}

	var result UserProfile
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (uc *UserClient) ChangePassword(req *ChangePasswordRequest) error {
	if req == nil {
		return NewSDKError(400, "request cannot be nil")
	}
	if req.OldPassword == "" {
		return NewSDKError(400, "old_password is required")
	}
	if req.NewPassword == "" {
		return NewSDKError(400, "new_password is required")
	}

	_, err := uc.doRequestWithRetry("POST", UserChangePasswordPath, req)
	return err
}
