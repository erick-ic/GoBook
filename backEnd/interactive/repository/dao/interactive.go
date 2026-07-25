package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InteractiveDAO 互动数据访问对象接口，定义互动数据的数据库操作
type InteractiveDAO interface {
	// IncrReadCnt 增加阅读数（Upsert + 原子递增）
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// InsertLikeInfo 点赞（事务内写入点赞记录 + 更新点赞数）
	InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) error
	// DeleteLikeInfo 取消点赞（预留接口，尚未实现）
	DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) error
	// Get 查询互动数据
	Get(ctx context.Context, biz string, id int64) (Interactive, error)
	// BatchIncrReadCnt 批量增加阅读数（事务内循环执行）
	BatchIncrReadCnt(ctx context.Context, bizs []string, ids []int64) error
	GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error)
	// InsertCollectBiz 在事务内保存用户收藏关系并增加业务对象的聚合收藏数。
	InsertCollectBiz(ctx context.Context, cb UserCollectionBiz) error
}

type interactiveDAO struct {
	db *gorm.DB
}

// InsertCollectBiz 在一个事务中完成收藏操作的两次数据库写入：
//  1. 向 UserCollectionBiz 写入“用户—业务对象—收藏夹”的明细关系。
//  2. Upsert Interactive；记录已存在时 collect_cnt 原子加 1，否则以 1 创建。
//
// UserCollectionBiz 上的联合唯一索引会阻止同一用户重复收藏同一个业务对象；
// 当前实现遇到重复收藏时会直接返回唯一键冲突错误，不会重复增加收藏数。
func (idao *interactiveDAO) InsertCollectBiz(ctx context.Context, cb UserCollectionBiz) error {
	now := time.Now().UnixMilli()
	cb.Utime = now
	cb.Ctime = now
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先保存用户收藏明细；失败时终止事务，聚合收藏数不会变化。
		err := tx.Create(&cb).Error
		if err != nil {
			return err
		}

		// 再按 biz + biz_id 更新聚合数据，供文章详情等读接口统一查询。
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"collect_cnt": gorm.Expr("`collect_cnt` + 1"),
				"utime":       now,
			}),
		}).Create(&Interactive{
			Biz:        cb.Biz,
			BizId:      cb.BizId,
			CollectCnt: 1,
			Ctime:      now,
			Utime:      now,
		}).Error
	})
}

func (idao *interactiveDAO) GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error) {
	var inters []Interactive
	err := idao.db.WithContext(ctx).
		Where("biz = ? AND biz_id IN ?", biz, ids).
		Find(&inters).Error
	return inters, err
}

// BatchIncrReadCnt 批量增加阅读数
// 批量消费比单次消费快的原因：
//  1. 批量消费开启一个事务，磁盘操作只执行一次（事务提交时才刷盘）
//  2. 刷新 redolog、undolog、binlog 到磁盘的次数远少于单次消费
func (idao *interactiveDAO) BatchIncrReadCnt(ctx context.Context, bizs []string, ids []int64) error {
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

// Get 查询互动数据
// 通过 biz + biz_id 唯一索引查询
func (idao *interactiveDAO) Get(ctx context.Context, biz string, id int64) (Interactive, error) {
	var inter Interactive
	err := idao.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ?", biz, id).
		First(&inter).Error
	return inter, err
}

// DeleteLikeInfo 取消点赞（预留接口，尚未实现）
func (idao *interactiveDAO) DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&UserLikeBiz{}).
			Where("uid=? AND biz_id = ? AND biz=?", uid, id, biz).
			Updates(map[string]interface{}{
				"utime":  now,
				"status": 0,
			}).Error
		if err != nil {
			return err
		}
		return tx.Model(&Interactive{}).
			Where("biz =? AND biz_id=?", biz, id).
			Updates(map[string]interface{}{
				"like_cnt": gorm.Expr("`like_cnt` - 1"),
				"utime":    now,
			}).Error
	})
}

