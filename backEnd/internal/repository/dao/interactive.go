package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrInteractiveNotFound 由仓储层识别，并转换成计数均为 0 的领域对象。
	// 新文章尚未产生互动行为时没有聚合记录，这是正常状态而不是接口错误。
	ErrInteractiveNotFound = gorm.ErrRecordNotFound
	// ErrInvalidInteractiveBatch 表示批量阅读事件中的业务类型与业务 ID 无法一一对应。
	ErrInvalidInteractiveBatch = errors.New("互动批量参数长度不一致")
)

// InteractiveDAO 互动数据访问对象接口，定义互动数据的数据库操作
//
//go:generate mockgen -source=./interactive.go -package=daomocks -destination=./mocks/interactive.mock.go InteractiveDAO
type InteractiveDAO interface {
	// IncrReadCnt 增加阅读数（Upsert + 原子递增）
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// InsertLikeInfo 点赞；changed 仅在点赞状态从未点赞变为已点赞时为 true。
	InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) (changed bool, err error)
	// DeleteLikeInfo 取消点赞；changed 仅在点赞状态从已点赞变为未点赞时为 true。
	DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) (changed bool, err error)
	// Get 查询互动数据
	Get(ctx context.Context, biz string, id int64) (Interactive, error)
	// Liked 查询指定用户当前是否已点赞。
	Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	// Collected 查询指定用户是否已收藏。
	Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	// BatchIncrReadCnt 批量增加阅读数（事务内循环执行）
	BatchIncrReadCnt(ctx context.Context, bizs []string, ids []int64) error
	GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error)
	// InsertCollectBiz 在事务内保存收藏关系；changed 仅在首次收藏时为 true。
	InsertCollectBiz(ctx context.Context, cb UserCollectionBiz) (changed bool, err error)
}

type interactiveDAO struct {
	db *gorm.DB
}

// InsertCollectBiz 在一个事务中完成收藏操作的两次数据库写入：
//  1. 向 UserCollectionBiz 写入“用户—业务对象—收藏夹”的明细关系。
//  2. Upsert Interactive；记录已存在时 collect_cnt 原子加 1，否则以 1 创建。
//
// UserCollectionBiz 上的联合唯一索引会阻止同一用户重复收藏同一个业务对象。
// 重复收藏按幂等成功处理，既不返回唯一键错误，也不会重复增加聚合收藏数。
func (idao *interactiveDAO) InsertCollectBiz(ctx context.Context, cb UserCollectionBiz) (bool, error) {
	now := time.Now().UnixMilli()
	cb.Utime = now
	cb.Ctime = now
	changed := false
	err := idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 联合唯一索引负责并发防重；唯一键冲突时 DoNothing 返回 RowsAffected=0。
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&cb)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		changed = true
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
	if err != nil {
		return false, err
	}
	return changed, nil
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
	if len(bizs) != len(ids) {
		return fmt.Errorf("%w: bizs=%d, ids=%d", ErrInvalidInteractiveBatch, len(bizs), len(ids))
	}
	return idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := NewInteractiveDAO(tx)
		for i := range bizs {
			err := txDAO.IncrReadCnt(ctx, bizs[i], ids[i])
			if err != nil {
				// 任意一条更新失败都必须返回错误，让整个批次回滚，避免部分提交后仍确认消息。
				return err
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

// Liked 查询有效的点赞关系。唯一索引保证结果最多一条，Count 不会把“未点赞”当成错误。
func (idao *interactiveDAO) Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	var cnt int64
	err := idao.db.WithContext(ctx).
		Model(&UserLikeBiz{}).
		Where("uid = ? AND biz_id = ? AND biz = ? AND status = ?", uid, id, biz, 1).
		Count(&cnt).Error
	return cnt > 0, err
}

// Collected 查询收藏关系是否存在。当前收藏关系没有软删除状态，存在记录即表示已收藏。
func (idao *interactiveDAO) Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	var cnt int64
	err := idao.db.WithContext(ctx).
		Model(&UserCollectionBiz{}).
		Where("uid = ? AND biz_id = ? AND biz = ?", uid, id, biz).
		Count(&cnt).Error
	return cnt > 0, err
}

// DeleteLikeInfo 仅把有效点赞切换为已取消，并在状态确实变化时递减聚合计数。
// 重复取消或从未点赞都直接返回 changed=false，避免数据库和 Redis 计数被重复扣减。
func (idao *interactiveDAO) DeleteLikeInfo(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	now := time.Now().UnixMilli()
	changed := false
	err := idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&UserLikeBiz{}).
			Where("uid = ? AND biz_id = ? AND biz = ? AND status = ?", uid, id, biz, 1).
			Updates(map[string]interface{}{
				"utime":  now,
				"status": 0,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		changed = true
		return tx.Model(&Interactive{}).
			Where("biz = ? AND biz_id = ?", biz, id).
			Updates(map[string]interface{}{
				// 数据异常时也不允许计数降为负数。
				"like_cnt": gorm.Expr("GREATEST(`like_cnt` - 1, 0)"),
				"utime":    now,
			}).Error
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// InsertLikeInfo 点赞
// 在事务内先执行幂等插入；唯一索引冲突时，再尝试把 status=0 的旧记录恢复为 1。
// 只有“首次插入成功”或“恢复成功”才增加聚合计数并返回 changed=true。
//
// 这种写法不再把 MySQL 的 ON DUPLICATE KEY UPDATE RowsAffected=2 误判为首次点赞，
// 因而同时覆盖首次点赞、重复点赞、取消后重新点赞以及并发重复请求。
func (idao *interactiveDAO) InsertLikeInfo(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	now := time.Now().UnixMilli()
	changed := false
	err := idao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先尝试首次插入；并发重复请求由联合唯一索引收敛为一次成功插入。
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&UserLikeBiz{
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

		stateChanged := res.RowsAffected == 1
		if !stateChanged {
			// 唯一键已存在时只恢复 status=0 的记录；有效点赞不会被重复更新。
			res = tx.Model(&UserLikeBiz{}).
				Where("uid = ? AND biz_id = ? AND biz = ? AND status = ?", uid, id, biz, 0).
				Updates(map[string]interface{}{
					"status": 1,
					"utime":  now,
				})
			if res.Error != nil {
				return res.Error
			}
			stateChanged = res.RowsAffected == 1
		}

		if !stateChanged {
			return nil
		}

		changed = true
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
	if err != nil {
		return false, err
	}
	return changed, nil
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
