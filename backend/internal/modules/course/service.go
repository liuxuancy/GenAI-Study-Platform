// 学生课程和套题。能看见一门课的规则在 visibility.go。套题在能看见所属课程时可见，自己创建的题始终可见。

package course

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/liuxuancy/Libs/logs"
	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/cos"
	"caveclass/internal/platform/errs"
)

type Service struct {
	database  *gorm.DB
	ai        *ai.Service
	aiBase    llm.Config
	cosClient *cos.Client
	cosLimits cos.Limits
}

// NewService 全局只初始化一次：启动时在路由表里创建。
func NewService(database *gorm.DB) *Service {
	_ = EnsureQuizIndexes(database)
	_ = ensureOptionalConversationCourse(database)
	return &Service{database: database}
}

// UseAI 挂上已调试好的模型门面。生成接口才需要它，其余课程接口不调用。
func (service *Service) UseAI(aiService *ai.Service) {
	service.ai = aiService
	if aiService != nil {
		service.aiBase = aiService.Config()
	}
}

// UseCOS 注入腾讯云 COS 客户端；未配置时上传接口返回明确错误。
func (service *Service) UseCOS(client *cos.Client, limits cos.Limits) {
	service.cosClient = client
	service.cosLimits = limits
}

// CourseListItem 一门课在学生列表里的样子。
// Visibility：private 仅自己，class 同班可见，assigned 教师发到自己所在的班。
type CourseListItem struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	CourseType   string `json:"courseType"`
	Description  string `json:"description"`
	Percent      int    `json:"percent"`
	Visibility   string `json:"visibility"`
	Owned        bool   `json:"owned"`
	ChapterCount int    `json:"chapterCount"`
}

type storedCourse struct {
	ID          int64
	Title       string
	CourseType  string
	Description string
	Audience    string
	Language    string
	ContentJSON string
	IsPublish   int
	OwnerID     int64
	Percent     int
	ClassCount  int
}

