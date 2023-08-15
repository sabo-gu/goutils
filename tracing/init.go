package tracing

import (
	"fmt"
	"io"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/transport"

	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/tracing/serious"
	"github.com/DoOR-Team/goutils/waitgroup"
)

// 表示当前模块是否启用
var Enable = false

type jaegerLoggerAdapter struct{}

func (l *jaegerLoggerAdapter) Error(msg string) {
	log.Println("Jaeger Error:", msg)
}

func (l *jaegerLoggerAdapter) Infof(msg string, args ...interface{}) {
	log.Println("Jaeger:", fmt.Sprintf(msg, args...))
}

func newJaegerReporter(rc *config.ReporterConfig, logger jaeger.Logger, maxPacketSize int) (jaeger.Reporter, error) {
	addr := fmt.Sprintf("http://%s/api/traces?format=jaeger.thrift", rc.LocalAgentHostPort)
	// BatchSize设置少点比较好，能反馈的比较及时，方便调试
	// 另外比QueueSize大的话也比较更容易引起Reporter的Queue满溢引起丢弃
	sender := transport.NewHTTPTransport(addr, transport.HTTPBatchSize(100))
	// sender = jaeger.NewUDPTransport()

	reporter := jaeger.NewRemoteReporter(
		sender,
		jaeger.ReporterOptions.QueueSize(rc.QueueSize),
		jaeger.ReporterOptions.BufferFlushInterval(rc.BufferFlushInterval),
		jaeger.ReporterOptions.Logger(logger))
	if rc.LogSpans && logger != nil {
		logger.Infof("Initializing logging reporter\n")
		reporter = jaeger.NewCompositeReporter(jaeger.NewLoggingReporter(logger), reporter)
	}
	return reporter, nil
}

func newJaegerTracer(spanServerHostPort string, namespace string, svcName string) (*tracer, error) {
	if svcName == "" || namespace == "" || spanServerHostPort == "" {
		return nil, fmt.Errorf("参数丢失")
	}
	cfg := config.Configuration{
		// 全都记录，忽略采样
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
	}

	var logger jaeger.Logger
	// if namespace == "production" { //防止jaeger记录服务一直报错，线上环境无需打reporter log之类的
	// 	logger = jaeger.NullLogger
	// } else {
	logger = &jaegerLoggerAdapter{}
	// }

	rc := &config.ReporterConfig{
		LogSpans:            false, // 是否打印log
		BufferFlushInterval: 2 * time.Second,
		LocalAgentHostPort:  spanServerHostPort,
		QueueSize:           3000,
	}
	reporter, err := newJaegerReporter(rc, logger, 0)
	if err != nil {
		return nil, err
	}

	ob := serious.NewObserver(func(sp opentracing.Span, traceID string, title string, msg string) {
		go func() { // 异步去调用即可
			err := alertDing(svcName, title, msg, "点击前往", "http://jaeger.hetu.xuelangyun.com/trace/"+traceID)
			if err != nil {
				log.Error(err)
			}
		}()
	})
	t, closer, err := cfg.New(
		svcName,
		config.ZipkinSharedRPCSpan(false),
		config.Logger(logger),
		config.Reporter(reporter),
		config.Tag("namespace", namespace),
		config.ContribObserver(ob),
	)
	if err != nil {
		return nil, err
	}

	return &tracer{
		Tracer: t,
		closer: closer,
	}, nil
}

type tracer struct {
	opentracing.Tracer
	closer io.Closer
}

func (t *tracer) Close() error {
	return t.closer.Close()
}

// func newZipkinTracer(svcName string, collectorAddr string) (*tracer, error) {
// 	collector, err := zipkin.NewHTTPCollector(collectorAddr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	recorder := zipkin.NewRecorder(collector, true, "", svcName)
// 	t, err := zipkin.NewTracer(
// 		recorder,
// 		zipkin.ClientServerSameSpan(true),
// 		zipkin.TraceID128Bit(true),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &tracer{
// 		Tracer: t,
// 		closer: collector,
// 	}, nil
// }

func init() {
	// tracer_address，表示开启tracing
	// tracing_address表示一个jaeger地址

	waitgroup.AddModCreator("Tracer_Cli", 0, func() waitgroup.Mod {
		appName := viper.GetString("k8s_appname")
		tracerAddr := viper.GetString("tracer_address")
		namespace := viper.GetString("k8s_namespace")
		log.Info("初始化tracing模块")
		if appName == "" || tracerAddr == "" || namespace == "" {
			// panic("配置项k8s_appname和tracer_address以及k8s_namespace不可为空")
			log.Warn("tracing初始化失败，缺少k8s_appname/k8s_namespace/tracer_address某些配置项")
			return &waitgroup.NoopMod{}
		}

		t, err := newJaegerTracer(tracerAddr, namespace, appName)
		if err != nil {
			panic(err)
		}
		// 直接注册为global tracer
		opentracing.SetGlobalTracer(t.Tracer)

		Enable = true
		log.Info("tracing模块初始化成功")
		return t
	})

}
