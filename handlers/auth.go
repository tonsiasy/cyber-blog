package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tonsiasy/cyber-blog/config"
	"github.com/tonsiasy/cyber-blog/models"
	"github.com/tonsiasy/cyber-blog/pkg/response"
	"github.com/tonsiasy/cyber-blog/pkg/utils"
	"gorm.io/gorm"
)

// RegisterInput 注册请求入参
type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	Email    string `json:"email" binding:"required,email"`
}

// LoginInput 登录请求入参
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserProfile 用户对外公开信息
type UserProfile struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Bio       string `json:"bio"`
	CreatedAt string `json:"created_at"`
}

// AuthHandler 认证控制器
type AuthHandler struct {
	cfg *config.Config
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// Register 用户注册接口
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请求参数校验失败", err.Error())
		return
	}

	// 1. 检查用户名是否已存在
	var count int64
	models.DB.Model(&models.User{}).Where("username = ?", input.Username).Count(&count)
	if count > 0 {
		response.Fail(c, http.StatusConflict, "该用户名已被注册")
		return
	}

	// 2. 检查邮箱是否已存在
	models.DB.Model(&models.User{}).Where("email = ?", input.Email).Count(&count)
	if count > 0 {
		response.Fail(c, http.StatusConflict, "该邮箱已被注册")
		return
	}

	// 3. 对明文密码执行 bcrypt 哈希
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		response.InternalServerError(c, "密码哈希失败")
		return
	}

	// 4. 持久化存储
	user := models.User{
		Username: input.Username,
		Password: hashedPassword,
		Email:    input.Email,
	}
	if err := models.DB.Create(&user).Error; err != nil {
		response.InternalServerError(c, "创建用户失败", err.Error())
		return
	}

	response.Created(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// Login 用户登录并签发 JWT
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请求参数校验失败", err.Error())
		return
	}

	// 1. 根据用户名检索用户
	var user models.User
	err := models.DB.Where("username = ?", input.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "用户名或密码错误")
			return
		}
		response.InternalServerError(c, "查询用户失败")
		return
	}

	// 2. 校验密码
	if !utils.CheckPassword(user.Password, input.Password) {
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	// 3. 签发 JWT Token
	token, err := utils.GenerateToken(
		user.ID,
		user.Username,
		h.cfg.JWT.Secret,
		h.cfg.JWT.ExpirationHours,
	)
	if err != nil {
		response.InternalServerError(c, "生成 Token 失败")
		return
	}

	response.Success(c, gin.H{
		"token": token,
		"user": UserProfile{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Bio:       user.Bio,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// Me 获取当前已认证用户的基本信息 (受 JWT 中间件保护)
func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "未授权访问")
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c, "无效的用户上下文")
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Bio:       user.Bio,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}
