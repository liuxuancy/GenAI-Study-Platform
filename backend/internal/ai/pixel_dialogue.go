package ai

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
)

const SceneGeneratePixelDialogue = "generate_pixel_dialogue"

type PixelDialogueLines struct {
	Touch string `json:"touch"`
	Rest  string `json:"rest"`
}

type PixelDialogue struct {
	Zh PixelDialogueLines `json:"zh"`
	En PixelDialogueLines `json:"en"`
}

func normalizePixelDialogue(dialogue PixelDialogue) (PixelDialogue, error) {
	lines := []*string{&dialogue.Zh.Touch, &dialogue.Zh.Rest, &dialogue.En.Touch, &dialogue.En.Rest}
	for _, line := range lines {
		*line = strings.TrimSpace(*line)
		if *line == "" || utf8.RuneCountInString(*line) > 120 || strings.ContainsAny(*line, "\r\n") {
			return PixelDialogue{}, aierr.OutputSchemaFailed(nil, "pixel dialogue lines must be single-line and 1 to 120 characters")
		}
	}
	return dialogue, nil
}

func (service *Service) GeneratePixelDialogue(ctx context.Context, personality string) (PixelDialogue, Meta, error) {
	if err := service.ready(); err != nil {
		return PixelDialogue{}, Meta{}, err
	}
	response, meta, err := service.runModel(ctx, SceneGeneratePixelDialogue, prompt.PixelDialogueV1, "", struct{ Personality string }{personality})
	if err != nil {
		return PixelDialogue{}, meta, err
	}
	raw, err := parse.ExtractJSON(response.Text)
	if err != nil {
		return PixelDialogue{}, meta, err
	}
	var dialogue PixelDialogue
	if err := json.Unmarshal([]byte(raw), &dialogue); err != nil {
		return PixelDialogue{}, meta, aierr.OutputParseFailed(err, "pixel dialogue JSON")
	}
	dialogue, err = normalizePixelDialogue(dialogue)
	if err != nil {
		return PixelDialogue{}, meta, err
	}
	meta.Validation.Passed = true
	return dialogue, meta, nil
}
