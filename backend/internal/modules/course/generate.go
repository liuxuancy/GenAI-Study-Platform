package course

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
	"caveclass/internal/usage"
)

// GenerateCourseInput 是学生或教师发起的整课生成。主题空着时用课程名。
type GenerateCourseInput struct {
	Provider          string `json:"provider"`
	Topic             string `json:"topic"`
	Audience          string `json:"audience"`
	Language          string `json:"language"`
	ExtraRequirements string `json:"extraRequirements"`
	ChapterCountHint  int    `json:"chapterCountHint"`
}

// GeneratedCourse 把课程详情和这次生成的元数据分开。元数据不是课程正文。
type GeneratedCourse struct {
	Course any     `json:"course"`
	Meta   ai.Meta `json:"meta"`
}

// GenerateOwnedCourse 调用已有的 GenerateCourse，写入 content_json，并把每章测验落成套题。
func (service *Service) GenerateOwnedCourse(ctx context.Context, principal *access.Principal, courseID int64, input GenerateCourseInput) (*GeneratedCourse, error) {
	if principal.Role != access.RoleStudent && principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only the course owner can generate a course")
	}
	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	if aiService == nil {
		return nil, errs.Internal(nil)
	}
	var stored struct {
		Title string
	}
	found := service.database.WithContext(ctx).Raw(
		"SELECT title FROM course WHERE id = ? AND owner_id = ?",
		courseID, principal.UserID,
	).Scan(&stored)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}

	topic := strings.TrimSpace(input.Topic)
	if topic == "" {
		topic = stored.Title
	}
	audience := strings.TrimSpace(input.Audience)
	if audience == "" {
		audience = aicourse.AudienceGeneral
	}
	language := strings.TrimSpace(input.Language)
	if language == "" {
		language = DefaultLanguage
	}

	generated, meta, err := aiService.GenerateCourse(ctx, ai.GenerateCourseInput{
		Topic:             topic,
		Audience:          audience,
		Language:          language,
		ExtraRequirements: input.ExtraRequirements,
		ChapterCountHint:  input.ChapterCountHint,
	})
	service.recordUsage(ctx, principal.UserID, usage.SceneGenerateCourse, meta, err)
	if err != nil {
		ai.LogError(ctx, err, meta, principal.UserID)
		return nil, businessAIError(err)
	}

	if err := service.fillMissingQuizzes(ctx, principal.UserID, topic, &generated, &meta, aiService); err != nil {
		return nil, err
	}
	if err := service.storeGeneratedCourse(ctx, principal, courseID, &generated); err != nil {
		return nil, err
	}

	var detail any
	if principal.Role == access.RoleTeacher {
		detail, err = service.GetTeacherCourse(ctx, principal, courseID)
	} else {
		detail, err = service.GetCourse(ctx, principal, courseID)
	}
	if err != nil {
		return nil, err
	}
	return &GeneratedCourse{Course: detail, Meta: meta}, nil
}

func (service *Service) fillMissingQuizzes(ctx context.Context, userID int64, topic string, generated *aicourse.StandardCourse, meta *ai.Meta, aiService *ai.Service) error {
	for index := range generated.Chapters {
		chapter := &generated.Chapters[index]
		if len(chapterQuizQuestions(*chapter)) > 0 {
			continue
		}
		activityKey := chapter.Key + "-quiz"
		activity, activityMeta, err := aiService.GenerateActivity(ctx, ai.GenerateActivityInput{
			Topic:             topic,
			Audience:          generated.Audience,
			Language:          generated.Language,
			ExtraRequirements: "生成章节测验，type 必须是 quiz_game，2 到 10 道选择题。",
			TargetChapterKey:  chapter.Key,
			TargetActivityKey: activityKey,
			ContextCourse:     generated,
		})
		service.recordUsage(ctx, userID, usage.SceneGenerateActivity, activityMeta, err)
		if err != nil {
			ai.LogError(ctx, err, activityMeta, userID)
			return businessAIError(err)
		}
		if len(activityQuizQuestions(activity)) == 0 {
			meta.Warnings = append(meta.Warnings, "chapter "+chapter.Key+" has no quiz")
			continue
		}
		activity.Order = len(chapter.Activities) + 1
		chapter.Activities = append(chapter.Activities, activity)
	}
	return nil
}

func (service *Service) storeGeneratedCourse(ctx context.Context, principal *access.Principal, courseID int64, generated *aicourse.StandardCourse) error {
	encoded, err := json.Marshal(generated)
	if err != nil {
		return err
	}
	publish := PublishOff
	if principal.Role == access.RoleTeacher {
		publish = PublishOn
	}
	return service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated := tx.Exec(
			`UPDATE course
			 SET description = ?, audience = ?, language = ?, content_json = ?, updated_at = ?
			 WHERE id = ? AND owner_id = ?`,
			nullIfEmpty(generated.Description), generated.Audience, generated.Language, string(encoded), time.Now().UTC(),
			courseID, principal.UserID,
		)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return errs.NotFound("course not found")
		}
		if err := deleteChapterQuizzes(tx, courseID); err != nil {
			return err
		}
		for _, chapter := range generated.Chapters {
			questions := chapterQuizQuestions(chapter)
			if len(questions) == 0 {
				continue
			}
			if len(questions) > 10 {
				questions = questions[:10]
			}
			if err := insertChapterQuiz(tx, courseID, principal.UserID, chapter, questions, publish); err != nil {
				return err
			}
		}
		return nil
	})
}

