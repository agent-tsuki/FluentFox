package learn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

type ChapterCompletionRepo interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	EnsureSettings(ctx context.Context, userID uuid.UUID) (LearningSettings, error)
	SetUnitCompleted(ctx context.Context, tx *gorm.DB, p *UnitProgress) error
	UpsertKnownCards(ctx context.Context, tx *gorm.DB, cards []SRSCard) error
}

var _ ChapterCompletionRepo = (*Repository)(nil)

type ChapterCompletionInput struct {
	Level   string `path:"level"   enum:"n5"     doc:"Lower-case JLPT level id"`
	Chapter string `path:"chapter" example:"ch5" doc:"Chapter slug"`
	Body    struct {
		Completed bool `json:"completed" doc:"true marks the chapter complete; false takes the mark away"`
	}
}

// The grammar track holds the chapter's completion; the vocab track's words
// join reviews alongside its sentences.
var chapterLevelTracks = map[string][]common.LearningTrack{
	"n5": {common.LearningTrackGrammarN5, common.LearningTrackVocabN5},
}

// ChapterCompletionService lets the learner mark a grammar chapter complete
// (or not). Marking it complete also puts the chapter's words and sentences
// into reviews, as taking its vocab quiz or exercise does.
type ChapterCompletionService struct {
	repo    ChapterCompletionRepo
	catalog *Catalog
	now     func() time.Time
	logger  *zap.Logger
}

func NewChapterCompletionService(repo ChapterCompletionRepo, catalog *Catalog, log *zap.Logger) *ChapterCompletionService {
	return &ChapterCompletionService{repo: repo, catalog: catalog, now: time.Now, logger: log}
}

func (s *ChapterCompletionService) Set(ctx context.Context, userID uuid.UUID, in *ChapterCompletionInput) (UnitProgressData, error) {
	tracks, ok := chapterLevelTracks[in.Level]
	if !ok {
		return UnitProgressData{}, exceptions.ErrGrammarChapterNotFound
	}
	pools := make([]*Pool, 0, len(tracks))
	for _, track := range tracks {
		pool, err := s.catalog.Load(ctx, track)
		if err != nil {
			return UnitProgressData{}, fmt.Errorf("set chapter completion: %w", err)
		}
		pools = append(pools, pool)
	}
	unit, ok := pools[0].UnitBySlug(in.Chapter)
	if !ok {
		return UnitProgressData{}, exceptions.ErrGrammarChapterNotFound
	}
	now := s.now().UTC()
	row := UnitProgress{UserID: userID, UnitType: unit.Type, UnitID: unit.ID}
	if in.Body.Completed {
		row.CompletedAt = &now
	}
	cards, err := s.chapterCards(ctx, userID, pools, unit.ID, in.Body.Completed, now)
	if err != nil {
		return UnitProgressData{}, err
	}
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.SetUnitCompleted(ctx, tx, &row); err != nil {
			return fmt.Errorf("mark chapter: %w", err)
		}
		return s.repo.UpsertKnownCards(ctx, tx, cards)
	})
	if err != nil {
		return UnitProgressData{}, fmt.Errorf("set chapter completion: %w", err)
	}
	return toUnitProgressData(&trackUnit{Track: tracks[0], Unit: unit}, &row), nil
}

// chapterCards are the chapter's cards as just learned (first answer rated
// Good), due again in minutes. Cards already being reviewed keep their
// schedule (UpsertKnownCards); taking the mark away removes nothing.
func (s *ChapterCompletionService) chapterCards(
	ctx context.Context, userID uuid.UUID, pools []*Pool, chapterID uuid.UUID, completed bool, now time.Time,
) ([]SRSCard, error) {
	if !completed {
		return nil, nil
	}
	settings, err := s.repo.EnsureSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("chapter cards: %w", err)
	}
	sched, err := newScheduler(&settings)
	if err != nil {
		return nil, fmt.Errorf("chapter cards: %w", err)
	}
	var cards []SRSCard
	for _, pool := range pools {
		for _, item := range pool.chapterItems(chapterID) {
			for _, facet := range common.SRS_ITEM_FACETS[item.Type] {
				fresh := srs.NewCard(now)
				learned, _, err := sched.Next(&fresh, now, srs.RatingGood)
				if err != nil {
					return nil, fmt.Errorf("chapter cards: %w", err)
				}
				cards = append(cards, SRSCard{
					ID: cardID(userID, item.Type, item.ID, facet), UserID: userID, ItemType: item.Type, ItemID: item.ID, Facet: facet,
					State: learned.State, Due: learned.Due, Stability: learned.Stability, Difficulty: learned.Difficulty,
					Reps: learned.Reps, Lapses: learned.Lapses, LearningStep: learned.LearningStep,
					ScheduledDays: learned.ScheduledDays, LastReviewAt: learned.LastReview, IntroducedAt: now,
				})
			}
		}
	}
	return cards, nil
}
