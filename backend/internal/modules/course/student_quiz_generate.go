package course

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/liuxuancy/Libs/logs"
	"gorm.io/gorm"

	"caveclass/internal/ai"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

type GenerateStudentQuizInput struct {
	ChapterKey string `json:"chapterKey"`
	Count      int    `json:"count"`
	Difficulty string `json:"difficulty"`
	Provider   string `json:"provider"`
}

type quizExcerpt struct {
	Title string   `json:"title"`
	Body  []string `json:"body"`
}

func quizCourseContext(contentJSON, chapterKey string) (string, string, error) {
	var content aicourse.StandardCourse
	if strings.TrimSpace(contentJSON) == "" {
		return "", "", errs.BadRequest("generate course content before creating a quiz")
	}
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return "", "", errs.Internal(err)
	}
	selected := make([]aicourse.Chapter, 0)
	for _, chapter := range content.Chapters {
		if chapterKey == "" || chapter.Key == chapterKey {
			selected = append(selected, chapter)
		}
	}
	if len(selected) == 0 {
		return "", "", errs.BadRequest("selected chapter has no course content")
	}
	excerpts := make([]quizExcerpt, 0, len(selected))
	for _, chapter := range selected {
		excerpt := quizExcerpt{Title: chapter.Title, Body: []string{}}
		for _, activity := range chapter.Activities {
			if activity.Type == aicourse.ActivityTypeQuizGame {
				continue
			}
			body := strings.TrimSpace(activityNote(activity))
			if body == "" || body == outlineContentPlaceholder || body == "待生成" || strings.HasPrefix(body, "请填写") {
				continue
			}
			excerpt.Body = append(excerpt.Body, body)
		}
		if len(excerpt.Body) == 0 {
			return "", "", errs.BadRequest("selected course or chapter has no learning content")
		}
		excerpts = append(excerpts, excerpt)
	}
	// Each chapter retains a representative excerpt; shrink uniformly to fit the model budget.
	for _, limit := range []int{2000, 1000, 500, 250, 100, 40} {
		bounded := make([]quizExcerpt, len(excerpts))
		for i, excerpt := range excerpts {
			bounded[i].Title = excerpt.Title
			budget := limit
			for _, body := range excerpt.Body {
				if budget <= 0 {
					break
				}
				runes := []rune(body)
				if len(runes) > budget {
					runes = runes[:budget]
				}
				bounded[i].Body = append(bounded[i].Body, string(runes))
				budget -= len(runes)
			}
		}
		encoded, err := json.Marshal(bounded)
		if err != nil {
			return "", "", err
		}
		if len(encoded) <= 22000 {
			title := ""
			if chapterKey != "" {
				title = selected[0].Title
			}
			return string(encoded), title, nil
		}
	}
	return "", "", errs.BadRequest("course content exceeds the model input limit")
}

func (service *Service) GenerateStudentQuiz(ctx context.Context, principal *access.Principal, courseID int64, input GenerateStudentQuizInput) (*QuestionListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can create questions")
	}
	input.ChapterKey = strings.TrimSpace(input.ChapterKey)
	if !allowedQuestionCount(input.Count) || !allowedDifficulty(input.Difficulty) || utf8.RuneCountInString(input.ChapterKey) > 128 {
		return nil, errs.BadRequest("invalid quiz settings")
	}
	load := func(db *gorm.DB) (storedCourse, error) {
		var row storedCourse
		found := db.Raw(`SELECT c.id, c.title, COALESCE(c.language, '') AS language, COALESCE(c.content_json, '') AS content_json
			FROM course c WHERE c.id = ? AND `+studentVisibleCourseSQL("c"),
			append([]any{courseID}, studentVisibleCourseArgs(principal.UserID)...)...).Scan(&row)
		if found.Error != nil {
			return row, found.Error
		}
		if found.RowsAffected == 0 {
			return row, errs.NotFound("course not found")
		}
		return row, nil
	}
	row, err := load(service.database.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	contextJSON, chapterTitle, err := quizCourseContext(row.ContentJSON, input.ChapterKey)
	if err != nil {
		return nil, err
	}
	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	questions, meta, callErr := aiService.GenerateStudentQuiz(ctx, contextJSON, row.Language, input.Difficulty, input.Count)
	service.recordUsage(ctx, principal.UserID, usage.SceneStudentQuiz, meta, callErr)
	if callErr != nil {
		ai.LogError(ctx, callErr, meta, principal.UserID)
		return nil, businessAIError(callErr)
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return nil, errs.Internal(err)
	}
	now := time.Now().UTC()
	suffix := " · " + input.Difficulty + " · " + now.Format("2006-01-02 15:04:05") + " · " + hex.EncodeToString(random)
	titleRunes := []rune(row.Title)
	if len(titleRunes) > 200-len([]rune(suffix)) {
		titleRunes = titleRunes[:200-len([]rune(suffix))]
	}
	title := string(titleRunes) + suffix
	var question CourseQuestion
	err = service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := load(tx)
		if err != nil {
			return err
		}
		if current.ContentJSON != row.ContentJSON {
			return errs.BadRequest("course content changed; please retry")
		}
		question = CourseQuestion{Title: title, CourseID: courseID, CreatorID: principal.UserID, QuestionCount: len(questions), Difficulty: input.Difficulty, IsPublish: PublishOff, CreationMethod: CreationMethodAI, PublishStatus: PublishStatusDraft, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&question).Error; err != nil {
			return err
		}
		for i, item := range questions {
			content := CourseQuestionContent{QuestionID: question.ID, Content: item.Question, Options: strings.Join(item.Options, "|"), CorrectAnswer: item.Options[item.CorrectIndex], Explanation: item.Explanation, QuestionType: QuestionTypeSingleChoice, SortOrder: i + 1, Score: 1, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&content).Error; err != nil {
				return err
			}
		}
		if input.ChapterKey != "" {
			return tx.Create(&StudentQuizChapter{QuestionID: question.ID, ChapterKey: input.ChapterKey, ChapterTitle: chapterTitle}).Error
		}
		return nil
	})
	if err != nil {
		logs.CtxError(ctx, "[GenerateStudentQuiz] save failed: %v", err)
		return nil, err
	}
	return &QuestionListItem{ID: question.ID, Title: title, CourseID: courseID, CourseTitle: row.Title, Difficulty: input.Difficulty, QuestionCount: len(questions), ChapterKey: input.ChapterKey, ChapterName: chapterTitle, CreationMethod: CreationMethodAI, CreatorID: principal.UserID, CreatorRole: principal.Role, PublishStatus: PublishStatusDraft}, nil
}
