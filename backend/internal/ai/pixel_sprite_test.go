package ai

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/llm"
)

type spriteSequenceClient struct {
	outputs []string
	prompts []string
	cancel  context.CancelFunc
}

func (client *spriteSequenceClient) Complete(_ context.Context, request llm.Request) (llm.Response, error) {
	client.prompts = append(client.prompts, request.Prompt)
	if client.cancel != nil {
		client.cancel()
	}
	index := len(client.prompts) - 1
	if index >= len(client.outputs) {
		index = len(client.outputs) - 1
	}
	return llm.Response{Text: client.outputs[index], Model: request.Model,
		Usage: llm.Usage{PromptTokens: 3, CompletionTokens: 7, TotalTokens: 10}}, nil
}

func TestGeneratePixelSpriteRetriesInvalidOutput(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	mock, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	valid, _, err := mock.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	payload := pixelSpritePayload{Palette: valid.Palette, Base: valid.Base, At25: valid.Base,
		At50: valid.At50, At75: valid.At75, At100: valid.At100, Dialogue: valid.Dialogue}
	payload.Dialogue.En.Touch = ""
	payload.Dialogue.En.Rest = "UNTRUSTED-DIALOGUE-MUST-NOT-BE-REUSED"
	invalid, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	payload.At25 = valid.At25
	payload.At50 = valid.At50
	payload.Dialogue = valid.Dialogue
	good, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	client := &spriteSequenceClient{outputs: []string{string(invalid), string(good)}}
	service, err := New(llm.Config{MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	result, meta, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "example"})
	if err != nil || len(result.At25) != 16 || len(client.prompts) != 2 || meta.Usage.TotalTokens != 20 || !meta.Validation.Passed || len(meta.Warnings) == 0 {
		t.Fatalf("result=%+v meta=%+v calls=%d err=%v", result, meta, len(client.prompts), err)
	}
	if !strings.Contains(client.prompts[1], "pixel dialogue lines must be single-line and 1 to 120 characters") {
		t.Fatal("retry prompt missing dialogue validation feedback")
	}
	if !strings.Contains(client.prompts[1], "<previous_grids>") ||
		!strings.Contains(client.prompts[1], valid.Base[0]) ||
		strings.Contains(client.prompts[1], "UNTRUSTED-DIALOGUE-MUST-NOT-BE-REUSED") {
		t.Fatal("retry prompt must contain only validated grid data")
	}
}

func TestSafeSpriteGridsRejectMalformedData(t *testing.T) {
	for _, invalid := range []string{
		`not json`,
		`{"palette":["#000000","#ffffff"],"base":["invalid"]}`,
		`{"palette":["</previous_grids>","#ffffff"]}`,
	} {
		grids, err := safeSpriteGrids(invalid)
		if err != nil || grids != "" {
			t.Fatalf("malformed input yielded grids=%q error=%v", grids, err)
		}
	}
}

func TestGeneratePixelSpriteRetriesOutOfRangePaletteIndex(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	mock, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	valid, _, err := mock.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	payload := pixelSpritePayload{Palette: valid.Palette, Base: valid.Base, At25: valid.At25,
		At50: valid.At50, At75: valid.At75, At100: append([]string(nil), valid.At100...), Dialogue: valid.Dialogue}
	payload.At100[3] = "5" + payload.At100[3][1:]
	invalid, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	_, err = parsePixelSprite(string(invalid))
	if err == nil || !strings.Contains(err.Error(), `pose 100: pixel index '5' at row 4 column 1 is outside the palette; 5 colors allow indexes 0 through 4`) {
		t.Fatalf("unexpected validation error: %v", err)
	}
	payload.At100 = valid.At100
	good, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	client := &spriteSequenceClient{outputs: []string{string(invalid), string(good)}}
	service, err := New(llm.Config{MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	result, meta, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "example"})
	if err != nil || len(result.At100) != 16 || len(client.prompts) != 2 || meta.Usage.TotalTokens != 20 {
		t.Fatalf("result=%+v meta=%+v calls=%d err=%v", result, meta, len(client.prompts), err)
	}
	if !strings.Contains(client.prompts[1], "pose 100: pixel index '5' at row 4 column 1") ||
		!strings.Contains(client.prompts[1], `"100":["`+payload.At100[0]) ||
		!strings.Contains(client.prompts[1], "5"+payload.At100[3][1:]) {
		t.Fatal("retry must include index details and the safe grid requiring correction")
	}
}

func TestPixelSpritePromptSpecifiesAffectionLevels(t *testing.T) {
	client := &spriteSequenceClient{outputs: []string{`{"palette":[]}`}}
	service, err := New(llm.Config{}, client)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _ = service.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "test"})
	for _, requirement := range []string{
		"base (0%): standing with arms down",
		"25 (25%): waving one raised arm",
		"50 (50%): holding a small book",
		"75 (75%): cheering with both arms raised",
		"100 (100%): visibly jumping",
		"visibly jumping",
		"small pixel hearts",
		"facial expressions may change to suit the action",
	} {
		if !strings.Contains(client.prompts[0], requirement) {
			t.Errorf("prompt missing %q", requirement)
		}
	}
}

