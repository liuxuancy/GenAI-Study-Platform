package course

import (
	"context"

	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// TeacherQuizOverviewItem 一门课下的套题/测验小节摘要。
type TeacherQuizOverviewItem struct {
	QuizID        int64  `json:"quizId"`
	Title         string `json:"title"`
	ChapterKey    string `json:"chapterKey"`
	SectionKey    string `json:"sectionKey"`
	QuestionCount int    `json:"questionCount"`
}

// StudentQuizOverviewStat 某学生对某套题的汇总。
type StudentQuizOverviewStat struct {
	QuizID    int64 `json:"quizId"`
	Answered  int   `json:"answered"`
	Total     int   `json:"total"`
	Submitted bool  `json:"submitted"`
	Score     *int  `json:"score"`
}

// TeacherStudentLearningRow 发布范围内一名学生的课内进度与测验汇总。
type TeacherStudentLearningRow struct {
	UserID   int64                     `json:"userId"`
	Username string                    `json:"username"`
	Nickname string                    `json:"nickname"`
	Percent  int                       `json:"percent"`
	Quizzes  []StudentQuizOverviewStat `json:"quizzes"`
}

// TeacherCourseLearningOverview 教师学情概览。
type TeacherCourseLearningOverview struct {
	Quizzes  []TeacherQuizOverviewItem   `json:"quizzes"`
	Students []TeacherStudentLearningRow `json:"students"`
}

// GetTeacherCourseLearningOverview GET /api/teacher/courses/:id/learning-overview
func (service *Service) GetTeacherCourseLearningOverview(ctx context.Context, principal *access.Principal, courseID int64) (*TeacherCourseLearningOverview, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can view course learning overview")
	}
	var ownerID int64
	found := service.database.WithContext(ctx).Raw(
		"SELECT owner_id FROM course WHERE id = ? AND owner_id = ?",
		courseID, principal.UserID,
	).Scan(&ownerID)
	if found.Error != nil {
		logs.CtxError(ctx, "[GetTeacherCourseLearningOverview] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}

	var quizzes []TeacherQuizOverviewItem
	err := service.database.WithContext(ctx).Raw(
		`SELECT id AS quiz_id, title,
		        COALESCE(chapter_key, '') AS chapter_key,
		        COALESCE(section_key, '') AS section_key,
		        question_count
		 FROM course_question
		 WHERE course_id = ?
		 ORDER BY chapter_key, section_key, id`,
		courseID,
	).Scan(&quizzes).Error
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourseLearningOverview] failed, err: %v", err)
		return nil, err
	}
	if quizzes == nil {
		quizzes = []TeacherQuizOverviewItem{}
	}

	var students []struct {
		UserID   int64
		Username string
		Nickname string
		Percent  int
	}
	err = service.database.WithContext(ctx).Raw(
		`SELECT DISTINCT u.id AS user_id, u.username, u.nickname, COALESCE(p.percent, 0) AS percent
		 FROM course_class cc
		 JOIN study_class_member m ON m.class_id = cc.class_id
		 JOIN sys_user u ON u.id = m.user_id AND u.role = ?
		 LEFT JOIN student_course_progress p ON p.user_id = u.id AND p.course_id = cc.course_id
		 WHERE cc.course_id = ?
		 ORDER BY u.id`,
		access.RoleStudent, courseID,
	).Scan(&students).Error
	if err != nil {
		logs.CtxError(ctx, "[GetTeacherCourseLearningOverview] failed, err: %v", err)
		return nil, err
	}

	type statRow struct {
		UserID    int64
		QuizID    int64
		Answered  int
		Total     int
		Submitted int
		Score     *int
	}
	var statRows []statRow
	if len(students) > 0 && len(quizzes) > 0 {
		err = service.database.WithContext(ctx).Raw(
			`SELECT s.user_id, q.id AS quiz_id,
			        COALESCE(qa.answered_questions, leg.answered, 0) AS answered,
			        COALESCE(NULLIF(qa.total_questions, 0), q.question_count, 0) AS total,
			        CASE
			          WHEN COALESCE(qa.status, '') = ? OR COALESCE(sqr.is_submit, 0) = ? THEN 1
			          ELSE 0
			        END AS submitted,
			        COALESCE(qa.score, leg_score.score) AS score
			 FROM (
			   SELECT DISTINCT u.id AS user_id
			   FROM course_class cc
			   JOIN study_class_member m ON m.class_id = cc.class_id
			   JOIN sys_user u ON u.id = m.user_id AND u.role = ?
			   WHERE cc.course_id = ?
			 ) s
			 CROSS JOIN course_question q
			 LEFT JOIN quiz_attempt qa ON qa.quiz_id = q.id AND qa.student_id = s.user_id AND qa.attempt_no = 1
			 LEFT JOIN (
			   SELECT user_id, question_id, COUNT(*) AS answered
			   FROM student_question_progress
			   WHERE COALESCE(user_option, '') <> ''
			   GROUP BY user_id, question_id
			 ) leg ON leg.user_id = s.user_id AND leg.question_id = q.id
			 LEFT JOIN student_question_record sqr ON sqr.user_id = s.user_id AND sqr.question_id = q.id
			 LEFT JOIN (
			   SELECT p.user_id, p.question_id,
			          CAST(ROUND(100.0 * SUM(CASE WHEN p.user_option = qc.correct_answer THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0)) AS SIGNED) AS score
			   FROM student_question_progress p
			   JOIN course_question_content qc ON qc.id = p.content_id
			   JOIN student_question_record r ON r.user_id = p.user_id AND r.question_id = p.question_id AND r.is_submit = ?
			   WHERE COALESCE(p.user_option, '') <> ''
			   GROUP BY p.user_id, p.question_id
			 ) leg_score ON leg_score.user_id = s.user_id AND leg_score.question_id = q.id
			 WHERE q.course_id = ?`,
			AttemptStatusSubmitted, SubmitOn,
			access.RoleStudent, courseID,
			SubmitOn, courseID,
		).Scan(&statRows).Error
		if err != nil {
			logs.CtxError(ctx, "[GetTeacherCourseLearningOverview] failed, err: %v", err)
			return nil, err
		}
	}

	statByUserQuiz := make(map[int64]map[int64]StudentQuizOverviewStat)
	for _, row := range statRows {
		if statByUserQuiz[row.UserID] == nil {
			statByUserQuiz[row.UserID] = make(map[int64]StudentQuizOverviewStat)
		}
		submitted := row.Submitted == 1
		score := row.Score
		if !submitted {
			score = nil
		}
		statByUserQuiz[row.UserID][row.QuizID] = StudentQuizOverviewStat{
			QuizID: row.QuizID, Answered: row.Answered, Total: row.Total,
			Submitted: submitted, Score: score,
		}
	}

	studentRows := make([]TeacherStudentLearningRow, 0, len(students))
	for _, student := range students {
		quizStats := make([]StudentQuizOverviewStat, 0, len(quizzes))
		for _, quiz := range quizzes {
			stat, ok := statByUserQuiz[student.UserID][quiz.QuizID]
			if !ok {
				stat = StudentQuizOverviewStat{
					QuizID: quiz.QuizID, Total: quiz.QuestionCount,
				}
			}
			quizStats = append(quizStats, stat)
		}
		studentRows = append(studentRows, TeacherStudentLearningRow{
			UserID: student.UserID, Username: student.Username, Nickname: student.Nickname,
			Percent: student.Percent, Quizzes: quizStats,
		})
	}
	if studentRows == nil {
		studentRows = []TeacherStudentLearningRow{}
	}

	output := &TeacherCourseLearningOverview{Quizzes: quizzes, Students: studentRows}
	logs.CtxInfo(ctx, "[GetTeacherCourseLearningOverview] done: course_id=%d len(quizzes)=%d len(students)=%d", courseID, len(output.Quizzes), len(output.Students))
	return output, nil
}
