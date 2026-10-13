package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"gorm.io/gorm"
)

// CreatePostInput 创建文章请求参数
type CreatePostInput struct {
	Title      string `json:"title" binding:"required,min=1,max=200"`
	Content    string `json:"content" binding:"required"`
	Slug       string `json:"slug"`
	Summary    string `json:"summary"`
	CoverImage string `json:"cover_image"`
	CategoryID *uint  `json:"category_id"`
	SeriesID   *uint  `json:"series_id"`
	OrderIndex int    `json:"order_index"`
}

// UpdatePostInput 更新文章请求参数
type UpdatePostInput struct {
	Title      *string `json:"title"`
	Content    *string `json:"content"`
	Slug       *string `json:"slug"`
	Summary    *string `json:"summary"`
	CoverImage *string `json:"cover_image"`
	CategoryID *uint   `json:"category_id"`
	SeriesID   *uint   `json:"series_id"`
	OrderIndex *int    `json:"order_index"`
	Published  *bool   `json:"published"`
}

// PostHandler 博客文章控制器
type PostHandler struct {
	cfg *config.Config
}

// NewPostHandler 创建文章处理器实例
func NewPostHandler(cfg *config.Config) *PostHandler {
	return &PostHandler{cfg: cfg}
}

// Create 发布新文章 (需 JWT 登录)
func (h *PostHandler) Create(c *gin.Context) {
	var input CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请求参数校验失败", err.Error())
		return
	}

	// 1. 从中间件上下文获取当前登录用户 ID
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权操作")
		return
	}
	userID := userIDVal.(uint)

	// 2. 校验分类是否存在 (若传入)
	if input.CategoryID != nil && *input.CategoryID > 0 {
		var cat models.Category
		if err := models.DB.First(&cat, *input.CategoryID).Error; err != nil {
			response.BadRequest(c, "指定的主题分类不存在")
			return
		}
	}

	// 3. 校验专栏是否存在 (若传入)
	if input.SeriesID != nil && *input.SeriesID > 0 {
		var series models.Series
		if err := models.DB.First(&series, *input.SeriesID).Error; err != nil {
			response.BadRequest(c, "指定的系列专栏不存在")
			return
		}
	}

	// 4. 构建 Post 实体 (未提供 slug 时将由 BeforeCreate 钩子自动生成)
	post := models.Post{
		Title:      input.Title,
		Content:    input.Content,
		Slug:       input.Slug,
		Summary:    input.Summary,
		CoverImage: input.CoverImage,
		UserID:     userID,
		CategoryID: input.CategoryID,
		SeriesID:   input.SeriesID,
		OrderIndex: input.OrderIndex,
		Published:  true,
	}

	if err := models.DB.Create(&post).Error; err != nil {
		response.InternalServerError(c, "保存文章失败", err.Error())
		return
	}

	// 预加载作者信息返回
	models.DB.Preload("User").Preload("Category").Preload("Series").First(&post, post.ID)
	response.Created(c, post)
}

// List 获取文章列表 (公开接口，支持分页与分类/专栏过滤)
func (h *PostHandler) List(c *gin.Context) {
	// 1. 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 {
		pageSize = 10
	}
	// 防御性硬限制：单页上限 50，防止恶意大查询打垮内存
	if pageSize > 50 {
		pageSize = 50
	}

	query := models.DB.Model(&models.Post{})

	// 2. 过滤参数
	if catIDStr := c.Query("category_id"); catIDStr != "" {
		if catID, err := strconv.Atoi(catIDStr); err == nil {
			query = query.Where("category_id = ?", catID)
		}
	}
	if seriesIDStr := c.Query("series_id"); seriesIDStr != "" {
		if seriesID, err := strconv.Atoi(seriesIDStr); err == nil {
			query = query.Where("series_id = ?", seriesID)
		}
	}

	// 3. 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.InternalServerError(c, "统计文章数量失败")
		return
	}

	// 4. 分页加载文章数据与关联实体
	var posts []models.Post
	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Preload("Category").
		Preload("Series").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		response.InternalServerError(c, "查询文章列表失败")
		return
	}

	response.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     posts,
	})
}

// GetByID 获取单篇文章详情 (公开接口)
func (h *PostHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	var post models.Post
	err := models.DB.Preload("User").
		Preload("Category").
		Preload("Series").
		First(&post, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文章不存在")
			return
		}
		response.InternalServerError(c, "获取文章详情失败")
		return
	}

	response.Success(c, post)
}

// Update 更新文章内容 (仅允许文章作者本人修改)
func (h *PostHandler) Update(c *gin.Context) {
	id := c.Param("id")

	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权操作")
		return
	}
	currentUserID := userIDVal.(uint)

	// 1. 查找文章是否存在 (404 语义判定)
	var post models.Post
	if err := models.DB.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文章不存在")
			return
		}
		response.InternalServerError(c, "查找文章失败")
		return
	}

	// 2. 检查作者权限 (403 权限判定：非本人严禁修改)
	if post.UserID != currentUserID {
		response.Forbidden(c, "无权修改他人的文章")
		return
	}

	// 3. 校验入参
	var input UpdatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "更新参数校验失败", err.Error())
		return
	}

	// 4. 选择性更新字段
	updates := make(map[string]interface{})
	if input.Title != nil && strings.TrimSpace(*input.Title) != "" {
		updates["title"] = *input.Title
	}
	if input.Content != nil {
		updates["content"] = *input.Content
	}
	if input.Slug != nil && strings.TrimSpace(*input.Slug) != "" {
		updates["slug"] = *input.Slug
	}
	if input.Summary != nil {
		updates["summary"] = *input.Summary
	}
	if input.CoverImage != nil {
		updates["cover_image"] = *input.CoverImage
	}
	if input.CategoryID != nil {
		updates["category_id"] = input.CategoryID
	}
	if input.SeriesID != nil {
		updates["series_id"] = input.SeriesID
	}
	if input.OrderIndex != nil {
		updates["order_index"] = *input.OrderIndex
	}
	if input.Published != nil {
		updates["published"] = *input.Published
	}

	if len(updates) > 0 {
		if err := models.DB.Model(&post).Updates(updates).Error; err != nil {
			response.InternalServerError(c, "更新文章失败", err.Error())
			return
		}
	}

	// 重新读取更新后的完整对象返回
	models.DB.Preload("User").Preload("Category").Preload("Series").First(&post, post.ID)
	response.Success(c, post)
}

// Delete 删除文章 (仅允许文章作者本人删除，自动级联清理附属评论)
func (h *PostHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权操作")
		return
	}
	currentUserID := userIDVal.(uint)

	// 1. 查找文章是否存在
	var post models.Post
	if err := models.DB.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文章不存在")
			return
		}
		response.InternalServerError(c, "查找文章失败")
		return
	}

	// 2. 检查作者权限 (非作者返回 403)
	if post.UserID != currentUserID {
		response.Forbidden(c, "无权删除他人的文章")
		return
	}

	// 3. 执行物理删除 (外键级联会自动删除所有关联 comments)
	if err := models.DB.Delete(&post).Error; err != nil {
		response.InternalServerError(c, "删除文章失败", err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "文章已成功删除",
		"id":      post.ID,
	})
}
