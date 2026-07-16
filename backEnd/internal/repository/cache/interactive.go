package cache

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/incr_cnt.lua
	luaIncrCnt string
)

const fieldReadCnt = "read_cnt"
const fieldLikeCnt = "like_cnt"
const fieldCollectCnt = "collect_cnt"

type InteractiveCache interface {
	IncrReadCntIfPresent(ctx context.Context, biz string, id int64) error
	IncrLikeCntIfPresent(ctx context.Context, biz string, id int64) error
}

type RedisInteractiveCache struct {
	client redis.Cmdable
}

func NewRedisInteractiveCache(client redis.Cmdable) InteractiveCache {
	return &RedisInteractiveCache{client: client}
}

func (rc *RedisInteractiveCache) IncrLikeCntIfPresent(ctx context.Context, biz string, id int64) error {
	key := rc.key(biz, id)
	return rc.client.Eval(ctx, luaIncrCnt, []string{key}, fieldLikeCnt, 1).Err()
}

func (rc *RedisInteractiveCache) IncrReadCntIfPresent(ctx context.Context, biz string, id int64) error {
	key := rc.key(biz, id)
	return rc.client.Eval(ctx, luaIncrCnt, []string{key}, fieldReadCnt, 1).Err()
}

func (rc *RedisInteractiveCache) key(biz string, id int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, id)
}
