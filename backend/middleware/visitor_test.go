package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestParseVisitorUserAgent(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		deviceType string
		model      string
		browser    string
		os         string
	}{
		{
			name:       "Android phone in WeChat",
			userAgent:  "Mozilla/5.0 (Linux; Android 14; SM-S918B Build/UP1A.231005.007) AppleWebKit/537.36 Version/4.0 Chrome/120.0.0.0 Mobile Safari/537.36 MicroMessenger/8.0.48",
			deviceType: "mobile",
			model:      "SM-S918B",
			browser:    "WeChat",
			os:         "Android",
		},
		{
			name:       "desktop Edge",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0",
			deviceType: "desktop",
			browser:    "Microsoft Edge",
			os:         "Windows",
		},
		{
			name:       "iPhone Safari",
			userAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			deviceType: "mobile",
			model:      "iPhone",
			browser:    "Safari",
			os:         "iOS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVisitorUserAgent(tt.userAgent)
			if got.deviceType != tt.deviceType || got.deviceModel != tt.model || got.browser != tt.browser || got.os != tt.os {
				t.Fatalf("parseVisitorUserAgent() = %#v", got)
			}
		})
	}
}

func TestVisitorRequestFilter(t *testing.T) {
	for _, path := range []string{"/", "/api/memo/list", "/api/user/login"} {
		req := httptest.NewRequest("GET", path, nil)
		if !isVisitorRequest(req) {
			t.Fatalf("expected %s to be tracked", path)
		}
	}
	for _, path := range []string{"/favicon.ico", "/_nuxt/app.js", "/upload/example.jpg"} {
		req := httptest.NewRequest("GET", path, nil)
		if isVisitorRequest(req) {
			t.Fatalf("expected %s to be ignored", path)
		}
	}
}

func TestClientIPAddressUsesOnlyXRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.18.0.2:4567"
	req.Header.Set("X-Forwarded-For", "198.51.100.88")
	req.Header.Set("X-Real-IP", "203.0.113.42")
	if got := clientIPAddress(req); got != "203.0.113.42" {
		t.Fatalf("expected X-Real-IP, got %q", got)
	}
}
