package course

import (
	"fmt"

	"caveclass/internal/framework/access"
)

const (
	VisibilityPrivate  = "private"
	VisibilityClass    = "class"
	VisibilityAssigned = "assigned"
)

// studentVisibleCourseSQL 拼出「谁能看见一门课」的三条规则，互不重叠。
// 别名只能是代码里的表别名。参数顺序：用户、PublishOn、用户、PublishOn、用户。
func studentVisibleCourseSQL(alias string) string {
	return fmt.Sprintf(`(
		%[1]s.owner_id = ?
		OR (
			%[1]s.is_publish = ?
			AND NOT EXISTS (SELECT 1 FROM course_class cc_vis WHERE cc_vis.course_id = %[1]s.id)
			AND EXISTS (
				SELECT 1
				FROM study_class_member viewer_member
				JOIN study_class_member owner_member
				  ON owner_member.class_id = viewer_member.class_id
				 AND owner_member.user_id = %[1]s.owner_id
				 AND owner_member.user_id <> viewer_member.user_id
				WHERE viewer_member.user_id = ?
			)
		)
		OR (
			%[1]s.is_publish = ?
			AND EXISTS (
				SELECT 1
				FROM course_class cc_pub
				JOIN study_class_member viewer_member
				  ON viewer_member.class_id = cc_pub.class_id
				 AND viewer_member.user_id = ?
				WHERE cc_pub.course_id = %[1]s.id
			)
		)
	)`, alias)
}

func studentVisibleCourseArgs(userID int64) []any {
	return []any{userID, PublishOn, userID, PublishOn, userID}
}

func visibilityLabel(isPublish int, classCount int) string {
	if isPublish != PublishOn {
		return VisibilityPrivate
	}
	if classCount > 0 {
		return VisibilityAssigned
	}
	return VisibilityClass
}

func shownLanguage(language string) string {
	if language == "" {
		return DefaultLanguage
	}
	return language
}

// 学生接口只对学生开放，所以「还有谁能看见」只数同班学生，不数教师。
const classmateCountSQL = `
	SELECT COUNT(DISTINCT other_member.user_id)
	FROM study_class_member mine
	JOIN study_class_member other_member
	  ON other_member.class_id = mine.class_id
	 AND other_member.user_id <> mine.user_id
	JOIN sys_user other_user ON other_user.id = other_member.user_id AND other_user.role = ?
	WHERE mine.user_id = ?`

func classmateRole() string { return access.RoleStudent }
