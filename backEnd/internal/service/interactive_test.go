package service_test

import (
	"GoBook/internal/domain"
	repomocks "GoBook/internal/repository/mocks"
	"GoBook/internal/service"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInteractiveService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := repomocks.NewMockInteractiveRepository(ctrl)
	svc := service.NewInteractiveService(repo)

	repo.EXPECT().
		Get(gomock.Any(), "article", int64(1)).
		Return(domain.Interactive{
			Biz:        "article",
			BizId:      1,
			ReadCnt:    10,
			LikeCnt:    20,
			CollectCnt: 30,
		}, nil)
	repo.EXPECT().
		Liked(gomock.Any(), "article", int64(1), int64(2)).
		Return(true, nil)
	repo.EXPECT().
		Collected(gomock.Any(), "article", int64(1), int64(2)).
		Return(false, nil)

	got, err := svc.Get(context.Background(), "article", 1, 2)

	require.NoError(t, err)
	assert.Equal(t, domain.Interactive{
		Biz:        "article",
		BizId:      1,
		ReadCnt:    10,
		LikeCnt:    20,
		CollectCnt: 30,
		Liked:      true,
		Collected:  false,
	}, got)
}
