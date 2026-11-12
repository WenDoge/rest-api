package handlers

import (
	"net/http"

	"github.com/WenDoge/rest-api/internal/auth"
	"github.com/WenDoge/rest-api/internal/config"
	"github.com/gin-gonic/gin"
)

const refreshCookieName = "refresh_token"
const refreshCookiePath = APIPrefix + "/auth"

func SetRefreshCookie(c *gin.Context, cfg *config.Config, tokens *auth.TokenPair) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, tokens.RefreshToken, int(auth.RefreshTTL.Seconds()), refreshCookiePath, "", cfg.CookieSecure, true)
}

func ClearRefreshCookie(c *gin.Context, cfg *config.Config) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", cfg.CookieSecure, true)
}
