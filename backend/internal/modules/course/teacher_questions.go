package course

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"caveclass/internal/ai"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

// TeacherQuestionItem 是教师题库中的一道选择题。
type TeacherQuestionItem struct {
	ID            int64    `json:"id"`
	SetID         int64    `json:"setId"`
	SetTitle      string   `json:"setTitle"`
	ChapterKey    string   `json:"chapterKey,omitempty"`
	SectionKey    string   `json:"sectionKey,omitempty"`
	Difficulty    string   `json:"difficulty"`
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correctAnswer"`
	Explanation   string   `json:"explanation"`
}

type TeacherQuestionInput struct {
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correctAnswer"`
	Explanation   string   `json:"explanation"`
	Difficulty    string   `json:"difficulty"`
	ChapterKey    string   `json:"chapterKey"`
	SectionKey    string   `json:"sectionKey"`
}

type ImportQuestionRow struct {
	Row           int      `json:"row"`
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correctAnswer"`
	Explanation   string   `json:"explanation"`
	Difficulty    string   `json:"difficulty"`
}

type QuestionImportInput struct {
	Rows           []ImportQuestionRow `json:"rows"`
	ChapterKey     string              `json:"chapterKey"`
	SectionKey     string              `json:"sectionKey"`
	CreationMethod string              `json:"creationMethod"`
}

type QuestionImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type QuestionImportPreview struct {
	ValidCount int                   `json:"validCount"`
	Rows       []ImportQuestionRow   `json:"rows"`
	Errors     []QuestionImportError `json:"errors"`
}

type AIQuestionInput struct {
	Provider          string `json:"provider"`
	ChapterKey        string `json:"chapterKey"`
	Count             int    `json:"count"`
	Difficulty        string `json:"difficulty"`
	ExtraRequirements string `json:"extraRequirements"`
}

type AIQuestionPreview struct {
	Questions []TeacherQuestionInput `json:"questions"`
	Meta      ai.Meta                `json:"meta"`
}

