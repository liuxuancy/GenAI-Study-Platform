package course

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/liuxuancy/Libs/logs"
	"gorm.io/gorm"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/cos"
	"caveclass/internal/platform/errs"
)

// CourseResourceItem API 列表项（不含 object_key）。
type CourseResourceItem struct {
	ID            int64     `json:"id"`
	CourseID      int64     `json:"courseId"`
	DisplayName   string    `json:"displayName"`
	OriginalName  string    `json:"originalName"`
	FileExtension string    `json:"fileExtension"`
	MimeType      string    `json:"mimeType"`
	FileSize      int64     `json:"fileSize"`
	ScopeType     string    `json:"scopeType"`
	ChapterKey    string    `json:"chapterKey"`
	SectionKey    string    `json:"sectionKey"`
	PublishStatus string    `json:"publishStatus"`
	PreviewStatus string    `json:"previewStatus"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CourseResourceListResult struct {
	Items []CourseResourceItem `json:"items"`
	Total int                  `json:"total"`
}

type UploadCourseResourceInput struct {
	ScopeType   string
	ChapterKey  string
	SectionKey  string
	DisplayName string
}

type PatchCourseResourceInput struct {
	DisplayName *string `json:"displayName"`
	ScopeType   *string `json:"scopeType"`
	ChapterKey  *string `json:"chapterKey"`
	SectionKey  *string `json:"sectionKey"`
}

type PublishCourseResourceInput struct {
	Published bool `json:"published"`
}

type ResourcePreviewResult struct {
	Kind    string `json:"kind"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

type ResourceDownloadResult struct {
	URL       string `json:"url"`
	FileName  string `json:"fileName"`
	ExpiresIn int64  `json:"expiresInSec"`
}

func (service *Service) requireCOS() (*cos.Client, cos.Limits, error) {
	if service.cosClient == nil {
		return nil, cos.Limits{}, errs.BadRequest("object storage is not configured")
	}
	limits := service.cosLimits
	if limits.DocMaxBytes == 0 {
		limits = cos.LoadLimits()
	}
	return service.cosClient, limits, nil
}

func (service *Service) validateResourceScopeKeys(ctx context.Context, courseID int64, scopeType, chapterKey, sectionKey string) error {
	switch scopeType {
	case ResourceScopeCourse:
		return nil
	case ResourceScopeChapter:
		var count int64
		if err := service.database.WithContext(ctx).Raw(
			`SELECT COUNT(*) FROM course_chapter WHERE course_id = ? AND chapter_key = ?`,
			courseID, chapterKey,
		).Scan(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			var contentJSON string
			if err := service.database.WithContext(ctx).Raw(
				`SELECT COALESCE(content_json, '') FROM course WHERE id = ?`, courseID,
			).Scan(&contentJSON).Error; err != nil {
				return err
			}
			if !chapterKeyInContent(contentJSON, chapterKey) {
				return errs.BadRequest("chapterKey not found in course")
			}
		}
		return nil
	case ResourceScopeSection:
		var count int64
		if err := service.database.WithContext(ctx).Raw(
			`SELECT COUNT(*) FROM course_section cs
			 JOIN course_chapter cc ON cc.id = cs.chapter_id
			 WHERE cc.course_id = ? AND cc.chapter_key = ? AND cs.section_key = ?`,
			courseID, chapterKey, sectionKey,
		).Scan(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			var contentJSON string
			if err := service.database.WithContext(ctx).Raw(
				`SELECT COALESCE(content_json, '') FROM course WHERE id = ?`, courseID,
			).Scan(&contentJSON).Error; err != nil {
				return err
			}
			if !sectionKeyInContent(contentJSON, chapterKey, sectionKey) {
				return errs.BadRequest("sectionKey not found in course")
			}
		}
		return nil
	default:
		return errs.BadRequest("invalid scopeType")
	}
}

func chapterKeyInContent(contentJSON, chapterKey string) bool {
	var parsed aicourse.StandardCourse
	if json.Unmarshal([]byte(contentJSON), &parsed) != nil {
		return false
	}
	for _, ch := range parsed.Chapters {
		if strings.TrimSpace(ch.Key) == chapterKey {
			return true
		}
	}
	return false
}

func sectionKeyInContent(contentJSON, chapterKey, sectionKey string) bool {
	var parsed aicourse.StandardCourse
	if json.Unmarshal([]byte(contentJSON), &parsed) != nil {
		return false
	}
	for _, ch := range parsed.Chapters {
		if strings.TrimSpace(ch.Key) != chapterKey {
			continue
		}
		for _, act := range ch.Activities {
			if strings.TrimSpace(act.Key) == sectionKey {
				return true
			}
		}
	}
	return false
}

func (service *Service) loadCourseResource(ctx context.Context, courseID, resourceID int64) (*CourseResource, error) {
	var row CourseResource
	err := service.database.WithContext(ctx).
		Where("id = ? AND course_id = ?", resourceID, courseID).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.NotFound("resource not found")
		}
		return nil, err
	}
	return &row, nil
}

