package routes

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/handlers"
	"github.com/tonsiasy/cyber-blog/middleware"
	"github.com/tonsiasy/cyber-blog/pkg/storage"
)

// SetupRouter 统一组织与注册系统 API 路由
func SetupRouter(r *gin.Engine, cfg *config.Config) {
	// 初始化存储服务 (支持 R2 云端与本地磁盘降级)
	storageService, err := storage.NewStorageService(context.Background(), cfg)
	if err != nil {
		log.Printf("[Routes] 初始化存储服务警告: %v，将降级使用本地存储", err)
		storageService = storage.NewLocalStorage("./uploads", "/uploads")
	}

	// 开放本地上传文件的静态访问
	r.Static("/uploads", "./uploads")

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
	commentHandler := handlers.NewCommentHandler(cfg)

	// 公开读取路由 (文章列表、详情、文章评论)
	publicPosts := apiV1.Group("/posts")
	{
		publicPosts.GET("", postHandler.List)
		publicPosts.GET("/:id", postHandler.GetByID)
		publicPosts.GET("/:id/comments", commentHandler.ListByPost)
	}

	// 保护写入路由 (需登录：发文、改删文、发评论)
	protectedPosts := apiV1.Group("/posts", middleware.AuthRequired(cfg))
	{
		protectedPosts.POST("", postHandler.Create)
		protectedPosts.PUT("/:id", postHandler.Update)
		protectedPosts.DELETE("/:id", postHandler.Delete)
		protectedPosts.POST("/:id/comments", commentHandler.Create)
	}

	// 3. 主题分类路由组 (/api/v1/categories/*)
	taxonomyHandler := handlers.NewTaxonomyHandler(cfg)
	categoriesGroup := apiV1.Group("/categories")
	{
		categoriesGroup.GET("", taxonomyHandler.ListCategories)
		categoriesGroup.GET("/:slug/posts", taxonomyHandler.GetPostsByCategorySlug)
		categoriesGroup.POST("", middleware.AuthRequired(cfg), taxonomyHandler.CreateCategory)
	}

	// 4. 系列专栏路由组 (/api/v1/series/*)
	seriesGroup := apiV1.Group("/series")
	{
		seriesGroup.GET("", taxonomyHandler.ListSeries)
		seriesGroup.GET("/:slug", taxonomyHandler.GetSeriesOutline)
		seriesGroup.POST("", middleware.AuthRequired(cfg), taxonomyHandler.CreateSeries)
	}

	// 5. 多媒体存储路由组 (/api/v1/media/*)
	mediaHandler := handlers.NewMediaHandler(cfg, storageService)
	mediaGroup := apiV1.Group("/media", middleware.AuthRequired(cfg))
	{
		mediaGroup.POST("/upload", mediaHandler.Upload)
		mediaGroup.POST("/presign", mediaHandler.Presign)
	}

	// 6. 课程大作业路由向前兼容 (/api/...)
	// 确保完全符合作业文档指定的无版本前缀路径
	legacyGroup := r.Group("/api")
	{
		// 认证
		legacyGroup.POST("/register", authHandler.Register)
		legacyGroup.POST("/login", authHandler.Login)

		// 公开文章与评论检索
		legacyGroup.GET("/posts", postHandler.List)
		legacyGroup.GET("/posts/:id", postHandler.GetByID)
		legacyGroup.GET("/posts/:id/comments", commentHandler.ListByPost)

		// 需认证文章与评论操作
		legacyProtected := legacyGroup.Group("", middleware.AuthRequired(cfg))
		{
			legacyProtected.POST("/posts", postHandler.Create)
			legacyProtected.PUT("/posts/:id", postHandler.Update)
			legacyProtected.DELETE("/posts/:id", postHandler.Delete)
			legacyProtected.POST("/posts/:id/comments", commentHandler.Create)
			legacyProtected.POST("/media/upload", mediaHandler.Upload)
			legacyProtected.POST("/media/presign", mediaHandler.Presign)
		}
	}
}
