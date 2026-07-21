package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
)

// InteractiveService 互动服务接口，定义点赞/收藏/阅读数等互动业务操作
// 调用链路：HTTP Handler → InteractiveService → InteractiveRepository → DAO + Cache
//
// biz 字段说明：
//   - 通过 biz 字段区分不同业务（如 "article"、"comment" 等）
//   - 同一套互动服务支持多业务复用，避免为每个业务单独实现

//go:generate mockgen -source=./interactive.go -package=svcmocks -destination=./mocks/interactive.mock.go InteractiveService
type InteractiveService interface {
	// IncrReadCnt 增加阅读数（由 Kafka 消费者调用，异步更新）
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// Like 点赞（幂等，重复调用不会重复计数）
	Like(ctx context.Context, biz string, articleId int64, uid int64) error
	// CancelLike 取消点赞
	CancelLike(ctx context.Context, biz string, articleId int64, uid int64) error
	// Get 获取互动数据（阅读数/点赞数/收藏数）
	Get(ctx context.Context, biz string, id int64) (domain.Interactive, error)
	// GetByIds 批量获取互动数据（用于排行榜计算）
	GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error)
}

// interactiveService 互动服务实现类
type interactiveService struct {
	repo repository.InteractiveRepository
}

// GetByIds 批量获取互动数据（预留接口，尚未实现）
// 用于排行榜服务批量查询多篇文章的点赞数
func (is *interactiveService) GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error) {
	//TODO implement me
	panic("implement me")
}

// Get 获取互动数据
// 调用链路：PubDetail Handler → Get → Repository.Get → DAO.Get
// 注意：当前直接查数据库，未走缓存（后续可优化为先查缓存）
func (is *interactiveService) Get(ctx context.Context, biz string, id int64) (domain.Interactive, error) {
	inter, err := is.repo.Get(ctx, biz, id)
	if err != nil {
		return domain.Interactive{}, err
	}
	return inter, nil
}

// Like 点赞
// 调用链路：Like Handler → Like → Repository.IncrLike → DAO.InsertLikeInfo + Cache.IncrLikeCntIfPresent
// 实现幂等：通过 UserLikeBiz 表的唯一索引 + OnConflict 实现重复点赞不计数
func (is *interactiveService) Like(ctx context.Context, biz string, articleId int64, uid int64) error {
	return is.repo.IncrLike(ctx, biz, articleId, uid)
}

// CancelLike 取消点赞
// 调用链路：CancelLike Handler → CancelLike → Repository.DecrLike → DAO.DeleteLikeInfo
func (is *interactiveService) CancelLike(ctx context.Context, biz string, articleId int64, uid int64) error {
	return is.repo.DecrLike(ctx, biz, articleId, uid)
}

// IncrReadCnt 增加阅读数
// 调用链路：Kafka 消费者 → IncrReadCnt → Repository.IncrReadCnt → DAO.IncrReadCnt + Cache.IncrReadCntIfPresent
// 注意：此方法由 Kafka 消费者异步调用，不在用户请求链路中
func (is *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return is.repo.IncrReadCnt(ctx, biz, bizId)
}

// NewInteractiveService 创建互动服务实例
func NewInteractiveService(repo repository.InteractiveRepository) InteractiveService {
	return &interactiveService{
		repo: repo,
	}
}
