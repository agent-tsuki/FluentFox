// Package main — cmd/api/main.go.
// The single entry point for the HTTP API server.

// @title           FluentFox API
// @version         1.0
// @description     Japanese language learning API.

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer <your_access_token>"

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	_ "github.com/fluentfox/api/docs"
	"github.com/fluentfox/api/internal/auth"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
	"github.com/fluentfox/api/pkg/validator"
)

func main() {
	cfg := config.Load()

	// ── Logger ──────────────────────────────────────────────────────────────
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

	// ── Database ─────────────────────────────────────────────────────────────
	pool, err := database.NewPool(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// ── Infrastructure ────────────────────────────────────────────────────────
	tokenMaker := token.NewMaker(
		cfg.JWTAccessSecret, cfg.JWTRefreshSecret,
		cfg.JWTAccessExpiryMinutes, cfg.JWTRefreshExpiryDays,
	)
	_ = validator.New()
	_ = tokenMaker

	// ── Handlers ──────────────────────────────────────────────────────────────
	userRepo := users.UserRepository(pool)
	authService := auth.NewAuthService(userRepo)
	authHandler := auth.NewAuthHandler(authService, log)

	// ── Router ─────────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger(log))

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		if err := pool.Ping(req.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"error","error":"database unreachable"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","env":%q}`, cfg.AppEnv)
	})

	// ── Server ─────────────────────────────────────────────────────────────────
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