func (service *Service) ListTeacherQuestions(ctx context.Context, principal *access.Principal, courseID int64) ([]TeacherQuestionItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	var rows []struct {
		ID, SetID                                          int64
		SetTitle, Difficulty, Content, Options             string
		CorrectAnswer, Explanation, ChapterKey, SectionKey string
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT qc.id, q.id AS set_id, q.title AS set_title, COALESCE(q.difficulty, '') AS difficulty,
		        qc.content, qc.options, qc.correct_answer, qc.explanation,
		        COALESCE(q.chapter_key, '') AS chapter_key, COALESCE(q.section_key, '') AS section_key
		 FROM course_question_content qc
		 JOIN course_question q ON q.id = qc.question_id
		 WHERE q.course_id = ? AND q.creator_id = ?
		 ORDER BY q.chapter_key, q.section_key, qc.sort_order, qc.id`,
		courseID, principal.UserID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]TeacherQuestionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, TeacherQuestionItem{
			ID: row.ID, SetID: row.SetID, SetTitle: row.SetTitle, Difficulty: row.Difficulty,
			ChapterKey: row.ChapterKey, SectionKey: row.SectionKey,
			Content: row.Content, Options: splitOptions(row.Options),
			CorrectAnswer: row.CorrectAnswer, Explanation: row.Explanation,
		})
	}
	return items, nil
}

func (service *Service) CreateTeacherQuestion(ctx context.Context, principal *access.Principal, courseID int64, input TeacherQuestionInput) (*TeacherQuestionItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	input = normalizeTeacherQuestion(input)
	if err := validateTeacherQuestion(input); err != nil {
		return nil, err
	}
	var item TeacherQuestionItem
	err := service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chapterKey := strings.TrimSpace(input.ChapterKey)
		sectionKey := strings.TrimSpace(input.SectionKey)
		if chapterKey == "" || sectionKey == "" {
			return errs.BadRequest("choose a quiz section before adding questions")
		}
		setID, err := findSectionQuizID(tx, courseID, chapterKey, sectionKey)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		contentID, err := appendQuizContent(tx, setID, input.Content, input.Options, input.CorrectAnswer, input.Explanation, now)
		if err != nil {
			return err
		}
		if err := publishQuizWithQuestions(tx, setID, now); err != nil {
			return err
		}
		var title string
		if err := tx.Raw(`SELECT title FROM course_question WHERE id = ?`, setID).Scan(&title).Error; err != nil {
			return err
		}
		item = TeacherQuestionItem{ID: contentID, SetID: setID, SetTitle: title, Difficulty: input.Difficulty,
			ChapterKey: chapterKey, SectionKey: sectionKey,
			Content: input.Content, Options: input.Options, CorrectAnswer: input.CorrectAnswer, Explanation: input.Explanation}
		return nil
	})
	return &item, err
}

func (service *Service) UpdateTeacherQuestion(ctx context.Context, principal *access.Principal, contentID int64, input TeacherQuestionInput) (*TeacherQuestionItem, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can edit questions")
	}
	input = normalizeTeacherQuestion(input)
	if err := validateTeacherQuestion(input); err != nil {
		return nil, err
	}
	result := service.database.WithContext(ctx).Exec(
		`UPDATE course_question_content
		 SET content = ?, options = ?, correct_answer = ?, explanation = ?
		 WHERE id = ? AND question_id IN (SELECT id FROM course_question WHERE creator_id = ?)`,
		input.Content, strings.Join(input.Options, "|"), input.CorrectAnswer, input.Explanation, contentID, principal.UserID,
	)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("question not found")
	}
	if err := service.database.WithContext(ctx).Exec(
		`UPDATE course_question SET difficulty = ? WHERE id = (SELECT question_id FROM course_question_content WHERE id = ?)`,
		input.Difficulty, contentID,
	).Error; err != nil {
		return nil, err
	}
	var row struct {
		SetID                int64
		SetTitle, Difficulty string
	}
	if err := service.database.WithContext(ctx).Raw(
		`SELECT q.id AS set_id, q.title AS set_title, q.difficulty
		 FROM course_question q JOIN course_question_content qc ON qc.question_id = q.id WHERE qc.id = ?`,
		contentID,
	).Scan(&row).Error; err != nil {
		return nil, err
	}
	return &TeacherQuestionItem{ID: contentID, SetID: row.SetID, SetTitle: row.SetTitle, Difficulty: input.Difficulty,
		Content: input.Content, Options: input.Options, CorrectAnswer: input.CorrectAnswer, Explanation: input.Explanation}, nil
}

// DeleteTeacherQuestion 删除一道题，并去掉学生在这道题上的作答。测验小节本身保留。
func (service *Service) DeleteTeacherQuestion(ctx context.Context, principal *access.Principal, contentID int64) error {
	if principal.Role != access.RoleTeacher {
		return errs.Forbidden("only teachers can delete questions")
	}
	return service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var quizID int64
		found := tx.Raw(
			`SELECT q.id FROM course_question q
			 JOIN course_question_content qc ON qc.question_id = q.id
			 WHERE qc.id = ? AND q.creator_id = ?`,
			contentID, principal.UserID,
		).Scan(&quizID)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected == 0 || quizID == 0 {
			return errs.NotFound("question not found")
		}
		for _, statement := range []string{
			`DELETE FROM quiz_answer WHERE question_id = ?`,
			`DELETE FROM student_question_progress WHERE content_id = ?`,
			`DELETE FROM course_question_content WHERE id = ?`,
		} {
			if err := tx.Exec(statement, contentID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(
			`UPDATE course_question
			 SET question_count = (SELECT COUNT(*) FROM course_question_content WHERE question_id = ?),
			     updated_at = ?
			 WHERE id = ?`,
			quizID, time.Now().UTC(), quizID,
		).Error
	})
}

func (service *Service) PreviewQuestionImport(ctx context.Context, principal *access.Principal, courseID int64, input QuestionImportInput) (*QuestionImportPreview, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	return validateImportRows(input.Rows), nil
}

func (service *Service) ConfirmQuestionImport(ctx context.Context, principal *access.Principal, courseID int64, input QuestionImportInput) ([]TeacherQuestionItem, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	preview := validateImportRows(input.Rows)
	if len(preview.Errors) > 0 || len(preview.Rows) == 0 {
		return nil, errs.BadRequest("question import contains invalid rows")
	}
	created := make([]TeacherQuestionItem, 0, len(preview.Rows))
	err := service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chapterKey := strings.TrimSpace(input.ChapterKey)
		sectionKey := strings.TrimSpace(input.SectionKey)
		if chapterKey == "" || sectionKey == "" {
			return errs.BadRequest("choose a quiz section before importing questions")
		}
		setID, err := findSectionQuizID(tx, courseID, chapterKey, sectionKey)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		method := CreationMethodExcel
		if input.CreationMethod == CreationMethodAI || input.CreationMethod == CreationMethodManual {
			method = input.CreationMethod
		}
		if err := tx.Exec(
			`UPDATE course_question SET difficulty = ?, creation_method = ?, updated_at = ? WHERE id = ?`,
			commonDifficulty(preview.Rows), method, now, setID,
		).Error; err != nil {
			return err
		}
		var title string
		if err := tx.Raw(`SELECT title FROM course_question WHERE id = ?`, setID).Scan(&title).Error; err != nil {
			return err
		}
		for _, row := range preview.Rows {
			contentID, err := appendQuizContent(tx, setID, row.Content, row.Options, row.CorrectAnswer, row.Explanation, now)
			if err != nil {
				return err
			}
			created = append(created, TeacherQuestionItem{ID: contentID, SetID: setID, SetTitle: title,
				ChapterKey: chapterKey, SectionKey: sectionKey,
				Difficulty: row.Difficulty, Content: row.Content, Options: row.Options,
				CorrectAnswer: row.CorrectAnswer, Explanation: row.Explanation})
		}
		return publishQuizWithQuestions(tx, setID, now)
	})
	return created, err
}

func (service *Service) GenerateTeacherQuestions(ctx context.Context, principal *access.Principal, courseID int64, input AIQuestionInput) (*AIQuestionPreview, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	if input.Count < 1 || input.Count > 20 {
		return nil, errs.BadRequest("count must be from 1 to 20")
	}
	if !allowedDifficulty(input.Difficulty) {
		return nil, errs.BadRequest("difficulty must be easy, medium, or hard")
	}
	var row storedCourse
	found := service.database.WithContext(ctx).Raw(
		`SELECT title, COALESCE(audience, '') AS audience, COALESCE(language, '') AS language,
		        COALESCE(content_json, '') AS content_json
		 FROM course WHERE id = ? AND owner_id = ?`,
		courseID, principal.UserID,
	).Scan(&row)
	if found.Error != nil {
		return nil, found.Error
	}
	var contextCourse aicourse.StandardCourse
	if row.ContentJSON != "" {
		if err := json.Unmarshal([]byte(row.ContentJSON), &contextCourse); err != nil {
			return nil, errs.Internal(err)
		}
	}
	chapterKey := strings.TrimSpace(input.ChapterKey)
	if chapterKey == "" && len(contextCourse.Chapters) > 0 {
		chapterKey = contextCourse.Chapters[0].Key
	}
	if chapterKey == "" {
		chapterKey = "question-bank"
	}
	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	requirements := fmt.Sprintf("只生成 quiz_game，包含 %d 道%s选择题。%s", input.Count, input.Difficulty, input.ExtraRequirements)
	activity, meta, err := aiService.GenerateActivity(ctx, ai.GenerateActivityInput{
		Topic: row.Title, Audience: shownAudience(row.Audience), Language: shownLanguage(row.Language),
		ExtraRequirements: requirements, TargetChapterKey: chapterKey,
		TargetActivityKey: fmt.Sprintf("%s-question-bank-%d", chapterKey, time.Now().UTC().Unix()),
		ContextCourse:     optionalCourseContext(row.ContentJSON, &contextCourse),
	})
	service.recordUsage(ctx, principal.UserID, usage.SceneGenerateActivity, meta, err)
	if err != nil {
		return nil, businessAIError(err)
	}
	questions := activityQuizQuestions(activity)
	items := make([]TeacherQuestionInput, 0, len(questions))
	for _, question := range questions {
		items = append(items, TeacherQuestionInput{Content: question.Question, Options: question.Options,
			CorrectAnswer: question.Options[question.CorrectIndex], Explanation: question.Explanation, Difficulty: input.Difficulty})
	}
	return &AIQuestionPreview{Questions: items, Meta: meta}, nil
}

func (service *Service) requireTeacherCourse(ctx context.Context, principal *access.Principal, courseID int64) error {
	if principal.Role != access.RoleTeacher {
		return errs.Forbidden("only teachers can manage questions")
	}
	var count int64
	if err := service.database.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM course WHERE id = ? AND owner_id = ?", courseID, principal.UserID,
	).Scan(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errs.NotFound("course not found")
	}
	return nil
}

func normalizeTeacherQuestion(input TeacherQuestionInput) TeacherQuestionInput {
	input.Content = strings.TrimSpace(input.Content)
	input.CorrectAnswer = strings.TrimSpace(input.CorrectAnswer)
	input.Explanation = strings.TrimSpace(input.Explanation)
	input.Difficulty = strings.ToLower(strings.TrimSpace(input.Difficulty))
	options := make([]string, 0, len(input.Options))
	for _, option := range input.Options {
		if trimmed := strings.TrimSpace(option); trimmed != "" {
			options = append(options, trimmed)
		}
	}
	input.Options = options
	return input
}

func validateTeacherQuestion(input TeacherQuestionInput) error {
	if input.Content == "" || utf8.RuneCountInString(input.Content) > 2000 {
		return errs.BadRequest("question content is required and must be at most 2000 characters")
	}
	if len(input.Options) < 2 || len(input.Options) > 8 {
		return errs.BadRequest("a choice question must have 2 to 8 options")
	}
	seen := map[string]struct{}{}
	for _, option := range input.Options {
		if _, exists := seen[option]; exists {
			return errs.BadRequest("question options must be unique")
		}
		seen[option] = struct{}{}
	}
	if !choiceIncluded(input.Options, input.CorrectAnswer) {
		return errs.BadRequest("correct answer must be one of the options")
	}
	if !allowedDifficulty(input.Difficulty) {
		return errs.BadRequest("difficulty must be easy, medium, or hard")
	}
	return nil
}

func validateImportRows(rows []ImportQuestionRow) *QuestionImportPreview {
	preview := &QuestionImportPreview{Rows: []ImportQuestionRow{}, Errors: []QuestionImportError{}}
	if len(rows) == 0 {
		preview.Errors = append(preview.Errors, QuestionImportError{Row: 0, Field: "file", Message: "没有可导入的数据行"})
		return preview
	}
	if len(rows) > 1000 {
		preview.Errors = append(preview.Errors, QuestionImportError{Row: 0, Field: "file", Message: "单次最多导入 1000 道题"})
		return preview
	}
	for index, row := range rows {
		if row.Row <= 0 {
			row.Row = index + 2
		}
		normalized := normalizeTeacherQuestion(TeacherQuestionInput{
			Content: row.Content, Options: row.Options, CorrectAnswer: row.CorrectAnswer,
			Explanation: row.Explanation, Difficulty: row.Difficulty,
		})
		row.Content, row.Options, row.CorrectAnswer = normalized.Content, normalized.Options, normalized.CorrectAnswer
		row.Explanation, row.Difficulty = normalized.Explanation, normalized.Difficulty
		if err := validateTeacherQuestion(normalized); err != nil {
			preview.Errors = append(preview.Errors, QuestionImportError{Row: row.Row, Field: "question", Message: err.Error()})
			continue
		}
		preview.Rows = append(preview.Rows, row)
	}
	preview.ValidCount = len(preview.Rows)
	return preview
}

func commonDifficulty(rows []ImportQuestionRow) string {
	if len(rows) == 0 {
		return "medium"
	}
	difficulty := rows[0].Difficulty
	for _, row := range rows[1:] {
		if row.Difficulty != difficulty {
			return "medium"
		}
	}
	return difficulty
}

func findSectionQuizID(tx *gorm.DB, courseID int64, chapterKey, sectionKey string) (int64, error) {
	var setID int64
	found := tx.Raw(
		`SELECT id FROM course_question WHERE course_id = ? AND chapter_key = ? AND section_key = ?`,
		courseID, chapterKey, sectionKey,
	).Scan(&setID)
	if found.Error != nil {
		return 0, found.Error
	}
	if found.RowsAffected == 0 || setID == 0 {
		return 0, errs.BadRequest("save the course after marking this section as a quiz")
	}
	return setID, nil
}

func deletePlaceholderQuestions(database *gorm.DB) error {
	if err := database.Exec(
		`DELETE FROM course_question_content
		 WHERE TRIM(content) IN ('（待生成）', '待生成', '示例题目')
		    OR TRIM(content) LIKE '请填写%'`,
	).Error; err != nil {
		return err
	}
	return database.Exec(
		`UPDATE course_question
		 SET question_count = (
		   SELECT COUNT(*) FROM course_question_content qc WHERE qc.question_id = course_question.id
		 )`,
	).Error
}

func appendQuizContent(tx *gorm.DB, quizID int64, content string, options []string, correct, explanation string, now time.Time) (int64, error) {
	if err := tx.Exec(
		`DELETE FROM course_question_content
		 WHERE question_id = ?
		   AND (TRIM(content) IN ('（待生成）', '待生成', '示例题目') OR TRIM(content) LIKE '请填写%')`,
		quizID,
	).Error; err != nil {
		return 0, err
	}
	var sortOrder int
	if err := tx.Raw(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM course_question_content WHERE question_id = ?`, quizID).Scan(&sortOrder).Error; err != nil {
		return 0, err
	}
	if err := tx.Exec(
		`INSERT INTO course_question_content
		 (question_id, content, options, correct_answer, explanation, question_type, sort_order, score, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		quizID, content, strings.Join(options, "|"), correct, explanation, QuestionTypeSingleChoice, sortOrder, now, now,
	).Error; err != nil {
		return 0, err
	}
	var contentID int64
	if err := tx.Raw(`SELECT id FROM course_question_content WHERE question_id = ? ORDER BY id DESC LIMIT 1`, quizID).Scan(&contentID).Error; err != nil {
		return 0, err
	}
	return contentID, nil
}

func publishQuizWithQuestions(tx *gorm.DB, quizID int64, now time.Time) error {
	return tx.Exec(
		`UPDATE course_question
		 SET question_count = (SELECT COUNT(*) FROM course_question_content WHERE question_id = ?),
		     is_publish = ?, publish_status = ?, published_at = COALESCE(published_at, ?), updated_at = ?
		 WHERE id = ?`,
		quizID, PublishOn, PublishStatusPublished, now, now, quizID,
	).Error
}

func optionalCourseContext(raw string, parsed *aicourse.StandardCourse) *aicourse.StandardCourse {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return parsed
}

func shownAudience(value string) string {
	if value == aicourse.AudienceCS || value == aicourse.AudienceNonCS || value == aicourse.AudienceGeneral {
		return value
	}
	return aicourse.AudienceGeneral
}
