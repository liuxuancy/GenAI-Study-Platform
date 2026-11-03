package course

import (
	"regexp"
	"strings"

	"caveclass/internal/framework/access"
)

var (
	chapterQuizTitlePattern = regexp.MustCompile(`^Quiz · .+ · \d+ · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	studentGeneratedTitle   = regexp.MustCompile(`^(.+) · (easy|medium|hard) · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
)

// effectivePublishStatusSQL 在 SELECT 里给 publish_status 列起别名 effective_publish。
const effectivePublishStatusSQL = `COALESCE(NULLIF(q.publish_status, ''), CASE WHEN q.is_publish = 1 THEN '` + PublishStatusPublished + `' ELSE '` + PublishStatusDraft + `' END)`

// studentVisibleQuizFilterSQL 非创建者只能看已发布且未归档的测验。
const studentVisibleQuizFilterSQL = `(q.creator_id = ? OR (` + effectivePublishStatusSQL + ` = '` + PublishStatusPublished + `' AND COALESCE(q.publish_status, '') <> '` + PublishStatusArchived + `'))`

func publishStatusFromLegacy(isPublish int) string {
	if isPublish == PublishOn {
		return PublishStatusPublished
	}
	return PublishStatusDraft
}

func inferCreationMethod(title string, chapterKey string, creatorRole string) string {
	title = strings.TrimSpace(title)
	if strings.HasPrefix(title, "Excel 导入") {
		return CreationMethodExcel
	}
	if chapterQuizTitlePattern.MatchString(title) {
		return CreationMethodAI
	}
	if strings.HasPrefix(title, "教师题库") {
		return CreationMethodManual
	}
	if studentGeneratedTitle.MatchString(title) {
		return CreationMethodManual
	}
	if creatorRole == access.RoleTeacher && strings.TrimSpace(chapterKey) != "" {
		return CreationMethodAI
	}
	if creatorRole == access.RoleTeacher {
		return CreationMethodManual
	}
	if creatorRole == access.RoleStudent {
		return CreationMethodManual
	}
	return ""
}
