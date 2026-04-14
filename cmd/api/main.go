package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/auth"
	"github.com/fluentfox/api/internal/services"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/mailer"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/telemetry"
	"github.com/fluentfox/api/pkg/token"
	"github.com/fluentfox/api/pkg/validator"
)

func main() {
	cfg := config.Load()

	// Logger
	var log *zap.Logger
	var err error
	if cfg.IsDevelopment() {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// Telemetry
	_, cleanupTelemetry, err := telemetry.Setup("fluentfox-api")
	if err != nil {
		log.Fatal("failed to setup telemetry", zap.Error(err))
	}
	defer cleanupTelemetry()

	// Database
	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// Infrastructure
	tokenMaker := token.NewMaker(
		cfg.JWTAccessSecret, cfg.JWTRefreshSecret,
		cfg.JWTAccessExpiryMinutes, cfg.JWTRefreshExpiryDays,
	)

	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.EmailFromAddress)
	mailService := services.NewMailService(smtpMailer, cfg.AppURL, log)

	v := validator.New()
	userRepo := users.NewRepository(db)

	// Router
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger(log))
	r.Use(otelgin.Middleware("fluentfox-api"))

	// Huma wraps the Gin router to provide automatic OpenAPI 3.1 spec + docs UI.
	// Plain Gin routes (health, metrics) registered on r still work alongside it.
	// Build config without the SchemaLinkTransformer so responses never include
	// a `$schema` field.
	humaConfig := huma.DefaultConfig("FluentFox API", "1.0.0")
	humaConfig.CreateHooks = nil // removes the $schema link transformer
	api := humagin.New(r, humaConfig)
	humautil.InitHumaErrors()

	//sync logs
	log, err = zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer log.Sync()
		// Domain routes — add new domains here
	authHandler := auth.NewHandler(
		auth.NewAuthService(userRepo, mailService, log),
		auth.NewTokenVerificationService(userRepo, log),
		auth.NewResendVerificationService(userRepo, mailService, log),
		auth.NewLogin(userRepo, tokenMaker, log),
		log, v,
	)
	auth.RegisterRoutes(api, authHandler)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "env": cfg.AppEnv})
	})

	// Prometheus metrics — scraped by Prometheus / Grafana
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("server starting", zap.String("addr", srv.Addr), zap.String("env", cfg.AppEnv))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-done
	log.Info("server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
}
