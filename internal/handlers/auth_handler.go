package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/WenDoge/rest-api/internal/auth"
	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/models"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	users  UserRepository
	tokens TokenRepository
	cfg    *config.Config
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}
type RegisterRequest struct {
	Username    string `json:"username"    binding:"required,min=3,max=255"`
	Password    string `json:"password"    binding:"required,min=6,max=50"`
	Email       string `json:"email"       binding:"required,email,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Age         int    `json:"age"         binding:"gte=0,lte=150"`
}

type RegisterResponse struct {
	models.User
	AuthResponse
}

type AuthResponse struct {
	AccessToken     string    `json:"access_token"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
}

func NewAuthHandler(users UserRepository, tokens TokenRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		users:  users,
		tokens: tokens,
		cfg:    cfg,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var loginRequest LoginRequest

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.users.GetByEmail(c.Request.Context(), loginRequest.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invalid credentials"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	authTokens, err := h.generateAndSetTokens(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate tokens error"})
		return
	}

	c.JSON(http.StatusOK, authTokens)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error binding json"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password" + err.Error()})
		return
	}

	user := &models.User{
		Email:       request.Email,
		Password:    string(hashedPassword),
		Username:    request.Username,
		Description: request.Description,
		Age:         request.Age,
	}

	createdUser, err := h.users.Create(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}

		if errors.Is(err, repository.ErrEmailExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	authTokens, err := h.generateAndSetTokens(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate tokens error"})
		return
	}

	c.JSON(http.StatusCreated, &RegisterResponse{*createdUser, *authTokens})
}

func (h *AuthHandler) generateAndSetTokens(c *gin.Context, user *models.User) (*AuthResponse, error) {
	tokens, err := auth.GenerateTokens(h.cfg, user)
	if err != nil {
		return nil, err
	}

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashToken(tokens.RefreshToken),
		ExpiresAt: tokens.RefreshExpiresAt,
	}

	if err := h.tokens.Create(c.Request.Context(), refreshToken); err != nil {
		return nil, err
	}

	SetRefreshCookie(c, h.cfg, tokens)

	return &AuthResponse{AccessToken: tokens.AccessToken, AccessExpiresAt: tokens.AccessExpiresAt}, nil
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenString, err := c.Cookie("refresh_token")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh_token not found"})
		return
	}

	parsedToken, err := auth.ParseJWT(h.cfg, refreshTokenString)

	if err != nil || !parsedToken.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)

	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token payload"})
		return
	}

	if claims["type"] != "refresh" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
		return
	}

	userID, ok := claims["user_id"].(string)

	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token payload"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {

		if errors.Is(err, repository.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "invalid credentials"})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	tokens, err := auth.GenerateTokens(h.cfg, user)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error generating tokens"})
		return
	}

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashToken(tokens.RefreshToken),
		ExpiresAt: tokens.RefreshExpiresAt,
	}

	err = h.tokens.Rotate(c.Request.Context(), auth.HashToken(refreshTokenString), refreshToken)

	if errors.Is(err, repository.ErrTokenNotFound) {
		ClearRefreshCookie(c, h.cfg)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh token revoked"})
		return
	}

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error rotating token"})
		return
	}

	SetRefreshCookie(c, h.cfg, tokens)

	c.JSON(http.StatusOK, AuthResponse{AccessToken: tokens.AccessToken, AccessExpiresAt: tokens.AccessExpiresAt})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshTokenString, err := c.Cookie("refresh_token")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh_token not found"})
		return
	}

	err = h.tokens.Delete(c.Request.Context(), auth.HashToken(refreshTokenString))
	if err != nil && !errors.Is(err, repository.ErrTokenNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ClearRefreshCookie(c, h.cfg)

	c.Status(http.StatusNoContent)
}
