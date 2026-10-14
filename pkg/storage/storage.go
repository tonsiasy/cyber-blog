package storage

import (
	"context"
	"io"
	"time"

	"github.com/tonsiasy/cyber-blog/config"
)

// StorageService 定义对象存储操作的统一接口规范。
// 无论是生产环境的 Cloudflare R2 (S3 兼容协议)，还是本地离线测试使用的 LocalStorage，
// 都实现该接口，实现控制反转与依赖解耦。
type StorageService interface {
	// Upload 上传文件流，存储并返回其公开访问的 URL
	Upload(ctx context.Context, body io.Reader, objectKey string, contentType string) (string, error)

	// GetPresignedPutURL 生成用于客户端直接向存储上传的预签名 PUT URL（限时有效）
	GetPresignedPutURL(ctx context.Context, objectKey string, contentType string, expiry time.Duration) (string, error)

	// GetPublicURL 根据对象在存储桶中的路径 key 获取公网可访问的 URL
	GetPublicURL(objectKey string) string
}

// NewStorageService 工厂函数：依据配置决定实例化 Cloudflare R2 存储还是降级使用本地磁盘存储。
// 当 R2 的必要认证信息（AccountID, AccessKeyID, SecretAccessKey, BucketName）均已提供时，
// 启用 R2 生产存储服务；否则自动使用 LocalStorage，确保本地开发、离线环境与自动化测试不受阻断。
func NewStorageService(ctx context.Context, cfg *config.Config) (StorageService, error) {
	if cfg.R2.AccountID != "" && cfg.R2.AccessKeyID != "" && cfg.R2.SecretAccessKey != "" && cfg.R2.BucketName != "" {
		return NewR2Storage(ctx, cfg.R2)
	}

	// 降级为本地文件存储
	return NewLocalStorage("./uploads", "/uploads"), nil
}
