package aierr_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/platform/errs"
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
	if !strings.Contains(err.Error(), aierr.CodeOutputParseFailed) || !strings.Contains(err.Error(), "no JSON object") {
		t.Fatalf("error text = %q", err.Error())
	}
}

func TestToAppErrorMapsStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{name: "invalid input", err: aierr.InvalidInput(nil, "topic"), status: errs.SystemBadRequest, msg: aierr.ErrInvalidInput.Msg},
		{name: "too long", err: aierr.InputTooLong(nil, "topic"), status: errs.SystemBadRequest, msg: aierr.ErrInputTooLong.Msg},
		{name: "injection", err: aierr.PromptInjectionSuspect(nil, "ignore previous"), status: errs.SystemBadRequest, msg: aierr.ErrPromptInjectionSuspect.Msg},
		{name: "parse", err: aierr.OutputParseFailed(nil, "invalid JSON"), status: errs.SystemBadRequest, msg: aierr.ErrOutputParseFailed.Msg},
		{name: "schema", err: aierr.OutputSchemaFailed(nil, "title: required"), status: errs.SystemBadRequest, msg: aierr.ErrOutputSchemaFailed.Msg},
		{name: "unavailable", err: aierr.ModelUnavailable(nil, "provider=openai"), status: errs.SystemInternal, msg: errs.ErrInternal.Msg},
		{name: "timeout", err: aierr.Timeout(nil, "60s"), status: errs.SystemInternal, msg: errs.ErrInternal.Msg},
		{name: "internal", err: aierr.Internal(nil, "bug"), status: errs.SystemInternal, msg: errs.ErrInternal.Msg},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appError := aierr.ToAppError(test.err)
			if appError.Code != test.status || appError.Msg != test.msg {
				t.Fatalf("app error = %+v, want status %d msg %q", appError, test.status, test.msg)
			}
			aiError, ok := aierr.As(test.err)
			if !ok || !strings.Contains(aiError.Error(), aiError.Code) {
				t.Fatalf("missing aiCode in %v", test.err)
			}
			for _, detail := range aiError.Details {
				if strings.Contains(appError.Msg, detail) {
					t.Fatalf("user message leaked detail %q: %q", detail, appError.Msg)
				}
			}
		})
	}

	timeoutErr := aierr.Timeout(nil, "60s")
	if !errors.Is(aierr.ToAppError(timeoutErr), aierr.ErrTimeout) {
		t.Fatal("500 conversion should keep the AI error as cause")
	}
	if errors.Is(aierr.ToAppError(aierr.InvalidInput(nil)), aierr.ErrInvalidInput) {
		t.Fatal("400 conversion drops aiCode; match the original error instead")
	}
}
