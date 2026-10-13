package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
)

// AuthRequired JWT 身份认证中间件
// 验证请求头中的 `Authorization: Bearer <token>`，成功则提取用户 Claims 注入上下文
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少 Authorization 认证请求头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Authorization 请求头格式必须为 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := utils.ParseToken(tokenString, cfg.JWT.Secret)
		if err != nil {
			response.Unauthorized(c, "Token 无效或已过期")
			c.Abort()
			return
		}

		// 将解析后的用户身份标识存入 Context，供后续 Handler 获取
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
