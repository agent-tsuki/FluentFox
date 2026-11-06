package content

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	contentfiles "github.com/fluentfox/api/content"
	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

// memStore holds the real embedded course content in memory: the sync
// services write into it and the content services read from it, so the
// benchmarks below measure everything a request costs except the database.
type memStore struct {
	ids map[string]uuid.UUID

	kanaUnits       []KanaUnit
	kana            []Kana
	kanaLookAlikes  []KanaLookAlike
	kanjiGroups     []KanjiGroup
	kanjiUnits      []KanjiUnit
	kanji           []Kanji
	kun             []KanjiKunReading
	vocab           []KanjiVocab
	parts           map[uuid.UUID]KanjiPart
	partLinks       []KanjiPartLink
	kanjiLookAlikes []KanjiLookAlike
}

func (m *memStore) id(key string) uuid.UUID {
	if id, ok := m.ids[key]; ok {
		return id
	}
	id := uuid.New()
	m.ids[key] = id
	return id
}

func (m *memStore) Transaction(_ context.Context, fn func(*gorm.DB) error) error { return fn(nil) }

func (m *memStore) UpsertKanaUnit(_ context.Context, _ *gorm.DB, u *KanaUnit) (uuid.UUID, error) {
	u.ID = m.id("kana_unit:" + string(u.Script) + u.Slug)
	m.kanaUnits = append(m.kanaUnits, *u)
	return u.ID, nil
}

func (m *memStore) UpsertKana(_ context.Context, _ *gorm.DB, k *Kana) (uuid.UUID, error) {
	k.ID = m.id("kana:" + string(k.Script) + k.Slug)
	m.kana = append(m.kana, *k)
	return k.ID, nil
}

func (m *memStore) UpsertKanaLookAlike(_ context.Context, _ *gorm.DB, la *KanaLookAlike) error {
	m.kanaLookAlikes = append(m.kanaLookAlikes, *la)
	return nil
}

func (m *memStore) UpsertKanjiGroup(_ context.Context, _ *gorm.DB, g *KanjiGroup) (uuid.UUID, error) {
	g.ID = m.id("kanji_group:" + string(g.Level) + g.Slug)
	m.kanjiGroups = append(m.kanjiGroups, *g)
	return g.ID, nil
}

func (m *memStore) UpsertKanjiUnit(_ context.Context, _ *gorm.DB, u *KanjiUnit) (uuid.UUID, error) {
	u.ID = m.id("kanji_unit:" + string(u.Level) + u.Slug)
	m.kanjiUnits = append(m.kanjiUnits, *u)
	return u.ID, nil
}

func (m *memStore) UpsertKanjiPart(_ context.Context, _ *gorm.DB, p *KanjiPart) (uuid.UUID, error) {
	p.ID = m.id("part:" + p.Char)
	m.parts[p.ID] = *p
	return p.ID, nil
}

func (m *memStore) UpsertKanji(_ context.Context, _ *gorm.DB, k *Kanji) (uuid.UUID, error) {
	k.ID = m.id("kanji:" + k.Char)
	m.kanji = append(m.kanji, *k)
	return k.ID, nil
}

func (m *memStore) UpsertKanjiKunReading(_ context.Context, _ *gorm.DB, r *KanjiKunReading) error {
	m.kun = append(m.kun, *r)
	return nil
}

func (m *memStore) UpsertKanjiVocab(_ context.Context, _ *gorm.DB, v *KanjiVocab) error {
	m.vocab = append(m.vocab, *v)
	return nil
}

func (m *memStore) UpsertKanjiPartLink(_ context.Context, _ *gorm.DB, l *KanjiPartLink) error {
	m.partLinks = append(m.partLinks, *l)
	return nil
}

func (m *memStore) FindKanjiIDsByChars(_ context.Context, _ *gorm.DB, chars []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	for _, c := range chars {
		if id, ok := m.ids["kanji:"+c]; ok {
			out[c] = id
		}
	}
	return out, nil
}

