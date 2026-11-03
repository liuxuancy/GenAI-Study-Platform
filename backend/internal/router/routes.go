// 路由表：所有接口都在这里登记，按模块分组。新增接口时在对应模块的分组里加一行。

package router

import (
	"context"

	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/modules/account"
	"caveclass/internal/modules/course"
	"caveclass/internal/modules/system"
	"caveclass/internal/platform/cos"
	"caveclass/internal/platform/logs"
	"caveclass/internal/platform/resp"
)

// registerRoutes api 是 /api 分组，下面的路径都会自动带上 /api 前缀。
//
// 每一行的写法：方法、路径、所需权限（中间件）、handler。所需权限三选一：
//   - 不写：公开接口，必须同时登记在 identity/public.go 的白名单里
//   - access.RequireLogin()：只要求登录
//   - access.RequirePermission("使用方:资源:动作")：要求持有该权限码
func registerRoutes(api *gin.RouterGroup, deps Deps) {
	// 认证：白名单以外的请求都要先校验 token，必须在登记任何接口之前挂上
	api.Use(identity.Authenticate(deps.TokenIssuer, deps.Cache))

	// 框架
	api.GET("/health", resp.ToGinHandler(healthHandler(deps.DB))) // 公开

	// account 模块 /api/account/*：登录、退出、me、个人信息、改密码
	accountHandler := account.NewHandler(account.NewService(deps.DB, deps.TokenIssuer, deps.Cache))
	accountRoutes := api.Group("/account")
	accountRoutes.POST("/login", resp.ToGinHandler(accountHandler.Login)) // 公开
	accountRoutes.POST("/logout", access.RequireLogin(), resp.ToGinHandler(accountHandler.Logout))
	accountRoutes.GET("/me", access.RequireLogin(), resp.ToGinHandler(accountHandler.GetCurrentUser))
	accountRoutes.PUT("/profile", access.RequireLogin(), resp.ToGinHandler(accountHandler.UpdateProfile))
	accountRoutes.PUT("/password", access.RequireLogin(), resp.ToGinHandler(accountHandler.ChangePassword))

	// course 模块同时服务学生、教师和管理员班级。班级表在 course 里，避免 system 与 course 互相 import。
	courseService := course.NewService(deps.DB)
	courseService.UseAI(deps.AI)
	cosCfg := cos.LoadConfig()
	if cosCfg.IsConfigured() {
		cosClient, err := cos.NewClient(cosCfg)
		if err != nil {
			logs.CtxWarn(context.Background(), "cos client init failed: %v", err)
		} else {
			courseService.UseCOS(cosClient, cos.LoadLimits())
		}
	}
	courseHandler := course.NewHandler(courseService)

	api.GET("/ai/providers", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetAIProviders))

	// system 模块 /api/system/*：用户管理、角色权限、班级，只有管理员使用
	systemHandler := system.NewHandler(
		system.NewUserService(deps.DB, deps.Cache, deps.UserDeletionGuards),
		system.NewRoleService(deps.DB, deps.Cache, deps.Catalog),
		system.NewUsageService(deps.DB),
	)
	systemRoutes := api.Group("/system")
	systemRoutes.GET("/users", access.RequirePermission(system.PermissionUserView), resp.ToGinHandler(systemHandler.GetUserList))
	systemRoutes.POST("/users", access.RequirePermission(system.PermissionUserCreate), resp.ToGinHandler(systemHandler.CreateUser))
	systemRoutes.PUT("/users/:id", access.RequirePermission(system.PermissionUserUpdate), resp.ToGinHandler(systemHandler.UpdateUser))
	systemRoutes.PUT("/users/:id/password", access.RequirePermission(system.PermissionUserResetPassword), resp.ToGinHandler(systemHandler.ResetUserPassword))
	systemRoutes.DELETE("/users/:id", access.RequirePermission(system.PermissionUserDelete), resp.ToGinHandler(systemHandler.DeleteUser))
	systemRoutes.GET("/roles", access.RequirePermission(system.PermissionRoleView), resp.ToGinHandler(systemHandler.GetRoleList))
	systemRoutes.PUT("/roles/:code/permissions", access.RequirePermission(system.PermissionRoleAssign), resp.ToGinHandler(systemHandler.AssignRolePermissions))
	systemRoutes.GET("/permissions", access.RequirePermission(system.PermissionCatalogList), resp.ToGinHandler(systemHandler.GetPermissionCatalog))
	systemRoutes.GET("/classes", access.RequirePermission(system.PermissionClassView), resp.ToGinHandler(courseHandler.GetClassList))
	systemRoutes.POST("/classes", access.RequirePermission(system.PermissionClassCreate), resp.ToGinHandler(courseHandler.CreateClass))
	systemRoutes.PUT("/classes/:id", access.RequirePermission(system.PermissionClassUpdate), resp.ToGinHandler(courseHandler.RenameClass))
	systemRoutes.PUT("/classes/:id/members", access.RequirePermission(system.PermissionClassUpdate), resp.ToGinHandler(courseHandler.SetClassMembers))
	systemRoutes.GET("/courses", access.RequirePermission(system.PermissionCourseView), resp.ToGinHandler(courseHandler.GetAdminCourseList))
	systemRoutes.PUT("/courses/:id/publish", access.RequirePermission(system.PermissionCourseUpdate), resp.ToGinHandler(courseHandler.SetAdminCoursePublished))
	systemRoutes.GET("/usage", access.RequirePermission(system.PermissionUsageView), resp.ToGinHandler(systemHandler.GetUsageReport))

	// course 模块 /api/student/*：学生可见的课程和套题
	api.GET("/workspace/skills", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListWorkspaceSkills))
	api.GET("/workspace/conversations", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListOwnedAIConversations))
	api.GET("/workspace/conversations/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetOwnedAIConversation))
	api.PUT("/workspace/conversations/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.RenameAIConversation))
	api.DELETE("/workspace/conversations/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.DeleteOwnedAIConversation))
	api.POST("/workspace/messages", access.RequireLogin(), resp.ToGinHandler(courseHandler.SendWorkspaceMessage))

	studentRoutes := api.Group("/student")
	studentRoutes.GET("/conversations", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListOwnedAIConversations))
	studentRoutes.GET("/courses", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourseList))
	studentRoutes.POST("/courses", access.RequireLogin(), resp.ToGinHandler(courseHandler.CreateStudentCourse))
	studentRoutes.GET("/courses/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourse))
	studentRoutes.PUT("/courses/:id/progress", access.RequireLogin(), resp.ToGinHandler(courseHandler.UpdateCourseProgress))
	studentRoutes.PUT("/courses/:id/sharing", access.RequireLogin(), resp.ToGinHandler(courseHandler.SetCourseSharing))
	studentRoutes.PUT("/courses/:id/content", access.RequireLogin(), resp.ToGinHandler(courseHandler.SaveStudentCourseContent))
	studentRoutes.POST("/courses/:id/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateStudentCourse))
	studentRoutes.POST("/courses/:id/chapters/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateStudentChapter))
	studentRoutes.POST("/courses/:id/activities/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateStudentActivity))
	studentRoutes.POST("/courses/:id/chapters/:chapterKey/ask", access.RequireLogin(), resp.ToGinHandler(courseHandler.AskChapterQuestion))
	studentRoutes.GET("/courses/:id/conversations", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListAIConversations))
	studentRoutes.GET("/courses/:id/conversations/:conversationId", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetAIConversation))
	studentRoutes.DELETE("/courses/:id/conversations/:conversationId", access.RequireLogin(), resp.ToGinHandler(courseHandler.DeleteAIConversation))
	studentRoutes.GET("/learning/dashboard", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetLearningDashboard))
	studentRoutes.POST("/learning/insights", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateLearningInsight))
	studentRoutes.POST("/courses/:id/learning/heartbeat", access.RequireLogin(), resp.ToGinHandler(courseHandler.RecordCourseHeartbeat))
	studentRoutes.GET("/questions", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetQuestionList))
	studentRoutes.POST("/questions", access.RequireLogin(), resp.ToGinHandler(courseHandler.CreateQuestion))
	studentRoutes.GET("/questions/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetQuestion))
	studentRoutes.PUT("/questions/:id/contents/:contentId", access.RequireLogin(), resp.ToGinHandler(courseHandler.AnswerQuestion))
	studentRoutes.GET("/courses/:id/quizzes", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListCourseQuizzes))

	api.GET("/quizzes/:quizId", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetQuiz))
	api.GET("/quizzes/:quizId/attempt", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetQuizAttempt))
	api.POST("/quizzes/:quizId/attempts", access.RequireLogin(), resp.ToGinHandler(courseHandler.StartQuizAttempt))
	api.PUT("/attempts/:attemptId/answers/:questionId", access.RequireLogin(), resp.ToGinHandler(courseHandler.UpdateAttemptAnswer))
	api.POST("/attempts/:attemptId/submit", access.RequireLogin(), resp.ToGinHandler(courseHandler.SubmitQuizAttempt))

	// course 模块 /api/teacher/*：教师自己的课，以及发到自己所在的班
	teacherRoutes := api.Group("/teacher")
	teacherRoutes.GET("/classes", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherClasses))
	teacherRoutes.GET("/courses", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherCourseList))
	teacherRoutes.POST("/courses", access.RequireLogin(), resp.ToGinHandler(courseHandler.CreateTeacherCourse))
	teacherRoutes.GET("/courses/:id", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherCourse))
	teacherRoutes.PUT("/courses/:id/classes", access.RequireLogin(), resp.ToGinHandler(courseHandler.PublishCourseToClasses))
	teacherRoutes.GET("/courses/:id/progress", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherCourseProgress))
	teacherRoutes.GET("/courses/:id/learning-overview", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherCourseLearningOverview))
	teacherRoutes.PUT("/courses/:id/content", access.RequireLogin(), resp.ToGinHandler(courseHandler.SaveTeacherCourseContent))
	teacherRoutes.POST("/courses/:id/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateTeacherCourse))
	teacherRoutes.POST("/courses/:id/chapters/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateTeacherChapter))
	teacherRoutes.POST("/courses/:id/activities/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateTeacherActivity))
	teacherRoutes.GET("/courses/:id/questions", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetTeacherQuestions))
	teacherRoutes.POST("/courses/:id/questions", access.RequireLogin(), resp.ToGinHandler(courseHandler.CreateTeacherQuestion))
	teacherRoutes.POST("/courses/:id/questions/import/preview", access.RequireLogin(), resp.ToGinHandler(courseHandler.PreviewTeacherQuestionImport))
	teacherRoutes.POST("/courses/:id/questions/import/confirm", access.RequireLogin(), resp.ToGinHandler(courseHandler.ConfirmTeacherQuestionImport))
	teacherRoutes.POST("/courses/:id/questions/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateTeacherQuestions))
	teacherRoutes.GET("/courses/:id/quality-report", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourseQualityReport))
	teacherRoutes.POST("/courses/:id/quality-report/generate", access.RequireLogin(), resp.ToGinHandler(courseHandler.GenerateCourseQualityReport))
	teacherRoutes.GET("/courses/:id/mermaid", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourseMermaid))
	teacherRoutes.PUT("/questions/:contentId", access.RequireLogin(), resp.ToGinHandler(courseHandler.UpdateTeacherQuestion))
	teacherRoutes.DELETE("/questions/:contentId", access.RequireLogin(), resp.ToGinHandler(courseHandler.DeleteTeacherQuestion))
	teacherRoutes.POST("/quizzes/:quizId/publish", access.RequireLogin(), resp.ToGinHandler(courseHandler.PublishQuiz))
	teacherRoutes.POST("/quizzes/:quizId/archive", access.RequireLogin(), resp.ToGinHandler(courseHandler.ArchiveQuiz))
	teacherRoutes.GET("/courses/:id/resources", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListTeacherCourseResources))
	teacherRoutes.POST("/courses/:id/resources", access.RequireLogin(), resp.ToGinHandler(courseHandler.UploadTeacherCourseResource))
	teacherRoutes.PATCH("/courses/:id/resources/:resourceId", access.RequireLogin(), resp.ToGinHandler(courseHandler.PatchTeacherCourseResource))
	teacherRoutes.PATCH("/courses/:id/resources/:resourceId/publish", access.RequireLogin(), resp.ToGinHandler(courseHandler.PublishTeacherCourseResource))
	teacherRoutes.DELETE("/courses/:id/resources/:resourceId", access.RequireLogin(), resp.ToGinHandler(courseHandler.DeleteTeacherCourseResource))

	studentRoutes.GET("/courses/:id/resources", access.RequireLogin(), resp.ToGinHandler(courseHandler.ListStudentCourseResources))

	api.GET("/courses/:courseId/resources/:resourceId/preview", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourseResourcePreview))
	api.GET("/courses/:courseId/resources/:resourceId/download", access.RequireLogin(), resp.ToGinHandler(courseHandler.GetCourseResourceDownload))
}
