package main

import (
	"GoBook/config"
	"GoBook/internal/repository/dao"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	InitViperV1()

	// 启动独立的 Prometheus 指标服务，监听 8081 端口
	// 与业务端口 8080 隔离，避免：
	//  1. 业务中间件（限流/JWT）干扰 metrics 采集
	//  2. 业务流量过大时阻塞 /metrics 响应
	//  3. metrics 暴露运行时敏感信息，独立端口便于网络层隔离
	initPrometheus()

	app := InitApp()
	for _, c := range app.Consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}

	server := app.Server
	server.Run(":8080")
}

// initPrometheus 启动独立的 HTTP 服务暴露 Prometheus 指标
// 监听 127.0.0.1:8081，仅本机可访问
// 生产环境部署时，Prometheus 通过 K8s Service 内部访问该端口
func initPrometheus() {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		// 只绑定 localhost，外部网络访问不到，保证安全
		err := http.ListenAndServe("127.0.0.1:8081", mux)
		if err != nil {
			panic(fmt.Errorf("Prometheus 指标服务启动失败: %w", err))
		}
	}()
}

func initWebServer() *gin.Engine {
	server := gin.Default()

	////d. 限流方式，一秒钟100次
	//redisClient := redis.NewClient(&redis.Options{
	//	//Addr: "localhost:6379",
	//	//Addr: "gobook-redis:11479",
	//	Addr: config.Config.Redis.Addr,
	//})
	//server.Use(ratelimit.NewBuilder(redisClient, time.Second, 100).Build())

	//处理跨域
	server.Use(cors.New(cors.Config{
		//prelight接口请求中，origin:http://localhost:3000
		//AllowOrigins: []string{"http://localhost:3000"},

		//access-control-request-method：POST
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},

		//access-control-request-headers:authorization,content-type
		//大小写都行
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},

		//暴露给客户端
		ExposeHeaders: []string{"X-Total-Count", "X-JWT-Token"},

		//是否允许携带cookie
		AllowCredentials: true,

		//复杂请求配置
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				//开发环境
				return true
			}
			return strings.Contains(origin, "xxx.com")
		},

		MaxAge: 12 * time.Hour,
	}))

	//设置session
	//步骤1: 数据存放在store
	//a. 数据存在 客户端（浏览器） 的 Cookie 里。
	//store := cookie.NewStore([]byte("secret"))

	//b. 数据存在 当前应用服务器的本地内存（RAM） 里，当前 Go 应用进程的运行内存（RAM）。
	//store := memstore.NewStore([]byte("3akQBTZmfkuEjQacH5hvUynDnmPvAf7Y"),
	//	[]byte("Z4d8tz8WDKXT3AvYJkmhEb5VEFfxHHS2"))

	//c. redis
	//数据存在独立的 Redis 服务器
	/*
	   - 16       : 连接池中最大空闲连接数（Redis 默认支持 16 个数据库，此处指连接数）
	   - "tcp"    : 网络协议
	   - "localhost:6379" : Redis 服务地址
	   - ""       : 用户名（Redis 6.0+ 支持 ACL，空表示无需用户名）
	   - ""       : 密码（空表示无密码）
	   - []byte("3akQBTZmfkuEjQacH5hvUynDnmPvAf7Y") : 身份验证密钥（用于签名 Session ID，防篡改）
	   - []byte("Z4d8tz8WDKXT3AvYJkmhEb5VEFfxHHS2") : 数据加密密钥（用于加密会话数据，保证数据私密性）
	*/
	//store, _ := redis.NewStore(
	//	16,
	//	"tcp",
	//	"localhost:6379",
	//	"",
	//	"",
	//	[]byte("3akQBTZmfkuEjQacH5hvUynDnmPvAf7Y"),
	//	[]byte("Z4d8tz8WDKXT3AvYJkmhEb5VEFfxHHS2"))

	//server.Use(sessions.Sessions("mysession", store))

	//步骤3: 登录校验
	//server.Use(
	//	middleware.NewLoginMiddlewareBuilder().
	//		IgnoreLoginMiddlewareBuilder("/users/login").
	//		IgnoreLoginMiddlewareBuilder("/users/signup").
	//		Build(),
	//)

	//server.Use(middleware.NewLoginJWTMiddlewareBuilder().
	//	IgnorePaths("/users/login").
	//	IgnorePaths("/users/signup").
	//	IgnorePaths("/users/sendSMSCode").
	//	IgnorePaths("/users/loginSMS").
	//	Build())

	return server
}

