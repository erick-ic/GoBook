package failover

import (
	"GoBook/internal/service/sms"
	"context"
	"errors"
	"log"
	"sync/atomic"
)

type TimeoutFailoverSMSService struct {
	//服务商
	svcs []sms.Service
	idx  int32
	//连续超时个数
	cnt int32
	//阈值，连续超时超出数值后，需要切换
	threshold int32
}

func NewTimeoutFailoverSMSService(svcs []sms.Service) sms.Service {
	return &TimeoutFailoverSMSService{}
}

func (t *TimeoutFailoverSMSService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	idx := atomic.LoadInt32(&t.idx)
	cnt := atomic.LoadInt32(&t.cnt)

	if cnt > t.threshold {
		//连续超时超出数值后，触发切换，需要新的下标
		newIdx := (idx + 1) % int32(len(t.svcs))
		if atomic.CompareAndSwapInt32(&t.idx, idx, newIdx) {
			//操作失败，说明切换了，成功往后挪一位
			atomic.StoreInt32(&t.cnt, 0)
		}
		//idx = newIdx
		idx = atomic.LoadInt32(&t.idx)
	}
	svc := t.svcs[idx]
	err := svc.Send(ctx, tplId, args, numbers...)
	switch err {
	case nil:
		//连续状态被打断
		atomic.AddInt32(&t.cnt, 0)
	case context.DeadlineExceeded:
		atomic.AddInt32(&t.cnt, 1)
	default:
		//计入日志
		log.Println(err)
	}
	return errors.New("所有服务商都失败...")
}
