package handler

import (
	"strconv"
	"time"

	"hjtpx/internal/models"
	"hjtpx/internal/repository"
	"hjtpx/internal/services"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	userRepo            *repository.UserRepository
	appRepo             *repository.AppRepository
	captchaRepo         *repository.CaptchaRepository
	verificationLogRepo *repository.VerificationLogRepository
	adminService       *services.AdminService
	jwtManager          *utils.JWTManager
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	captchaRepo *repository.CaptchaRepository,
	verificationLogRepo *repository.VerificationLogRepository,
	jwtManager *utils.JWTManager,
) *AdminHandler {
	return &AdminHandler{
		userRepo:            userRepo,
		appRepo:             nil,
		captchaRepo:         captchaRepo,
		verificationLogRepo: verificationLogRepo,
		jwtManager:          jwtManager,
	}
}

func NewAdminHandlerWithService(
	userRepo *repository.UserRepository,
	appRepo *repository.AppRepository,
	captchaRepo *repository.CaptchaRepository,
	verificationLogRepo *repository.VerificationLogRepository,
	adminService *services.AdminService,
	jwtManager *utils.JWTManager,
) *AdminHandler {
	return &AdminHandler{
		userRepo:            userRepo,
		appRepo:             appRepo,
		captchaRepo:         captchaRepo,
		verificationLogRepo: verificationLogRepo,
		adminService:        adminService,
		jwtManager:          jwtManager,
	}
}

type CreateAdminRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Username string `json:"username"`
}

func (h *AdminHandler) GetDashboard(c *gin.Context) {
	userCount, err := h.userRepo.Count()
	if err != nil {
		utils.InternalError(c, "Failed to fetch user count")
		return
	}

	captchaStats, err := h.captchaRepo.GetStats()
	if err != nil {
		utils.InternalError(c, "Failed to fetch captcha stats")
		return
	}

	captchaStats["total_users"] = userCount

	utils.Success(c, captchaStats)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := h.userRepo.FindAll(page, pageSize)
	if err != nil {
		utils.InternalError(c, "Failed to fetch users")
		return
	}

	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"app_id":     user.AppID,
			"status":     user.Status,
			"last_login": user.LastLogin,
			"created_at": user.CreatedAt,
		})
	}

	utils.Paginate(c, userList, page, pageSize, total)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userRepo.FindByID(uint(userID))
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	utils.Success(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"app_id":     user.AppID,
		"status":     user.Status,
		"last_login": user.LastLogin,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	var req models.UpdateUserRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	user, err := h.userRepo.FindByID(uint(userID))
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Status != 0 {
		user.Status = req.Status
	}

	if err := h.userRepo.Update(user); err != nil {
		utils.InternalError(c, "Failed to update user")
		return
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"status":   user.Status,
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.userRepo.Delete(uint(userID)); err != nil {
		utils.InternalError(c, "Failed to delete user")
		return
	}

	utils.Success(c, gin.H{
		"message": "User deleted successfully",
	})
}

func (h *AdminHandler) ListCaptchas(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.DefaultQuery("status", "")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	captchas, total, err := h.captchaRepo.FindAll(page, pageSize, status)
	if err != nil {
		utils.InternalError(c, "Failed to fetch captchas")
		return
	}

	var captchaList []gin.H
	for _, captcha := range captchas {
		captchaList = append(captchaList, gin.H{
			"id":           captcha.ID,
			"token":        captcha.Token,
			"type":         captcha.Type,
			"status":       captcha.Status,
			"expires_at":   captcha.ExpiresAt,
			"verify_count": captcha.VerifyCount,
			"max_verify":   captcha.MaxVerify,
			"ip_address":   captcha.IPAddress,
			"created_at":   captcha.CreatedAt,
		})
	}

	utils.Paginate(c, captchaList, page, pageSize, total)
}

func (h *AdminHandler) GetCaptchaStats(c *gin.Context) {
	stats, err := h.captchaRepo.GetStats()
	if err != nil {
		utils.InternalError(c, "Failed to fetch captcha stats")
		return
	}

	utils.Success(c, stats)
}

func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	existingUser, _ := h.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		utils.BadRequest(c, "Email already registered")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "Failed to hash password")
		return
	}

	user := &models.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		Username:  req.Username,
		AppID:     0,
		Status:    1,
	}

	if err := h.userRepo.Create(user); err != nil {
		utils.InternalError(c, "Failed to create admin")
		return
	}

	utils.Created(c, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	})
}

func (h *AdminHandler) AdminLogin(c *gin.Context) {
	var req LoginRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		utils.Unauthorized(c, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "Invalid credentials")
		return
	}

	user.LastLogin = time.Now()
	h.userRepo.Update(user)

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username, user.AppID, "admin")
	if err != nil {
		utils.InternalError(c, "Failed to generate token")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"created_at": user.CreatedAt,
		},
	})
}

func (h *AdminHandler) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var apps []models.App
	var total int64
	var err error

	if h.appRepo != nil {
		apps, total, err = h.appRepo.FindAll(page, pageSize)
	} else if h.adminService != nil {
		apps, total, err = h.adminService.ListApps(page, pageSize)
	} else {
		utils.InternalError(c, "App repository not initialized")
		return
	}

	if err != nil {
		utils.InternalError(c, "Failed to fetch apps")
		return
	}

	var appList []gin.H
	for _, app := range apps {
		appList = append(appList, gin.H{
			"id":         app.ID,
			"name":       app.Name,
			"app_key":    app.AppKey,
			"app_secret": app.AppSecret,
			"status":     app.Status,
			"domain":     app.Domain,
			"owner_id":   app.OwnerID,
			"created_at": app.CreatedAt,
		})
	}

	utils.Paginate(c, appList, page, pageSize, total)
}

