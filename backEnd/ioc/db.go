package ioc

import (
	"GoBook/internal/repository/dao"
	"GoBook/pkg/logger"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
	"gorm.io/plugin/prometheus"
)

func InitDB(l logger.LoggerV1) *gorm.DB {
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

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: gormLogger.New(
			//自定义 Writer
			gormLoggerFunc(l.Debug),
			gormLogger.Config{
				//慢查询阈值，超过阈值，记录
				SlowThreshold: time.Millisecond * 10,
				//忽略“记录未找到”错误
				IgnoreRecordNotFoundError: true,
				//日志级别
				LogLevel: gormLogger.Info,
			}),
	})
	if err != nil {
		//一旦初始化过程报错，应用就取消启动
		//panic相当于整个goroutine结束
		panic(err)
	}

	//GORM统计接口情况
	err = db.Use(prometheus.New(prometheus.Config{
		DBName:          "goBook",
		RefreshInterval: 15,
		StartServer:     false,
		MetricsCollector: []prometheus.MetricsCollector{
			&prometheus.MySQL{
				VariableNames: []string{"thread_id"},
			},
		},
	}))
	if err != nil {
		panic(err)
	}

	//接入opentelemetry
	err = db.Use(tracing.NewPlugin(
		tracing.WithDBSystem("gobook"),
		tracing.WithQueryFormatter(func(query string) string {
			l.Debug("", logger.String("query", query))
			return query
		}),
	))
	if err != nil {
		panic(err)
	}

	//自动初始化表
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(msg string, args ...interface{}) {
	g(msg, logger.Field{Key: "args", Value: args})
}
