package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
)

const (
	SceneGeneratePixelSprite = "generate_pixel_sprite"
	pixelGridSize            = 16
)

// PixelSpriteInput 是服务端已经选定的外观。模型只负责画五张网格。
type PixelSpriteInput struct {
	CourseID      int64
	Title         string
	RoleType      string
	BodyType      string
	HairColor     string
	HairStyle     string
	FacialFeature string
	Accessory     string
	SilhouetteKey string
	Personality   string
}

// PixelSpriteSet 是同一调色板下的初始图和四档表情动作。
type PixelSpriteSet struct {
	Palette  []string
	Base     []string
	At25     []string
	At50     []string
	At75     []string
	At100    []string
	Dialogue PixelDialogue
}

type pixelSpritePayload struct {
	Palette  []string      `json:"palette"`
	Base     []string      `json:"base"`
	At25     []string      `json:"25"`
	At50     []string      `json:"50"`
	At75     []string      `json:"75"`
	At100    []string      `json:"100"`
	Dialogue PixelDialogue `json:"dialogue"`
}

// GeneratePixelSprite 一次调用画出初始形象和 25%、50%、75%、100% 的表情动作。
func (service *Service) GeneratePixelSprite(ctx context.Context, input PixelSpriteInput) (PixelSpriteSet, Meta, error) {
	if err := service.ready(); err != nil {
		return PixelSpriteSet{}, Meta{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	data := prompt.PixelSpriteData{
		CourseID:      input.CourseID,
		Title:         input.Title,
		RoleType:      input.RoleType,
		BodyType:      input.BodyType,
		HairColor:     input.HairColor,
		HairStyle:     input.HairStyle,
		FacialFeature: input.FacialFeature,
		Accessory:     input.Accessory,
		SilhouetteKey: input.SilhouetteKey,
		Personality:   input.Personality,
	}
	var total Usage
	var latencyMs int64
	var fallbackUsed bool
	for attempt := 0; ; attempt++ {
		response, meta, err := service.runModel(ctx, SceneGeneratePixelSprite, prompt.PixelSpriteV1, "", data)
		total.PromptTokens += meta.Usage.PromptTokens
		total.CompletionTokens += meta.Usage.CompletionTokens
		total.TotalTokens += meta.Usage.TotalTokens
		latencyMs += meta.LatencyMs
		fallbackUsed = fallbackUsed || meta.FallbackUsed
		meta.Usage = total
		meta.LatencyMs = latencyMs
		meta.FallbackUsed = fallbackUsed
		if err != nil {
			return PixelSpriteSet{}, meta, err
		}
		set, err := parsePixelSprite(response.Text)
		if err == nil {
			meta.Validation.Passed = true
			if attempt > 0 {
				meta.Warnings = append(meta.Warnings, "pixel sprite output was retried after validation failed")
			}
			return set, meta, nil
		}
		if (!errors.Is(err, aierr.ErrOutputParseFailed) && !errors.Is(err, aierr.ErrOutputSchemaFailed)) || attempt >= service.config.MaxRetries {
			return PixelSpriteSet{}, meta, err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return PixelSpriteSet{}, meta, completeError(ctxErr)
		}
		data.ValidationFeedback = "The response must be a complete, valid JSON object matching the requested structure."
		if aiError, ok := aierr.As(err); ok && errors.Is(err, aierr.ErrOutputSchemaFailed) && len(aiError.Details) > 0 {
			data.ValidationFeedback = strings.Join(aiError.Details, "; ")
		}
		data.PreviousSprite, err = safeSpriteGrids(response.Text)
		if err != nil {
			return PixelSpriteSet{}, meta, err
		}
		slog.WarnContext(ctx, "retrying invalid pixel sprite output",
			slog.String("reason", data.ValidationFeedback), slog.Int("attempt", attempt+1))
	}
}

func parsePixelSprite(raw string) (PixelSpriteSet, error) {
	extracted, err := parse.ExtractJSON(raw)
	if err != nil {
		return PixelSpriteSet{}, err
	}
	var payload pixelSpritePayload
	if err := json.Unmarshal([]byte(extracted), &payload); err != nil {
		return PixelSpriteSet{}, aierr.OutputParseFailed(err, "pixel sprite JSON")
	}
	palette, err := normalizePalette(payload.Palette)
	if err != nil {
		return PixelSpriteSet{}, err
	}
	frames := [][]string{payload.Base, payload.At25, payload.At50, payload.At75, payload.At100}
	names := []string{"base", "25", "50", "75", "100"}
	normalized := make([][]string, len(frames))
	for index, frame := range frames {
		grid, err := normalizePixelGrid(frame, palette)
		if err != nil {
			return PixelSpriteSet{}, aierr.OutputSchemaFailed(err, "pose "+names[index]+": "+strings.Join(aierrDetails(err), "; "))
		}
		normalized[index] = grid
	}
	dialogue, err := normalizePixelDialogue(payload.Dialogue)
	if err != nil {
		return PixelSpriteSet{}, err
	}
	return PixelSpriteSet{
		Palette:  palette,
		Base:     normalized[0],
		At25:     normalized[1],
		At50:     normalized[2],
		At75:     normalized[3],
		At100:    normalized[4],
		Dialogue: dialogue,
	}, nil
}

func safeSpriteGrids(raw string) (string, error) {
	extracted, err := parse.ExtractJSON(raw)
	if err != nil {
		return "", nil
	}
	var payload pixelSpritePayload
	if err := json.Unmarshal([]byte(extracted), &payload); err != nil {
		return "", nil
	}
	palette, err := normalizePalette(payload.Palette)
	if err != nil {
		return "", nil
	}
	frames := [][]string{payload.Base, payload.At25, payload.At50, payload.At75, payload.At100}
	for _, frame := range frames {
		if len(frame) != pixelGridSize {
			return "", nil
		}
		for _, row := range frame {
			if len(row) != pixelGridSize {
				return "", nil
			}
			for i := 0; i < len(row); i++ {
				if _, ok := paletteIndex(row[i]); row[i] != '.' && !ok {
					return "", nil
				}
			}
		}
	}
	grids := struct {
		Palette []string `json:"palette"`
		Base    []string `json:"base"`
		At25    []string `json:"25"`
		At50    []string `json:"50"`
		At75    []string `json:"75"`
		At100   []string `json:"100"`
	}{palette, payload.Base, payload.At25, payload.At50, payload.At75, payload.At100}
	encoded, err := json.Marshal(grids)
	if err != nil {
		return "", aierr.Internal(err, "encode pixel grids for retry")
	}
	return string(encoded), nil
}

func aierrDetails(err error) []string {
	if aiError, ok := aierr.As(err); ok {
		return aiError.Details
	}
	return []string{"invalid grid"}
}

func normalizePalette(colors []string) ([]string, error) {
	if len(colors) < 2 || len(colors) > 8 {
		return nil, aierr.OutputSchemaFailed(nil, "palette must contain 2 to 8 colors")
	}
	normalized := make([]string, len(colors))
	for index, color := range colors {
		color = strings.ToLower(strings.TrimSpace(color))
		if len(color) != 7 || color[0] != '#' || !isHexColor(color[1:]) {
			return nil, aierr.OutputSchemaFailed(nil, "palette color must be #rrggbb")
		}
		normalized[index] = color
	}
	return normalized, nil
}

func isHexColor(value string) bool {
	for index := 0; index < len(value); index++ {
		char := value[index]
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		default:
			return false
		}
	}
	return len(value) == 6
}

func normalizePixelGrid(rows []string, palette []string) ([]string, error) {
	if len(rows) != pixelGridSize {
		return nil, aierr.OutputSchemaFailed(nil, "pixel grid must be 16 rows")
	}
	normalized := make([]string, pixelGridSize)
	opaque := 0
	for y, row := range rows {
		if utf8.RuneCountInString(row) != pixelGridSize || len(row) != pixelGridSize {
			return nil, aierr.OutputSchemaFailed(nil, "each pixel row must be 16 characters")
		}
		for x := 0; x < pixelGridSize; x++ {
			char := row[x]
			if char == '.' {
				continue
			}
			index, ok := paletteIndex(char)
			if !ok {
				return nil, aierr.OutputSchemaFailed(nil, fmt.Sprintf("invalid pixel symbol at row %d column %d; use '.' or indexes 0 through %d", y+1, x+1, len(palette)-1))
			}
			if index >= len(palette) {
				return nil, aierr.OutputSchemaFailed(nil, fmt.Sprintf("pixel index %q at row %d column %d is outside the palette; %d colors allow indexes 0 through %d", char, y+1, x+1, len(palette), len(palette)-1))
			}
			opaque++
		}
		normalized[y] = row
	}
	if opaque == 0 {
		return nil, aierr.OutputSchemaFailed(nil, "pixel grid is empty")
	}
	return normalized, nil
}

func paletteIndex(char byte) (int, bool) {
	switch {
	case char >= '0' && char <= '9':
		return int(char - '0'), true
	case char >= 'a' && char <= 'f':
		return int(char-'a') + 10, true
	default:
		return 0, false
	}
}
