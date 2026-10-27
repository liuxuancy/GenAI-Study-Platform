package llm

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"caveclass/internal/ai/aierr"
)

type loggerKey struct{}

// WithLogger 让这一次调用把用量和降级写到指定 logger。没有时用 slog 默认 logger。
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

// Call 按配置调用模型：每个模型先调用 1 次，再重试 MaxRetries 次。
// 主模型在超时、不可用或其它传输错误上失败后，改用备用模型。备用模型成功时 FallbackUsed 为 true。
// 调用方取消 context 时不再重试，也不降级。Timeout 大于 0 时，每次尝试单独计时。
// 成功后写一条用量日志，字段为 model、fallbackUsed、promptTokens、completionTokens、totalTokens。不记录 prompt 和密钥。
func Call(ctx context.Context, client Client, cfg Config, request Request) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		return Response{}, aierr.Internal(nil, "ai client is nil")
	}
	primary := request.Model
	if primary == "" {
		primary = cfg.ModelPrimary
	}
	if primary == "" {
		primary = DefaultModelPrimary
	}
	response, err := attemptModel(ctx, client, cfg, primary, request.Prompt)
	if err == nil {
		logUsage(ctx, response)
		return response, nil
	}
	if cfg.ModelFallback == "" || cfg.ModelFallback == primary || !retryable(ctx, err) {
		logFailure(ctx, primary, err)
		return Response{}, err
	}
	logFallback(ctx, primary, cfg.ModelFallback, err)
	response, fallbackErr := attemptModel(ctx, client, cfg, cfg.ModelFallback, request.Prompt)
	if fallbackErr != nil {
		logFailure(ctx, cfg.ModelFallback, fallbackErr)
		return Response{}, fallbackErr
	}
	response.FallbackUsed = true
	logUsage(ctx, response)
	return response, nil
}

func attemptModel(ctx context.Context, client Client, cfg Config, model, prompt string) (Response, error) {
	attempts := cfg.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if last == nil {
				last = err
			}
			break
		}
		response, err := completeOnce(ctx, client, cfg.Timeout, model, prompt)
		if err == nil {
			return response, nil
		}
		last = err
		if !retryable(ctx, err) {
			break
		}
	}
	if last == nil {
		last = aierr.ModelUnavailable(nil, "model="+model)
	}
	return Response{}, last
}

func completeOnce(ctx context.Context, client Client, timeout time.Duration, model, prompt string) (Response, error) {
	attemptCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		attemptCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	response, err := client.Complete(attemptCtx, Request{Model: model, Prompt: prompt})
	if err != nil {
		return Response{}, err
	}
	if response.Model == "" {
		response.Model = model
	}
	return response, nil
}

func retryable(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	aiError, ok := aierr.As(err)
	if !ok {
		return true
	}
	switch aiError.Code {
	case aierr.CodeTimeout, aierr.CodeModelUnavailable:
		return true
	default:
		return false
	}
}

func logUsage(ctx context.Context, response Response) {
	loggerFrom(ctx).InfoContext(ctx, "ai model usage",
		slog.String("model", response.Model),
		slog.Bool("fallbackUsed", response.FallbackUsed),
		slog.Int("promptTokens", response.Usage.PromptTokens),
		slog.Int("completionTokens", response.Usage.CompletionTokens),
		slog.Int("totalTokens", response.Usage.TotalTokens),
	)
}

func logFallback(ctx context.Context, primary, fallback string, err error) {
	loggerFrom(ctx).WarnContext(ctx, "ai model fallback",
		slog.String("modelRequested", primary),
		slog.String("modelFallback", fallback),
		slog.String("aiCode", codeOf(err)),
	)
}

func logFailure(ctx context.Context, model string, err error) {
	loggerFrom(ctx).ErrorContext(ctx, "ai model call failed",
		slog.String("model", model),
		slog.String("aiCode", codeOf(err)),
	)
}

func codeOf(err error) string {
	if aiError, ok := aierr.As(err); ok {
		return aiError.Code
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return aierr.CodeTimeout
	}
	return aierr.CodeModelUnavailable
}

func loggerFrom(ctx context.Context) *slog.Logger {
	if ctx != nil {
		logger, _ := ctx.Value(loggerKey{}).(*slog.Logger)
		if logger != nil {
			return logger
		}
	}
	return slog.Default()
}
