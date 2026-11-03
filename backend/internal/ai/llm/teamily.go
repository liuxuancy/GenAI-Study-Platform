package llm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"caveclass/internal/ai/aierr"
)

// NewTeamilyClient 调用 Teamily agent responses。每次请求使用新的 thread，并带 Idempotency-Key。
func NewTeamilyClient(endpoint, apiKey, model string) Client {
	return &teamilyClient{
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

type teamilyClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

type teamilyRequest struct {
	Input  string `json:"input"`
	Thread struct {
		Mode string `json:"mode"`
	} `json:"thread"`
	Wait bool `json:"wait"`
}

type teamilyResponse struct {
	Status  string `json:"status"`
	PollURL string `json:"poll_url"`
	Output  struct {
		Text string `json:"text"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (client *teamilyClient) Complete(ctx context.Context, request Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(request.Prompt) > TeamilyMaxInputChars {
		return Response{}, aierr.InputTooLong(nil, fmt.Sprintf("prompt exceeds %d characters", TeamilyMaxInputChars))
	}
	body := teamilyRequest{Input: request.Prompt, Wait: true}
	body.Thread.Mode = "new"
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, aierr.Internal(err, "encode teamily request")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Response{}, aierr.Internal(err, "build teamily request")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+client.apiKey)
	httpRequest.Header.Set("Idempotency-Key", newIdempotencyKey())
	parsed, status, raw, err := client.roundTrip(ctx, httpRequest)
	if err != nil {
		return Response{}, err
	}
	seenUsage := parsed.Usage
	pollURL := parsed.PollURL
	for teamilyPending(status, parsed) {
		if pollURL == "" {
			return Response{}, teamilyStatusError(status, parsed, raw)
		}
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		pollRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			return Response{}, aierr.Internal(err, "build teamily poll")
		}
		pollRequest.Header.Set("Authorization", "Bearer "+client.apiKey)
		parsed, status, raw, err = client.roundTrip(ctx, pollRequest)
		if err != nil {
			return Response{}, err
		}
		if parsed.PollURL != "" {
			pollURL = parsed.PollURL
		}
		if parsed.Usage.InputTokens > 0 || parsed.Usage.OutputTokens > 0 {
			seenUsage = parsed.Usage
		}
	}
	if parsed.Usage.InputTokens == 0 && parsed.Usage.OutputTokens == 0 {
		parsed.Usage = seenUsage
	}
	if status != http.StatusOK {
		return Response{}, teamilyStatusError(status, parsed, raw)
	}
	text := strings.TrimSpace(parsed.Output.Text)
	if parsed.Status != "" && parsed.Status != "completed" {
		return Response{}, aierr.ModelUnavailable(fmt.Errorf("teamily status %s", parsed.Status), "status="+parsed.Status)
	}
	if text == "" {
		return Response{}, aierr.ModelUnavailable(fmt.Errorf("empty teamily output"), "status="+parsed.Status)
	}
	model := request.Model
	if model == "" {
		model = client.model
	}
	if model == "" {
		model = DefaultModelTeamily
	}
	return Response{
		Text:  parsed.Output.Text,
		Model: model,
		Usage: Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}

func teamilyPending(status int, parsed teamilyResponse) bool {
	if strings.TrimSpace(parsed.Output.Text) != "" && parsed.Status == "completed" {
		return false
	}
	switch parsed.Status {
	case "running", "queued", "pending", "in_progress":
		return true
	}
	return status == http.StatusAccepted
}

func (client *teamilyClient) roundTrip(ctx context.Context, httpRequest *http.Request) (teamilyResponse, int, []byte, error) {
	httpClient := client.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil {
			return teamilyResponse{}, 0, nil, ctx.Err()
		}
		return teamilyResponse{}, 0, nil, err
	}
	defer httpResponse.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(httpResponse.Body, 4<<20))
	if err != nil {
		return teamilyResponse{}, httpResponse.StatusCode, nil, err
	}
	var parsed teamilyResponse
	if len(bytes.TrimSpace(raw)) > 0 {
		if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil && httpResponse.StatusCode == http.StatusOK {
			return teamilyResponse{}, httpResponse.StatusCode, raw, aierr.OutputParseFailed(unmarshalErr, "teamily response is not JSON")
		}
	}
	return parsed, httpResponse.StatusCode, raw, nil
}

func teamilyStatusError(status int, parsed teamilyResponse, raw []byte) error {
	message := ""
	if parsed.Error != nil {
		message = parsed.Error.Message
	}
	if message == "" {
		message = strings.TrimSpace(parsed.Output.Text)
	}
	if message == "" {
		message = strings.TrimSpace(string(raw))
	}
	if len(message) > 300 {
		message = message[:300]
	}
	detail := fmt.Sprintf("status=%d %s", status, message)
	lower := strings.ToLower(message)
	if status == http.StatusBadRequest {
		if strings.Contains(lower, "32000") || strings.Contains(lower, "too long") || strings.Contains(lower, "length") || strings.Contains(lower, "limit") {
			return aierr.InputTooLong(fmt.Errorf("teamily request failed"), detail)
		}
		return aierr.InvalidInput(fmt.Errorf("teamily request failed"), detail)
	}
	if status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status >= http.StatusInternalServerError {
		return aierr.ModelUnavailable(fmt.Errorf("teamily request failed"), detail)
	}
	return aierr.Internal(fmt.Errorf("teamily request failed"), detail)
}

func newIdempotencyKey() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "teamily-request"
	}
	return hex.EncodeToString(buf)
}
