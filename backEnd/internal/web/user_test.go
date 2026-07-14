package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	svcmocks "GoBook/internal/service/mocks"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// 核心指令：
// mockgen -source=internal/service/user.go -package=svcmocks -destination=internal/service/mocks/user.mock.go
func TestMock(t *testing.T) {
	//1.初始化控制器，创建一个控制器，负责管理所有 Mock 对象的生命周期和期望校验。
	//传入 *testing.T 是为了在断言失败时自动报告错误。
	ctrl := gomock.NewController(t)

	//2.这是最关键的一行。
	//校验所有设置的 期望（EXPECT） 是否都被调用了（比如你设置了必须调用 1 次，但代码没调，这里就会报错）。
	//校验调用时的参数是否匹配预期。
	//记住：必须加 defer ctrl.Finish()，否则 Mock 的校验不会执行。
	defer ctrl.Finish()

	//3.创建mock对象并设置行为，对应mock.user.go中的方法
	//由 mockgen 生成的构造函数，传入控制器，生成一个实现了 UserService 接口的 Mock 对象。
	userSvc := svcmocks.NewMockUserService(ctrl)

	//4.获取该 Mock 对象的期望记录器，用于设置“当某个方法被调用时，应该发生什么”。
	userSvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).
		// SignUp(ctx context.Context, u domain.User) error
		Return(errors.New("mock error"))

	//5.执行调用并打印结果
	err := userSvc.SignUp(context.Background(), domain.User{
		Email: "111@qq.com",
	})
	t.Log(err)
	//mock error
}

// mockgen -source=internal/service/user.go -package=svcmocks -destination=internal/service/mocks/user.mock.go

// 结构体切片
//
//	[]struct{}{
//		{
//			"attr":"",
//			"func":""
//		}
//	}
func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		//用例名称
		name string

		//上下文控制
		ctx context.Context
		//原始 HTTP 请求体（JSON 字符串），模拟前端/客户端发来的数据
		reqBody string

		//返回 Mock Service 的函数
		mock func(ctrl *gomock.Controller) service.UserService

		//预期的 HTTP 状态码（如 200/400）
		expectCode int
		//预期的响应体字符串
		expectBody string
	}{
		{
			name: "注册成功",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"hello@123",
"confirm_password":"hello@123"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "222@qq.com",
					Password: "hello@123",
				}).
					Return(nil)
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "SignUp success~",
		},
		{
			name: "参数不对，Bind失败！",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"hello@123",
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "邮箱格式不对！",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@q",
"password":"hello@123",
"confirm_password":"hello@123"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "邮箱格式错误！",
		},
		{
			name: "两次输入的密码不匹配",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"hello@122",
"confirm_password":"hello@123"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "两次输入的密码不一致",
		},
		{
			name: "密码格式错误！",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"111",
"confirm_password":"111"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "密码必须大于8位，包含数字、特殊字符",
		},
		{
			name: "邮箱重复了！",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"hello@123",
"confirm_password":"hello@123"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "222@qq.com",
					Password: "hello@123",
				}).
					Return(service.ErrUserDuplicated)
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "邮箱重复，请换一个！",
		},
		{
			name: "系统异常！",
			ctx:  context.Background(),
			reqBody: `
{
"email":"222@qq.com",
"password":"hello@123",
"confirm_password":"hello@123"
}
`,
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().SignUp(gomock.Any(), domain.User{
					Email:    "222@qq.com",
					Password: "hello@123",
				}).
					Return(errors.New("任意一个 error"))
				return userSvc
			},
			expectCode: http.StatusOK,
			expectBody: "系统错误",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//Step1.初始化控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//构造真实 Handler：
			server := gin.Default()
			h := NewUserHandler(tc.mock(ctrl), nil, nil, nil)
			//注册路由
			h.RegisterUsersRouters(server)

			//构造http请求：
			req, errr := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, errr)
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			t.Log(req)

			//处理响应：
			resp := httptest.NewRecorder()
			t.Log(resp)

			//http请求进入gin框架的入口
			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectCode, resp.Code)
			assert.Equal(t, tc.expectBody, resp.Body.String())
		})
	}
}
