package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RegisterGoogleOAuthRoutes uses Gin directly because the first two endpoints
// are browser navigations that must send HTTP redirects rather than JSON. The
// exchange endpoint remains JSON and is covered by the application's global
// CORS middleware.
func RegisterGoogleOAuthRoutes(router gin.IRouter, service *GoogleOAuthService, tokenMaker *token.Maker, refreshMaxAge int, log *zap.Logger) {
	router.GET("/auth/oauth/google/start", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errGoogleOAuthUnavailable, log)
			return
		}
		destination, err := service.Start(c.Request.Context(), c.Query("return_to"))
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		state := queryValue(destination, "state")
		if state == "" {
			writeGoogleOAuthError(c, errors.New("OAuth authorization URL omitted state"), log)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(oauthStateCookieName, state, int(googleOAuthStateTTL.Seconds()), "/auth/oauth/google", "", service.cookieSecure, true)
		c.Redirect(http.StatusFound, destination)
	})

	router.GET("/auth/oauth/google/callback", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errGoogleOAuthUnavailable, log)
			return
		}
		state := c.Query("state")
		cookieState, cookieErr := c.Cookie(oauthStateCookieName)
		// Delete the browser binding even when validation fails so an old value
		// cannot be replayed on a later callback.
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(oauthStateCookieName, "", -1, "/auth/oauth/google", "", service.cookieSecure, true)
		if cookieErr != nil || subtle.ConstantTimeCompare([]byte(cookieState), []byte(state)) != 1 {
			c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_failed"))
			return
		}
		// Google reports cancellations and consent failures in its query string.
		// Send a generic code to our frontend, never the provider's raw message.
		if c.Query("error") != "" {
			isLink, err := service.CancelCallback(c.Request.Context(), state)
			if err != nil {
				log.Warn("Google OAuth cancellation failed", zap.Error(err))
				c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_failed"))
				return
			}
			if isLink {
				c.Redirect(http.StatusFound, service.frontendGoogleLinkURL("oauth_cancelled"))
				return
			}
			c.Redirect(http.StatusFound, service.frontendErrorURL("oauth_cancelled"))
			return
		}
		result, err := service.CompleteCallback(c.Request.Context(), c.Query("state"), c.Query("code"))
		if err != nil {
			code := "oauth_failed"
			if appErr, ok := exceptions.As(err); ok {
				code = strings.ToLower(appErr.Code)
			}
			log.Warn("Google OAuth callback failed", zap.Error(err))
			if result.IsLink {
				c.Redirect(http.StatusFound, service.frontendGoogleLinkURL(code))
				return
			}
			c.Redirect(http.StatusFound, service.frontendErrorURL(code))
			return
		}
		c.Redirect(http.StatusFound, result.Destination)
	})

	router.POST("/auth/oauth/google/exchange", func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errGoogleOAuthUnavailable, log)
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
		c.SetSameSite(http.SameSiteLaxMode)
		cookie := refreshCookie(response.RefreshToken, refreshMaxAge)
		http.SetCookie(c.Writer, &cookie)
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": response})
	})

	linkAuth := []gin.HandlerFunc{middleware.RequireAuth(tokenMaker), middleware.RequireEmailVerified()}
	router.GET("/auth/oauth/google/link/status", append(linkAuth, func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errGoogleOAuthUnavailable, log)
			return
		}
		status, err := service.LinkStatus(c.Request.Context(), middleware.ContextUserID(c.Request.Context()))
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": status})
	})...)
	router.POST("/auth/oauth/google/link/start", append(linkAuth, func(c *gin.Context) {
		if service == nil {
			writeGoogleOAuthError(c, errGoogleOAuthUnavailable, log)
			return
		}
		destination, err := service.StartLink(c.Request.Context(), middleware.ContextUserID(c.Request.Context()))
		if err != nil {
			writeGoogleOAuthError(c, err, log)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"authorization_url": destination}})
	})...)
}

func queryValue(rawURL, key string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get(key)
}

func writeGoogleOAuthError(c *gin.Context, err error, log *zap.Logger) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "an unexpected error occurred"
	if appErr, ok := exceptions.As(err); ok {
		status = appErr.Status
		code = appErr.Code
		message = appErr.Message
	} else {
		middleware.LoggerFromContext(c.Request.Context(), log).Error(humautil.InternalErrorLogMessage, zap.String("route", c.FullPath()), zap.Error(err))
	}
	// Keep the envelope consistent with Huma endpoints so the frontend has one
	// error contract even though GET callback routes are redirect-oriented.
	c.JSON(status, gin.H{
		"status": "failed",
		"error": gin.H{
			"log_id":     uuid.NewString(),
			"error_code": code,
			"message":    message,
		},
	})
}
