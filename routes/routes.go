package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/handlers"
	"github.com/tonsiasy/cyber-blog/middleware"
)

// SetupRouter 统一组织与注册系统 API 路由
func SetupRouter(r *gin.Engine, cfg *config.Config) {
	apiV1 := r.Group("/api/v1")

	// 1. 用户认证路由组 (/api/v1/auth/*)
	authHandler := handlers.NewAuthHandler(cfg)
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.GET("/me", middleware.AuthRequired(cfg), authHandler.Me)
	}

	// 2. 课程大作业路由向前兼容 (/api/register, /api/login)
	// 确保完全符合作业文档指定的直接路径
	legacyGroup := r.Group("/api")
	{
		legacyGroup.POST("/register", authHandler.Register)
		legacyGroup.POST("/login", authHandler.Login)
	}
}
