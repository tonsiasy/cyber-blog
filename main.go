package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/routes"
)

func main() {
	// 1. 加载应用配置
	cfg := config.LoadConfig()

	// 2. 初始化数据库与自动迁移
	_, err := models.InitDB(cfg)
	if err != nil {
		log.Fatalf("[Main] 数据库初始化失败: %v", err)
	}

	// 3. 设置 Gin 运行模式 (debug / release)
	gin.SetMode(cfg.Server.GinMode)

	// 4. 创建 Gin 引擎实例 (默认集成 Logger 与 Recovery 中间件)
	r := gin.Default()

	// 5. 注册服务健康探测与心跳路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "pong",
			"app":     "cyber-blog",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 6. 注册业务 API 路由
	routes.SetupRouter(r, cfg)

	// 6. 监听端口并启动 HTTP 服务
	addr := ":" + cfg.Server.Port
	log.Printf("[Main] Cyber-Blog 后端服务启动，正在监听端口 %s ...", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[Main] 服务启动失败: %v", err)
	}
}
