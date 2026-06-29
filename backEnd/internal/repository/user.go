package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
)

var (
	ErrUserDuplicated = dao.ErrUserDuplicated
	ErrUserNotFound   = dao.ErrUserNotFound
)

type UserRepository struct {
	dao *dao.UserDAO
}

func NewUserRepository(dao *dao.UserDAO) *UserRepository {
	return &UserRepository{dao: dao}
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
