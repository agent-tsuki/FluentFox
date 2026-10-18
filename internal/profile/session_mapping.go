package profile

import (
	"net/netip"
	"strings"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/useragent"
)

// The stored IPv4 prefix is a /24, so its last octet is always 0 and is shown
// as "x" rather than as a real-looking address.
const maskedOctet = "x"

func toSessionResponse(s *users.RefreshToken, currentID uuid.UUID) SessionResponse {
	var raw string
	if s.UserAgent != nil {
		raw = *s.UserAgent
	}
	ua := useragent.Parse(raw)
	return SessionResponse{
		ID:         s.ID,
		Browser:    ua.Browser,
		OS:         ua.OS,
		DeviceType: ua.DeviceType,
		IPApprox:   approximateIP(s.IPPrefix),
		Location: SessionLocationResponse{
			CountryCode: s.CountryCode,
			Region:      s.Region,
			City:        s.City,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
			Source:      s.LocationSource,
			AccuracyM:   s.LocationAccuracyM,
		},
		IsVPN:      s.IsVPN,
		IsCurrent:  currentID != uuid.Nil && s.ID == currentID,
		CreatedAt:  s.CreatedAt,
		LastUsedAt: s.LastUsedAt,
	}
}

// "203.0.113.0/24" becomes "203.0.113.x"; an IPv6 /48 becomes "2001:db8:abcd::/48".
func approximateIP(prefix *string) string {
	if prefix == nil {
		return ""
	}
	p, err := netip.ParsePrefix(*prefix)
	if err != nil {
		return ""
	}
	if !p.Addr().Is4() {
		return p.String()
	}
	addr := p.Addr().String()
	return addr[:strings.LastIndex(addr, ".")+1] + maskedOctet
}

// The device that logged in most recently, or nil when none is signed in.
func latestLogin(sessions []users.RefreshToken, currentID uuid.UUID) *SessionResponse {
	var latest *users.RefreshToken
	for i := range sessions {
		if latest == nil || sessions[i].CreatedAt.After(latest.CreatedAt) {
			latest = &sessions[i]
		}
	}
	if latest == nil {
		return nil
	}
	resp := toSessionResponse(latest, currentID)
	return &resp
}
