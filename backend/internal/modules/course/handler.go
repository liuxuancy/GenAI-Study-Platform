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

// GetAIProviders GET /api/ai/providers
func (handler *Handler) GetAIProviders(ctx *gin.Context) (any, error) {
	if _, err := access.GetCurrentPrincipal(ctx); err != nil {
		return nil, err
	}
	return handler.service.AIProviderOptions(), nil
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

// GenerateStudentChapter POST /api/student/courses/:id/chapters/generate
func (handler *Handler) GenerateStudentChapter(ctx *gin.Context) (any, error) {
	return handler.generateChapter(ctx, access.RoleStudent)
}

// GenerateTeacherChapter POST /api/teacher/courses/:id/chapters/generate
func (handler *Handler) GenerateTeacherChapter(ctx *gin.Context) (any, error) {
	return handler.generateChapter(ctx, access.RoleTeacher)
}

func (handler *Handler) generateChapter(ctx *gin.Context, role string) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != role {
		return nil, errs.Forbidden("role cannot generate this course")
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input GenerateChapterInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateOwnedChapter(ctx.Request.Context(), principal, courseID, input)
}

// GenerateStudentActivity POST /api/student/courses/:id/activities/generate
func (handler *Handler) GenerateStudentActivity(ctx *gin.Context) (any, error) {
	return handler.generateActivity(ctx, access.RoleStudent)
}

// GenerateTeacherActivity POST /api/teacher/courses/:id/activities/generate
func (handler *Handler) GenerateTeacherActivity(ctx *gin.Context) (any, error) {
	return handler.generateActivity(ctx, access.RoleTeacher)
}

func (handler *Handler) generateActivity(ctx *gin.Context, role string) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if principal.Role != role {
		return nil, errs.Forbidden("role cannot generate this course")
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input GenerateActivityInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateOwnedActivity(ctx.Request.Context(), principal, courseID, input)
}

// AskChapterQuestion POST /api/student/courses/:id/chapters/:chapterKey/ask
func (handler *Handler) AskChapterQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input ChapterQAInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.AskChapterQuestion(ctx.Request.Context(), principal, courseID, ctx.Param("chapterKey"), input)
}

// GenerateGalScript POST /api/student/courses/:id/chapters/:chapterKey/gal-script
func (handler *Handler) GenerateGalScript(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input GalScriptInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateGalScript(ctx.Request.Context(), principal, courseID, ctx.Param("chapterKey"), input)
}

// ListAIConversations GET /api/student/courses/:id/conversations
func (handler *Handler) ListAIConversations(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.ListAIConversations(ctx.Request.Context(), principal, courseID)
}

// ListOwnedAIConversations GET /api/student/conversations
func (handler *Handler) ListOwnedAIConversations(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.ListOwnedAIConversations(ctx.Request.Context(), principal)
}

func (handler *Handler) GetOwnedAIConversation(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	conversationID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetOwnedAIConversation(ctx.Request.Context(), principal, conversationID)
}

// GetAIConversation GET /api/student/courses/:id/conversations/:conversationId
func (handler *Handler) GetAIConversation(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	conversationID, err := resp.GetInt64PathParam(ctx, "conversationId")
	if err != nil {
		return nil, err
	}
	return handler.service.GetAIConversation(ctx.Request.Context(), principal, courseID, conversationID)
}

// DeleteAIConversation DELETE /api/student/courses/:id/conversations/:conversationId
func (handler *Handler) DeleteAIConversation(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	conversationID, err := resp.GetInt64PathParam(ctx, "conversationId")
	if err != nil {
		return nil, err
	}
	if err := handler.service.DeleteAIConversation(ctx.Request.Context(), principal, courseID, conversationID); err != nil {
		return nil, err
	}
	return map[string]bool{"deleted": true}, nil
}

