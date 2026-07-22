// Command import_wechat_moments imports a WeChat Moments JSON export into a
// Moments database. It deliberately uses the same S3-compatible configuration
// that the running application uses, while retaining an import marker in Ext so
// that interrupted runs can be safely resumed.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/glebarez/sqlite"
	projectdb "github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/vo"
	"gorm.io/gorm"
)

type exportFile struct {
	ExportTime string       `json:"exportTime"`
	TotalPosts int          `json:"totalPosts"`
	Posts      []sourcePost `json:"posts"`
}

type sourcePost struct {
	ID          string          `json:"id"`
	Nickname    string          `json:"nickname"`
	CreateTime  int64           `json:"createTime"`
	ContentDesc string          `json:"contentDesc"`
	Type        int             `json:"type"`
	Media       []sourceMedia   `json:"media"`
	Likes       []string        `json:"likes"`
	Comments    []sourceComment `json:"comments"`
	Location    sourceLocation  `json:"location"`
}

type sourceMedia struct {
	URL       string `json:"url"`
	Thumb     string `json:"thumb"`
	LocalPath string `json:"localPath"`
}

type sourceComment struct {
	Nickname     string `json:"nickname"`
	Content      string `json:"content"`
	RefNickname  string `json:"refNickname"`
	RefCommentID string `json:"refCommentId"`
}

type sourceLocation struct {
	City           string `json:"city"`
	Country        string `json:"country"`
	POIName        string `json:"poiName"`
	POIAddressName string `json:"poiAddressName"`
}

