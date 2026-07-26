//go:build wireinject

package main

import (
	"GoBook/interactive/events"
	repository2 "GoBook/interactive/repository"
	cache2 "GoBook/interactive/repository/cache"
	dao2 "GoBook/interactive/repository/dao"
	service2 "GoBook/interactive/service"
	"GoBook/internal/events/article"
	"GoBook/internal/repository"
	articleRepo "GoBook/internal/repository/article"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	article2 "GoBook/internal/repository/dao/article"
	"GoBook/internal/service"
	"GoBook/internal/web"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/ioc"

	"github.com/google/wire"
)

// RankingServiceSet 串起“批量计算 -> 仓储 -> Redis 缓存”的排行榜依赖。
var RankingServiceSet = wire.NewSet(
	cache.NewRankingRedisCache,
	cache.NewRankingLocalCache,
	repository.NewCachedRankingRepository,
	service.NewBatchRankingService,
)

func InitApp() *App {
	wire.Build(
		//基础的三方依赖
		ioc.InitDB,
		ioc.InitRedis,

		ioc.InitRlockClient,

		//提供 *zap.Logger
		ioc.InitLogger,

		//初始化Kafka
		ioc.InitSaramaClient,
		ioc.InitSyncProducer,
		//单次消费
		//ioc.InitConsumers,
		//批量消费
		ioc.InitBatchConsumers,

		RankingServiceSet,
		ioc.InitRankingJob,
		ioc.InitJobs,
		ioc.InitClosers,

		article.NewKafkaProducer,
		//单次
		//article.NewInteractiveReadEventConsumer,
		//批量
		events.NewInteractiveReadEventBatchConsumer,

		// 同时装配本地实现和远程 gRPC，通过配置阈值灰度切流。
		ioc.InitInteractiveGRPCClient,

		//初始化DAO，缓存
		dao.NewUserDAO,
		dao2.NewInteractiveDAO,
		article2.NewArticleDAO,
		article2.NewAuthorDAO,
		article2.NewReaderDAO,
		cache.NewUserCache,
		cache.NewCodeCache,
		cache.NewRedisArticleCache,
		cache2.NewRedisInteractiveCache,

		//初始化repo
		repository.NewUserRepository,
		repository.NewCodeRepository,
		repository2.NewInteractiveRepository,
		articleRepo.NewArticleRepository,

		//初始化service
		service.NewUserService,
		service.NewCodeService,
		service.NewArticleService,
		service2.NewInteractiveService,
		ioc.InitSMSService,
		ioc.InitOAuth2WechatService,
		ioc.NewOAuth2WechatConfig,
		ijwt.NewRedisJWTHandler,

		//初始化handler
		web.NewUserHandler,
		web.NewOAuth2WechatHandler,
		web.NewArticleHandler,

		//测试metric
		web.NewObserverAbilityHandler,

		//初始化gin、路由、中间件
		ioc.InitGin,
		ioc.InitMiddleware,

		wire.Struct(new(App), "*"),
	)
	return new(App)
}
