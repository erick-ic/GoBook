package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"context"
	"errors"
	"log"
)

var (
	ErrUserDuplicated = dao.ErrUserDuplicated
	ErrUserNotFound   = dao.ErrUserNotFound
)

type UserRepository struct {
	dao   *dao.UserDAO
	cache *cache.UserCache
}

func NewUserRepository(dao *dao.UserDAO, cache *cache.UserCache) *UserRepository {
	return &UserRepository{
		dao:   dao,
		cache: cache,
	}
}

func (ur *UserRepository) Create(ctx context.Context, u domain.User) error {
	return ur.dao.Insert(ctx, dao.User{
		Email:    u.Email,
		Password: u.Password,
	})
}

func (ur *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := ur.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{
		Id:       int64(u.Id),
		Email:    u.Email,
		Password: u.Password,
	}, nil
}

func (ur *UserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	/*
		缓存需要面临的问题：数据一致性、缓存崩了
		数据查找顺序：
				1.先从cache里找
				2.没找到，再从dao里找
				3.从dao找到了，回写cache
	*/

	//1.先从cache里找
	u, cacheErr := ur.cache.Get(ctx, id)
	if cacheErr == nil {
		return u, nil // 缓存命中
	}

	//2.缓存未命中（redis.Nil）或缓存故障（其他错误）
	//redis没有这个数据，去数据库找
	ue, err := ur.dao.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	u = domain.User{
		Id:       int64(ue.Id),
		Email:    ue.Email,
		Password: ue.Password,
	}

	//如果是缓存未命中（而非故障），异步回写缓存
	if errors.Is(cacheErr, cache.ErrNotExists) {
		//开启协程，不阻塞主流程
		go func() {
			//从数据库中查到数据，回写redis
			if setErr := ur.cache.Set(ctx, u); setErr != nil {
				// 计入监控，或打印日志
				log.Printf("回写缓存失败: uid=%d, err=%v", u.Id, setErr)
			} else {
				log.Printf("回写缓存成功: uid=%d", u.Id)
			}
		}()
	}
	// 如果缓存故障（如连接超时），我们也可以不回写，或者同步尝试，但不要阻塞

	return u, err
}
