package course

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/resp"
)

// ListTeacherCourseResources GET /api/teacher/courses/:id/resources
func (handler *Handler) ListTeacherCourseResources(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.ListTeacherCourseResources(ctx.Request.Context(), principal, courseID,
		ctx.Query("publish"), ctx.Query("type"), ctx.Query("chapterKey"), ctx.Query("search"))
}

// UploadTeacherCourseResource POST /api/teacher/courses/:id/resources
func (handler *Handler) UploadTeacherCourseResource(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return nil, errs.BadRequest("file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, errs.BadRequest("failed to read upload")
	}
	defer file.Close()
	input := UploadCourseResourceInput{
		ScopeType:   ctx.PostForm("scopeType"),
		ChapterKey:  ctx.PostForm("chapterKey"),
		SectionKey:  ctx.PostForm("sectionKey"),
		DisplayName: ctx.PostForm("displayName"),
	}
	return handler.service.UploadTeacherCourseResource(ctx.Request.Context(), principal, courseID,
		fileHeader.Filename, fileHeader.Size, file, input)
}

// PatchTeacherCourseResource PATCH /api/teacher/courses/:id/resources/:resourceId
func (handler *Handler) PatchTeacherCourseResource(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	resourceID, err := resp.GetInt64PathParam(ctx, "resourceId")
	if err != nil {
		return nil, err
	}
	var body PatchCourseResourceInput
	if err := resp.BindJSON(ctx, &body); err != nil {
		return nil, err
	}
	return handler.service.PatchTeacherCourseResource(ctx.Request.Context(), principal, courseID, resourceID, body)
}

// PublishTeacherCourseResource PATCH /api/teacher/courses/:id/resources/:resourceId/publish
func (handler *Handler) PublishTeacherCourseResource(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	resourceID, err := resp.GetInt64PathParam(ctx, "resourceId")
	if err != nil {
		return nil, err
	}
	var body PublishCourseResourceInput
	if err := resp.BindJSON(ctx, &body); err != nil {
		return nil, err
	}
	return handler.service.PublishTeacherCourseResource(ctx.Request.Context(), principal, courseID, resourceID, body)
}

// DeleteTeacherCourseResource DELETE /api/teacher/courses/:id/resources/:resourceId
func (handler *Handler) DeleteTeacherCourseResource(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	resourceID, err := resp.GetInt64PathParam(ctx, "resourceId")
	if err != nil {
		return nil, err
	}
	if err := handler.service.DeleteTeacherCourseResource(ctx.Request.Context(), principal, courseID, resourceID); err != nil {
		return nil, err
	}
	return gin.H{"ok": true}, nil
}

// ListStudentCourseResources GET /api/student/courses/:id/resources
func (handler *Handler) ListStudentCourseResources(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.ListStudentCourseResources(ctx.Request.Context(), principal, courseID,
		ctx.Query("type"), ctx.Query("chapterKey"), ctx.Query("search"))
}

// GetCourseResourcePreview GET /api/courses/:courseId/resources/:resourceId/preview
func (handler *Handler) GetCourseResourcePreview(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := courseResourcePathCourseID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := resp.GetInt64PathParam(ctx, "resourceId")
	if err != nil {
		return nil, err
	}
	return handler.service.GetCourseResourcePreview(ctx.Request.Context(), principal, courseID, resourceID)
}

// GetCourseResourceDownload GET /api/courses/:courseId/resources/:resourceId/download
func (handler *Handler) GetCourseResourceDownload(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := courseResourcePathCourseID(ctx)
	if err != nil {
		return nil, err
	}
	resourceID, err := resp.GetInt64PathParam(ctx, "resourceId")
	if err != nil {
		return nil, err
	}
	result, err := handler.service.GetCourseResourceDownload(ctx.Request.Context(), principal, courseID, resourceID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(ctx.Query("redirect"), "1") || strings.EqualFold(ctx.Query("redirect"), "true") {
		ctx.Redirect(302, result.URL)
		return nil, nil
	}
	return result, nil
}

func courseResourcePathCourseID(ctx *gin.Context) (int64, error) {
	if raw := strings.TrimSpace(ctx.Param("courseId")); raw != "" {
		return strconv.ParseInt(raw, 10, 64)
	}
	return resp.GetInt64PathParam(ctx, "id")
}
