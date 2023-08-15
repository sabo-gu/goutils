package tracing

import (
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// metadataReaderWriter satisfies both the opentracing.TextMapReader and
// opentracing.TextMapWriter interfaces.
type metadataReaderWriter struct {
	metadata.MD
}

func (w metadataReaderWriter) Set(key, val string) {
	// The GRPC HPACK implementation rejects any uppercase keys here.
	//
	// As such, since the HTTP_HEADERS format is case-insensitive anyway, we
	// blindly lowercase the key (which is guaranteed to work in the
	// Inject/Extract sense per the OpenTracing spec).
	key = strings.ToLower(key)
	w.MD[key] = append(w.MD[key], val)
}

func (w metadataReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range w.MD {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}

	return nil
}

// NewGRPCUnaryServerInterceptor 建议自行在interceptor里做panic的recover处理
func NewGRPCUnaryServerInterceptor(interceptor grpc.UnaryServerInterceptor, options ...TracingOption) grpc.UnaryServerInterceptor {
	tOpts := &tracingOptions{}
	for _, opt := range options {
		opt(tOpts)
	}

	if interceptor == nil {
		interceptor = func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp interface{}, err error) {
			return handler(ctx, req)
		}
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		tracer := opentracing.GlobalTracer()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, metadataReaderWriter{md})

		var op string
		if tOpts.opNameFunc != nil {
			op = tOpts.opNameFunc()
		}
		if op == "" {
			op = fmt.Sprintf("GRPC %s", info.FullMethod)
		}
		sp := tracer.StartSpan(
			op,
			ext.RPCServerOption(spanCtx),
		)
		defer sp.Finish()
		defer func() {
			r := recover() //简单recover记录下，再丢出去
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

		//设置tag
		ext.Component.Set(sp, "grpc")

		//记录请求
		traceGRPCMDWithSpan(sp, md)
		traceGRPCRequestWithSpan(sp, req, !tOpts.disableTracingGRPCRequstBody, tOpts.maxBodyLogSize)

		//执行请求，gls包裹
		setGlsTracingSpan(sp, func() {
			ctx = opentracing.ContextWithSpan(ctx, sp)
			resp, err = interceptor(ctx, req, info, handler)
		})

		//uid
		uid := sp.BaggageItem(BaggageItemKeyUserID)
		if uid != "" {
			sp.SetTag(TagKeyUserID, uid)
		}

		//记录错误
		if err != nil {
			otgrpc.SetSpanTags(sp, err, false) //这个里面能设置下错误code和class，grpc的标准
			ext.Error.Set(sp, true)            //上个方法里并没有设置
			sp.LogFields(ErrorField(err))      //server端不需要在这里的堆栈信息
		}

		//记录resp
		traceGRPCResponseWithSpan(sp, resp, !tOpts.disableTracingGRPCResponseBody, tOpts.maxBodyLogSize)
		return
	}
}

// NewGRPCUnaryClientInterceptor ...
func NewGRPCUnaryClientInterceptor(interceptor grpc.UnaryClientInterceptor, options ...TracingOption) grpc.UnaryClientInterceptor {
	tOpts := &tracingOptions{}
	for _, opt := range options {
		opt(tOpts)
	}

	if interceptor == nil {
		interceptor = func(
			ctx context.Context,
			method string,
			req, resp interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return invoker(ctx, method, req, resp, cc, opts...)
		}
	}

	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		if opentracing.SpanFromContext(ctx) == nil {
			//如果ctx里没传，就从gls获取
			glsSpan := getGlsTracingSpan()
			if glsSpan != nil {
				ctx = opentracing.ContextWithSpan(ctx, glsSpan)
			}
		}

		var parentCtx opentracing.SpanContext
		if parent := opentracing.SpanFromContext(ctx); parent != nil {
			parentCtx = parent.Context()
		}

		var op string
		if tOpts.opNameFunc != nil {
			op = tOpts.opNameFunc()
		}
		if op == "" {
			op = fmt.Sprintf("GRPC_CLI %s", method)
		}
		sp := opentracing.GlobalTracer().StartSpan(
			op,
			opentracing.ChildOf(parentCtx),
			ext.SpanKindRPCClient,
		)
		defer sp.Finish()
		defer func() {
			r := recover() //简单recover记录下，再丢出去
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

		//设置tag
		ext.Component.Set(sp, "grpc")

		//设置carrier
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		mdWriter := metadataReaderWriter{md}
		err = sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, mdWriter)
		if err != nil {
			ext.Error.Set(sp, true)
			sp.LogFields(ErrorField(errors.Wrap(err, "Tracer.Inject() failed")))
		}
		ctx = metadata.NewOutgoingContext(ctx, md)

		//记录请求
		traceGRPCMDWithSpan(sp, md)
		traceGRPCRequestWithSpan(sp, req, !tOpts.disableTracingGRPCRequstBody, tOpts.maxBodyLogSize)

		//执行请求
		err = interceptor(ctx, method, req, resp, cc, invoker, opts...)

		//uid
		uid := sp.BaggageItem(BaggageItemKeyUserID)
		if uid != "" {
			sp.SetTag(TagKeyUserID, uid)
		}

		//记录错误
		if err != nil {
			otgrpc.SetSpanTags(sp, err, true) //这个里面能设置下错误code和class，grpc的标准
			// ext.Error.Set(sp, true) //上个语句已经设置
			sp.LogFields(ErrorField(errors.Wrap(err, "Invoke failed")))
		}

		//记录resp
		traceGRPCResponseWithSpan(sp, resp, !tOpts.disableTracingGRPCResponseBody, tOpts.maxBodyLogSize)

		return
	}
}

func traceGRPCRequestWithSpan(sp opentracing.Span, req interface{}, traceBody bool, maxBodyLogSize int) {
	if !traceBody {
		return
	}

	jsn, err := jsoniter.Marshal(req)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal request.body failed")))
	} else {
		sp.LogFields(log.String("request.body", pruneBodyLog(string(jsn), maxBodyLogSize)))
	}
}

func traceGRPCResponseWithSpan(sp opentracing.Span, resp interface{}, traceBody bool, maxBodyLogSize int) {
	if !traceBody {
		return
	}

	jsn, err := jsoniter.Marshal(resp)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal response.body failed")))
	} else {
		sp.LogFields(log.String("response.body", pruneBodyLog(string(jsn), maxBodyLogSize)))
	}
}

func traceGRPCMDWithSpan(sp opentracing.Span, md metadata.MD) {
	jsn, err := jsoniter.Marshal(md)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal metadata failed")))
	} else {
		sp.LogFields(log.String("metadata", string(jsn)))
	}
}
