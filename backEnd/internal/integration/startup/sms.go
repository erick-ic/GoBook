package startup

import (
	"GoBook/internal/service/sms"
	"GoBook/internal/service/sms/memory"
)

func InitSMSService() sms.Service {
	//方便更换SMS服务
	return memory.NewMemoService()
	////限流
	//svc := ratelimit.NewRatelimitSMSService(memory.NewMemoService(),
	//	ratelimit2.NewRedisSlideWindowLimiter(cmd, time.Second, 100))
	////重试
	//return retryable.NewService(svc, 3)
}