func (service *Service) canAccessResource(ctx context.Context, principal *access.Principal, row *CourseResource, requirePublished bool) error {
	if row.CourseID == 0 {
		return errs.NotFound("resource not found")
	}
	switch principal.Role {
	case access.RoleTeacher:
		return service.requireTeacherCourse(ctx, principal, row.CourseID)
	case access.RoleStudent:
		if err := service.requireStudentCourse(ctx, principal, row.CourseID); err != nil {
			return err
		}
		if requirePublished && row.PublishStatus != PublishStatusPublished {
			return errs.NotFound("resource not found")
		}
		return nil
	default:
		return errs.Forbidden("access denied")
	}
}

func rowToItem(row CourseResource) CourseResourceItem {
	return CourseResourceItem{
		ID:            row.ID,
		CourseID:      row.CourseID,
		DisplayName:   row.DisplayName,
		OriginalName:  row.OriginalName,
		FileExtension: row.FileExtension,
		MimeType:      row.MimeType,
		FileSize:      row.FileSize,
		ScopeType:     row.ScopeType,
		ChapterKey:    row.ChapterKey,
		SectionKey:    row.SectionKey,
		PublishStatus: row.PublishStatus,
		PreviewStatus: row.PreviewStatus,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

// ListTeacherCourseResources GET /api/teacher/courses/:id/resources
func (service *Service) ListTeacherCourseResources(ctx context.Context, principal *access.Principal, courseID int64, publishFilter, extFilter, chapterKey, search string) (*CourseResourceListResult, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[ListTeacherCourseResources] failed, err: %v", err)
		return nil, err
	}
	query := service.database.WithContext(ctx).Model(&CourseResource{}).Where("course_id = ?", courseID)
	publishFilter = strings.ToLower(strings.TrimSpace(publishFilter))
	switch publishFilter {
	case "published":
		query = query.Where("publish_status = ?", PublishStatusPublished)
	case "draft":
		query = query.Where("publish_status = ?", PublishStatusDraft)
	}
	if ext := strings.TrimSpace(strings.ToLower(extFilter)); ext != "" && ext != "all" {
		query = query.Where("file_extension = ?", ext)
	}
	if ck := strings.TrimSpace(chapterKey); ck != "" && ck != "all" {
		query = query.Where("(scope_type = ? OR chapter_key = ? OR (scope_type = ? AND chapter_key = ?))",
			ResourceScopeCourse, ck, ResourceScopeChapter, ck)
	}
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + s + "%"
		query = query.Where("display_name LIKE ? OR original_name LIKE ?", like, like)
	}
	var rows []CourseResource
	if err := query.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		logs.CtxError(ctx, "[ListTeacherCourseResources] failed, err: %v", err)
		return nil, err
	}
	items := make([]CourseResourceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToItem(row))
	}
	output := &CourseResourceListResult{Items: items, Total: len(items)}
	logs.CtxInfo(ctx, "[ListTeacherCourseResources] done: course_id=%d total=%d", courseID, output.Total)
	return output, nil
}

