package course

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai/quality"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/usage"
)

type ChapterQAMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChapterQAInput struct {
	Provider       string             `json:"provider"`
	Question       string             `json:"question"`
	ActivityKey    string             `json:"activityKey"`
	Selection      string             `json:"selection"`
	ConversationID int64              `json:"conversationId"`
	History        []ChapterQAMessage `json:"history"`
}

type ChapterQAResult struct {
	Answer         string  `json:"answer"`
	ConversationID int64   `json:"conversationId"`
	Meta           ai.Meta `json:"meta"`
}

type QualityReportInput struct {
	Provider string `json:"provider"`
	Rubric   string `json:"rubric"`
}

type StoredQualityReport struct {
	Report    quality.Report `json:"report"`
	Meta      *ai.Meta       `json:"meta,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type MermaidResult struct {
	Source string `json:"source"`
}

// AskChapterQuestion 先按学生可见性取课，再严格截取目标单章作为唯一课程上下文。
func (service *Service) AskChapterQuestion(ctx context.Context, principal *access.Principal, courseID int64, chapterKey string, input ChapterQAInput) (*ChapterQAResult, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can ask chapter questions")
	}
	var row storedCourse
	found := service.database.WithContext(ctx).Raw(
		`SELECT c.id, COALESCE(c.language, '') AS language, COALESCE(c.content_json, '') AS content_json
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
	var chapter *aicourse.Chapter
	for index := range content.Chapters {
		if content.Chapters[index].Key == strings.TrimSpace(chapterKey) {
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
	history, err := service.recentConversationHistory(ctx, principal.UserID, courseID, input.ConversationID)
	if err != nil {
		return nil, err
	}
	activityKey, sectionTitle := sectionTitleFor(chapter, input.ActivityKey)
	answer, meta, callErr := aiService.AskChapter(ctx, *chapter, shownLanguage(row.Language), ai.ChapterQuestion{
		Question:    input.Question,
		ActivityKey: activityKey,
		Selection:   input.Selection,
		History:     history,
	})
	service.recordUsage(ctx, principal.UserID, usage.SceneChapterQA, meta, callErr)
	if callErr != nil {
		ai.LogError(ctx, callErr, meta, principal.UserID)
		if aiError, ok := aierr.As(callErr); ok && aiError.Code == aierr.CodeInputTooLong {
			return nil, errs.BadRequest("当前小节内容和对话历史超过了模型输入限制。请缩短问题，或新开对话后再试。")
		}
		return nil, businessAIError(callErr)
	}
	conversationID, err := service.saveChapterExchange(
		ctx, principal.UserID, courseID, input.ConversationID,
		strings.TrimSpace(chapterKey), activityKey, sectionTitle, strings.TrimSpace(input.Question), answer,
	)
	if err != nil {
		return nil, err
	}
	courseRef := courseID
	chapterRef := strings.TrimSpace(chapterKey)
	service.recordLearningEvent(ctx, principal.UserID, EventChapterQA, &courseRef, nil, chapterRef, 0)
	return &ChapterQAResult{Answer: answer, ConversationID: conversationID, Meta: meta}, nil
}

func (service *Service) GetCourseQualityReport(ctx context.Context, principal *access.Principal, courseID int64) (*StoredQualityReport, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	var row CourseQualityReport
	found := service.database.WithContext(ctx).Raw(
		"SELECT course_id, overall_score, report_json, created_at FROM course_quality_report WHERE course_id = ?", courseID,
	).Scan(&row)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("quality report not found")
	}
	var report quality.Report
	if err := json.Unmarshal([]byte(row.ReportJSON), &report); err != nil {
		return nil, errs.Internal(err)
	}
	return &StoredQualityReport{Report: report, CreatedAt: row.CreatedAt}, nil
}

func (service *Service) GenerateCourseQualityReport(ctx context.Context, principal *access.Principal, courseID int64, input QualityReportInput) (*StoredQualityReport, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	var raw string
	found := service.database.WithContext(ctx).Raw(
		"SELECT COALESCE(content_json, '') FROM course WHERE id = ? AND owner_id = ?", courseID, principal.UserID,
	).Scan(&raw)
	if found.Error != nil {
		return nil, found.Error
	}
	content, err := parseStoredContent(raw)
	if err != nil {
		return nil, err
	}
	aiService, err := service.aiForGenerate(input.Provider)
	if err != nil {
		return nil, err
	}
	report, meta, callErr := aiService.EvaluateCourse(ctx, content, input.Rubric)
	service.recordUsage(ctx, principal.UserID, usage.SceneEvaluateCourse, meta, callErr)
	if callErr != nil {
		ai.LogError(ctx, callErr, meta, principal.UserID)
		return nil, businessAIError(callErr)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	createdAt := time.Now().UTC()
	err = service.database.WithContext(ctx).Exec(
		`INSERT INTO course_quality_report (course_id, overall_score, report_json, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(course_id) DO UPDATE SET overall_score = excluded.overall_score,
		 report_json = excluded.report_json, created_at = excluded.created_at`,
		courseID, report.OverallScore, string(encoded), createdAt,
	).Error
	if err != nil {
		return nil, err
	}
	return &StoredQualityReport{Report: report, Meta: &meta, CreatedAt: createdAt}, nil
}

func (service *Service) GetCourseMermaid(ctx context.Context, principal *access.Principal, courseID int64) (*MermaidResult, error) {
	if err := service.requireTeacherCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	var raw string
	if err := service.database.WithContext(ctx).Raw(
		"SELECT COALESCE(content_json, '') FROM course WHERE id = ? AND owner_id = ?", courseID, principal.UserID,
	).Scan(&raw).Error; err != nil {
		return nil, err
	}
	content, err := parseStoredContent(raw)
	if err != nil {
		return nil, err
	}
	return &MermaidResult{Source: courseMermaid(content)}, nil
}

func parseStoredContent(raw string) (aicourse.StandardCourse, error) {
	if strings.TrimSpace(raw) == "" {
		return aicourse.StandardCourse{}, errs.BadRequest("course has no generated content")
	}
	var content aicourse.StandardCourse
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return aicourse.StandardCourse{}, errs.Internal(err)
	}
	return content, nil
}

func courseMermaid(content aicourse.StandardCourse) string {
	var lines []string
	lines = append(lines, "flowchart TD", fmt.Sprintf(`  course["%s"]`, mermaidLabel(content.Title)))
	chapters := append([]aicourse.Chapter(nil), content.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool { return chapters[i].Order < chapters[j].Order })
	for chapterIndex, chapter := range chapters {
		chapterID := fmt.Sprintf("chapter%d", chapterIndex)
		lines = append(lines, fmt.Sprintf(`  %s["%s"]`, chapterID, mermaidLabel(chapter.Title)))
		lines = append(lines, "  course --> "+chapterID)
		activities := append([]aicourse.Activity(nil), chapter.Activities...)
		sort.SliceStable(activities, func(i, j int) bool { return activities[i].Order < activities[j].Order })
		for activityIndex, activity := range activities {
			activityID := fmt.Sprintf("%s_activity%d", chapterID, activityIndex)
			lines = append(lines, fmt.Sprintf(`  %s["%s · %s"]`, activityID, mermaidLabel(activity.Title), mermaidLabel(activity.Type)))
			lines = append(lines, fmt.Sprintf("  %s --> %s", chapterID, activityID))
		}
	}
	return strings.Join(lines, "\n")
}

func mermaidLabel(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\r", " ", "\n", " ")
	return replacer.Replace(strings.TrimSpace(value))
}
