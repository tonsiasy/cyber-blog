package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 聚合整个应用的所有配置项
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	R2       R2Config
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port    string
	GinMode string
}

// DatabaseConfig 数据库连接配置
type DatabaseConfig struct {
	Driver     string // "sqlite" 或 "postgres"
	SQLitePath string // 当 Driver 为 sqlite 时的本地数据库文件路径
	Host       string
	Port       string
	User       string
	Password   string
	DBName     string
	SSLMode    string
	TimeZone   string
}

// JWTConfig 用户认证令牌配置
type JWTConfig struct {
	Secret          string
	ExpirationHours int
}

// R2Config Cloudflare R2 对象存储配置 (兼容 S3 协议)
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicDomain    string
}

// LoadConfig 从 .env 文件和系统环境变量中加载配置
func LoadConfig() *Config {
	// 尝试加载本地 .env 文件
	// 注意：在 Docker 容器或生产环境中，配置通常直接通过系统环境变量注入，
	// 因此即便找不到 .env 文件也不应 panic，记录日志即可。
	if err := godotenv.Load(); err != nil {
		log.Println("[Config] 未找到 .env 文件或读取失败，将直接读取系统环境变量")
	}

	expHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "72"))
	if err != nil {
		expHours = 72
	}

	return &Config{
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Driver:     getEnv("DB_DRIVER", "sqlite"),
			SQLitePath: getEnv("DB_SQLITE_PATH", "cyber_blog.db"),
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			User:       getEnv("DB_USER", "blog_user"),
			Password:   getEnv("DB_PASSWORD", "your_secure_password"),
			DBName:     getEnv("DB_NAME", "cyber_blog"),
			SSLMode:    getEnv("DB_SSLMODE", "disable"),
			TimeZone:   getEnv("DB_TIMEZONE", "Asia/Shanghai"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "cyber-blog-default-secret-key-2026"),
			ExpirationHours: expHours,
		},
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", ""),
			PublicDomain:    getEnv("R2_PUBLIC_DOMAIN", ""),
		},
	}
}

// getEnv 读取指定环境变量，若为空则返回提供的默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
