package cos

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// 验证 PDF 预签名 URL 同时包含 disposition 与 content-type（SDK 会写入 q-url-param-list）。
func TestPresignedPDFInlineGetURLQueryParams(t *testing.T) {
	cfg := Config{
		Bucket:    os.Getenv("COS_BUCKET"),
		Region:    os.Getenv("COS_REGION"),
		SecretID:  os.Getenv("COS_SECRET_ID"),
		SecretKey: os.Getenv("COS_SECRET_KEY"),
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		t.Skip("COS credentials not set")
	}
	if cfg.Bucket == "" {
		cfg.Bucket = defaultBucket
	}
	if cfg.Region == "" {
		cfg.Region = "ap-hongkong"
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := client.PresignedPDFInlineGetURL(
		context.Background(),
		"courses/1/originals/test.pdf",
		"sample.pdf",
		15*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if !strings.Contains(q.Get("response-content-disposition"), "inline") {
		t.Fatalf("disposition = %q", q.Get("response-content-disposition"))
	}
	if q.Get("response-content-type") != "application/pdf" {
		t.Fatalf("content-type = %q", q.Get("response-content-type"))
	}
	paramList := q.Get("q-url-param-list")
	if !strings.Contains(paramList, "response-content-disposition") || !strings.Contains(paramList, "response-content-type") {
		t.Fatalf("q-url-param-list = %q, want both signed params", paramList)
	}
}
