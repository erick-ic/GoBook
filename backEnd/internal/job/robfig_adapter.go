package job

import (
	"GoBook/pkg/logger"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type RankingJobAdapter struct {
	j Job
	l logger.LoggerV1
	p prometheus.Summary
}

// NewRankingJobAdapter 适配器写法
func NewRankingJobAdapter(j Job, l logger.LoggerV1) *RankingJobAdapter {
	p := prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "cron_job",
		ConstLabels: map[string]string{
			"job": j.Name(),
		},
	})
	prometheus.MustRegister(p)

	return &RankingJobAdapter{
		j: j,
		l: l,
		p: p,
	}
}
func (r *RankingJobAdapter) Run() {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		r.p.Observe(float64(duration))
	}()

	err := r.j.Run()
	if err != nil {
		r.l.Error(
			"运行任务失败！",
			logger.Error(err),
			logger.String("job", r.j.Name()),
		)
	}

}
