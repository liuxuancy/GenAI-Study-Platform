package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
	"caveclass/internal/platform/errs"
)

const playgroundUID int64 = 0

type apiResult struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

type providerBody struct {
	Provider string `json:"provider"`
}

type generateCourseBody struct {
	Provider          string `json:"provider"`
	Topic             string `json:"topic"`
	Audience          string `json:"audience"`
	Language          string `json:"language"`
	ExtraRequirements string `json:"extraRequirements"`
	ChapterCountHint  int    `json:"chapterCountHint"`
	Model             string `json:"model"`
}

type generateChapterBody struct {
	Provider          string                 `json:"provider"`
	Topic             string                 `json:"topic"`
	Audience          string                 `json:"audience"`
	Language          string                 `json:"language"`
	ExtraRequirements string                 `json:"extraRequirements"`
	TargetChapterKey  string                 `json:"targetChapterKey"`
	ContextCourse     *course.StandardCourse `json:"contextCourse"`
	Model             string                 `json:"model"`
}

type generateActivityBody struct {
	Provider          string                 `json:"provider"`
	Topic             string                 `json:"topic"`
	Audience          string                 `json:"audience"`
	Language          string                 `json:"language"`
	ExtraRequirements string                 `json:"extraRequirements"`
	TargetChapterKey  string                 `json:"targetChapterKey"`
	TargetActivityKey string                 `json:"targetActivityKey"`
	ContextCourse     *course.StandardCourse `json:"contextCourse"`
	Model             string                 `json:"model"`
}

type evaluateBody struct {
	Provider string                `json:"provider"`
	Course   course.StandardCourse `json:"course"`
	Rubric   string                `json:"rubric"`
}

func main() {
	envErr := loadLocalEnv()
	// 先加载 .env 再初始化日志，LOG_ 环境变量才生效
	logConfig := logs.LoadConfigFromEnv()
	if logConfig.ServiceName == "" {
		logConfig.ServiceName = "ai-playground"
	}
	logs.Init(logConfig)
	defer logs.Shutdown(context.Background())

	if envErr != nil {
		logs.CtxWarn(context.Background(), "警告: %v（仍可用 mock）", envErr)
	}
	baseCfg, err := llm.LoadConfig()
	if err != nil {
		exitWithError("读取 AI 配置失败: %v", err)
	}
	staticRoot, err := findStaticRoot()
	if err != nil {
		exitWithError("%v", err)
	}
	addr := getenv("AI_PLAYGROUND_ADDR", "127.0.0.1:8765")

	mux := http.NewServeMux()
	mux.HandleFunc("/playground/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		writeOK(w, llm.ProviderAvailability(baseCfg))
	})
	mux.HandleFunc("/playground/generate-course", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateCourse(w, r, baseCfg)
	})
	mux.HandleFunc("/playground/generate-chapter", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateChapter(w, r, baseCfg)
	})
	mux.HandleFunc("/playground/generate-activity", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateActivity(w, r, baseCfg)
	})
	mux.HandleFunc("/playground/evaluate", func(w http.ResponseWriter, r *http.Request) {
		handleEvaluate(w, r, baseCfg)
	})
	mux.Handle("/", http.FileServer(http.Dir(staticRoot)))

	logs.CtxInfo(context.Background(), "AI playground 静态目录: %s", staticRoot)
	logs.CtxInfo(context.Background(), "在浏览器打开 http://%s/", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		exitWithError("playground exited: %v", err)
	}
}

// exitWithError 打一条错误日志、把没发完的日志发完，再以状态码 1 退出（os.Exit 不会执行 defer）。
func exitWithError(format string, args ...any) {
	logs.CtxError(context.Background(), format, args...)
	_ = logs.Shutdown(context.Background())
	os.Exit(1)
}

