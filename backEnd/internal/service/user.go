package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (us *UserService) SignUp(ctx context.Context, u domain.User) error {
	//加密
	//存储
	return us.repo.Create(ctx, u)
}
