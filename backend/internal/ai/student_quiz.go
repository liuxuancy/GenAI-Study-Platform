package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
)

const SceneStudentQuiz = "student_quiz"

// GenerateStudentQuiz requires already selected and bounded course excerpts.
func (service *Service) GenerateStudentQuiz(ctx context.Context, contextJSON, language, difficulty string, count int) ([]course.QuizQuestion, Meta, error) {
	if err := service.ready(); err != nil {
		return nil, Meta{}, err
	}
	if strings.TrimSpace(contextJSON) == "" || !json.Valid([]byte(contextJSON)) || (count != 2 && count != 5 && count != 10) ||
		(difficulty != "easy" && difficulty != "medium" && difficulty != "hard") {
		return nil, Meta{}, aierr.InvalidInput(nil, "quiz input")
	}
	data := prompt.StudentQuizData{Language: insightLanguage(language), Count: count, Difficulty: difficulty, ContextJSON: contextJSON}
	rendered, _, err := prompt.Render(prompt.StudentQuizV1, data)
	if err != nil {
		return nil, Meta{}, err
	}
	if len(rendered) > chapterPromptBudget() {
		return nil, Meta{}, aierr.InputTooLong(nil, "quiz context exceeds input budget")
	}
	response, meta, err := service.runModel(ctx, SceneStudentQuiz, prompt.StudentQuizV1, "", data)
	if err != nil {
		return nil, meta, err
	}
	raw, err := parse.ExtractJSON(response.Text)
	if err != nil {
		return nil, meta, err
	}
	var result struct {
		Questions []struct {
			Question     string   `json:"question"`
			Options      []string `json:"options"`
			CorrectIndex *int     `json:"correctIndex"`
			Explanation  string   `json:"explanation"`
		} `json:"questions"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, meta, aierr.OutputParseFailed(err, "invalid quiz JSON")
	}
	if len(result.Questions) != count {
		return nil, meta, aierr.OutputSchemaFailed(nil, "question count does not match request")
	}
	questions := make([]course.QuizQuestion, 0, count)
	seenQuestions := map[string]bool{}
	for _, question := range result.Questions {
		trimmedQuestion := strings.TrimSpace(question.Question)
		if trimmedQuestion == "" || utf8.RuneCountInString(question.Question) > 2000 || seenQuestions[trimmedQuestion] ||
			strings.TrimSpace(question.Explanation) == "" || len(question.Options) < 2 || len(question.Options) > 8 ||
			question.CorrectIndex == nil || *question.CorrectIndex < 0 || *question.CorrectIndex >= len(question.Options) {
			return nil, meta, aierr.OutputSchemaFailed(nil, "invalid question")
		}
		seenQuestions[trimmedQuestion] = true
		seen := map[string]bool{}
		for _, option := range question.Options {
			if strings.TrimSpace(option) != option || option == "" || strings.Contains(option, "|") ||
				utf8.RuneCountInString(option) > 200 || seen[option] {
				return nil, meta, aierr.OutputSchemaFailed(nil, "invalid question options")
			}
			seen[option] = true
		}
		questions = append(questions, course.QuizQuestion{Question: question.Question, Options: question.Options, CorrectIndex: *question.CorrectIndex, Explanation: question.Explanation})
	}
	meta.Validation.Passed = true
	meta.Warnings = append(meta.Warnings, fmt.Sprintf("generated from selected course excerpts (%d questions)", count))
	return questions, meta, nil
}
