//go:build wireinject

package startup

import (
	"GoBook/internal/repository"
	"GoBook/internal/repository/article"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	articleDAO "GoBook/internal/repository/dao/article"
	"GoBook/internal/service"
	"GoBook/internal/web"
	"GoBook/ioc"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

// 第三方依赖
var thirdProviderSet = wire.NewSet(
	InitDB,
	InitRedis,
	InitLogger,
)

var userSvcProviderSet = wire.NewSet(
	cache.NewUserCache,
	dao.NewUserDAO,
	repository.NewUserRepository,
	service.NewUserService,
)

var codeSvcProviderSet = wire.NewSet(
	cache.NewCodeCache,
	repository.NewCodeRepository,
	service.NewCodeService,
)
var articleSvcProviderSet = wire.NewSet(
	articleDAO.NewArticleDAO,
	articleDAO.NewAuthorDAO,
	articleDAO.NewReaderDAO,
	cache.NewRedisArticleCache,
	article.NewArticleRepository,
	service.NewArticleService,
)

var interactiveProviderSet = wire.NewSet(
	cache.NewRedisInteractiveCache,
	dao.NewInteractiveDAO,
	repository.NewInteractiveRepository,
	service.NewInteractiveService,
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 当前集成测试不验证 Kafka，注入无网络副作用的 Producer。
		InitArticleProducer,
		InitSMSService,
		InitOAuth2WechatService,
		NewOAuth2WechatConfig,

		// service
		thirdProviderSet,
		userSvcProviderSet,
		codeSvcProviderSet,
		articleSvcProviderSet,
		interactiveProviderSet,

		// handler
		web.NewUserHandler,
		web.NewOAuth2WechatHandler,
		web.NewArticleHandler,
		web.NewObserverAbilityHandler,
		NewRedisJWTHandler,

		//初始化gin、路由、中间件
		ioc.InitGin,
		ioc.InitMiddleware,
	)
	return new(gin.Engine)
}

func InitArticleHandler() *web.ArticleHandler {
	wire.Build(
		thirdProviderSet,
		articleDAO.NewArticleDAO,
		articleDAO.NewAuthorDAO,
		articleDAO.NewReaderDAO,
		cache.NewRedisArticleCache,
		article.NewArticleRepository,
		service.NewArticleService,
		service.NewInteractiveService,
		dao.NewInteractiveDAO,
		cache.NewRedisInteractiveCache,
		repository.NewInteractiveRepository,
		web.NewArticleHandler,
		// 编辑文章测试不需要发送阅读事件。
		InitArticleProducer,
	)
	return &web.ArticleHandler{}
}

//func InitUserSvc() service.UserService {
//	wire.Build(thirdProviderSet, userSvcProviderSet)
//	return service.UserService{nil, nil}
//}
