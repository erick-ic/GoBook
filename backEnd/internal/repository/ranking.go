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

// CachedRankingRepository 将排行榜仓储操作委托给缓存层。
// ReplaceTopN 表达的是整份榜单覆盖，而不是在旧榜单上增量追加。
type CachedRankingRepository struct {
	cache cache.RankingCache
}

func NewCachedRankingRepository(cache cache.RankingCache) RankingRepository {
	return &CachedRankingRepository{cache: cache}
}

func (repo *CachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	return repo.cache.Get(ctx)
}

func (repo *CachedRankingRepository) ReplaceTopN(ctx context.Context, arts []domain.Article) error {
	return repo.cache.Set(ctx, arts)
}
