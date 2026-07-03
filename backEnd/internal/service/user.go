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

// SignUp 注册
func (us *UserService) SignUp(ctx context.Context, u domain.User) error {
	/*
		bcrypt.GenerateFromPassword：接受明文密码（[]byte）和计算成本（cost）参数。
		cost（成本）：控制加密强度，bcrypt.DefaultCost 为 10，值越大加密越慢也越安全（通常 10~12 即可）。
		自动加盐：bcrypt 会自动生成随机盐（Salt）并嵌入最终哈希字符串中，无需额外存储盐值。
		返回值：加密后的哈希字符串，格式如 $2a$10$...，可直接以字符串形式存入数据库。
	*/
	// 1. 生成加密哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// 2. 用哈希值替换明文密码
	u.Password = string(hash)

	// 3. 存储到数据库
	return us.repo.Create(ctx, u)
}

// Login 登录
func (us *UserService) Login(ctx context.Context, email, password string) (domain.User, error) {
	/*
		bcrypt.CompareHashAndPassword：接受已存储的哈希（[]byte）和登录时输入的明文密码（[]byte）。
		内部流程：该函数会从哈希中提取盐值，对明文密码使用相同盐值重新计算哈希，然后与存储的哈希比较。
		匹配与错误：
			若密码正确，返回 nil。
			若密码错误，返回 bcrypt.ErrMismatchedHashAndPassword（通常被包装为自定义错误如 ErrInvalidUserPassword）。
	*/
	// 1. 根据邮箱从数据库查询用户信息（含加密密码）
	u, err := us.repo.FindByEmail(ctx, email)
	//未找到用户
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, ErrUserNotFund
	}
	if err != nil {
		return domain.User{}, err
	}
	// 2. 比较明文密码与数据库中存储的哈希
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		//密码错误，写入日志
		return domain.User{}, ErrInvalidUserPassword
	}
	return u, nil
}

// Profile 简介
func (us *UserService) Profile(ctx context.Context, id int64) (domain.User, error) {
	u, err := us.repo.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}
