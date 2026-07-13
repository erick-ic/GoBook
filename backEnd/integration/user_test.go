package integration

import (
	"GoBook/integration/startup"
	"GoBook/internal/web"
	"GoBook/ioc"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserHandler_e2e_SendLoginSMSCode(t *testing.T) {
	//构建完整的Gin引擎，依赖于wire生成的依赖注入
	server := startup.InitWebServer()
	//初始化redis
	rdb := ioc.InitRedis()

	testCases := []struct {
		//用例名称
		name string

		//上下文管理
		ctx context.Context
		//JSON请求体
		reqBody string

		//测试前置操作，用于准备测试数据，如往 Redis 预置验证码
		before func(t *testing.T)
		//测试后清理/验证，如删除对应的key，之后进行断言
		after func(t *testing.T)

		//期望状态吗
		expectCode int
		//期望响应体
		expectBody web.Result
	}{
		{
			name: "发送成功～",
			ctx:  context.Background(),
			reqBody: `{
"phone":"18712345678"
}`,
			before: func(t *testing.T) {
				//Redis 暂无数据，不需要处理
			},
			after: func(t *testing.T) {
				//context.WithTimeout 控制 Redis 操作超时，防止测试卡死
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				val, err := rdb.GetDel(ctx, "phone_code:login:18712345678").Result()
				cancel()
				assert.NoError(t, err)
				//无法获取redis中验证码具体数值，判断长度进行。
				assert.True(t, len(val) == 6)
			},
			expectCode: http.StatusOK,
			expectBody: web.Result{
				Code: 0,
				Msg:  "发送成功～",
			},
		},
		{
			name: "发送频繁！",
			ctx:  context.Background(),
			reqBody: `{
"phone":"18712345678"
}`,
			before: func(t *testing.T) {
				//Redis 中已经存了一个验证码
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				//发送验证码刚过30秒
				_, err := rdb.Set(ctx, "phone_code:login:18712345678",
					"123456",
					time.Minute*9+time.Second*30).Result()
				cancel()
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				val, err := rdb.GetDel(ctx, "phone_code:login:18712345678").Result()
				cancel()
				assert.NoError(t, err)
				//验证码没有被覆盖，仍为“123456”
				assert.Equal(t, "123456", val)
			},
			expectCode: http.StatusOK,
			expectBody: web.Result{
				Code: 5,
				Msg:  "短信发送频繁，请稍后再试！",
			},
		},
		{
			name: "系统错误！",
			ctx:  context.Background(),
			reqBody: `{
"phone":"18712345678"
}`,
			before: func(t *testing.T) {
				//Redis 中已经存了一个验证码，但不存在过期时间
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				_, err := rdb.Set(ctx, "phone_code:login:18712345678", "123456", 0).Result()
				cancel()
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				val, err := rdb.GetDel(ctx, "phone_code:login:18712345678").Result()
				cancel()
				assert.NoError(t, err)
				//保留验证码
				assert.Equal(t, "123456", val)
			},
			expectCode: http.StatusOK,
			expectBody: web.Result{
				Code: 5,
				Msg:  "系统异常!",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//准备数据
			tc.before(t)

			//构造http请求：
			req, err := http.NewRequest(http.MethodPost, "/users/sendSMSCode", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			//设置请求头
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			t.Log(req)

			//创建响应记录器，处理响应：
			resp := httptest.NewRecorder()
			t.Log(resp)

			//http请求进入gin框架的入口
			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectCode, resp.Code)

			var webRes web.Result
			//反序列化到web.Result结构体
			err = json.NewDecoder(resp.Body).Decode(&webRes)
			require.NoError(t, err)
			assert.Equal(t, tc.expectBody, webRes)

			//验证redis状态或清理
			tc.after(t)
		})
	}
}
