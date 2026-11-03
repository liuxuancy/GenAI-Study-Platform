package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	aicourse "caveclass/internal/ai/course"
)

// activityContextBudget 为 prompt 模板留出空间，避免 Teamily input 超过 32000。
const activityContextBudget = 24000

// qualityContextBudget 供 content_quality_v1 嵌入课程 JSON 时使用（整段 prompt 仍须低于 Teamily 32000）。
const qualityContextBudget = 23500

// marshalCourseForQuality 截断各小节正文，保证整课 JSON 适合 Teamily 等 input 上限。
func marshalCourseForQuality(full aicourse.StandardCourse) (string, error) {
	markdownLimits := []int{2500, 1500, 800, 400, 200, 80, 0}
	for _, markdownLimit := range markdownLimits {
		if text, ok := encodeQualityCourse(slimCourseForQuality(full, markdownLimit)); ok {
			return text, nil
		}
	}
	if text, ok := encodeQualityCourse(slimCourseForQualityOutline(full, 120)); ok {
		return text, nil
	}
	if text, ok := encodeQualityCourse(slimCourseStructureOnly(full)); ok {
		return text, nil
	}
	for chapters := len(full.Chapters); chapters >= 1; chapters-- {
		trimmed := full
		trimmed.Chapters = append([]aicourse.Chapter(nil), full.Chapters[:chapters]...)
		if text, ok := encodeQualityCourse(slimCourseStructureOnly(trimmed)); ok {
			return text, nil
		}
	}
	return "", aierr.InputTooLong(nil, fmt.Sprintf("course context exceeds %d bytes after truncation", qualityContextBudget))
}

func encodeQualityCourse(course aicourse.StandardCourse) (string, bool) {
	encoded, err := json.Marshal(course)
	if err != nil {
		return "", false
	}
	if len(encoded) > qualityContextBudget {
		return "", false
	}
	return string(encoded), true
}

func slimCourseForQualityOutline(full aicourse.StandardCourse, excerptRunes int) aicourse.StandardCourse {
	chapters := make([]aicourse.Chapter, len(full.Chapters))
	for index, chapter := range full.Chapters {
		chapters[index] = trimChapterPayloadsOutline(chapter, excerptRunes)
	}
	return aicourse.StandardCourse{
		SchemaVersion:    full.SchemaVersion,
		Title:            full.Title,
		Description:      truncateRunes(strings.TrimSpace(full.Description), 300),
		Audience:         full.Audience,
		Language:         full.Language,
		EstimatedMinutes: full.EstimatedMinutes,
		Chapters:         chapters,
	}
}

func slimCourseStructureOnly(full aicourse.StandardCourse) aicourse.StandardCourse {
	chapters := make([]aicourse.Chapter, len(full.Chapters))
	for index, chapter := range full.Chapters {
		trimmed := chapter
		if len(trimmed.Summary) > 0 {
			trimmed.Summary = truncateRunes(trimmed.Summary, 200)
		}
		trimmed.Activities = make([]aicourse.Activity, len(chapter.Activities))
		for actIndex, activity := range chapter.Activities {
			payload, _ := json.Marshal(aicourse.RichTextPayload{
				Markdown:   truncateRunes(activity.Title, 80),
				Highlights: []string{},
			})
			trimmed.Activities[actIndex] = aicourse.Activity{
				Key:     activity.Key,
				Type:    activity.Type,
				Title:   truncateRunes(activity.Title, 120),
				Order:   activity.Order,
				Payload: payload,
			}
		}
		chapters[index] = trimmed
	}
	return aicourse.StandardCourse{
		SchemaVersion:    full.SchemaVersion,
		Title:            truncateRunes(full.Title, 160),
		Description:      truncateRunes(strings.TrimSpace(full.Description), 200),
		Audience:         full.Audience,
		Language:         full.Language,
		EstimatedMinutes: full.EstimatedMinutes,
		Chapters:         chapters,
	}
}

func trimChapterPayloadsOutline(chapter aicourse.Chapter, excerptRunes int) aicourse.Chapter {
	trimmed := chapter
	if len(trimmed.Summary) > 0 {
		trimmed.Summary = truncateRunes(trimmed.Summary, 200)
	}
	trimmed.Activities = make([]aicourse.Activity, len(chapter.Activities))
	for index, activity := range chapter.Activities {
		trimmed.Activities[index] = activityWithExcerptPayload(activity, excerptRunes)
	}
	return trimmed
}

func activityWithExcerptPayload(activity aicourse.Activity, excerptRunes int) aicourse.Activity {
	excerpt := excerptFromActivity(activity, excerptRunes)
	payload, err := json.Marshal(aicourse.RichTextPayload{Markdown: excerpt, Highlights: []string{}})
	if err != nil {
		return activity
	}
	copy := activity
	copy.Title = truncateRunes(copy.Title, 120)
	copy.Payload = payload
	return copy
}

