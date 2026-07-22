package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/vo"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

const storageDeletionRetryInterval = 5 * time.Minute

func loadFullSysConfig(database *gorm.DB) (vo.FullSysConfigVO, error) {
	var sysConfig db.SysConfig
	if err := database.First(&sysConfig).Error; err != nil {
		return vo.FullSysConfigVO{}, err
	}

	var fullConfig vo.FullSysConfigVO
	if err := json.Unmarshal([]byte(sysConfig.Content), &fullConfig); err != nil {
		return vo.FullSysConfigVO{}, err
	}
	applyFullSysConfigDefaults(sysConfig.Content, &fullConfig)
	return fullConfig, nil
}

func s3DeletionConfigUsable(s3Config vo.S3VO) bool {
	return strings.TrimSpace(s3Config.Domain) != "" &&
		strings.TrimSpace(s3Config.Bucket) != "" &&
		strings.TrimSpace(s3Config.Endpoint) != "" &&
		strings.TrimSpace(s3Config.AccessKey) != "" &&
		strings.TrimSpace(s3Config.SecretKey) != ""
}

func newS3Client(s3Config vo.S3VO) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(s3Config.Region),
		config.WithEndpointResolver(
			aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: s3Config.Endpoint}, nil
			}),
		),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3Config.AccessKey, s3Config.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// resourceURLObjectKey accepts only URLs that are below the configured public
// resource domain. The resulting key is the same key used by the S3 API; the
// bucket name is never inferred from, or appended to, a public URL.
func resourceURLObjectKey(resourceDomain, rawURL string) (string, bool) {
	base, err := url.Parse(strings.TrimSpace(resourceDomain))
	if err != nil || base.Scheme == "" || base.Host == "" || base.User != nil {
		return "", false
	}
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || target.Scheme == "" || target.Host == "" || target.User != nil {
		return "", false
	}
	if !strings.EqualFold(base.Scheme, target.Scheme) || !strings.EqualFold(base.Host, target.Host) {
		return "", false
	}

	basePath := strings.TrimRight(base.EscapedPath(), "/")
	targetPath := target.EscapedPath()
	prefix := "/"
	if basePath != "" {
		prefix = basePath + "/"
	}
	if !strings.HasPrefix(targetPath, prefix) {
		return "", false
	}

	escapedKey := strings.TrimPrefix(targetPath, basePath)
	escapedKey = strings.TrimPrefix(escapedKey, "/")
	key, err := url.PathUnescape(escapedKey)
	if err != nil || key == "" || strings.ContainsRune(key, '\x00') {
		return "", false
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return "", false
		}
	}
	return key, true
}

func memoS3ObjectKeys(memo db.Memo, s3Config vo.S3VO) map[string]struct{} {
	keys := make(map[string]struct{})
	addURL := func(rawURL string) {
		if key, ok := resourceURLObjectKey(s3Config.Domain, rawURL); ok {
			keys[key] = struct{}{}
		}
	}

	for _, imageURL := range strings.Split(memo.Imgs, ",") {
		addURL(strings.TrimSpace(imageURL))
	}
	addURL(memo.ExternalFavicon)

	var ext vo.MemoExt
	if err := json.Unmarshal([]byte(memo.Ext), &ext); err == nil {
		addURL(ext.DoubanBook.Image)
		addURL(ext.DoubanMovie.Image)
		addURL(ext.Video.Value)
	}
	return keys
}

func sysConfigS3ObjectKeys(sysConfig vo.FullSysConfigVO) map[string]struct{} {
	keys := make(map[string]struct{})
	addURL := func(rawURL string) {
		if key, ok := resourceURLObjectKey(sysConfig.S3.Domain, rawURL); ok {
			keys[key] = struct{}{}
		}
	}

	addURL(sysConfig.Favicon)
	addURL(sysConfig.BackgroundMusicUrl)
	for _, music := range sysConfig.BackgroundMusicList {
		addURL(music.URL)
		addURL(music.LyricsURL)
	}
	return keys
}

