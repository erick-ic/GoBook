package job

import (
	"GoBook/pkg/logger"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
)

type CronJobBuilder struct {
	l      logger.LoggerV1
	vector *prometheus.SummaryVec
}

// NewCronJobBuilder 创建所有定时任务共用的日志和耗时指标包装器。
func NewCronJobBuilder(l logger.LoggerV1) *CronJobBuilder {
	vector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "GoBook",
		Name:      "job",
		Subsystem: "GoBook_cron",
		Help:      "统计定时任务的执行情况",
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.9:   0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, []string{"job", "success"})
	prometheus.MustRegister(vector)
	return &CronJobBuilder{
		l:      l,
		vector: vector,
	}
}

// Build 将业务 Job 转成 cron.Job，并统一记录开始、结束、错误及执行耗时。
// 业务错误已经在这里记录，因此不会再向不支持 error 返回值的 cron.Job 传播。
func (jb *CronJobBuilder) Build(job Job) cron.Job {
	name := job.Name()
	return cronJobAdapterFunc(func() error {
		start := time.Now()
		jb.l.Info("任务开始", logger.String("job", name))
		var success bool

		defer func() {
			jb.l.Info("任务结束", logger.String("job", name))
			duration := time.Since(start)
			jb.vector.WithLabelValues(
				name,
				strconv.FormatBool(success),
			).Observe(float64(duration.Milliseconds()))

		}()

		err := job.Run()
		success = err == nil
		if err != nil {
			jb.l.Error("任务运行失败", logger.String("job", name), logger.Error(err))
		}
		return nil
	})
}

type cronJobAdapterFunc func() error

// Run 使函数类型满足 cron.Job 接口。
func (c cronJobAdapterFunc) Run() {
	_ = c()
}
