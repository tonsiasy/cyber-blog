package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/pkg/storage"
)

func TestLocalStorage_UploadAndPublicURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cyber-blog-storage-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ls := storage.NewLocalStorage(tempDir, "/uploads")
	ctx := context.Background()

	content := "fake image bytes content"
	reader := bytes.NewReader([]byte(content))
	key := "2026/10/avatar.png"

	url, err := ls.Upload(ctx, reader, key, "image/png")
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}

	expectedURL := "/uploads/2026/10/avatar.png"
	if url != expectedURL {
		t.Errorf("期望 URL 为 %s, 实际得到 %s", expectedURL, url)
	}

	// 验证本地磁盘文件确实写入成功
	savedPath := filepath.Join(tempDir, "2026", "10", "avatar.png")
	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("读取磁盘写入文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容不匹配: 期望 %s, 实际 %s", content, string(data))
	}

	// 测试预签名模拟 URL
	presignedURL, err := ls.GetPresignedPutURL(ctx, key, "image/png", 10*time.Minute)
	if err != nil {
		t.Fatalf("获取本地预签名 URL 失败: %v", err)
	}
	if presignedURL != expectedURL {
		t.Errorf("期望预签名 URL 为 %s, 实际得到 %s", expectedURL, presignedURL)
	}
}

func TestNewStorageService_FallbackToLocal(t *testing.T) {
	cfg := &config.Config{
		R2: config.R2Config{
			AccountID:       "",
			AccessKeyID:     "",
			SecretAccessKey: "",
			BucketName:      "",
		},
	}

	svc, err := storage.NewStorageService(context.Background(), cfg)
	if err != nil {
		t.Fatalf("构造 StorageService 失败: %v", err)
	}

	if _, ok := svc.(*storage.LocalStorage); !ok {
		t.Errorf("期望空配置降级为 *LocalStorage, 实际得到 %T", svc)
	}
}

func TestNewR2Storage_Validation(t *testing.T) {
	ctx := context.Background()

	// 缺少必要字段应报错
	_, err := storage.NewR2Storage(ctx, config.R2Config{
		AccountID: "my-account-id",
	})
	if err == nil {
		t.Errorf("期望缺少参数时返回错误，实际未报错")
	}

	// 提供完整必要参数应成功创建
	r2Cfg := config.R2Config{
		AccountID:       "test-account-123",
		AccessKeyID:     "test-key-id",
		SecretAccessKey: "test-secret-key",
		BucketName:      "test-bucket",
		PublicDomain:    "https://media.cyber-blog.com",
	}

	r2Storage, err := storage.NewR2Storage(ctx, r2Cfg)
	if err != nil {
		t.Fatalf("初始化 R2Storage 失败: %v", err)
	}

	publicURL := r2Storage.GetPublicURL("uploads/cover.jpg")
	expectedURL := "https://media.cyber-blog.com/uploads/cover.jpg"
	if publicURL != expectedURL {
		t.Errorf("期望公共域名 URL 为 %s, 实际得到 %s", expectedURL, publicURL)
	}

	// 测试无 PublicDomain 时的 fallback 格式
	r2CfgNoDomain := config.R2Config{
		AccountID:       "test-account-123",
		AccessKeyID:     "test-key-id",
		SecretAccessKey: "test-secret-key",
		BucketName:      "test-bucket",
		PublicDomain:    "",
	}
	r2StorageNoDomain, err := storage.NewR2Storage(ctx, r2CfgNoDomain)
	if err != nil {
		t.Fatalf("初始化 R2Storage (无域名) 失败: %v", err)
	}

	fallbackURL := r2StorageNoDomain.GetPublicURL("uploads/cover.jpg")
	if !strings.Contains(fallbackURL, "test-account-123.r2.cloudflarestorage.com/test-bucket/uploads/cover.jpg") {
		t.Errorf("期望默认 fallback URL 包含 bucket 和 key, 实际为 %s", fallbackURL)
	}
}
