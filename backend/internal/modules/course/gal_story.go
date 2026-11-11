package course

import (
	"context"
	"sort"
	"strings"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/gal"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

// GalScriptInput 是生成 Gal 剧本的请求体。
type GalScriptInput struct {
	Provider    string `json:"provider"`
	ActivityKey string `json:"activityKey"`
	DetailLevel string `json:"detailLevel"`
	Personality string `json:"personality"`
}

// GalScriptResult 是 Gal 剧本生成结果。
type GalScriptResult struct {
	Title          string           `json:"title"`
	ChapterKey     string           `json:"chapterKey"`
	NextChapterKey string           `json:"nextChapterKey,omitempty"`
	Script         []gal.ScriptItem `json:"script"`
	Meta           ai.Meta          `json:"meta"`
}

// GenerateGalScript 为学生可见课程的一章生成 Gal 剧本。
func (service *Service) GenerateGalScript(ctx context.Context, principal *access.Principal, courseID int64, chapterKey string, input GalScriptInput) (*GalScriptResult, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can generate gal scripts")
	}
	var row storedCourse
	found := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title, COALESCE(c.language, '') AS language, COALESCE(c.content_json, '') AS content_json
		 FROM course c WHERE c.id = ? AND `+studentVisibleCourseSQL("c"),
		append([]any{courseID}, studentVisibleCourseArgs(principal.UserID)...)...,
	).Scan(&row)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}
	content, err := parseStoredContent(row.ContentJSON)
	if err != nil {
		return nil, err
	}
	chapterKey = strings.TrimSpace(chapterKey)
	var chapter *aicourse.Chapter
	for index := range content.Chapters {
		if content.Chapters[index].Key == chapterKey {
			chapter = &content.Chapters[index]
			break
		}
	}
	if chapter == nil {
		return nil, errs.NotFound("chapter not found")
	}
	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	script, meta, callErr := aiService.GenerateGalScript(ctx, *chapter, shownLanguage(row.Language), ai.GalScriptInput{
		CourseTitle: row.Title,
		DetailLevel: input.DetailLevel,
		Personality: input.Personality,
		ActivityKey: input.ActivityKey,
	})
	service.recordUsage(ctx, principal.UserID, usage.SceneGalStory, meta, callErr)
	if callErr != nil {
		ai.LogError(ctx, callErr, meta, principal.UserID)
		if aiError, ok := aierr.As(callErr); ok && aiError.Code == aierr.CodeInputTooLong {
			return nil, errs.BadRequest("当前章节内容过长，无法生成故事模式。请换用简略深度或缩小章节后再试。")
		}
		return nil, businessAIError(callErr)
	}
	courseRef := courseID
	chapterRef := chapterKey
	service.recordLearningEvent(ctx, principal.UserID, EventStorySession, &courseRef, nil, chapterRef, 0)
	return &GalScriptResult{
		Title:          script.Title,
		ChapterKey:     chapterKey,
		NextChapterKey: nextChapterKey(content, chapterKey),
		Script:         script.Script,
		Meta:           meta,
	}, nil
}

func nextChapterKey(content aicourse.StandardCourse, currentKey string) string {
	chapters := append([]aicourse.Chapter(nil), content.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool {
		if chapters[i].Order != chapters[j].Order {
			return chapters[i].Order < chapters[j].Order
		}
		return chapters[i].Key < chapters[j].Key
	})
	for index, chapter := range chapters {
		if chapter.Key == currentKey && index+1 < len(chapters) {
			return chapters[index+1].Key
		}
	}
	return ""
}
