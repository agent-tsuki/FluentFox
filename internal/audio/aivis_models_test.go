package audio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestEnsureAivisModelsInstallsOnlyMissing(t *testing.T) {
	var mu sync.Mutex
	var installed []string
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/aivm_models":
			_, _ = w.Write([]byte(`{"have-1": {}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/aivm_models/install":
			mu.Lock()
			installed = append(installed, r.FormValue("url"))
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer engine.Close()

	if err := EnsureAivisModels(context.Background(), engine.URL, []string{"have-1", "new-2"}, zap.NewNop()); err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 || !strings.Contains(installed[0], "/aivm-models/new-2/download") {
		t.Fatalf("installed %v, want only new-2", installed)
	}
}
