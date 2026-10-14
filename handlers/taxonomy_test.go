package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
)

func TestTaxonomyAndSeriesFlow(t *testing.T) {
	testDB := "test_taxonomy_flow.db"
	defer os.Remove(testDB)

	r := setupTestRouter(t, testDB)
	secret := "integration-test-secret-key-999"

	// 1. 准备测试作者与 Token
	userHash, _ := utils.HashPassword("Author123!")
	author := models.User{Username: "szabo", Password: userHash, Email: "szabo@smartcontracts.org"}
	models.DB.Create(&author)
	token, _ := utils.GenerateToken(author.ID, author.Username, secret, 24)

	// 2. 测试创建主题分类 (POST /api/v1/categories)
	catPayload := map[string]string{
		"name":        "Cryptography & Web3",
		"slug":        "crypto-web3",
		"description": "Zero-Knowledge proofs, consensus mechanisms and smart contracts.",
	}
	catBody, _ := json.Marshal(catPayload)
	reqCat, _ := http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(catBody))
	reqCat.Header.Set("Content-Type", "application/json")
	reqCat.Header.Set("Authorization", "Bearer "+token)
	wCat := httptest.NewRecorder()
	r.ServeHTTP(wCat, reqCat)

	if wCat.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on category creation, got %d: %s", wCat.Code, wCat.Body.String())
	}

	var catResp struct {
		Code int             `json:"code"`
		Data models.Category `json:"data"`
	}
	_ = json.Unmarshal(wCat.Body.Bytes(), &catResp)
	categoryID := catResp.Data.ID

	// 3. 测试创建系列专栏 (POST /api/v1/series)
	seriesPayload := map[string]string{
		"title":       "Smart Contracts from Scratch",
		"slug":        "smart-contracts-scratch",
		"description": "A comprehensive deep dive into EVM and Solidity.",
	}
	seriesBody, _ := json.Marshal(seriesPayload)
	reqSeries, _ := http.NewRequest(http.MethodPost, "/api/v1/series", bytes.NewBuffer(seriesBody))
	reqSeries.Header.Set("Content-Type", "application/json")
	reqSeries.Header.Set("Authorization", "Bearer "+token)
	wSeries := httptest.NewRecorder()
	r.ServeHTTP(wSeries, reqSeries)

	if wSeries.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on series creation, got %d: %s", wSeries.Code, wSeries.Body.String())
	}

	var seriesResp struct {
		Code int           `json:"code"`
		Data models.Series `json:"data"`
	}
	_ = json.Unmarshal(wSeries.Body.Bytes(), &seriesResp)
	seriesID := seriesResp.Data.ID

	// 4. 插入 3 篇属于该分类和专栏的文章，乱序传入 order_index (如 Chapter 2, 1, 3)
	postCh2 := models.Post{
		Title:      "Chapter 2: Solidity Syntax",
		Content:    "Variables, control structures, and modifiers.",
		UserID:     author.ID,
		CategoryID: &categoryID,
		SeriesID:   &seriesID,
		OrderIndex: 2,
	}
	models.DB.Create(&postCh2)

	postCh1 := models.Post{
		Title:      "Chapter 1: EVM Fundamentals",
		Content:    "Opcodes, gas mechanics, and storage slots.",
		UserID:     author.ID,
		CategoryID: &categoryID,
		SeriesID:   &seriesID,
		OrderIndex: 1,
	}
	models.DB.Create(&postCh1)

	postCh3 := models.Post{
		Title:      "Chapter 3: Security & Reentrancy",
		Content:    "Checks-Effects-Interactions pattern.",
		UserID:     author.ID,
		CategoryID: &categoryID,
		SeriesID:   &seriesID,
		OrderIndex: 3,
	}
	models.DB.Create(&postCh3)

	// 5. 测试分类列表及其文章聚合计数 (GET /api/v1/categories)
	reqListCat, _ := http.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	wListCat := httptest.NewRecorder()
	r.ServeHTTP(wListCat, reqListCat)
	if wListCat.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on list categories, got %d", wListCat.Code)
	}

	var listCatResp struct {
		Code int `json:"code"`
		Data []struct {
			models.Category
			PostCount int64 `json:"post_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wListCat.Body.Bytes(), &listCatResp)
	if len(listCatResp.Data) == 0 || listCatResp.Data[0].PostCount != 3 {
		t.Fatalf("Expected post_count 3 for category, got: %+v", listCatResp.Data)
	}
	t.Logf("Categories list verified with post_count=%d", listCatResp.Data[0].PostCount)

	// 6. 测试按分类别名检索文章 (GET /api/v1/categories/crypto-web3/posts)
	reqCatPosts, _ := http.NewRequest(http.MethodGet, "/api/v1/categories/crypto-web3/posts", nil)
	wCatPosts := httptest.NewRecorder()
	r.ServeHTTP(wCatPosts, reqCatPosts)
	if wCatPosts.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on category posts retrieval, got %d", wCatPosts.Code)
	}

	var catPostsResp struct {
		Code int `json:"code"`
		Data struct {
			Total int           `json:"total"`
			Posts []models.Post `json:"posts"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCatPosts.Body.Bytes(), &catPostsResp)
	if catPostsResp.Data.Total != 3 {
		t.Fatalf("Expected 3 posts in category, got %d", catPostsResp.Data.Total)
	}

	// 7. 测试查询专栏章节大纲并验证 order_index 升序排列 (GET /api/v1/series/smart-contracts-scratch)
	reqSeriesOutline, _ := http.NewRequest(http.MethodGet, "/api/v1/series/smart-contracts-scratch", nil)
	wSeriesOutline := httptest.NewRecorder()
	r.ServeHTTP(wSeriesOutline, reqSeriesOutline)
	if wSeriesOutline.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on series outline, got %d", wSeriesOutline.Code)
	}

	var seriesOutlineResp struct {
		Code int `json:"code"`
		Data struct {
			ChapterCount int           `json:"chapter_count"`
			Chapters     []models.Post `json:"chapters"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSeriesOutline.Body.Bytes(), &seriesOutlineResp)
	if seriesOutlineResp.Data.ChapterCount != 3 {
		t.Fatalf("Expected 3 chapters in series, got %d", seriesOutlineResp.Data.ChapterCount)
	}

	// 验证章节按 order_index 严格升序 (1, 2, 3)
	chapters := seriesOutlineResp.Data.Chapters
	if chapters[0].OrderIndex != 1 || chapters[1].OrderIndex != 2 || chapters[2].OrderIndex != 3 {
		t.Fatalf("Chapters are not sorted by order_index ascending: [%d, %d, %d]",
			chapters[0].OrderIndex, chapters[1].OrderIndex, chapters[2].OrderIndex)
	}
	t.Logf("Series chapter outline verified strictly sorted: Chapter 1 -> 2 -> 3")

	// 8. 测试不存在的分类与专栏 slug 返回 404
	reqCat404, _ := http.NewRequest(http.MethodGet, "/api/v1/categories/non-existent-slug/posts", nil)
	wCat404 := httptest.NewRecorder()
	r.ServeHTTP(wCat404, reqCat404)
	if wCat404.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for missing category slug, got %d", wCat404.Code)
	}

	reqSeries404, _ := http.NewRequest(http.MethodGet, "/api/v1/series/non-existent-slug", nil)
	wSeries404 := httptest.NewRecorder()
	r.ServeHTTP(wSeries404, reqSeries404)
	if wSeries404.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 for missing series slug, got %d", wSeries404.Code)
	}
}
