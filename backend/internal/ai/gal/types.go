// Package gal 定义视觉小说剧本 JSON 的结构。
package gal

import "strings"

const (
	DetailBrief    = "brief"
	DetailDetailed = "detailed"
	DetailAcademic = "academic"

	PersonalityTsundere = "tsundere"
	PersonalityGentle   = "gentle"
	PersonalityStrict   = "strict"

	KindDialogue = "dialogue"
	KindChoice   = "choice"
)

// ScriptItem 是剧本中的一步：对话或选择题。
type ScriptItem struct {
	Kind string `json:"kind,omitempty"`

	Speaker string `json:"speaker,omitempty"`
	Text    string `json:"text,omitempty"`
	Emotion string `json:"emotion,omitempty"`
	Note       string `json:"note,omitempty"`
	Background string `json:"background,omitempty"`

	Prompt           string   `json:"prompt,omitempty"`
	Options          []string `json:"options,omitempty"`
	CorrectIndex     int      `json:"correctIndex,omitempty"`
	FeedbackCorrect  string   `json:"feedbackCorrect,omitempty"`
	FeedbackWrong    string   `json:"feedbackWrong,omitempty"`
	EmotionOnCorrect string   `json:"emotionOnCorrect,omitempty"`
	EmotionOnWrong   string   `json:"emotionOnWrong,omitempty"`
}

// ScriptResponse 是模型输出的完整剧本。
type ScriptResponse struct {
	Title  string       `json:"title"`
	Script []ScriptItem `json:"script"`
}

// NormalizeScriptItem 补全 kind，兼容旧版仅含 text 的对话项。
func NormalizeScriptItem(item *ScriptItem) {
	if item == nil {
		return
	}
	kind := strings.TrimSpace(item.Kind)
	if kind == KindChoice || (strings.TrimSpace(item.Prompt) != "" && len(item.Options) > 0) {
		item.Kind = KindChoice
		return
	}
	item.Kind = KindDialogue
}

func ValidDetailLevel(value string) bool {
	switch value {
	case DetailBrief, DetailDetailed, DetailAcademic, "":
		return true
	default:
		return false
	}
}

func ValidPersonality(value string) bool {
	switch value {
	case PersonalityTsundere, PersonalityGentle, PersonalityStrict, "":
		return true
	default:
		return false
	}
}

func ValidEmotion(value string) bool {
	switch value {
	case "normal", "happy", "angry", "surprised", "shy", "proud", "":
		return true
	default:
		return false
	}
}

func ValidBackground(value string) bool {
	switch value {
	case "abstract-nodes", "campus-path", "cave-hub", "chapter-clear", "classroom-soft",
		"computer-lab", "library", "night-lamp", "quiz-focus", "study-desk", "":
		return true
	default:
		return false
	}
}
