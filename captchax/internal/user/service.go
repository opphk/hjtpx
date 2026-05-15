package user

import (
	"errors"
	"time"

	"captchax/internal/model"
	"captchax/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Service struct {
	repo      *repository.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(repo *repository.UserRepository, jwtSecret string, tokenTTLSeconds int) *Service {
	ttl := time.Duration(tokenTTLSeconds) * time.Second
	if tokenTTLSeconds <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  ttl,
	}
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) Register(req *model.UserRegisterRequest) (*model.User, error) {
	exists, err := s.repo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	emailExists, err := s.repo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, errors.New("email already exists")
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Name:     req.Name,
		Role:     "user",
		Status:   "active",
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Login(req *model.UserLoginRequest) (*model.UserTokenResponse, error) {
	user, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if !user.CheckPassword(req.Password) {
		return nil, errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user.LastLogin = &now
	s.repo.Update(user)

	return &model.UserTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

func (s *Service) generateToken(user *model.User) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(s.tokenTTL)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "captchax",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return signedToken, expiresAt.Unix(), nil
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *Service) GetUser(id uint) (*model.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) UpdateUser(id uint, req *model.UserUpdateRequest) (*model.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Email != "" && req.Email != user.Email {
		emailExists, err := s.repo.ExistsByEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if emailExists {
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) DeleteUser(id uint) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	return s.repo.Delete(id)
}

func (s *Service) ListUsers(page, pageSize int, search, role, status string) ([]model.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, search, role, status)
}

func (s *Service) ChangePassword(id uint, req *model.UserChangePasswordRequest) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	if !user.CheckPassword(req.OldPassword) {
		return errors.New("old password is incorrect")
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		return err
	}

	return s.repo.Update(user)
}

func (s *Service) UpdateRole(id uint, role string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	user.Role = role
	return s.repo.Update(user)
}

func (s *Service) UpdateStatus(id uint, status string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	user.Status = status
	return s.repo.Update(user)
}