func (m *memStore) UpsertKanjiLookAlike(_ context.Context, _ *gorm.DB, la *KanjiLookAlike) error {
	m.kanjiLookAlikes = append(m.kanjiLookAlikes, *la)
	return nil
}

func (m *memStore) ListKanaUnits(_ context.Context, s common.KanaScript) ([]KanaUnit, error) {
	var out []KanaUnit
	for _, u := range m.kanaUnits {
		if u.Script == s {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *memStore) ListKana(_ context.Context, s common.KanaScript) ([]Kana, error) {
	var out []Kana
	for _, k := range m.kana {
		if k.Script == s {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *memStore) ListKanaLookAlikes(_ context.Context, s common.KanaScript) ([]KanaLookAlikeRow, error) {
	byID := map[uuid.UUID]Kana{}
	for _, k := range m.kana {
		byID[k.ID] = k
	}
	var out []KanaLookAlikeRow
	for _, la := range m.kanaLookAlikes {
		if other, ok := byID[la.OtherKanaID]; ok && byID[la.KanaID].Script == s {
			out = append(out, KanaLookAlikeRow{KanaID: la.KanaID, OtherSlug: other.Slug, OtherChar: other.Character, Note: la.Note})
		}
	}
	return out, nil
}

func (m *memStore) ListKanjiGroups(_ context.Context, l common.JLPTLevel) ([]KanjiGroup, error) {
	var out []KanjiGroup
	for _, g := range m.kanjiGroups {
		if g.Level == l {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *memStore) ListKanjiUnits(_ context.Context, l common.JLPTLevel) ([]KanjiUnit, error) {
	var out []KanjiUnit
	for _, u := range m.kanjiUnits {
		if u.Level == l {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *memStore) ListKanji(_ context.Context, l common.JLPTLevel) ([]Kanji, error) {
	var out []Kanji
	for _, k := range m.kanji {
		if k.Level == l {
			out = append(out, k)
		}
	}
	return out, nil
}

func (m *memStore) GetKanjiByChar(_ context.Context, char string) (Kanji, error) {
	for _, k := range m.kanji {
		if k.Char == char {
			return k, nil
		}
	}
	return Kanji{}, gorm.ErrRecordNotFound
}

func (m *memStore) GetKanjiUnit(_ context.Context, id uuid.UUID) (KanjiUnit, error) {
	for _, u := range m.kanjiUnits {
		if u.ID == id {
			return u, nil
		}
	}
	return KanjiUnit{}, gorm.ErrRecordNotFound
}

func idSet(ids []uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func (m *memStore) ListKanjiKunReadings(_ context.Context, ids []uuid.UUID) ([]KanjiKunReading, error) {
	want, out := idSet(ids), []KanjiKunReading{}
	for _, r := range m.kun {
		if want[r.KanjiID] {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memStore) ListKanjiParts(_ context.Context, ids []uuid.UUID) ([]KanjiPartRow, error) {
	want, out := idSet(ids), []KanjiPartRow{}
	for _, l := range m.partLinks {
		if want[l.KanjiID] {
			p := m.parts[l.PartID]
			out = append(out, KanjiPartRow{KanjiID: l.KanjiID, Char: p.Char, Name: p.Name})
		}
	}
	return out, nil
}

func (m *memStore) ListKanjiVocab(_ context.Context, ids []uuid.UUID) ([]KanjiVocab, error) {
	want, out := idSet(ids), []KanjiVocab{}
	for _, v := range m.vocab {
		if want[v.KanjiID] {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *memStore) ListKanjiLookAlikes(_ context.Context, ids []uuid.UUID) ([]KanjiLookAlikeRow, error) {
	want, out := idSet(ids), []KanjiLookAlikeRow{}
	chars := map[uuid.UUID]string{}
	for _, k := range m.kanji {
		chars[k.ID] = k.Char
	}
	for _, la := range m.kanjiLookAlikes {
		if want[la.KanjiID] {
			out = append(out, KanjiLookAlikeRow{KanjiID: la.KanjiID, OtherChar: chars[la.OtherKanjiID], Note: la.Note})
		}
	}
	return out, nil
}

func syncEmbedded[T any](tb testing.TB, dir string, sync func(context.Context, *T) error) {
	tb.Helper()
	paths, err := fs.Glob(contentfiles.FS, path.Join(dir, "*.json"))
	if err != nil {
		tb.Fatal(err)
	}
	for _, p := range paths {
		raw, err := fs.ReadFile(contentfiles.FS, p)
		if err != nil {
			tb.Fatal(err)
		}
		var file T
		if err := json.Unmarshal(raw, &file); err != nil {
			tb.Fatalf("%s: %v", p, err)
		}
		if err := sync(context.Background(), &file); err != nil {
			tb.Fatalf("%s: %v", p, err)
		}
	}
}

// newBenchRouter builds the production middleware chain (minus tracing export)
// around the content routes, serving the embedded content from memory.
func newBenchRouter(tb testing.TB) http.Handler {
	tb.Helper()
	gin.SetMode(gin.ReleaseMode)
	store := &memStore{ids: map[string]uuid.UUID{}, parts: map[uuid.UUID]KanjiPart{}}
	log := zap.NewNop()
	syncEmbedded(tb, "kana", NewKanaSyncService(store, log).Sync)
	syncEmbedded(tb, "kanji", NewKanjiSyncService(store, log).Sync)

	r := gin.New()
	r.Use(middleware.CORS([]string{"https://foxsensei.com"}))
	r.Use(middleware.Logger(log))
	r.Use(middleware.RequestInfoCapture(false))
	r.Use(middleware.Compress())
	r.Use(middleware.PublicCache(middleware.PublicCacheRule{Prefix: "/content/", CacheControl: common.CONTENT_HTTP_CACHE_CONTROL}))
	r.Use(middleware.ServerTiming())
	cfg := huma.DefaultConfig("bench", "1")
	cfg.CreateHooks = nil
	api := humagin.New(r, cfg)
	humautil.InitHumaErrors()
	RegisterRoutes(api, NewHandler(
		NewKanaContentService(store, "https://cdn.example.test", log),
		NewKanjiContentService(store, "https://cdn.example.test", log),
		log,
	))
	return r
}

var benchPaths = []string{
	"/content/kana/hiragana",
	"/content/kanji/n5/index",
	"/content/kanji/n5/detail",
	"/content/kanji/chars/" + url.PathEscape("日"),
}

// BenchmarkContentEndpoints is the whole request (routing, production
// middleware, service, JSON, gzip) with the database replaced by memory: the
// time the API adds on top of its queries. After the first request the
// response comes from the in-memory content cache, as it does in production.
// Run it with: go test ./internal/content -run '^$' -bench Content -benchmem.
func BenchmarkContentEndpoints(b *testing.B) {
	h := newBenchRouter(b)
	for _, p := range benchPaths {
		for _, gzip := range []bool{false, true} {
			name := p
			if gzip {
				name += "/gzip"
			}
			b.Run(name, func(b *testing.B) {
				var size int
				for b.Loop() {
					req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, p, nil)
					if gzip {
						req.Header.Set("Accept-Encoding", "gzip")
					}
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, req)
					if rec.Code != http.StatusOK {
						b.Fatalf("%s: status %d: %s", p, rec.Code, rec.Body.String())
					}
					size = rec.Body.Len()
				}
				b.ReportMetric(float64(size), "resp-bytes")
			})
		}
	}
}

func TestBenchRouterServesEmbeddedContent(t *testing.T) {
	h := newBenchRouter(t)
	for _, p := range benchPaths {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", p, rec.Code)
		}
		if rec.Body.Len() < 500 {
			t.Fatalf("%s: suspiciously small body (%d bytes): %s", p, rec.Body.Len(), fmt.Sprint(rec.Body.String()))
		}
	}
}
