package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	cachemocks "GoBook/internal/repository/cache/mocks"
	"GoBook/internal/repository/dao"
	daomocks "GoBook/internal/repository/dao/mocks"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockgen -source=internal/repository/dao/user.go -package=daomocks -destination=internal/repository/dao/mocks/user.mock.go
// mockgen -source=internal/repository/cache/user.go -package=cachemocks -destination=internal/repository/cache/mocks/user.mock.go
func TestCacheUserRepository_FindById(t *testing.T) {
	now := time.Now()
	//去除毫秒外的部分
	now = time.UnixMilli(now.UnixMilli())
	testCases := []struct {
		name       string
		ctx        context.Context
		id         int64
		mock       func(ctrl *gomock.Controller) (dao.UserDAO, cache.UserCache)
		expectUser domain.User
		expectErr  error
	}{
		{
			name: "缓存未命中，查询成功！",
			ctx:  context.Background(),
			id:   1,
			mock: func(ctrl *gomock.Controller) (dao.UserDAO, cache.UserCache) {
				//缓存未命中
				cacheMock := cachemocks.NewMockUserCache(ctrl)
				cacheMock.EXPECT().Get(gomock.Any(), int64(1)).Return(domain.User{}, cache.ErrNotExists)

				//数据库查询成功
				daoMock := daomocks.NewMockUserDAO(ctrl)
				daoMock.EXPECT().FindById(gomock.Any(), int64(1)).Return(dao.User{
					Id: 1,
					Email: sql.NullString{
						String: "111@qq.com",
						Valid:  true,
					},
					Password: "123456",
					Phone: sql.NullString{
						String: "18888888888",
						Valid:  true,
					},
					Ctime: now.UnixMilli(),
					Utime: now.UnixMilli(),
				}, nil)

				//回写缓存
				cacheMock.EXPECT().Set(gomock.All(), domain.User{
					Id:       1,
					Email:    "111@qq.com",
					Password: "123456",
					Phone:    "18888888888",
					Ctime:    now,
				}).Return(nil)

				return daoMock, cacheMock
			},
			expectUser: domain.User{
				Id:       1,
				Email:    "111@qq.com",
				Password: "123456",
				Phone:    "18888888888",
				Ctime:    now,
			},
			expectErr: nil,
		},
		{
			name: "缓存命中，查询成功～",
			ctx:  context.Background(),
			id:   1,
			mock: func(ctrl *gomock.Controller) (dao.UserDAO, cache.UserCache) {
				//缓存未命中
				cacheMock := cachemocks.NewMockUserCache(ctrl)
				cacheMock.EXPECT().Get(gomock.Any(), int64(1)).Return(domain.User{
					Id:       1,
					Email:    "111@qq.com",
					Password: "123456",
					Phone:    "18888888888",
					Ctime:    now,
				}, nil)

				daoMock := daomocks.NewMockUserDAO(ctrl)

				return daoMock, cacheMock
			},
			expectUser: domain.User{
				Id:       1,
				Email:    "111@qq.com",
				Password: "123456",
				Phone:    "18888888888",
				Ctime:    now,
			},
			expectErr: nil,
		},
		{
			name: "缓存未命中，查询失败！",
			ctx:  context.Background(),
			id:   1,
			mock: func(ctrl *gomock.Controller) (dao.UserDAO, cache.UserCache) {
				//缓存未命中
				cacheMock := cachemocks.NewMockUserCache(ctrl)
				cacheMock.EXPECT().Get(gomock.Any(), int64(1)).Return(domain.User{}, cache.ErrNotExists)

				//数据库查询失败
				daoMock := daomocks.NewMockUserDAO(ctrl)
				daoMock.EXPECT().FindById(gomock.Any(), int64(1)).Return(dao.User{}, errors.New("mock db error"))

				return daoMock, cacheMock
			},
			expectUser: domain.User{},
			expectErr:  errors.New("mock db error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//初始化控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//func NewUserRepository(dao dao.UserDAO, cache cache.UserCache) UserRepository
			ud, uc := tc.mock(ctrl)
			repo := NewUserRepository(ud, uc)
			u, err := repo.FindById(tc.ctx, tc.id)
			assert.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectUser, u)
			time.Sleep(time.Second)
		})
	}
}
