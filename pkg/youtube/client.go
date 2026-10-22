// Package youtube reads public channel and playlist data from the YouTube
// Data API v3 with an API key.
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://www.googleapis.com/youtube/v3/"
	// The API's largest page size for playlistItems.list and ids per videos.list.
	maxPageSize = 50
	// Bounds the body we read; a full page of 50 items is well under 1 MB.
	maxResponseBytes = 4 << 20
	partParam        = "part"
)

var ErrNotFound = errors.New("youtube: not found")

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// timeout applies to each API call; a playlist sync makes several.
func NewClient(apiKey string, timeout time.Duration) *Client {
	return &Client{apiKey: apiKey, baseURL: defaultBaseURL, client: &http.Client{Timeout: timeout}}
}

// ChannelLookup names a channel by id or by handle (with or without "@").
// The id wins when both are set.
type ChannelLookup struct {
	ID     string
	Handle string
}

type Channel struct {
	ID          string
	Title       string
	Description string
	// The channel's custom URL, such as "@tokiniandy".
	Handle    string
	AvatarURL string
	BannerURL string
	// The trailer shown to visitors who are not subscribed; empty when none is set.
	TrailerVideoID  string
	SubscriberCount *int64
	VideoCount      *int64
}

type PlaylistItem struct {
	VideoID      string
	Title        string
	ThumbnailURL string
	Position     int
}

type thumbnail struct {
	URL string `json:"url"`
}

type thumbnails struct {
	Default thumbnail `json:"default"`
	Medium  thumbnail `json:"medium"`
	High    thumbnail `json:"high"`
}

func (t thumbnails) best() string {
	for _, u := range []string{t.High.URL, t.Medium.URL, t.Default.URL} {
		if u != "" {
			return u
		}
	}
	return ""
}

type channelsResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title       string     `json:"title"`
			Description string     `json:"description"`
			CustomURL   string     `json:"customUrl"`
			Thumbnails  thumbnails `json:"thumbnails"`
		} `json:"snippet"`
		Statistics struct {
			SubscriberCount       string `json:"subscriberCount"`
			HiddenSubscriberCount bool   `json:"hiddenSubscriberCount"`
			VideoCount            string `json:"videoCount"`
		} `json:"statistics"`
		BrandingSettings struct {
			Channel struct {
				UnsubscribedTrailer string `json:"unsubscribedTrailer"`
			} `json:"channel"`
			Image struct {
				BannerExternalURL string `json:"bannerExternalUrl"`
			} `json:"image"`
		} `json:"brandingSettings"`
	} `json:"items"`
}

func (c *Client) Channel(ctx context.Context, lookup ChannelLookup) (Channel, error) {
	q := url.Values{partParam: {"snippet,brandingSettings,statistics"}}
	switch {
	case lookup.ID != "":
		q.Set("id", lookup.ID)
	case lookup.Handle != "":
		q.Set("forHandle", lookup.Handle)
	default:
		return Channel{}, errors.New("youtube: channel lookup needs an id or a handle")
	}
	var body channelsResponse
	if err := c.get(ctx, "channels", q, &body); err != nil {
		return Channel{}, err
	}
	if len(body.Items) == 0 {
		return Channel{}, ErrNotFound
	}
	item := body.Items[0]
	ch := Channel{
		ID:             item.ID,
		Title:          item.Snippet.Title,
		Description:    item.Snippet.Description,
		Handle:         item.Snippet.CustomURL,
		AvatarURL:      item.Snippet.Thumbnails.best(),
		BannerURL:      item.BrandingSettings.Image.BannerExternalURL,
		TrailerVideoID: item.BrandingSettings.Channel.UnsubscribedTrailer,
		VideoCount:     parseCount(item.Statistics.VideoCount),
	}
	if !item.Statistics.HiddenSubscriberCount {
		ch.SubscriberCount = parseCount(item.Statistics.SubscriberCount)
	}
	return ch, nil
}

type playlistItemsResponse struct {
	NextPageToken string `json:"nextPageToken"`
	Items         []struct {
		Snippet struct {
			Title      string     `json:"title"`
			Position   int        `json:"position"`
			Thumbnails thumbnails `json:"thumbnails"`
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}

// PlaylistItems reads at most maxPages pages of 50 items, in playlist order.
func (c *Client) PlaylistItems(ctx context.Context, playlistID string, maxPages int) ([]PlaylistItem, error) {
	q := url.Values{
		partParam:    {"snippet"},
		"playlistId": {playlistID},
		"maxResults": {strconv.Itoa(maxPageSize)},
	}
	var items []PlaylistItem
	for range maxPages {
		var body playlistItemsResponse
		if err := c.get(ctx, "playlistItems", q, &body); err != nil {
			return nil, err
		}
		for _, it := range body.Items {
			items = append(items, PlaylistItem{
				VideoID:      it.Snippet.ResourceID.VideoID,
				Title:        it.Snippet.Title,
				ThumbnailURL: it.Snippet.Thumbnails.best(),
				Position:     it.Snippet.Position,
			})
		}
		if body.NextPageToken == "" {
			break
		}
		q.Set("pageToken", body.NextPageToken)
	}
	return items, nil
}

type videosResponse struct {
	Items []struct {
		ID             string `json:"id"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

// VideoDurations returns the length of each video it can see. Private and
// deleted videos are missing from the result.
func (c *Client) VideoDurations(ctx context.Context, videoIDs []string) (map[string]time.Duration, error) {
	out := make(map[string]time.Duration, len(videoIDs))
	for start := 0; start < len(videoIDs); start += maxPageSize {
		batch := videoIDs[start:min(start+maxPageSize, len(videoIDs))]
		q := url.Values{partParam: {"contentDetails"}, "id": {strings.Join(batch, ",")}}
		var body videosResponse
		if err := c.get(ctx, "videos", q, &body); err != nil {
			return nil, err
		}
		for _, v := range body.Items {
			d, err := ParseDuration(v.ContentDetails.Duration)
			if err != nil {
				return nil, fmt.Errorf("youtube: video %s: %w", v.ID, err)
			}
			out[v.ID] = d
		}
	}
	return out, nil
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) get(ctx context.Context, resource string, q url.Values, out any) error {
	q.Set("key", c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+resource+"?"+q.Encode(), http.NoBody)
	if err != nil {
		return fmt.Errorf("youtube: build %s request: %w", resource, redact(err))
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("youtube: %s request: %w", resource, redact(err))
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, maxResponseBytes))
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		var e errorResponse
		_ = dec.Decode(&e) //nolint:errcheck // the status alone is enough when the body is not JSON
		return fmt.Errorf("youtube: %s status %d: %s", resource, resp.StatusCode, e.Error.Message)
	}
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("youtube: decode %s response: %w", resource, err)
	}
	return nil
}

func parseCount(raw string) *int64 {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// The API key is part of the request URL, and *url.Error prints that URL, so
// only the underlying cause is kept.
func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
