package models

import "time"

// Series 系列专栏模型
type Series struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:100;not null" json:"title"`
	Slug        string    `gorm:"size:100;not null;uniqueIndex" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	CoverImage  string    `gorm:"size:255" json:"cover_image"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"author,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
