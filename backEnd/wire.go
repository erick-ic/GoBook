//go:build wireinject

package main

import (
	"GoBook/internal/repository"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"GoBook/internal/service"
	"GoBook/internal/web"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/ioc"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		//基础的三方依赖
		ioc.InitDB, ioc.InitRedis,

		//初始化DAO，缓存
		dao.NewUserDAO,
		cache.NewUserCache,
		cache.NewCodeCache,

		//初始化repo
		repository.NewUserRepository,
		repository.NewCodeRepository,

		//初始化service
		service.NewUserService,
		service.NewCodeService,
		ioc.InitSMSService,
		ioc.InitOAuth2WechatService,
		ioc.NewOAuth2WechatConfig,
		ijwt.NewRedisJWTHandler,

		//初始化handler
		web.NewUserHandler,
		web.NewOAuth2WechatHandler,

		//初始化gin、路由、中间件
		ioc.InitGin,
		ioc.InitMiddleware,
	)
	return new(gin.Engine)
}
