// course 模块的声明：课程、进度、套题、题目内容、做题进度和套题记录。列表接口只要求登录，不声明权限码。

package course

import (
	"caveclass/internal/framework/access"
	"caveclass/internal/usage"
)

// Module 实现 module.Module。
type Module struct{}

func (Module) GetName() string                        { return "course" }
func (Module) GetNamespaces() []string                { return nil }
func (Module) GetPermissions() []access.PermissionDef { return nil }
func (Module) GetDefaultGrants() map[string][]string  { return nil }
func (Module) GetModels() []any {
	return []any{
		&Course{},
		&CoursePixelAvatar{},
		&CourseChapter{},
		&CourseSection{},
		&StudentCourseProgress{},
		&StudentAchievement{},
		&StudentLearningEvent{},
		&StudentAIConversation{},
		&StudentAIMessage{},
		&CourseQuestion{},
		&StudentQuizChapter{},
		&CourseQuestionContent{},
		&StudentQuestionProgress{},
		&StudentQuestionRecord{},
		&QuizAttempt{},
		&QuizAnswer{},
		&MigrationLog{},
		&StudyClass{},
		&StudyClassMember{},
		&CourseClass{},
		&CourseQualityReport{},
		&CourseResource{},
		&usage.Usage{},
	}
}
