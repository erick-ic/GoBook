package ratelimit

import (
	"GoBook/internal/service/sms"
	"GoBook/pkg/ratelimit"
	"context"
	"fmt"
)

var errLimited = fmt.Errorf("触发了限流")

type RatelimitSMSService struct {
	svc     sms.Service       // 被装饰的短信服务（如腾讯云、阿里云）
	limiter ratelimit.Limiter // 限流器
}

func NewRatelimitSMSService(svc sms.Service, limiter ratelimit.Limiter) sms.Service {
	return &RatelimitSMSService{
		svc:     svc,
		limiter: limiter,
	}
}

func (s *RatelimitSMSService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	limited, err := s.limiter.Limited(ctx, "sms:tencent")
	if err != nil {
		//系统错误
		//保守策略：限流，下游需要注意
		//容错策略：不限流，业务可用性要求很高，需要全面的状态

		//包装错误
		return fmt.Errorf("短信服务判断是否限流报错: %w", err)
	}
	if limited {
		return errLimited
	}

	//在此处加入代码，新特性
	err = s.svc.Send(ctx, tplId, args, numbers...)
	//在此处加入代码，新特性
	return err
}
