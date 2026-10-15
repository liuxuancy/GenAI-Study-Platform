package errs

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIsMatchesByCode(t *testing.T) {
	err := Forbidden("不是你的课程")
	if !errors.Is(err, ErrForbidden) {
		t.Error("custom-message Forbidden should match ErrForbidden")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("Forbidden must not match ErrNotFound")
	}
	wrapped := fmt.Errorf("update course: %w", err)
	if !errors.Is(wrapped, ErrForbidden) {
		t.Error("match should survive fmt.Errorf wrapping")
	}
}

func TestFrom(t *testing.T) {
	if got := From(fmt.Errorf("ctx: %w", NotFound("课程不存在"))); got.Code != SystemNotFound || got.Msg != "课程不存在" {
		t.Errorf("From kept wrong error: %+v", got)
	}

	cause := errors.New("disk full")
	got := From(cause)
	if got.Code != SystemInternal || got.Msg != ErrInternal.Msg {
		t.Errorf("unknown error should become Internal with generic msg, got %+v", got)
	}
	if !errors.Is(got, cause) {
		t.Error("Internal should keep the cause for logging")
	}
	if !strings.Contains(got.Error(), "disk full") {
		t.Errorf("Error() should include cause for logs: %q", got.Error())
	}
}
