package web

import (
	"GoBook/internal/domain"
	svcmocks "GoBook/internal/service/mocks"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/logger"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestArticleHandler_PublishArticleDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	articleSvc := svcmocks.NewMockArticleService(ctrl)
	interactiveSvc := svcmocks.NewMockInteractiveService(ctrl)
	handler := NewArticleHandler(articleSvc, &logger.NopLogger{}, interactiveSvc)
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.Local)

	articleSvc.EXPECT().
		GetByPubId(gomock.Any(), int64(1), int64(2)).
		Return(domain.Article{
			Id:      1,
			Title:   "标题",
			Content: "内容",
			Status:  domain.ArticleStatusPublished,
			Ctime:   now,
			Utime:   now,
		}, nil)
	interactiveSvc.EXPECT().
		Get(gomock.Any(), "article", int64(1), int64(2)).
		Return(domain.Interactive{
			Biz:        "article",
			BizId:      1,
			ReadCnt:    10,
			LikeCnt:    20,
			CollectCnt: 30,
			Liked:      true,
			Collected:  true,
		}, nil)

	server := gin.New()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("claims", &ijwt.UserClaims{Uid: 2})
	})
	handler.RegisterRouters(server)

	req := httptest.NewRequest(http.MethodGet, "/pub/1", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var got struct {
		Code int       `json:"code"`
		Data ArticleVO `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, int64(1), got.Data.Id)
	assert.Equal(t, int64(10), got.Data.ReadCnt)
	assert.Equal(t, int64(20), got.Data.LikeCnt)
	assert.Equal(t, int64(30), got.Data.CollectCnt)
	assert.True(t, got.Data.Liked)
	assert.True(t, got.Data.Collected)
}

func TestArticleHandler_PublishArticleDetailRejectsInvalidClaimsWithoutPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler := NewArticleHandler(
		svcmocks.NewMockArticleService(ctrl),
		&logger.NopLogger{},
		svcmocks.NewMockInteractiveService(ctrl),
	)

	server := gin.New()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("claims", "invalid claims")
	})
	handler.RegisterRouters(server)

	req := httptest.NewRequest(http.MethodGet, "/pub/1", nil)
	resp := httptest.NewRecorder()

	require.NotPanics(t, func() {
		server.ServeHTTP(resp, req)
	})
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}
