// cos-probe：阶段 1 连通性探针。在 backend 目录执行 go run ./cmd/cos-probe
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"caveclass/internal/platform/cos"
)

func main() {
	if err := loadDotEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}

	cfg := cos.LoadConfig()
	fmt.Println("COS config:", cfg.RedactedSummary())
	if !cfg.IsConfigured() {
		fmt.Fprintln(os.Stderr, "缺少 COS_SECRET_ID 或 COS_SECRET_KEY。请在 backend/.env 填写（勿提交 Git），参考 backend/.env.example。")
		os.Exit(2)
	}

	client, err := cos.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	const testCourseID int64 = 1 // 探针用虚拟 courseId，非真实课程数据
	objectKey, err := cos.OriginalObjectKey(testCourseID, "phase1-probe.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "object key: %v\n", err)
		os.Exit(1)
	}
	payload := []byte("caveclass cos phase1 probe " + time.Now().UTC().Format(time.RFC3339))

	fmt.Println("upload:", objectKey)
	if err := client.PutBytes(ctx, objectKey, "text/plain; charset=utf-8", payload); err != nil {
		fmt.Fprintf(os.Stderr, "upload failed: %v\n", err)
		os.Exit(1)
	}

	size, exists, err := client.Head(ctx, objectKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "head failed: %v\n", err)
		_ = client.Delete(ctx, objectKey)
		os.Exit(1)
	}
	if !exists {
		fmt.Fprintln(os.Stderr, "head: object not found after upload")
		_ = client.Delete(ctx, objectKey)
		os.Exit(1)
	}
	fmt.Printf("head: exists=true size=%d (expected %d)\n", size, len(payload))

	downloaded, err := client.GetBytes(ctx, objectKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "download failed: %v\n", err)
		_ = client.Delete(ctx, objectKey)
		os.Exit(1)
	}
	if !bytes.Equal(downloaded, payload) {
		fmt.Fprintf(os.Stderr, "content mismatch: got %d bytes\n", len(downloaded))
		_ = client.Delete(ctx, objectKey)
		os.Exit(1)
	}
	fmt.Println("download: content verified")

	if err := client.Delete(ctx, objectKey); err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}
	_, existsAfter, err := client.Head(ctx, objectKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "head after delete: %v\n", err)
		os.Exit(1)
	}
	if existsAfter {
		fmt.Fprintln(os.Stderr, "object still exists after delete")
		os.Exit(1)
	}
	fmt.Println("delete: ok")
	fmt.Println("COS phase1 probe: SUCCESS")
}
