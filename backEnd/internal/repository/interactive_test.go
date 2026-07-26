package repository_test

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"GoBook/internal/repository/cache/mocks"
	"GoBook/internal/repository/dao"
	daomocks "GoBook/internal/repository/dao/mocks"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInteractiveRepository_GetWithoutAggregate(t *testing.T) {
	ctrl := gomock.NewController(t)
	interactiveDAO := daomocks.NewMockInteractiveDAO(ctrl)
	interactiveCache := cachemocks.NewMockInteractiveCache(ctrl)
	repo := repository.NewInteractiveRepository(interactiveDAO, interactiveCache)

	interactiveDAO.EXPECT().
		Get(gomock.Any(), "article", int64(1)).
		Return(dao.Interactive{}, dao.ErrInteractiveNotFound)

	got, err := repo.Get(context.Background(), "article", 1)

	require.NoError(t, err)
	assert.Equal(t, domain.Interactive{
		Biz:   "article",
		BizId: 1,
	}, got)
}

func TestInteractiveRepository_CollectCacheFollowsStateTransition(t *testing.T) {
	testCases := []struct {
		name    string
		changed bool
	}{
		{
			name:    "首次收藏更新缓存",
			changed: true,
		},
		{
			name: "重复收藏不更新缓存",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			interactiveDAO := daomocks.NewMockInteractiveDAO(ctrl)
			interactiveCache := cachemocks.NewMockInteractiveCache(ctrl)
			repo := repository.NewInteractiveRepository(interactiveDAO, interactiveCache)

			interactiveDAO.EXPECT().
				InsertCollectBiz(gomock.Any(), dao.UserCollectionBiz{
					Uid:   2,
					Biz:   "article",
					BizId: 1,
					Cid:   3,
				}).
				Return(tc.changed, nil)
			if tc.changed {
				interactiveCache.EXPECT().
					IncrCollectCntIfPresent(gomock.Any(), "article", int64(1)).
					Return(nil)
			}

			err := repo.AddCollectItem(context.Background(), "article", 1, 3, 2)
			require.NoError(t, err)
		})
	}
}

func TestInteractiveRepository_LikeCacheFollowsStateTransition(t *testing.T) {
	testCases := []struct {
		name  string
		like  bool
		setup func(*daomocks.MockInteractiveDAO, *cachemocks.MockInteractiveCache)
	}{
		{
			name: "首次点赞更新缓存",
			like: true,
			setup: func(interactiveDAO *daomocks.MockInteractiveDAO, interactiveCache *cachemocks.MockInteractiveCache) {
				interactiveDAO.EXPECT().
					InsertLikeInfo(gomock.Any(), "article", int64(1), int64(2)).
					Return(true, nil)
				interactiveCache.EXPECT().
					IncrLikeCntIfPresent(gomock.Any(), "article", int64(1)).
					Return(nil)
			},
		},
		{
			name: "重复点赞不更新缓存",
			like: true,
			setup: func(interactiveDAO *daomocks.MockInteractiveDAO, _ *cachemocks.MockInteractiveCache) {
				interactiveDAO.EXPECT().
					InsertLikeInfo(gomock.Any(), "article", int64(1), int64(2)).
					Return(false, nil)
			},
		},
		{
			name: "首次取消更新缓存",
			setup: func(interactiveDAO *daomocks.MockInteractiveDAO, interactiveCache *cachemocks.MockInteractiveCache) {
				interactiveDAO.EXPECT().
					DeleteLikeInfo(gomock.Any(), "article", int64(1), int64(2)).
					Return(true, nil)
				interactiveCache.EXPECT().
					DecrLikeCntIfPresent(gomock.Any(), "article", int64(1)).
					Return(nil)
			},
		},
		{
			name: "重复取消不更新缓存",
			setup: func(interactiveDAO *daomocks.MockInteractiveDAO, _ *cachemocks.MockInteractiveCache) {
				interactiveDAO.EXPECT().
					DeleteLikeInfo(gomock.Any(), "article", int64(1), int64(2)).
					Return(false, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			interactiveDAO := daomocks.NewMockInteractiveDAO(ctrl)
			interactiveCache := cachemocks.NewMockInteractiveCache(ctrl)
			repo := repository.NewInteractiveRepository(interactiveDAO, interactiveCache)
			tc.setup(interactiveDAO, interactiveCache)

			var err error
			if tc.like {
				err = repo.IncrLike(context.Background(), "article", 1, 2)
			} else {
				err = repo.DecrLike(context.Background(), "article", 1, 2)
			}
			require.NoError(t, err)
		})
	}
}

func TestInteractiveRepository_LikeDAOErrorDoesNotTouchCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	interactiveDAO := daomocks.NewMockInteractiveDAO(ctrl)
	interactiveCache := cachemocks.NewMockInteractiveCache(ctrl)
	repo := repository.NewInteractiveRepository(interactiveDAO, interactiveCache)
	wantErr := errors.New("database error")

	interactiveDAO.EXPECT().
		InsertLikeInfo(gomock.Any(), "article", int64(1), int64(2)).
		Return(false, wantErr)

	err := repo.IncrLike(context.Background(), "article", 1, 2)
	assert.ErrorIs(t, err, wantErr)
}
