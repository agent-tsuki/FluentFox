// Package useragent turns a User-Agent header into a browser, OS and device
// type for display. It is a best guess for a device list, not for security.
package useragent

import "strings"

const (
	unknown     = "Unknown"
	mobileToken = "Mobile"
)

const (
	chrome  = "Chrome"
	firefox = "Firefox"
	iPadOS  = "iPadOS"
	iOS     = "iOS"
	android = "Android"
	windows = "Windows"
	linux   = "Linux"
)

const (
	DeviceDesktop = "desktop"
	DeviceMobile  = "mobile"
	DeviceTablet  = "tablet"
	DeviceUnknown = "unknown"
)

type Info struct {
	Browser    string
	OS         string
	DeviceType string
}

type rule struct {
	token string
	name  string
}

// Order matters: Edge and Opera also send "Chrome", and Chrome sends "Safari".
var browserRules = []rule{
	{"Edg/", "Edge"},
	{"OPR/", "Opera"},
	{"SamsungBrowser/", "Samsung Internet"},
	{"CriOS/", chrome},
	{"FxiOS/", firefox},
	{"Firefox/", firefox},
	{"Chrome/", chrome},
	{"Safari/", "Safari"},
}

// iPad and iPhone come before "Mac OS X", which their user agents also contain.
var osRules = []rule{
	{"iPad", iPadOS},
	{"iPhone", iOS},
	{android, android},
	{windows, windows},
	{"CrOS", "ChromeOS"},
	{"Mac OS X", "macOS"},
	{linux, linux},
}

func Parse(ua string) Info {
	info := Info{
		Browser: match(ua, browserRules),
		OS:      match(ua, osRules),
	}
	info.DeviceType = deviceType(ua, info.OS)
	return info
}

func match(ua string, rules []rule) string {
	for _, r := range rules {
		if strings.Contains(ua, r.token) {
			return r.name
		}
	}
	return unknown
}

func deviceType(ua, os string) string {
	switch {
	case os == iPadOS, os == android && !strings.Contains(ua, mobileToken):
		return DeviceTablet
	case os == iOS, os == android, strings.Contains(ua, mobileToken):
		return DeviceMobile
	case os == unknown:
		return DeviceUnknown
	default:
		return DeviceDesktop
	}
}