// ListStudentCourseResources GET /api/student/courses/:id/resources
func (service *Service) ListStudentCourseResources(ctx context.Context, principal *access.Principal, courseID int64, extFilter, chapterKey, search string) (*CourseResourceListResult, error) {
	if err := service.requireStudentCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[ListStudentCourseResources] failed, err: %v", err)
		return nil, err
	}
	query := service.database.WithContext(ctx).Model(&CourseResource{}).
		Where("course_id = ? AND publish_status = ?", courseID, PublishStatusPublished)
	if ext := strings.TrimSpace(strings.ToLower(extFilter)); ext != "" && ext != "all" {
		query = query.Where("file_extension = ?", ext)
	}
	if ck := strings.TrimSpace(chapterKey); ck != "" && ck != "all" {
		query = query.Where("(scope_type = ? OR chapter_key = ? OR (scope_type = ? AND chapter_key = ?))",
			ResourceScopeCourse, ck, ResourceScopeChapter, ck)
	}
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + s + "%"
		query = query.Where("display_name LIKE ? OR original_name LIKE ?", like, like)
	}
	var rows []CourseResource
	if err := query.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		logs.CtxError(ctx, "[ListStudentCourseResources] failed, err: %v", err)
		return nil, err
	}
	items := make([]CourseResourceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToItem(row))
	}
	output := &CourseResourceListResult{Items: items, Total: len(items)}
	logs.CtxInfo(ctx, "[ListStudentCourseResources] done: course_id=%d total=%d", courseID, output.Total)
	return output, nil
}

// UploadTeacherCourseResource POST multipart
func (service *Service) UploadTeacherCourseResource(ctx context.Context, principal *access.Principal, courseID int64, filename string, size int64, reader io.Reader, input UploadCourseResourceInput) (*CourseResourceItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	return service.uploadCourseResource(ctx, principal, courseID, filename, size, reader, input, PublishStatusDraft)
}

