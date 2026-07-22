package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	repomocks "GoBook/internal/repository/mocks"
	"GoBook/pkg/logger"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

// mockgen -source=internal/repository/user.go -package=repomocks -destination=internal/repository/mocks/user.mock.go
// mockgen -source=internal/repository/code.go -package=repomocks -destination=internal/repository/mocks/code.mock.go

//go:generate mockgen -source=../repository/user.go -package=repomocks -destination=internal/repository/mocks/user.mock.go
//go:generate mockgen -source=../repository/code.go -package=repomocks -destination=internal/repository/mocks/code.mock.go
func Test_userService_Login(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name string

		ctx      context.Context
		email    string
		password string

		mock func(ctrl *gomock.Controller) repository.UserRepository

		expectUser domain.User
		expectCode int
		expectErr  error
	}{
		{
			name:     "success",
			ctx:      context.Background(),
			email:    "111@qq.com",
			password: "hello@123",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "111@qq.com").
					Return(domain.User{
						Email:    "111@qq.com",
						Password: "$2a$10$H3HUJV2QOVsIPXKmIGqIVeE/jIjz3kF/vcGAh231n6t5oII46AEI2",
						Phone:    "17516161818",
						Ctime:    now,
					}, nil)
				return repo

			},
			expectUser: domain.User{
				Email:    "111@qq.com",
				Password: "$2a$10$H3HUJV2QOVsIPXKmIGqIVeE/jIjz3kF/vcGAh231n6t5oII46AEI2",
				Phone:    "17516161818",
				Ctime:    now,
			},
			expectCode: http.StatusOK,
			expectErr:  nil,
		},
		{
			name:     "用户不存在！",
			ctx:      context.Background(),
			email:    "111@qq.com",
			password: "hello@123",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "111@qq.com").
					Return(domain.User{}, repository.ErrUserNotFound)
				return repo

			},
			expectUser: domain.User{},
			expectCode: http.StatusOK,
			expectErr:  ErrInvalidUserPassword,
		},
		{
			name:     "DB错误！",
			ctx:      context.Background(),
			email:    "111@qq.com",
			password: "hello@123",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "111@qq.com").
					Return(domain.User{}, errors.New("任意错误！"))
				return repo

			},
			expectUser: domain.User{},
			expectCode: http.StatusOK,
			expectErr:  errors.New("任意错误！"),
		},
		{
			name:     "密码错误！",
			ctx:      context.Background(),
			email:    "111@qq.com",
			password: "hello@456",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(gomock.Any(), "111@qq.com").
					Return(domain.User{}, errors.New("账号/邮箱或密码不对"))
				return repo

			},
			expectUser: domain.User{},
			expectCode: http.StatusOK,
			expectErr:  ErrInvalidUserPassword,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//初始化控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//func NewUserService(repo repository.UserRepository) UserService
			svc := NewUserService(tc.mock(ctrl), &logger.NopLogger{})
			u, err := svc.Login(tc.ctx, tc.email, tc.password)
			assert.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectUser, u)
		})
	}
}

func Test_bcryptPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("hello@123"), bcrypt.DefaultCost)
	if err == nil {
		t.Log(string(hash))
	}
}
