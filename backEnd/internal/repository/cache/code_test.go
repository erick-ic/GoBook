package cache

import (
	"GoBook/internal/repository/cache/redismocks"
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockgen -package=redismocks -destination=internal/repository/cache/redismocks/cmdable.mock.go github.com/redis/go-redis/v9 Cmdable
func TestRedisCodeCache_SetCode(t *testing.T) {
	testCases := []struct {
		name  string
		ctx   context.Context
		biz   string
		phone string
		code  string

		mock      func(ctrl *gomock.Controller) redis.Cmdable
		expectErr error
	}{
		{
			name:  "验证码设置成功～",
			ctx:   context.Background(),
			biz:   "login",
			phone: "187",
			code:  "123456",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)

				//cmd.EXPECT().Eval返回的是具体类型Cmd
				//创建一个空的结果对象
				res := redis.NewCmd(context.Background())
				//设置结果
				res.SetVal(int64(0))

				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					[]string{"phone_code:login:187"},
					[]any{"123456"},
				).Return(res) //返回预制结果

				return cmd
			},
			expectErr: nil,
		},
		{
			name:  "redis错误！",
			ctx:   context.Background(),
			biz:   "login",
			phone: "187",
			code:  "123456",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				res := redis.NewCmd(context.Background())

				res.SetErr(errors.New("mock redis error"))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					[]string{"phone_code:login:187"},
					[]any{"123456"},
				).Return(res)

				return cmd
			},
			expectErr: errors.New("mock redis error"),
		},
		{
			name:  "发送频繁！",
			ctx:   context.Background(),
			biz:   "login",
			phone: "187",
			code:  "123456",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(-1))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					[]string{"phone_code:login:187"},
					[]any{"123456"},
				).Return(res)

				return cmd
			},
			expectErr: ErrCodeSendTooMany,
		},
		{
			name:  "系统错误！",
			ctx:   context.Background(),
			biz:   "login",
			phone: "187",
			code:  "123456",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := redismocks.NewMockCmdable(ctrl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(-10))
				cmd.EXPECT().Eval(gomock.Any(), luaSetCode,
					[]string{"phone_code:login:187"},
					[]any{"123456"},
				).Return(res)

				return cmd
			},
			expectErr: errors.New("系统错误！"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//创建控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//func NewCodeCache(client redis.Cmdable) CodeCache
			c := NewCodeCache(tc.mock(ctrl))
			err := c.SetCode(tc.ctx, tc.biz, tc.phone, tc.code)
			assert.Equal(t, tc.expectErr, err)
		})
	}
}
