package notification

import (
	"strconv"

	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type NotificationItem struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

type NotificationListResponse struct {
	Items    []NotificationItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

func (h *Handler) GetNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserID(c)

	notifications, total, err := h.service.GetNotifications(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	items := make([]NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, NotificationItem{
			ID:        n.ID,
			Title:     n.Title,
			Content:   n.Content,
			Type:      n.Type,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, NotificationListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID := getUserID(c)

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"count": count})
}

func (h *Handler) MarkAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"marked": true})
}

func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID := getUserID(c)

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"marked": true})
}

func (h *Handler) DeleteNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	if err := h.service.DeleteNotification(c.Request.Context(), uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

func getUserID(c *gin.Context) uint {
	if id, exists := c.Get("admin_id"); exists {
		return id.(uint)
	}
	if id, exists := c.Get("user_id"); exists {
		return id.(uint)
	}
	return 0
}