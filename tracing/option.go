package tracing

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrUnneededTracing = errors.New("unneeded tracing")
)

type tracingOptions struct {
	maxBodyLogSize int

	opNameFunc func() string

	httpCheckRequestFunc  func(req *http.Request, forServer bool) error
	httpCheckResponseFunc func(meta *HTTPResponseMeta, body []byte, forServer bool) error

	disableTracingGORMResultBody   bool
	disableTracingHTTPRequstBody   bool
	disableTracingHTTPResponseBody bool
	disableTracingGRPCRequstBody   bool
	disableTracingGRPCResponseBody bool
}

type TracingOption func(*tracingOptions)

func TracingOperationNameFunc(opNameFunc func() string) TracingOption {
	return func(options *tracingOptions) {
		options.opNameFunc = opNameFunc
	}
}

func TracingHTTPCheckRequestFunc(httpCheckRequestFunc func(req *http.Request, forServer bool) error) TracingOption {
	return func(options *tracingOptions) {
		options.httpCheckRequestFunc = httpCheckRequestFunc
	}
}

func TracingHTTPCheckResponseFunc(httpCheckResponseFunc func(meta *HTTPResponseMeta, body []byte, forServer bool) error) TracingOption {
	return func(options *tracingOptions) {
		options.httpCheckResponseFunc = httpCheckResponseFunc
	}
}

// 过期的，兼容老API
func TracingHTTPCheckErrorFunc(httpCheckResponseFunc func(meta *HTTPResponseMeta, body []byte, forServer bool) error) TracingOption {
	return func(options *tracingOptions) {
		options.httpCheckResponseFunc = httpCheckResponseFunc
	}
}

// 最大的body log长度，超过的话，会修剪，并不会丢失
func TracingMaxBodyLogSize(size int) TracingOption {
	return func(options *tracingOptions) {
		options.maxBodyLogSize = size
	}
}

func TracingGORMResultBody(enable bool) TracingOption {
	return func(options *tracingOptions) {
		options.disableTracingGORMResultBody = !enable
	}
}

func TracingHTTPRequstBody(enable bool) TracingOption {
	return func(options *tracingOptions) {
		options.disableTracingHTTPRequstBody = !enable
	}
}

func TracingHTTPResponseBody(enable bool) TracingOption {
	return func(options *tracingOptions) {
		options.disableTracingHTTPResponseBody = !enable
	}
}

func TracingGRPCRequstBody(enable bool) TracingOption {
	return func(options *tracingOptions) {
		options.disableTracingGRPCRequstBody = !enable
	}
}

func TracingGRPCResponseBody(enable bool) TracingOption {
	return func(options *tracingOptions) {
		options.disableTracingGRPCResponseBody = !enable
	}
}

func pruneBodyLog(log string, maxBodyLogSize int) string {
	if maxBodyLogSize <= 0 { //无需修剪
		return log
	}

	le := len(log)
	if le <= maxBodyLogSize {
		return log
	}

	r := fmt.Sprintf("Body is too large(%d), prune to %d-->\n", le, maxBodyLogSize)
	return r + log[0:maxBodyLogSize-len(r)]
}