//func initUser(db *gorm.DB, redisClient redis.Cmdable) *web.UserHandler {
//	ud := dao.NewUserDAO(db)
//
//	uc := cache.NewUserCache(redisClient)
//	repo := repository.NewUserRepository(ud, uc)
//	svc := service.NewUserService(repo)
//
//	memoSMS := memory.NewMemoService()
//	codeCache := cache.NewCodeCache(redisClient)
//	codeRepo := repository.NewCodeRepository(codeCache)
//	codeSvc := service.NewCodeService(codeRepo, memoSMS)
//
//	u := web.NewUserHandler(svc, codeSvc, redisClient)
//	return u
//}

func initDB() *gorm.DB {
	//数据库连接
	//dsn := "root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local"
	//dsn := "root:root@tcp(gobook-mysql:11309)/gobook?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := config.Config.DB.DSN
	db, err := gorm.Open(mysql.Open(dsn))
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

func initRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		//Addr: "localhost:6379",
		//Addr: "gobook-redis:11479",
		Addr: config.Config.Redis.Addr,
	})
	return redisClient
}

func InitViperV1() {
	//1.文件名
	viper.SetConfigName("dev")

	//2.确定文件dev的格式
	viper.SetConfigType("yaml")

	//确定文件的路径，可以指定多个路径
	//当前目录下的config子目录
	viper.AddConfigPath("./config")
	//viper.AddConfigPath("$HOME/.appname")
	//viper.AddConfigPath(".")

	//读取文件到viper
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

func InitViperV2() {
	////设置默认值，dev.yaml没有db.mysql.dsn时生效，否则以该文件为准。
	//viper.SetDefault("db.mysql.dsn",
	//	"root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local")

	viper.SetConfigFile("./config/dev.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config fileV2: %w", err))
	}
}

func InitViperV3() {
	//--config=config/dev.yaml
	cFile := pflag.String("config", "./config/dev.yaml", "指定配置文件路径")
	//从命名行解析
	pflag.Parse()
	viper.SetConfigFile(*cFile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config fileV3: %w", err))
	}
}

func InitViperRemote() {
	viper.SetConfigType("yaml")
	err := viper.AddRemoteProvider("etcd3", "127.0.0.1:12379", "/backEnd")
	if err != nil {
		panic(err)
	}
	err = viper.ReadRemoteConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config fileV4: %w", err))
	}
}

func InitWatchViper() {
	cFile := pflag.String("config", "./config/dev.yaml", "指定配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*cFile)

	//实时监听配置变更
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println(e.Name, e.Op)
	})

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config fileV5: %w", err))
	}
}

func initLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	zap.L().Info("logger Info...")
	//2026-07-12T19:36:15.092+0800	INFO	backEnd/main.go:262	logger Info...

	zap.L().Error("系统错误！", zap.Error(err))
	//2026-07-12T19:36:15.093+0800	ERROR	backEnd/main.go:264	系统错误！
	//main.initLogger
	///Users/erick/Code/Golang/GoBook/backEnd/main.go:264
	//main.main
	///Users/erick/Code/Golang/GoBook/backEnd/main.go:41
	//runtime.main
	///opt/homebrew/opt/go/libexec/src/runtime/proc.go:290

	zap.L().Info(
		"Info:",
		zap.Error(errors.New("Info--Error")),
		zap.Int64("Info-Int64-id:", 21),
		zap.Any("Info--key:", "hello zap any"),
	)
	//2026-07-12T19:36:15.093+0800	INFO	backEnd/main.go:266	Info:
	//	{"error": "Info--Error", "Info-Int64-id:": 21, "Info--key:": "hello zap any"}
}
