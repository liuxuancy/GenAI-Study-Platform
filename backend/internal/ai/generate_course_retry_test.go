package ai_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
)

type courseSequenceClient struct {
	outputs []string
	calls   int
	cancel  context.CancelFunc
}

func (client *courseSequenceClient) Complete(_ context.Context, request llm.Request) (llm.Response, error) {
	index := client.calls
	client.calls++
	if client.cancel != nil {
		client.cancel()
	}
	if index >= len(client.outputs) {
		index = len(client.outputs) - 1
	}
	return llm.Response{
		Text: client.outputs[index], Model: request.Model,
		Usage: llm.Usage{PromptTokens: 3, CompletionTokens: 7, TotalTokens: 10},
	}, nil
}

func TestGenerateCourseRetriesUnbalancedOutput(t *testing.T) {
	client := &courseSequenceClient{outputs: []string{`{"chapters":[`, llm.MockText()}}
	service, err := ai.New(llm.Config{Provider: llm.ProviderTeamily, ModelPrimary: "teamily", MaxRetries: 1}, client)
	if err != nil {
		t.Fatal(err)
	}
	result, meta, err := service.GenerateCourse(context.Background(), ai.GenerateCourseInput{Topic: "Git", Audience: course.AudienceGeneral})
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 2 || result.Validate() != nil || meta.Usage.TotalTokens != 20 || !strings.Contains(strings.Join(meta.Warnings, ","), "retried") {
		t.Fatalf("calls=%d result=%+v meta=%+v", client.calls, result, meta)
	}
}

func TestGenerateCourseRetryExhaustionAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name       string
		maxRetries int
		cancel     bool
		wantCalls  int
		wantError  error
	}{
		{"disabled", 0, false, 1, aierr.ErrOutputParseFailed},
		{"exhausted", 2, false, 3, aierr.ErrOutputParseFailed},
		{"cancelled", 2, true, 1, aierr.ErrTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &courseSequenceClient{outputs: []string{`{"chapters":[`}}
			if test.cancel {
				client.cancel = cancel
			}
			service, err := ai.New(llm.Config{MaxRetries: test.maxRetries}, client)
			if err != nil {
				t.Fatal(err)
			}
			result, meta, err := service.GenerateCourse(ctx, ai.GenerateCourseInput{Topic: "Git", Audience: course.AudienceGeneral})
			if !errors.Is(err, test.wantError) || client.calls != test.wantCalls || len(result.Chapters) != 0 || meta.Usage.TotalTokens != 10*test.wantCalls {
				t.Fatalf("calls=%d result=%+v meta=%+v err=%v", client.calls, result, meta, err)
			}
		})
	}
}

func TestGenerateCourseDoesNotRetrySchemaFailure(t *testing.T) {
	client := &courseSequenceClient{outputs: []string{`{"title":"Missing chapters"}`}}
	service, err := ai.New(llm.Config{MaxRetries: 2}, client)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GenerateCourse(context.Background(), ai.GenerateCourseInput{Topic: "Git", Audience: course.AudienceGeneral})
	if !errors.Is(err, aierr.ErrOutputSchemaFailed) || client.calls != 1 {
		t.Fatalf("calls=%d err=%v", client.calls, err)
	}
}
