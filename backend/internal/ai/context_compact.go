package ai

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	aicourse "caveclass/internal/ai/course"
)

// activityContextBudget 为 prompt 模板留出空间，避免 Teamily input 超过 32000。
const activityContextBudget = 24000

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
	default:
		return activity
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
