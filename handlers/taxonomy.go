package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"gorm.io/gorm"
)

// CreateCategoryInput 创建分类入参
type CreateCategoryInput struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// CreateSeriesInput 创建专栏入参
type CreateSeriesInput struct {
	Title       string `json:"title" binding:"required,min=2,max=100"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CoverImage  string `json:"cover_image"`
}

// CategoryWithCount 带文章计数的分类视图
type CategoryWithCount struct {
	models.Category
	PostCount int64 `json:"post_count"`
}

// SeriesWithCount 带章节计数的专栏视图
type SeriesWithCount struct {
	models.Series
	ChapterCount int64 `json:"chapter_count"`
}

// TaxonomyHandler 主题分类与系列专栏控制器
type TaxonomyHandler struct {
	cfg *config.Config
}

// NewTaxonomyHandler 创建主题与专栏处理器实例
func NewTaxonomyHandler(cfg *config.Config) *TaxonomyHandler {
	return &TaxonomyHandler{cfg: cfg}
}

// ---------------------- Category 分类接口 ----------------------

// ListCategories 获取全部分类及文章聚合计数 (公开)
func (h *TaxonomyHandler) ListCategories(c *gin.Context) {
	var categories []models.Category
	if err := models.DB.Order("created_at DESC").Find(&categories).Error; err != nil {
		response.InternalServerError(c, "获取分类列表失败")
		return
	}

	result := make([]CategoryWithCount, len(categories))
	for i, cat := range categories {
		var count int64
		models.DB.Model(&models.Post{}).Where("category_id = ?", cat.ID).Count(&count)
		result[i] = CategoryWithCount{
			Category:  cat,
			PostCount: count,
		}
	}

	response.Success(c, result)
}

// CreateCategory 创建新主题分类 (需认证)
func (h *TaxonomyHandler) CreateCategory(c *gin.Context) {
	var input CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "分类参数校验失败", err.Error())
		return
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Name)
	}

	// 唯一性冲突校验
	var count int64
	models.DB.Model(&models.Category{}).
		Where("name = ? OR slug = ?", input.Name, slug).
		Count(&count)
	if count > 0 {
		response.Fail(c, http.StatusConflict, "该分类名称或别名已存在")
		return
	}

	category := models.Category{
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
	}

	if err := models.DB.Create(&category).Error; err != nil {
		response.InternalServerError(c, "创建分类失败")
		return
	}

	response.Created(c, category)
}

// GetPostsByCategorySlug 按分类别名检索归属文章列表 (公开)
func (h *TaxonomyHandler) GetPostsByCategorySlug(c *gin.Context) {
	slug := c.Param("slug")

	var category models.Category
	if err := models.DB.Where("slug = ?", slug).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "指定主题分类不存在")
			return
		}
		response.InternalServerError(c, "查询分类失败")
		return
	}

	var posts []models.Post
	err := models.DB.Where("category_id = ?", category.ID).
		Preload("User").
		Preload("Series").
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		response.InternalServerError(c, "查询分类文章失败")
		return
	}

	response.Success(c, gin.H{
		"category": category,
		"total":    len(posts),
		"posts":    posts,
	})
}

// ---------------------- Series 系列专栏接口 ----------------------

// ListSeries 获取所有系列专栏大纲 (公开)
func (h *TaxonomyHandler) ListSeries(c *gin.Context) {
	var seriesList []models.Series
	if err := models.DB.Preload("User").Order("created_at DESC").Find(&seriesList).Error; err != nil {
		response.InternalServerError(c, "获取专栏列表失败")
		return
	}

	result := make([]SeriesWithCount, len(seriesList))
	for i, s := range seriesList {
		var count int64
		models.DB.Model(&models.Post{}).Where("series_id = ?", s.ID).Count(&count)
		result[i] = SeriesWithCount{
			Series:       s,
			ChapterCount: count,
		}
	}

	response.Success(c, result)
}

// CreateSeries 创建系列专栏 (需认证)
func (h *TaxonomyHandler) CreateSeries(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权操作")
		return
	}
	userID := userIDVal.(uint)

	var input CreateSeriesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "专栏参数校验失败", err.Error())
		return
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(input.Title)
	}

	var count int64
	models.DB.Model(&models.Series{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		response.Fail(c, http.StatusConflict, "该专栏别名已存在")
		return
	}

	series := models.Series{
		Title:       input.Title,
		Slug:        slug,
		Description: input.Description,
		CoverImage:  input.CoverImage,
		UserID:      userID,
	}

	if err := models.DB.Create(&series).Error; err != nil {
		response.InternalServerError(c, "创建专栏失败")
		return
	}

	models.DB.Preload("User").First(&series, series.ID)
	response.Created(c, series)
}

// GetSeriesOutline 获取专栏大纲与章节顺序列表 (公开)
// 满足 spec: chapters 必须严格按 order_index ASC, created_at ASC 排序
func (h *TaxonomyHandler) GetSeriesOutline(c *gin.Context) {
	slug := c.Param("slug")

	var series models.Series
	if err := models.DB.Preload("User").Where("slug = ?", slug).First(&series).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "指定专栏不存在")
			return
		}
		response.InternalServerError(c, "查询专栏失败")
		return
	}

	// 复合索引 idx_series_order 加速按章节次序查询
	var chapters []models.Post
	err := models.DB.Where("series_id = ?", series.ID).
		Preload("User").
		Order("order_index ASC, created_at ASC").
		Find(&chapters).Error

	if err != nil {
		response.InternalServerError(c, "查询专栏章节失败")
		return
	}

	response.Success(c, gin.H{
		"series":        series,
		"chapter_count": len(chapters),
		"chapters":      chapters,
	})
}

// slugify 简单生成 URL 友好的 slug 辅助函数
func slugify(input string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9\p{Han}]+`)
	clean := reg.ReplaceAllString(strings.ToLower(strings.TrimSpace(input)), "-")
	return strings.Trim(clean, "-")
}
