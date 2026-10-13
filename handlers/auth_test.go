package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/routes"
)

func setupTestRouter(t *testing.T, dbPath string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: dbPath,
		},
		JWT: config.JWTConfig{
			Secret:          "integration-test-secret-key-999",
			ExpirationHours: 24,
		},
	}

	_, err := models.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	r := gin.New()
	routes.SetupRouter(r, cfg)
	return r
}

func TestAuthFlow(t *testing.T) {
	testDB := "test_auth_flow.db"
	defer os.Remove(testDB)

	r := setupTestRouter(t, testDB)

	// 1. 测试正常用户注册
	regPayload := map[string]string{
		"username": "nakamoto",
		"password": "Password123!",
		"email":    "nakamoto@cyberblog.io",
	}
	body, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on registration, got %d: %s", w.Code, w.Body.String())
	}

	// 2. 测试重复用户名注册 (预期 409 Conflict)
	reqDup, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	r.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict on duplicate username, got %d: %s", wDup.Code, wDup.Body.String())
	}

	// 3. 测试错误密码登录 (预期 401 Unauthorized)
	wrongLoginPayload := map[string]string{
		"username": "nakamoto",
		"password": "WrongPassword!",
	}
	bodyWrong, _ := json.Marshal(wrongLoginPayload)
	reqWrong, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(bodyWrong))
	reqWrong.Header.Set("Content-Type", "application/json")
	wWrong := httptest.NewRecorder()
	r.ServeHTTP(wWrong, reqWrong)

	if wWrong.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized on wrong password, got %d: %s", wWrong.Code, wWrong.Body.String())
	}

	// 4. 测试正确密码登录并获取 Token (预期 200 OK)
	loginPayload := map[string]string{
		"username": "nakamoto",
		"password": "Password123!",
	}
	bodyLogin, _ := json.Marshal(loginPayload)
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(bodyLogin))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on login, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil || loginResp.Data.Token == "" {
		t.Fatalf("Failed to parse JWT token from login response: %s", wLogin.Body.String())
	}
	token := loginResp.Data.Token
	t.Logf("Obtained valid JWT Token: %s...", token[:15])

	// 5. 测试无 Token 访问受保护接口 /api/v1/auth/me (预期 401 Unauthorized)
	reqNoToken, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	wNoToken := httptest.NewRecorder()
	r.ServeHTTP(wNoToken, reqNoToken)

	if wNoToken.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized without token, got %d", wNoToken.Code)
	}

	// 6. 测试携带有效 Token 访问受保护接口 /api/v1/auth/me (预期 200 OK)
	reqWithToken, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqWithToken.Header.Set("Authorization", "Bearer "+token)
	wWithToken := httptest.NewRecorder()
	r.ServeHTTP(wWithToken, reqWithToken)

	if wWithToken.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with valid token, got %d: %s", wWithToken.Code, wWithToken.Body.String())
	}

	// 7. 测试作业规范兼容路径: POST /api/login
	reqLegacy, _ := http.NewRequest(http.MethodPost, "/api/login", bytes.NewBuffer(bodyLogin))
	reqLegacy.Header.Set("Content-Type", "application/json")
	wLegacy := httptest.NewRecorder()
	r.ServeHTTP(wLegacy, reqLegacy)

	if wLegacy.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on legacy route /api/login, got %d", wLegacy.Code)
	}
	t.Logf("Legacy assignment routes verified successfully!")
}
