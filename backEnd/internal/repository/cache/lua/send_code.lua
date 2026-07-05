--Redis上的key
--phone_code:login:1871515....
-- 手机号对应的key
local key = KEYS[1] -- 对应 []string{cc.key(biz, phone)}
--验证次数
local countKey = key .. ":countKey"
--验证码
local val = ARGV[1] -- 对应传入的 code
--过期时间
local ttl = tonumber(redis.call("ttl", key))
if ttl == -1 then
    --https://redis.io/docs/latest/commands/ttl/
    --key存在，但无过期时间，属于系统错误
    return -2
elseif ttl == -2 or ttl < 540 then
    --key不存在，或设定的验证码有效期（十分钟）已经过去了一分钟。
    redis.call("set", key, val)
    redis.call("expire", key, 600)
    --设置验证次数：3
    redis.call("set", countKey, 3)
    redis.call("expire", countKey, 600)
    --一切正常
    return 0
else
    --发送频繁
    return -1
end