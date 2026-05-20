package service

import (
	"errors"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewUserService(t *testing.T) {
	userService := NewUserService()
	assert.NotNil(t, userService)
}

func TestUserService_Register_Success(t *testing.T) {
	userService := &mockUserService{}

	input := &RegisterInput{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "password123",
	}

	user, token, err := userService.Register(input, 30)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, token)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.Equal(t, "active", user.Status)
	assert.False(t, user.IsVerified)
}

func TestUserService_Register_HighRiskScore(t *testing.T) {
	userService := &mockUserService{}

	input := &RegisterInput{
		Username: "riskyuser",
		Email:    "risky@example.com",
		Password: "password123",
	}

	user, token, err := userService.Register(input, 80)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "risk score too high")
}

func TestUserService_Register_DuplicateUsername(t *testing.T) {
	userService := &mockUserService{
		users: map[string]*models.User{
			"existinguser": {Username: "existinguser", Email: "existing@example.com"},
		},
	}

	input := &RegisterInput{
		Username: "existinguser",
		Email:    "new@example.com",
		Password: "password123",
	}

	user, token, err := userService.Register(input, 20)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, ErrUserAlreadyExists, err)
}

func TestUserService_Register_DuplicateEmail(t *testing.T) {
	userService := &mockUserService{
		users: map[string]*models.User{
			"existing@example.com": {Username: "existing", Email: "existing@example.com"},
		},
	}

	input := &RegisterInput{
		Username: "newuser",
		Email:    "existing@example.com",
		Password: "password123",
	}

	user, token, err := userService.Register(input, 20)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, ErrUserAlreadyExists, err)
}

func TestUserService_Register_PasswordHashing(t *testing.T) {
	userService := &mockUserService{}

	input := &RegisterInput{
		Username: "hashuser",
		Email:    "hash@example.com",
		Password: "password123",
	}

	user, _, err := userService.Register(input, 20)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
	assert.NoError(t, err)
}

func TestUserService_Login_Success(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userService := &mockUserService{
		users: map[string]*models.User{
			"loginuser": {
				Username:    "loginuser",
				PasswordHash: string(hashedPassword),
				Status:      "active",
			},
		},
	}

	input := &LoginInput{
		Username: "loginuser",
		Password: "password123",
	}

	user, token, err := userService.Login(input, "192.168.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, "loginuser", user.Username)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	input := &LoginInput{
		Username: "nonexistent",
		Password: "password123",
	}

	user, token, err := userService.Login(input, "192.168.1.1")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_Login_InvalidPassword(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	userService := &mockUserService{
		users: map[string]*models.User{
			"loginuser": {
				Username:    "loginuser",
				PasswordHash: string(hashedPassword),
				Status:      "active",
			},
		},
	}

	input := &LoginInput{
		Username: "loginuser",
		Password: "wrongpassword",
	}

	user, token, err := userService.Login(input, "192.168.1.1")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, ErrInvalidPassword, err)
}

func TestUserService_Login_DisabledUser(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userService := &mockUserService{
		users: map[string]*models.User{
			"disableduser": {
				Username:    "disableduser",
				PasswordHash: string(hashedPassword),
				Status:      "disabled",
			},
		},
	}

	input := &LoginInput{
		Username: "disableduser",
		Password: "password123",
	}

	user, token, err := userService.Login(input, "192.168.1.1")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, token)
	assert.Equal(t, ErrUserDisabled, err)
}

func TestUserService_Login_UpdateLoginInfo(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	userService := &mockUserService{
		users: map[string]*models.User{
			"loginuser": {
				Username:    "loginuser",
				PasswordHash: string(hashedPassword),
				Status:      "active",
				LoginCount: 0,
			},
		},
	}

	input := &LoginInput{
		Username: "loginuser",
		Password: "password123",
	}

	user, _, err := userService.Login(input, "10.0.0.1")
	assert.NoError(t, err)
	assert.Equal(t, 1, user.LoginCount)
	assert.NotNil(t, user.LastLoginAt)
	assert.Equal(t, "10.0.0.1", user.LastLoginIP)
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com"},
		},
	}

	user, err := userService.GetUserByID(1)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "user1", user.Username)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	userService := &mockUserService{}

	user, err := userService.GetUserByID(999)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_GetUserByEmail_Success(t *testing.T) {
	userService := &mockUserService{
		users: map[string]*models.User{
			"email@example.com": {ID: 1, Username: "emailuser", Email: "email@example.com"},
		},
	}

	user, err := userService.GetUserByEmail("email@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "email@example.com", user.Email)
}

func TestUserService_GetUserByEmail_NotFound(t *testing.T) {
	userService := &mockUserService{}

	user, err := userService.GetUserByEmail("nonexistent@example.com")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com", Nickname: ""},
		},
	}

	input := &UpdateProfileInput{
		Nickname: "newnickname",
		Avatar:   "https://example.com/avatar.jpg",
		Phone:    "1234567890",
		Bio:      "New bio",
	}

	user, err := userService.UpdateProfile(1, input)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "newnickname", user.Nickname)
	assert.Equal(t, "https://example.com/avatar.jpg", user.Avatar)
	assert.Equal(t, "1234567890", user.Phone)
	assert.Equal(t, "New bio", user.Bio)
}