// GetCourseList 当前学生可见的课程和自己的进度。没有进度行时 percent 为 0。
func (service *Service) GetCourseList(ctx context.Context, principal *access.Principal) ([]CourseListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view courses")
	}

	var rows []storedCourse
	err := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title, COALESCE(c.course_type, '') AS course_type, COALESCE(c.description, '') AS description,
		        COALESCE(c.content_json, '') AS content_json, COALESCE(p.percent, 0) AS percent, c.is_publish, c.owner_id,
		        (SELECT COUNT(*) FROM course_class cc WHERE cc.course_id = c.id) AS class_count
		 FROM course c
		 LEFT JOIN student_course_progress p ON p.course_id = c.id AND p.user_id = ?
		 WHERE `+studentVisibleCourseSQL("c")+`
		 ORDER BY c.id`,
		append([]any{principal.UserID}, studentVisibleCourseArgs(principal.UserID)...)...,
	).Scan(&rows).Error
	if err != nil {
		logs.CtxError(ctx, "[GetCourseList] failed, err: %v", err)
		return nil, err
	}
	items := make([]CourseListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, CourseListItem{
			ID:           row.ID,
			Title:        row.Title,
			CourseType:   row.CourseType,
			Description:  row.Description,
			Percent:      row.Percent,
			Visibility:   visibilityLabel(row.IsPublish, row.ClassCount),
			Owned:        row.OwnerID == principal.UserID,
			ChapterCount: courseChapterCount(row.ContentJSON),
		})
	}
	logs.CtxInfo(ctx, "[GetCourseList] done: len(items)=%d", len(items))
	return items, nil
}

func courseChapterCount(contentJSON string) int {
	if strings.TrimSpace(contentJSON) == "" {
		return 0
	}
	var content struct {
		Chapters []json.RawMessage `json:"chapters"`
	}
	if json.Unmarshal([]byte(contentJSON), &content) != nil {
		return 0
	}
	return len(content.Chapters)
}

// QuestionListItem 一套题在答题列表里的样子。Percent 是已作答题数除以 question_count。
type QuestionListItem struct {
	ID             int64   `json:"id"`
	Title          string  `json:"title"`
	CourseTitle    string  `json:"courseTitle"`
	Difficulty     string  `json:"difficulty"`
	Answered       int     `json:"answered"`
	QuestionCount  int     `json:"questionCount"`
	Percent        int     `json:"percent"`
	CourseID       int64   `json:"courseId"`
	ChapterKey     string  `json:"chapterKey,omitempty"`
	ChapterName    string  `json:"chapterName,omitempty"`
	CreatorID      int64   `json:"creatorId"`
	CreatorName    string  `json:"creatorName"`
	CreatorRole    string  `json:"creatorRole"`
	IsSubmit       bool    `json:"isSubmit"`
	Score          *int    `json:"score,omitempty"`
	CreatedAt      *string `json:"createdAt,omitempty"`
	CreationMethod string  `json:"creationMethod,omitempty"`
	PublishStatus  string  `json:"publishStatus,omitempty"`
	SectionKey     string  `json:"sectionKey,omitempty"`
	SectionName    string  `json:"sectionName,omitempty"`
	AttemptID      *int64  `json:"attemptId,omitempty"`
}

type questionProgressRow struct {
	ID             int64
	Title          string
	CourseTitle    string
	Difficulty     string
	QuestionCount  int
	Answered       int
	CourseID       int64
	ChapterKey     string
	ContentJSON    string
	CreatorID      int64
	CreatorName    string
	CreatorRole    string
	IsSubmit       int
	Score          *int
	CreationMethod string
	PublishStatus  string
	SectionKey     string
	SectionName    string
	AttemptID      *int64
}

// GetQuestionList 当前学生能看见所属课程的套题，以及自己创建的套题。进度是答题进度，对错不计入。
func (service *Service) GetQuestionList(ctx context.Context, principal *access.Principal) ([]QuestionListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view questions")
	}

	var rows []questionProgressRow
	err := service.database.WithContext(ctx).Raw(
		`SELECT q.id, q.title, c.title AS course_title, COALESCE(q.difficulty, '') AS difficulty, q.question_count,
		        SUM(CASE WHEN p.user_option <> '' THEN 1 ELSE 0 END) AS answered,
		        q.course_id, COALESCE(q.chapter_key, '') AS chapter_key, COALESCE(c.content_json, '') AS content_json,
		        q.creator_id, COALESCE(u.nickname, u.username, '') AS creator_name, COALESCE(u.role, '') AS creator_role,
		        COALESCE(r.is_submit, 0) AS is_submit,
		        COALESCE(q.creation_method, '') AS creation_method,
		        COALESCE(q.publish_status, '') AS publish_status,
		        COALESCE(cs.section_key, q.section_key, '') AS section_key,
		        COALESCE(cs.title, '') AS section_name,
		        (SELECT id FROM quiz_attempt qa WHERE qa.quiz_id = q.id AND qa.student_id = ? AND qa.attempt_no = 1 LIMIT 1) AS attempt_id,
		        CASE WHEN COALESCE(r.is_submit, 0) = ?
		             THEN (SELECT CAST(ROUND(100.0 * SUM(CASE WHEN sp.user_option = qc2.correct_answer AND sp.user_option <> '' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0)) AS SIGNED)
		                   FROM course_question_content qc2
		                   LEFT JOIN student_question_progress sp ON sp.content_id = qc2.id AND sp.user_id = ?
		                   WHERE qc2.question_id = q.id)
		             ELSE NULL END AS score
		 FROM course_question q
		 JOIN course_question_content qc ON qc.question_id = q.id
		 JOIN course c ON c.id = q.course_id
		 JOIN sys_user u ON u.id = q.creator_id
		 LEFT JOIN course_section cs ON cs.id = q.section_id
		 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
		 LEFT JOIN student_question_record r ON r.question_id = q.id AND r.user_id = ?
		 WHERE (q.creator_id = ? OR `+studentVisibleCourseSQL("c")+`)
		   AND (q.creator_id = ? OR (`+effectivePublishStatusSQL+` = '`+PublishStatusPublished+`' AND COALESCE(q.publish_status, '') <> '`+PublishStatusArchived+`'))
		 GROUP BY q.id, q.title, c.title, q.difficulty, q.question_count, q.course_id, q.chapter_key, c.content_json,
		          q.creator_id, u.nickname, u.username, u.role, r.is_submit, q.creation_method, q.publish_status,
		          cs.section_key, cs.title, q.section_key
		 ORDER BY q.id`,
		append(append([]any{principal.UserID, SubmitOn, principal.UserID, principal.UserID, principal.UserID, principal.UserID}, studentVisibleCourseArgs(principal.UserID)...), principal.UserID)...,
	).Scan(&rows).Error
	if err != nil {
		logs.CtxError(ctx, "[GetQuestionList] failed, err: %v", err)
		return nil, err
	}

	items := make([]QuestionListItem, 0, len(rows))
	for _, row := range rows {
		chapterName := chapterTitleFromJSON(row.ContentJSON, row.ChapterKey)
		items = append(items, QuestionListItem{
			ID:             row.ID,
			Title:          row.Title,
			CourseTitle:    row.CourseTitle,
			Difficulty:     row.Difficulty,
			Answered:       row.Answered,
			QuestionCount:  row.QuestionCount,
			Percent:        answerPercent(row.Answered, row.QuestionCount),
			CourseID:       row.CourseID,
			ChapterKey:     row.ChapterKey,
			ChapterName:    chapterName,
			CreatorID:      row.CreatorID,
			CreatorName:    row.CreatorName,
			CreatorRole:    row.CreatorRole,
			IsSubmit:       row.IsSubmit == SubmitOn,
			Score:          row.Score,
			CreationMethod: row.CreationMethod,
			PublishStatus:  row.PublishStatus,
			SectionKey:     row.SectionKey,
			SectionName:    row.SectionName,
			AttemptID:      row.AttemptID,
		})
	}
	logs.CtxInfo(ctx, "[GetQuestionList] done: len(items)=%d", len(items))
	return items, nil
}

func answerPercent(answered int, questionCount int) int {
	if questionCount <= 0 || answered <= 0 {
		return 0
	}
	percent := answered * 100 / questionCount
	if percent > 100 {
		return 100
	}
	return percent
}

// CourseDetail 学生看到的一门课。没有正文时 Tree 为 null。
type CourseDetail struct {
	ID             int64       `json:"id"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Audience       string      `json:"audience"`
	Language       string      `json:"language"`
	Visibility     string      `json:"visibility"`
	Shared         bool        `json:"shared"`
	ClassmateCount int         `json:"classmateCount"`
	Owned          bool        `json:"owned"`
	Percent        int         `json:"percent"`
	Tree           *BubbleNode `json:"tree"`
}

