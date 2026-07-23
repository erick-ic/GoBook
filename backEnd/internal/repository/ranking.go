package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	"context"
)

type RankingRepository interface {
	ReplaceTopN(ctx context.Context, arts []domain.Article) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

// CachedRankingRepository 为排行榜组合进程内缓存和 Redis：
// 读取采用“本地优先、Redis 回源并回填本地”的 cache-aside 策略；
// 写入则整体替换两级缓存中的榜单快照。
type CachedRankingRepository struct {
	// Redis 层依赖 RankingCache 接口，既能匹配 Wire provider，也便于测试时替换为 mock。
	redisCache cache.RankingCache
	// 本地层使用独立接口，避免与 Redis provider 冲突，也便于单独 mock。
	localCache cache.LocalRankingCache
}

// NewCachedRankingRepository 组装排行榜使用的两级缓存。
func NewCachedRankingRepository(redis cache.RankingCache, local cache.LocalRankingCache) RankingRepository {
	return &CachedRankingRepository{redisCache: redis, localCache: local}
}

func (repo *CachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	// 一级缓存命中时直接返回，避免每次查询都访问 Redis。
	data, err := repo.localCache.Get(ctx)
	if err == nil {
		return data, nil
	}

	// 本地未命中时读取 Redis；回源成功后回填本地缓存，供后续请求复用。
	data, err = repo.redisCache.Get(ctx)
	if err == nil {
		// 本地缓存写入当前不会失败，且不应影响已经成功取得的 Redis 数据。
		_ = repo.localCache.Set(ctx, data)
	}
	return data, nil
}

func (repo *CachedRankingRepository) ReplaceTopN(ctx context.Context, arts []domain.Article) error {
	// 本地缓存采用尽力写入；Redis 是跨实例共享的数据源，其写入结果决定本次刷新是否成功。
	_ = repo.localCache.Set(ctx, arts)
	return repo.redisCache.Set(ctx, arts)
}
