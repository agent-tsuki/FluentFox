package learn

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// PlannedQuestion is one question a test asked.
type PlannedQuestion struct {
	CardID       uuid.UUID           `json:"card_id"`
	QuestionType common.QuestionType `json:"question_type"`
}

// QuestionPlan is the questions a test asked, in order, stored with its
// session (study_session.question_plan).
type QuestionPlan []PlannedQuestion

func planOf(questions []Question) QuestionPlan {
	plan := make(QuestionPlan, 0, len(questions))
	for i := range questions {
		plan = append(plan, PlannedQuestion{CardID: questions[i].CardID, QuestionType: common.QuestionType(questions[i].QuestionType)})
	}
	return plan
}

// asksPerCard is how many times the plan asks about each card. Answers are
// matched by card only: the client may ask a card the other way round (the
// final test mixes in listening), but never more often than planned.
func (p QuestionPlan) asksPerCard() map[uuid.UUID]int {
	out := make(map[uuid.UUID]int, len(p))
	for _, q := range p {
		out[q.CardID]++
	}
	return out
}

func (p *QuestionPlan) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case nil:
		*p = nil
		return nil
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("scan question plan: unexpected type %T", src)
	}
	if err := json.Unmarshal(raw, p); err != nil {
		return fmt.Errorf("scan question plan: %w", err)
	}
	return nil
}

// Value stores a nil plan as NULL.
func (p QuestionPlan) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("encode question plan: %w", err)
	}
	return string(raw), nil
}
