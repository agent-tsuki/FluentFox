package useragent

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want Info
	}{
		{
			name: "Chrome on Windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome", OS: "Windows", DeviceType: DeviceDesktop},
		},
		{
			name: "Edge is not mistaken for Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
			want: Info{Browser: "Edge", OS: "Windows", DeviceType: DeviceDesktop},
		},
		{
			name: "Safari on iPhone",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Safari", OS: "iOS", DeviceType: DeviceMobile},
		},
		{
			name: "Chrome on an Android tablet",
			ua:   "Mozilla/5.0 (Linux; Android 14; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome", OS: "Android", DeviceType: DeviceTablet},
		},
		{
			name: "Firefox on macOS",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.6; rv:131.0) Gecko/20100101 Firefox/131.0",
			want: Info{Browser: "Firefox", OS: "macOS", DeviceType: DeviceDesktop},
		},
		{
			name: "empty user agent is unknown",
			ua:   "",
			want: Info{Browser: unknown, OS: unknown, DeviceType: DeviceUnknown},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(tt.ua); got != tt.want {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
