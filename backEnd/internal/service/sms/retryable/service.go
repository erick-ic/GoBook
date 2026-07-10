package retryable

import (
	"GoBook/internal/service/sms"
	"context"
	"errors"
)

type Service struct {
	svc      sms.Service
	retryMax int
}

func NewService(svc sms.Service, retryMax int) sms.Service {
	return &Service{
		svc:      svc,
		retryMax: retryMax,
	}
}

func (s *Service) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	err := s.svc.Send(ctx, tplId, args, numbers...)
	cnt := 1
	for err != nil && cnt < s.retryMax {
		err = s.svc.Send(ctx, tplId, args, numbers...)
		if err == nil {
			return nil
		}
		cnt++
	}
	return errors.New("重试都失败了")
}
