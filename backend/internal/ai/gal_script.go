package ai

import (
	"context"
	"encoding/json"
	"strings"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/gal"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
)

const SceneGalStory = "gal_story"

// GalScriptInput 是生成 Gal 剧本的入参。
type GalScriptInput struct {
	CourseTitle string
	DetailLevel string
	Personality string
	ActivityKey string
}

// GenerateGalScript 根据章节 JSON 生成视觉小说剧本。
func (service *Service) GenerateGalScript(ctx context.Context, chapter course.Chapter, language string, input GalScriptInput) (gal.ScriptResponse, Meta, error) {
	if err := service.ready(); err != nil {
		return gal.ScriptResponse{}, Meta{}, err
	}
	if err := chapter.Validate(); err != nil {
		return gal.ScriptResponse{}, Meta{}, invalidCourse(err)
	}
	detailLevel := strings.TrimSpace(input.DetailLevel)
	if detailLevel == "" {
		detailLevel = gal.DetailDetailed
	}
	if !gal.ValidDetailLevel(detailLevel) {
		return gal.ScriptResponse{}, Meta{}, aierr.InvalidInput(nil, "detailLevel")
	}
	personality := strings.TrimSpace(input.Personality)
	if personality == "" {
		personality = gal.PersonalityGentle
	}
	if !gal.ValidPersonality(personality) {
		return gal.ScriptResponse{}, Meta{}, aierr.InvalidInput(nil, "personality")
	}
	if language != course.LanguageEnUS {
		language = course.LanguageZhCN
	}
	data, err := fitGalScript(chapter, language, input.CourseTitle, detailLevel, personality, strings.TrimSpace(input.ActivityKey), service.config.ModelPrimary)
	if err != nil {
		return gal.ScriptResponse{}, Meta{}, err
	}
	response, meta, err := service.runModel(ctx, SceneGalStory, prompt.GalScriptV1, "", data)
	if err != nil {
		return gal.ScriptResponse{}, meta, err
	}
	parsed, _, err := parse.ParseGalScriptJSON(response.Text, true)
	if err != nil {
		return gal.ScriptResponse{}, meta, err
	}
	parsed.Script = MergeChapterQuizzes(chapter, parsed.Script)
	meta.Validation.Passed = true
	return parsed, meta, nil
}

func fitGalScript(chapter course.Chapter, language, courseTitle, detailLevel, personality, activityKey, model string) (prompt.GalScriptData, error) {
	activityTitle, matchedKey, fullBody := matchedActivity(chapter, activityKey)
	budget := chapterPromptBudget()
	markdownLimits := []int{2500, 1500, 800, 400, 200}
	for _, limit := range markdownLimits {
		trimmed := trimChapterPayloads(chapter, limit)
		encoded, err := json.Marshal(trimmed)
		if err != nil {
			return prompt.GalScriptData{}, aierr.InvalidInput(err, "chapter")
		}
		bodyLimits := []int{4000, 2000, 800, 0}
		for _, bodyLimit := range bodyLimits {
			body := ""
			if bodyLimit > 0 {
				body = truncateRunes(fullBody, bodyLimit)
			}
			data := galScriptData(language, courseTitle, string(encoded), detailLevel, personality, matchedKey, activityTitle, body)
			rendered, _, renderErr := prompt.Render(prompt.GalScriptV1, data)
			if renderErr != nil {
				return prompt.GalScriptData{}, renderErr
			}
			if len(rendered) <= budget {
				_ = model
				return data, nil
			}
		}
	}
	return prompt.GalScriptData{}, aierr.InputTooLong(nil, "chapter prompt exceeds input budget for gal script")
}

func galScriptData(language, courseTitle, chapterJSON, detailLevel, personality, activityKey, activityTitle, activityBody string) prompt.GalScriptData {
	speaker := "小洞"
	if language == course.LanguageEnUS {
		speaker = "CaveGuide"
	}
	return prompt.GalScriptData{
		Language:               language,
		CourseTitle:            strings.TrimSpace(courseTitle),
		ChapterJSON:            chapterJSON,
		ActivityKey:            activityKey,
		ActivityTitle:          activityTitle,
		ActivityBody:           activityBody,
		SpeakerName:            speaker,
		DetailInstruction:      galDetailInstruction(detailLevel, language),
		PersonalityInstruction: galPersonalityInstruction(personality, language),
		TurnRequirement:        galTurnRequirement(detailLevel, language),
	}
}

func galDetailInstruction(detailLevel, language string) string {
	if language == course.LanguageEnUS {
		switch detailLevel {
		case gal.DetailBrief:
			return "Keep the script concise, about 12–18 dialogue lines."
		case gal.DetailAcademic:
			return "Be rigorous and terminology-aware; about 25–35 lines with deeper analysis."
		default:
			return "Balanced depth; about 18–28 dialogue lines covering all sections."
		}
	}
	switch detailLevel {
	case gal.DetailBrief:
		return "讲解简明，剧本约 12–18 句对话。"
	case gal.DetailAcademic:
		return "讲解偏学术、术语后需解释，剧本约 25–35 句。"
	default:
		return "讲解适中细致，剧本约 18–28 句，覆盖各小节。"
	}
}

func galPersonalityInstruction(personality, language string) string {
	if language == course.LanguageEnUS {
		switch personality {
		case gal.PersonalityTsundere:
			return "Playfully tsundere: tease the student but still teach carefully."
		case gal.PersonalityStrict:
			return "Strict coach: push the student to stay focused, no fluff."
		default:
			return "Warm and encouraging, like a patient mentor."
		}
	}
	switch personality {
	case gal.PersonalityTsundere:
		return "略带傲娇：嘴上吐槽同学学得慢，但讲解很认真。"
	case gal.PersonalityStrict:
		return "严厉教官风：要求同学跟上节奏，不许走神。"
	default:
		return "温柔鼓励：像耐心的学姐，多肯定同学的进步。"
	}
}

func galTurnRequirement(detailLevel, language string) string {
	if language == course.LanguageEnUS {
		return "The script array MUST meet the line count in Depth. Include optional note fields for technical terms."
	}
	return "script 数组必须满足 Depth 中的句数要求；遇到术语可在 note 字段补充简短解释。"
}
