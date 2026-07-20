package ioc

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// InitOpentelemetry 初始化 OpenTelemetry 链路追踪
// 完成 3 件事：
//  1. 创建 Resource（标识当前服务，附加到所有 span 上）
//  2. 创建 Propagator（跨进程传递 trace 上下文，如 HTTP header 中的 traceparent）
//  3. 创建 TracerProvider（管理 tracer，内置 Zipkin exporter 批量上报 span）
//
// 返回值：shutdown 函数，应用退出时调用以刷新尚未上报的 span
// 调用方：main.go 中 ioc.InitOpentelemetry()，并用 defer 调用返回的 shutdown
func InitOpentelemetry() func(ctx context.Context) {
	// 1. 创建 Resource，标识服务名和版本
	// 这些信息会附加到每个 span 上，在 Zipkin UI 中可按服务名筛选
	res, err := newResource("gobook", "v0.0.1")
	if err != nil {
		panic(err)
	}

	// 2. 创建 Propagator，用于在客户端和服务端之间传递 tracing 上下文
	// 例如：A 服务调用 B 服务时，A 把 trace_id 写入 HTTP header，B 从 header 读取并延续 trace
	prop := newPropagator()
	// 设置为全局 Propagator，otelgin 等 instrument 库会自动使用它
	otel.SetTextMapPropagator(prop)

	// 3. 创建 TracerProvider，核心组件，用来在打点时构建 trace
	// 内部配置了 Zipkin exporter，span 会批量发送到 Zipkin
	tp, err := newTraceProvider(res)
	if err != nil {
		panic(err)
	}
	// 设置为全局 TracerProvider，otelgin 等 instrument 库会自动使用它
	otel.SetTracerProvider(tp)

	// 返回 shutdown 函数：应用退出时调用，确保缓冲区中未上报的 span 被刷新到 Zipkin
	return func(ctx context.Context) {
		_ = tp.Shutdown(ctx)
	}
}

// newResource 创建 Resource，描述当前服务的信息
// Resource 会被附加到所有 span 上，用于在 Zipkin 中标识"这条 trace 来自哪个服务"
// 参考 OpenTelemetry 语义约定：https://opentelemetry.io/docs/specs/semconv/
func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	// resource.Default() 包含主机名、OS、进程等默认属性
	// resource.Merge 合并默认属性和自定义属性
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),       // 服务名，Zipkin 中按此筛选
			semconv.ServiceVersion(serviceVersion), // 服务版本，便于区分不同版本
		))
}

// newTraceProvider 创建 TracerProvider
// TracerProvider 是 OpenTelemetry 的核心，负责：
//  1. 创建 Tracer（业务代码用 tracer.Start 创建 span）
//  2. 配置 Exporter（把 span 上报到后端，这里是 Zipkin）
//  3. 配置 SpanProcessor（批量上报策略）
func newTraceProvider(res *resource.Resource) (*trace.TracerProvider, error) {
	// 创建 Zipkin exporter
	// 将 span 数据通过 HTTP POST 上报到 http://localhost:9411/api/v2/spans
	// 9411 是 Zipkin 的默认端口
	exporter, err := zipkin.New(
		"http://localhost:9411/api/v2/spans")
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		// WithBatcher 配置批量上报处理器（BatchSpanProcessor）
		// 内部维护一个队列，span 不会立即上报，而是按以下策略批量发送：
		//  - 队列满（默认 2048）触发上报
		//  - 每 batchTimeout 触发一次上报
		// 批量上报能减少 HTTP 请求次数，降低对业务性能的影响
		trace.WithBatcher(exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			// 生产环境用默认 5s 即可；演示环境用 1s 以便快速看到结果
			trace.WithBatchTimeout(time.Second)),
		// WithResource 把服务信息附加到所有 span 上
		trace.WithResource(res),
	)
	return traceProvider, nil
}

// newPropagator 创建 Propagator，用于跨进程传递 trace 上下文
// 场景：A 服务通过 HTTP 调用 B 服务时，需要把 trace_id 传过去，
//
//	B 服务接到后才能把 B 的 span 接到 A 的 trace 上，形成完整链路
func newPropagator() propagation.TextMapPropagator {
	// CompositeTextMapPropagator 组合多个 Propagator
	return propagation.NewCompositeTextMapPropagator(
		// TraceContext：W3C Trace Context 标准，通过 traceparent header 传递
		//   格式：traceparent: 00-{trace-id}-{parent-span-id}-{trace-flags}
		propagation.TraceContext{},
		// Baggage：跨进程传递业务自定义键值对（如 user_id、locale）
		//   与 trace 无关，但常配合使用，在整条调用链中透传业务上下文
		propagation.Baggage{},
	)
}
