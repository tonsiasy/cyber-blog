package models

import "time"

// Comment 文章评论实体模型
type Comment struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Content string `gorm:"type:text;not null" json:"content"`

	// 评论者 (用户删除时其评论级联删除)
	UserID uint `gorm:"not null;index" json:"user_id"`
	User   User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`

	// 所属文章：复合索引 (post_id, created_at)，文章删除时其下评论全部级联删除
	PostID uint `gorm:"not null;index:idx_post_comment_created,priority:1" json:"post_id"`
	Post   Post `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"` // 避免 JSON 序列化时的死循环引用

	CreatedAt time.Time `gorm:"index:idx_post_comment_created,priority:2" json:"created_at"`
}
