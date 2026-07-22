package db

import "time"

// StorageDeletionTask is a durable, retryable request to remove one object
// from an S3-compatible bucket. It deliberately stores no credentials: the
// current system S3 credentials are used only when the saved bucket and
// endpoint still match the active configuration.
type StorageDeletionTask struct {
	ID uint `gorm:"column:id;primaryKey" json:"id,omitempty"`

	MemoID    int32  `gorm:"column:memoId;not null" json:"memoId,omitempty"`
	Bucket    string `gorm:"column:bucket;size:255;not null;uniqueIndex:idx_storage_delete_object" json:"bucket,omitempty"`
	Endpoint  string `gorm:"column:endpoint;type:text;not null;uniqueIndex:idx_storage_delete_object" json:"endpoint,omitempty"`
	Region    string `gorm:"column:region;size:80" json:"region,omitempty"`
	ObjectKey string `gorm:"column:objectKey;type:text;not null;uniqueIndex:idx_storage_delete_object" json:"objectKey,omitempty"`

	AttemptCount  uint       `gorm:"column:attemptCount;default:0;not null" json:"attemptCount,omitempty"`
	LastAttemptAt *time.Time `gorm:"column:lastAttemptAt" json:"lastAttemptAt,omitempty"`
	LastError     string     `gorm:"column:lastError;type:text" json:"lastError,omitempty"`
	CreatedAt     *time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP;not null" json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `gorm:"column:updatedAt;not null" json:"updatedAt,omitempty"`
}

func (StorageDeletionTask) TableName() string {
	return "StorageDeletionTask"
}
