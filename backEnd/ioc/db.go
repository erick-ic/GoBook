package ioc

import (
	"GoBook/internal/repository/dao"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	//数据库连接
	//dsn := "root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local"
	//dsn := "root:root@tcp(gobook-mysql:11309)/gobook?charset=utf8mb4&parseTime=True&loc=Local"

	//方式1:
	//dsn := config.Config.DB.DSN

	//方式2:
	//dsn := viper.GetString("db.dsn")

	//方式3:
	//初始化时，读取配置
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	//var cfg Config = Config{
	//	//DSN: "root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local",
	//}
	var cfg Config

	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN))
	if err != nil {
		//一旦初始化过程报错，应用就取消启动
		//panic相当于整个goroutine结束
		panic(err)
	}

	//自动初始化表
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