// GetCourse 学生打开自己能看见的课。看不见和没有这门课都是 404。
func (service *Service) GetCourse(ctx context.Context, principal *access.Principal, courseID int64) (*CourseDetail, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view courses")
	}

	var row storedCourse
	result := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title, COALESCE(c.description, '') AS description, COALESCE(c.audience, '') AS audience,
		        COALESCE(c.language, '') AS language, COALESCE(c.content_json, '') AS content_json,
		        c.is_publish, c.owner_id,
		        (SELECT COUNT(*) FROM course_class cc WHERE cc.course_id = c.id) AS class_count
		 FROM course c
		 WHERE c.id = ? AND `+studentVisibleCourseSQL("c"),
		append([]any{courseID}, studentVisibleCourseArgs(principal.UserID)...)...,
	).Scan(&row)
	if result.Error != nil {
		logs.CtxError(ctx, "[GetCourse] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}
	output, err := service.studentCourseDetail(ctx, principal.UserID, row)
	if err != nil {
		logs.CtxError(ctx, "[GetCourse] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[GetCourse] done: id=%d title=%q visibility=%s owned=%t percent=%d", output.ID, output.Title, output.Visibility, output.Owned, output.Percent)
	return output, nil
}

func (service *Service) studentCourseDetail(ctx context.Context, viewerID int64, row storedCourse) (*CourseDetail, error) {
	questions, err := service.chapterQuestions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	tree, err := buildCourseTree(row.Title, row.Description, row.ContentJSON, questions)
	if err != nil {
		return nil, err
	}
	classmates, err := service.countClassmates(ctx, row.OwnerID)
	if err != nil {
		return nil, err
	}
	var percent int
	if err := service.database.WithContext(ctx).Raw(
		`SELECT COALESCE((SELECT percent FROM student_course_progress WHERE user_id = ? AND course_id = ?), 0)`,
		viewerID, row.ID,
	).Scan(&percent).Error; err != nil {
		return nil, err
	}
	return &CourseDetail{
		ID:             row.ID,
		Title:          row.Title,
		Description:    row.Description,
		Audience:       row.Audience,
		Language:       shownLanguage(row.Language),
		Visibility:     visibilityLabel(row.IsPublish, row.ClassCount),
		Shared:         row.IsPublish == PublishOn && row.ClassCount == 0,
		ClassmateCount: classmates,
		Owned:          row.OwnerID == viewerID,
		Percent:        percent,
		Tree:           tree,
	}, nil
}

// CourseProgress 当前学生对一门课的进度百分比。
type CourseProgress struct {
	Percent int `json:"percent"`
}

// UpdateCourseProgress 把当前学生对可见课程的进度写成给定百分比。没有进度行时插入。
func (service *Service) UpdateCourseProgress(ctx context.Context, principal *access.Principal, courseID int64, percent int) (*CourseProgress, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can update course progress")
	}
	if percent < 0 || percent > 100 {
		return nil, errs.BadRequest("percent must be from 0 to 100")
	}
	if _, err := service.GetCourse(ctx, principal, courseID); err != nil {
		logs.CtxError(ctx, "[UpdateCourseProgress] failed, err: %v", err)
		return nil, err
	}

	result := service.database.WithContext(ctx).Exec(
		`UPDATE student_course_progress SET percent = ? WHERE user_id = ? AND course_id = ?`,
		percent, principal.UserID, courseID,
	)
	if result.Error != nil {
		logs.CtxError(ctx, "[UpdateCourseProgress] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		result = service.database.WithContext(ctx).Exec(
			`INSERT INTO student_course_progress (user_id, course_id, percent) VALUES (?, ?, ?)`,
			principal.UserID, courseID, percent,
		)
		if result.Error != nil {
			logs.CtxError(ctx, "[UpdateCourseProgress] failed, err: %v", result.Error)
			return nil, result.Error
		}
	}
	courseRef := courseID
	service.recordLearningEvent(ctx, principal.UserID, EventCourseProgress, &courseRef, nil, "", 0)
	output := &CourseProgress{Percent: percent}
	logs.CtxInfo(ctx, "[UpdateCourseProgress] done: course_id=%d percent=%d", courseID, output.Percent)
	return output, nil
}

// QuestionDetail 一套题的标题、所属课程、进度，以及按 id 排序的题目。未交卷时不返回答案与解析。
type QuestionDetail struct {
	ID            int64                 `json:"id"`
	Title         string                `json:"title"`
	CourseTitle   string                `json:"courseTitle"`
	Difficulty    string                `json:"difficulty"`
	QuestionCount int                   `json:"questionCount"`
	Percent       int                   `json:"percent"`
	IsSubmit      bool                  `json:"isSubmit"`
	Score         *int                  `json:"score,omitempty"`
	Questions     []QuestionContentItem `json:"questions"`
}

// QuestionContentItem 一道题的内容、选项、已保存选项；交卷后含对错、正确答案与解析。
type QuestionContentItem struct {
	ID            int64    `json:"id"`
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	UserOption    string   `json:"userOption"`
	IsCorrect     bool     `json:"isCorrect,omitempty"`
	CorrectAnswer string   `json:"correctAnswer"`
	Explanation   string   `json:"explanation"`
}

// AnswerInput 保存或提交当前题所选的一项。Submit 为 true 时同时把这套题标为已提交。
type AnswerInput struct {
	Option string `json:"option"`
	Submit bool   `json:"submit"`
}

type questionContentRow struct {
	ID            int64
	Content       string
	Options       string
	UserOption    string
	CorrectAnswer string
	Explanation   string
}

type questionDetailHeaderRow struct {
	ID            int64
	Title         string
	CourseTitle   string
	Difficulty    string
	QuestionCount int
	Answered      int
	IsSubmit      int
	Score         *int
}

// GetQuestion 可见范围与列表相同：能看见所属课程，或这套题是自己创建的。否则 404。
func (service *Service) GetQuestion(ctx context.Context, principal *access.Principal, questionID int64) (*QuestionDetail, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view questions")
	}

	var row questionDetailHeaderRow
	result := service.database.WithContext(ctx).Raw(
		`SELECT q.id, q.title, c.title AS course_title, COALESCE(q.difficulty, '') AS difficulty, q.question_count,
		        SUM(CASE WHEN p.user_option <> '' THEN 1 ELSE 0 END) AS answered,
		        COALESCE(r.is_submit, 0) AS is_submit,
		        CASE WHEN COALESCE(r.is_submit, 0) = ?
		             THEN (SELECT CAST(ROUND(100.0 * SUM(CASE WHEN sp.user_option = qc2.correct_answer AND sp.user_option <> '' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0)) AS SIGNED)
		                   FROM course_question_content qc2
		                   LEFT JOIN student_question_progress sp ON sp.content_id = qc2.id AND sp.user_id = ?
		                   WHERE qc2.question_id = q.id)
		             ELSE NULL END AS score
		 FROM course_question q
		 JOIN course_question_content qc ON qc.question_id = q.id
		 JOIN course c ON c.id = q.course_id
		 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
		 LEFT JOIN student_question_record r ON r.question_id = q.id AND r.user_id = ?
		 WHERE q.id = ? AND (q.creator_id = ? OR `+studentVisibleCourseSQL("c")+`)
		   AND (q.creator_id = ? OR (`+effectivePublishStatusSQL+` = '`+PublishStatusPublished+`' AND COALESCE(q.publish_status, '') <> '`+PublishStatusArchived+`'))
		 GROUP BY q.id, q.title, c.title, q.difficulty, q.question_count, r.is_submit`,
		append(append([]any{SubmitOn, principal.UserID, principal.UserID, principal.UserID, questionID, principal.UserID}, studentVisibleCourseArgs(principal.UserID)...), principal.UserID)...,
	).Scan(&row)
	if result.Error != nil {
		logs.CtxError(ctx, "[GetQuestion] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("question not found")
	}

	var contents []questionContentRow
	err := service.database.WithContext(ctx).Raw(
		`SELECT qc.id, qc.content, qc.options, COALESCE(p.user_option, '') AS user_option,
		        qc.correct_answer, qc.explanation
		 FROM course_question_content qc
		 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
		 WHERE qc.question_id = ?
		 ORDER BY qc.id`,
		principal.UserID, questionID,
	).Scan(&contents).Error
	if err != nil {
		logs.CtxError(ctx, "[GetQuestion] failed, err: %v", err)
		return nil, err
	}

	submitted := row.IsSubmit == SubmitOn
	questions := make([]QuestionContentItem, 0, len(contents))
	for _, content := range contents {
		item := QuestionContentItem{
			ID:         content.ID,
			Content:    content.Content,
			Options:    splitOptions(content.Options),
			UserOption: content.UserOption,
		}
		if submitted {
			item.CorrectAnswer = content.CorrectAnswer
			item.Explanation = content.Explanation
			item.IsCorrect = content.UserOption != "" && content.UserOption == content.CorrectAnswer
		}
		questions = append(questions, item)
	}
	output := &QuestionDetail{
		ID:            row.ID,
		Title:         row.Title,
		CourseTitle:   row.CourseTitle,
		Difficulty:    row.Difficulty,
		QuestionCount: row.QuestionCount,
		Percent:       answerPercent(row.Answered, row.QuestionCount),
		IsSubmit:      submitted,
		Score:         row.Score,
		Questions:     questions,
	}
	logs.CtxInfo(ctx, "[GetQuestion] done: id=%d title=%q question_count=%d percent=%d is_submit=%t score=%s", output.ID, output.Title, output.QuestionCount, output.Percent, output.IsSubmit, formatOptionalInt(output.Score))
	return output, nil
}

func splitOptions(options string) []string {
	if options == "" {
		return []string{}
	}
	return strings.Split(options, "|")
}

// AnswerQuestion 把当前学生的选项写入做题进度。submit 为 true 时再把套题记录标为已提交。
func (service *Service) AnswerQuestion(ctx context.Context, principal *access.Principal, questionID int64, contentID int64, option string, submit bool) (*QuestionDetail, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can answer questions")
	}

	var content questionContentRow
	result := service.database.WithContext(ctx).Raw(
		`SELECT qc.id, qc.content, qc.options
		 FROM course_question_content qc
		 JOIN course_question q ON q.id = qc.question_id
		 JOIN course c ON c.id = q.course_id
		 WHERE qc.id = ? AND qc.question_id = ?
		   AND (q.creator_id = ? OR `+studentVisibleCourseSQL("c")+`)
		   AND (q.creator_id = ? OR (`+effectivePublishStatusSQL+` = '`+PublishStatusPublished+`' AND COALESCE(q.publish_status, '') <> '`+PublishStatusArchived+`'))`,
		append(append([]any{contentID, questionID, principal.UserID}, studentVisibleCourseArgs(principal.UserID)...), principal.UserID)...,
	).Scan(&content)
	if result.Error != nil {
		logs.CtxError(ctx, "[AnswerQuestion] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("question not found")
	}
	if !optionAllowed(content.Options, option) {
		return nil, errs.BadRequest("option is not one of the choices")
	}

	if err := saveStudentAnswer(service.database.WithContext(ctx), principal.UserID, questionID, contentID, option); err != nil {
		logs.CtxError(ctx, "[AnswerQuestion] failed, err: %v", err)
		return nil, err
	}
	if submit {
		if err := markQuestionSubmitted(service.database.WithContext(ctx), principal.UserID, questionID); err != nil {
			logs.CtxError(ctx, "[AnswerQuestion] failed, err: %v", err)
			return nil, err
		}
	}
	if err := syncQuizAttemptFromLegacy(ctx, service.database, principal.UserID, questionID); err != nil {
		logs.CtxError(ctx, "[AnswerQuestion] failed, err: %v", err)
		return nil, err
	}
	service.recordQuizLearningEvents(ctx, principal.UserID, questionID, submit)
	output, err := service.GetQuestion(ctx, principal, questionID)
	if err != nil {
		logs.CtxError(ctx, "[AnswerQuestion] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[AnswerQuestion] done: content_id=%d option=%q submit=%t id=%d title=%q question_count=%d percent=%d is_submit=%t score=%s", contentID, option, submit, output.ID, output.Title, output.QuestionCount, output.Percent, output.IsSubmit, formatOptionalInt(output.Score))
	return output, nil
}

func saveStudentAnswer(database *gorm.DB, userID int64, questionID int64, contentID int64, option string) error {
	updated := database.Exec(
		`UPDATE student_question_progress SET user_option = ? WHERE user_id = ? AND content_id = ?`,
		option, userID, contentID,
	)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected > 0 {
		return nil
	}
	inserted := database.Exec(
		`INSERT INTO student_question_progress (user_id, content_id, question_id, user_option) VALUES (?, ?, ?, ?)`,
		userID, contentID, questionID, option,
	)
	return inserted.Error
}

func markQuestionSubmitted(database *gorm.DB, userID int64, questionID int64) error {
	updated := database.Exec(
		`UPDATE student_question_record SET is_submit = ? WHERE user_id = ? AND question_id = ?`,
		SubmitOn, userID, questionID,
	)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected > 0 {
		return nil
	}
	inserted := database.Exec(
		`INSERT INTO student_question_record (user_id, question_id, is_submit) VALUES (?, ?, ?)`,
		userID, questionID, SubmitOn,
	)
	return inserted.Error
}

// CreateQuestionInput 确认后写入的一套题。难度写入套题表，并拼进标题保证唯一。
type CreateQuestionInput struct {
	CourseID   int64                `json:"courseId"`
	Difficulty string               `json:"difficulty"`
	Questions  []CreateQuestionItem `json:"questions"`
}

// CreateQuestionItem 预览里的一道题。
type CreateQuestionItem struct {
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correctAnswer"`
}

