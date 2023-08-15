package tracing

import (
	"fmt"
	stdlog "log"
	"math/rand"
	"strings"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	jaeger "github.com/uber/jaeger-client-go"
	context "golang.org/x/net/context"

	"github.com/DoOR-Team/goutils/tracing/serious"
)

/*
	// 如果不记录当前调用
	// ext.SamplingPriority.Set(tracing.CurrentSpan(), 0)
*/

const BaggageItemKeyUserID = "user.id"
const TagKeyUserID = "user.id"

func Printf(format string, v ...interface{}) {
	stdlog.Printf(format, v...)
	format = strings.TrimSuffix(format, "\n")
	LogEvent(fmt.Sprintf(format, v...))
}

func ErrorField(err error) log.Field {
	// 为什么不使用log.Error？
	// 因为这个最终输出会是err.Error()，不会将pkg.error的stacks打印出来
	// 我们一些时候还是希望有这种信息
	// jaeger里对object类型的filed会做fmt.Sprintf("%+v",obj)处理
	// 所以我们用这个
	return log.Object("error", err)
}

func GetTraceID() string {
	if requestId, ok := mgr.GetValue(traceIdKey); ok {
		return requestId.(string)
	} else {
		return ""
	}
}

func GetPrevMethod() string {
	if prevMethod, ok := mgr.GetValue(prevMethodKey); ok {
		return prevMethod.(string)
	} else {
		return ""
	}
}

func GetPrevApp() string {
	if prevApp, ok := mgr.GetValue(prevAppKey); ok {
		return prevApp.(string)
	} else {
		return ""
	}
}

func GetUserID() string {
	if resp, ok := mgr.GetValue(userIdKey); ok {
		return resp.(string)
	} else {
		return ""
	}
}
func MakeNewTraceID() string {
	traceID := GetTraceID()
	if traceID == "" {
		return getTraceString()
	} else {
		return traceID + "-" + getTraceString()
	}
}

func MakeNewFrom(old string) string {
	if old == "" {
		return getTraceString()
	}
	return old + "-" + getTraceString()
}

// 下面这些供外部使用，服从gls
func CurrentSpan() opentracing.Span {
	return getGlsTracingSpan()
}

func CurrentTraceID() string {
	curSp := CurrentSpan()
	if sp, ok := curSp.(*jaeger.Span); ok {
		if spCtx, ok := sp.Context().(jaeger.SpanContext); ok {
			var traceID string
			if spCtx.TraceID().High == 0 {
				traceID = fmt.Sprintf("%x", spCtx.TraceID().Low)
			} else {
				traceID = fmt.Sprintf("%x%016x", spCtx.TraceID().High, spCtx.TraceID().Low)
			}
			return traceID
		}
	}
	return ""
}

// 取指定长度随机字符串
func getTraceString() string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// 进行切片转换
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()
	bytes := []byte(str)
	result := [8]byte{}
	result[7] = bytes[now.Nanosecond()%62]
	result[6] = bytes[now.Second()%62]
	result[5] = bytes[now.Minute()%62]
	result[4] = bytes[(now.Hour()+r.Intn(1000))%24]
	result[3] = bytes[r.Intn(62)]
	result[2] = bytes[r.Intn(62)]
	result[1] = bytes[r.Intn(62)]
	result[0] = bytes[r.Intn(62)]

	return string(result[:])
	// //声明空数组

	// //
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// for i := 0; i < lenght; i++ {
	//	result = append(result, bytes[r.Intn(len(bytes))])
	// }
	// return string(result)
}

func LogFields(f log.Field) {
	glsSpan := getGlsTracingSpan()
	if glsSpan != nil {
		glsSpan.LogFields(f)
	}
}

func SetTag(key string, value interface{}) opentracing.Span {
	glsSpan := getGlsTracingSpan()
	if glsSpan != nil {
		return glsSpan.SetTag(key, value)
	}
	return nil
}

func LogEvent(e string) {
	LogFields(log.String("event", e))
}

func Logf(format string, v ...interface{}) {
	if viper.Get("k8s_namespace") == "dev" {
		stdlog.Printf(format, v...)
	}
	format = strings.TrimSuffix(format, "\n")
	LogEvent(fmt.Sprintf(format, v...))
}

func LogError(err error) {
	glsSpan := getGlsTracingSpan()
	if glsSpan != nil {
		glsSpan.LogFields(ErrorField(err))
		ext.Error.Set(glsSpan, true)
	}
}

// 标记严重，例如若使用了seriousObserver，在最终在标记严重的状态下发生了错误，会发生报警
func SignSerious(s bool) {
	glsSpan := getGlsTracingSpan()
	if glsSpan != nil {
		serious.SignSerious(glsSpan, s)
	}
}

// 开始无gls的执行，主要用处是一些情况下断链，防止调用链过长
func StartNonGlsTracingSpanCall(call func()) {
	setNonGlsTracingSpan(call)
}

// 开始一段调用的便捷方法
func StartSpanWithContext(ctx context.Context, call func(sp opentracing.Span) error,
	operationName string, component string, opts ...opentracing.StartSpanOption) error {
	return StartSpanWithContextV2(ctx, func(ctx context.Context, sp opentracing.Span) error {
		return call(sp)
	}, operationName, component, opts...)
}

func StartSpanWithContextV2(ctx context.Context, call func(ctx context.Context, sp opentracing.Span) error,
	operationName string, component string, opts ...opentracing.StartSpanOption) error {
	if opentracing.SpanFromContext(ctx) == nil {
		// 如果ctx里没传，就从gls获取
		glsSpan := getGlsTracingSpan()
		if glsSpan != nil {
			ctx = opentracing.ContextWithSpan(ctx, glsSpan)
		}
	}

	var parentCtx opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	opts = append(opts, opentracing.ChildOf(parentCtx))
	opts = append(opts, opentracing.Tag{Key: string(ext.Component), Value: component})

	sp := opentracing.GlobalTracer().StartSpan(
		operationName,
		opts...,
	)
	defer sp.Finish()
	defer func() {
		r := recover() // 简单recover记录下，再丢出去
		if r != nil {
			ext.Error.Set(sp, true)
			perr, ok := r.(error)
			if !ok {
				perr = fmt.Errorf(fmt.Sprintln(r))
			}
			sp.LogFields(ErrorField(errors.Wrap(perr, "panic")))

			panic(r)
		}
	}()

	var err error
	setGlsTracingSpan(sp, func() {
		ctx = opentracing.ContextWithSpan(ctx, sp)
		err = call(ctx, sp)
	})

	// uid
	uid := sp.BaggageItem(BaggageItemKeyUserID)
	if uid != "" {
		sp.SetTag(TagKeyUserID, uid)
	}

	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(err))
	}
	return err
}
