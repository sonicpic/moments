package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	model "github.com/kingwrcy/moments/db"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

const (
	visitorCookieName    = "moments_visitor"
	visitorTouchInterval = 5 * time.Minute
)

type visitorUserAgent struct {
	deviceType  string
	deviceModel string
	browser     string
	browserVer  string
	os          string
	osVersion   string
}

// Visitor records page/API visits without adding client-side tracking code.
// Static files are deliberately ignored so assets do not inflate visit counts.
func Visitor(injector do.Injector) echo.MiddlewareFunc {
	database := do.MustInvoke[*gorm.DB](injector)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isVisitorRequest(c.Request()) {
				touchVisitor(database, c)
			}
			return next(c)
		}
	}
}

func isVisitorRequest(request *http.Request) bool {
	if request.Method != http.MethodGet && request.Method != http.MethodPost {
		return false
	}
	return request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/api/")
}

func touchVisitor(database *gorm.DB, c echo.Context) {
	visitorID := visitorCookie(c)
	now := time.Now()
	userAgent := c.Request().UserAgent()
	info := parseVisitorUserAgent(userAgent)

	var visitor model.Visitor
	err := database.Where("visitorId = ?", visitorID).First(&visitor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = database.Create(&model.Visitor{
				VisitorID:   visitorID,
				IPAddress:   clientIPAddress(c.Request()),
				UserAgent:   userAgent,
				DeviceType:  info.deviceType,
				DeviceModel: info.deviceModel,
				Browser:     info.browser,
				BrowserVer:  info.browserVer,
				OS:          info.os,
				OSVersion:   info.osVersion,
				FirstSeenAt: &now,
				LastSeenAt:  &now,
				VisitCount:  1,
			}).Error
		}
		return
	}

	// One browser can make many API calls for one page load. Throttling updates
	// avoids a database write for each request while retaining useful last-seen
	// and visit-count information.
	if visitor.LastSeenAt != nil && now.Sub(*visitor.LastSeenAt) < visitorTouchInterval {
		return
	}

	_ = database.Model(&model.Visitor{}).Where("id = ?", visitor.ID).Updates(map[string]any{
		"ipAddress":   clientIPAddress(c.Request()),
		"userAgent":   userAgent,
		"deviceType":  info.deviceType,
		"deviceModel": info.deviceModel,
		"browser":     info.browser,
		"browserVer":  info.browserVer,
		"os":          info.os,
		"osVersion":   info.osVersion,
		"lastSeenAt":  now,
		"visitCount":  gorm.Expr("visitCount + ?", 1),
	}).Error
}

func visitorCookie(c echo.Context) string {
	if cookie, err := c.Cookie(visitorCookieName); err == nil {
		if _, err := uuid.Parse(cookie.Value); err == nil {
			return cookie.Value
		}
	}

	visitorID := uuid.NewString()
	c.SetCookie(&http.Cookie{
		Name:     visitorCookieName,
		Value:    visitorID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	return visitorID
}

// clientIPAddress deliberately trusts only X-Real-IP. Nginx owns this header
// and sets it from Cloudflare's verified client address; X-Forwarded-For is
// not accepted directly from a client.
func clientIPAddress(request *http.Request) string {
	if ip := net.ParseIP(strings.TrimSpace(request.Header.Get(echo.HeaderXRealIP))); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(request.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return ""
}

func parseVisitorUserAgent(raw string) visitorUserAgent {
	lower := strings.ToLower(raw)
	info := visitorUserAgent{deviceType: "desktop", browser: "Unknown", os: "Unknown"}

	switch {
	case containsAny(lower, "bot", "spider", "crawler", "curl/", "wget/"):
		info.deviceType = "bot"
	case strings.Contains(lower, "ipad"):
		info.deviceType = "tablet"
		info.deviceModel = "iPad"
	case strings.Contains(lower, "iphone"):
		info.deviceType = "mobile"
		info.deviceModel = "iPhone"
	case strings.Contains(lower, "android"):
		info.deviceType = "mobile"
		info.deviceModel = androidDeviceModel(raw)
	case containsAny(lower, "mobile", "windows phone"):
		info.deviceType = "mobile"
	}

	switch {
	case strings.Contains(raw, "Android"):
		info.os = "Android"
		info.osVersion = tokenVersion(raw, "Android ")
	case strings.Contains(raw, "iPhone OS "):
		info.os = "iOS"
		info.osVersion = strings.ReplaceAll(tokenVersion(raw, "iPhone OS "), "_", ".")
	case strings.Contains(raw, "CPU OS "):
		info.os = "iPadOS"
		info.osVersion = strings.ReplaceAll(tokenVersion(raw, "CPU OS "), "_", ".")
	case strings.Contains(raw, "Windows NT "):
		info.os = "Windows"
		info.osVersion = tokenVersion(raw, "Windows NT ")
	case strings.Contains(raw, "Mac OS X "):
		info.os = "macOS"
		info.osVersion = strings.ReplaceAll(tokenVersion(raw, "Mac OS X "), "_", ".")
	case strings.Contains(lower, "linux"):
		info.os = "Linux"
	}

	switch {
	case strings.Contains(raw, "MicroMessenger/"):
		info.browser, info.browserVer = "WeChat", tokenVersion(raw, "MicroMessenger/")
	case strings.Contains(raw, "Edg/"):
		info.browser, info.browserVer = "Microsoft Edge", tokenVersion(raw, "Edg/")
	case strings.Contains(raw, "EdgiOS/"):
		info.browser, info.browserVer = "Microsoft Edge", tokenVersion(raw, "EdgiOS/")
	case strings.Contains(raw, "OPR/"):
		info.browser, info.browserVer = "Opera", tokenVersion(raw, "OPR/")
	case strings.Contains(raw, "Firefox/"):
		info.browser, info.browserVer = "Firefox", tokenVersion(raw, "Firefox/")
	case strings.Contains(raw, "FxiOS/"):
		info.browser, info.browserVer = "Firefox", tokenVersion(raw, "FxiOS/")
	case strings.Contains(raw, "CriOS/"):
		info.browser, info.browserVer = "Chrome", tokenVersion(raw, "CriOS/")
	case strings.Contains(raw, "Chrome/"):
		info.browser, info.browserVer = "Chrome", tokenVersion(raw, "Chrome/")
	case strings.Contains(raw, "Version/") && strings.Contains(raw, "Safari/"):
		info.browser, info.browserVer = "Safari", tokenVersion(raw, "Version/")
	}

	return info
}

func androidDeviceModel(raw string) string {
	buildAt := strings.Index(raw, " Build/")
	if buildAt < 0 {
		buildAt = strings.Index(raw, " BUILD/")
	}
	if buildAt < 0 {
		return ""
	}

	openAt := strings.LastIndex(raw[:buildAt], "(")
	if openAt < 0 {
		return ""
	}
	parts := strings.Split(raw[openAt+1:buildAt], ";")
	if len(parts) == 0 {
		return ""
	}
	model := strings.TrimSpace(parts[len(parts)-1])
	if model == "" || strings.EqualFold(model, "K") || strings.EqualFold(model, "wv") {
		return ""
	}
	return model
}

func tokenVersion(raw, token string) string {
	start := strings.Index(raw, token)
	if start < 0 {
		return ""
	}
	value := raw[start+len(token):]
	if end := strings.IndexAny(value, " ;)"); end >= 0 {
		value = value[:end]
	}
	return strings.TrimSpace(value)
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
