// Package validate 校验 AI 入参，并拦截明显的提示词注入。
package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
)

const (
	MaxTopicRunes             = 500
	MaxExtraRequirementsRunes = 2000
	MaxChapterCountHint       = 30
	MaxModelRunes             = 100
)

// CourseInput 是整课生成的入参。空 language 会在校验时补成 zh-CN。
type CourseInput struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	ChapterCountHint  int
	Model             string
	TemplateID        string
}

var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|prior)\s+(instructions|prompts|rules)`),
	regexp.MustCompile(`(?i)disregard\s+(all\s+)?(previous|above|prior)`),
	regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an|the)\b`),
	regexp.MustCompile(`(?i)<\|\s*(im_start|system)\s*\|>`),
	regexp.MustCompile(`</?\s*user_topic\b`),
	regexp.MustCompile(`</?\s*extra_requirements\b`),
	regexp.MustCompile(`忽略(之前|以上|前面)(的)?(所有)?(指令|提示|提示词|规则)`),
	regexp.MustCompile(`忘记(你的|之前的)?(指令|设定|角色|规则)`),
	regexp.MustCompile(`你现在扮演`),
}

// NormalizeCourse 去掉首尾空白、补默认语言，并拒绝超长、非法枚举和注入。
// maxTopicLen 小于等于 0 时使用 500。
func NormalizeCourse(input CourseInput, maxTopicLen int) (CourseInput, error) {
	if maxTopicLen <= 0 {
		maxTopicLen = MaxTopicRunes
	}
	input.Topic = strings.TrimSpace(input.Topic)
	input.Audience = strings.TrimSpace(input.Audience)
	input.Language = strings.TrimSpace(input.Language)
	input.ExtraRequirements = strings.TrimSpace(input.ExtraRequirements)
	input.Model = strings.TrimSpace(input.Model)
	input.TemplateID = strings.TrimSpace(input.TemplateID)

	if input.Topic == "" {
		return CourseInput{}, aierr.InvalidInput(nil, "topic")
	}
	if utf8.RuneCountInString(input.Topic) > maxTopicLen {
		return CourseInput{}, aierr.InputTooLong(nil, fmt.Sprintf("topic exceeds %d characters", maxTopicLen))
	}
	switch input.Audience {
	case course.AudienceCS, course.AudienceNonCS, course.AudienceGeneral:
	default:
		return CourseInput{}, aierr.InvalidInput(nil, "audience")
	}
	if input.Language == "" {
		input.Language = course.LanguageZhCN
	} else if input.Language != course.LanguageZhCN && input.Language != course.LanguageEnUS {
		return CourseInput{}, aierr.InvalidInput(nil, "language")
	}
	if utf8.RuneCountInString(input.ExtraRequirements) > MaxExtraRequirementsRunes {
		return CourseInput{}, aierr.InputTooLong(nil, fmt.Sprintf("extraRequirements exceeds %d characters", MaxExtraRequirementsRunes))
	}
	if input.ChapterCountHint < 0 || input.ChapterCountHint > MaxChapterCountHint {
		return CourseInput{}, aierr.InvalidInput(nil, "chapterCountHint")
	}
	if utf8.RuneCountInString(input.Model) > MaxModelRunes {
		return CourseInput{}, aierr.InvalidInput(nil, "model")
	}
	if suspectInjection(input.Topic) {
		return CourseInput{}, aierr.PromptInjectionSuspect(nil, "topic")
	}
	if suspectInjection(input.ExtraRequirements) {
		return CourseInput{}, aierr.PromptInjectionSuspect(nil, "extraRequirements")
	}
	return input, nil
}

// ChapterInput 是单章生成的入参。
type ChapterInput struct {
	CourseInput
	TargetChapterKey string
}

// ActivityInput 是单 activity 生成的入参。
type ActivityInput struct {
	ChapterInput
	TargetActivityKey string
}

// NormalizeChapter 校验整课公共字段，并要求 targetChapterKey。
func NormalizeChapter(input ChapterInput, maxTopicLen int) (ChapterInput, error) {
	checked, err := NormalizeCourse(input.CourseInput, maxTopicLen)
	if err != nil {
		return ChapterInput{}, err
	}
	input.CourseInput = checked
	key, err := normalizeKey("targetChapterKey", input.TargetChapterKey)
	if err != nil {
		return ChapterInput{}, err
	}
	input.TargetChapterKey = key
	return input, nil
}

// NormalizeActivity 在章节入参之外再要求 targetActivityKey。
func NormalizeActivity(input ActivityInput, maxTopicLen int) (ActivityInput, error) {
	checked, err := NormalizeChapter(input.ChapterInput, maxTopicLen)
	if err != nil {
		return ActivityInput{}, err
	}
	input.ChapterInput = checked
	key, err := normalizeKey("targetActivityKey", input.TargetActivityKey)
	if err != nil {
		return ActivityInput{}, err
	}
	input.TargetActivityKey = key
	return input, nil
}

func normalizeKey(field, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", aierr.InvalidInput(nil, field)
	}
	if utf8.RuneCountInString(key) > course.MaxKeyRunes {
		return "", aierr.InvalidInput(nil, field)
	}
	if suspectInjection(key) {
		return "", aierr.PromptInjectionSuspect(nil, field)
	}
	return key, nil
}

func suspectInjection(text string) bool {
	if text == "" {
		return false
	}
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
