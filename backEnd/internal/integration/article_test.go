package integration

import (
	"GoBook/internal/domain"
	startup2 "GoBook/internal/integration/startup"
	newDAO "GoBook/internal/repository/dao/article"
	ijwt "GoBook/internal/web/jwt"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ArticleTestSuite struct {
	suite.Suite
	server *gin.Engine
	db     *gorm.DB
}

func (s *ArticleTestSuite) SetupSuite() {
	//在所有测试执行前，初始化配置
	//s.server = startup.InitWebServer()
	s.server = gin.Default()

	//模拟用户登录，即存在session
	s.server.Use(func(ctx *gin.Context) {
		ctx.Set("claims", &ijwt.UserClaims{
			Uid: 21,
		})
	})
	s.db = startup2.InitDB()

	//articleHandler := web.NewArticleHandler()
	articleHandler := startup2.InitArticleHandler()
	articleHandler.RegisterRouters(s.server)
}

func (s *ArticleTestSuite) TearDownTest() {
	// 兜底清理测试数据，避免影响其他集成测试。
	s.cleanArticles(s.T())
}

func (s *ArticleTestSuite) cleanArticles(t *testing.T) {
	t.Helper()
	// TRUNCATE 同时重置自增主键，保证“新建帖子”用例稳定得到 ID 1。
	require.NoError(t, s.db.Exec("TRUNCATE TABLE articles").Error)
}

func TestArticle(t *testing.T) {
	suite.Run(t, &ArticleTestSuite{})
}

func (s *ArticleTestSuite) TestEdit() {
	t := s.T()
	testCases := []struct {
		name string

		ctx     context.Context
		article Article

		before func(t *testing.T)
		after  func(t *testing.T)

		expectCode int
		expectBody Result[int64]
	}{
		{
			name: "新建帖子成功",
			ctx:  context.Background(),
			article: Article{
				Title:   "我的标题",
				Content: "我的内容",
			},
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				//验证数据库数据
				var article newDAO.Article
				err := s.db.Where("id = ?", 1).First(&article).Error
				assert.NoError(t, err)
				assert.True(t, article.Ctime > 0)
				assert.True(t, article.Utime > 0)
				article.Ctime = 0
				article.Utime = 0
				assert.Equal(t, newDAO.Article{
					Id:       1,
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 21,
					Ctime:    0,
					Utime:    0,
					Status:   domain.ArticleStatusUnPublished.ToUint8(),
				}, article)
			},
			expectCode: http.StatusOK,
			expectBody: Result[int64]{
				Msg:  "编辑成功～",
				Data: 1,
			},
		},
		{
			name: "编辑帖子成功",
			ctx:  context.Background(),
			article: Article{
				Id:      2,
				Title:   "我的标题111",
				Content: "我的内容111",
			},
			before: func(t *testing.T) {
				//模拟已经存在的数据
				err := s.db.Create(&newDAO.Article{
					Id:       2,
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 21,
					Ctime:    111,
					Utime:    222,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}).Error
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				//验证数据库数据
				var article newDAO.Article
				err := s.db.Where("id = ?", 2).First(&article).Error
				assert.NoError(t, err)

				//确保已经更新
				assert.True(t, article.Utime > 222)
				article.Utime = 0
				assert.Equal(t, newDAO.Article{
					Id:       2,
					Title:    "我的标题111",
					Content:  "我的内容111",
					AuthorId: 21,
					Ctime:    111,
					Status:   domain.ArticleStatusUnPublished.ToUint8(),
				}, article)
			},
			expectCode: http.StatusOK,
			expectBody: Result[int64]{
				Msg:  "编辑成功～",
				Data: 2,
			},
		},
		{
			name: "修改别人的帖子失败",
			ctx:  context.Background(),
			article: Article{
				Id:      3,
				Title:   "我的标题111",
				Content: "我的内容111",
			},
			before: func(t *testing.T) {
				//模拟已经存在的数据
				err := s.db.Create(&newDAO.Article{
					Id:      3,
					Title:   "我的标题",
					Content: "我的内容",
					//测试模拟的用户ID是21，帖子作者ID是1，即正在修改别人的帖子
					AuthorId: 1,
					Ctime:    111,
					Utime:    222,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}).Error
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				//验证数据库数据
				var article newDAO.Article
				err := s.db.Where("id = ?", 3).First(&article).Error
				assert.NoError(t, err)
				assert.Equal(t, newDAO.Article{
					Id:       3,
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 1,
					Ctime:    111,
					Utime:    222,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}, article)
			},
			// 当前接口使用统一业务响应：业务失败由 Result.Code 表示，HTTP 状态仍为 200。
			expectCode: http.StatusOK,
			expectBody: Result[int64]{
				Code: 5,
				Msg:  "系统错误！",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// testify 的 SetupTest/TearDownTest 只包裹 TestEdit，不会逐个包裹内部的
			// t.Run，因此每个表驱动子用例都要独立清理，避免主键和数据相互污染。
			s.cleanArticles(t)
			t.Cleanup(func() {
				s.cleanArticles(t)
			})

			//准备数据
			tc.before(t)

			//构造http请求：
			reqBody, err := json.Marshal(tc.article)
			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/articles/edit", bytes.NewBuffer([]byte(reqBody)))
			require.NoError(t, err)
			//设置请求头
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			t.Log(req)

			//创建响应记录器，处理响应：
			resp := httptest.NewRecorder()
			t.Log(resp)

			//http请求进入gin框架的入口
			s.server.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectCode, resp.Code)

			var webRes Result[int64]
			//反序列化到web.Result结构体
			err = json.NewDecoder(resp.Body).Decode(&webRes)
			require.NoError(t, err)
			assert.Equal(t, tc.expectBody, webRes)

			//验证redis状态或清理
			tc.after(t)
		})
	}
}

type Article struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
type Result[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}
