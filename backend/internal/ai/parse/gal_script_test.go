package parse

import (
	"testing"

	"caveclass/internal/ai/gal"
)

func TestParseGalScriptJSON(t *testing.T) {
	raw := `{"title":"Intro","script":[{"speaker":"小洞","text":"你好","emotion":"happy"}]}`
	parsed, _, err := ParseGalScriptJSON(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Intro" || len(parsed.Script) != 1 || parsed.Script[0].Kind != gal.KindDialogue || parsed.Script[0].Emotion != "happy" {
		t.Fatalf("parsed = %+v", parsed)
	}
	choiceRaw := `{"title":"Quiz","script":[{"kind":"choice","prompt":"Q?","options":["A","B"],"correctIndex":1,"feedbackCorrect":"ok","feedbackWrong":"no"}]}`
	choiceParsed, _, err := ParseGalScriptJSON(choiceRaw, true)
	if err != nil {
		t.Fatal(err)
	}
	if choiceParsed.Script[0].Kind != gal.KindChoice {
		t.Fatalf("choice = %+v", choiceParsed.Script[0])
	}
	_, _, err = ParseGalScriptJSON(`{"title":"","script":[]}`, true)
	if err == nil {
		t.Fatal("expected validation error")
	}
	_, _, err = ParseGalScriptJSON(`{"title":"x","script":[{"speaker":"a","text":"b","emotion":"invalid"}]}`, true)
	if err == nil {
		t.Fatal("expected emotion error")
	}
	if !gal.ValidEmotion("proud") {
		t.Fatal("proud should be valid")
	}
}
