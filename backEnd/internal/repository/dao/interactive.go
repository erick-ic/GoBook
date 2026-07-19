package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InteractiveDAO interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) error
	DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) error
	Get(ctx context.Context, biz string, id int64) (Interactive, error)
	BatchIncrReadCnt(ctx context.Context, bizs []string, ids []int64) error
}
type interactiveDAO struct {
	db *gorm.DB
}

func (idao *interactiveDAO) BatchIncrReadCnt(ctx context.Context, bizs []string, ids []int64) error {
	//同样都是10条消息，为什么批量消费比较单次消费快？
	//1.批量消费开启一个事务，磁盘操作只执行一次
	//2.刷新redolog、undolog、binlog到磁盘，批量消费远少于单次消费
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := NewInteractiveDAO(tx)
		for i := range bizs {
			err := txDAO.IncrReadCnt(ctx, bizs[i], ids[i])
			if err != nil {
				//记入日志
			}
		}
		return nil
	})
}

func (idao *interactiveDAO) Get(ctx context.Context, biz string, id int64) (Interactive, error) {
	var inter Interactive
	err := idao.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ?", biz, id).
		First(&inter).Error
	return inter, err
}

func (idao *interactiveDAO) DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) error {
	//TODO implement me
	panic("implement me")
}

func (idao *interactiveDAO) InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"utime":  now,
				"status": 1,
			}),
		}).Create(&UserLikeBiz{
			Uid:    uid,
			Biz:    biz,
			BizId:  id,
			Status: 1,
			Ctime:  now,
			Utime:  now,
		}).Error
		if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"like_cnt": gorm.Expr("`like_cnt` + 1"),
				"utime":    now,
			}),
		}).Create(&Interactive{
			Biz:     biz,
			BizId:   id,
			LikeCnt: 1,
			Ctime:   now,
			Utime:   now,
		}).Error
	})
}

func (idao *interactiveDAO) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	now := time.Now().UnixMilli()
	return idao.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]interface{}{
			"read_cnt": gorm.Expr("`read_cnt` + 1"),
			"utime":    now,
		}),
	}).Create(&Interactive{
		BizId:   bizId,
		Biz:     biz,
		ReadCnt: 1,
		Ctime:   now,
		Utime:   now,
	}).Error
}

func NewInteractiveDAO(db *gorm.DB) InteractiveDAO {
	return &interactiveDAO{
		db: db,
	}
}

type Interactive struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	//业务标识
	//在bizId,bizName上创建联合索引，区分度更高
	BizId int64  `gorm:"uniqueIndex:biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:biz_type_id"`

	ReadCnt    int64
	LikeCnt    int64
	CollectCnt int64
	Ctime      int64
	Utime      int64
}

type UserLikeBiz struct {
	Id    int64  `gorm:"primaryKey,autoIncrement"`
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"`

	Status uint8 //软删除，表示存储状态，0 删除，1 有效
	Ctime  int64
	Utime  int64
}
