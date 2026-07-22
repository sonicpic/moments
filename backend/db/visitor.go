package db

import "time"

// Visitor stores the server-side information observed for one browser.
// VisitorID is an opaque, random value stored in an HttpOnly cookie; it is not
// derived from the visitor's IP address or browser fingerprint.
type Visitor struct {
	ID uint `gorm:"column:id;primaryKey" json:"id,omitempty"`

	VisitorID string `gorm:"column:visitorId;size:36;not null;uniqueIndex" json:"-"`
	IPAddress string `gorm:"column:ipAddress;size:45" json:"ipAddress,omitempty"`
	UserAgent string `gorm:"column:userAgent;type:text" json:"userAgent,omitempty"`

	DeviceType  string `gorm:"column:deviceType;size:20" json:"deviceType,omitempty"`
	DeviceModel string `gorm:"column:deviceModel;size:160" json:"deviceModel,omitempty"`
	Browser     string `gorm:"column:browser;size:80" json:"browser,omitempty"`
	BrowserVer  string `gorm:"column:browserVer;size:40" json:"browserVer,omitempty"`
	OS          string `gorm:"column:os;size:80" json:"os,omitempty"`
	OSVersion   string `gorm:"column:osVersion;size:40" json:"osVersion,omitempty"`

	FirstSeenAt *time.Time `gorm:"column:firstSeenAt;not null" json:"firstSeenAt,omitempty"`
	LastSeenAt  *time.Time `gorm:"column:lastSeenAt;not null" json:"lastSeenAt,omitempty"`
	VisitCount  uint64     `gorm:"column:visitCount;default:0;not null" json:"visitCount,omitempty"`
}

func (Visitor) TableName() string {
	return "Visitor"
}
