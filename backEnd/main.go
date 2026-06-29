package main

import (
	"GoBook/internal/repository"
	"GoBook/internal/repository/dao"
	"GoBook/internal/service"
	"GoBook/internal/web"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
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

	ud := dao.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)
	u := web.NewUserHandler(svc)

	server := gin.Default()
	server.Use(cors.New(cors.Config{
		//prelight接口请求中，origin:http://localhost:3000
		//AllowOrigins: []string{"http://localhost:3000"},

		//access-control-request-method：POST
		AllowMethods: []string{"POST"},

		//access-control-request-headers:authorization,content-type
		//大小写都行
		AllowHeaders: []string{"Content-Type", "Authorization"},

		//ExposeHeaders: []string{"x-jwt-token"},

		//是否允许携带cookie
		AllowCredentials: true,

		//复杂请求配置
		AllowOriginFunc: func(origin string) bool {
			if strings.HasSuffix(origin, "http://localhost") {
				//开发环境
				return true
			}
			return strings.Contains(origin, "xxx.com")
		},

		MaxAge: 12 * time.Hour,
	}))

	u.RegisterUsersRouters(server)

	server.Run(":8080")
}