// s3ObjectReferencedElsewhere protects an object if another database record
// still references it. Deleting a memo is uncommon and the dataset is small,
// so a full reference check is preferable to unsafe string matching.
func s3ObjectReferencedElsewhere(tx *gorm.DB, sysConfig vo.FullSysConfigVO, excludedMemoID int32, objectKey string) (bool, error) {
	var memos []db.Memo
	if err := tx.Select("id", "imgs", "ext", "externalFavicon").Where("id <> ?", excludedMemoID).Find(&memos).Error; err != nil {
		return false, err
	}
	for _, memo := range memos {
		if _, exists := memoS3ObjectKeys(memo, sysConfig.S3)[objectKey]; exists {
			return true, nil
		}
	}

	var users []db.User
	if err := tx.Select("id", "avatarUrl", "coverUrl", "favicon").Find(&users).Error; err != nil {
		return false, err
	}
	for _, user := range users {
		for _, resourceURL := range []string{user.AvatarUrl, user.CoverUrl, user.Favicon} {
			if key, ok := resourceURLObjectKey(sysConfig.S3.Domain, resourceURL); ok && key == objectKey {
				return true, nil
			}
		}
	}

	var friends []db.Friend
	if err := tx.Select("icon").Find(&friends).Error; err != nil {
		return false, err
	}
	for _, friend := range friends {
		if key, ok := resourceURLObjectKey(sysConfig.S3.Domain, friend.Icon); ok && key == objectKey {
			return true, nil
		}
	}

	_, exists := sysConfigS3ObjectKeys(sysConfig)[objectKey]
	return exists, nil
}

func queueMemoS3DeletionTasks(tx *gorm.DB, memo db.Memo, sysConfig vo.FullSysConfigVO) ([]uint, error) {
	taskIDs := make([]uint, 0)
	for objectKey := range memoS3ObjectKeys(memo, sysConfig.S3) {
		inUse, err := s3ObjectReferencedElsewhere(tx, sysConfig, memo.Id, objectKey)
		if err != nil {
			return nil, err
		}
		if inUse {
			continue
		}

		var task db.StorageDeletionTask
		err = tx.Where("bucket = ? AND endpoint = ? AND objectKey = ?",
			strings.TrimSpace(sysConfig.S3.Bucket),
			normalizeS3Endpoint(sysConfig.S3.Endpoint),
			objectKey,
		).First(&task).Error
		switch {
		case err == nil:
			taskIDs = append(taskIDs, task.ID)
		case errors.Is(err, gorm.ErrRecordNotFound):
			task = db.StorageDeletionTask{
				MemoID:    memo.Id,
				Bucket:    strings.TrimSpace(sysConfig.S3.Bucket),
				Endpoint:  normalizeS3Endpoint(sysConfig.S3.Endpoint),
				Region:    strings.TrimSpace(sysConfig.S3.Region),
				ObjectKey: objectKey,
			}
			if err := tx.Create(&task).Error; err != nil {
				return nil, err
			}
			taskIDs = append(taskIDs, task.ID)
		default:
			return nil, err
		}
	}
	return taskIDs, nil
}

func deleteMemoRelationsAndQueueS3Cleanup(database *gorm.DB, memo db.Memo, sysConfig vo.FullSysConfigVO) ([]uint, error) {
	if sysConfig.EnableS3 && !s3DeletionConfigUsable(sysConfig.S3) {
		return nil, errors.New("S3配置不完整，未删除朋友圈")
	}

	var taskIDs []uint
	err := database.Transaction(func(tx *gorm.DB) error {
		if sysConfig.EnableS3 {
			var err error
			taskIDs, err = queueMemoS3DeletionTasks(tx, memo, sysConfig)
			if err != nil {
				return err
			}
		}
		if err := tx.Where("memoId = ?", memo.Id).Delete(&db.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("memoId = ?", memo.Id).Delete(&db.MemoLike{}).Error; err != nil {
			return err
		}
		if result := tx.Delete(&memo); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return errors.New("删除朋友圈失败")
		}
		return nil
	})
	return taskIDs, err
}

