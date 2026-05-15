package user

import (
	"captchax/internal/repository"

	"gorm.io/gorm"
)

type Module struct {
	handlers       *Handlers
	authMiddleware *AuthMiddleware
}

func New(db *gorm.DB, jwtSecret string, tokenTTLSeconds int) *Module {
	userRepo := repository.NewUserRepository(db)
	svc := NewService(userRepo, jwtSecret, tokenTTLSeconds)
	handlers := NewHandlers(svc)
	authMiddleware := NewAuthMiddleware(svc)

	return &Module{
		handlers:       handlers,
		authMiddleware: authMiddleware,
	}
}

func (m *Module) GetHandlers() *Handlers {
	return m.handlers
}

func (m *Module) GetAuthMiddleware() *AuthMiddleware {
	return m.authMiddleware
}