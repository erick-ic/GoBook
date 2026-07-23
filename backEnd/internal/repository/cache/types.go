package cache

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit"
)

// Cache 是通用键值缓存的抽象草稿，与排行榜专用的 RankingCache 相互独立。
type Cache interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) ekit.AnyValue
}

// LocalCache 和 RedisCache 预留给通用缓存的具体实现，目前尚未实现行为。
type LocalCache struct{}

type RedisCache struct{}

// DoubleCache 计划组合本地缓存和 Redis，统一封装双级缓存策略。
// 当前 Set/Get 仍为占位实现，业务流程暂时由 CachedRankingRepository 直接编排。
type DoubleCache struct {
	local Cache
	redis Cache
}

func (d *DoubleCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (d *DoubleCache) Get(ctx context.Context, key string) ekit.AnyValue {
	//TODO implement me
	panic("implement me")
}
