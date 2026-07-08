package failover

import (
	"GoBook/internal/service/sms"
	"context"
	"errors"
	"log"
	"sync/atomic"
)

type FailoverSMSService struct {
	svcs []sms.Service
	idx  uint64
}

func NewFailoverSMSService(svcs []sms.Service, idx uint64) sms.Service {
	return &FailoverSMSService{
		svcs: svcs,
		idx:  idx,
	}
}

func (f FailoverSMSService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	//轮询
	for _, svc := range f.svcs {
		err := svc.Send(ctx, tplId, args, numbers...)
		//发送成功
		if err == nil {
			return nil
		}
		//监控，计入日志
		log.Println(err)
	}
	//全部失败
	return errors.New("所有服务商都失败...")
}

func (f FailoverSMSService) SendV1(ctx context.Context, tplId string, args []string, numbers ...string) error {
	//取下一个节点作为起始节点，目的是每次轮询不要从零开始
	idx := atomic.AddUint64(&f.idx, 1)
	length := uint64(len(f.svcs))
	for i := idx; i < idx+length; i++ {
		//svcs[i]可能出现索引越界的情况，所有取值为i%uint64(length)
		err := f.svcs[i%length].Send(ctx, tplId, args, numbers...)
		switch err {
		case nil:
			return nil
			//超时、用户取消
		case context.DeadlineExceeded, context.Canceled:
			return errors.New("timeout")
		default:
			//计入日志
		}
	}
	return errors.New("所有服务商都失败...")
}
