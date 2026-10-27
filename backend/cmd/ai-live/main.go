package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
)

func main() {
	if err := loadLocalEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "读取 .env 失败: %v\n", err)
		os.Exit(1)
	}
	service, err := ai.NewFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 AI 服务失败: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()
	fmt.Println("1/4 生成整课")
	generated, meta, err := service.GenerateCourse(ctx, ai.GenerateCourseInput{
		Topic:            "Git 入门",
		Audience:         course.AudienceNonCS,
		ChapterCountHint: 2,
	})
	if err != nil {
		fail("GenerateCourse", meta, err)
	}
	printMeta("整课", meta)
	fmt.Printf("  标题=%s 章节=%d\n", generated.Title, len(generated.Chapters))

	fmt.Println("2/4 生成单章")
	chapter, meta, err := service.GenerateChapter(ctx, ai.GenerateChapterInput{
		Topic:            "分支与合并",
		Audience:         course.AudienceNonCS,
		TargetChapterKey: "ch-live-1",
		ContextCourse:    &generated,
	})
	if err != nil {
		fail("GenerateChapter", meta, err)
	}
	printMeta("单章", meta)
	fmt.Printf("  key=%s activities=%d\n", chapter.Key, len(chapter.Activities))

	fmt.Println("3/4 生成 activity")
	activity, meta, err := service.GenerateActivity(ctx, ai.GenerateActivityInput{
		Topic:             "查看仓库状态",
		Audience:          course.AudienceNonCS,
		TargetChapterKey:  "ch-live-1",
		TargetActivityKey: "act-live-1",
		ContextCourse:     &generated,
	})
	if err != nil {
		fail("GenerateActivity", meta, err)
	}
	printMeta("activity", meta)
	fmt.Printf("  key=%s type=%s\n", activity.Key, activity.Type)

	fmt.Println("4/4 质量评估")
	report, meta, err := service.EvaluateCourse(ctx, generated, "看练习是否具体")
	if err != nil {
		fail("EvaluateCourse", meta, err)
	}
	printMeta("评估", meta)
	fmt.Printf("  overallScore=%d dimensions=%d suggestions=%d\n", report.OverallScore, len(report.Dimensions), len(report.Suggestions))
	fmt.Println("四项真实调用都成功")
}

func printMeta(name string, meta ai.Meta) {
	fmt.Printf("  %s model=%s fallback=%v latency=%s tokens=%d warnings=%d\n",
		name, meta.ModelUsed, meta.FallbackUsed, time.Duration(meta.LatencyMs)*time.Millisecond, meta.Usage.TotalTokens, len(meta.Warnings))
	if len(meta.Warnings) > 0 {
		fmt.Printf("  warnings=%s\n", strings.Join(meta.Warnings, "; "))
	}
}

func fail(name string, meta ai.Meta, err error) {
	code := ""
	if aiError, ok := aierr.As(err); ok {
		code = aiError.Code
	}
	fmt.Fprintf(os.Stderr, "%s 失败 code=%s err=%v\nmeta=%+v\n", name, code, err, meta)
	os.Exit(1)
}

func loadLocalEnv() error {
	for _, path := range []string{".env", "backend/.env"} {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			if key == "" || os.Getenv(key) != "" {
				continue
			}
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
		return scanner.Err()
	}
	return fmt.Errorf("没有找到 .env。请在 backend/.env 写入 AI_PROVIDER、AI_API_KEY 和 AI_MODEL_PRIMARY，且不要提交这个文件")
}
