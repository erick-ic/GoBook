package job

import (
	"GoBook/internal/service"
	"GoBook/pkg/logger"
	"context"
	"sync"
	"time"

	rlock "github.com/gotomicro/redis-lock"
)

type RankingJob struct {
	svc     service.RankingService
	timeout time.Duration
	l       logger.LoggerV1
	// client 和 key 用于在多个应用实例之间竞争同一排行榜任务。
	client *rlock.Client
	key    string

	lock      *rlock.Lock
	localLock *sync.Mutex //引入内置的锁
}

// NewRankingJob 把排行榜业务封装成可由调度器执行的任务，并注入 Redis 锁客户端。
// 当前排行榜使用固定的锁 key，确保所有实例竞争的是同一把锁。
func NewRankingJob(
	svc service.RankingService,
	timeout time.Duration,
	l logger.LoggerV1,
	client *rlock.Client,
) *RankingJob {
	return &RankingJob{
		svc:       svc,
		timeout:   timeout,
		l:         l,
		client:    client,
		key:       "rlock:cron_job:ranking",
		localLock: &sync.Mutex{},
	}
}

func (r *RankingJob) Name() string {
	return "ranking"
}

//1.基本方案：只能控制同一时刻只有一个goroutine计算，但是无法控制计算之后，其他机器再去计算。
// Run 在任务超时范围内竞争分布式锁，只有成功持锁的实例才会刷新排行榜。
// 这样部署多个应用实例时，同一调度周期也只会执行一次实际计算。
//func (r *RankingJob) Run() error {
//	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
//	defer cancel()
//
//	// 锁的存活时间与任务超时一致：即使实例异常退出，锁最终也会自动过期。
//	// 获取失败时每 100ms 重试一次，最多重试 3 次，避免长时间阻塞 cron 调度。
//	lock, err := r.client.Lock(
//		ctx,
//		r.key,
//		r.timeout,
//		&rlock.FixIntervalRetry{
//			Interval: time.Millisecond * 100,
//			Max:      3,
//		},
//		r.timeout,
//	)
//	if err != nil {
//		return err
//	}
//
//	defer func() {
//		// 任务 context 可能已经超时或被取消，因此释放锁要使用独立 context，
//		// 否则 Unlock 会立即失败，只能等待 Redis 中的锁自然过期。
//		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
//		defer cancel()
//
//		err = lock.Unlock(ctx)
//		if err != nil {
//			r.l.Error("释放分布式锁失败:ranking_job", logger.Error(err))
//		}
//	}()
//
//	// 排行榜计算与缓存刷新都共享任务 context，超过时限后下游可及时终止。
//	return r.svc.TopN(ctx)
//}

// 扩大锁的范围：在启动时拿到锁，不管计算多少次都不释放锁。
func (r *RankingJob) Run() error {
	r.localLock.Lock()
	defer r.localLock.Unlock()

	//抢分布式锁
	if r.lock == nil {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		lock, err := r.client.Lock(
			ctx,
			r.key,
			r.timeout,
			&rlock.FixIntervalRetry{
				Interval: time.Millisecond * 100,
				Max:      3,
			},
			r.timeout,
		)
		if err != nil {
			// localLock 已通过 defer 统一释放，此处不能再次手动解锁。
			r.l.Warn("获取分布式锁失败", logger.Error(err))
			return nil
		}

		r.lock = lock
		go func() {
			// AutoRefresh 会持续运行到锁被 Close 主动释放或续约失败。
			// 续约期间不能持有 localLock，否则 Close 无法读取锁并触发退出。
			refreshErr := lock.AutoRefresh(r.timeout/2, time.Second)
			if refreshErr != nil {
				// 续约失败了
				// 无法中断当下正在调度的热榜计算（如果有）
				r.l.Error("续约失败", logger.Error(refreshErr))
			}

			r.localLock.Lock()
			// 只清理当前续约协程对应的锁，避免覆盖后续重新取得的新锁。
			if r.lock == lock {
				r.lock = nil
			}
			r.localLock.Unlock()
		}()
	}

	//抢到了锁
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	return r.svc.TopN(ctx)
}

// Close 主动释放 RankingJob 持有的分布式锁，使其他实例可以接管排行榜任务。
// 尚未成功取得锁时直接返回，保证应用关闭流程可重复、安全地调用。
func (r *RankingJob) Close() error {
	r.localLock.Lock()
	lock := r.lock
	r.localLock.Unlock()

	if lock == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return lock.Unlock(ctx)
}
