// Package main — cmd/api/main.go.
// The single entry point for the HTTP API server.
// Wires all dependencies together and starts the server.
// This is the only place in the codebase that may use init-like setup sequences.
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
	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/admin"
	"github.com/fluentfox/api/internal/auth"
	"github.com/fluentfox/api/internal/chapter"
	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/internal/progress"
	"github.com/fluentfox/api/internal/quiz"
	"github.com/fluentfox/api/internal/shop"
	"github.com/fluentfox/api/internal/srs"
	"github.com/fluentfox/api/internal/streak"
	"github.com/fluentfox/api/internal/user"
	"github.com/fluentfox/api/internal/xp"
	"github.com/fluentfox/api/pkg/cache"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/mailer"
	"github.com/fluentfox/api/pkg/storage"
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

	// ── Cache ─────────────────────────────────────────────────────────────────
	contentCache := cache.NewMemoryStore(pool)
	if err := contentCache.WarmUp(context.Background()); err != nil {
		log.Fatal("cache warm up failed", zap.Error(err))
	}

	// ── Infrastructure ────────────────────────────────────────────────────────
	tokenMaker := token.NewMaker(
		cfg.JWTAccessSecret, cfg.JWTRefreshSecret,
		cfg.JWTAccessExpiryMinutes, cfg.JWTRefreshExpiryDays,
	)
	v := validator.New()
	emailer := mailer.NewResendMailer(cfg.ResendAPIKey, cfg.EmailFromAddress)
	store := storage.NewR2Store(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2BucketName, cfg.R2PublicURL)
	_ = store // used by future upload handlers

	// ── Repositories ──────────────────────────────────────────────────────────
	authRepo := auth.NewRepository(pool)
	userRepo := user.NewRepository(pool)
	chapterRepo := chapter.NewRepository(pool)
	srsRepo := srs.NewRepository(pool)
	quizRepo := quiz.NewRepository(pool)
	progressRepo := progress.NewRepository(pool)
	streakRepo := streak.NewRepository(pool)
	xpRepo := xp.NewRepository(pool)
	shopRepo := shop.NewRepository(pool)
	adminRepo := admin.NewRepository(pool)

	// ── Services ───────────────────────────────────────────────────────────────
	xpSvc := xp.NewService(xpRepo)
	authSvc := auth.NewService(authRepo, userRepo, tokenMaker, emailer, cfg.AppURL, cfg.JWTRefreshExpiryDays)
	userSvc := user.NewService(userRepo)
	chapterSvc := chapter.NewService(chapterRepo)
	srsSvc := srs.NewService(srsRepo)
	quizSvc := quiz.NewService(quizRepo)
	progressSvc := progress.NewService(progressRepo)
	streakSvc := streak.NewService(streakRepo)
	shopSvc := shop.NewService(shopRepo, xpSvc)
	adminSvc := admin.NewService(adminRepo)

	// ── Handlers ───────────────────────────────────────────────────────────────
	authH := auth.NewHandler(authSvc, v, log)
	userH := user.NewHandler(userSvc, v, log)
	chapterH := chapter.NewHandler(chapterSvc, log)
	srsH := srs.NewHandler(srsSvc, v, log)
	quizH := quiz.NewHandler(quizSvc, v, log)
	progressH := progress.NewHandler(progressSvc, log)
	streakH := streak.NewHandler(streakSvc, log)
	xpH := xp.NewHandler(xpSvc, log)
	shopH := shop.NewHandler(shopSvc, v, log)
	adminH := admin.NewHandler(adminSvc, v, log)

	// ── Rate limiter ───────────────────────────────────────────────────────────
	rl := middleware.NewRateLimiter(10, time.Minute)

	// ── Router ─────────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger(log))

	// Health check — used by Railway to confirm deployment.
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

	// Auth routes — rate-limited.
	r.Route("/auth", func(r chi.Router) {
		r.With(rl.Limit).Post("/register", authH.Register)
		r.With(rl.Limit).Post("/login", authH.Login)
		r.With(rl.Limit).Post("/forgot-password", authH.ForgotPassword)
		r.Post("/refresh", authH.Refresh)
		r.Post("/logout", authH.Logout)
		r.Post("/reset-password", authH.ResetPassword)
		r.Post("/verify-email", authH.VerifyEmail)
	})

	// Authenticated routes.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(tokenMaker))

		r.Route("/users", func(r chi.Router) {
			r.Get("/me", userH.GetMe)
			r.Get("/me/profile", userH.GetProfile)
			r.Patch("/me/profile", userH.UpdateProfile)
			r.Get("/me/settings", userH.GetSettings)
			r.Patch("/me/settings", userH.UpdateSettings)
			r.Post("/me/change-password", userH.ChangePassword)
		})

		r.Route("/chapters", func(r chi.Router) {
			r.Get("/", chapterH.List)
			r.Get("/{slug}", chapterH.GetDetail)
		})

		r.Route("/srs", func(r chi.Router) {
			r.Get("/due", srsH.GetDueCards)
			r.Get("/due/count", srsH.GetDueCount)
			r.Post("/review", srsH.SubmitReview)
		})

		r.Route("/quiz", func(r chi.Router) {
			r.Post("/sessions", quizH.StartSession)
			r.Post("/sessions/{id}/answers", quizH.SubmitAnswer)
			r.Post("/sessions/{id}/finish", quizH.FinishSession)
		})

		r.Get("/progress", progressH.GetOverall)

		r.Get("/streak", streakH.GetStreak)

		r.Route("/xp", func(r chi.Router) {
			r.Get("/", xpH.GetXP)
			r.Get("/leaderboard", xpH.GetLeaderboard)
		})

		r.Route("/shop", func(r chi.Router) {
			r.Get("/", shopH.ListItems)
			r.Post("/purchase", shopH.Purchase)
			r.Get("/inventory", shopH.GetInventory)
		})

		// Admin routes — require admin role.
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "superadmin"))
			r.Get("/stats", adminH.GetStats)
			r.Get("/users", adminH.ListUsers)
			r.Post("/users/{id}/ban", adminH.BanUser)
		})
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

	// suppress unused import warnings during scaffolding
	_ = contentCache
	_ = emailer
}
