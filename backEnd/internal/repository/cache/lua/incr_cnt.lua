-- 原子递增缓存中的互动计数（阅读数/点赞数/收藏数）
-- 使用 Lua 脚本保证 EXISTS + HINCRBY 的原子性，避免并发问题
--
-- KEYS[1]：缓存 key，格式为 interactive:{biz}:{id}，如 interactive:article:123
-- ARGV[1]：Hash 字段名，可选值：read_cnt / like_cnt / collect_cnt
-- ARGV[2]：增量值，通常为 1
--
-- 返回值：
--   1：缓存存在，已执行 HINCRBY
--   0：缓存不存在，不执行任何操作（Cache-If-Present 策略）
--
-- 为什么用 Lua 脚本？
--   如果分开调用 EXISTS 和 HINCRBY，两个命令之间存在时间窗口，
--   可能在 EXISTS 返回 1 后、HINCRBY 执行前，key 被 DEL 删除，
--   导致 HINCRBY 重新创建一个只有单个字段的 Hash，数据不完整。
--   Lua 脚本在 Redis 中是原子执行的，避免这个并发问题。

-- 具体业务的缓存 key
local key = KEYS[1]
-- 要递增的字段名（read_cnt / like_cnt / collect_cnt）
local cntKey = ARGV[1]
-- 增量值
local delta = tonumber(ARGV[2])

-- 先判断缓存是否存在
local exist = redis.call("EXISTS", key)
if exist == 1 then
    -- 缓存存在，原子递增对应字段
    redis.call("HINCRBY", key, cntKey, delta)
    return 1
else
    -- 缓存不存在，不执行任何操作（避免缓存击穿）
    return 0
end