// CreateQuestion 把确认后的预览写入套题。学生创建的套题不公开。
func (service *Service) CreateQuestion(ctx context.Context, principal *access.Principal, input CreateQuestionInput) (*QuestionListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can create questions")
	}
	if err := validateCreateQuestion(input); err != nil {
		logs.CtxError(ctx, "[CreateQuestion] failed, err: %v", err)
		return nil, err
	}

	var course struct {
		ID    int64
		Title string
	}
	result := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title
		 FROM course c
		 WHERE c.id = ? AND `+studentVisibleCourseSQL("c"),
		append([]any{input.CourseID}, studentVisibleCourseArgs(principal.UserID)...)...,
	).Scan(&course)
	if result.Error != nil {
		logs.CtxError(ctx, "[CreateQuestion] failed, err: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}

	now := time.Now().UTC()
	title := questionSetTitle(course.Title, input.Difficulty, now)
	tx := service.database.WithContext(ctx).Begin()
	if tx.Error != nil {
		logs.CtxError(ctx, "[CreateQuestion] failed, err: %v", tx.Error)
		return nil, tx.Error
	}
	questionID, err := insertQuestionSet(tx, title, course.ID, principal.UserID, input.Difficulty, input.Questions)
	if err != nil {
		tx.Rollback()
		logs.CtxError(ctx, "[CreateQuestion] failed, err: %v", err)
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		logs.CtxError(ctx, "[CreateQuestion] failed, err: %v", err)
		return nil, err
	}
	output := &QuestionListItem{
		ID:            questionID,
		Title:         title,
		CourseTitle:   course.Title,
		Difficulty:    input.Difficulty,
		Answered:      0,
		QuestionCount: len(input.Questions),
		Percent:       0,
		CourseID:      course.ID,
		CreatorID:     principal.UserID,
		CreatorRole:   principal.Role,
	}
	logs.CtxInfo(ctx, "[CreateQuestion] done: resp=%+v", output)
	return output, nil
}

func validateCreateQuestion(input CreateQuestionInput) error {
	if !allowedQuestionCount(len(input.Questions)) {
		return errs.BadRequest("question count must be 2, 5, or 10")
	}
	if !allowedDifficulty(input.Difficulty) {
		return errs.BadRequest("difficulty must be easy, medium, or hard")
	}
	for _, item := range input.Questions {
		if strings.TrimSpace(item.Content) == "" || len(item.Options) < 2 {
			return errs.BadRequest("each question needs content and at least two options")
		}
		if !choiceIncluded(item.Options, item.CorrectAnswer) {
			return errs.BadRequest("correct answer must be one of the options")
		}
	}
	return nil
}

func allowedQuestionCount(count int) bool {
	return count == 2 || count == 5 || count == 10
}

func allowedDifficulty(difficulty string) bool {
	return difficulty == "easy" || difficulty == "medium" || difficulty == "hard"
}

func choiceIncluded(options []string, answer string) bool {
	if answer == "" {
		return false
	}
	for _, option := range options {
		if option == answer {
			return true
		}
	}
	return false
}

func questionSetTitle(courseTitle string, difficulty string, now time.Time) string {
	suffix := " · " + difficulty + " · " + now.Format("2006-01-02 15:04:05")
	runes := []rune(courseTitle)
	room := 200 - len([]rune(suffix))
	if room < 1 {
		room = 1
	}
	if len(runes) > room {
		runes = runes[:room]
	}
	return string(runes) + suffix
}

func insertQuestionSet(tx *gorm.DB, title string, courseID int64, creatorID int64, difficulty string, items []CreateQuestionItem) (int64, error) {
	now := time.Now().UTC()
	result := tx.Exec(
		`INSERT INTO course_question
		 (title, course_id, creator_id, question_count, is_publish, difficulty, creation_method, publish_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		title, courseID, creatorID, len(items), PublishOff, difficulty, CreationMethodManual, PublishStatusDraft, now, now,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	var questionID int64
	found := tx.Raw(`SELECT id FROM course_question WHERE title = ?`, title).Scan(&questionID)
	if found.Error != nil {
		return 0, found.Error
	}
	if found.RowsAffected == 0 {
		return 0, errs.NotFound("question not found")
	}
	for index, item := range items {
		sortOrder := index + 1
		inserted := tx.Exec(
			`INSERT INTO course_question_content
			 (question_id, content, options, correct_answer, explanation, question_type, sort_order, score, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			questionID, item.Content, strings.Join(item.Options, "|"), item.CorrectAnswer, "",
			QuestionTypeSingleChoice, sortOrder, 1, now, now,
		)
		if inserted.Error != nil {
			return 0, inserted.Error
		}
	}
	return questionID, nil
}

func optionAllowed(options string, option string) bool {
	if option == "" {
		return false
	}
	for _, choice := range splitOptions(options) {
		if choice == option {
			return true
		}
	}
	return false
}

// formatOptionalInt 打日志用：可选的整数（如未交卷时的分数），没有写 nil，有就写值（直接 %v 会打出指针地址）。
func formatOptionalInt(value *int) string {
	if value == nil {
		return "nil"
	}
	return strconv.Itoa(*value)
}
