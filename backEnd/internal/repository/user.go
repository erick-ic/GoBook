package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
)

var ErrUserDuplicated = dao.ErrUserDuplicated

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
