package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
)

func TestPostCRUDAndOwnership(t *testing.T) {
	testDB := "test_post_crud.db"
	defer os.Remove(testDB)

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	// 1. 在数据库中准备两个用户：Alice (作者) 和 Bob (非作者)
	aliceHash, _ := utils.HashPassword("AlicePass123!")
	alice := models.User{Username: "alice", Password: aliceHash, Email: "alice@cyber.org"}
	models.DB.Create(&alice)

	bobHash, _ := utils.HashPassword("BobPass123!")
	bob := models.User{Username: "bob", Password: bobHash, Email: "bob@cyber.org"}
	models.DB.Create(&bob)

	// 签发 Token
	aliceToken, _ := utils.GenerateToken(alice.ID, alice.Username, secret, 24)
	bobToken, _ := utils.GenerateToken(bob.ID, bob.Username, secret, 24)

	// 2. 测试未登录用户发布文章 -> 401 Unauthorized
	postPayload := map[string]interface{}{
		"title":   "Decentralized Web Architecture",
		"content": "Why decentralization matters in Web3.",
	}
	body, _ := json.Marshal(postPayload)

	reqUnauth, _ := http.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBuffer(body))
	reqUnauth.Header.Set("Content-Type", "application/json")
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for unauthenticated post creation, got %d", wUnauth.Code)
	}

	// 3. 测试 Alice 创建文章 -> 201 Created
	reqAliceCreate, _ := http.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBuffer(body))
	reqAliceCreate.Header.Set("Content-Type", "application/json")
	reqAliceCreate.Header.Set("Authorization", "Bearer "+aliceToken)
	wAliceCreate := httptest.NewRecorder()
	r.ServeHTTP(wAliceCreate, reqAliceCreate)

	if wAliceCreate.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on Alice post creation, got %d: %s", wAliceCreate.Code, wAliceCreate.Body.String())
	}

	var createdPostResp struct {
		Code int         `json:"code"`
		Data models.Post `json:"data"`
	}
	_ = json.Unmarshal(wAliceCreate.Body.Bytes(), &createdPostResp)
	postID := createdPostResp.Data.ID
	if postID == 0 || createdPostResp.Data.UserID != alice.ID {
		t.Fatalf("Post creation verification failed: id=%d, userID=%d", postID, createdPostResp.Data.UserID)
	}
	t.Logf("Alice successfully created post with ID=%d, Slug=%s", postID, createdPostResp.Data.Slug)

	// 4. 测试公开获取文章列表 -> 200 OK
	reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/posts?page=1&page_size=5", nil)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on post list query, got %d", wList.Code)
	}

	// 5. 测试公开获取单篇文章详情 -> 200 OK
	reqGet, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", postID), nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on get post by id, got %d", wGet.Code)
	}

	// 6. 测试 Bob (非作者) 尝试修改 Alice 的文章 -> 403 Forbidden
	updatePayload := map[string]interface{}{
		"title": "Hacked Title by Bob",
	}
	updateBody, _ := json.Marshal(updatePayload)

	reqBobUpdate, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/posts/%d", postID), bytes.NewBuffer(updateBody))
	reqBobUpdate.Header.Set("Content-Type", "application/json")
	reqBobUpdate.Header.Set("Authorization", "Bearer "+bobToken)
	wBobUpdate := httptest.NewRecorder()
	r.ServeHTTP(wBobUpdate, reqBobUpdate)

	if wBobUpdate.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-author attempts update, got %d: %s", wBobUpdate.Code, wBobUpdate.Body.String())
	}
	t.Logf("Non-author modification correctly rejected with 403 Forbidden")

	// 7. 测试修改不存在的文章 ID -> 404 Not Found
	reqMissingUpdate, _ := http.NewRequest(http.MethodPut, "/api/v1/posts/99999", bytes.NewBuffer(updateBody))
	reqMissingUpdate.Header.Set("Content-Type", "application/json")
	reqMissingUpdate.Header.Set("Authorization", "Bearer "+aliceToken)
	wMissingUpdate := httptest.NewRecorder()
	r.ServeHTTP(wMissingUpdate, reqMissingUpdate)

	if wMissingUpdate.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found on missing post update, got %d", wMissingUpdate.Code)
	}

	// 8. 测试 Alice (作者) 成功更新文章 -> 200 OK
	aliceUpdatePayload := map[string]interface{}{
		"title":   "Decentralized Web Architecture (Updated)",
		"content": "Updated content with Layer 2 details.",
	}
	aliceUpdateBody, _ := json.Marshal(aliceUpdatePayload)

	reqAliceUpdate, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/posts/%d", postID), bytes.NewBuffer(aliceUpdateBody))
	reqAliceUpdate.Header.Set("Content-Type", "application/json")
	reqAliceUpdate.Header.Set("Authorization", "Bearer "+aliceToken)
	wAliceUpdate := httptest.NewRecorder()
	r.ServeHTTP(wAliceUpdate, reqAliceUpdate)

	if wAliceUpdate.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on author post update, got %d", wAliceUpdate.Code)
	}

	// 9. 测试 Bob (非作者) 尝试删除 Alice 的文章 -> 403 Forbidden
	reqBobDelete, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", postID), nil)
	reqBobDelete.Header.Set("Authorization", "Bearer "+bobToken)
	wBobDelete := httptest.NewRecorder()
	r.ServeHTTP(wBobDelete, reqBobDelete)

	if wBobDelete.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-author attempts delete, got %d", wBobDelete.Code)
	}
	t.Logf("Non-author deletion correctly rejected with 403 Forbidden")

	// 10. 测试 Alice (作者) 成功删除自己的文章 -> 200 OK
	reqAliceDelete, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", postID), nil)
	reqAliceDelete.Header.Set("Authorization", "Bearer "+aliceToken)
	wAliceDelete := httptest.NewRecorder()
	r.ServeHTTP(wAliceDelete, reqAliceDelete)

	if wAliceDelete.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on author post delete, got %d", wAliceDelete.Code)
	}

	// 11. 验证删除后再次查询该文章 -> 404 Not Found
	reqAfterDelete, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", postID), nil)
	wAfterDelete := httptest.NewRecorder()
	r.ServeHTTP(wAfterDelete, reqAfterDelete)

	if wAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for deleted post, got %d", wAfterDelete.Code)
	}
	t.Logf("Post successfully deleted and verified")
}
