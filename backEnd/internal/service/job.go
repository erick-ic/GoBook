package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"GoBook/pkg/logger"
	"context"
	"time"
)

type JobService interface {
	// Preempt 抢占任务，并为成功抢到的任务启动续约和释放机制。
	Preempt(ctx context.Context) (domain.Job, error)
	// ResetNextTime 在任务执行成功后更新下一次计划执行时间。
	ResetNextTime(ctx context.Context, j domain.Job) error
}

// jobService 管理一次任务从“抢占成功”到“停止续约并释放”的持有周期。
type jobService struct {
	repo            repository.JobRepository
	l               logger.LoggerV1
	refreshInterval time.Duration
}

// ResetNextTime 根据任务的 Cron 表达式计算并保存下一次触发时间。
func (js *jobService) ResetNextTime(ctx context.Context, j domain.Job) error {
	nextTime := j.NextTime()
	return js.repo.UpdateNextTime(ctx, j.Id, nextTime)
}

// Preempt 在 Repository 抢占成功后启动定时心跳，并把清理逻辑封装到 CancelFunc：
// Scheduler 无需了解续约细节，只需保证任务结束时调用 CancelFunc。
func (js *jobService) Preempt(ctx context.Context) (domain.Job, error) {
	//抢占
	res, err := js.repo.Preempt(ctx)
	if err != nil {
		return domain.Job{}, err
	}

	//续约
	//ch := make(chan struct{})
	//go func() {
	//	ticker := time.NewTicker(time.Second)
	//
	//	for {
	//		select {
	//		case <-ticker.C:
	//			//续约
	//			j.refresh(res.Id)
	//		case <-ch:
	//			//结束
	//			return
	//		}
	//	}
	//}()
	ticker := time.NewTicker(js.refreshInterval)
	go func() {
		// 定期更新 utime，供后续的超时检测或故障转移逻辑判断任务是否仍被持有。
		for range ticker.C {
			js.refresh(res.Id)
		}
	}()

	// 任务结束时先停止后续心跳，再用独立的短超时 Context 释放数据库状态。
	// 使用 Background 是为了避免原调度请求 Context 已取消，导致 Release 无法执行。
	res.CancelFunc = func() {
		ticker.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		er := js.repo.Release(ctx, res.Id)
		if er != nil {
			js.l.Error(
				"释放job失败", logger.Error(er),
				logger.Int64("jid", res.Id),
			)
		}
	}

	return res, nil
}

func NewJobService(repo repository.JobRepository, l logger.LoggerV1) JobService {
	return &jobService{
		repo:            repo,
		l:               l,
		refreshInterval: time.Minute,
	}
}

// refresh 为单次心跳设置独立超时，避免数据库异常长期占用续约 goroutine。
func (js *jobService) refresh(id int64) {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second)
	defer cancle()

	err := js.repo.UpdateTime(ctx, id)
	if err != nil {
		js.l.Error("续约失败", logger.Error(err), logger.Int64("id", id))
	}
}
