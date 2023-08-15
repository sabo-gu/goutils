package serious

import (
	"fmt"
	"strconv"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
)

type SeriousObserver struct {
	alert func(opentracing.Span, string, string, string)
}

func NewObserver(alert func(sp opentracing.Span, traceID string, title string, msg string)) *SeriousObserver {
	return &SeriousObserver{alert: alert}
}

func (o *SeriousObserver) OnStartSpan(sp opentracing.Span, operationName string, options opentracing.StartSpanOptions) (jaeger.ContribSpanObserver, bool) {
	if o.alert == nil {
		return nil, false
	}
	return newSpanObserver(o.alert, sp, operationName, options), true
}

type spanObserver struct {
	alert func(opentracing.Span, string, string, string)

	sp            opentracing.Span
	operationName string
	options       opentracing.StartSpanOptions

	mux     sync.Mutex
	err     bool
	serious bool
}

func newSpanObserver(
	alert func(opentracing.Span, string, string, string),
	sp opentracing.Span,
	operationName string,
	options opentracing.StartSpanOptions,
) *spanObserver {
	so := &spanObserver{
		alert:         alert,
		sp:            sp,
		operationName: operationName,
		options:       options,
	}
	for k, v := range options.Tags {
		so.handleTagInLock(k, v)
	}
	return so
}

// handleTags watches for special tags
func (so *spanObserver) handleTagInLock(key string, value interface{}) {
	if key == string(ext.Error) {
		if v, ok := value.(bool); ok {
			so.err = v
		} else if v, ok := value.(string); ok {
			if vv, err := strconv.ParseBool(v); err == nil {
				so.err = vv
			}
		}
		return
	}
	if key == seriousTagKey {
		if v, ok := value.(bool); ok {
			so.serious = v
		} else if v, ok := value.(string); ok {
			if vv, err := strconv.ParseBool(v); err == nil {
				so.serious = vv
			}
		}
		return
	}
}

// OnSetOperationName implements ContribSpanObserver
func (so *spanObserver) OnSetOperationName(operationName string) {
	so.mux.Lock()
	defer so.mux.Unlock()
	so.operationName = operationName
}

// OnSetTag implements ContribSpanObserver
func (so *spanObserver) OnSetTag(key string, value interface{}) {
	so.mux.Lock()
	defer so.mux.Unlock()
	so.handleTagInLock(key, value)
}

// OnFinish implements ContribSpanObserver
func (so *spanObserver) OnFinish(options opentracing.FinishOptions) {
	so.mux.Lock()
	defer so.mux.Unlock()

	if so.operationName == "" || so.sp == nil || so.alert == nil {
		return
	}

	if so.err && so.serious {
		if sp, ok := so.sp.(*jaeger.Span); ok {
			if spCtx, ok := sp.Context().(jaeger.SpanContext); ok {
				var traceID string
				if spCtx.TraceID().High == 0 {
					traceID = fmt.Sprintf("%x", spCtx.TraceID().Low)
				} else {
					traceID = fmt.Sprintf("%x%016x", spCtx.TraceID().High, spCtx.TraceID().Low)
				}

				//这里的LogRecords并不会把所有log给到
				//TODO:还没做，需要找到field里key是error的玩意
				msg := ""
				//遍历所有log，为error key的记录

				// rs := sp.LogRecords()
				rs := sp.Logs()
				for _, r := range rs {
					for _, f := range r.Fields {
						if f.Key() == "error" {
							msg += fmt.Sprintf("%+v", f.Value())
						}
					}
				}

				so.alert(sp, traceID, so.operationName, msg)
			}
		}
	}
}
