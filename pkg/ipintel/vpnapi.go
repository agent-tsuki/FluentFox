// Package ipintel looks up where an IP address is and whether it is a VPN,
// using the vpnapi.io API.
package ipintel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"
)

const (
	vpnapiBaseURL = "https://vpnapi.io/api/"
	// Bounds the body we read; real responses are under 2 KB.
	maxResponseBytes = 64 << 10
)

// Result is the location of an address. Coordinates are city-level for an IP.
type Result struct {
	CountryCode string
	Region      string
	City        string
	Latitude    float64
	Longitude   float64
	IsVPN       bool
}

type VPNAPI struct {
	apiKey string
	client *http.Client
}

func NewVPNAPI(apiKey string, timeout time.Duration) *VPNAPI {
	return &VPNAPI{apiKey: apiKey, client: &http.Client{Timeout: timeout}}
}

type vpnapiResponse struct {
	Message  string `json:"message"`
	Security struct {
		VPN   bool `json:"vpn"`
		Proxy bool `json:"proxy"`
		Tor   bool `json:"tor"`
	} `json:"security"`
	Location struct {
		City        string     `json:"city"`
		Region      string     `json:"region"`
		CountryCode string     `json:"country_code"`
		Latitude    coordinate `json:"latitude"`
		Longitude   coordinate `json:"longitude"`
	} `json:"location"`
}

// coordinate accepts "37.7510" (what vpnapi.io sends) as well as 37.751.
type coordinate float64

func (c *coordinate) UnmarshalJSON(raw []byte) error {
	text := string(bytes.Trim(raw, `"`))
	if text == "" || text == "null" {
		*c = 0
		return nil
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return fmt.Errorf("coordinate %q: %w", text, err)
	}
	*c = coordinate(v)
	return nil
}

func (q *VPNAPI) Lookup(ctx context.Context, ip netip.Addr) (Result, error) {
	endpoint := vpnapiBaseURL + url.PathEscape(ip.String()) + "?key=" + url.QueryEscape(q.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return Result{}, fmt.Errorf("vpnapi: build request: %w", redact(err))
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("vpnapi: request: %w", redact(err))
	}
	defer resp.Body.Close()

	var body vpnapiResponse
	decodeErr := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, maxResponseBytes)).Decode(&body)
	// Errors such as a bad key or a used-up daily quota come back as {"message": ...}.
	if resp.StatusCode != http.StatusOK || body.Message != "" {
		return Result{}, fmt.Errorf("vpnapi: status %d: %s", resp.StatusCode, body.Message)
	}
	if decodeErr != nil {
		return Result{}, fmt.Errorf("vpnapi: decode response: %w", decodeErr)
	}

	return Result{
		CountryCode: body.Location.CountryCode,
		Region:      body.Location.Region,
		City:        body.Location.City,
		Latitude:    float64(body.Location.Latitude),
		Longitude:   float64(body.Location.Longitude),
		// iCloud Private Relay ("relay") is left out on purpose: it is on by
		// default for many iPhone users, who would otherwise need an email code
		// at every login.
		IsVPN: body.Security.VPN || body.Security.Proxy || body.Security.Tor,
	}, nil
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
