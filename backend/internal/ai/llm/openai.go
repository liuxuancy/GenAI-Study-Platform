package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"caveclass/internal/ai/aierr"
)

// NewCompatibleClient 调用 OpenAI 兼容的 chat completions。DeepSeek 使用这个客户端。
func NewCompatibleClient(endpoint, apiKey, model string) Client {
	return &compatibleClient{
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

type compatibleClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

type chatRequest struct {
	Model          string           `json:"model"`
	Messages       []chatMessage    `json:"messages"`
	Temperature    float64          `json:"temperature"`
	ResponseFormat *chatResponseFmt `json:"response_format,omitempty"`
}

type chatResponseFmt struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (client *compatibleClient) Complete(ctx context.Context, request Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	model := request.Model
	if model == "" {
		model = client.model
	}
	body := chatRequest{
		Model:       model,
		Messages:    []chatMessage{{Role: "user", Content: request.Prompt}},
		Temperature: 0.2,
	}
	if request.JSONResponse {
		body.ResponseFormat = &chatResponseFmt{Type: "json_object"}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, aierr.Internal(err, "encode chat request")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Response{}, aierr.Internal(err, "build chat request")
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+client.apiKey)
	httpClient := client.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		return Response{}, err
	}
	defer httpResponse.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(httpResponse.Body, 4<<20))
	if err != nil {
		return Response{}, err
	}
	var parsed chatResponse
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil && httpResponse.StatusCode == http.StatusOK {
			return Response{}, aierr.OutputParseFailed(err, "chat response is not JSON")
		}
	}
	if httpResponse.StatusCode != http.StatusOK {
		return Response{}, statusError(httpResponse.StatusCode, parsed, raw)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return Response{}, aierr.ModelUnavailable(fmt.Errorf("empty completion"), "model="+model)
	}
	used := parsed.Model
	if used == "" {
		used = model
	}
	return Response{
		Text:  parsed.Choices[0].Message.Content,
		Model: used,
		Usage: Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}, nil
}

func statusError(status int, parsed chatResponse, raw []byte) error {
	message := ""
	if parsed.Error != nil {
		message = parsed.Error.Message
	}
	if message == "" {
		message = strings.TrimSpace(string(raw))
	}
	if len(message) > 300 {
		message = message[:300]
	}
	detail := fmt.Sprintf("status=%d %s", status, message)
	if status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status >= http.StatusInternalServerError {
		return aierr.ModelUnavailable(fmt.Errorf("chat completion failed"), detail)
	}
	return aierr.Internal(fmt.Errorf("chat completion failed"), detail)
}
