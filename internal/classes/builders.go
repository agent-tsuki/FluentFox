package classes

import (
	"cmp"
	"slices"

	"github.com/fluentfox/api/internal/common"
)

func toCourseCard(l *CourseListing) CourseCard {
	c := l.Course
	return CourseCard{
		Slug: c.Slug, Title: c.Title, Summary: c.Summary,
		Level: string(c.Level), Platform: string(c.Platform),
		IsFree: c.IsFree, IsFeatured: c.IsFeatured,
		ThumbnailURL:     c.ThumbnailURL,
		VideoCount:       l.VideoCount,
		TotalDurationSec: c.TotalDurationSec,
		Tags:             listOrEmpty(c.Tags),
		Channel: ChannelCard{
			Slug:            l.Channel.Slug,
			Name:            l.Channel.EffectiveName(),
			AvatarURL:       l.Channel.Effective(common.ChannelFieldAvatarURL),
			AccentColor:     l.Channel.AccentColor,
			SubscriberCount: l.Channel.SubscriberCount,
		},
	}
}

// Featured first, then the editors' sort order, then title.
func sortListings(listings []CourseListing) {
	slices.SortStableFunc(listings, func(a, b CourseListing) int {
		if a.Course.IsFeatured != b.Course.IsFeatured {
			if a.Course.IsFeatured {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(a.Course.SortOrder, b.Course.SortOrder), cmp.Compare(a.Course.Title, b.Course.Title))
	})
}

func toCards(listings []CourseListing) []CourseCard {
	cards := make([]CourseCard, 0, len(listings))
	for i := range listings {
		cards = append(cards, toCourseCard(&listings[i]))
	}
	return cards
}

func toChannelProfile(ch *Channel) ChannelProfile {
	return ChannelProfile{
		Slug: ch.Slug, Name: ch.EffectiveName(), Platform: string(ch.Platform), URL: ch.URL,
		Handle: ch.Handle, YouTubeChannelID: ch.YouTubeChannelID,
		AvatarURL:       ch.Effective(common.ChannelFieldAvatarURL),
		BannerURL:       ch.Effective(common.ChannelFieldBannerURL),
		Description:     ch.Effective(common.ChannelFieldDescription),
		IntroVideoID:    ch.Effective(common.ChannelFieldIntroVideoID),
		SubscriberCount: ch.SubscriberCount, VideoCount: ch.VideoCount, AccentColor: ch.AccentColor,
		TeachingStyle: ch.TeachingStyle,
		BestFor:       listOrEmpty(ch.BestFor), NotFor: listOrEmpty(ch.NotFor),
		Pros: listOrEmpty(ch.Pros), Cons: listOrEmpty(ch.Cons),
		Verdict: ch.Verdict,
	}
}

func toAdminChannel(ch *Channel) AdminChannel {
	field := func(f common.ChannelField) OverridableField {
		p, _ := ch.fieldPair(f)
		return OverridableField{YouTube: p.YouTube, Custom: p.Custom, Effective: effective(p.Custom, p.YouTube)}
	}
	return AdminChannel{
		ChannelProfile: toChannelProfile(ch),
		ID:             ch.ID, IsPublished: ch.IsPublished, LastSyncedAt: ch.LastSyncedAt,
		Fields: AdminChannelFields{
			Name:         field(common.ChannelFieldName),
			AvatarURL:    field(common.ChannelFieldAvatarURL),
			BannerURL:    field(common.ChannelFieldBannerURL),
			Description:  field(common.ChannelFieldDescription),
			IntroVideoID: field(common.ChannelFieldIntroVideoID),
		},
	}
}

// related picks other visible courses: the same channel's, and the same
// level's from other channels.
func related(course *Course, visible []CourseListing) (moreFromChannel, similar []CourseCard) {
	sortListings(visible)
	moreFromChannel, similar = []CourseCard{}, []CourseCard{}
	for i := range visible {
		l := &visible[i]
		switch {
		case l.Course.ID == course.ID:
		case l.Course.ChannelID == course.ChannelID:
			if len(moreFromChannel) < common.CLASS_RELATED_LIMIT {
				moreFromChannel = append(moreFromChannel, toCourseCard(l))
			}
		case l.Course.Level == course.Level:
			if len(similar) < common.CLASS_RELATED_LIMIT {
				similar = append(similar, toCourseCard(l))
			}
		}
	}
	return moreFromChannel, similar
}

func toCourseDetail(course *Course, ch *Channel, videos []Video, visible []CourseListing) CourseDetail {
	more, similar := related(course, visible)
	out := CourseDetail{
		Slug: course.Slug, Title: course.Title, Summary: course.Summary,
		Level: string(course.Level), Platform: string(course.Platform), URL: course.URL,
		PlaylistID: course.PlaylistID, IsFree: course.IsFree, PriceNote: course.PriceNote,
		ThumbnailURL:     course.ThumbnailURL,
		Tags:             listOrEmpty(course.Tags),
		WhatYouLearn:     listOrEmpty(course.WhatYouLearn),
		Curriculum:       slices.Clone(course.Curriculum),
		TotalDurationSec: course.TotalDurationSec,
		LastSyncedAt:     course.LastSyncedAt,
		Videos:           make([]VideoResponse, 0, len(videos)),
		Channel:          toChannelProfile(ch),
		MoreFromChannel:  more,
		Similar:          similar,
	}
	if out.Curriculum == nil {
		out.Curriculum = []CurriculumSection{}
	}
	for _, v := range videos {
		out.Videos = append(out.Videos, VideoResponse{
			YouTubeVideoID: v.YouTubeVideoID, Title: v.Title, ThumbnailURL: v.ThumbnailURL,
			DurationSec: v.DurationSec, Position: v.Position,
		})
	}
	return out
}

func toAdminCourse(course *Course, ch *Channel, videos []Video, visible []CourseListing) AdminCourse {
	return AdminCourse{
		CourseDetail: toCourseDetail(course, ch, videos, visible),
		ID:           course.ID, ChannelID: course.ChannelID,
		IsPublished: course.IsPublished, IsFeatured: course.IsFeatured, SortOrder: course.SortOrder,
	}
}
