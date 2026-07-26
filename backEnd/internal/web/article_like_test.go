package web

import (
	svcmocks "GoBook/internal/service/mocks"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/logger"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestArticleHandler_LikeUsesRequestedState(t *testing.T) {
	testCases := []struct {
		name    string
		like    *bool
		prepare func(*svcmocks.MockInteractiveService)
		wantMsg string
	}{
		{
			name: "点赞",
			like: boolPointer(true),
			prepare: func(svc *svcmocks.MockInteractiveService) {
				svc.EXPECT().
					Like(gomock.Any(), "article", int64(1), int64(2)).
					Return(nil)
			},
			wantMsg: "点赞成功～",
		},
		{
			name: "取消点赞",
			like: boolPointer(false),
			prepare: func(svc *svcmocks.MockInteractiveService) {
				svc.EXPECT().
					CancelLike(gomock.Any(), "article", int64(1), int64(2)).
					Return(nil)
			},
			wantMsg: "取消点赞成功～",
		},
		{
			name: "旧请求未传like时默认点赞",
			prepare: func(svc *svcmocks.MockInteractiveService) {
				svc.EXPECT().
					Like(gomock.Any(), "article", int64(1), int64(2)).
					Return(nil)
			},
			wantMsg: "点赞成功～",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			interactiveSvc := svcmocks.NewMockInteractiveService(ctrl)
			handler := NewArticleHandler(
				svcmocks.NewMockArticleService(ctrl),
				&logger.NopLogger{},
				interactiveSvc,
			)
			tc.prepare(interactiveSvc)
			ctx, _ := gin.CreateTestContext(nil)

			got, err := handler.Like(ctx, ArticleLikeReq{
				Id:   1,
				Like: tc.like,
			}, &ijwt.UserClaims{Uid: 2})

			require.NoError(t, err)
			assert.Equal(t, Result{
				Code: 0,
				Msg:  tc.wantMsg,
			}, got)
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}
