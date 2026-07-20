package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"context"
)

// InteractiveRepository 互动仓储接口，定义互动数据访问操作
// 调用链路：InteractiveService → InteractiveRepository → DAO + Cache
//
// 核心职责：
//  1. 领域模型 ↔ DAO 实体的转换
//  2. 数据库和缓存的协同（先写库，再更新缓存）
//  3. 保证数据最终一致性（数据库优先，缓存按需更新）
type InteractiveRepository interface {
	// IncrReadCnt 增加阅读数（先写库，再按需更新缓存）
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// BatchIncrReadCnt 批量增加阅读数（用于 Kafka 批量消费）
	BatchIncrReadCnt(ctx context.Context, biz []string, bizId []int64) error
	// IncrLike 点赞（先写库，再按需更新缓存）
	IncrLike(ctx context.Context, biz string, id int64, uid int64) error
	// DecrLike 取消点赞
	DecrLike(ctx context.Context, biz string, id int64, uid int64) error
	// Get 查询互动数据（阅读数/点赞数/收藏数）
	Get(ctx context.Context, biz string, id int64) (domain.Interactive, error)
}

// interactiveRepository 互动仓储实现类
type interactiveRepository struct {
	dao   dao.InteractiveDAO     // 互动DAO，操作数据库
	cache cache.InteractiveCache // 互动缓存，操作 Redis
}

// Get 查询互动数据
// 调用链路：Service.Get → Repository.Get → DAO.Get → 数据库
// 注意：当前直接查数据库，未走缓存（后续可优化为先查缓存）
func (ir *interactiveRepository) Get(ctx context.Context, biz string, id int64) (domain.Interactive, error) {
	inter, err := ir.dao.Get(ctx, biz, id)
	if err != nil {
		return domain.Interactive{}, err
	}
	res := ir.toDomain(inter)
	return res, nil
}

// IncrLike 点赞
// 调用链路：Service.Like → Repository.IncrLike → DAO.InsertLikeInfo + Cache.IncrLikeCntIfPresent
//
// 执行流程：
//  1. 写入 UserLikeBiz 表（记录用户点赞行为，幂等）
//  2. 更新 Interactive 表的 like_cnt（原子递增）
//  3. 缓存中 like_cnt +1（仅当缓存存在时才更新，避免缓存击穿）
func (ir *interactiveRepository) IncrLike(ctx context.Context, biz string, id int64, uid int64) error {
	err := ir.dao.InsertLikeInfo(ctx, biz, id, uid)
	if err != nil {
		return err
	}
	return ir.cache.IncrLikeCntIfPresent(ctx, biz, id)
}

// DecrLike 取消点赞
// 调用链路：Service.CancelLike → Repository.DecrLike → DAO.DeleteLikeInfo
// TODO: 当前未更新缓存的 like_cnt，可能导致缓存与数据库不一致
func (ir *interactiveRepository) DecrLike(ctx context.Context, biz string, id int64, uid int64) error {
	err := ir.dao.DeleteLikeInfo(ctx, biz, id, uid)
	if err != nil {
		return err
	}
	return nil
}

// IncrReadCnt 增加阅读数
// 调用链路：Kafka 消费者 → Service.IncrReadCnt → Repository.IncrReadCnt → DAO + Cache
//
// 缓存策略：Cache-If-Present
//  1. 先更新数据库（保证数据准确性）
//  2. 再更新缓存（仅当 key 存在时才 HINCRBY，避免缓存击穿）
//  3. 如果缓存不存在，不主动创建（下次查询时由其他逻辑填充）
func (ir *interactiveRepository) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	err := ir.dao.IncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	return ir.cache.IncrReadCntIfPresent(ctx, biz, bizId)
}

// BatchIncrReadCnt 批量增加阅读数
// 调用链路：Kafka 批量消费者 → Service.BatchIncrReadCnt → Repository.BatchIncrReadCnt → DAO.BatchIncrReadCnt
// 用于批量消费场景，通过事务减少磁盘 IO 次数，提升吞吐量
func (ir *interactiveRepository) BatchIncrReadCnt(ctx context.Context, biz []string, bizId []int64) error {
	err := ir.dao.BatchIncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	return nil
}

// NewInteractiveRepository 创建互动仓储实例
func NewInteractiveRepository(dao dao.InteractiveDAO, cache cache.InteractiveCache) InteractiveRepository {
	return &interactiveRepository{
		dao:   dao,
		cache: cache,
	}
}

// toDomain 将DAO实体转换为领域模型
func (ir *interactiveRepository) toDomain(ie dao.Interactive) domain.Interactive {
	return domain.Interactive{
		ReadCnt:    ie.ReadCnt,
		LikeCnt:    ie.LikeCnt,
		CollectCnt: ie.CollectCnt,
	}
}
