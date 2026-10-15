package handler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/httpserver"
	"github.com/fluentfox/api/pkg/telemetry"
)

var (
	setupOnce  sync.Once
	appHandler http.Handler
	setupErr   error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	setupOnce.Do(setup)
	if setupErr != nil {
		http.Error(w, "API initialization failed", http.StatusInternalServerError)
		return
	}

	restorePath(r)
	appHandler.ServeHTTP(w, r)
}

func setup() {
	defer func() {
		if recovered := recover(); recovered != nil {
			setupErr = fmt.Errorf("load configuration: %v", recovered)
		}
	}()

	cfg := config.Load()
	log, err := newLogger(cfg)
	if err != nil {
		setupErr = fmt.Errorf("initialise logger: %w", err)
		return
	}

	if _, _, err := telemetry.Setup("fluentfox-api"); err != nil {
		setupErr = fmt.Errorf("setup telemetry: %w", err)
		return
	}

	appHandler, _, setupErr = httpserver.NewHandler(cfg, log)
}

func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.IsDevelopment() {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func restorePath(r *http.Request) {
	query := r.URL.Query()
	path, rewritten := query["__fluentfox_path"]
	if !rewritten {
		return
	}

	value := ""
	if len(path) > 0 {
		value = path[0]
	}
	r.URL.Path = "/" + strings.TrimPrefix(value, "/")
	query.Del("__fluentfox_path")
	r.URL.RawQuery = query.Encode()
}
