package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Post 博客文章模型
type Post struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Title      string    `gorm:"size:200;not null" json:"title"`
	Slug       string    `gorm:"size:200;not null;uniqueIndex" json:"slug"`
	Summary    string    `gorm:"size:500" json:"summary"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CoverImage string    `gorm:"size:255" json:"cover_image"`

	// 作者关联 (文章必须有作者，作者被删除时级联删除其文章)
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"author,omitempty"`

	// 主题分类关联 (可选，分类被删除时文章保留，category_id 置空)
	CategoryID *uint     `gorm:"index" json:"category_id,omitempty"`
	Category   *Category `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category,omitempty"`

	// 系列专栏关联 (可选，专栏被删除时文章保留，series_id 置空)
	SeriesID *uint   `gorm:"index" json:"series_id,omitempty"`
	Series   *Series `gorm:"foreignKey:SeriesID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"series,omitempty"`

	// 复合索引：加速专栏目录的章节顺序查询
	OrderIndex int  `gorm:"default:0;index:idx_series_order,priority:2" json:"order_index"`
	Published  bool `gorm:"default:true;index" json:"published"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM 钩子：在记录创建前自动处理 Slug 兜底
func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(p.Slug) == "" {
		p.Slug = generateSlug(p.Title)
	}
	return nil
}

// generateSlug 基于标题和随机哈希生成 URL 友好且唯一的 slug
func generateSlug(title string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9\p{Han}]+`)
	clean := reg.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-")
	clean = strings.Trim(clean, "-")

	b := make([]byte, 4)
	_, _ = rand.Read(b)
	randSuffix := hex.EncodeToString(b)

	if clean == "" {
		return fmt.Sprintf("post-%d-%s", time.Now().Unix(), randSuffix)
	}
	return fmt.Sprintf("%s-%s", clean, randSuffix)
}
