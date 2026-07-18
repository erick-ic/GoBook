//go:build wireinject

package startup

import (
	article3 "GoBook/internal/events/article"
	"GoBook/internal/repository"
	"GoBook/internal/repository/article"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	article2 "GoBook/internal/repository/dao/article"
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
		cache.NewRedisArticleCache,
		article2.NewArticleDAO,
		article2.NewAuthorDAO,
		article2.NewReaderDAO,

		//初始化repo
		//repository.NewUserRepository,
		repository.NewCodeRepository,
		article.NewArticleRepository,
		//article.NewArticleAuthorRepository,
		//article.NewArticleReaderRepository,

		//初始化service
		//service.NewUserService,
		service.NewCodeService,
		service.NewArticleService,
		service.NewInteractiveService,
		dao.NewInteractiveDAO,
		cache.NewRedisInteractiveCache,
		repository.NewInteractiveRepository,

		//kafka
		ioc.InitSaramaClient,
		ioc.InitSyncProducer,
		article3.NewKafkaProducer,
		wire.Bind(new(article3.Producer), new(*article3.KafkaProducer)),
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
		article2.NewArticleDAO,
		article2.NewAuthorDAO,
		article2.NewReaderDAO,
		cache.NewRedisArticleCache,
		article.NewArticleRepository,
		service.NewArticleService,
		service.NewInteractiveService,
		dao.NewInteractiveDAO,
		cache.NewRedisInteractiveCache,
		repository.NewInteractiveRepository,
		web.NewArticleHandler,
		ioc.InitSaramaClient,
		ioc.InitSyncProducer,
		article3.NewKafkaProducer,
		wire.Bind(new(article3.Producer), new(*article3.KafkaProducer)),
	)
	return &web.ArticleHandler{}
}

//func InitUserSvc() service.UserService {
//	wire.Build(thirdProviderSet, userSvcProviderSet)
//	return service.UserService{nil, nil}
//}
