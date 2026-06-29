package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserDuplicated = repository.ErrUserDuplicated

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
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)

	//存储
	return us.repo.Create(ctx, u)
}
