// course 的 HTTP handler：取当前用户并调用 service。路由登记在 internal/router/routes.go。

package course

import (
	"io"

	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/resp"
)

type Handler struct {
	service *Service
}

// NewHandler 全局只初始化一次：启动时在路由表里创建。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetCourseList GET /api/student/courses
func (handler *Handler) GetCourseList(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.GetCourseList(ctx.Request.Context(), principal)
}

// GetQuestionList GET /api/student/questions
func (handler *Handler) GetQuestionList(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.GetQuestionList(ctx.Request.Context(), principal)
}

// GetCourse GET /api/student/courses/:id
func (handler *Handler) GetCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetCourse(ctx.Request.Context(), principal, courseID)
}

// UpdateCourseProgress PUT /api/student/courses/:id/progress
func (handler *Handler) UpdateCourseProgress(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input CourseProgress
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.UpdateCourseProgress(ctx.Request.Context(), principal, courseID, input.Percent)
}

// GetQuestion GET /api/student/questions/:id
func (handler *Handler) GetQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	questionID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetQuestion(ctx.Request.Context(), principal, questionID)
}

// AnswerQuestion PUT /api/student/questions/:id/contents/:contentId
func (handler *Handler) AnswerQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	questionID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	contentID, err := resp.GetInt64PathParam(ctx, "contentId")
	if err != nil {
		return nil, err
	}
	var input AnswerInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.AnswerQuestion(ctx.Request.Context(), principal, questionID, contentID, input.Option, input.Submit)
}

// GenerateStudentCourse POST /api/student/courses/:id/generate
func (handler *Handler) GenerateStudentCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can generate courses")
	}
	return handler.generateCourse(ctx, principal)
}

// GenerateTeacherCourse POST /api/teacher/courses/:id/generate
func (handler *Handler) GenerateTeacherCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can generate courses")
	}
	return handler.generateCourse(ctx, principal)
}

func (handler *Handler) generateCourse(ctx *gin.Context, principal *access.Principal) (any, error) {
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input GenerateCourseInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateOwnedCourse(ctx.Request.Context(), principal, courseID, input)
}

// CreateStudentCourse POST /api/student/courses
func (handler *Handler) CreateStudentCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input CreateCourseInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.CreateStudentCourse(ctx.Request.Context(), principal, input)
}

// SetCourseSharing PUT /api/student/courses/:id/sharing
func (handler *Handler) SetCourseSharing(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input SharingInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	if input.Shared == nil {
		return nil, errs.BadRequest("shared is required")
	}
	return handler.service.SetCourseSharing(ctx.Request.Context(), principal, courseID, *input.Shared)
}

// SaveStudentCourseContent PUT /api/student/courses/:id/content
func (handler *Handler) SaveStudentCourseContent(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can save course content")
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, errs.BadRequest("invalid request")
	}
	if err := handler.service.SaveCourseContent(ctx.Request.Context(), principal, courseID, string(body)); err != nil {
		return nil, err
	}
	return handler.service.GetCourse(ctx.Request.Context(), principal, courseID)
}

// GetTeacherClasses GET /api/teacher/classes
func (handler *Handler) GetTeacherClasses(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.GetTeacherClasses(ctx.Request.Context(), principal)
}

// GetTeacherCourseList GET /api/teacher/courses
func (handler *Handler) GetTeacherCourseList(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.GetTeacherCourseList(ctx.Request.Context(), principal)
}

// CreateTeacherCourse POST /api/teacher/courses
func (handler *Handler) CreateTeacherCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input CreateCourseInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.CreateTeacherCourse(ctx.Request.Context(), principal, input)
}

// GetTeacherCourse GET /api/teacher/courses/:id
func (handler *Handler) GetTeacherCourse(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetTeacherCourse(ctx.Request.Context(), principal, courseID)
}

// PublishCourseToClasses PUT /api/teacher/courses/:id/classes
func (handler *Handler) PublishCourseToClasses(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input PublishClassesInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	if input.ClassIDs == nil {
		return nil, errs.BadRequest("classIds is required")
	}
	return handler.service.PublishCourseToClasses(ctx.Request.Context(), principal, courseID, *input.ClassIDs)
}

// GetTeacherCourseProgress GET /api/teacher/courses/:id/progress
func (handler *Handler) GetTeacherCourseProgress(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetTeacherCourseProgress(ctx.Request.Context(), principal, courseID)
}

// SaveTeacherCourseContent PUT /api/teacher/courses/:id/content
func (handler *Handler) SaveTeacherCourseContent(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can save course content")
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, errs.BadRequest("invalid request")
	}
	if err := handler.service.SaveCourseContent(ctx.Request.Context(), principal, courseID, string(body)); err != nil {
		return nil, err
	}
	return handler.service.GetTeacherCourse(ctx.Request.Context(), principal, courseID)
}

// GetClassList GET /api/system/classes
func (handler *Handler) GetClassList(ctx *gin.Context) (any, error) {
	return handler.service.GetClassList(ctx.Request.Context())
}

// CreateClass POST /api/system/classes
func (handler *Handler) CreateClass(ctx *gin.Context) (any, error) {
	var input RenameClassInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.CreateClass(ctx.Request.Context(), input.Name)
}

// RenameClass PUT /api/system/classes/:id
func (handler *Handler) RenameClass(ctx *gin.Context) (any, error) {
	classID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input RenameClassInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.RenameClass(ctx.Request.Context(), classID, input.Name)
}

// SetClassMembers PUT /api/system/classes/:id/members
func (handler *Handler) SetClassMembers(ctx *gin.Context) (any, error) {
	classID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input ClassMembersInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	if input.UserIDs == nil {
		return nil, errs.BadRequest("userIds is required")
	}
	return handler.service.SetClassMembers(ctx.Request.Context(), classID, *input.UserIDs)
}

// CreateQuestion POST /api/student/questions
func (handler *Handler) CreateQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input CreateQuestionInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.CreateQuestion(ctx.Request.Context(), principal, input)
}
