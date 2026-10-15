package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterXOAuthRoutes mounts X's OAuth 1.0a browser endpoints. OAuth 1.0a
// has no state parameter, so the signed request token is additionally bound to
// an HttpOnly, SameSite cookie before the user is redirected to X.
func RegisterXOAuthRoutes(router gin.IRouter, service *XOAuthService, tokenMaker *token.Maker, log *zap.Logger) {
	router.GET("/auth/oauth/x/start", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errXOAuthUnavailable, log)
			return
		}
		start, err := service.Start(c.Request.Context(), c.Query("return_to"))
		if err != nil {
			code := "oauth_x_failed"
			if appErr, ok := exceptions.As(err); ok {
				code = strings.ToLower(appErr.Code)
			}
			if !errors.Is(err, errXOAuthUnavailable) {
				log.Warn("X OAuth start failed", zap.Error(err))
			}
			c.Redirect(http.StatusFound, service.frontendErrorURL(code))
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(xOAuthTokenCookieName, start.RequestToken, int(xOAuthStateTTL.Seconds()), "/auth/oauth/x", "", service.cookieSecure, true)
		c.Redirect(http.StatusFound, start.AuthorizationURL)
	})

	router.GET("/auth/oauth/x/callback", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errXOAuthUnavailable, log)
			return
		}
		callbackToken := c.Query("oauth_token")
		if callbackToken == "" {
			callbackToken = c.Query("denied")
		}
		cookieToken, cookieErr := c.Cookie(xOAuthTokenCookieName)
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(xOAuthTokenCookieName, "", -1, "/auth/oauth/x", "", service.cookieSecure, true)
		if cookieErr != nil || callbackToken == "" || subtle.ConstantTimeCompare([]byte(cookieToken), []byte(callbackToken)) != 1 {
			c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_x_failed"))
			return
		}

		if c.Query("denied") != "" || c.Query("oauth_verifier") == "" {
			isLink, err := service.CancelCallback(c.Request.Context(), callbackToken)
			if err != nil {
				log.Warn("X OAuth cancellation failed", zap.Error(err))
				c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_x_failed"))
				return
			}
			if isLink {
				c.Redirect(http.StatusFound, service.frontendXLinkURL("oauth_cancelled"))
				return
			}
			c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_cancelled"))
			return
		}

		result, err := service.CompleteCallback(c.Request.Context(), callbackToken, c.Query("oauth_verifier"))
		if err != nil {
			code := "oauth_x_failed"
			if appErr, ok := exceptions.As(err); ok {
				code = strings.ToLower(appErr.Code)
			}
			log.Warn("X OAuth callback failed", zap.Error(err))
			if result.IsLink {
				c.Redirect(http.StatusFound, service.frontendXLinkURL(code))
				return
			}
			c.Redirect(http.StatusFound, service.frontendErrorURL(code))
			return
		}
		c.Redirect(http.StatusFound, result.Destination)
	})

	router.POST("/auth/oauth/x/exchange", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errXOAuthUnavailable, log)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		var input OAuthExchangeRequest
		if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Code) == "" {
			writeGoogleOAuthError(c, exceptions.ErrBadRequest, log)
			return
		}
		response, _, err := service.Exchange(c.Request.Context(), input.Code)
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": response})
	})

	linkAuth := []gin.HandlerFunc{middleware.RequireAuth(tokenMaker), middleware.RequireEmailVerified()}
	router.GET("/auth/oauth/x/link/status", append(linkAuth, func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errXOAuthUnavailable, log)
			return
		}
		status, err := service.LinkStatus(c.Request.Context(), middleware.ContextUserID(c.Request.Context()))
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": status})
	})...)
	router.POST("/auth/oauth/x/link/start", append(linkAuth, func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errXOAuthUnavailable, log)
			return
		}
		start, err := service.StartLink(c.Request.Context(), middleware.ContextUserID(c.Request.Context()))
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"authorization_url": start.AuthorizationURL}})
	})...)
}
