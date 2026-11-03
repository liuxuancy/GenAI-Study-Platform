package main

import (
	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
)

func newServiceForProvider(base llm.Config, provider string) (*ai.Service, error) {
	return ai.NewForProvider(base, provider)
}
