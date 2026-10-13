package models_test

import (
	"os"
	"testing"

	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
)

func TestInitDBAndAutoMigrate(t *testing.T) {
	testDBPath := "test_cyber_blog.db"
	defer os.Remove(testDBPath)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver:     "sqlite",
			SQLitePath: testDBPath,
		},
	}

	db, err := models.InitDB(cfg)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// 1. 测试创建用户
	user := models.User{
		Username: "satoshi",
		Password: "hashed_secret_password",
		Email:    "satoshi@bitcoin.org",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("Expected non-zero user ID")
	}

	// 2. 测试创建分类与专栏
	category := models.Category{
		Name: "Web3 Engineering",
		Slug: "web3",
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	series := models.Series{
		Title:  "Mastering Go Internals",
		Slug:   "mastering-go",
		UserID: user.ID,
	}
	if err := db.Create(&series).Error; err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// 3. 测试创建文章，验证自动生成的 Slug 钩子与关联
	post := models.Post{
		Title:      "Genesis Block Explained",
		Content:    "# The Genesis Block\nBitcoin was born.",
		UserID:     user.ID,
		CategoryID: &category.ID,
		SeriesID:   &series.ID,
		OrderIndex: 1,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}
	if post.Slug == "" {
		t.Fatalf("Expected non-empty auto-generated slug on post")
	}
	t.Logf("Created post with auto-generated slug: %s", post.Slug)

	// 4. 测试发表评论
	comment := models.Comment{
		Content: "Fascinating architectural write-up!",
		UserID:  user.ID,
		PostID:  post.ID,
	}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}

	// 5. 验证外键级联删除：删除 Post，附随的 Comment 必须被自动级联删除
	if err := db.Delete(&post).Error; err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	var commentCount int64
	db.Model(&models.Comment{}).Where("post_id = ?", post.ID).Count(&commentCount)
	if commentCount != 0 {
		t.Fatalf("Expected 0 comments after cascading post deletion, got %d", commentCount)
	}
	t.Logf("Cascade deletion verified: comments linked to post deleted successfully")

	// 6. 验证外键保留策略：删除 Series，Post 的 series_id 应变为 NULL (或保留原文章)
	post2 := models.Post{
		Title:    "Another Article in Series",
		Content:  "Content here",
		UserID:   user.ID,
		SeriesID: &series.ID,
	}
	if err := db.Create(&post2).Error; err != nil {
		t.Fatalf("Failed to create post2: %v", err)
	}

	if err := db.Delete(&series).Error; err != nil {
		t.Fatalf("Failed to delete series: %v", err)
	}

	var fetchedPost models.Post
	if err := db.First(&fetchedPost, post2.ID).Error; err != nil {
		t.Fatalf("Expected post2 to still exist after series deletion: %v", err)
	}
	if fetchedPost.SeriesID != nil {
		t.Fatalf("Expected series_id to be set to NULL after series deletion, got %v", *fetchedPost.SeriesID)
	}
	t.Logf("Set Null verified: post retained with series_id nullified")
}
