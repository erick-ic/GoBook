package cache

import (
	"GoBook/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ArticleCache interface {
	GetFirstPage(ctx context.Context, authorId int64) ([]domain.Article, error)
	SetFirstPage(ctx context.Context, authorId int64, articles []domain.Article) error
	DelFirstPage(ctx context.Context, authorId int64) error
}

type RedisArticleCache struct {
	client redis.Cmdable
}

func NewRedisArticleCache(client redis.Cmdable) ArticleCache {
	return &RedisArticleCache{
		client: client,
	}
}

func (r *RedisArticleCache) GetFirstPage(ctx context.Context, authorId int64) ([]domain.Article, error) {
	data, err := r.client.Get(ctx, r.key(authorId)).Bytes()
	if err != nil {
		return nil, err
	}
	var articles []domain.Article
	err = json.Unmarshal(data, &articles)
	return articles, err
}

func (r *RedisArticleCache) SetFirstPage(ctx context.Context, authorId int64, articles []domain.Article) error {
	// 空结果不缓存，避免后续请求命中"空缓存"导致看不到新创建的文章
	if len(articles) == 0 {
		return nil
	}
	for i := 0; i < len(articles); i++ {
		articles[i].Content = articles[i].Abstract()
	}
	data, err := json.Marshal(articles)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(authorId), data, time.Second*10).Err()
}

func (r *RedisArticleCache) key(authorId int64) string {
	return fmt.Sprintf("article:first_page:%d", authorId)
}

func (r *RedisArticleCache) DelFirstPage(ctx context.Context, authorId int64) error {
	return r.client.Del(ctx, r.key(authorId)).Err()
}