func (handler *Handler) RenameAIConversation(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	conversationID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input struct {
		Title string `json:"title"`
	}
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	if err := handler.service.RenameAIConversation(ctx.Request.Context(), principal, conversationID, input.Title); err != nil {
		return nil, err
	}
	return map[string]bool{"updated": true}, nil
}

func (handler *Handler) DeleteOwnedAIConversation(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	conversationID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	if err := handler.service.DeleteOwnedAIConversation(ctx.Request.Context(), principal, conversationID); err != nil {
		return nil, err
	}
	return map[string]bool{"deleted": true}, nil
}

func (handler *Handler) SendWorkspaceMessage(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input WorkspaceMessageInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.SendWorkspaceMessage(ctx.Request.Context(), principal, input)
}

func (handler *Handler) ListWorkspaceSkills(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.ListWorkspaceSkills(principal)
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

// GetTeacherCourseLearningOverview GET /api/teacher/courses/:id/learning-overview
func (handler *Handler) GetTeacherCourseLearningOverview(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetTeacherCourseLearningOverview(ctx.Request.Context(), principal, courseID)
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

// GetTeacherQuestions GET /api/teacher/courses/:id/questions
func (handler *Handler) GetTeacherQuestions(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.ListTeacherQuestions(ctx.Request.Context(), principal, courseID)
}

// CreateTeacherQuestion POST /api/teacher/courses/:id/questions
func (handler *Handler) CreateTeacherQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input TeacherQuestionInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.CreateTeacherQuestion(ctx.Request.Context(), principal, courseID, input)
}

// UpdateTeacherQuestion PUT /api/teacher/questions/:contentId
func (handler *Handler) UpdateTeacherQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	contentID, err := resp.GetInt64PathParam(ctx, "contentId")
	if err != nil {
		return nil, err
	}
	var input TeacherQuestionInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.UpdateTeacherQuestion(ctx.Request.Context(), principal, contentID, input)
}

// DeleteTeacherQuestion DELETE /api/teacher/questions/:contentId
func (handler *Handler) DeleteTeacherQuestion(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	contentID, err := resp.GetInt64PathParam(ctx, "contentId")
	if err != nil {
		return nil, err
	}
	if err := handler.service.DeleteTeacherQuestion(ctx.Request.Context(), principal, contentID); err != nil {
		return nil, err
	}
	return gin.H{"deleted": true}, nil
}

// PreviewTeacherQuestionImport POST /api/teacher/courses/:id/questions/import/preview
func (handler *Handler) PreviewTeacherQuestionImport(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input QuestionImportInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.PreviewQuestionImport(ctx.Request.Context(), principal, courseID, input)
}

// ConfirmTeacherQuestionImport POST /api/teacher/courses/:id/questions/import/confirm
func (handler *Handler) ConfirmTeacherQuestionImport(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input QuestionImportInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.ConfirmQuestionImport(ctx.Request.Context(), principal, courseID, input)
}

// GenerateTeacherQuestions POST /api/teacher/courses/:id/questions/generate
func (handler *Handler) GenerateTeacherQuestions(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input AIQuestionInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateTeacherQuestions(ctx.Request.Context(), principal, courseID, input)
}

func (handler *Handler) GetCourseQualityReport(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetCourseQualityReport(ctx.Request.Context(), principal, courseID)
}

func (handler *Handler) GenerateCourseQualityReport(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input QualityReportInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateCourseQualityReport(ctx.Request.Context(), principal, courseID, input)
}

