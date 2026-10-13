package models

import "time"

// Category 主题分组模型
type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Slug        string    `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
