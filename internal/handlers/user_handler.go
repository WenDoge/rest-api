package handlers

import (
	"errors"
	"net/http"

	"github.com/WenDoge/rest-api/internal/config"
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
	cfg   *config.Config
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

type UpdateRequest struct {
	Username    *string `json:"username"    binding:"omitnil,min=3,max=255"`
	Email       *string `json:"email"       binding:"omitnil,email,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Age         *int    `json:"age"         binding:"omitnil,gte=0,lte=150"`
}

func NewUserHandler(users UserRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{users: users, cfg: cfg}
}

// GetMe получение профиля текущего пользователя
//
//	@Summary	Мой профиль
//	@Tags		users
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Failure	401	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/users/my [get]
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

// List список пользователей с пагинацией и фильтром по username
//
//	@Summary	Список пользователей
//	@Tags		users
//	@Produce	json
//	@Security	BearerAuth
//	@Param		username	query		string	false	"Фильтр по username"
//	@Param		limit		query		int		false	"Размер страницы"	default(20)	minimum(1)	maximum(100)
//	@Param		offset		query		int		false	"Смещение"			default(0)	minimum(0)
//	@Success	200			{object}	handlers.ListUsersResponse
//	@Failure	400			{object}	map[string]string
//	@Failure	401			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/users [get]
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

// Update частичное обновление профиля текущего пользователя
//
//	@Summary		Обновление профиля
//	@Description	Частичное обновление, передаются только те поля, которые нужно изменить
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		handlers.UpdateRequest	true	"Изменяемые поля"
//	@Success		200		{object}	models.User
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/users/my [patch]
func (h *UserHandler) Update(c *gin.Context) {
	var request UpdateRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString(middleware.UserIDKey)

	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request"})
		return
	}

	user, err := h.users.Update(c.Request.Context(), userID, repository.UpdateUserParams{
		Username:    request.Username,
		Email:       request.Email,
		Description: request.Description,
		Age:         request.Age,
	})

	if errors.Is(err, repository.ErrUserNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if errors.Is(err, repository.ErrEmailExists) {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}

	if errors.Is(err, repository.ErrUserExists) {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Delete мягкое удаление текущего пользователя
//
//	@Summary		Удаление профиля
//	@Description	пользователь помечается удалённым, refresh-cookie очищается
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	models.User
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/users/my [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized request"})
		return
	}

	user, err := h.users.Delete(c.Request.Context(), userID)

	if errors.Is(err, repository.ErrUserNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ClearRefreshCookie(c, h.cfg)
	c.JSON(http.StatusOK, user)
}
