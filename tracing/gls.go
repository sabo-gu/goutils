package tracing

import (
	"github.com/jtolds/gls"
	opentracing "github.com/opentracing/opentracing-go"
)

var (
	mgr           = gls.NewContextManager()
	traceIdKey    = gls.GenSym()
	userIdKey     = gls.GenSym()
	prevMethodKey = gls.GenSym()
	prevAppKey    = gls.GenSym()
)

type glsSpanKey struct{}

var glsTracingSpanKey = glsSpanKey{}

func getGlsTracingSpan() opentracing.Span {
	val, ok := mgr.GetValue(glsTracingSpanKey)
	if ok {
		s, ok := val.(opentracing.Span)
		if ok {
			return s
		}
	}
	return nil
}

func setGlsTracingSpan(sp opentracing.Span, call func()) {
	mgr.SetValues(gls.Values{
		glsTracingSpanKey: sp,
	}, call)
}

func setNonGlsTracingSpan(call func()) {
	mgr.SetValues(gls.Values{
		glsTracingSpanKey: nil,
	}, call)
}
