package handlers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"gorm.io/gorm"
)

// CreateCommentInput 创建评论入参
type CreateCommentInput struct {
	Content string `json:"content" binding:"required,min=1,max=1000"`
}

// CommentHandler 评论控制器
type CommentHandler struct {
	cfg *config.Config
}

// NewCommentHandler 创建评论处理器实例
func NewCommentHandler(cfg *config.Config) *CommentHandler {
	return &CommentHandler{cfg: cfg}
}

// Create 发表评论 (需 JWT 登录)
func (h *CommentHandler) Create(c *gin.Context) {
	// 1. 获取文章 ID 参数
	postIDStr := c.Param("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "非法的文章 ID")
		return
	}

	// 2. 检查文章是否存在 (404 判定)
	var post models.Post
	if err := models.DB.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文章不存在，无法发表评论")
			return
		}
		response.InternalServerError(c, "查询文章失败")
		return
	}

	// 3. 从 JWT 认证上下文获取当前登录用户 ID
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权操作")
		return
	}
	userID := userIDVal.(uint)

	// 4. 参数校验
	var input CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "评论内容校验失败", err.Error())
		return
	}

	// 5. 持久化存储评论
	comment := models.Comment{
		Content: input.Content,
		UserID:  userID,
		PostID:  uint(postID),
	}

	if err := models.DB.Create(&comment).Error; err != nil {
		response.InternalServerError(c, "保存评论失败", err.Error())
		return
	}

	// 预加载评论者 User 信息返回
	models.DB.Preload("User").First(&comment, comment.ID)
	response.Created(c, comment)
}

// ListByPost 获取某篇文章的所有评论列表 (公开接口，按时间正序排列)
func (h *CommentHandler) ListByPost(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "非法的文章 ID")
		return
	}

	// 1. 检查文章是否存在
	var post models.Post
	if err := models.DB.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文章不存在")
			return
		}
		response.InternalServerError(c, "查询文章失败")
		return
	}

	// 2. 查询该文章的所有评论，按时间正序排序 (利用複合索引 idx_post_comment_created)
	var comments []models.Comment
	err = models.DB.Where("post_id = ?", postID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments).Error

	if err != nil {
		response.InternalServerError(c, "查询评论列表失败")
		return
	}

	response.Success(c, gin.H{
		"post_id": postID,
		"total":   len(comments),
		"items":   comments,
	})
}
