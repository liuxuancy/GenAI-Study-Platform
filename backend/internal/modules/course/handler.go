// course 的 HTTP handler：取当前用户并调用 service。路由登记在 internal/router/routes.go。

package course

import (
	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
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
