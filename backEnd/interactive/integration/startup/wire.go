//go:build wireinject

package startup

import (
	"GoBook/interactive/repository"
	"GoBook/interactive/repository/cache"
	"GoBook/interactive/repository/dao"
	"GoBook/interactive/service"

	"github.com/google/wire"
)

// 第三方依赖
var thirdProviderSet = wire.NewSet(
	InitDB,
	InitRedis,
	InitLogger,
)

var interactiveProviderSet = wire.NewSet(
	cache.NewRedisInteractiveCache,
	dao.NewInteractiveDAO,
	repository.NewInteractiveRepository,
	service.NewInteractiveService,
)

func InitInteractiveService() service.InteractiveService {
	wire.Build(
		thirdProviderSet,
		interactiveProviderSet,
	)
	return nil
}
