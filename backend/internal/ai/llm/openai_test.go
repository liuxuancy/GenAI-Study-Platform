package llm_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/llm"
)

func TestCompatibleClientReadsChatCompletion(t *testing.T) {
	var sawAuth, sawModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") == "Bearer test-key"
		body, _ := io.ReadAll(r.Body)
		sawModel = strings.Contains(string(body), `"model":"deepseek-chat"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"deepseek-chat",
			"choices":[{"message":{"role":"assistant","content":"{\"ok\":true}"}}],
			"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}
		}`))
	}))
	defer server.Close()

	client := llm.NewCompatibleClient(server.URL, "test-key", "deepseek-chat")
	response, err := client.Complete(context.Background(), llm.Request{Prompt: "return json"})
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || !sawModel {
		t.Fatalf("auth=%v model=%v", sawAuth, sawModel)
	}
	if response.Text != `{"ok":true}` || response.Model != "deepseek-chat" || response.Usage.TotalTokens != 5 {
		t.Fatalf("response = %+v", response)
	}
}

func TestCompatibleClientMapsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"busy"}}`, http.StatusTooManyRequests)
	}))
	defer server.Close()
	client := llm.NewCompatibleClient(server.URL, "test-key", "deepseek-chat")
	_, err := client.Complete(context.Background(), llm.Request{})
	if !errors.Is(err, aierr.ErrModelUnavailable) {
		t.Fatalf("err = %v", err)
	}

	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer denied.Close()
	client = llm.NewCompatibleClient(denied.URL, "test-key", "deepseek-chat")
	_, err = client.Complete(context.Background(), llm.Request{})
	if !errors.Is(err, aierr.ErrInternal) || errors.Is(err, aierr.ErrModelUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeepSeekClientRequiresKey(t *testing.T) {
	_, err := llm.NewClient(llm.Config{Provider: llm.ProviderDeepSeek})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
