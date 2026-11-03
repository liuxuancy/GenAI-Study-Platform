package cos_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"caveclass/internal/platform/cos"
)

func loadDotEnvForTest(t *testing.T) {
	t.Helper()
	for _, path := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range bytes.Split(data, []byte("\n")) {
			text := string(bytes.TrimSpace(line))
			if text == "" || text[0] == '#' {
				continue
			}
			key, value, ok := bytes.Cut([]byte(text), []byte("="))
			if !ok {
				continue
			}
			k := string(bytes.TrimSpace(key))
			v := string(bytes.TrimSpace(value))
			v = stringsTrimQuotes(v)
			if k != "" && os.Getenv(k) == "" {
				t.Setenv(k, v)
			}
		}
		return
	}
}

func stringsTrimQuotes(s string) string {
	s = stringsTrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

func stringsTrimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func TestCOSUploadHeadDownloadDelete(t *testing.T) {
	loadDotEnvForTest(t)
	cfg := cos.LoadConfig()
	if !cfg.IsConfigured() {
		t.Skip("COS_SECRET_ID/COS_SECRET_KEY not set; skip live COS test")
	}
	client, err := cos.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	key, err := cos.OriginalObjectKey(1, "integration.txt")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("integration test")
	t.Cleanup(func() { _ = client.Delete(context.Background(), key) })

	if err := client.PutBytes(ctx, key, "text/plain", body); err != nil {
		t.Fatalf("put: %v", err)
	}
	_, ok, err := client.Head(ctx, key)
	if err != nil || !ok {
		t.Fatalf("head: ok=%v err=%v", ok, err)
	}
	got, err := client.GetBytes(ctx, key)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("get: err=%v equal=%v", err, bytes.Equal(got, body))
	}
	if err := client.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
