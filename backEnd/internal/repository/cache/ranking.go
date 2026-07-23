package cache

import (
	"GoBook/internal/domain"
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RankingCache interface {
	Set(ctx context.Context, arts []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

// NewRankingRedisCache 使用固定 key 保存当前榜单；过期时间应长于刷新周期，
// 使偶发的一次计算失败不会立即造成榜单不可用。
func NewRankingRedisCache(client redis.Cmdable) RankingCache {
	return &RankingRedisCache{
		client:     client,
		key:        "ranking:top_n",
		expiration: time.Minute * 30,
	}
}

func (r *RankingRedisCache) Set(ctx context.Context, arts []domain.Article) error {
	// 排行榜只用于摘要展示，写缓存前裁剪正文以控制 Redis value 大小。
	for i := range arts {
		arts[i].Content = arts[i].Abstract()
	}
	val, err := json.Marshal(arts)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key, val, r.expiration).Err()
}

// Get 读取的是最近一次完整替换的排行榜快照。
func (r *RankingRedisCache) Get(ctx context.Context) ([]domain.Article, error) {
	val, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		return nil, err
	}
	var res []domain.Article
	err = json.Unmarshal(val, &res)
	return res, err
}

type RankingRedisCache struct {
	client     redis.Cmdable
	key        string
	expiration time.Duration
}
