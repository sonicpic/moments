package handler

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	projectdb "github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/vo"
	"gorm.io/gorm"
)

func TestResourceURLObjectKey(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		resourceURL    string
		expectedKey    string
		shouldBeParsed bool
	}{
		{
			name:           "bucket root",
			domain:         "https://img.example.com",
			resourceURL:    "https://img.example.com/2026/07/22/a.jpg",
			expectedKey:    "2026/07/22/a.jpg",
			shouldBeParsed: true,
		},
		{
			name:           "domain trailing slash and thumbnail query",
			domain:         "https://img.example.com/",
			resourceURL:    "https://img.example.com/imports/wechat/a.jpg?imageMogr2/thumbnail/300x",
			expectedKey:    "imports/wechat/a.jpg",
			shouldBeParsed: true,
		},
		{
			name:           "configured resource path prefix",
			domain:         "https://cdn.example.com/media",
			resourceURL:    "https://cdn.example.com/media/2026/a.jpg",
			expectedKey:    "2026/a.jpg",
			shouldBeParsed: true,
		},
		{
			name:           "similar but untrusted host",
			domain:         "https://img.example.com",
			resourceURL:    "https://img.example.com.evil.invalid/2026/a.jpg",
			shouldBeParsed: false,
		},
		{
			name:           "path traversal is rejected",
			domain:         "https://img.example.com",
			resourceURL:    "https://img.example.com/2026/%2E%2E/secret.jpg",
			shouldBeParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := resourceURLObjectKey(tt.domain, tt.resourceURL)
			if ok != tt.shouldBeParsed || actual != tt.expectedKey {
				t.Fatalf("resourceURLObjectKey() = (%q, %t), expected (%q, %t)", actual, ok, tt.expectedKey, tt.shouldBeParsed)
			}
		})
	}
}

func TestDeleteMemoRelationsAndQueueS3Cleanup(t *testing.T) {
	database := newS3CleanupTestDB(t)
	now := time.Now()
	pinned := false
	showType := int32(1)
	memo := projectdb.Memo{
		Id:        1,
		UserId:    1,
		Imgs:      "https://img.example.com/2026/07/22/a.jpg",
		CreatedAt: &now,
		UpdatedAt: &now,
		Pinned:    &pinned,
		ShowType:  &showType,
	}
	if err := database.Create(&memo).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&projectdb.Comment{Id: 1, MemoId: memo.Id, Content: "comment", CreatedAt: &now, UpdatedAt: &now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&projectdb.MemoLike{MemoID: memo.Id, VisitorID: "guest:test", CreatedAt: &now}).Error; err != nil {
		t.Fatal(err)
	}

	taskIDs, err := deleteMemoRelationsAndQueueS3Cleanup(database, memo, testFullS3Config())
	if err != nil {
		t.Fatal(err)
	}
	if len(taskIDs) != 1 {
		t.Fatalf("expected one queued object deletion, got %d", len(taskIDs))
	}

	assertTableCount(t, database, &projectdb.Memo{}, 0)
	assertTableCount(t, database, &projectdb.Comment{}, 0)
	assertTableCount(t, database, &projectdb.MemoLike{}, 0)

	var task projectdb.StorageDeletionTask
	if err := database.First(&task, taskIDs[0]).Error; err != nil {
		t.Fatal(err)
	}
	if task.MemoID != memo.Id || task.ObjectKey != "2026/07/22/a.jpg" || task.Bucket != "moments" {
		t.Fatalf("unexpected deletion task: %#v", task)
	}
}

func TestSharedS3ObjectIsNotQueuedForDeletion(t *testing.T) {
	database := newS3CleanupTestDB(t)
	now := time.Now()
	pinned := false
	showType := int32(1)
	sharedURL := "https://img.example.com/2026/07/22/shared.jpg"
	memo := projectdb.Memo{Id: 1, UserId: 1, Imgs: sharedURL, CreatedAt: &now, UpdatedAt: &now, Pinned: &pinned, ShowType: &showType}
	otherMemo := projectdb.Memo{Id: 2, UserId: 1, Imgs: sharedURL, CreatedAt: &now, UpdatedAt: &now, Pinned: &pinned, ShowType: &showType}
	if err := database.Create(&memo).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&otherMemo).Error; err != nil {
		t.Fatal(err)
	}

	taskIDs, err := deleteMemoRelationsAndQueueS3Cleanup(database, memo, testFullS3Config())
	if err != nil {
		t.Fatal(err)
	}
	if len(taskIDs) != 0 {
		t.Fatalf("shared object must not be queued for deletion, got %#v", taskIDs)
	}
	assertTableCount(t, database, &projectdb.Memo{}, 1)
	assertTableCount(t, database, &projectdb.StorageDeletionTask{}, 0)
}

func newS3CleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&projectdb.Memo{}, &projectdb.Comment{}, &projectdb.MemoLike{}, &projectdb.StorageDeletionTask{}, &projectdb.User{}, &projectdb.Friend{}); err != nil {
		t.Fatal(err)
	}
	return database
}

func testFullS3Config() vo.FullSysConfigVO {
	return vo.FullSysConfigVO{
		EnableS3: true,
		S3: vo.S3VO{
			Domain:    "https://img.example.com",
			Bucket:    "moments",
			Endpoint:  "https://account.example.r2.cloudflarestorage.com",
			Region:    "auto",
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		},
	}
}

func assertTableCount(t *testing.T, database *gorm.DB, model any, expected int64) {
	t.Helper()
	var actual int64
	if err := database.Model(model).Count(&actual).Error; err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("expected %d rows for %T, got %d", expected, model, actual)
	}
}
