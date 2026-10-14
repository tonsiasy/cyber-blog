package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
)

func TestMediaUpload_AuthRequired(t *testing.T) {
	testDB := "test_media_auth.db"
	defer os.Remove(testDB)
	defer os.RemoveAll("./uploads")

	r := setupTestRouter(t, testDB)

	// 1. 未携带 Token 请求上传接口应被拒绝 (401)
	reqUpload, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload", nil)
	wUpload := httptest.NewRecorder()
	r.ServeHTTP(wUpload, reqUpload)
	if wUpload.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for unauthenticated upload, got %d", wUpload.Code)
	}

	// 2. 未携带 Token 请求预签名接口应被拒绝 (401)
	reqPresign, _ := http.NewRequest(http.MethodPost, "/api/v1/media/presign", nil)
	wPresign := httptest.NewRecorder()
	r.ServeHTTP(wPresign, reqPresign)
	if wPresign.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for unauthenticated presign, got %d", wPresign.Code)
	}
}

func TestMediaUpload_DirectUploadAndStaticAccess(t *testing.T) {
	testDB := "test_media_upload.db"
	defer os.Remove(testDB)
	defer os.RemoveAll("./uploads")

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	// 准备用户与 Token
	userHash, _ := utils.HashPassword("Author123!")
	user := models.User{Username: "uploader", Password: userHash, Email: "uploader@cyber.dev"}
	models.DB.Create(&user)
	token, _ := utils.GenerateToken(user.ID, user.Username, secret, 24)

	// 构建 multipart/form-data 请求体
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("创建 multipart form file 失败: %v", err)
	}
	// 写入 1x1 真实 PNG 格式魔数字节
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00}
	_, _ = part.Write(pngMagic)
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on upload, got %d: %s", w.Code, w.Body.String())
	}

	var uploadResp struct {
		Code int `json:"code"`
		Data struct {
			URL         string `json:"url"`
			Key         string `json:"key"`
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("解析上传响应失败: %v", err)
	}

	if uploadResp.Data.URL == "" || uploadResp.Data.Key == "" {
		t.Errorf("返回的 URL 或 Key 为空: %+v", uploadResp.Data)
	}
	if !strings.HasSuffix(uploadResp.Data.Key, ".png") {
		t.Errorf("对象 Key 后缀应为 .png, 实际为 %s", uploadResp.Data.Key)
	}

	// 验证本地静态文件路由可以直接 GET 下载
	reqStatic, _ := http.NewRequest(http.MethodGet, uploadResp.Data.URL, nil)
	wStatic := httptest.NewRecorder()
	r.ServeHTTP(wStatic, reqStatic)

	if wStatic.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK accessing static file at %s, got %d", uploadResp.Data.URL, wStatic.Code)
	}
}

func TestMediaUpload_InvalidFileType(t *testing.T) {
	testDB := "test_media_invalid.db"
	defer os.Remove(testDB)
	defer os.RemoveAll("./uploads")

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	token, _ := utils.GenerateToken(1, "tester", secret, 24)

	// 尝试上传非法类型 (纯文本文件或二进制可执行脚本)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "malicious.sh")
	_, _ = part.Write([]byte("#!/bin/bash\necho hack"))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for unsupported file type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMediaPresign_Flow(t *testing.T) {
	testDB := "test_media_presign.db"
	defer os.Remove(testDB)
	defer os.RemoveAll("./uploads")

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	token, _ := utils.GenerateToken(1, "tester", secret, 24)

	// 1. 合法请求预签名直传凭据
	validPayload := map[string]string{
		"filename":     "architecture-diagram.webp",
		"content_type": "image/webp",
	}
	payloadBytes, _ := json.Marshal(validPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/presign", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on presign, got %d: %s", w.Code, w.Body.String())
	}

	var presignResp struct {
		Code int `json:"code"`
		Data struct {
			UploadURL string `json:"upload_url"`
			PublicURL string `json:"public_url"`
			Key       string `json:"key"`
			ExpiresIn int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &presignResp); err != nil {
		t.Fatalf("解析预签名响应失败: %v", err)
	}

	if presignResp.Data.UploadURL == "" || presignResp.Data.PublicURL == "" {
		t.Errorf("预签名返回的 UploadURL 或 PublicURL 为空: %+v", presignResp.Data)
	}
	if presignResp.Data.ExpiresIn != 900 {
		t.Errorf("期望 ExpiresIn 为 900 秒 (15分钟), 实际得到 %d", presignResp.Data.ExpiresIn)
	}

	// 2. 非法类型拒绝
	invalidPayload := map[string]string{
		"filename":     "document.pdf",
		"content_type": "application/pdf",
	}
	invalidBytes, _ := json.Marshal(invalidPayload)

	reqInvalid, _ := http.NewRequest(http.MethodPost, "/api/v1/media/presign", bytes.NewBuffer(invalidBytes))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Authorization", "Bearer "+token)

	wInvalid := httptest.NewRecorder()
	r.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for pdf presign, got %d: %s", wInvalid.Code, wInvalid.Body.String())
	}
}
