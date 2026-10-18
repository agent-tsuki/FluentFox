package ipintel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

const testKey = "secret-key-123"

func newTestVPNAPI(t *testing.T, status int, body string) *VPNAPI {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	q := NewVPNAPI(testKey, time.Second)
	q.client.Transport = rewriteTo(srv.URL)
	return q
}

// rewriteTo sends every request to the test server while keeping the path and
// query, so the real URL (and the key inside it) is still exercised.
type rewriteTo string

func (base rewriteTo) RoundTrip(r *http.Request) (*http.Response, error) {
	target := string(base) + r.URL.Path + "?" + r.URL.RawQuery
	clone, err := http.NewRequestWithContext(r.Context(), r.Method, target, http.NoBody) //nolint:gosec // local httptest server
	if err != nil {
		return nil, err
	}
	return http.DefaultTransport.RoundTrip(clone)
}

func TestVPNAPI_Lookup(t *testing.T) {
	ip := netip.MustParseAddr("203.0.113.7")
	tests := []struct {
		name    string
		status  int
		body    string
		want    Result
		wantErr string
	}{
		{
			name:   "maps location with string coordinates",
			status: http.StatusOK,
			body:   `{"ip":"203.0.113.7","security":{"vpn":false,"proxy":false,"tor":false,"relay":false},"location":{"city":"Shibuya","region":"Tokyo","country_code":"JP","latitude":"35.6640","longitude":"139.6982"}}`,
			want:   Result{CountryCode: "JP", Region: "Tokyo", City: "Shibuya", Latitude: 35.664, Longitude: 139.6982},
		},
		{
			name:   "VPN is flagged",
			status: http.StatusOK,
			body:   `{"security":{"vpn":true},"location":{"country_code":"NL","latitude":"","longitude":""}}`,
			want:   Result{CountryCode: "NL", IsVPN: true},
		},
		{
			name:   "Tor and proxies count as VPN",
			status: http.StatusOK,
			body:   `{"security":{"tor":true,"proxy":true},"location":{"country_code":"DE"}}`,
			want:   Result{CountryCode: "DE", IsVPN: true},
		},
		{
			name:   "iCloud Private Relay alone is not a VPN",
			status: http.StatusOK,
			body:   `{"security":{"relay":true},"location":{"country_code":"US","latitude":37.75,"longitude":-97.82}}`,
			want:   Result{CountryCode: "US", Latitude: 37.75, Longitude: -97.82},
		},
		{name: "bad key is an error", status: http.StatusUnauthorized, body: `{"message":"Invalid API key"}`, wantErr: "Invalid API key"},
		{name: "used-up quota is an error even with 200", status: http.StatusOK, body: `{"message":"You have exceeded your daily limit"}`, wantErr: "daily limit"},
		{name: "server error is an error", status: http.StatusBadGateway, body: `<html>bad gateway</html>`, wantErr: "status 502"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestVPNAPI(t, tt.status, tt.body).Lookup(context.Background(), ip)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVPNAPI_ErrorsNeverContainTheKey(t *testing.T) {
	q := NewVPNAPI(testKey, time.Millisecond)
	q.client.Transport = rewriteTo("http://127.0.0.1:1")

	_, err := q.Lookup(context.Background(), netip.MustParseAddr("203.0.113.7"))
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatalf("error leaks the API key: %v", err)
	}
}
