// 密码哈希（bcrypt）：存储时只存哈希，校验时重新计算比较。

package identity

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

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

// CompareWithDummyHash 用户不存在时也做一次同样耗时的比较，
// 让"用户不存在"和"密码错误"的响应时间一致，避免被用来探测哪些用户名存在。
func CompareWithDummyHash(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
}

// dummyHash 第一次用到时才生成，之后复用同一个
var dummyHash = sync.OnceValue(generateDummyHash)

func generateDummyHash() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	return hash
}

// UseFastHashForTests 把计算成本降到最低，只供测试使用。
func UseFastHashForTests() {
	bcryptCost = bcrypt.MinCost
}
