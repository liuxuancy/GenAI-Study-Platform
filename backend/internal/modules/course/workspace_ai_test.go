package course

import (
	"strings"
	"testing"
)

func TestRenderSkillPrompt_generatePracticeCount(t *testing.T) {
	skill, ok := findSkill("generate_practice")
	if !ok {
		t.Fatal("generate_practice skill missing")
	}
	base := map[string]string{"topic": "链表", "difficulty": "中等", "questionType": "混合"}

	t.Run("valid count", func(t *testing.T) {
		params := cloneStringMap(base)
		params["count"] = "8"
		prompt, err := renderSkillPrompt(skill, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(prompt, "8") {
			t.Fatalf("expected count in prompt: %q", prompt)
		}
	})

	t.Run("empty count uses default in prompt", func(t *testing.T) {
		prompt, err := renderSkillPrompt(skill, base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(prompt, "3") {
			t.Fatalf("expected default count in prompt: %q", prompt)
		}
	})

	for _, count := range []string{"0", "9", "3.5", "abc"} {
		t.Run("reject "+count, func(t *testing.T) {
			params := cloneStringMap(base)
			params["count"] = count
			_, err := renderSkillPrompt(skill, params)
			if err == nil || !strings.Contains(err.Error(), "题目数量") {
				t.Fatalf("expected count range error, got %v", err)
			}
		})
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
