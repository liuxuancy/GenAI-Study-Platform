package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
)

func TestNewMetaCopiesLogID(t *testing.T) {
	ctx := logs.WithLogID(context.Background(), "req-42")
	meta := ai.NewMeta(ctx, ai.SceneFullCourse)
	if meta.Scene != ai.SceneFullCourse || meta.LogID != "req-42" {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.Warnings == nil || meta.Validation.RepairedFields == nil {
		t.Fatal("slices should be empty, not nil")
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"logId":"req-42"`)) || !bytes.Contains(encoded, []byte(`"templateId"`)) {
		t.Fatalf("json = %s", encoded)
	}

	var empty *ai.Meta
	empty.FillLogID(ctx)
}

func TestLogErrorFields(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	buffer := &bytes.Buffer{}
	logs.Setup(slog.NewJSONHandler(buffer, nil))

	ctx := logs.WithLogID(context.Background(), "req-7")
	meta := ai.NewMeta(ctx, ai.SceneFullCourse)
	meta.TemplateID = "full_course_v1"
	err := aierr.OutputSchemaFailed(nil, "title: required")
	ai.LogError(ctx, err, meta, 42)

	var line map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &line); err != nil {
		t.Fatalf("not json: %q", buffer.String())
	}
	if line["aiCode"] != aierr.CodeOutputSchemaFailed {
		t.Fatalf("aiCode = %#v", line["aiCode"])
	}
	if line["uid"] != float64(42) {
		t.Fatalf("uid = %#v", line["uid"])
	}
	if line[logs.LogIDKey] != "req-7" {
		t.Fatalf("log_id = %#v", line[logs.LogIDKey])
	}
	details, _ := line["details"].([]any)
	if len(details) != 1 || details[0] != "title: required" {
		t.Fatalf("details = %#v", line["details"])
	}
	loggedMeta, _ := line["meta"].(map[string]any)
	if loggedMeta["logId"] != "req-7" || loggedMeta["scene"] != ai.SceneFullCourse || loggedMeta["templateId"] != "full_course_v1" {
		t.Fatalf("meta = %#v", loggedMeta)
	}
}