func normalizeS3Endpoint(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func storageTaskMatchesConfig(task db.StorageDeletionTask, sysConfig vo.FullSysConfigVO) bool {
	return sysConfig.EnableS3 &&
		strings.TrimSpace(sysConfig.S3.Bucket) == task.Bucket &&
		normalizeS3Endpoint(sysConfig.S3.Endpoint) == task.Endpoint &&
		strings.TrimSpace(sysConfig.S3.Region) == task.Region
}

func markStorageDeletionTaskFailed(database *gorm.DB, taskID uint, err error) {
	now := time.Now()
	message := err.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	_ = database.Model(&db.StorageDeletionTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"attemptCount":  gorm.Expr("attemptCount + ?", 1),
		"lastAttemptAt": now,
		"lastError":     message,
	}).Error
}

// processStorageDeletionTasks attempts the specified durable tasks immediately.
// Failed tasks remain in SQLite and are retried by the background worker.
func processStorageDeletionTasks(database *gorm.DB, log zerolog.Logger, taskIDs []uint) int {
	if len(taskIDs) == 0 {
		return 0
	}

	sysConfig, err := loadFullSysConfig(database)
	if err != nil {
		for _, taskID := range taskIDs {
			markStorageDeletionTaskFailed(database, taskID, err)
		}
		log.Error().Err(err).Msg("读取S3删除任务配置失败")
		return len(taskIDs)
	}
	if !sysConfig.EnableS3 || !s3DeletionConfigUsable(sysConfig.S3) {
		err := errors.New("当前S3配置不完整")
		for _, taskID := range taskIDs {
			markStorageDeletionTaskFailed(database, taskID, err)
		}
		log.Error().Msg("S3删除任务等待完整配置")
		return len(taskIDs)
	}

	client, err := newS3Client(sysConfig.S3)
	if err != nil {
		for _, taskID := range taskIDs {
			markStorageDeletionTaskFailed(database, taskID, err)
		}
		log.Error().Err(err).Msg("创建S3删除客户端失败")
		return len(taskIDs)
	}

	failed := 0
	for _, taskID := range taskIDs {
		var task db.StorageDeletionTask
		if err := database.First(&task, taskID).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				failed++
				log.Error().Err(err).Uint("taskId", taskID).Msg("读取S3删除任务失败")
			}
			continue
		}
		if !storageTaskMatchesConfig(task, sysConfig) {
			err := errors.New("当前S3配置与待删除对象不匹配")
			markStorageDeletionTaskFailed(database, task.ID, err)
			failed++
			log.Warn().Uint("taskId", task.ID).Int32("memoId", task.MemoID).Msg("S3删除任务等待匹配的存储配置")
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(task.Bucket),
			Key:    aws.String(task.ObjectKey),
		})
		cancel()
		if err != nil {
			markStorageDeletionTaskFailed(database, task.ID, err)
			failed++
			log.Error().Err(err).Uint("taskId", task.ID).Int32("memoId", task.MemoID).Msg("删除S3对象失败，将自动重试")
			continue
		}
		if err := database.Delete(&task).Error; err != nil {
			failed++
			log.Error().Err(err).Uint("taskId", task.ID).Msg("S3对象已删除，但清理删除任务失败")
		}
	}
	return failed
}

func retryPendingStorageDeletionTasks(database *gorm.DB, log zerolog.Logger) {
	var taskIDs []uint
	if err := database.Model(&db.StorageDeletionTask{}).Order("id ASC").Limit(100).Pluck("id", &taskIDs).Error; err != nil {
		log.Error().Err(err).Msg("读取待删除S3对象任务失败")
		return
	}
	processStorageDeletionTasks(database, log, taskIDs)
}

// StartStorageDeletionWorker retries durable S3 deletion tasks after startup
// and periodically thereafter. DeleteObject is idempotent, so a retry after an
// interrupted request is safe.
func StartStorageDeletionWorker(injector do.Injector) {
	database := do.MustInvoke[*gorm.DB](injector)
	log := do.MustInvoke[zerolog.Logger](injector)

	go func() {
		retryPendingStorageDeletionTasks(database, log)
		ticker := time.NewTicker(storageDeletionRetryInterval)
		defer ticker.Stop()
		for range ticker.C {
			retryPendingStorageDeletionTasks(database, log)
		}
	}()
}

func storageDeletionPendingMessage(failed int) string {
	return fmt.Sprintf("%d 个S3对象等待自动清理", failed)
}
