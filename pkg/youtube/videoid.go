package youtube

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var ErrInvalidVideoID = errors.New("youtube: not a video id or video URL")

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

var videoHosts = map[string]bool{
	"youtube.com":              true,
	"www.youtube.com":          true,
	"m.youtube.com":            true,
	"music.youtube.com":        true,
	"youtube-nocookie.com":     true,
	"www.youtube-nocookie.com": true,
}

const shortLinkHost = "youtu.be"

// Path prefixes that are followed directly by the video id.
var idPathPrefixes = []string{"/embed/", "/shorts/", "/live/", "/v/"}

// ParseVideoID accepts a bare id or a watch, youtu.be, embed, shorts or live
// URL, and returns the 11-character video id.
func ParseVideoID(input string) (string, error) {
	input = strings.TrimSpace(input)
	if videoIDPattern.MatchString(input) {
		return input, nil
	}
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return "", ErrInvalidVideoID
	}
	host := strings.ToLower(u.Hostname())
	var id string
	switch {
	case host == shortLinkHost:
		id = firstSegment(strings.TrimPrefix(u.Path, "/"))
	case videoHosts[host]:
		id = idFromYouTubePath(u)
	}
	if !videoIDPattern.MatchString(id) {
		return "", ErrInvalidVideoID
	}
	return id, nil
}

func idFromYouTubePath(u *url.URL) string {
	if u.Path == "/watch" {
		return u.Query().Get("v")
	}
	for _, prefix := range idPathPrefixes {
		if rest, ok := strings.CutPrefix(u.Path, prefix); ok {
			return firstSegment(rest)
		}
	}
	return ""
}

func firstSegment(path string) string {
	segment, _, _ := strings.Cut(path, "/")
	return segment
}
