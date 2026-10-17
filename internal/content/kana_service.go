package content

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type KanaContentService struct {
	repo           KanaReader
	strokesBaseURL string
	logger         *zap.Logger
}

func NewKanaContentService(repo KanaReader, strokesBaseURL string, log *zap.Logger) *KanaContentService {
	return &KanaContentService{repo: repo, strokesBaseURL: strokesBaseURL, logger: log}
}

func (s *KanaContentService) GetCourse(ctx context.Context, rawScript string) (KanaCourseResponse, error) {
	script, err := parseKanaScript(rawScript)
	if err != nil {
		return KanaCourseResponse{}, err
	}
	units, err := s.repo.ListKanaUnits(ctx, script)
	if err != nil {
		return KanaCourseResponse{}, fmt.Errorf("get kana course: %w", err)
	}
	kana, err := s.repo.ListKana(ctx, script)
	if err != nil {
		return KanaCourseResponse{}, fmt.Errorf("get kana course: %w", err)
	}
	lookAlikes, err := s.repo.ListKanaLookAlikes(ctx, script)
	if err != nil {
		return KanaCourseResponse{}, fmt.Errorf("get kana course: %w", err)
	}

	unitSlugs := make(map[uuid.UUID]string, len(units))
	for i := range units {
		unitSlugs[units[i].ID] = units[i].Slug
	}
	return KanaCourseResponse{
		Script:         string(script),
		StrokesBaseURL: s.strokesBaseURL,
		Groups:         buildKanaGroups(script, units),
		Units:          buildKanaUnits(units, kana),
		Kana:           buildKanaChars(kana, unitSlugs, groupKanaLookAlikes(lookAlikes)),
	}, nil
}

func parseKanaScript(raw string) (common.KanaScript, error) {
	switch script := common.KanaScript(raw); script {
	case common.KanaScriptHiragana, common.KanaScriptKatakana:
		return script, nil
	default:
		return "", exceptions.ErrKanaScriptNotFound
	}
}

// Groups follow the order their first unit appears in the path.
func buildKanaGroups(script common.KanaScript, units []KanaUnit) []KanaGroupDTO {
	groups := []KanaGroupDTO{}
	index := map[common.KanaGroup]int{}
	for i := range units {
		u := &units[i]
		g, ok := index[u.GroupKey]
		if !ok {
			text, found := kanaGroupTexts[script][u.GroupKey]
			if !found {
				text = kanaGroupText{Title: string(u.GroupKey)}
			}
			g = len(groups)
			index[u.GroupKey] = g
			groups = append(groups, KanaGroupDTO{ID: string(u.GroupKey), Title: text.Title, Summary: text.Summary, UnitIDs: []string{}})
		}
		groups[g].UnitIDs = append(groups[g].UnitIDs, u.Slug)
	}
	return groups
}

func buildKanaUnits(units []KanaUnit, kana []Kana) []KanaUnitDTO {
	kanaIDs := make(map[uuid.UUID][]string, len(units))
	for i := range kana {
		kanaIDs[kana[i].UnitID] = append(kanaIDs[kana[i].UnitID], kana[i].Slug)
	}
	out := make([]KanaUnitDTO, 0, len(units))
	for i := range units {
		u := &units[i]
		ids := kanaIDs[u.ID]
		if ids == nil {
			ids = []string{}
		}
		out = append(out, KanaUnitDTO{
			ID: u.Slug, Title: u.Title, Summary: u.Summary, KanaIDs: ids,
			YoutubeID: u.YoutubeID, VideoCredit: u.VideoCredit,
		})
	}
	return out
}

func groupKanaLookAlikes(rows []KanaLookAlikeRow) map[uuid.UUID][]KanaLookAlikeDTO {
	out := make(map[uuid.UUID][]KanaLookAlikeDTO)
	for _, r := range rows {
		out[r.KanaID] = append(out[r.KanaID], KanaLookAlikeDTO{ID: r.OtherSlug, Char: r.OtherChar, Note: r.Note})
	}
	return out
}

func buildKanaChars(kana []Kana, unitSlugs map[uuid.UUID]string, lookAlikes map[uuid.UUID][]KanaLookAlikeDTO) []KanaChar {
	out := make([]KanaChar, 0, len(kana))
	for i := range kana {
		k := &kana[i]
		similar := lookAlikes[k.ID]
		if similar == nil {
			similar = []KanaLookAlikeDTO{}
		}
		out = append(out, KanaChar{
			ID:          k.Slug,
			Character:   k.Character,
			Romaji:      k.Romaji,
			Type:        kanaChartType(k.GroupKey),
			IsCompound:  utf8.RuneCountInString(k.Character) > 1,
			AltRomaji:   nonNil(k.AltRomaji),
			GroupID:     string(k.GroupKey),
			UnitID:      unitSlugs[k.UnitID],
			Position:    k.Position,
			StrokeCount: k.StrokeCount,
			LookAlikes:  similar,
		})
	}
	return out
}

func kanaChartType(group common.KanaGroup) string {
	if group == common.KanaGroupCombo {
		return common.KANA_CHART_TYPE_COMPOUND
	}
	return string(group)
}
