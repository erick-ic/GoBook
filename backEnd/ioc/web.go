package ioc

import (
	"GoBook/internal/web"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/internal/web/middleware"
	"GoBook/internal/web/middleware/logger"
	"GoBook/pkg/ginx/middleware/ratelimit"
	logger2 "GoBook/pkg/logger"
	"context"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func InitGin(mdls []gin.HandlerFunc, hdl *web.UserHandler, oauth2 *web.OAuth2WechatHandler) *gin.Engine {
	server := gin.Default()
	server.Use(mdls...)
	hdl.RegisterUsersRouters(server)
	oauth2.RegisterWechatRouters(server)
	return server
}

func InitMiddleware(redisClient redis.Cmdable, jwtHandler ijwt.JWTHandler, l logger2.LoggerV1) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		//跨域中间件
		handleCors(),
		logger.NewMiddlewareBuilder(func(ctx context.Context, al *logger.AccessLog) {
			l.Debug("HTTP请求", logger2.Field{Key: "al", Value: al})
		}).SetAllowReqBody().Build(),
		//路由中间件
		middleware.NewLoginJWTMiddlewareBuilder(jwtHandler).
			IgnorePaths("/users/login").
			IgnorePaths("/users/signup").
			IgnorePaths("/users/sendSMSCode").
			IgnorePaths("/users/loginSMS").
			IgnorePaths("/oauth2/wechat/authurl").
			IgnorePaths("/oauth2/wechat/callback").
			IgnorePaths("/users/refreshToken").
			Build(),
		//限流中间件
		ratelimit.NewBuilder(redisClient, time.Second, 100).Build(),
	}
}

func handleCors() gin.HandlerFunc {
	return cors.New(cors.Config{
		//prelight接口请求中，origin:http://localhost:3000
		//AllowOrigins: []string{"http://localhost:3000"},

		//access-control-request-method：POST
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},

		//access-control-request-headers:authorization,content-type
		//大小写都行
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},

		//暴露给客户端
		ExposeHeaders: []string{"X-Total-Count", "X-JWT-Token", "x-refresh-token"},

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
	})
}
