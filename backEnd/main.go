package main

import (
	"GoBook/internal/repository"
	"GoBook/internal/repository/dao"
	"GoBook/internal/service"
	"GoBook/internal/web"
	"GoBook/internal/web/middleware"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	db := initDB()
	server := initWebServer()

	u := initUser(db)
	u.RegisterUsersRouters(server)

	server.Run(":8080")
}

func initWebServer() *gin.Engine {
	server := gin.Default()
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
	store, _ := redis.NewStore(
		16,
		"tcp",
		"localhost:6379",
		"",
		"",
		[]byte("3akQBTZmfkuEjQacH5hvUynDnmPvAf7Y"),
		[]byte("Z4d8tz8WDKXT3AvYJkmhEb5VEFfxHHS2"))

	////d.限流，一秒钟100次
	//redisClient := redis.NewClient(&redis.Options{
	//	Addr: "localhost:6379",
	//})
	//server.Use(ratelimit.NewBuilder(redisClient, time.Second, 100).Build())
	//store := memstore.NewStore([]byte("3akQBTZmfkuEjQacH5hvUynDnmPvAf7Y"),
	//	[]byte("Z4d8tz8WDKXT3AvYJkmhEb5VEFfxHHS2"))
	//
	server.Use(sessions.Sessions("mysession", store))
	//步骤3: 登录校验
	//server.Use(
	//	middleware.NewLoginMiddlewareBuilder().
	//		IgnoreLoginMiddlewareBuilder("/users/login").
	//		IgnoreLoginMiddlewareBuilder("/users/signup").
	//		Build(),
	//)

	server.Use(middleware.NewLoginJWTMiddlewareBuilder().
		IgnorePaths("/users/login").
		IgnorePaths("/users/signup").
		Build())
	return server
}

func initUser(db *gorm.DB) *web.UserHandler {
	ud := dao.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)
	u := web.NewUserHandler(svc)
	return u
}

func initDB() *gorm.DB {
	//数据库连接
	dsn := "root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local"
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
