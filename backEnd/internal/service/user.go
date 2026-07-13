package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"GoBook/pkg/logger"
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

type UserService interface {
	SignUp(ctx context.Context, u domain.User) error
	Login(ctx context.Context, email, password string) (domain.User, error)
	Profile(ctx context.Context, id int64) (domain.User, error)
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
	FindOrCreateByWechat(ctx context.Context, wechatInfo domain.WechatInfo) (domain.User, error)
}
type userService struct {
	repo   repository.UserRepository
	logger logger.LoggerV1
}

func NewUserService(repo repository.UserRepository, l logger.LoggerV1) UserService {
	return &userService{
		repo:   repo,
		logger: l,
	}
}

// SignUp 注册
func (us *userService) SignUp(ctx context.Context, u domain.User) error {
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
func (us *userService) Login(ctx context.Context, email, password string) (domain.User, error) {
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
		return domain.User{}, ErrInvalidUserPassword
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
func (us *userService) Profile(ctx context.Context, id int64) (domain.User, error) {
	u, err := us.repo.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (us *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	u, err := us.repo.FindByPhone(ctx, phone)
	//1. 先检查用户是否存在
	if !errors.Is(err, repository.ErrUserNotFound) {
		// 如果错误不是“未找到”，说明要么找到了，要么DB出错了，直接返回
		return u, err
	}
	////用法1：直接使用包变量
	//zap.L().Info("用户未注册", zap.String("phone", phone))

	////用法2：使用注入的logger
	//us.logger.Info("用户未注册", zap.String("phone", phone))

	//用法3：
	us.logger.Info("用户未注册", logger.String("phone", phone))

	// 2. 用户不存在，新建一个（仅赋值手机号，邮箱为空）
	user := domain.User{
		Phone: phone,
	}
	err = us.repo.Create(ctx, user)
	// 如果创建冲突（极低概率并发创建），忽略冲突错误
	if err != nil && !errors.Is(err, repository.ErrUserDuplicated) {
		return user, err
	}

	// 3. 重新查询并返回（处理主从延迟问题）
	return us.repo.FindByPhone(ctx, phone)
}

func (us *userService) FindOrCreateByWechat(ctx context.Context, wechatInfo domain.WechatInfo) (domain.User, error) {
	u, err := us.repo.FindByWechat(ctx, wechatInfo.OpenId)
	//1. 先检查用户是否存在
	if !errors.Is(err, repository.ErrUserNotFound) {
		// 如果错误不是“未找到”，说明要么找到了，要么DB出错了，直接返回
		return u, err
	}
	// 2. 用户不存在，新建一个（仅赋值手机号，邮箱为空）
	user := domain.User{
		WechatInfo: wechatInfo,
	}
	err = us.repo.Create(ctx, user)
	// 如果创建冲突（极低概率并发创建），忽略冲突错误
	if err != nil && !errors.Is(err, repository.ErrUserDuplicated) {
		return user, err
	}

	// 3. 重新查询并返回（处理主从延迟问题）
	return us.repo.FindByWechat(ctx, wechatInfo.OpenId)
}
