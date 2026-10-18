package identity

import (
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	UseFastHashForTests()
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "admin123" {
		t.Fatal("hash must not equal the plain password")
	}
	if !CheckPassword(hash, "admin123") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "admin124") || CheckPassword(hash, "") {
		t.Error("wrong password accepted")
	}
}

// 同一个密码哈希两次，结果应该不同（随机盐），但都能通过校验。
func TestHashPasswordUsesRandomSalt(t *testing.T) {
	UseFastHashForTests()
	first, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("two hashes of the same password are identical; salt is not random")
	}
	if !CheckPassword(first, "admin123") || !CheckPassword(second, "admin123") {
		t.Error("both hashes should verify")
	}
}

func TestHashPasswordRejectsTooLong(t *testing.T) {
	UseFastHashForTests()
	if _, err := HashPassword(strings.Repeat("a", 73)); err == nil {
		t.Error("expected error for password longer than 72 bytes")
	}
}
