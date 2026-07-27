package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestInteractiveDAOInsertLikeInfo(t *testing.T) {
	testCases := []struct {
		name        string
		prepare     func(sqlmock.Sqlmock)
		wantChanged bool
	}{
		{
			name: "首次点赞",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `interactives`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantChanged: true,
		},
		{
			name: "重复点赞",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
		},
		{
			name: "取消后重新点赞",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `interactives`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantChanged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interactiveDAO, mock := newInteractiveDAOForTest(t)
			tc.prepare(mock)

			changed, err := interactiveDAO.InsertLikeInfo(context.Background(), "article", 1, 2)

			require.NoError(t, err)
			assert.Equal(t, tc.wantChanged, changed)
		})
	}
}

func TestInteractiveDAOInsertCollectBiz(t *testing.T) {
	testCases := []struct {
		name        string
		prepare     func(sqlmock.Sqlmock)
		wantChanged bool
	}{
		{
			name: "首次收藏",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `user_collection_bizs`") + ".*ON DUPLICATE KEY UPDATE").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `interactives`")).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantChanged: true,
		},
		{
			name: "重复收藏",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `user_collection_bizs`") + ".*ON DUPLICATE KEY UPDATE").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interactiveDAO, mock := newInteractiveDAOForTest(t)
			tc.prepare(mock)

			changed, err := interactiveDAO.InsertCollectBiz(context.Background(), UserCollectionBiz{
				Uid:   1,
				Biz:   "article",
				BizId: 1,
				Cid:   1,
			})

			require.NoError(t, err)
			assert.Equal(t, tc.wantChanged, changed)
		})
	}
}

func TestInteractiveDAODeleteLikeInfo(t *testing.T) {
	testCases := []struct {
		name        string
		prepare     func(sqlmock.Sqlmock)
		wantChanged bool
	}{
		{
			name: "取消有效点赞",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `interactives`")).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantChanged: true,
		},
		{
			name: "重复取消",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_like_bizs`")).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interactiveDAO, mock := newInteractiveDAOForTest(t)
			tc.prepare(mock)

			changed, err := interactiveDAO.DeleteLikeInfo(context.Background(), "article", 1, 2)

			require.NoError(t, err)
			assert.Equal(t, tc.wantChanged, changed)
		})
	}
}

func TestInteractiveDAOBatchIncrReadCnt(t *testing.T) {
	t.Run("参数长度不一致", func(t *testing.T) {
		interactiveDAO, _ := newInteractiveDAOForTest(t)

		err := interactiveDAO.BatchIncrReadCnt(
			context.Background(),
			[]string{"article", "article"},
			[]int64{1},
		)

		assert.ErrorIs(t, err, ErrInvalidInteractiveBatch)
	})

	t.Run("中途失败回滚整个批次", func(t *testing.T) {
		interactiveDAO, mock := newInteractiveDAOForTest(t)
		wantErr := errors.New("database error")
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `interactives`")).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `interactives`")).
			WillReturnError(wantErr)
		mock.ExpectRollback()

		err := interactiveDAO.BatchIncrReadCnt(
			context.Background(),
			[]string{"article", "article"},
			[]int64{1, 2},
		)

		assert.ErrorIs(t, err, wantErr)
	})
}

func newInteractiveDAOForTest(t *testing.T) (InteractiveDAO, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, sqlDB.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	db, err := gorm.Open(gormMysql.New(gormMysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	return NewInteractiveDAO(db), mock
}
