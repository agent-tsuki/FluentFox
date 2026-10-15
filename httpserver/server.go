// Package httpserver builds the HTTP handler used by both the local server and Vercel.
package httpserver

import (
	"fmt"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/auth"
	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/services"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/emailcheck"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/mailer"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
	"github.com/fluentfox/api/pkg/validator"
)

func NewHandler(cfg *config.Config, log *zap.Logger) (http.Handler, func(), error) {
	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get database connection: %w", err)
	}
	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			log.Error("close database connection", zap.Error(err))
		}
	}

	tokenMaker := token.NewMaker(
		cfg.JWTAccessSecret,
		cfg.JWTAccessExpiryMinutes, cfg.JWTRefreshExpiryDays,
	)
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.EmailFromAddress)
	mailService := services.NewMailService(smtpMailer, cfg.AppURL, log)
	userRepo := users.NewRepository(db)

	emailChecker, err := emailcheck.New(net.DefaultResolver, common.EMAIL_MX_LOOKUP_TIMEOUT, cfg.EmailMXCheckEnabled)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("load email domain lists: %w", err)
	}

	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger(log))
	r.Use(otelgin.Middleware("fluentfox-api"))

	humaConfig := huma.DefaultConfig("FluentFox API", "1.0.0")
	humaConfig.CreateHooks = nil
	api := humagin.New(r, humaConfig)
	humautil.InitHumaErrors()
	humautil.RegisterBearerScheme(api)
	api.UseMiddleware(humautil.AuthMiddleware(api, tokenMaker))

	staleReleaser := auth.NewStaleAccountReleaser(userRepo, log)
	loginService := auth.NewLogin(userRepo, tokenMaker, log)
	googleOAuth, err := auth.NewGoogleOAuthService(cfg, userRepo, loginService, log)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("configure Google OAuth: %w", err)
	}
	authHandler := auth.NewHandler(
		auth.NewAuthService(userRepo, mailService, emailChecker, staleReleaser, log),
		auth.NewTokenVerificationService(userRepo, log),
		auth.NewResendVerificationService(userRepo, mailService, log),
		loginService,
		auth.NewChangeEmailService(userRepo, mailService, emailChecker, staleReleaser, log),
		auth.NewPasswordResetService(userRepo, mailService, cfg.JWTAccessSecret, log),
		log, validator.New(),
	)
	auth.RegisterRoutes(api, authHandler)
	auth.RegisterGoogleOAuthRoutes(r, googleOAuth, tokenMaker, log)

	r.GET("/health", func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "env": cfg.AppEnv})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r, cleanup, nil
}