func handleGenerateCourse(w http.ResponseWriter, r *http.Request, base llm.Config) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body generateCourseBody
	if !decodeJSON(w, r, &body) {
		return
	}
	service, err := newServiceForProvider(base, normalizeProvider(body.Provider))
	if err != nil {
		writeAIError(w, ai.Meta{}, err)
		return
	}
	ctx := r.Context()
	generated, meta, err := service.GenerateCourse(ctx, ai.GenerateCourseInput{
		Topic:             body.Topic,
		Audience:          body.Audience,
		Language:          body.Language,
		ExtraRequirements: body.ExtraRequirements,
		ChapterCountHint:  body.ChapterCountHint,
		Model:             body.Model,
	})
	if err != nil {
		writeAIError(w, meta, err)
		return
	}
	writeOK(w, map[string]any{"course": generated, "meta": meta})
}

func handleGenerateChapter(w http.ResponseWriter, r *http.Request, base llm.Config) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body generateChapterBody
	if !decodeJSON(w, r, &body) {
		return
	}
	service, err := newServiceForProvider(base, normalizeProvider(body.Provider))
	if err != nil {
		writeAIError(w, ai.Meta{}, err)
		return
	}
	chapter, meta, err := service.GenerateChapter(r.Context(), ai.GenerateChapterInput{
		Topic:             body.Topic,
		Audience:          body.Audience,
		Language:          body.Language,
		ExtraRequirements: body.ExtraRequirements,
		TargetChapterKey:  body.TargetChapterKey,
		ContextCourse:     body.ContextCourse,
		Model:             body.Model,
	})
	if err != nil {
		writeAIError(w, meta, err)
		return
	}
	writeOK(w, map[string]any{"chapter": chapter, "meta": meta})
}

func handleGenerateActivity(w http.ResponseWriter, r *http.Request, base llm.Config) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body generateActivityBody
	if !decodeJSON(w, r, &body) {
		return
	}
	service, err := newServiceForProvider(base, normalizeProvider(body.Provider))
	if err != nil {
		writeAIError(w, ai.Meta{}, err)
		return
	}
	activity, meta, err := service.GenerateActivity(r.Context(), ai.GenerateActivityInput{
		Topic:             body.Topic,
		Audience:          body.Audience,
		Language:          body.Language,
		ExtraRequirements: body.ExtraRequirements,
		TargetChapterKey:  body.TargetChapterKey,
		TargetActivityKey: body.TargetActivityKey,
		ContextCourse:     body.ContextCourse,
		Model:             body.Model,
	})
	if err != nil {
		writeAIError(w, meta, err)
		return
	}
	writeOK(w, map[string]any{"activity": activity, "meta": meta})
}

func handleEvaluate(w http.ResponseWriter, r *http.Request, base llm.Config) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body evaluateBody
	if !decodeJSON(w, r, &body) {
		return
	}
	service, err := newServiceForProvider(base, normalizeProvider(body.Provider))
	if err != nil {
		writeAIError(w, ai.Meta{}, err)
		return
	}
	report, meta, err := service.EvaluateCourse(r.Context(), body.Course, body.Rubric)
	if err != nil {
		writeAIError(w, meta, err)
		return
	}
	writeOK(w, map[string]any{"report": report, "meta": meta})
}

func normalizeProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return llm.ProviderMock
	}
	return provider
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeFail(w, errs.SystemBadRequest, "invalid request: "+err.Error())
		return false
	}
	return true
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, apiResult{Code: errs.SystemOK, Msg: "ok", Data: data})
}

func writeFail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, apiResult{Code: code, Msg: msg})
}

func writeAIError(w http.ResponseWriter, meta ai.Meta, err error) {
	ai.LogError(context.Background(), err, meta, playgroundUID)
	appError := aierr.ToAppError(err)
	if appError == nil {
		writeFail(w, errs.SystemInternal, "internal error")
		return
	}
	writeJSON(w, appError.Code, apiResult{Code: appError.Code, Msg: appError.Msg})
}

func writeJSON(w http.ResponseWriter, status int, payload apiResult) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logs.CtxError(context.Background(), "写响应失败: %v", err)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	writeFail(w, http.StatusMethodNotAllowed, "method not allowed")
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func findStaticRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(wd, "docs", "UI", "b-ai"),
		filepath.Join(wd, "..", "docs", "UI", "b-ai"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("找不到 docs/UI/b-ai/index.html，请在仓库根目录或 backend 目录启动")
}
