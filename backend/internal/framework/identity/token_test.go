package identity

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

// fixedClock 可手动拨动的时钟。
type fixedClock struct {
	current time.Time
}

func (clock *fixedClock) Now() time.Time { return clock.current }

func newTestIssuer() (*TokenIssuer, *fixedClock) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	issuer := NewTokenIssuer(testSecret, time.Hour)
	issuer.now = clock.Now
	return issuer, clock
}

func TestIssueAndParse(t *testing.T) {
	issuer, clock := newTestIssuer()
	token, expiresAt, err := issuer.Issue(42, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.Equal(clock.current.Add(time.Hour)) {
		t.Errorf("expiresAt = %v", expiresAt)
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.TokenVersion != 3 {
		t.Errorf("claims = %+v", claims)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	issuer, clock := newTestIssuer()
	token, _, err := issuer.Issue(42, 0)
	if err != nil {
		t.Fatal(err)
	}
	clock.current = clock.current.Add(time.Hour + time.Second)
	if _, err := issuer.Parse(token); err == nil {
		t.Error("expired token accepted")
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	issuer, _ := newTestIssuer()
	token, _, err := issuer.Issue(42, 0)
	if err != nil {
		t.Fatal(err)
	}
	other := NewTokenIssuer("another-secret", time.Hour)
	other.now = issuer.now
	if _, err := other.Parse(token); err == nil {
		t.Error("token signed with another secret accepted")
	}
}

// 伪造的 token：alg=none 不签名、换用其他算法、签发方不对、缺少过期时间、没有用户 ID，都必须拒绝。
func TestParseRejectsForgedTokens(t *testing.T) {
	issuer, clock := newTestIssuer()
	validClaims := TokenClaims{
		UserID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuerName,
			ExpiresAt: jwt.NewNumericDate(clock.current.Add(time.Hour)),
		},
	}

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	hs512Token := sign(t, jwt.SigningMethodHS512, validClaims)

	wrongIssuer := validClaims
	wrongIssuer.Issuer = "someone-else"
	noExpiry := validClaims
	noExpiry.ExpiresAt = nil
	noUser := validClaims
	noUser.UserID = 0

	cases := map[string]string{
		"alg none":     noneToken,
		"HS512":        hs512Token,
		"wrong issuer": sign(t, jwt.SigningMethodHS256, wrongIssuer),
		"no expiry":    sign(t, jwt.SigningMethodHS256, noExpiry),
		"no user id":   sign(t, jwt.SigningMethodHS256, noUser),
		"garbage":      "not-a-jwt",
		"empty":        "",
	}
	for name, token := range cases {
		if _, err := issuer.Parse(token); err == nil {
			t.Errorf("%s: forged token accepted", name)
		}
	}
}

func sign(t *testing.T, method jwt.SigningMethod, claims TokenClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}
