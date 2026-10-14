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

func TestCommentFlow(t *testing.T) {
	testDB := "test_comment_flow.db"
	defer os.Remove(testDB)

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	// 1. 准备测试用户和文章
	userHash, _ := utils.HashPassword("Commenter123!")
	user := models.User{Username: "charlie", Password: userHash, Email: "charlie@web3.io"}
	models.DB.Create(&user)
	token, _ := utils.GenerateToken(user.ID, user.Username, secret, 24)

	post := models.Post{
		Title:   "Zero-Knowledge Proofs in Practice",
		Content: "An exploration of zk-SNARKs and zk-STARKs.",
		UserID:  user.ID,
	}
	models.DB.Create(&post)

	// 2. 测试未授权发表评论 -> 401 Unauthorized
	commentPayload := map[string]string{
		"content": "Brilliant introduction to ZKPs!",
	}
	commentBody, _ := json.Marshal(commentPayload)

	reqUnauth, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bytes.NewBuffer(commentBody))
	reqUnauth.Header.Set("Content-Type", "application/json")
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for unauthenticated comment, got %d", wUnauth.Code)
	}

	// 3. 测试对不存在的文章发表评论 -> 404 Not Found
	reqMissing, _ := http.NewRequest(http.MethodPost, "/api/v1/posts/99999/comments", bytes.NewBuffer(commentBody))
	reqMissing.Header.Set("Content-Type", "application/json")
	reqMissing.Header.Set("Authorization", "Bearer "+token)
	wMissing := httptest.NewRecorder()
	r.ServeHTTP(wMissing, reqMissing)
	if wMissing.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for comment on non-existent post, got %d", wMissing.Code)
	}

	// 4. 测试成功发表评论 -> 201 Created
	reqCreate, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), bytes.NewBuffer(commentBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on comment creation, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	var commentResp struct {
		Code int            `json:"code"`
		Data models.Comment `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &commentResp)
	if commentResp.Data.ID == 0 || commentResp.Data.PostID != post.ID || commentResp.Data.UserID != user.ID {
		t.Fatalf("Comment data verification failed: %+v", commentResp.Data)
	}
	if commentResp.Data.User.Username != "charlie" {
		t.Fatalf("Expected preloaded commenter username 'charlie', got %s", commentResp.Data.User.Username)
	}
	t.Logf("Comment successfully created: ID=%d, Author=%s", commentResp.Data.ID, commentResp.Data.User.Username)

	// 5. 测试公开获取文章评论列表 -> 200 OK
	reqList, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), nil)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on comment list query, got %d", wList.Code)
	}

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Total int              `json:"total"`
			Items []models.Comment `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if listResp.Data.Total != 1 || len(listResp.Data.Items) != 1 {
		t.Fatalf("Expected 1 comment in list, got total=%d", listResp.Data.Total)
	}
	t.Logf("Comment list successfully retrieved: %d comment(s)", listResp.Data.Total)

	// 6. 测试大作业兼容路由: POST /api/posts/:id/comments & GET /api/posts/:id/comments
	legacyCommentPayload := map[string]string{
		"content": "Second comment via legacy assignment route.",
	}
	legacyBody, _ := json.Marshal(legacyCommentPayload)
	reqLegacyCreate, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/posts/%d/comments", post.ID), bytes.NewBuffer(legacyBody))
	reqLegacyCreate.Header.Set("Content-Type", "application/json")
	reqLegacyCreate.Header.Set("Authorization", "Bearer "+token)
	wLegacyCreate := httptest.NewRecorder()
	r.ServeHTTP(wLegacyCreate, reqLegacyCreate)

	if wLegacyCreate.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on legacy comment creation, got %d", wLegacyCreate.Code)
	}

	reqLegacyList, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/posts/%d/comments", post.ID), nil)
	wLegacyList := httptest.NewRecorder()
	r.ServeHTTP(wLegacyList, reqLegacyList)

	if wLegacyList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on legacy comment list query, got %d", wLegacyList.Code)
	}
	t.Logf("Legacy comment endpoints verified successfully!")
}
