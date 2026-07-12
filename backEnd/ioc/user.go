package ioc

import (
	"GoBook/internal/repository"
	"GoBook/internal/service"

	"go.uber.org/zap"
)

func InitUserHandler(repo repository.UserRepository, l *zap.Logger) service.UserService {
	return service.NewUserService(repo, l)
}
