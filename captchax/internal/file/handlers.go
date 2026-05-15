package file

import (
	"net/http"
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

type UploadResponse struct {
	ID           uint   `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
}

type FileListResponse struct {
	Files    []FileItem `json:"files"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type FileItem struct {
	ID           uint   `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	CreatedAt    string `json:"created_at"`
}

func (h *Handler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}

	userID := getUserID(c)

	file, err := h.service.Upload(c.Request.Context(), userID, fileHeader)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, UploadResponse{
		ID:           file.ID,
		Filename:     file.Filename,
		OriginalName: file.OriginalName,
		MimeType:     file.MimeType,
		Size:         file.Size,
		URL:          "/api/v1/files/" + strconv.FormatUint(uint64(file.ID), 10) + "/download",
	})
}

func (h *Handler) Download(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	file, err := h.service.Download(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+file.OriginalName+"\"")
	c.Header("Content-Type", file.MimeType)
	c.File(file.Path)
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := getUserID(c)

	files, total, err := h.service.List(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	items := make([]FileItem, 0, len(files))
	for _, f := range files {
		items = append(items, FileItem{
			ID:           f.ID,
			Filename:     f.Filename,
			OriginalName: f.OriginalName,
			MimeType:     f.MimeType,
			Size:         f.Size,
			CreatedAt:    f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, FileListResponse{
		Files:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *Handler) ServeUploadedFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	uploadDir := h.service.GetUploadDir()
	c.File(uploadDir + "/" + filename)
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