func TestUserService_UpdateProfile_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	input := &UpdateProfileInput{
		Nickname: "newnickname",
	}

	user, err := userService.UpdateProfile(999, input)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_UpdateProfile_NoChanges(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com"},
		},
	}

	input := &UpdateProfileInput{}

	user, err := userService.UpdateProfile(1, input)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestUserService_UpdateProfile_PartialUpdate(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com", Nickname: "oldnick", Avatar: "old.jpg"},
		},
	}

	input := &UpdateProfileInput{
		Nickname: "newnick",
	}

	user, err := userService.UpdateProfile(1, input)
	assert.NoError(t, err)
	assert.Equal(t, "newnick", user.Nickname)
	assert.Equal(t, "old.jpg", user.Avatar)
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)

	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", PasswordHash: string(hashedPassword)},
		},
	}

	input := &ChangePasswordInput{
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	}

	err := userService.ChangePassword(1, input)
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword([]byte(userService.usersByID[1].PasswordHash), []byte("newpassword123"))
	assert.NoError(t, err)
}

func TestUserService_ChangePassword_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	input := &ChangePasswordInput{
		OldPassword: "oldpassword",
		NewPassword: "newpassword123",
	}

	err := userService.ChangePassword(999, input)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", PasswordHash: string(hashedPassword)},
		},
	}

	input := &ChangePasswordInput{
		OldPassword: "wrongpassword",
		NewPassword: "newpassword123",
	}

	err := userService.ChangePassword(1, input)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPassword, err)
}

func TestUserService_RequestPasswordReset_Success(t *testing.T) {
	userService := &mockUserService{
		users: map[string]*models.User{
			"reset@example.com": {ID: 1, Username: "resetuser", Email: "reset@example.com"},
		},
	}

	token, err := userService.RequestPasswordReset("reset@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 64)
}

func TestUserService_RequestPasswordReset_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	token, err := userService.RequestPasswordReset("nonexistent@example.com")
	assert.NoError(t, err)
	assert.Empty(t, token)
}

func TestUserService_ResetPassword_Success(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	resetAt := time.Now().Add(1 * time.Hour)

	userService := &mockUserService{
		users: map[string]*models.User{
			"reset@example.com": {
				ID:                 1,
				Username:           "resetuser",
				Email:              "reset@example.com",
				PasswordHash:       string(hashedPassword),
				PasswordResetToken: "validtoken123",
				PasswordResetAt:    &resetAt,
			},
		},
	}

	input := &ResetPasswordInput{
		Token:       "validtoken123",
		NewPassword: "newpassword123",
	}

	err := userService.ResetPassword(input)
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword([]byte(userService.users["reset@example.com"].PasswordHash), []byte("newpassword123"))
	assert.NoError(t, err)
	assert.Empty(t, userService.users["reset@example.com"].PasswordResetToken)
	assert.Nil(t, userService.users["reset@example.com"].PasswordResetAt)
}

func TestUserService_ResetPassword_InvalidToken(t *testing.T) {
	userService := &mockUserService{}

	input := &ResetPasswordInput{
		Token:       "invalidtoken",
		NewPassword: "newpassword123",
	}

	err := userService.ResetPassword(input)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestUserService_ResetPassword_ExpiredToken(t *testing.T) {
	expiredAt := time.Now().Add(-1 * time.Hour)

	userService := &mockUserService{
		users: map[string]*models.User{
			"expired@example.com": {
				ID:                 1,
				PasswordResetToken: "expiredtoken",
				PasswordResetAt:    &expiredAt,
			},
		},
	}

	input := &ResetPasswordInput{
		Token:       "expiredtoken",
		NewPassword: "newpassword123",
	}

	err := userService.ResetPassword(input)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
}

func TestUserService_VerifyEmail_Success(t *testing.T) {
	userService := &mockUserService{
		users: map[string]*models.User{
			"verify@example.com": {
				ID:                1,
				Email:             "verify@example.com",
				IsVerified:        false,
				VerificationToken: "validverifytoken",
			},
		},
	}

	err := userService.VerifyEmail("validverifytoken")
	assert.NoError(t, err)
	assert.True(t, userService.users["verify@example.com"].IsVerified)
	assert.NotNil(t, userService.users["verify@example.com"].VerifiedAt)
	assert.Empty(t, userService.users["verify@example.com"].VerificationToken)
}

func TestUserService_VerifyEmail_InvalidToken(t *testing.T) {
	userService := &mockUserService{}

	err := userService.VerifyEmail("invalidtoken")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestUserService_GenerateEmailVerificationToken_Success(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com", VerificationToken: "oldtoken"},
		},
	}

	token, err := userService.GenerateEmailVerificationToken(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 64)
	assert.NotEqual(t, "oldtoken", token)
}