func (handler *Handler) GetCourseMermaid(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.GetCourseMermaid(ctx.Request.Context(), principal, courseID)
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

// GetAdminCourseList GET /api/system/courses
func (handler *Handler) GetAdminCourseList(ctx *gin.Context) (any, error) {
	return handler.service.GetAdminCourseList(ctx.Request.Context())
}

// SetAdminCoursePublished PUT /api/system/courses/:id/publish
func (handler *Handler) SetAdminCoursePublished(ctx *gin.Context) (any, error) {
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input SetPublishInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.SetAdminCoursePublished(ctx.Request.Context(), courseID, *input.Published)
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

// ListCourseQuizzes GET /api/student/courses/:id/quizzes
func (handler *Handler) ListCourseQuizzes(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return handler.service.ListCourseQuizzes(ctx.Request.Context(), principal, courseID)
}

// GetQuiz GET /api/quizzes/:quizId
func (handler *Handler) GetQuiz(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	quizID, err := resp.GetInt64PathParam(ctx, "quizId")
	if err != nil {
		return nil, err
	}
	return handler.service.GetQuizByID(ctx.Request.Context(), principal, quizID)
}

// GetQuizAttempt GET /api/quizzes/:quizId/attempt
func (handler *Handler) GetQuizAttempt(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	quizID, err := resp.GetInt64PathParam(ctx, "quizId")
	if err != nil {
		return nil, err
	}
	return handler.service.GetQuizAttempt(ctx.Request.Context(), principal, quizID)
}

// StartQuizAttempt POST /api/quizzes/:quizId/attempts
func (handler *Handler) StartQuizAttempt(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	quizID, err := resp.GetInt64PathParam(ctx, "quizId")
	if err != nil {
		return nil, err
	}
	return handler.service.StartQuizAttempt(ctx.Request.Context(), principal, quizID)
}

// UpdateAttemptAnswer PUT /api/attempts/:attemptId/answers/:questionId
func (handler *Handler) UpdateAttemptAnswer(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	attemptID, err := resp.GetInt64PathParam(ctx, "attemptId")
	if err != nil {
		return nil, err
	}
	questionID, err := resp.GetInt64PathParam(ctx, "questionId")
	if err != nil {
		return nil, err
	}
	var input AttemptAnswerInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.UpdateAttemptAnswer(ctx.Request.Context(), principal, attemptID, questionID, input.Answer)
}

// SubmitQuizAttempt POST /api/attempts/:attemptId/submit
func (handler *Handler) SubmitQuizAttempt(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	attemptID, err := resp.GetInt64PathParam(ctx, "attemptId")
	if err != nil {
		return nil, err
	}
	return handler.service.SubmitQuizAttempt(ctx.Request.Context(), principal, attemptID)
}

// PublishQuiz POST /api/teacher/quizzes/:quizId/publish
func (handler *Handler) PublishQuiz(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	quizID, err := resp.GetInt64PathParam(ctx, "quizId")
	if err != nil {
		return nil, err
	}
	if err := handler.service.PublishQuiz(ctx.Request.Context(), principal, quizID); err != nil {
		return nil, err
	}
	return gin.H{"published": true}, nil
}

// ArchiveQuiz POST /api/teacher/quizzes/:quizId/archive
func (handler *Handler) ArchiveQuiz(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	quizID, err := resp.GetInt64PathParam(ctx, "quizId")
	if err != nil {
		return nil, err
	}
	if err := handler.service.ArchiveQuiz(ctx.Request.Context(), principal, quizID); err != nil {
		return nil, err
	}
	return gin.H{"archived": true}, nil
}

// GetLearningDashboard GET /api/student/learning/dashboard
func (handler *Handler) GetLearningDashboard(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	year, month := parseDashboardMonth(ctx.Query("year"), ctx.Query("month"))
	return handler.service.GetLearningDashboard(ctx.Request.Context(), principal, year, month)
}

// GenerateLearningInsight POST /api/student/learning/insights
func (handler *Handler) GenerateLearningInsight(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input LearningInsightInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.GenerateLearningInsight(ctx.Request.Context(), principal, input)
}

// RecordCourseHeartbeat POST /api/student/courses/:id/learning/heartbeat
func (handler *Handler) RecordCourseHeartbeat(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	courseID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input HeartbeatInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	if err := handler.service.RecordCourseHeartbeat(ctx.Request.Context(), principal, courseID, input.Seconds); err != nil {
		return nil, err
	}
	return gin.H{"recorded": true}, nil
}
