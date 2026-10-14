package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"github.com/tonsiasy/cyber-blog/pkg/storage"
)

// allowedMimeTypes 限制允许上传的多媒体 MIME 类型
var allowedMimeTypes = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

// maxUploadSize 最大上传限制为 10MB
const maxUploadSize = 10 * 1024 * 1024

// MediaHandler 处理图片与多媒体资源的直传与预签名分发
type MediaHandler struct {
	cfg     *config.Config
	storage storage.StorageService
}

// NewMediaHandler 实例化 MediaHandler
func NewMediaHandler(cfg *config.Config, s storage.StorageService) *MediaHandler {
	return &MediaHandler{
		cfg:     cfg,
		storage: s,
	}
}

// Upload 处理服务端转发上传 (POST /api/v1/media/upload)
func (h *MediaHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传文件 (form-data 字段名必须为: file)")
		return
	}

	if fileHeader.Size > maxUploadSize {
		response.BadRequest(c, "文件大小不能超过 10MB")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "读取上传文件失败")
		return
	}
	defer file.Close()

	// 读取前 512 字节探测文件真实 MIME 类型
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	// 将文件读取游标复位到起点，以供后续完整写入
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	detectedType := http.DetectContentType(buffer[:n])
	headerType := fileHeader.Header.Get("Content-Type")

	// 优先以 Header 传入的有效类型为准，若为二进制通用类型则使用探针类型
	contentType := headerType
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectedType
	}

	// 针对 SVG 进行特殊适配 (DetectContentType 往往将 svg 探测为 text/xml)
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".svg" {
		contentType = "image/svg+xml"
	}

	// 校验是否在允许的图片类型范围内
	defaultExt, allowed := allowedMimeTypes[contentType]
	if !allowed {
		response.BadRequest(c, fmt.Sprintf("不支持的文件类型: %s，仅支持 jpg/png/gif/webp/svg 图片格式", contentType))
		return
	}

	if ext == "" {
		ext = defaultExt
	}

	objectKey := generateObjectKey("uploads", ext)

	// 上传到对象存储
	publicURL, err := h.storage.Upload(c.Request.Context(), file, objectKey, contentType)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("文件存储失败: %v", err))
		return
	}

	response.Success(c, gin.H{
		"url":          publicURL,
		"key":          objectKey,
		"filename":     fileHeader.Filename,
		"size":         fileHeader.Size,
		"content_type": contentType,
	})
}

// PresignRequest 预签名申请请求体
type PresignRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
}

// Presign 处理预签名 PUT URL 申请 (POST /api/v1/media/presign)
func (h *MediaHandler) Presign(c *gin.Context) {
	var req PresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败: filename 和 content_type 均为必填项")
		return
	}

	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	defaultExt, allowed := allowedMimeTypes[contentType]
	if !allowed {
		response.BadRequest(c, fmt.Sprintf("不支持的文件类型: %s，仅支持 jpg/png/gif/webp/svg 图片格式", contentType))
		return
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext == "" {
		ext = defaultExt
	}

	objectKey := generateObjectKey("uploads", ext)
	expiry := 15 * time.Minute

	uploadURL, err := h.storage.GetPresignedPutURL(c.Request.Context(), objectKey, contentType, expiry)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("生成预签名 URL 失败: %v", err))
		return
	}

	publicURL := h.storage.GetPublicURL(objectKey)

	response.Success(c, gin.H{
		"upload_url":   uploadURL,
		"public_url":   publicURL,
		"key":          objectKey,
		"content_type": contentType,
		"expires_in":   int(expiry.Seconds()),
	})
}

// generateObjectKey 生成日期分片且防碰撞的对象存储路径 key
func generateObjectKey(prefix, ext string) string {
	now := time.Now()
	randomBytes := make([]byte, 16)
	_, _ = rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	if !strings.HasPrefix(ext, ".") && ext != "" {
		ext = "." + ext
	}

	return fmt.Sprintf("%s/%04d/%02d/%s%s", strings.Trim(prefix, "/"), now.Year(), now.Month(), randomHex, ext)
}
