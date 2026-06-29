package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserDuplicated      = repository.ErrUserDuplicated
	ErrUserNotFund         = repository.ErrUserNotFound
	ErrInvalidUserPassword = errors.New("账号/邮箱或密码不对")
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
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)

	//存储
	return us.repo.Create(ctx, u)
}

func (us *UserService) Login(ctx context.Context, email, password string) (domain.User, error) {
	//找到目标用户
	u, err := us.repo.FindByEmail(ctx, email)
	//未找到用户
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, ErrUserNotFund
	}
	if err != nil {
		return domain.User{}, err
	}
	//比较密码
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		//密码错误，写入日志
		return domain.User{}, ErrInvalidUserPassword
	}
	return u, nil
}
