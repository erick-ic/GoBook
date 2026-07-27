package repository

import (
	"context"
	"errors"
	"testing"

	"GoBook/interactive/domain"
	"GoBook/interactive/repository/dao"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveRepositoryCacheFollowsStateChange(t *testing.T) {
	testCases := []struct {
		name              string
		prepare           func(*stubInteractiveDAO)
		action            func(InteractiveRepository) error
		wantLikeIncr      int
		wantLikeDecr      int
		wantCollectionInc int
	}{
		{
			name: "首次点赞更新缓存",
			prepare: func(dao *stubInteractiveDAO) {
				dao.likeChanged = true
			},
			action: func(repo InteractiveRepository) error {
				return repo.IncrLike(context.Background(), "article", 1, 2)
			},
			wantLikeIncr: 1,
		},
		{
			name:    "重复点赞不更新缓存",
			prepare: func(*stubInteractiveDAO) {},
			action: func(repo InteractiveRepository) error {
				return repo.IncrLike(context.Background(), "article", 1, 2)
			},
		},
		{
			name: "首次取消更新缓存",
			prepare: func(dao *stubInteractiveDAO) {
				dao.cancelLikeChanged = true
			},
			action: func(repo InteractiveRepository) error {
				return repo.DecrLike(context.Background(), "article", 1, 2)
			},
			wantLikeDecr: 1,
		},
		{
			name:    "重复取消不更新缓存",
			prepare: func(*stubInteractiveDAO) {},
			action: func(repo InteractiveRepository) error {
				return repo.DecrLike(context.Background(), "article", 1, 2)
			},
		},
		{
			name: "首次收藏更新缓存",
			prepare: func(dao *stubInteractiveDAO) {
				dao.collectionChanged = true
			},
			action: func(repo InteractiveRepository) error {
				return repo.AddCollectItem(context.Background(), "article", 1, 3, 2)
			},
			wantCollectionInc: 1,
		},
		{
			name:    "重复收藏不更新缓存",
			prepare: func(*stubInteractiveDAO) {},
			action: func(repo InteractiveRepository) error {
				return repo.AddCollectItem(context.Background(), "article", 1, 3, 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interactiveDAO := &stubInteractiveDAO{}
			interactiveCache := &stubInteractiveCache{}
			tc.prepare(interactiveDAO)
			repo := NewInteractiveRepository(interactiveDAO, interactiveCache)

			err := tc.action(repo)

			require.NoError(t, err)
			assert.Equal(t, tc.wantLikeIncr, interactiveCache.likeIncrements)
			assert.Equal(t, tc.wantLikeDecr, interactiveCache.likeDecrements)
			assert.Equal(t, tc.wantCollectionInc, interactiveCache.collectionIncrements)
		})
	}
}

func TestInteractiveRepositoryGetWithoutAggregate(t *testing.T) {
	repo := NewInteractiveRepository(
		&stubInteractiveDAO{getErr: dao.ErrInteractiveNotFound},
		&stubInteractiveCache{},
	)

	got, err := repo.Get(context.Background(), "article", 1)

	require.NoError(t, err)
	assert.Equal(t, domain.Interactive{
		Biz:   "article",
		BizId: 1,
	}, got)
}

func TestInteractiveRepositoryDAOErrorDoesNotTouchCache(t *testing.T) {
	wantErr := errors.New("database error")
	interactiveCache := &stubInteractiveCache{}
	repo := NewInteractiveRepository(
		&stubInteractiveDAO{likeErr: wantErr},
		interactiveCache,
	)

	err := repo.IncrLike(context.Background(), "article", 1, 2)

	assert.ErrorIs(t, err, wantErr)
	assert.Zero(t, interactiveCache.likeIncrements)
}

type stubInteractiveDAO struct {
	likeChanged       bool
	cancelLikeChanged bool
	collectionChanged bool
	likeErr           error
	getErr            error
}

func (s *stubInteractiveDAO) IncrReadCnt(context.Context, string, int64) error {
	return nil
}

func (s *stubInteractiveDAO) InsertLikeInfo(context.Context, string, int64, int64) (bool, error) {
	return s.likeChanged, s.likeErr
}

func (s *stubInteractiveDAO) DeleteLikeInfo(context.Context, string, int64, int64) (bool, error) {
	return s.cancelLikeChanged, nil
}

func (s *stubInteractiveDAO) Get(context.Context, string, int64) (dao.Interactive, error) {
	return dao.Interactive{}, s.getErr
}

func (s *stubInteractiveDAO) Liked(context.Context, string, int64, int64) (bool, error) {
	return false, nil
}

func (s *stubInteractiveDAO) Collected(context.Context, string, int64, int64) (bool, error) {
	return false, nil
}

func (s *stubInteractiveDAO) BatchIncrReadCnt(context.Context, []string, []int64) error {
	return nil
}

func (s *stubInteractiveDAO) GetByIds(context.Context, string, []int64) ([]dao.Interactive, error) {
	return nil, nil
}

func (s *stubInteractiveDAO) InsertCollectBiz(context.Context, dao.UserCollectionBiz) (bool, error) {
	return s.collectionChanged, nil
}

type stubInteractiveCache struct {
	likeIncrements       int
	likeDecrements       int
	collectionIncrements int
}

func (s *stubInteractiveCache) IncrReadCntIfPresent(context.Context, string, int64) error {
	return nil
}

func (s *stubInteractiveCache) IncrLikeCntIfPresent(context.Context, string, int64) error {
	s.likeIncrements++
	return nil
}

func (s *stubInteractiveCache) IncrCollectCntIfPresent(context.Context, string, int64) error {
	s.collectionIncrements++
	return nil
}

func (s *stubInteractiveCache) DecrLikeCntIfPresent(context.Context, string, int64) error {
	s.likeDecrements++
	return nil
}
