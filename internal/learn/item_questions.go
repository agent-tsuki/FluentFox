package learn

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/srs"
)

// ItemSessionRepo is what practice and tests need: the user's cards and
// mix-ups for the track, and somewhere to store the session.
type ItemSessionRepo interface {
	LearnerRepo
	ListTrackCards(ctx context.Context, userID uuid.UUID, track common.LearningTrack) ([]SRSCard, error)
	ListConfusions(ctx context.Context, userID uuid.UUID) ([]UserConfusion, error)
	CreateSession(ctx context.Context, s *StudySession) error
}

var _ ItemSessionRepo = (*Repository)(nil)

type cardSlot struct {
	itemID uuid.UUID
	facet  common.CardFacet
}

// itemQuestions builds questions for items a practice or test picked. Each
// question uses the user's card for the facet or, when there is none, the
// derived id that the first answer creates (as lessons do).
type itemQuestions struct {
	learner  *learner
	pool     *Pool
	cards    []SRSCard
	bySlot   map[cardSlot]*SRSCard
	confused map[uuid.UUID][]uuid.UUID
	builder  *QuestionBuilder
	mode     common.AnswerMode
}

func loadItemQuestions(
	ctx context.Context, repo ItemSessionRepo, catalog *Catalog, builder *QuestionBuilder, l *learner, track common.LearningTrack,
) (*itemQuestions, error) {
	pool, err := catalog.Load(ctx, track)
	if err != nil {
		return nil, fmt.Errorf("load item questions: %w", err)
	}
	cards, err := repo.ListTrackCards(ctx, l.UserID, track)
	if err != nil {
		return nil, fmt.Errorf("load item questions: %w", err)
	}
	confusions, err := repo.ListConfusions(ctx, l.UserID)
	if err != nil {
		return nil, fmt.Errorf("load item questions: %w", err)
	}
	q := &itemQuestions{
		learner: l, pool: pool, cards: cards, bySlot: make(map[cardSlot]*SRSCard, len(cards)),
		confused: groupConfusions(confusions), builder: builder, mode: l.Settings.AnswerMode,
	}
	for i := range cards {
		q.bySlot[cardSlot{itemID: cards[i].ItemID, facet: cards[i].Facet}] = &cards[i]
	}
	return q, nil
}

func (q *itemQuestions) input(item *Item, facet common.CardFacet) *QuestionInput {
	card := QuestionCard{ID: cardID(q.learner.UserID, item.Type, item.ID, facet), State: srs.StateNew, Facet: facet}
	if c, ok := q.bySlot[cardSlot{itemID: item.ID, facet: facet}]; ok {
		card = QuestionCard{ID: c.ID, State: c.State, Stability: c.Stability, Facet: facet, IsLeech: c.IsLeech}
	}
	return &QuestionInput{Card: card, Item: item, Pool: q.pool, Confused: q.confused[item.ID], Mode: q.mode}
}

func (q *itemQuestions) build(item *Item, facet common.CardFacet) Question {
	return q.builder.Build(q.input(item, facet))
}

func (q *itemQuestions) buildAs(item *Item, facet common.CardFacet, qt common.QuestionType) Question {
	return q.builder.BuildAs(q.input(item, facet), qt)
}

// everyFacet asks each facet of each item, up to limit questions (0 means
// no limit).
func (q *itemQuestions) everyFacet(items []*Item, limit int) []Question {
	out := make([]Question, 0, len(items))
	for _, item := range items {
		for _, facet := range common.SRS_ITEM_FACETS[item.Type] {
			if limit > 0 && len(out) == limit {
				return out
			}
			out = append(out, q.build(item, facet))
		}
	}
	return out
}

// introduced lists the cards that have been scheduled at least once, in id
// order.
func (q *itemQuestions) introduced() []*SRSCard {
	var out []*SRSCard
	for i := range q.cards {
		if q.cards[i].State != srs.StateNew {
			out = append(out, &q.cards[i])
		}
	}
	return out
}

func (p *Pool) UnitBySlug(slug string) (Unit, bool) {
	for _, u := range p.Units {
		if u.Slug == slug {
			return u, true
		}
	}
	return Unit{}, false
}

// ItemsWhere lists the items, in path order, whose unit passes keep.
func (p *Pool) ItemsWhere(keep func(Unit) bool) []*Item {
	units := make(map[uuid.UUID]bool, len(p.Units))
	for _, u := range p.Units {
		units[u.ID] = keep(u)
	}
	var out []*Item
	for i := range p.Items {
		if units[p.Items[i].UnitID] {
			out = append(out, &p.Items[i])
		}
	}
	return out
}

func (p *Pool) ItemByChar(char string) (*Item, bool) {
	for i := range p.Items {
		if p.Items[i].Char == char {
			return &p.Items[i], true
		}
	}
	return nil, false
}
