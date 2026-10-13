package models

import "time"

// User 用户实体模型
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"` // 密码密文在序列化为 JSON 时绝不输出
	Email     string    `gorm:"size:100;not null;uniqueIndex" json:"email"`
	Avatar    string    `gorm:"size:255" json:"avatar"` // 头像地址 (Cloudflare R2)
	Bio       string    `gorm:"size:255" json:"bio"`    // 个人简介
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
