local key = KEYS[1]
--用户输入的code
local expectCode = ARGV[1]
local code = redis.call("get", key)
--验证次数
local countKey = key..":countKey"
local cnt = tonumber(redis.call("get"), countKey)
if cnt <= 0 then
    --说明一直输错
    return -1
elseif expectCode == code then
    --输入正确
    redis.call("set", countKey, -1)
    return 0
else
    --用户手抖输错了
    redis.call("decr", countKey)
    return -2
end
