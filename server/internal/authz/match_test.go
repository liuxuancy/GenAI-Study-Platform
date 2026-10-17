package authz

import (
	"encoding/json"
	"os"
	"testing"
)

// 与前端共用的用例文件
const sharedCasesPath = "../../../docs/permission-cases.json"

type permissionCase struct {
	Granted  string `json:"granted"`
	Required string `json:"required"`
	Match    bool   `json:"match"`
	Note     string `json:"note"`
}

func loadSharedCases(t *testing.T) []permissionCase {
	t.Helper()
	content, err := os.ReadFile(sharedCasesPath)
	if err != nil {
		t.Fatalf("read shared cases: %v", err)
	}
	var file struct {
		Cases []permissionCase `json:"cases"`
	}
	if err := json.Unmarshal(content, &file); err != nil {
		t.Fatalf("parse shared cases: %v", err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("shared cases file has no cases")
	}
	return file.Cases
}

func TestMatchSharedCases(t *testing.T) {
	for _, testCase := range loadSharedCases(t) {
		got := Match(testCase.Granted, testCase.Required)
		if got != testCase.Match {
			t.Errorf("Match(%q, %q) = %v, want %v (%s)",
				testCase.Granted, testCase.Required, got, testCase.Match, testCase.Note)
		}
	}
}

func TestValidCode(t *testing.T) {
	valid := []string{"teacher:course:view", "teacher:*", "teacher:course:*", "*", "a"}
	invalid := []string{"", ":", "teacher:", ":teacher", "teacher::view", "teacher:*:view", "teacher:cour*", "**", "teacher:**"}
	for _, code := range valid {
		if !ValidCode(code) {
			t.Errorf("ValidCode(%q) = false, want true", code)
		}
	}
	for _, code := range invalid {
		if ValidCode(code) {
			t.Errorf("ValidCode(%q) = true, want false", code)
		}
	}
}

func TestIsWildcard(t *testing.T) {
	cases := map[string]bool{
		"*":                   true,
		"teacher:*":           true,
		"teacher:course:*":    true,
		"teacher:course:view": false,
	}
	for code, want := range cases {
		if got := IsWildcard(code); got != want {
			t.Errorf("IsWildcard(%q) = %v, want %v", code, got, want)
		}
	}
}