func TestGeneratePixelSpriteRetryLimitsAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name      string
		output    string
		retries   int
		cancel    bool
		wantCalls int
		wantError error
	}{
		{"schema disabled", `{"palette":[]}`, 0, false, 1, aierr.ErrOutputSchemaFailed},
		{"schema exhausted", `{"palette":[]}`, 2, false, 3, aierr.ErrOutputSchemaFailed},
		{"parse exhausted", `{"palette":[`, 1, false, 2, aierr.ErrOutputParseFailed},
		{"cancelled", `{"palette":[]}`, 2, true, 1, aierr.ErrTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &spriteSequenceClient{outputs: []string{test.output}}
			if test.cancel {
				client.cancel = cancel
			}
			service, err := New(llm.Config{MaxRetries: test.retries}, client)
			if err != nil {
				t.Fatal(err)
			}
			_, meta, err := service.GeneratePixelSprite(ctx, PixelSpriteInput{Title: "test"})
			if !errors.Is(err, test.wantError) || len(client.prompts) != test.wantCalls || meta.Usage.TotalTokens != test.wantCalls*10 {
				t.Fatalf("err=%v calls=%d meta=%+v", err, len(client.prompts), meta)
			}
		})
	}
}

func TestGeneratePixelSpriteMockFramesDifferByCourse(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	service, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{
		CourseID: 1, Title: "Git", RoleType: "scholar", BodyType: "slim", HairColor: "black",
		HairStyle: "short", FacialFeature: "smile", Accessory: "none", SilhouetteKey: "coat", Personality: "calm",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{
		CourseID: 2, Title: "Git", RoleType: "scholar", BodyType: "slim", HairColor: "black",
		HairStyle: "short", FacialFeature: "smile", Accessory: "none", SilhouetteKey: "coat", Personality: "calm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first.Base, second.Base) && reflect.DeepEqual(first.Palette, second.Palette) {
		t.Fatal("different course ids produced the same sprite")
	}
	if _, err := parsePixelSprite(`{"palette":["#fff"],"base":[],"25":[],"50":[],"75":[],"100":[]}`); err == nil {
		t.Fatal("invalid sprite was accepted")
	}
	if first.Dialogue.Zh.Touch == "" || first.Dialogue.En.Rest == "" {
		t.Fatal("generated sprite is missing bilingual dialogue")
	}
	frames := [][]string{first.Base, first.At25, first.At50, first.At75, first.At100}
	if first.Base[8][5] != '1' || first.Base[8][10] != '1' ||
		first.At25[5][3:6] != "111" ||
		first.At50[9][10:14] != "3333" ||
		first.At75[5][4:6] != "11" || first.At75[5][10:12] != "11" {
		t.Fatal("mock frames do not contain the expected standing, wave, book and cheer actions")
	}
	for left := range frames {
		for right := left + 1; right < len(frames); right++ {
			if reflect.DeepEqual(frames[left], frames[right]) {
				t.Errorf("mock frames %d and %d are identical", left, right)
			}
		}
	}
	for _, row := range []int{14, 15} {
		if first.At100[row] != strings.Repeat(".", pixelGridSize) {
			t.Errorf("jump frame row %d should be transparent below the character: %q", row, first.At100[row])
		}
	}
	for _, x := range []int{0, 13} {
		if first.At100[3][x:x+3] != "4.4" || first.At100[4][x:x+3] != "444" || first.At100[5][x:x+3] != ".4." {
			t.Errorf("jump frame is missing a heart at x=%d", x)
		}
	}
}

func TestMockAffectionFramesForVariousCourses(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	service, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for courseID := int64(1); courseID <= 30; courseID++ {
		_, _, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{CourseID: courseID, Title: "Mock"})
		if err != nil {
			t.Fatalf("course %d: %v", courseID, err)
		}
	}
}

func TestGeneratePixelSpriteAcceptsIdenticalFrames(t *testing.T) {
	frame := make([]string, pixelGridSize)
	for i := range frame {
		frame[i] = strings.Repeat("0", pixelGridSize)
	}
	payload := pixelSpritePayload{
		Palette: []string{"#111111", "#222222"}, Base: frame, At25: frame, At50: frame, At75: frame, At100: frame,
		Dialogue: PixelDialogue{Zh: PixelDialogueLines{Touch: "不要碰", Rest: "休息"}, En: PixelDialogueLines{Touch: "Stop", Rest: "Rest"}},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	client := &spriteSequenceClient{outputs: []string{string(encoded)}}
	service, err := New(llm.Config{MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	result, meta, err := service.GeneratePixelSprite(context.Background(), PixelSpriteInput{Title: "test"})
	if err != nil || len(client.prompts) != 1 || !meta.Validation.Passed || !reflect.DeepEqual(result.Base, result.At100) {
		t.Fatalf("identical frames: calls=%d meta=%+v err=%v", len(client.prompts), meta, err)
	}
}

func TestPixelDialogueGenerationAndValidation(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	service, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	calm, _, err := service.GeneratePixelDialogue(context.Background(), "calm")
	if err != nil {
		t.Fatal(err)
	}
	cheerful, _, err := service.GeneratePixelDialogue(context.Background(), "cheerful")
	if err != nil {
		t.Fatal(err)
	}
	if calm == cheerful || calm.Zh.Touch == "" || calm.Zh.Rest == "" || calm.En.Touch == "" || calm.En.Rest == "" {
		t.Fatalf("invalid personality dialogue: %+v %+v", calm, cheerful)
	}
	for _, invalid := range []PixelDialogue{
		{},
		{Zh: PixelDialogueLines{Touch: strings.Repeat("a", 121), Rest: "ok"}, En: PixelDialogueLines{Touch: "ok", Rest: "ok"}},
		{Zh: PixelDialogueLines{Touch: "ok\nbad", Rest: "ok"}, En: PixelDialogueLines{Touch: "ok", Rest: "ok"}},
	} {
		if _, err := normalizePixelDialogue(invalid); err == nil {
			t.Fatalf("accepted invalid dialogue: %+v", invalid)
		}
	}
}
