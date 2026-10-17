// Package identity 身份认证（你是谁）：密码哈希；阶段 4 加入 JWT 和认证中间件。
package identity

import "golang.org/x/crypto/bcrypt"

// bcrypt 计算成本，越高越慢越安全。测试中调低以加快速度。
var bcryptCost = bcrypt.DefaultCost

// HashPassword 返回密码的 bcrypt 哈希。密码超过 72 字节时返回错误。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 判断密码是否与哈希匹配。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// UseFastHashForTests 把计算成本降到最低，只供测试使用。
func UseFastHashForTests() {
	bcryptCost = bcrypt.MinCost
}
