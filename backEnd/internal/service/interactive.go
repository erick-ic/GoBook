package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"

	"golang.org/x/sync/errgroup"
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
	// Get 获取互动计数以及当前用户的点赞、收藏状态。
	Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error)
	// GetByIds 批量获取互动数据（用于排行榜计算）
	GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error)
	// Collect 将业务对象加入用户指定的收藏夹，并增加该业务对象的聚合收藏数。
	// id 是业务对象 ID，cid 是收藏夹 ID，uid 是执行收藏的用户 ID。
	Collect(ctx context.Context, biz string, id int64, cid int64, uid int64) error
}

// interactiveService 互动服务实现类
type interactiveService struct {
	repo repository.InteractiveRepository
}

// Collect 将收藏操作交给仓储层。
// 数据库事务以及数据库与 Redis 缓存之间的更新顺序由 Repository 统一协调。
func (is *interactiveService) Collect(ctx context.Context, biz string, id int64, cid int64, uid int64) error {
	return is.repo.AddCollectItem(ctx, biz, id, cid, uid)
}

// GetByIds 批量获取互动数据，用于排行榜服务查询多篇文章的点赞数。
// ids 为空时直接返回空结果，避免向下游发起无意义的数据库查询。
func (is *interactiveService) GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error) {
	if len(ids) == 0 {
		// 查询成功，但结果为空
		return map[int64]domain.Interactive{}, nil
	}

	iters, err := is.repo.GetByIds(ctx, biz, ids)
	if err != nil {
		// 查询失败，没有有效结果
		return nil, err
	}
	res := make(map[int64]domain.Interactive, len(iters))
	for _, inter := range iters {
		res[inter.BizId] = inter
	}
	return res, nil
}

// Get 获取互动计数以及当前用户的点赞、收藏状态。
// 调用链路：PubDetail Handler → Get → Repository.Get → DAO.Get
// 三个查询彼此独立，并发执行可避免串行增加文章详情接口的延迟。
func (is *interactiveService) Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error) {
	var (
		inter     domain.Interactive
		liked     bool
		collected bool
	)
	group, queryCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		inter, err = is.repo.Get(queryCtx, biz, id)
		return err
	})
	group.Go(func() error {
		var err error
		liked, err = is.repo.Liked(queryCtx, biz, id, uid)
		return err
	})
	group.Go(func() error {
		var err error
		collected, err = is.repo.Collected(queryCtx, biz, id, uid)
		return err
	})
	if err := group.Wait(); err != nil {
		return domain.Interactive{}, err
	}

	inter.Liked = liked
	inter.Collected = collected
	return inter, nil
}

// Like 点赞
// 调用链路：Like Handler → Like → Repository.IncrLike → DAO.InsertLikeInfo + Cache.IncrLikeCntIfPresent
// 实现幂等：通过 UserLikeBiz 表的唯一索引和状态条件更新，确保重复点赞不计数
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
