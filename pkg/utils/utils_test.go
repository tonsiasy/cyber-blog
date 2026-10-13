package utils_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
)

func TestPasswordHashing(t *testing.T) {
	rawPassword := "SuperSecret123!"

	hash, err := utils.HashPassword(rawPassword)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == rawPassword {
		t.Fatalf("Password was not hashed!")
	}

	if !utils.CheckPassword(hash, rawPassword) {
		t.Fatalf("CheckPassword expected true for correct password")
	}

	if utils.CheckPassword(hash, "WrongPassword") {
		t.Fatalf("CheckPassword expected false for wrong password")
	}
}

func TestJWTGenerationAndParsing(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := uint(42)
	username := "vitalik"

	// 1. 测试成功签发与解析
	token, err := utils.GenerateToken(userID, username, secret, 2)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != userID || claims.Username != username {
		t.Fatalf("Claims mismatch: got userID=%d, username=%s", claims.UserID, claims.Username)
	}

	// 2. 测试错误密钥校验失败
	_, err = utils.ParseToken(token, "wrong-secret-key")
	if err == nil {
		t.Fatalf("Expected ParseToken to fail with wrong secret")
	}

	// 3. 测试过期 Token 校验失败
	expiredClaims := utils.CustomClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	expiredToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte(secret))
	_, err = utils.ParseToken(expiredToken, secret)
	if err == nil {
		t.Fatalf("Expected ParseToken to fail on expired token")
	}
}