// InsertLikeInfo 点赞
// 在一个事务内完成两步操作，保证数据一致性：
//  1. 写入 UserLikeBiz 表（记录用户的点赞行为）
//     - 使用 OnConflict 实现幂等：重复点赞只更新 utime 和 status，不会插入新记录
//  2. 更新 Interactive 表的 like_cnt（原子递增）
//     - 使用 OnConflict + gorm.Expr 实现：存在则 like_cnt+1，不存在则插入 like_cnt=1
//
// 防重复点赞原理：
//   - 通过 RowsAffected 判断是 INSERT 还是 UPDATE
//   - RowsAffected=1：新插入记录（首次点赞），like_cnt +1
//   - RowsAffected=0：触发 ON CONFLICT UPDATE（已点赞过），不增加 like_cnt
//   - 配合 UserLikeBiz.status 字段，支持"取消点赞后再点赞"的场景
func (idao *interactiveDAO) InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 步骤1：写入用户点赞记录（幂等，通过唯一索引 uid_biz_type_id 防重）
		// ON DUPLICATE KEY UPDATE 时，RowsAffected 的取值：
		//   - 1：新插入记录（INSERT）
		//   - 2：更新了记录（UPDATE，MySQL 把 INSERT 算作 0 行，UPDATE 算作 2 行）
		//   - 0：触发 ON CONFLICT 但字段值没变化（MySQL 特殊行为）
		// 这里用 ==1 判断是否为真正的新插入
		res := tx.Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"utime":  now,
				"status": 1, // 1=有效，0=已取消
			}),
		}).Create(&UserLikeBiz{
			Uid:    uid,
			Biz:    biz,
			BizId:  id,
			Status: 1,
			Ctime:  now,
			Utime:  now,
		})
		if res.Error != nil {
			return res.Error
		}

		// 只有新插入（首次点赞）才更新互动表的点赞数
		// 重复点赞（RowsAffected != 1）不增加 like_cnt，避免点赞数虚增
		if res.RowsAffected != 1 {
			return nil
		}

		// 步骤2：更新互动表的点赞数（原子递增）
		// gorm.Expr("`like_cnt` + 1") 等价于 SQL: like_cnt = like_cnt + 1
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

// IncrReadCnt 增加阅读数
// 使用 Upsert 模式：存在则原子递增 read_cnt，不存在则插入 read_cnt=1
// gorm.Expr("`read_cnt` + 1") 等价于 SQL: read_cnt = read_cnt + 1
// 此方法会被 Kafka 消费者调用（异步更新阅读数）
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

// NewInteractiveDAO 创建互动DAO实例
func NewInteractiveDAO(db *gorm.DB) InteractiveDAO {
	return &interactiveDAO{
		db: db,
	}
}

// Interactive 互动数据表实体
// 记录某个业务实体的聚合互动数据（阅读数/点赞数/收藏数）
type Interactive struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 业务标识：通过 biz + biz_id 唯一标识一个业务实体的互动数据
	// 在 biz_id, biz 上创建联合唯一索引，区分度高，支持多业务复用
	BizId int64  `gorm:"uniqueIndex:biz_type_id"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:biz_type_id"`

	ReadCnt    int64 // 阅读数
	LikeCnt    int64 // 点赞数
	CollectCnt int64 // 收藏数
	Ctime      int64 // 创建时间（毫秒时间戳）
	Utime      int64 // 更新时间（毫秒时间戳）
}

// UserLikeBiz 用户点赞记录表实体
// 记录用户对某个业务实体的点赞行为，用于：
//  1. 防止重复点赞（通过唯一索引 uid_biz_type_id）
//  2. 查询用户是否点赞过（用于前端展示"已点赞"状态）
//  3. 软删除：通过 status 字段标记取消点赞（0=已取消，1=有效）
type UserLikeBiz struct {
	Id    int64  `gorm:"primaryKey,autoIncrement"`
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`                   // 用户ID
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`                   // 业务实体ID
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"` // 业务标识

	Status uint8 // 状态：0=已取消，1=有效（软删除标记）
	Ctime  int64 // 创建时间（毫秒时间戳）
	Utime  int64 // 更新时间（毫秒时间戳）
}

// UserCollectionBiz 记录一条用户收藏明细。
// Interactive 保存文章维度的聚合收藏数，本表则用于回答“谁收藏了什么、放在哪个收藏夹”。
type UserCollectionBiz struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// uid + biz + biz_id 唯一，避免同一用户重复收藏同一业务对象。
	Uid   int64  `gorm:"uniqueIndex:uid_biz_type_id"`                   // 收藏用户 ID
	BizId int64  `gorm:"uniqueIndex:uid_biz_type_id"`                   // 被收藏的业务对象 ID，文章场景下为文章 ID
	Biz   string `gorm:"type:varchar(128);uniqueIndex:uid_biz_type_id"` // 业务类型，例如 article
	Cid   int64  `gorm:"index"`                                         // 收藏夹 ID，建立索引以支持按收藏夹查询
	Utime int64  // 更新时间（毫秒时间戳）
	Ctime int64  // 创建时间（毫秒时间戳）
}
