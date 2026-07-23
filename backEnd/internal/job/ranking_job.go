package job

import (
	"GoBook/internal/service"
	"GoBook/pkg/logger"
	"context"
	"time"
)

type RankingJob struct {
	svc     service.RankingService
	timeout time.Duration
	l       logger.LoggerV1
}

// NewRankingJob 把排行榜业务封装成可由调度器执行的任务。
func NewRankingJob(svc service.RankingService, timeout time.Duration, l logger.LoggerV1) *RankingJob {
	return &RankingJob{
		svc:     svc,
		timeout: timeout,
		l:       l,
	}
}

func (r *RankingJob) Name() string {
	return "ranking"
}

// Run 为单次计算设置独立超时，防止任务执行过久并与下一轮调度重叠。
func (r *RankingJob) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	return r.svc.TopN(ctx)
}
