package cache

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Lua 脚本通过 //go:embed 嵌入到二进制文件中
var (
	//go:embed lua/incr_cnt.lua
	luaIncrCnt string
)

// Redis Hash 字段名常量，对应 Interactive 表的字段
const (
	fieldReadCnt    = "read_cnt"
	fieldLikeCnt    = "like_cnt"
	fieldCollectCnt = "collect_cnt"
)

// InteractiveCache 互动数据缓存接口
// 缓存策略：Cache-If-Present
//  1. 读操作：不主动查缓存（当前 Get 走数据库，后续可优化）
//  2. 写操作：数据库更新后，仅当缓存 key 存在时才更新缓存
//  3. 避免缓存击穿：不主动创建缓存，防止恶意请求打穿到数据库
type InteractiveCache interface {
	// IncrReadCntIfPresent 增加阅读数缓存（仅当 key 存在时才 HINCRBY）
	IncrReadCntIfPresent(ctx context.Context, biz string, id int64) error
	// IncrLikeCntIfPresent 增加点赞数缓存（仅当 key 存在时才 HINCRBY）
	IncrLikeCntIfPresent(ctx context.Context, biz string, id int64) error
}

// RedisInteractiveCache 互动数据 Redis 缓存实现
type RedisInteractiveCache struct {
	client redis.Cmdable
}

func NewRedisInteractiveCache(client redis.Cmdable) InteractiveCache {
	return &RedisInteractiveCache{client: client}
}

// IncrLikeCntIfPresent 增加点赞数缓存
// 通过 Lua 脚本实现原子操作：先 EXISTS 判断 key 是否存在，存在才 HINCRBY
// 参数：fieldLikeCnt（字段名）, 1（增量）
func (rc *RedisInteractiveCache) IncrLikeCntIfPresent(ctx context.Context, biz string, id int64) error {
	key := rc.key(biz, id)
	return rc.client.Eval(ctx, luaIncrCnt, []string{key}, fieldLikeCnt, 1).Err()
}

// IncrReadCntIfPresent 增加阅读数缓存
// 通过 Lua 脚本实现原子操作：先 EXISTS 判断 key 是否存在，存在才 HINCRBY
// 参数：fieldReadCnt（字段名）, 1（增量）
func (rc *RedisInteractiveCache) IncrReadCntIfPresent(ctx context.Context, biz string, id int64) error {
	key := rc.key(biz, id)
	return rc.client.Eval(ctx, luaIncrCnt, []string{key}, fieldReadCnt, 1).Err()
}

// key 生成 Redis 缓存 key
// 格式：interactive:{biz}:{id}，如 interactive:article:123
// 使用 Hash 结构存储：{read_cnt: 10, like_cnt: 5, collect_cnt: 2}
func (rc *RedisInteractiveCache) key(biz string, id int64) string {
	return fmt.Sprintf("interactive:%s:%d", biz, id)
}
