package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage 提供本地文件系统的对象存储实现。
// 用于本地无网络环境开发、单元测试与离线演示，与 R2Storage 实现完全相同的 StorageService 接口。
type LocalStorage struct {
	baseDir      string
	publicPrefix string
}

// NewLocalStorage 构造本地文件存储实例
func NewLocalStorage(baseDir, publicPrefix string) *LocalStorage {
	return &LocalStorage{
		baseDir:      baseDir,
		publicPrefix: strings.TrimRight(publicPrefix, "/"),
	}
}

// Upload 将文件流保存到本地磁盘目录
func (l *LocalStorage) Upload(ctx context.Context, body io.Reader, objectKey string, contentType string) (string, error) {
	cleanKey := strings.TrimPrefix(objectKey, "/")
	targetPath := filepath.Join(l.baseDir, filepath.FromSlash(cleanKey))

	// 确保父级目录存在
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return "", fmt.Errorf("创建存储目录失败: %w", err)
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("创建本地目标文件失败: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, body); err != nil {
		return "", fmt.Errorf("写入本地文件内容失败: %w", err)
	}

	return l.GetPublicURL(cleanKey), nil
}

// GetPresignedPutURL 本地模式下直接返回本地相对上传路由（用于模拟直传）
func (l *LocalStorage) GetPresignedPutURL(ctx context.Context, objectKey string, contentType string, expiry time.Duration) (string, error) {
	cleanKey := strings.TrimPrefix(objectKey, "/")
	return fmt.Sprintf("%s/%s", l.publicPrefix, cleanKey), nil
}

// GetPublicURL 返回本地静态路由访问路径
func (l *LocalStorage) GetPublicURL(objectKey string) string {
	cleanKey := strings.TrimPrefix(objectKey, "/")
	return fmt.Sprintf("%s/%s", l.publicPrefix, cleanKey)
}
