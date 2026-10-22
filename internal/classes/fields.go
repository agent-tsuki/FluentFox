package classes

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// Change-log fields for events that are not a single column.
const (
	logFieldCreated      = "created"
	logFieldDeleted      = "deleted"
	logFieldVideosSynced = "videos_synced"
)

// Public value of an overridable field: the editor's choice wins over YouTube.
func effective(custom, youtube *string) *string {
	if custom != nil {
		return custom
	}
	return youtube
}

type fieldPair struct {
	YouTube *string
	Custom  *string
}

func (c *Channel) fieldPair(f common.ChannelField) (fieldPair, bool) {
	switch f {
	case common.ChannelFieldName:
		return fieldPair{c.NameYouTube, c.NameCustom}, true
	case common.ChannelFieldAvatarURL:
		return fieldPair{c.AvatarURLYouTube, c.AvatarURLCustom}, true
	case common.ChannelFieldBannerURL:
		return fieldPair{c.BannerURLYouTube, c.BannerURLCustom}, true
	case common.ChannelFieldDescription:
		return fieldPair{c.DescriptionYouTube, c.DescriptionCustom}, true
	case common.ChannelFieldIntroVideoID:
		return fieldPair{c.IntroVideoIDYouTube, c.IntroVideoIDCustom}, true
	}
	return fieldPair{}, false
}

func (c *Channel) Effective(f common.ChannelField) *string {
	p, _ := c.fieldPair(f)
	return effective(p.Custom, p.YouTube)
}

func (c *Channel) EffectiveName() string {
	if name := c.Effective(common.ChannelFieldName); name != nil {
		return *name
	}
	return c.Slug
}

// Column names follow the field: name -> name_custom / name_youtube.
func customColumn(f common.ChannelField) string  { return string(f) + "_custom" }
func youtubeColumn(f common.ChannelField) string { return string(f) + "_youtube" }

// changeSet collects the columns a write changes, so the same list drives the
// UPDATE and the change-log rows. Unchanged values are skipped.
type changeSet struct {
	columns map[string]any
	logs    []fieldChange
}

type fieldChange struct {
	Field    string
	OldValue *string
	NewValue *string
}

func newChangeSet() *changeSet {
	return &changeSet{columns: map[string]any{}}
}

func (s *changeSet) Empty() bool { return len(s.columns) == 0 }

func (s *changeSet) Text(column string, oldValue, newValue *string) {
	if equalPtr(oldValue, newValue) {
		return
	}
	s.columns[column] = newValue
	s.logs = append(s.logs, fieldChange{Field: column, OldValue: oldValue, NewValue: newValue})
}

func (s *changeSet) Str(column, oldValue, newValue string) {
	if oldValue == newValue {
		return
	}
	s.columns[column] = newValue
	s.logs = append(s.logs, fieldChange{Field: column, OldValue: &oldValue, NewValue: &newValue})
}

func (s *changeSet) Value(column string, oldValue, newValue any) {
	oldText, newText := jsonText(oldValue), jsonText(newValue)
	if equalPtr(oldText, newText) {
		return
	}
	s.columns[column] = newValue
	s.logs = append(s.logs, fieldChange{Field: column, OldValue: oldText, NewValue: newText})
}

// Note logs an event that changes no column of the entity itself.
func (s *changeSet) Note(field, value string) {
	s.logs = append(s.logs, fieldChange{Field: field, NewValue: &value})
}

func (s *changeSet) Rows(actorID uuid.UUID, entity common.ResourceType, entityID uuid.UUID, now time.Time) []ChangeLog {
	rows := make([]ChangeLog, 0, len(s.logs))
	for _, l := range s.logs {
		rows = append(rows, ChangeLog{
			ActorID: actorID, EntityType: string(entity), EntityID: entityID,
			Field: l.Field, OldValue: l.OldValue, NewValue: l.NewValue, CreatedAt: now,
		})
	}
	return rows
}

func equalPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Lists, numbers and booleans are logged as JSON text.
func jsonText(v any) *string {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil || string(raw) == "null" {
		return nil
	}
	s := string(raw)
	return &s
}

func listOrEmpty(l []string) []string {
	if l == nil {
		return []string{}
	}
	return slices.Clone(l)
}
