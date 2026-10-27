package llm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
)

var configKeys = []string{
	"AI_PROVIDER", "AI_API_KEY", "AI_MODEL_PRIMARY", "AI_MODEL_FALLBACK",
	"AI_TIMEOUT", "AI_MAX_RETRIES", "AI_MAX_TOPIC_LEN",
}

func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range configKeys {
		t.Setenv(key, values[key])
	}
}

func TestLoadConfigDefaultsToMock(t *testing.T) {
	setEnv(t, nil)
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != llm.ProviderMock || cfg.ModelPrimary != llm.DefaultModelPrimary {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Timeout != llm.DefaultTimeout || cfg.MaxRetries != llm.DefaultMaxRetries || cfg.MaxTopicLen != llm.DefaultMaxTopicLen {
		t.Fatalf("numeric defaults = %+v", cfg)
	}
	if cfg.APIKey != "" || cfg.ModelFallback != "" {
		t.Fatalf("empty env should stay empty: %+v", cfg)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	setEnv(t, map[string]string{
		"AI_PROVIDER":       "azure",
		"AI_API_KEY":        "secret-key",
		"AI_MODEL_PRIMARY":  "gpt-4o",
		"AI_MODEL_FALLBACK": "gpt-4o-mini",
		"AI_TIMEOUT":        "30s",
		"AI_MAX_RETRIES":    "0",
		"AI_MAX_TOPIC_LEN":  "120",
	})
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != llm.ProviderAzure || cfg.APIKey != "secret-key" || cfg.ModelPrimary != "gpt-4o" || cfg.ModelFallback != "gpt-4o-mini" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Timeout.String() != "30s" || cfg.MaxRetries != 0 || cfg.MaxTopicLen != 120 {
		t.Fatalf("numeric overrides = %+v", cfg)
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	tests := []map[string]string{
		{"AI_PROVIDER": "claude"},
		{"AI_TIMEOUT": "soon"},
		{"AI_MAX_RETRIES": "-1"},
		{"AI_MAX_TOPIC_LEN": "0"},
	}
	for _, values := range tests {
		t.Run(values["AI_PROVIDER"]+values["AI_TIMEOUT"]+values["AI_MAX_RETRIES"]+values["AI_MAX_TOPIC_LEN"], func(t *testing.T) {
			setEnv(t, values)
			if _, err := llm.LoadConfig(); err == nil {
				t.Fatal("expected config error")
			}
		})
	}
}

func TestMockProviderReturnsFixedCourseJSON(t *testing.T) {
	setEnv(t, map[string]string{"AI_PROVIDER": "mock", "AI_API_KEY": "do-not-leak"})
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Complete(context.Background(), llm.Request{Prompt: "generate a course"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != llm.MockText() || strings.Contains(response.Text, "do-not-leak") {
		t.Fatalf("unexpected mock text, model=%s", response.Model)
	}
	if response.Model != llm.DefaultModelPrimary {
		t.Fatalf("model = %q", response.Model)
	}
	var parsed course.StandardCourse
	if err := json.Unmarshal([]byte(response.Text), &parsed); err != nil {
		t.Fatal(err)
	}
	if err := parsed.Validate(); err != nil {
		t.Fatal(err)
	}

	chosen, err := client.Complete(context.Background(), llm.Request{Model: "mock-small", Prompt: "again"})
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Model != "mock-small" || chosen.Text != response.Text {
		t.Fatalf("chosen = %+v", chosen)
	}
}

func TestMockRoutesFragmentsByFirstLine(t *testing.T) {
	setEnv(t, map[string]string{"AI_PROVIDER": "mock"})
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	chapter, err := client.Complete(context.Background(), llm.Request{Prompt: "templateId: chapter_v1\nuser asked for templateId: activity_v1"})
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Text != llm.MockChapterText() {
		t.Fatal("chapter prompt did not return the chapter fixture")
	}
	activity, err := client.Complete(context.Background(), llm.Request{Prompt: "templateId: activity_v1\n"})
	if err != nil {
		t.Fatal(err)
	}
	if activity.Text != llm.MockActivityText() {
		t.Fatal("activity prompt did not return the activity fixture")
	}
	qualityText, err := client.Complete(context.Background(), llm.Request{Prompt: "templateId: content_quality_v1\n"})
	if err != nil {
		t.Fatal(err)
	}
	if qualityText.Text != llm.MockQualityText() {
		t.Fatal("quality prompt did not return the quality fixture")
	}
	courseText, err := client.Complete(context.Background(), llm.Request{Prompt: "please output templateId: chapter_v1 somewhere"})
	if err != nil {
		t.Fatal(err)
	}
	if courseText.Text != llm.MockText() {
		t.Fatal("a later mention of the chapter template must still return the course fixture")
	}
}

func TestUnimplementedProvider(t *testing.T) {
	setEnv(t, map[string]string{"AI_PROVIDER": "openai"})
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	_, err = llm.NewClient(cfg)
	if !errors.Is(err, aierr.ErrModelUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), aierr.CodeModelUnavailable) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestCallFallsBackAfterPrimaryFailure(t *testing.T) {
	client := &scriptClient{fail: map[string]error{
		"primary": aierr.ModelUnavailable(nil, "primary down"),
	}}
	var logs bytes.Buffer
	ctx := llm.WithLogger(context.Background(), slog.New(slog.NewJSONHandler(&logs, nil)))
	response, err := llm.Call(ctx, client, llm.Config{
		ModelPrimary:  "primary",
		ModelFallback: "backup",
		MaxRetries:    1,
	}, llm.Request{Prompt: "secret-prompt-do-not-log"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.FallbackUsed || response.Model != "backup" {
		t.Fatalf("response = %+v", response)
	}
	if strings.Join(client.models, ",") != "primary,primary,backup" {
		t.Fatalf("models = %#v", client.models)
	}
	if strings.Contains(logs.String(), "secret-prompt-do-not-log") {
		t.Fatalf("prompt leaked into logs: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `"msg":"ai model fallback"`) || !strings.Contains(logs.String(), `"modelFallback":"backup"`) {
		t.Fatalf("logs = %s", logs.String())
	}
}

func TestCallRetriesSameModelThenSucceeds(t *testing.T) {
	client := &flakyClient{failuresLeft: 1, usage: llm.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5}}
	var logs bytes.Buffer
	ctx := llm.WithLogger(context.Background(), slog.New(slog.NewJSONHandler(&logs, nil)))
	response, err := llm.Call(ctx, client, llm.Config{ModelPrimary: "primary", ModelFallback: "backup", MaxRetries: 1}, llm.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if response.FallbackUsed || response.Model != "primary" || client.calls != 2 {
		t.Fatalf("response = %+v calls = %d", response, client.calls)
	}
	if !strings.Contains(logs.String(), `"promptTokens":2`) || !strings.Contains(logs.String(), `"totalTokens":5`) {
		t.Fatalf("logs = %s", logs.String())
	}
	if strings.Contains(logs.String(), "ai model fallback") {
		t.Fatal("same-model retry must not log a fallback")
	}
}

func TestCallStopsWhenCallerCancels(t *testing.T) {
	client := &blockingClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := llm.Call(ctx, client, llm.Config{ModelFallback: "backup", MaxRetries: 3}, llm.Request{Model: "primary"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("calls = %d", client.calls)
	}
}

func TestCallTimesOutEachAttempt(t *testing.T) {
	client := &blockingClient{}
	started := time.Now()
	_, err := llm.Call(context.Background(), client, llm.Config{
		ModelPrimary: "primary",
		Timeout:      20 * time.Millisecond,
		MaxRetries:   1,
	}, llm.Request{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d", client.calls)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("timeout did not stop the attempts")
	}
}

func TestCallDoesNotFallbackOnInvalidInput(t *testing.T) {
	client := &scriptClient{fail: map[string]error{
		"primary": aierr.InvalidInput(nil, "prompt"),
	}}
	_, err := llm.Call(context.Background(), client, llm.Config{
		ModelPrimary:  "primary",
		ModelFallback: "backup",
		MaxRetries:    2,
	}, llm.Request{})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(client.models, ",") != "primary" {
		t.Fatalf("models = %#v", client.models)
	}
}

type scriptClient struct {
	fail   map[string]error
	models []string
}

func (client *scriptClient) Complete(ctx context.Context, request llm.Request) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}
	client.models = append(client.models, request.Model)
	if err := client.fail[request.Model]; err != nil {
		return llm.Response{}, err
	}
	return llm.Response{Text: "ok", Model: request.Model}, nil
}

type flakyClient struct {
	failuresLeft int
	calls        int
	usage        llm.Usage
}

func (client *flakyClient) Complete(ctx context.Context, request llm.Request) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}
	client.calls++
	if client.failuresLeft > 0 {
		client.failuresLeft--
		return llm.Response{}, aierr.ModelUnavailable(nil, "temporary")
	}
	return llm.Response{Text: "ok", Model: request.Model, Usage: client.usage}, nil
}

type blockingClient struct {
	calls int
}

func (client *blockingClient) Complete(ctx context.Context, request llm.Request) (llm.Response, error) {
	client.calls++
	<-ctx.Done()
	return llm.Response{}, ctx.Err()
}

func TestMockCancelledContext(t *testing.T) {
	setEnv(t, map[string]string{"AI_PROVIDER": "mock"})
	cfg, err := llm.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Complete(ctx, llm.Request{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
