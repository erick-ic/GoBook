--local key = KEYS[1]
----用户输入的code
--local inputCode = ARGV[1]
--local code = redis.call("get", key)
----验证次数
--local countKey = key..":countKey"
--local cnt = tonumber(redis.call("get", countKey))
--if cnt <= 0 then
--    --说明一直输错
--    return -1
--elseif inputCode == code then
--    --输入正确
--    redis.call("set", countKey, -1)
--    return 0
--else
--    --用户手抖输错了
--    redis.call("decr", countKey)
--    return -2
--end

local key = KEYS[1]
local inputCode = ARGV[1]
local countKey = key .. ":countKey"

-- 获取存储的验证码
local storedCode = redis.call("get", key)

-- 如果验证码不存在，视为错误（返回 -2），但此时计数器可能也不存在，不进行减操作
if storedCode == false then
    return -2   -- 验证码已过期或不存在，按错误处理
end

-- 获取剩余尝试次数（注意：这里用 countKey）
local cnt = tonumber(redis.call("get", countKey))
if cnt == nil or cnt <= 0 then
    return -1   -- 次数已耗尽（或计数器异常）
end

-- 比较验证码
if storedCode == inputCode then
    -- 验证成功，将计数器置为 -1，防止再次使用（保持你原来的设计）
    redis.call("set", countKey, -1)
    return 0
else
    -- 验证码错误，次数减1，返回 -2
    redis.call("decr", countKey)
    return -2
end