func (service *Service) UploadStudentCourseResource(ctx context.Context, principal *access.Principal, courseID int64, filename string, size int64, reader io.Reader, input UploadCourseResourceInput) (*CourseResourceItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can upload student course resources")
	}
	var count int64
	if err := service.database.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM course WHERE id = ? AND owner_id = ?`, courseID, principal.UserID,
	).Scan(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errs.NotFound("course not found")
	}
	return service.uploadCourseResource(ctx, principal, courseID, filename, size, reader, input, PublishStatusPublished)
}

func (service *Service) uploadCourseResource(ctx context.Context, principal *access.Principal, courseID int64, filename string, size int64, reader io.Reader, input UploadCourseResourceInput, publishStatus string) (*CourseResourceItem, error) {
	client, limits, err := service.requireCOS()
	if err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	head := make([]byte, 512)
	n, readErr := io.ReadFull(reader, head)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return nil, errs.BadRequest("failed to read upload")
	}
	head = head[:n]
	stream := io.MultiReader(bytes.NewReader(head), reader)

	ext, mimeType, err := validateResourceFile(filename, size, head, limits)
	if err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	scopeType, chapterKey, sectionKey, err := normalizeResourceScope(input.ScopeType, input.ChapterKey, input.SectionKey)
	if err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	if err := service.validateResourceScopeKeys(ctx, courseID, scopeType, chapterKey, sectionKey); err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	objectKey, err := cos.OriginalObjectKey(courseID, filename)
	if err != nil {
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, errs.BadRequest(err.Error())
	}
	if err := client.PutReader(ctx, objectKey, mimeType, stream, size); err != nil {
		return nil, errs.Internal(errors.New("upload to object storage failed"))
	}
	storedSize, exists, err := client.Head(ctx, objectKey)
	if err != nil || !exists {
		_ = client.Delete(ctx, objectKey)
		return nil, errs.Internal(errors.New("upload verification failed"))
	}
	if storedSize > 0 && storedSize != size {
		_ = client.Delete(ctx, objectKey)
		return nil, errs.Internal(errors.New("upload size mismatch"))
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(filename)
	}
	now := time.Now().UTC()
	row := CourseResource{
		CourseID:         courseID,
		ChapterKey:       chapterKey,
		SectionKey:       sectionKey,
		UploaderID:       principal.UserID,
		OriginalName:     strings.TrimSpace(filename),
		DisplayName:      displayName,
		ObjectKey:        objectKey,
		FileExtension:    ext,
		MimeType:         mimeType,
		FileSize:         size,
		StorageProvider:  ResourceStorageTencentCOS,
		ScopeType:        scopeType,
		PublishStatus:    publishStatus,
		PreviewStatus:    initialPreviewStatus(ext),
		PreviewObjectKey: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := service.database.WithContext(ctx).Create(&row).Error; err != nil {
		_ = client.Delete(ctx, objectKey)
		logs.CtxError(ctx, "[UploadTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	item := rowToItem(row)
	logs.CtxInfo(ctx, "[UploadTeacherCourseResource] done: resp=%+v", item)
	return &item, nil
}

// PatchTeacherCourseResource PATCH metadata
func (service *Service) PatchTeacherCourseResource(ctx context.Context, principal *access.Principal, courseID, resourceID int64, input PatchCourseResourceInput) (*CourseResourceItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[PatchTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	row, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[PatchTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	scopeType := row.ScopeType
	chapterKey := row.ChapterKey
	sectionKey := row.SectionKey
	if input.ScopeType != nil {
		var normErr error
		scopeType, chapterKey, sectionKey, normErr = normalizeResourceScope(*input.ScopeType, coalesceStr(input.ChapterKey, row.ChapterKey), coalesceStr(input.SectionKey, row.SectionKey))
		if normErr != nil {
			return nil, normErr
		}
	} else {
		if input.ChapterKey != nil {
			chapterKey = strings.TrimSpace(*input.ChapterKey)
		}
		if input.SectionKey != nil {
			sectionKey = strings.TrimSpace(*input.SectionKey)
		}
	}
	if err := service.validateResourceScopeKeys(ctx, courseID, scopeType, chapterKey, sectionKey); err != nil {
		logs.CtxError(ctx, "[PatchTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if input.DisplayName != nil {
		name := strings.TrimSpace(*input.DisplayName)
		if name == "" {
			return nil, errs.BadRequest("displayName is required")
		}
		updates["display_name"] = name
	}
	if input.ScopeType != nil || input.ChapterKey != nil || input.SectionKey != nil {
		updates["scope_type"] = scopeType
		updates["chapter_key"] = chapterKey
		updates["section_key"] = sectionKey
	}
	if err := service.database.WithContext(ctx).Model(row).Updates(updates).Error; err != nil {
		logs.CtxError(ctx, "[PatchTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	updated, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[PatchTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	item := rowToItem(*updated)
	logs.CtxInfo(ctx, "[PatchTeacherCourseResource] done: resp=%+v", item)
	return &item, nil
}

func coalesceStr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// PublishTeacherCourseResource PATCH publish
func (service *Service) PublishTeacherCourseResource(ctx context.Context, principal *access.Principal, courseID, resourceID int64, input PublishCourseResourceInput) (*CourseResourceItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[PublishTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	row, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[PublishTeacherCourseResource] failed, err: %v", err)
		return nil, err
	}
	status := PublishStatusDraft
	if input.Published {
		status = PublishStatusPublished
	}
	if err := service.database.WithContext(ctx).Model(row).Updates(map[string]any{
		"publish_status": status,
		"updated_at":     time.Now().UTC(),
	}).Error; err != nil {
		return nil, err
	}
	row.PublishStatus = status
	item := rowToItem(*row)
	logs.CtxInfo(ctx, "[PublishTeacherCourseResource] done: resp=%+v", item)
	return &item, nil
}

// DeleteTeacherCourseResource soft delete + COS delete
func (service *Service) DeleteTeacherCourseResource(ctx context.Context, principal *access.Principal, courseID, resourceID int64) error {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[DeleteTeacherCourseResource] failed, err: %v", err)
		return err
	}
	row, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[DeleteTeacherCourseResource] failed, err: %v", err)
		return err
	}
	if err := service.database.WithContext(ctx).Delete(row).Error; err != nil {
		logs.CtxError(ctx, "[DeleteTeacherCourseResource] failed, err: %v", err)
		return err
	}
	if service.cosClient != nil {
		_ = service.cosClient.Delete(ctx, row.ObjectKey)
	}
	logs.CtxInfo(ctx, "[DeleteTeacherCourseResource] done: course_id=%d resource_id=%d", courseID, resourceID)
	return nil
}

// GetCourseResourcePreview shared preview
func (service *Service) GetCourseResourcePreview(ctx context.Context, principal *access.Principal, courseID, resourceID int64) (*ResourcePreviewResult, error) {
	row, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
		return nil, err
	}
	requirePublished := principal.Role == access.RoleStudent
	if err := service.canAccessResource(ctx, principal, row, requirePublished); err != nil {
		logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
		return nil, err
	}
	kind := previewKindForResource(row)
	limits := service.cosLimits
	if limits.DocMaxBytes == 0 {
		limits = cos.LoadLimits()
	}
	switch kind {
	case "pdf":
		client, _, err := service.requireCOS()
		if err != nil {
			logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
			return nil, err
		}
		url, err := client.PresignedPDFInlineGetURL(ctx, row.ObjectKey, row.DisplayName, limits.PresignTTL)
		if err != nil {
			logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
			return nil, errs.Internal(fmt.Errorf("presign pdf preview: %w", err))
		}
		output := &ResourcePreviewResult{Kind: kind, URL: url}
		logs.CtxInfo(ctx, "[GetCourseResourcePreview] done: resource_id=%d resp=%+v", resourceID, output)
		return output, nil
	case "image":
		client, _, err := service.requireCOS()
		if err != nil {
			logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
			return nil, err
		}
		url, err := client.PresignedInlineGetURL(ctx, row.ObjectKey, row.DisplayName, limits.PresignTTL)
		if err != nil {
			logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
			return nil, errs.Internal(fmt.Errorf("presign preview: %w", err))
		}
		output := &ResourcePreviewResult{Kind: kind, URL: url}
		logs.CtxInfo(ctx, "[GetCourseResourcePreview] done: resource_id=%d resp=%+v", resourceID, output)
		return output, nil
	case "office":
		if limits.DocPreviewEnabled {
			client, _, err := service.requireCOS()
			if err != nil {
				logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
				return nil, err
			}
			url, err := client.PresignedDocPreviewURL(ctx, row.ObjectKey, limits.PresignTTL)
			if err != nil {
				logs.CtxError(ctx, "[GetCourseResourcePreview] failed, err: %v", err)
				return nil, errs.Internal(fmt.Errorf("presign doc preview: %w", err))
			}
			output := &ResourcePreviewResult{Kind: "iframe", URL: url}
			logs.CtxInfo(ctx, "[GetCourseResourcePreview] done: resource_id=%d resp=%+v", resourceID, output)
			return output, nil
		}
		output := &ResourcePreviewResult{
			Kind:    "unsupported",
			Message: "当前文件暂不支持在线预览，请下载查看",
		}
		logs.CtxInfo(ctx, "[GetCourseResourcePreview] done: resource_id=%d resp=%+v", resourceID, output)
		return output, nil
	default:
		output := &ResourcePreviewResult{
			Kind:    "unsupported",
			Message: "当前文件暂不支持在线预览，请下载查看",
		}
		logs.CtxInfo(ctx, "[GetCourseResourcePreview] done: resource_id=%d resp=%+v", resourceID, output)
		return output, nil
	}
}

// GetCourseResourceDownload shared download URL
func (service *Service) GetCourseResourceDownload(ctx context.Context, principal *access.Principal, courseID, resourceID int64) (*ResourceDownloadResult, error) {
	row, err := service.loadCourseResource(ctx, courseID, resourceID)
	if err != nil {
		logs.CtxError(ctx, "[GetCourseResourceDownload] failed, err: %v", err)
		return nil, err
	}
	requirePublished := principal.Role == access.RoleStudent
	if err := service.canAccessResource(ctx, principal, row, requirePublished); err != nil {
		logs.CtxError(ctx, "[GetCourseResourceDownload] failed, err: %v", err)
		return nil, err
	}
	client, limits, err := service.requireCOS()
	if err != nil {
		logs.CtxError(ctx, "[GetCourseResourceDownload] failed, err: %v", err)
		return nil, err
	}
	url, err := client.PresignedGetURL(ctx, row.ObjectKey, row.DisplayName, limits.PresignTTL)
	if err != nil {
		logs.CtxError(ctx, "[GetCourseResourceDownload] failed, err: %v", err)
		return nil, errs.Internal(fmt.Errorf("presign download: %w", err))
	}
	ttlSec := int64(limits.PresignTTL.Seconds())
	if ttlSec <= 0 {
		ttlSec = 900
	}
	output := &ResourceDownloadResult{
		URL:       url,
		FileName:  row.DisplayName,
		ExpiresIn: ttlSec,
	}
	logs.CtxInfo(ctx, "[GetCourseResourceDownload] done: resource_id=%d resp=%+v", resourceID, output)
	return output, nil
}
