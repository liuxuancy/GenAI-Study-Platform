package cos

import "testing"

func TestOriginalObjectKey(t *testing.T) {
	key, err := OriginalObjectKey(42, "讲义.PDF")
	if err != nil {
		t.Fatal(err)
	}
	if !stringsHasPrefix(key, "courses/42/originals/") {
		t.Fatalf("unexpected key prefix: %q", key)
	}
	if !stringsHasSuffix(key, ".pdf") {
		t.Fatalf("expected .pdf suffix: %q", key)
	}
}

func TestOriginalObjectKeyRejectsBadExt(t *testing.T) {
	if _, err := OriginalObjectKey(1, "evil.exe"); err == nil {
		t.Fatal("expected reject for .exe")
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
