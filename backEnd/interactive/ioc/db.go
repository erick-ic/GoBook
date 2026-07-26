package ioc

import (
	"GoBook/interactive/repository/dao"
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 根据 db.dsn 创建互动服务数据库连接，并确保互动相关表存在。
func InitDB() *gorm.DB {
	type Config struct {
		DSN string `mapstructure:"dsn"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("db", &cfg); err != nil {
		panic(err)
	}
	if cfg.DSN == "" {
		panic("interactive: db.dsn 不能为空")
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("interactive: 连接数据库失败: %w", err))
	}

	if err = dao.InitTable(db); err != nil {
		panic(fmt.Errorf("interactive: 初始化数据表失败: %w", err))
	}
	return db
}
