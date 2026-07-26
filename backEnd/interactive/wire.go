//go:build wireinject

package main

import (
	"GoBook/interactive/events"
	"GoBook/interactive/grpc"
	"GoBook/interactive/ioc"
	"GoBook/interactive/repository"
	"GoBook/interactive/repository/cache"
	"GoBook/interactive/repository/dao"
	"GoBook/interactive/service"

	"github.com/google/wire"
)

// thirdProviderSet 汇总数据库、缓存、日志和 Kafka 等基础设施依赖。
var thirdProviderSet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitLogger,
	ioc.InitSaramaClient,
)

// interactiveProviderSet 描述互动业务从 DAO 到 Service 的依赖链。
var interactiveProviderSet = wire.NewSet(
	cache.NewRedisInteractiveCache,
	dao.NewInteractiveDAO,
	repository.NewInteractiveRepository,
	service.NewInteractiveService,
)

// InitApp 装配可独立运行的互动微服务。
func InitApp() *App {
	wire.Build(
		thirdProviderSet,
		interactiveProviderSet,
		grpc.NewInteractiveServiceServer,
		// 启动阅读事件消费者。
		events.NewInteractiveReadEventConsumer,
		ioc.InitConsumers,

		ioc.InitGRPCxServer,

		// 注入 App 的全部字段。
		wire.Struct(new(App), "*"),
	)
	return new(App)
}