type reportEvent struct {
	SourceID string   `json:"sourceId,omitempty"`
	Status   string   `json:"status"`
	MemoID   int32    `json:"memoId,omitempty"`
	Images   int      `json:"images,omitempty"`
	Comments int      `json:"comments,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Error    string   `json:"error,omitempty"`
}

type summary struct {
	StartedAt        time.Time `json:"startedAt"`
	FinishedAt       time.Time `json:"finishedAt"`
	Source           string    `json:"source"`
	DryRun           bool      `json:"dryRun"`
	Backup           string    `json:"backup,omitempty"`
	Report           string    `json:"report"`
	TotalPosts       int       `json:"totalPosts"`
	ImportedPosts    int       `json:"importedPosts"`
	SkippedPosts     int       `json:"skippedPosts"`
	FailedPosts      int       `json:"failedPosts"`
	UploadedImages   int       `json:"uploadedImages"`
	ImportedComments int       `json:"importedComments"`
	UnsupportedMedia int       `json:"unsupportedMedia"`
	Warnings         int       `json:"warnings"`
}

type uploader struct {
	client *s3.Client
	bucket string
	domain string
	dryRun bool
}

func main() {
	sourcePath := flag.String("source", "", "微信朋友圈导出的 JSON 文件")
	mediaDir := flag.String("media-dir", "", "媒体目录，默认使用 JSON 所在目录")
	dbPath := flag.String("db", "/home/docker/moments/data/db.sqlite", "Moments SQLite 数据库路径")
	ownerID := flag.Int("owner-id", 1, "导入到的 Moments 用户 ID")
	dryRun := flag.Bool("dry-run", false, "仅检查和生成报告，不上传或写入")
	auditOnly := flag.Bool("audit-only", false, "仅统计已导入记录及重复来源 ID，不上传或写入")
	deduplicate := flag.Bool("deduplicate", false, "删除由并发导入造成的重复记录，仅保留每个来源 ID 最早的一条")
	limit := flag.Int("limit", 0, "最多处理多少条，0 表示全部")
	reportDir := flag.String("report-dir", "", "报告输出目录，默认使用 JSON 所在目录")
	flag.Parse()

	if *sourcePath == "" {
		fatalf("必须提供 --source")
	}
	if *ownerID <= 0 {
		fatalf("--owner-id 必须大于 0")
	}

	export, err := loadExport(*sourcePath)
	if err != nil {
		fatalf("读取导出文件失败: %v", err)
	}
	if *mediaDir == "" {
		*mediaDir = filepath.Dir(*sourcePath)
	}
	if *reportDir == "" {
		*reportDir = filepath.Dir(*sourcePath)
	}
	if err := os.MkdirAll(*reportDir, 0755); err != nil {
		fatalf("创建报告目录失败: %v", err)
	}

	now := time.Now().UTC()
	prefix := fmt.Sprintf("moments-import-%s", now.Format("20060102T150405Z"))
	eventPath := filepath.Join(*reportDir, prefix+".jsonl")
	summaryPath := filepath.Join(*reportDir, prefix+".summary.json")
	eventFile, err := os.Create(eventPath)
	if err != nil {
		fatalf("创建导入报告失败: %v", err)
	}
	defer eventFile.Close()

	result := summary{
		StartedAt:  now,
		Source:     *sourcePath,
		DryRun:     *dryRun,
		Report:     eventPath,
		TotalPosts: len(export.Posts),
	}
	defer func() {
		result.FinishedAt = time.Now().UTC()
		data, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr == nil {
			_ = os.WriteFile(summaryPath, data, 0644)
		}
	}()

	gdb, err := gorm.Open(sqlite.Open(*dbPath+"?_pragma=busy_timeout(10000)"), &gorm.Config{})
	if err != nil {
		fatalf("打开 Moments 数据库失败: %v", err)
	}
	lockFile, err := acquireImportLock(*dbPath)
	if err != nil {
		fatalf("已有另一个朋友圈导入或维护任务正在使用该数据库，请等待其完成后重试: %v", err)
	}
	defer func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
	}()
	if err := validateOwner(gdb, int32(*ownerID)); err != nil {
		fatalf("导入用户校验失败: %v", err)
	}
	if *auditOnly {
		if err := auditImportedPosts(gdb); err != nil {
			fatalf("导入记录审计失败: %v", err)
		}
		return
	}
	if *deduplicate {
		backupPath := filepath.Join(filepath.Dir(*dbPath), fmt.Sprintf("before-wechat-import-deduplicate-%s.sqlite", now.Format("20060102T150405Z")))
		if err := backupDatabase(gdb, backupPath); err != nil {
			fatalf("创建去重前数据库备份失败: %v", err)
		}
		removedMemos, removedComments, err := deduplicateImportedPosts(gdb)
		if err != nil {
			fatalf("去重失败: %v", err)
		}
		fmt.Printf("已删除 %d 条重复朋友圈及其 %d 条关联评论。备份：%s\n", removedMemos, removedComments, backupPath)
		return
	}

	var sysConfig projectdb.SysConfig
	if err := gdb.First(&sysConfig).Error; err != nil {
		fatalf("读取系统 S3 配置失败: %v", err)
	}
	var fullConfig vo.FullSysConfigVO
	if err := json.Unmarshal([]byte(sysConfig.Content), &fullConfig); err != nil {
		fatalf("解析系统 S3 配置失败: %v", err)
	}
	if !fullConfig.EnableS3 {
		fatalf("系统未启用 S3，已停止导入以避免图片落到本地")
	}
	if fullConfig.S3.Bucket == "" || fullConfig.S3.Domain == "" || fullConfig.S3.Endpoint == "" || fullConfig.S3.AccessKey == "" || fullConfig.S3.SecretKey == "" {
		fatalf("S3 配置不完整，已停止导入")
	}

	// Even a dry run constructs this lightweight configuration holder, so that
	// the reported public URLs use the exact same domain/key join as a real run.
	// No network request is made until upload is called.
	storage, err := newUploader(fullConfig, *dryRun)
	if err != nil {
		fatalf("初始化 S3 上传器失败: %v", err)
	}
	if !*dryRun {
		backupPath := filepath.Join(filepath.Dir(*dbPath), fmt.Sprintf("before-wechat-import-%s.sqlite", now.Format("20060102T150405Z")))
		if err := backupDatabase(gdb, backupPath); err != nil {
			fatalf("创建导入前数据库备份失败: %v", err)
		}
		result.Backup = backupPath
	}

	posts := append([]sourcePost(nil), export.Posts...)
	sort.Slice(posts, func(i, j int) bool { return posts[i].CreateTime < posts[j].CreateTime })
	if *limit > 0 && len(posts) > *limit {
		posts = posts[:*limit]
	}

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	for index, post := range posts {
		event := importPost(context.Background(), gdb, storage, post, *mediaDir, int32(*ownerID), shanghai, *dryRun)
		writeEvent(eventFile, event)
		switch event.Status {
		case "imported":
			result.ImportedPosts++
			result.UploadedImages += event.Images
			result.ImportedComments += event.Comments
		case "skipped":
			result.SkippedPosts++
		case "failed":
			result.FailedPosts++
		}
		result.Warnings += len(event.Warnings)
		result.UnsupportedMedia += countUnsupported(event.Warnings)
		if (index+1)%25 == 0 || index+1 == len(posts) {
			fmt.Fprintf(os.Stderr, "已处理 %d/%d 条朋友圈（新增 %d，跳过 %d，失败 %d）\n", index+1, len(posts), result.ImportedPosts, result.SkippedPosts, result.FailedPosts)
		}
	}

	fmt.Printf("导入完成：新增 %d，跳过 %d，失败 %d；图片 %d，评论 %d。\n报告：%s\n汇总：%s\n", result.ImportedPosts, result.SkippedPosts, result.FailedPosts, result.UploadedImages, result.ImportedComments, eventPath, summaryPath)
}

func loadExport(path string) (exportFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return exportFile{}, err
	}
	defer file.Close()
	var result exportFile
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return exportFile{}, err
	}
	if len(result.Posts) == 0 {
		return exportFile{}, errors.New("导出文件不包含 posts")
	}
	return result, nil
}

func validateOwner(gdb *gorm.DB, ownerID int32) error {
	var user projectdb.User
	if err := gdb.First(&user, ownerID).Error; err != nil {
		return err
	}
	return nil
}

func acquireImportLock(dbPath string) (*os.File, error) {
	file, err := os.OpenFile(dbPath+".wechat-import.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func auditImportedPosts(gdb *gorm.DB) error {
	const marker = "%\"wechatMomentId\":%"
	var total int64
	if err := gdb.Model(&projectdb.Memo{}).Where("ext LIKE ?", marker).Count(&total).Error; err != nil {
		return err
	}

	type duplicate struct {
		Ext   string
		Count int64
	}
	var duplicates []duplicate
	if err := gdb.Model(&projectdb.Memo{}).
		Select("ext, COUNT(*) AS count").
		Where("ext LIKE ?", marker).
		Group("ext").
		Having("COUNT(*) > 1").
		Find(&duplicates).Error; err != nil {
		return err
	}

	var duplicateRows int64
	for _, item := range duplicates {
		duplicateRows += item.Count - 1
	}

	importedMemos := gdb.Model(&projectdb.Memo{}).Select("id, imgs").Where("ext LIKE ?", marker)
	var importedComments int64
	if err := gdb.Model(&projectdb.Comment{}).Where("memoId IN (?)", importedMemos.Select("id")).Count(&importedComments).Error; err != nil {
		return err
	}
	var memoImages []struct{ Imgs string }
	if err := gdb.Model(&projectdb.Memo{}).Select("imgs").Where("ext LIKE ?", marker).Find(&memoImages).Error; err != nil {
		return err
	}
	imageRefs := 0
	for _, memo := range memoImages {
		if strings.TrimSpace(memo.Imgs) == "" {
			continue
		}
		imageRefs += len(strings.Split(memo.Imgs, ","))
	}

	fmt.Printf("导入标识记录：%d；唯一来源：%d；重复来源：%d；多余记录：%d；图片引用：%d；关联评论：%d。\n", total, total-duplicateRows, len(duplicates), duplicateRows, imageRefs, importedComments)
	for _, item := range duplicates {
		fmt.Printf("重复 %d 次：%s\n", item.Count, item.Ext)
	}
	return nil
}

func deduplicateImportedPosts(gdb *gorm.DB) (int64, int64, error) {
	const marker = "%\"wechatMomentId\":%"
	type duplicate struct {
		Ext string
	}
	var duplicates []duplicate
	if err := gdb.Model(&projectdb.Memo{}).
		Select("ext").
		Where("ext LIKE ?", marker).
		Group("ext").
		Having("COUNT(*) > 1").
		Find(&duplicates).Error; err != nil {
		return 0, 0, err
	}

	var removedMemos, removedComments int64
	err := gdb.Transaction(func(tx *gorm.DB) error {
		for _, item := range duplicates {
			var records []projectdb.Memo
			if err := tx.Where("ext = ?", item.Ext).Order("id ASC").Find(&records).Error; err != nil {
				return err
			}
			if len(records) < 2 {
				continue
			}
			ids := make([]int32, 0, len(records)-1)
			for _, memo := range records[1:] {
				ids = append(ids, memo.Id)
			}
			var commentCount int64
			if err := tx.Model(&projectdb.Comment{}).Where("memoId IN ?", ids).Count(&commentCount).Error; err != nil {
				return err
			}
			if err := tx.Where("memoId IN ?", ids).Delete(&projectdb.Comment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", ids).Delete(&projectdb.Memo{}).Error; err != nil {
				return err
			}
			removedMemos += int64(len(ids))
			removedComments += commentCount
		}
		return nil
	})
	return removedMemos, removedComments, err
}

func newUploader(sysConfig vo.FullSysConfigVO, dryRun bool) (*uploader, error) {
	awsConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(sysConfig.S3.Region),
		config.WithEndpointResolver(
			aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: sysConfig.S3.Endpoint}, nil
			}),
		),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(sysConfig.S3.AccessKey, sysConfig.S3.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	return &uploader{client: s3.NewFromConfig(awsConfig), bucket: sysConfig.S3.Bucket, domain: sysConfig.S3.Domain, dryRun: dryRun}, nil
}

func backupDatabase(gdb *gorm.DB, destination string) error {
	escaped := strings.ReplaceAll(destination, "'", "''")
	return gdb.Exec("VACUUM INTO '" + escaped + "'").Error
}

func importPost(ctx context.Context, gdb *gorm.DB, storage *uploader, post sourcePost, mediaDir string, ownerID int32, timezone *time.Location, dryRun bool) reportEvent {
	event := reportEvent{SourceID: post.ID}
	if post.ID == "" || post.CreateTime <= 0 {
		event.Status = "failed"
		event.Error = "缺少朋友圈 ID 或创建时间"
		return event
	}

	alreadyImported, err := hasImportedPost(gdb, post.ID)
	if err != nil {
		event.Status = "failed"
		event.Error = fmt.Sprintf("检查重复导入失败: %v", err)
		return event
	}
	if alreadyImported {
		event.Status = "skipped"
		event.Warnings = []string{"已存在相同微信朋友圈 ID，未重复导入"}
		return event
	}

	createdAt := time.Unix(post.CreateTime, 0).In(timezone)
	imageURLs := make([]string, 0, len(post.Media))
	warnings := make([]string, 0)
	for index, item := range post.Media {
		if item.LocalPath == "" {
			if item.URL != "" {
				warnings = append(warnings, fmt.Sprintf("第 %d 个媒体仅有远程链接，未导入：%s", index+1, abbreviatedURL(item.URL)))
			}
			continue
		}
		localPath, err := safeMediaPath(mediaDir, item.LocalPath)
		if err != nil {
			event.Status = "failed"
			event.Error = fmt.Sprintf("第 %d 个媒体路径非法: %v", index+1, err)
			return event
		}
		if _, err := os.Stat(localPath); err != nil {
			event.Status = "failed"
			event.Error = fmt.Sprintf("第 %d 个本地图片不存在: %s", index+1, item.LocalPath)
			return event
		}

		key, publicURL, err := importObjectKey(createdAt, post.ID, index, localPath, storage)
		if err != nil {
			event.Status = "failed"
			event.Error = fmt.Sprintf("第 %d 个图片对象键生成失败: %v", index+1, err)
			return event
		}
		if !dryRun {
			if err := storage.upload(ctx, key, localPath); err != nil {
				event.Status = "failed"
				event.Error = fmt.Sprintf("第 %d 个图片上传 S3 失败: %v", index+1, err)
				return event
			}
		}
		imageURLs = append(imageURLs, publicURL)
	}

	if dryRun {
		event.Status = "imported"
		event.Images = len(imageURLs)
		event.Comments = len(post.Comments)
		event.Warnings = warnings
		return event
	}

	ext := map[string]any{
		"wechatMomentId": post.ID,
		"wechatType":     post.Type,
	}
	extJSON, err := json.Marshal(ext)
	if err != nil {
		event.Status = "failed"
		event.Error = fmt.Sprintf("生成扩展信息失败: %v", err)
		return event
	}

	pinned := false
	showType := int32(1)
	memo := projectdb.Memo{
		Content:      strings.TrimSpace(post.ContentDesc),
		Imgs:         strings.Join(imageURLs, ","),
		FavCount:     int32(len(post.Likes)),
		CommentCount: int32(len(post.Comments)),
		UserId:       ownerID,
		CreatedAt:    &createdAt,
		UpdatedAt:    &createdAt,
		Location:     formatLocation(post.Location),
		Pinned:       &pinned,
		Ext:          string(extJSON),
		ShowType:     &showType,
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&memo).Error; err != nil {
			return err
		}
		for index, sourceComment := range post.Comments {
			commentTime := createdAt.Add(time.Duration(index+1) * time.Second)
			comment := projectdb.Comment{
				Content:   strings.TrimSpace(sourceComment.Content),
				ReplyTo:   strings.TrimSpace(sourceComment.RefNickname),
				Username:  strings.TrimSpace(sourceComment.Nickname),
				Author:    "wechat:" + strings.TrimSpace(sourceComment.Nickname),
				CreatedAt: &commentTime,
				UpdatedAt: &commentTime,
				MemoId:    memo.Id,
			}
			if err := tx.Create(&comment).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		event.Status = "failed"
		event.Error = fmt.Sprintf("写入数据库失败: %v", err)
		return event
	}

	event.Status = "imported"
	event.MemoID = memo.Id
	event.Images = len(imageURLs)
	event.Comments = len(post.Comments)
	event.Warnings = warnings
	return event
}

func hasImportedPost(gdb *gorm.DB, sourceID string) (bool, error) {
	var count int64
	err := gdb.Model(&projectdb.Memo{}).
		Where("ext LIKE ?", "%\"wechatMomentId\":\""+sourceID+"\"%").
		Count(&count).Error
	return count > 0, err
}

func safeMediaPath(mediaDir, sourcePath string) (string, error) {
	clean := filepath.Clean(sourcePath)
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("路径超出媒体目录")
	}
	fullPath := filepath.Join(mediaDir, clean)
	relative, err := filepath.Rel(mediaDir, fullPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("路径超出媒体目录")
	}
	return fullPath, nil
}

func importObjectKey(createdAt time.Time, sourceID string, index int, localPath string, storage *uploader) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(localPath))
	if ext == "" || len(ext) > 12 {
		return "", "", errors.New("无法识别安全的文件扩展名")
	}
	key := fmt.Sprintf("imports/wechat/%s/%s_%02d%s", createdAt.Format("2006/01/02"), sourceID, index, ext)
	return key, joinResourceURL(storage.domain, key), nil
}

func (u *uploader) upload(ctx context.Context, key, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(localPath))
	if contentType == "" {
		buffer := make([]byte, 512)
		read, readErr := io.ReadFull(file, buffer)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return readErr
		}
		contentType = "application/octet-stream"
		if read > 0 {
			contentType = http.DetectContentType(buffer[:read])
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	return err
}

func formatLocation(location sourceLocation) string {
	parts := make([]string, 0, 3)
	for _, value := range []string{location.City, location.POIAddressName, location.POIName} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		duplicate := false
		for _, existing := range parts {
			if existing == value {
				duplicate = true
				break
			}
		}
		if !duplicate {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " · ")
}

func joinResourceURL(domain, objectKey string) string {
	return strings.TrimRight(domain, "/") + "/" + strings.TrimLeft(objectKey, "/")
}

func abbreviatedURL(value string) string {
	if len(value) <= 120 {
		return value
	}
	return value[:117] + "..."
}

func countUnsupported(warnings []string) int {
	count := 0
	for _, warning := range warnings {
		if strings.Contains(warning, "仅有远程链接") {
			count++
		}
	}
	return count
}

func writeEvent(file *os.File, event reportEvent) {
	data, err := json.Marshal(event)
	if err == nil {
		_, _ = file.Write(append(data, '\n'))
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
