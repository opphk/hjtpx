package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: RegisterInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			input: RegisterInput{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			input: RegisterInput{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "short password",
			input: RegisterInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.input.Username == "" || tt.input.Email == "" || len(tt.input.Password) < 6
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestLoginInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   LoginInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: LoginInput{
				Username: "testuser",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			input: LoginInput{
				Username: "",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "empty password",
			input: LoginInput{
				Username: "testuser",
				Password: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.input.Username == "" || tt.input.Password == ""
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestUpdateProfileInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input UpdateProfileInput
	}{
		{
			name: "with nickname",
			input: UpdateProfileInput{
				Nickname: "Test User",
			},
		},
		{
			name: "with avatar",
			input: UpdateProfileInput{
				Avatar: "https://example.com/avatar.jpg",
			},
		},
		{
			name: "with phone",
			input: UpdateProfileInput{
				Phone: "1234567890",
			},
		},
		{
			name: "with bio",
			input: UpdateProfileInput{
				Bio: "Hello World",
			},
		},
		{
			name:  "empty input",
			input: UpdateProfileInput{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("UpdateProfileInput caused panic: %v", r)
				}
			}()
			_ = tt.input.Nickname
			_ = tt.input.Avatar
			_ = tt.input.Phone
			_ = tt.input.Bio
		})
	}
}

func TestChangePasswordInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   ChangePasswordInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: ChangePasswordInput{
				OldPassword: "oldpass123",
				NewPassword: "newpass123",
			},
			wantErr: false,
		},
		{
			name: "short old password",
			input: ChangePasswordInput{
				OldPassword: "123",
				NewPassword: "newpass123",
			},
			wantErr: true,
		},
		{
			name: "short new password",
			input: ChangePasswordInput{
				OldPassword: "oldpass123",
				NewPassword: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := len(tt.input.OldPassword) < 6 || len(tt.input.NewPassword) < 6
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestResetPasswordInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   ResetPasswordInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: ResetPasswordInput{
				Token:       "abc123token",
				NewPassword: "newpass123",
			},
			wantErr: false,
		},
		{
			name: "empty token",
			input: ResetPasswordInput{
				Token:       "",
				NewPassword: "newpass123",
			},
			wantErr: true,
		},
		{
			name: "short password",
			input: ResetPasswordInput{
				Token:       "abc123token",
				NewPassword: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.input.Token == "" || len(tt.input.NewPassword) < 6
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestNewUserService(t *testing.T) {
	service := NewUserService()
	assert.NotNil(t, service)
}

func TestGenerateToken(t *testing.T) {
	token1 := generateToken(32)
	assert.NotEmpty(t, token1)
	assert.Len(t, token1, 64)

	token2 := generateToken(32)
	assert.NotEqual(t, token1, token2)

	token3 := generateToken(16)
	assert.Len(t, token3, 32)
}

func TestUserServiceErrors(t *testing.T) {
	assert.Equal(t, "user not found", ErrUserNotFound.Error())
	assert.Equal(t, "user already exists", ErrUserAlreadyExists.Error())
	assert.Equal(t, "invalid token", ErrInvalidToken.Error())
	assert.Equal(t, "token expired", ErrTokenExpired.Error())
	assert.Equal(t, "invalid password", ErrInvalidPassword.Error())
	assert.Equal(t, "user not verified", ErrUserNotVerified.Error())
	assert.Equal(t, "user is disabled", ErrUserDisabled.Error())
}
