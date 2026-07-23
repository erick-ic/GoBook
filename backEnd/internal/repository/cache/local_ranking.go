package cache

import (
	"GoBook/internal/domain"
	"context"
	"errors"
	"time"

	"github.com/ecodeclub/ekit/syncx/atomicx"
)

// LocalRankingCache 描述排行榜在单个应用实例中的一级缓存能力。
// 它与 Redis 使用的 RankingCache 分开定义，便于 Wire 区分两种缓存依赖。
type LocalRankingCache interface {
	Set(ctx context.Context, arts []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RankingLocalCache struct {
	// topN 和 ddl 分别保存榜单快照及其失效时间；原子容器保证并发读写安全。
	topN       *atomicx.Value[[]domain.Article]
	ddl        *atomicx.Value[time.Time]
	expiration time.Duration
}

// NewRankingLocalCache 创建进程内排行榜缓存。
// 当前实现会自行初始化原子容器，并使用固定的 10 分钟有效期。
func NewRankingLocalCache() LocalRankingCache {
	return &RankingLocalCache{
		topN:       atomicx.NewValue[[]domain.Article](),
		ddl:        atomicx.NewValueOf[time.Time](time.Now()),
		expiration: time.Minute * 10,
	}
}

// Set 用新榜单整体覆盖旧快照，并从写入时刻重新计算过期时间。
func (r *RankingLocalCache) Set(ctx context.Context, arts []domain.Article) error {
	r.topN.Store(arts)
	ddl := time.Now().Add(r.expiration)
	r.ddl.Store(ddl)
	return nil
}

// Get 仅返回仍在有效期内的非空榜单；未初始化和过期统一视为缓存未命中，
// 由仓储层继续回源 Redis。
func (r *RankingLocalCache) Get(ctx context.Context) ([]domain.Article, error) {
	ddl := r.ddl.Load()
	arts := r.topN.Load()
	if len(arts) == 0 || ddl.Before(time.Now()) {
		return nil, errors.New("本地缓存过期")
	}
	return arts, nil
}
