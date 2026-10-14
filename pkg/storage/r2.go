package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tonsiasy/cyber-blog/config"
)

// R2Storage 基于 AWS SDK v2 实现与 Cloudflare R2 对象存储的交互。
// Cloudflare R2 兼容 S3 协议，支持零出站流量费用（Zero Egress Fees）。
type R2Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	publicDomain  string
	accountID     string
}

// NewR2Storage 构造 Cloudflare R2 存储客户端。
func NewR2Storage(ctx context.Context, cfg config.R2Config) (*R2Storage, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("cloudflare R2 配置不完整: account_id, access_key, secret_key, bucket 均不能为空")
	}

	// Cloudflare R2 S3 兼容 API 的标准 Endpoint 格式
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// 使用静态凭证和 auto 区域加载 AWS SDK 基础配置
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		awsConfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("加载 AWS SDK 配置失败: %w", err)
	}

	// 实例化 S3 Client 并覆盖 BaseEndpoint 指向 Cloudflare R2
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	presignClient := s3.NewPresignClient(client)

	return &R2Storage{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.BucketName,
		publicDomain:  strings.TrimRight(cfg.PublicDomain, "/"),
		accountID:     cfg.AccountID,
	}, nil
}

// Upload 流式上传文件至 Cloudflare R2 存储桶，并返回可公开访问的 URL
func (r *R2Storage) Upload(ctx context.Context, body io.Reader, objectKey string, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(objectKey),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("上传对象到 R2 失败: %w", err)
	}

	return r.GetPublicURL(objectKey), nil
}

// GetPresignedPutURL 生成带有过期时间的预签名 PUT URL，供前端或第三方客户端直接把文件上传至 R2
func (r *R2Storage) GetPresignedPutURL(ctx context.Context, objectKey string, contentType string, expiry time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}

	req, err := r.presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("生成预签名上传 URL 失败: %w", err)
	}

	return req.URL, nil
}

// GetPublicURL 根据 objectKey 拼装公网访问 URL
func (r *R2Storage) GetPublicURL(objectKey string) string {
	cleanKey := strings.TrimPrefix(objectKey, "/")
	if r.publicDomain != "" {
		return fmt.Sprintf("%s/%s", r.publicDomain, cleanKey)
	}
	// 若未绑定自定义域名或 r2.dev 访问域名，降级使用 S3 Endpoint 路径
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", r.accountID, r.bucket, cleanKey)
}
