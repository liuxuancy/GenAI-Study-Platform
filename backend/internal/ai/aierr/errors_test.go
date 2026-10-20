package aierr_test

import (
	"errors"
	"fmt"
	"testing"

	"caveclass/internal/ai/aierr"
)

func TestIsMatchesByCode(t *testing.T) {
	err := aierr.OutputParseFailed(nil, "no JSON object")
	if !errors.Is(err, aierr.ErrOutputParseFailed) {
		t.Fatal("parse error should match ErrOutputParseFailed")
	}
	if errors.Is(err, aierr.ErrOutputSchemaFailed) {
		t.Fatal("parse error must not match schema error")
	}
	wrapped := fmt.Errorf("generate course: %w", err)
	if !errors.Is(wrapped, aierr.ErrOutputParseFailed) {
		t.Fatal("match should survive wrapping")
	}
	if !errors.Is(err, err) {
		t.Error("error should match itself")
	}
}
