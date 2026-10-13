package models

import (
	"fmt"
	"log"
	"time"

	"github.com/tonsiasy/cyber-blog/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库单例对象
var DB *gorm.DB

// InitDB 初始化数据库连接、连接池与数据表迁移
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Driver {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
			cfg.Database.Host,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
			cfg.Database.Port,
			cfg.Database.SSLMode,
			cfg.Database.TimeZone,
		)
		log.Printf("[Database] 正在连接 PostgreSQL (%s:%s/%s)...", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		dialector = postgres.Open(dsn)

	case "sqlite":
		fallthrough
	default:
		log.Printf("[Database] 正在连接 SQLite (%s)...", cfg.Database.SQLitePath)
		dialector = sqlite.Open(cfg.Database.SQLitePath)
	}

	// GORM 配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 若为 SQLite，显式开启外键约束开关 (SQLite 默认不强制检查外键)
	if cfg.Database.Driver == "sqlite" || cfg.Database.Driver == "" {
		if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
			log.Printf("[Database] 警告: 启用 SQLite 外键检查失败: %v", err)
		}
	}

	// 底层连接池调优
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)                  // 空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最长存活周期

	// 自动迁移所有核心数据模型
	log.Println("[Database] 正在执行数据模型自动迁移 (AutoMigrate)...")
	err = db.AutoMigrate(
		&User{},
		&Category{},
		&Series{},
		&Post{},
		&Comment{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据模型自动迁移失败: %w", err)
	}

	log.Println("[Database] 数据模型迁移完成，表与索引已就绪！")
	DB = db
	return db, nil
}
