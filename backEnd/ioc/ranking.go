package ioc

import (
	"GoBook/internal/job"
	"GoBook/internal/service"
	"GoBook/pkg/logger"
	"time"

	"github.com/robfig/cron/v3"
)

func InitRankingJob(
	svc service.RankingService,
	l logger.LoggerV1,
) *job.RankingJob {
	// 单次计算最多运行 30 秒，超时会通过 context 通知下游停止。
	return job.NewRankingJob(svc, time.Second*30, l)
}

// InitJobs 集中注册应用内的定时任务，但不在依赖注入阶段启动；
// cron 的启停由 main 管理，确保它与应用生命周期一致。
func InitJobs(l logger.LoggerV1, rankingJob *job.RankingJob) *cron.Cron {
	expr := cron.New(cron.WithSeconds())
	cronBuild := job.NewCronJobBuilder(l)
	// 启用秒字段后，该表达式表示每 3 分钟的第 0 秒运行一次。
	_, err := expr.AddJob("0 */3 * * * ?", cronBuild.Build(rankingJob))
	if err != nil {
		panic(err)
	}

	return expr
}
