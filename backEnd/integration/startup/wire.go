//go:build wireinject

package startup

import (
	"GoBook/internal/repository"
	"GoBook/internal/repository/article"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"GoBook/internal/service"
	"GoBook/internal/web"
	"GoBook/ioc"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

var thirdProviderSet = wire.NewSet(InitDB, InitRedis, InitLogger)

var userSvcProviderSet = wire.NewSet(
	dao.NewUserDAO,
	cache.NewUserCache,
	repository.NewUserRepository,
	service.NewUserService,
)

func InitWebServer() *gin.Engine {
	wire.Build(
		////基础的三方依赖
		//InitDB, InitRedis,

		////提供 *zap.Logger
		//InitLogger,

		thirdProviderSet,
		userSvcProviderSet,

		//初始化DAO，缓存
		//dao.NewUserDAO,
		//cache.NewUserCache,
		cache.NewCodeCache,
		dao.NewArticleDAO,

		//初始化repo
		//repository.NewUserRepository,
		repository.NewCodeRepository,
		article.NewArticleRepository,

		//初始化service
		//service.NewUserService,
		service.NewCodeService,
		service.NewArticleService,
		InitSMSService,
		InitOAuth2WechatService,
		NewOAuth2WechatConfig,
		NewRedisJWTHandler,

		//初始化handler
		web.NewUserHandler,
		web.NewOAuth2WechatHandler,
		web.NewArticleHandler,

		//初始化gin、路由、中间件
		ioc.InitGin,
		ioc.InitMiddleware,
	)
	return new(gin.Engine)
}

func InitArticleHandler() *web.ArticleHandler {
	wire.Build(
		thirdProviderSet,
		dao.NewArticleDAO,
		article.NewArticleRepository,
		service.NewArticleService,
		web.NewArticleHandler,
	)
	return &web.ArticleHandler{}
}

//func InitUserSvc() service.UserService {
//	wire.Build(thirdProviderSet, userSvcProviderSet)
//	return service.UserService{nil, nil}
//}
