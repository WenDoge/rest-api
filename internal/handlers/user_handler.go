package handlers

import (
	"errors"
	"net/http"

	"github.com/WenDoge/rest-api/internal/middleware"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit  = 20
	DefaultOffset = 0
)

type UserHandler struct {
	users UserRepository
}

type ListUsersQuery struct {
	Username *string `form:"username"`
	Limit    *int    `form:"limit"    binding:"omitempty,min=1,max=100"`
	Offset   *int    `form:"offset"   binding:"omitempty,min=0"`
}

type ListUsersResponse struct {
	Items  []models.User `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

func NewUserHandler(users UserRepository) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {

		if errors.Is(err, repository.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "invalid credentials"})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error to get user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) List(c *gin.Context) {
	var query ListUsersQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := DefaultLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	offset := DefaultOffset
	if query.Offset != nil {
		offset = *query.Offset
	}

	opts := repository.ListUsersOptions{
		Filter:     repository.ListUsersFilter{Username: query.Username},
		Navigation: repository.ListUsersNavigation{Limit: limit, Offset: offset},
	}

	users, total, err := h.users.List(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Items:  *users,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}