func chapterQuizQuestions(chapter aicourse.Chapter) []aicourse.QuizQuestion {
	for _, activity := range chapter.Activities {
		questions := activityQuizQuestions(activity)
		if len(questions) > 0 {
			return questions
		}
	}
	return nil
}

func activityQuizQuestions(activity aicourse.Activity) []aicourse.QuizQuestion {
	if activity.Type != aicourse.ActivityTypeQuizGame {
		return nil
	}
	var payload aicourse.QuizGamePayload
	if json.Unmarshal(activity.Payload, &payload) != nil {
		return nil
	}
	kept := make([]aicourse.QuizQuestion, 0, len(payload.Questions))
	for _, question := range payload.Questions {
		if len(question.Options) < 2 || question.CorrectIndex < 0 || question.CorrectIndex >= len(question.Options) {
			continue
		}
		if strings.TrimSpace(question.Question) == "" {
			continue
		}
		kept = append(kept, question)
	}
	return kept
}

func deleteChapterQuizzes(tx *gorm.DB, courseID int64) error {
	var ids []int64
	err := tx.Raw(
		`SELECT id FROM course_question
		 WHERE course_id = ? AND chapter_key IS NOT NULL AND chapter_key <> ''`,
		courseID,
	).Scan(&ids).Error
	if err != nil {
		return err
	}
	for _, questionID := range ids {
		for _, statement := range []string{
			"DELETE FROM student_question_progress WHERE question_id = ?",
			"DELETE FROM student_question_record WHERE question_id = ?",
			"DELETE FROM course_question_content WHERE question_id = ?",
			"DELETE FROM course_question WHERE id = ?",
		} {
			if err := tx.Exec(statement, questionID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func insertChapterQuiz(tx *gorm.DB, courseID int64, creatorID int64, chapter aicourse.Chapter, questions []aicourse.QuizQuestion, isPublish int) error {
	title := chapterQuizTitle(courseID, chapter.Key, time.Now().UTC())
	result := tx.Exec(
		`INSERT INTO course_question
		 (title, course_id, creator_id, question_count, is_publish, difficulty, chapter_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		title, courseID, creatorID, len(questions), isPublish, "medium", chapter.Key,
	)
	if result.Error != nil {
		return result.Error
	}
	var questionID int64
	found := tx.Raw("SELECT id FROM course_question WHERE title = ?", title).Scan(&questionID)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		return errs.NotFound("question not found")
	}
	for _, question := range questions {
		inserted := tx.Exec(
			`INSERT INTO course_question_content
			 (question_id, content, options, correct_answer, explanation)
			 VALUES (?, ?, ?, ?, ?)`,
			questionID, question.Question, strings.Join(question.Options, "|"), question.Options[question.CorrectIndex], question.Explanation,
		)
		if inserted.Error != nil {
			return inserted.Error
		}
	}
	return nil
}

func chapterQuizTitle(courseID int64, chapterKey string, now time.Time) string {
	suffix := " · " + chapterKey + " · " + strconv.FormatInt(courseID, 10) + " · " + now.Format("2006-01-02 15:04:05")
	runes := []rune("Quiz")
	room := 200 - len([]rune(suffix))
	if room < 1 {
		room = 1
	}
	if len(runes) > room {
		runes = runes[:room]
	}
	return string(runes) + suffix
}

func (service *Service) recordUsage(ctx context.Context, userID int64, scene string, meta ai.Meta, callErr error) {
	status := usage.StatusSuccess
	if callErr != nil {
		status = usage.StatusFailed
	}
	err := service.database.WithContext(ctx).Exec(
		`INSERT INTO ai_usage
		 (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, scene, meta.ModelUsed, meta.Usage.PromptTokens, meta.Usage.CompletionTokens, meta.Usage.TotalTokens,
		status, time.Now().UTC(),
	).Error
	if err != nil {
		logs.CtxError(ctx, "record ai usage: %v", err)
	}
}

func businessAIError(err error) error {
	aiError, ok := aierr.As(err)
	if !ok {
		return err
	}
	if aiError.Status() == errs.SystemBadRequest {
		return errs.BadRequest(aiError.Msg)
	}
	return errs.Internal(err)
}

func (service *Service) aiForGenerate(requestedProvider string) (*ai.Service, error) {
	base := service.aiBase
	if base.Provider == "" && service.ai != nil {
		base = service.ai.Config()
	}
	provider := strings.ToLower(strings.TrimSpace(requestedProvider))
	if provider == "" {
		provider = llm.DefaultGenerateProvider(base)
	}
	switch provider {
	case llm.ProviderDeepSeek, llm.ProviderTeamily:
	case llm.ProviderMock:
		if base.Provider == llm.ProviderMock {
			break
		}
		return nil, errs.BadRequest("provider must be deepseek or teamily")
	default:
		return nil, errs.BadRequest("provider must be deepseek or teamily")
	}
	cfg, err := llm.ConfigForProvider(base, provider)
	if err != nil {
		return nil, errs.BadRequest(err.Error())
	}
	if cfg.Provider != llm.ProviderMock && strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errs.BadRequest("AI provider " + provider + " is not configured on the server")
	}
	if service.ai != nil && service.ai.Config().Provider == cfg.Provider && service.ai.Config().APIKey == cfg.APIKey {
		return service.ai, nil
	}
	return ai.NewForProvider(base, provider)
}

// AIProviderOptions 返回前端可选的生成模型与默认项。
func (service *Service) AIProviderOptions() llm.GenerateProviderOptions {
	base := service.aiBase
	if base.Provider == "" && service.ai != nil {
		base = service.ai.Config()
	}
	return ai.GenerateProviderOptions(base)
}
