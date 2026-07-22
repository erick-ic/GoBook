package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	svcmocks "GoBook/internal/service/mocks"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/logger"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestArticleHandler_Publish(t *testing.T) {
	testCases := []struct {
		name string

		ctx     context.Context
		reqBody string

		mock func(ctrl *gomock.Controller) service.ArticleService

		expectCode int
		expectBody Result
	}{
		{
			name: "新建并发表",
			ctx:  context.Background(),
			reqBody: `{
"title":"标题01",
"content":"内容01"

}`,
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				articleSvc.EXPECT().Publish(gomock.Any(), domain.Article{
					Title:   "标题01",
					Content: "内容01",
					Author: domain.Author{
						Id: 21,
					},
				}).Return(int64(1), nil)
				return articleSvc
			},
			expectCode: http.StatusOK,
			/*
				type Result struct {
					Code int    `json:"code"`
					Msg  string `json:"msg"`
					Data any    `json:"data"`
				}
			*/
			//Result中的Data字段类型是any，在json中会被转换为float64
			expectBody: Result{
				Code: 0,
				Msg:  "发表成功～",
				Data: float64(1),
			},
		},
		{
			name: "发表失败",
			ctx:  context.Background(),
			reqBody: `{
"title":"标题01",
"content":"内容01"

}`,
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				articleSvc := svcmocks.NewMockArticleService(ctrl)
				articleSvc.EXPECT().Publish(gomock.Any(), domain.Article{
					Title:   "标题01",
					Content: "内容01",
					Author: domain.Author{
						Id: 21,
					},
				}).Return(int64(0), errors.New("发表帖子失败"))
				return articleSvc
			},
			expectCode: http.StatusOK,
			/*
				type Result struct {
					Code int    `json:"code"`
					Msg  string `json:"msg"`
					Data any    `json:"data"`
				}
			*/
			//Result中的Data字段类型是any，在json中会被转换为float64
			expectBody: Result{
				Code: 5,
				Msg:  "系统错误！",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			server := gin.Default()
			//模拟用户登录，即存在session
			server.Use(func(ctx *gin.Context) {
				ctx.Set("claims", &ijwt.UserClaims{
					Uid: 21,
				})
			})

			h := NewArticleHandler(tc.mock(ctrl), &logger.NopLogger{}, nil)
			h.RegisterRouters(server)

			req, err := http.NewRequest(http.MethodPost, "/articles/publish", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			t.Log(req)

			resp := httptest.NewRecorder()
			t.Log(resp)

			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectCode, resp.Code)

			var webRes Result
			//反序列化到web.Result结构体
			err = json.NewDecoder(resp.Body).Decode(&webRes)
			require.NoError(t, err)
			assert.Equal(t, tc.expectBody, webRes)
		})
	}
}
