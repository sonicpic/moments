package db

import "time"

// MemoLike keeps one like for each signed-in account or anonymous visitor.
// VisitorID is stored in an HttpOnly cookie for guests, so a repeated click can
// reliably cancel only that visitor's own like.
type MemoLike struct {
	ID        uint       `gorm:"column:id;primaryKey" json:"id,omitempty"`
	MemoID    int32      `gorm:"column:memoId;not null;uniqueIndex:idx_memo_like_visitor" json:"memoId,omitempty"`
	VisitorID string     `gorm:"column:visitorId;not null;size:80;uniqueIndex:idx_memo_like_visitor" json:"-"`
	CreatedAt *time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP" json:"createdAt,omitempty"`
}

func (MemoLike) TableName() string {
	return "MemoLike"
}
