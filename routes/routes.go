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

	// 2. 文章路由组 (/api/v1/posts/*)
	postHandler := handlers.NewPostHandler(cfg)
	
	// 公开读取路由
	publicPosts := apiV1.Group("/posts")
	{
		publicPosts.GET("", postHandler.List)
		publicPosts.GET("/:id", postHandler.GetByID)
	}

	// 保护写入路由 (需登录且校验作者)
	protectedPosts := apiV1.Group("/posts", middleware.AuthRequired(cfg))
	{
		protectedPosts.POST("", postHandler.Create)
		protectedPosts.PUT("/:id", postHandler.Update)
		protectedPosts.DELETE("/:id", postHandler.Delete)
	}

	// 3. 课程大作业路由向前兼容 (/api/...)
	// 确保完全符合作业文档指定的无版本前缀路径
	legacyGroup := r.Group("/api")
	{
		// 认证
		legacyGroup.POST("/register", authHandler.Register)
		legacyGroup.POST("/login", authHandler.Login)

		// 公开文章检索
		legacyGroup.GET("/posts", postHandler.List)
		legacyGroup.GET("/posts/:id", postHandler.GetByID)

		// 需认证文章操作
		legacyProtected := legacyGroup.Group("", middleware.AuthRequired(cfg))
		{
			legacyProtected.POST("/posts", postHandler.Create)
			legacyProtected.PUT("/posts/:id", postHandler.Update)
			legacyProtected.DELETE("/posts/:id", postHandler.Delete)
		}
	}
}
