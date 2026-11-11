package parse

import (
	"encoding/json"
	"strconv"
	"strings"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/gal"
)

// ParseGalScriptJSON 解析 Gal 剧本 JSON。
func ParseGalScriptJSON(rawText string, repair bool) (gal.ScriptResponse, Report, error) {
	_ = repair
	extracted, err := ExtractJSON(rawText)
	if err != nil {
		return gal.ScriptResponse{}, Report{}, err
	}
	var parsed gal.ScriptResponse
	if err := unmarshalGalJSON(extracted, &parsed); err != nil {
		return gal.ScriptResponse{}, Report{}, err
	}
	for index := range parsed.Script {
		gal.NormalizeScriptItem(&parsed.Script[index])
	}
	problems := galScriptProblems(parsed)
	if len(problems) > 0 {
		return gal.ScriptResponse{}, Report{}, schemaFailed(&course.ValidationError{Problems: problems})
	}
	return parsed, Report{}, nil
}

func unmarshalGalJSON(extracted string, parsed *gal.ScriptResponse) error {
	err := json.Unmarshal([]byte(extracted), parsed)
	if err == nil {
		return nil
	}
	relaxed := RelaxJSON(extracted)
	if relaxed != extracted {
		if err2 := json.Unmarshal([]byte(relaxed), parsed); err2 == nil {
			return nil
		}
	}
	return aierr.OutputParseFailed(err, "invalid JSON")
}

func galScriptProblems(parsed gal.ScriptResponse) []course.Problem {
	var problems []course.Problem
	if strings.TrimSpace(parsed.Title) == "" {
		problems = append(problems, course.Problem{Field: "title", Message: "must not be empty"})
	}
	if len(parsed.Script) == 0 {
		problems = append(problems, course.Problem{Field: "script", Message: "must contain at least one line"})
	}
	for index, item := range parsed.Script {
		prefix := "script[" + strconv.Itoa(index) + "]"
		switch item.Kind {
		case gal.KindChoice:
			problems = append(problems, choiceProblems(prefix, item)...)
		default:
			problems = append(problems, dialogueProblems(prefix, item)...)
		}
	}
	return problems
}

func dialogueProblems(prefix string, item gal.ScriptItem) []course.Problem {
	var problems []course.Problem
	if strings.TrimSpace(item.Speaker) == "" {
		problems = append(problems, course.Problem{Field: prefix + ".speaker", Message: "must not be empty"})
	}
	if strings.TrimSpace(item.Text) == "" {
		problems = append(problems, course.Problem{Field: prefix + ".text", Message: "must not be empty"})
	}
	if !gal.ValidEmotion(strings.TrimSpace(item.Emotion)) {
		problems = append(problems, course.Problem{Field: prefix + ".emotion", Message: "invalid emotion"})
	}
	if !gal.ValidBackground(strings.TrimSpace(item.Background)) {
		problems = append(problems, course.Problem{Field: prefix + ".background", Message: "invalid background"})
	}
	return problems
}

func choiceProblems(prefix string, item gal.ScriptItem) []course.Problem {
	var problems []course.Problem
	if strings.TrimSpace(item.Prompt) == "" {
		problems = append(problems, course.Problem{Field: prefix + ".prompt", Message: "must not be empty"})
	}
	if len(item.Options) < 2 {
		problems = append(problems, course.Problem{Field: prefix + ".options", Message: "must contain at least 2 options"})
	}
	if item.CorrectIndex < 0 || item.CorrectIndex >= len(item.Options) {
		problems = append(problems, course.Problem{Field: prefix + ".correctIndex", Message: "out of range"})
	}
	if strings.TrimSpace(item.FeedbackCorrect) == "" {
		problems = append(problems, course.Problem{Field: prefix + ".feedbackCorrect", Message: "must not be empty"})
	}
	if strings.TrimSpace(item.FeedbackWrong) == "" {
		problems = append(problems, course.Problem{Field: prefix + ".feedbackWrong", Message: "must not be empty"})
	}
	if emotion := strings.TrimSpace(item.EmotionOnCorrect); emotion != "" && !gal.ValidEmotion(emotion) {
		problems = append(problems, course.Problem{Field: prefix + ".emotionOnCorrect", Message: "invalid emotion"})
	}
	if emotion := strings.TrimSpace(item.EmotionOnWrong); emotion != "" && !gal.ValidEmotion(emotion) {
		problems = append(problems, course.Problem{Field: prefix + ".emotionOnWrong", Message: "invalid emotion"})
	}
	return problems
}
