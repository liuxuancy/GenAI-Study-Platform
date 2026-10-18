// JWT 登录凭证：只放用户 ID 和 token 版本，不放权限码，权限每次请求按用户 ID 查询。

package identity

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenIssuerName = "caveclass"

// TokenClaims token 里携带的内容。
type TokenClaims struct {
	UserID       int64 `json:"uid"`
	TokenVersion int   `json:"tv"` // 必须等于 sys_user.token_version，否则 token 已失效
	jwt.RegisteredClaims
}

// TokenIssuer 负责签发和解析 token。
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time // 测试时替换成假时钟
}

// NewTokenIssuer 全局只初始化一次：启动时由 main.go 用 JWT_SECRET 和 JWT_TTL 创建。
func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Issue 为用户签发一个 token，返回 token 和过期时间。
func (issuer *TokenIssuer) Issue(userID int64, tokenVersion int) (string, time.Time, error) {
	issuedAt := issuer.now()
	expiresAt := issuedAt.Add(issuer.ttl)
	claims := TokenClaims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuerName,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(issuer.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse 校验签名、签发方和过期时间，返回 token 里的内容。只接受 HS256。
func (issuer *TokenIssuer) Parse(token string) (*TokenClaims, error) {
	var claims TokenClaims
	_, err := jwt.ParseWithClaims(token, &claims, issuer.signingKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // 拒绝 alg=none 等其他算法
		jwt.WithIssuer(tokenIssuerName),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(issuer.now),
	)
	if err != nil {
		return nil, err
	}
	if claims.UserID <= 0 {
		return nil, errors.New("token has no user id")
	}
	return &claims, nil
}

// signingKey 供 jwt 库在校验签名时取密钥。
func (issuer *TokenIssuer) signingKey(*jwt.Token) (any, error) {
	return issuer.secret, nil
}