func excerptFromActivity(activity aicourse.Activity, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 80
	}
	switch activity.Type {
	case aicourse.ActivityTypeRichText:
		var payload aicourse.RichTextPayload
		if json.Unmarshal(activity.Payload, &payload) == nil && strings.TrimSpace(payload.Markdown) != "" {
			return truncateRunes(payload.Markdown, maxRunes)
		}
	case aicourse.ActivityTypeCoding:
		var payload aicourse.CodingPayload
		if json.Unmarshal(activity.Payload, &payload) == nil && strings.TrimSpace(payload.PromptMarkdown) != "" {
			return truncateRunes(payload.PromptMarkdown, maxRunes)
		}
	case aicourse.ActivityTypeMockInterview:
		var payload aicourse.MockInterviewPayload
		if json.Unmarshal(activity.Payload, &payload) == nil {
			if text := strings.TrimSpace(payload.ScenarioMarkdown); text != "" {
				return truncateRunes(text, maxRunes)
			}
		}
	case aicourse.ActivityTypeQuizGame:
		var payload aicourse.QuizGamePayload
		if json.Unmarshal(activity.Payload, &payload) == nil {
			if text := strings.TrimSpace(payload.InstructionsMarkdown); text != "" {
				return truncateRunes(text, maxRunes)
			}
			if len(payload.Questions) > 0 {
				return truncateRunes(payload.Questions[0].Question, maxRunes)
			}
		}
	}
	raw := strings.TrimSpace(string(activity.Payload))
	if raw == "" {
		return truncateRunes(activity.Title, maxRunes)
	}
	return truncateRunes(raw, maxRunes)
}

func slimCourseForQuality(full aicourse.StandardCourse, markdownLimit int) aicourse.StandardCourse {
	chapters := make([]aicourse.Chapter, len(full.Chapters))
	for index, chapter := range full.Chapters {
		chapters[index] = trimChapterPayloads(chapter, markdownLimit)
	}
	return aicourse.StandardCourse{
		SchemaVersion:    full.SchemaVersion,
		Title:            full.Title,
		Description:      truncateRunes(strings.TrimSpace(full.Description), 400),
		Audience:         full.Audience,
		Language:         full.Language,
		EstimatedMinutes: full.EstimatedMinutes,
		Chapters:         chapters,
	}
}

// marshalActivityContext 只带上目标章节，并截断过长正文，供章末测验等 activity 生成使用。
func marshalActivityContext(contextCourse *aicourse.StandardCourse, chapterKey string) (string, error) {
	if contextCourse == nil {
		return "", nil
	}
	chapter, ok := findChapter(contextCourse, chapterKey)
	if !ok {
		return marshalContext(contextCourse)
	}
	for markdownLimit := 2500; markdownLimit >= 300; markdownLimit -= 200 {
		slim := slimCourseForActivity(contextCourse, chapter, markdownLimit)
		encoded, err := json.Marshal(slim)
		if err != nil {
			return "", err
		}
		if len(encoded) <= activityContextBudget {
			return string(encoded), nil
		}
	}
	encoded, err := json.Marshal(slimCourseForActivity(contextCourse, chapter, 300))
	if err != nil {
		return "", err
	}
	text := string(encoded)
	if len(text) > activityContextBudget {
		text = text[:activityContextBudget]
	}
	return text, nil
}

func findChapter(course *aicourse.StandardCourse, chapterKey string) (aicourse.Chapter, bool) {
	for _, chapter := range course.Chapters {
		if chapter.Key == chapterKey {
			return chapter, true
		}
	}
	return aicourse.Chapter{}, false
}

func slimCourseForActivity(full *aicourse.StandardCourse, chapter aicourse.Chapter, markdownLimit int) aicourse.StandardCourse {
	return aicourse.StandardCourse{
		SchemaVersion: full.SchemaVersion,
		Title:         full.Title,
		Description:   truncateRunes(strings.TrimSpace(full.Description), 400),
		Audience:      full.Audience,
		Language:      full.Language,
		Chapters:      []aicourse.Chapter{trimChapterPayloads(chapter, markdownLimit)},
	}
}

func trimChapterPayloads(chapter aicourse.Chapter, markdownLimit int) aicourse.Chapter {
	trimmed := chapter
	trimmed.Activities = make([]aicourse.Activity, len(chapter.Activities))
	for index, activity := range chapter.Activities {
		trimmed.Activities[index] = trimActivityPayload(activity, markdownLimit)
	}
	if len(trimmed.Summary) > 0 {
		trimmed.Summary = truncateRunes(trimmed.Summary, 600)
	}
	return trimmed
}

func trimActivityPayload(activity aicourse.Activity, markdownLimit int) aicourse.Activity {
	if len(activity.Payload) == 0 {
		return activity
	}
	switch activity.Type {
	case aicourse.ActivityTypeRichText:
		var payload struct {
			Markdown   string   `json:"markdown"`
			Highlights []string `json:"highlights"`
		}
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return activity
		}
		payload.Markdown = truncateRunes(payload.Markdown, markdownLimit)
		if len(payload.Highlights) > 8 {
			payload.Highlights = payload.Highlights[:8]
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return activity
		}
		copy := activity
		copy.Payload = encoded
		return copy
	case aicourse.ActivityTypeCoding:
		var payload struct {
			Language       string `json:"language"`
			PromptMarkdown string `json:"promptMarkdown"`
		}
		if json.Unmarshal(activity.Payload, &payload) != nil {
			return activityWithExcerptPayload(activity, markdownLimit)
		}
		payload.PromptMarkdown = truncateRunes(payload.PromptMarkdown, markdownLimit)
		encoded, err := json.Marshal(payload)
		if err != nil {
			return activity
		}
		copy := activity
		copy.Payload = encoded
		return copy
	case aicourse.ActivityTypeMockInterview, aicourse.ActivityTypeQuizGame:
		return activityWithExcerptPayload(activity, markdownLimit)
	default:
		return activityWithExcerptPayload(activity, markdownLimit)
	}
}

func truncateRunes(text string, maxRunes int) string {
	if maxRunes <= 0 || text == "" {
		return text
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "…"
}
