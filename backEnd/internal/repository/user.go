package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

var (
	ErrUserDuplicated = dao.ErrUserDuplicated
	ErrUserNotFound   = dao.ErrUserNotFound
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
}
type CacheUserRepository struct {
	dao   dao.UserDAO
	cache cache.UserCache
}

func NewUserRepository(dao dao.UserDAO, cache cache.UserCache) UserRepository {
	return &CacheUserRepository{
		dao:   dao,
		cache: cache,
	}
}

// Create 创建
func (ur *CacheUserRepository) Create(ctx context.Context, u domain.User) error {
	//return ur.dao.Insert(ctx, dao.User{
	//	Email:    u.Email,
	//	Password: u.Password,
	//})
	return ur.dao.Insert(ctx, ur.domainToEntity(u)) // string → sql.NullString
}

// FindByPhone 查找
func (ur *CacheUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	u, err := ur.dao.FindByPhone(ctx, phone)
	if err != nil {
		return domain.User{}, err
	}
	//return domain.User{
	//	Id:       int64(u.Id),
	//	Email:    u.Email,
	//	Password: u.Password,
	//}, nil
	return ur.entityToDomain(u), nil // sql.NullString → string
}

func (ur *CacheUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := ur.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	//return domain.User{
	//	Id:       int64(u.Id),
	//	Email:    u.Email,
	//	Password: u.Password,
	//}, nil
	return ur.entityToDomain(u), nil
}

func (ur *CacheUserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
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

	//u = domain.User{
	//	Id:       int64(ue.Id),
	//	Email:    ue.Email,
	//	Password: ue.Password,
	//}
	u = ur.entityToDomain(ue)

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

// DAO → Domain
func (ur *CacheUserRepository) entityToDomain(u dao.User) domain.User {
	return domain.User{
		Id:       int64(u.Id),
		Email:    u.Email.String,
		Password: u.Password,
		Phone:    u.Phone.String,
		Ctime:    time.UnixMilli(u.Ctime),
	}
}

// Domain → DAO/、
func (ur *CacheUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id:       int(u.Id),
		Email:    sql.NullString{String: u.Email, Valid: u.Email != ""},
		Password: u.Password,
		Phone:    sql.NullString{String: u.Phone, Valid: u.Phone != ""},
		Ctime:    u.Ctime.UnixMilli(),
	}
}
