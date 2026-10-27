package llm_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/llm"
)

func TestTeamilyClientReadsOutputText(t *testing.T) {
	var sawAuth, sawWait, sawKey bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") == "Bearer test-key"
		sawKey = r.Header.Get("Idempotency-Key") != ""
		body, _ := io.ReadAll(r.Body)
		sawWait = strings.Contains(string(body), `"wait":true`) && strings.Contains(string(body), `"mode":"new"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"completed",
			"output":{"text":"{\"ok\":true}","artifacts":[]},
			"usage":{"input_tokens":4,"output_tokens":2,"credits":1}
		}`))
	}))
	defer server.Close()

	client := llm.NewTeamilyClient(server.URL, "test-key", "teamily")
	response, err := client.Complete(context.Background(), llm.Request{Prompt: "return json"})
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || !sawWait || !sawKey {
		t.Fatalf("auth=%v wait=%v key=%v", sawAuth, sawWait, sawKey)
	}
	if response.Text != `{"ok":true}` || response.Model != "teamily" || response.Usage.TotalTokens != 6 {
		t.Fatalf("response = %+v", response)
	}
}

func TestTeamilyClientPollsUntilCompleted(t *testing.T) {
	var server *httptest.Server
	polls := 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"status":"running","poll_url":%q}`, server.URL)
			return
		}
		polls++
		w.Header().Set("Content-Type", "application/json")
		if polls == 1 {
			_, _ = w.Write([]byte(`{"status":"running","usage":{"input_tokens":8,"output_tokens":3}}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"completed","output":{"text":"{\"ok\":true}"}}`))
	}))
	defer server.Close()
	client := llm.NewTeamilyClient(server.URL, "test-key", "teamily")
	response, err := client.Complete(context.Background(), llm.Request{Prompt: "course"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != `{"ok":true}` || response.Usage.PromptTokens != 8 || response.Usage.TotalTokens != 11 {
		t.Fatalf("response = %+v", response)
	}
}

func TestTeamilyClientMapsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"busy"}}`, http.StatusTooManyRequests)
	}))
	defer server.Close()
	client := llm.NewTeamilyClient(server.URL, "test-key", "teamily")
	_, err := client.Complete(context.Background(), llm.Request{})
	if !errors.Is(err, aierr.ErrModelUnavailable) {
		t.Fatalf("err = %v", err)
	}

	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"failed","output":{"text":""}}`))
	}))
	defer denied.Close()
	client = llm.NewTeamilyClient(denied.URL, "test-key", "teamily")
	_, err = client.Complete(context.Background(), llm.Request{})
	if !errors.Is(err, aierr.ErrModelUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestTeamilyClientRequiresKey(t *testing.T) {
	_, err := llm.NewClient(llm.Config{Provider: llm.ProviderTeamily})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
