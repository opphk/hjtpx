package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
	Limit    int
	SortBy   string
	SortOrder string
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

func PaginationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize)))

		if page < 1 {
			page = 1
		}

		if pageSize < 1 {
			pageSize = DefaultPageSize
		} else if pageSize > MaxPageSize {
			pageSize = MaxPageSize
		}

		offset := (page - 1) * pageSize

		sortBy := c.DefaultQuery("sort_by", "created_at")
		sortOrder := c.DefaultQuery("sort_order", "desc")

		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc"
		}

		allowedSortFields := map[string]bool{
			"id":          true,
			"created_at":  true,
			"updated_at":  true,
			"name":        true,
			"status":      true,
			"email":       true,
			"username":    true,
			"last_login":  true,
		}

		if !allowedSortFields[sortBy] {
			sortBy = "created_at"
		}

		params := PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			Offset:    offset,
			Limit:     pageSize,
			SortBy:    sortBy,
			SortOrder: sortOrder,
		}

		c.Set("pagination", params)
		c.Next()
	}
}

func GetPaginationParams(c *gin.Context) PaginationParams {
	if params, exists := c.Get("pagination"); exists {
		return params.(PaginationParams)
	}
	return PaginationParams{
		Page:      1,
		PageSize:  DefaultPageSize,
		Offset:    0,
		Limit:     DefaultPageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

func SetPaginationHeaders(c *gin.Context, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	c.Header("X-Total-Pages", strconv.Itoa(totalPages))
	c.Header("X-Current-Page", strconv.Itoa(page))
	c.Header("X-Page-Size", strconv.Itoa(pageSize))
	c.Header("X-Has-Next", strconv.FormatBool(page < totalPages))
	c.Header("X-Has-Prev", strconv.FormatBool(page > 1))
}
