package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/httpserver"
	"github.com/fluentfox/api/pkg/telemetry"
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

	handler, cleanup, err := httpserver.NewHandler(cfg, log)
	if err != nil {
		log.Fatal("failed to build HTTP handler", zap.Error(err))
	}
	defer cleanup()

	// Server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
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