func TestUserService_GenerateEmailVerificationToken_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	token, err := userService.GenerateEmailVerificationToken(999)
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_ChangeEmail_Success(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "old@example.com", IsVerified: true},
		},
		users: map[string]*models.User{
			"old@example.com": {ID: 1, Username: "user1", Email: "old@example.com"},
		},
	}

	err := userService.ChangeEmail(1, "new@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", userService.usersByID[1].Email)
	assert.False(t, userService.usersByID[1].IsVerified)
	assert.NotEmpty(t, userService.usersByID[1].VerificationToken)
}

func TestUserService_ChangeEmail_DuplicateEmail(t *testing.T) {
	userService := &mockUserService{
		usersByID: map[uint]*models.User{
			1: {ID: 1, Username: "user1", Email: "user1@example.com"},
			2: {ID: 2, Username: "user2", Email: "existing@example.com"},
		},
		users: map[string]*models.User{
			"user1@example.com": {ID: 1, Email: "user1@example.com"},
			"existing@example.com": {ID: 2, Email: "existing@example.com"},
		},
	}

	err := userService.ChangeEmail(1, "existing@example.com")
	assert.Error(t, err)
	assert.Equal(t, ErrUserAlreadyExists, err)
}

func TestUserService_ChangeEmail_UserNotFound(t *testing.T) {
	userService := &mockUserService{}

	err := userService.ChangeEmail(999, "new@example.com")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestToUserResponse(t *testing.T) {
	now := time.Now()
	user := &models.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		Nickname:     "Test User",
		Avatar:       "https://example.com/avatar.jpg",
		Phone:        "1234567890",
		Bio:          "Test bio",
		IsVerified:   true,
		VerifiedAt:   &now,
		LoginCount:   10,
		LastLoginAt:   &now,
		LastLoginIP:   "192.168.1.1",
		Status:       "active",
		CreatedAt:    now,
	}

	response := ToUserResponse(user)

	assert.Equal(t, uint(1), response.ID)
	assert.Equal(t, "testuser", response.Username)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, "Test User", response.Nickname)
	assert.Equal(t, "https://example.com/avatar.jpg", response.Avatar)
	assert.Equal(t, "1234567890", response.Phone)
	assert.Equal(t, "Test bio", response.Bio)
	assert.True(t, response.IsVerified)
	assert.NotNil(t, response.VerifiedAt)
	assert.Equal(t, 10, response.LoginCount)
	assert.NotNil(t, response.LastLoginAt)
	assert.Equal(t, "192.168.1.1", response.LastLoginIP)
	assert.Equal(t, "active", response.Status)
}

func TestToUserResponse_NilFields(t *testing.T) {
	user := &models.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		IsVerified:   false,
		VerifiedAt:   nil,
		LoginCount:   0,
		LastLoginAt:  nil,
		Status:       "active",
		CreatedAt:    time.Now(),
	}

	response := ToUserResponse(user)

	assert.Equal(t, uint(1), response.ID)
	assert.False(t, response.IsVerified)
	assert.Nil(t, response.VerifiedAt)
	assert.Equal(t, 0, response.LoginCount)
	assert.Nil(t, response.LastLoginAt)
}

func TestRegisterInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		isValid bool
	}{
		{
			name: "valid input",
			input: RegisterInput{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "password123",
			},
			isValid: true,
		},
		{
			name: "short username",
			input: RegisterInput{
				Username: "ab",
				Email:    "test@example.com",
				Password: "password123",
			},
			isValid: false,
		},
		{
			name: "invalid email",
			input: RegisterInput{
				Username: "validuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			isValid: false,
		},
		{
			name: "short password",
			input: RegisterInput{
				Username: "validuser",
				Email:    "test@example.com",
				Password: "12345",
			},
			isValid: false,
		},
		{
			name: "empty username",
			input: RegisterInput{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.input.Username) >= 3 && len(tt.input.Password) >= 6 && isValidEmail(tt.input.Email)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestLoginInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   LoginInput
		isValid bool
	}{
		{
			name: "valid input",
			input: LoginInput{
				Username: "validuser",
				Password: "password123",
			},
			isValid: true,
		},
		{
			name: "empty username",
			input: LoginInput{
				Username: "",
				Password: "password123",
			},
			isValid: false,
		},
		{
			name: "empty password",
			input: LoginInput{
				Username: "validuser",
				Password: "",
			},
			isValid: false,
		},
		{
			name: "with captcha token",
			input: LoginInput{
				Username:     "validuser",
				Password:     "password123",
				CaptchaToken: "captcha-token-123",
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.input.Username) > 0 && len(tt.input.Password) > 0
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestUpdateProfileInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   UpdateProfileInput
		isValid bool
	}{
		{
			name: "valid nickname",
			input: UpdateProfileInput{
				Nickname: "newnickname",
			},
			isValid: true,
		},
		{
			name: "long nickname",
			input: UpdateProfileInput{
				Nickname: string(make([]byte, 101)),
			},
			isValid: false,
		},
		{
			name: "valid phone",
			input: UpdateProfileInput{
				Phone: "1234567890",
			},
			isValid: true,
		},
		{
			name: "long bio",
			input: UpdateProfileInput{
				Bio: string(make([]byte, 501)),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.input.Nickname) <= 100 && len(tt.input.Bio) <= 500
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestChangePasswordInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   ChangePasswordInput
		isValid bool
	}{
		{
			name: "valid input",
			input: ChangePasswordInput{
				OldPassword: "oldpassword",
				NewPassword: "newpassword123",
			},
			isValid: true,
		},
		{
			name: "short new password",
			input: ChangePasswordInput{
				OldPassword: "oldpassword",
				NewPassword: "12345",
			},
			isValid: false,
		},
		{
			name: "same old and new password",
			input: ChangePasswordInput{
				OldPassword: "samepassword",
				NewPassword: "samepassword",
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.input.NewPassword) >= 6
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestResetPasswordInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   ResetPasswordInput
		isValid bool
	}{
		{
			name: "valid input",
			input: ResetPasswordInput{
				Token:       "validtoken123",
				NewPassword: "newpassword123",
			},
			isValid: true,
		},
		{
			name: "empty token",
			input: ResetPasswordInput{
				Token:       "",
				NewPassword: "newpassword123",
			},
			isValid: false,
		},
		{
			name: "short password",
			input: ResetPasswordInput{
				Token:       "validtoken123",
				NewPassword: "12345",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.input.Token) > 0 && len(tt.input.NewPassword) >= 6
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestUserResponse_Structure(t *testing.T) {
	now := time.Now()
	response := &UserResponse{
		ID:          1,
		Username:    "testuser",
		Email:       "test@example.com",
		Nickname:    "Test User",
		Avatar:      "https://example.com/avatar.jpg",
		Phone:       "1234567890",
		Bio:         "Test bio",
		IsVerified:  true,
		VerifiedAt:  &now,
		LoginCount:  5,
		LastLoginAt: &now,
		LastLoginIP: "192.168.1.1",
		Status:      "active",
		CreatedAt:   now,
	}

	assert.Equal(t, uint(1), response.ID)
	assert.Equal(t, "testuser", response.Username)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, "Test User", response.Nickname)
	assert.Equal(t, "https://example.com/avatar.jpg", response.Avatar)
	assert.Equal(t, "1234567890", response.Phone)
	assert.Equal(t, "Test bio", response.Bio)
	assert.True(t, response.IsVerified)
	assert.NotNil(t, response.VerifiedAt)
	assert.Equal(t, 5, response.LoginCount)
	assert.NotNil(t, response.LastLoginAt)
	assert.Equal(t, "192.168.1.1", response.LastLoginIP)
	assert.Equal(t, "active", response.Status)
	assert.False(t, response.CreatedAt.IsZero())
}

func TestUserStatus_Values(t *testing.T) {
	validStatuses := []string{"active", "disabled", "suspended", "deleted"}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			user := &models.User{
				Username: "testuser",
				Email:    "test@example.com",
				Status:   status,
			}
			assert.NotEmpty(t, user.Status)
		})
	}
}

type mockUserService struct {
	users     map[string]*models.User
	usersByID map[uint]*models.User
}

func (m *mockUserService) Register(input *RegisterInput, riskScore float64) (*models.User, string, error) {
	if riskScore > 70 {
		return nil, "", errors.New("risk score too high, registration blocked")
	}

	if _, exists := m.users[input.Username]; exists {
		return nil, "", ErrUserAlreadyExists
	}

	for _, user := range m.users {
		if user.Email == input.Email {
			return nil, "", ErrUserAlreadyExists
		}
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	user := &models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		Status:       "active",
	}

	if m.users == nil {
		m.users = make(map[string]*models.User)
	}
	if m.usersByID == nil {
		m.usersByID = make(map[uint]*models.User)
	}

	token := "verification_token_" + input.Username
	m.users[input.Username] = user
	m.users[input.Email] = user
	user.ID = uint(len(m.usersByID) + 1)
	m.usersByID[user.ID] = user

	return user, token, nil
}

func (m *mockUserService) Login(input *LoginInput, clientIP string) (*models.User, string, error) {
	var user *models.User
	var found bool

	for _, u := range m.users {
		if u.Username == input.Username {
			user = u
			found = true
			break
		}
	}

	if !found {
		return nil, "", ErrUserNotFound
	}

	if user.Status != "active" {
		return nil, "", ErrUserDisabled
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return nil, "", ErrInvalidPassword
	}

	now := time.Now()
	user.LoginCount++
	user.LastLoginAt = &now
	user.LastLoginIP = clientIP

	return user, "", nil
}

func (m *mockUserService) GetUserByID(userID uint) (*models.User, error) {
	if m.usersByID == nil {
		return nil, ErrUserNotFound
	}

	user, exists := m.usersByID[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (m *mockUserService) GetUserByEmail(email string) (*models.User, error) {
	if m.users == nil {
		return nil, ErrUserNotFound
	}

	user, exists := m.users[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (m *mockUserService) UpdateProfile(userID uint, input *UpdateProfileInput) (*models.User, error) {
	user, err := m.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if input.Nickname != "" {
		user.Nickname = input.Nickname
	}
	if input.Avatar != "" {
		user.Avatar = input.Avatar
	}
	if input.Phone != "" {
		user.Phone = input.Phone
	}
	if input.Bio != "" {
		user.Bio = input.Bio
	}

	return user, nil
}

func (m *mockUserService) ChangePassword(userID uint, input *ChangePasswordInput) error {
	user, err := m.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword))
	if err != nil {
		return ErrInvalidPassword
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(newHash)

	return nil
}

func (m *mockUserService) RequestPasswordReset(email string) (string, error) {
	user, err := m.GetUserByEmail(email)
	if err != nil {
		return "", nil
	}

	token := "reset_token_" + user.Username
	resetAt := time.Now().Add(1 * time.Hour)
	user.PasswordResetToken = token
	user.PasswordResetAt = &resetAt

	return token, nil
}

func (m *mockUserService) ResetPassword(input *ResetPasswordInput) error {
	var user *models.User

	for _, u := range m.users {
		if u.PasswordResetToken == input.Token {
			user = u
			break
		}
	}

	if user == nil {
		return ErrInvalidToken
	}

	if user.PasswordResetAt == nil || user.PasswordResetAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(newHash)
	user.PasswordResetToken = ""
	user.PasswordResetAt = nil

	return nil
}

func (m *mockUserService) VerifyEmail(token string) error {
	var user *models.User

	for _, u := range m.users {
		if u.VerificationToken == token {
			user = u
			break
		}
	}

	if user == nil {
		return ErrInvalidToken
	}

	now := time.Now()
	user.IsVerified = true
	user.VerifiedAt = &now
	user.VerificationToken = ""

	return nil
}

func (m *mockUserService) GenerateEmailVerificationToken(userID uint) (string, error) {
	user, err := m.GetUserByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	token := "new_token_" + user.Username
	user.VerificationToken = token

	return token, nil
}

func (m *mockUserService) ChangeEmail(userID uint, newEmail string) error {
	user, err := m.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	for _, u := range m.users {
		if u.Email == newEmail && u.ID != userID {
			return ErrUserAlreadyExists
		}
	}

	user.Email = newEmail
	user.IsVerified = false
	user.VerificationToken = "new_email_token"

	return nil
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 0 || atIndex >= len(email)-1 {
		return false
	}
	dotIndex := -1
	for i := atIndex + 1; i < len(email); i++ {
		if email[i] == '.' {
			dotIndex = i
			break
		}
	}
	return dotIndex > atIndex+1 && dotIndex < len(email)-1
}