func (h *AdminHandler) GetApp(c *gin.Context) {
	appIDStr := c.Param("id")
	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid app ID")
		return
	}

	var app *models.App
	if h.appRepo != nil {
		app, err = h.appRepo.FindByID(uint(appID))
	} else if h.adminService != nil {
		app, err = h.adminService.GetApp(uint(appID))
	} else {
		utils.InternalError(c, "App repository not initialized")
		return
	}

	if err != nil {
		utils.NotFound(c, "App not found")
		return
	}

	utils.Success(c, gin.H{
		"id":          app.ID,
		"name":        app.Name,
		"app_key":     app.AppKey,
		"app_secret":  app.AppSecret,
		"status":      app.Status,
		"domain":      app.Domain,
		"owner_id":    app.OwnerID,
		"created_at":  app.CreatedAt,
		"updated_at":  app.UpdatedAt,
	})
}

func (h *AdminHandler) CreateApp(c *gin.Context) {
	var req models.CreateAppRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	if h.appRepo != nil {
		existingApp, _ := h.appRepo.GetByAppKey(req.AppKey)
		if existingApp != nil {
			utils.BadRequest(c, "App key already exists")
			return
		}

		app := &models.App{
			Name:      req.Name,
			AppKey:    req.AppKey,
			AppSecret: req.AppSecret,
			Domain:    req.Domain,
			OwnerID:   req.OwnerID,
			Status:    1,
		}

		if err := h.appRepo.Create(app); err != nil {
			utils.InternalError(c, "Failed to create app")
			return
		}

		utils.Created(c, gin.H{
			"id":       app.ID,
			"name":     app.Name,
			"app_key":  app.AppKey,
			"status":   app.Status,
		})
	} else if h.adminService != nil {
		app, err := h.adminService.CreateApp(&req)
		if err != nil {
			utils.BadRequest(c, "Failed to create app: "+err.Error())
			return
		}

		utils.Created(c, gin.H{
			"id":      app.ID,
			"name":    app.Name,
			"app_key": app.AppKey,
			"status":  app.Status,
		})
	} else {
		utils.InternalError(c, "App repository not initialized")
	}
}

func (h *AdminHandler) UpdateApp(c *gin.Context) {
	appIDStr := c.Param("id")
	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid app ID")
		return
	}

	var req models.UpdateAppRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	if h.appRepo != nil {
		app, err := h.appRepo.FindByID(uint(appID))
		if err != nil {
			utils.NotFound(c, "App not found")
			return
		}

		if req.Name != "" {
			app.Name = req.Name
		}
		if req.AppSecret != "" {
			app.AppSecret = req.AppSecret
		}
		if req.Status != 0 {
			app.Status = req.Status
		}
		if req.Domain != "" {
			app.Domain = req.Domain
		}

		if err := h.appRepo.Update(app); err != nil {
			utils.InternalError(c, "Failed to update app")
			return
		}

		utils.Success(c, gin.H{
			"id":      app.ID,
			"name":    app.Name,
			"app_key": app.AppKey,
			"status":  app.Status,
		})
	} else if h.adminService != nil {
		app, err := h.adminService.UpdateApp(uint(appID), &req)
		if err != nil {
			utils.BadRequest(c, "Failed to update app: "+err.Error())
			return
		}

		utils.Success(c, gin.H{
			"id":      app.ID,
			"name":    app.Name,
			"app_key": app.AppKey,
			"status":  app.Status,
		})
	} else {
		utils.InternalError(c, "App repository not initialized")
	}
}

func (h *AdminHandler) DeleteApp(c *gin.Context) {
	appIDStr := c.Param("id")
	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid app ID")
		return
	}

	var errDel error
	if h.appRepo != nil {
		errDel = h.appRepo.Delete(uint(appID))
	} else if h.adminService != nil {
		errDel = h.adminService.DeleteApp(uint(appID))
	} else {
		utils.InternalError(c, "App repository not initialized")
		return
	}

	if errDel != nil {
		utils.InternalError(c, "Failed to delete app")
		return
	}

	utils.Success(c, gin.H{
		"message": "App deleted successfully",
	})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	if h.adminService != nil {
		user, err := h.adminService.CreateUser(&req)
		if err != nil {
			utils.BadRequest(c, "Failed to create user: "+err.Error())
			return
		}

		utils.Created(c, gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"status":   user.Status,
		})
	} else {
		existingUser, _ := h.userRepo.FindByEmail(req.Email)
		if existingUser != nil {
			utils.BadRequest(c, "Email already exists")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.InternalError(c, "Failed to hash password")
			return
		}

		user := &models.User{
			Email:    req.Email,
			Password: string(hashedPassword),
			Username: req.Username,
			AppID:    req.AppID,
			Status:   1,
		}

		if err := h.userRepo.Create(user); err != nil {
			utils.InternalError(c, "Failed to create user")
			return
		}

		utils.Created(c, gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
			"status":   user.Status,
		})
	}
}
