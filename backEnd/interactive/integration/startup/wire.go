//go:build wireinject

package startup

import (
	"GoBook/interactive/grpc"
	"GoBook/interactive/repository"
	"GoBook/interactive/repository/cache"
	"GoBook/interactive/repository/dao"
	"GoBook/interactive/service"

	"github.com/google/wire"
)

// thirdProviderSet 汇总集成测试使用的数据库、Redis 和日志依赖。
var thirdProviderSet = wire.NewSet(
	InitDB,
	InitRedis,
	InitLogger,
)

// interactiveProviderSet 描述互动服务的完整业务依赖链。
var interactiveProviderSet = wire.NewSet(
	cache.NewRedisInteractiveCache,
	dao.NewInteractiveDAO,
	repository.NewInteractiveRepository,
	service.NewInteractiveService,
)

// InitInteractiveService 装配供服务层集成测试使用的互动服务。
func InitInteractiveService() service.InteractiveService {
	wire.Build(
		thirdProviderSet,
		interactiveProviderSet,
	)
	return nil
}

// InitInteractiveGRPCService 装配供 gRPC 适配层集成测试使用的服务端。
func InitInteractiveGRPCService() *grpc.InteractiveServiceServer {
	wire.Build(
		thirdProviderSet,
		interactiveProviderSet,
		grpc.NewInteractiveServiceServer,
	)
	return nil
}
