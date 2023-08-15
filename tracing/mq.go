package tracing

import (
	"context"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	AMQPCarrierJSONKey = "OpenTracing_AMQPCarrierJSON"
)

// PublishToAMQP 请自行处理do方法内部的panic返回错误
func PublishToAMQP(ctx context.Context, publishing *amqp.Publishing, bus string, do func(ctx context.Context, sp opentracing.Span, publishing *amqp.Publishing) error, options ...TracingOption) (ferr error) {
	opts := &tracingOptions{}
	for _, opt := range options {
		opt(opts)
	}

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

	var op string
	if opts.opNameFunc != nil {
		op = opts.opNameFunc()
	}
	if op == "" {
		op = fmt.Sprintf("AMQP PUBLISH %s", bus)
	}
	sp := opentracing.GlobalTracer().StartSpan(
		op,
		opentracing.ChildOf(parentCtx),
		ext.SpanKindProducer,
		opentracing.Tag{Key: string(ext.Component), Value: "amqp"},
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

	// 塞入carrier，保证两端可以响应
	carrier := opentracing.TextMapCarrier{}
	if err := opentracing.GlobalTracer().Inject(sp.Context(), opentracing.TextMap, carrier); err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Inject TextMap carrier failed")))
	}

	// 设置消息topic之类的
	ext.MessageBusDestination.Set(sp, bus)

	// 记录request
	requestMeta := &struct {
		*amqp.Publishing
		OmitBody bool `json:"Body,omitempty"`
	}{
		Publishing: publishing,
	}
	jsn, err := jsoniter.Marshal(requestMeta)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal request.meta failed")))
	} else {
		sp.LogFields(log.String("request.meta", string(jsn)))
	}

	if len(publishing.Body) > 0 {
		sp.LogFields(log.String("request.body", string(publishing.Body)))
	}

	jsn, err = jsoniter.Marshal(carrier)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal carrier failed")))
	} else {
		publishing.Headers[AMQPCarrierJSONKey] = string(jsn)
	}

	// 执行并记录错误
	ctx = opentracing.ContextWithSpan(ctx, sp)
	ferr = do(ctx, sp, publishing)

	// uid
	uid := sp.BaggageItem(BaggageItemKeyUserID)
	if uid != "" {
		sp.SetTag(TagKeyUserID, uid)
	}

	if ferr != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(ferr, "Publish failed")))
	}
	return
}

// ConsumeFromAMQP 请自行处理do方法内部的panic返回错误
func ConsumeFromAMQP(delivery *amqp.Delivery, do func(sp opentracing.Span, d *amqp.Delivery) (bool, error), options ...TracingOption) (fack bool, ferr error) {
	opts := &tracingOptions{}
	for _, opt := range options {
		opt(opts)
	}

	var parentCtx opentracing.SpanContext
	// 解出来传递的carrier
	val, ok := delivery.Headers[AMQPCarrierJSONKey].(string)
	if ok {
		carrier := opentracing.TextMapCarrier{}
		err := jsoniter.Unmarshal([]byte(val), &carrier)
		if err == nil {
			spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, carrier)
			if err == nil {
				parentCtx = spanCtx
			}
		}
	}

	bus := delivery.RoutingKey

	var op string
	if opts.opNameFunc != nil {
		op = opts.opNameFunc()
	}
	if op == "" {
		op = fmt.Sprintf("AMQP CONSUME %s", bus)
	}
	sp := opentracing.GlobalTracer().StartSpan(
		op,
		opentracing.FollowsFrom(parentCtx),
		ext.SpanKindConsumer,
		opentracing.Tag{Key: string(ext.Component), Value: "amqp"},
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

	// 设置消息topic之类的
	ext.MessageBusDestination.Set(sp, bus)

	// 记录request
	responseMeta := &struct {
		*amqp.Delivery
		OmitBody bool `json:"Body,omitempty"`
	}{
		Delivery: delivery,
	}
	jsn, err := jsoniter.Marshal(responseMeta)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal request.meta failed")))
	} else {
		sp.LogFields(log.String("request.meta", string(jsn)))
	}
	if len(delivery.Body) > 0 {
		sp.LogFields(log.String("request.body", string(delivery.Body)))
	}

	// 执行，包裹gls
	setGlsTracingSpan(sp, func() {
		fack, ferr = do(sp, delivery)
	})

	// uid
	uid := sp.BaggageItem(BaggageItemKeyUserID)
	if uid != "" {
		sp.SetTag(TagKeyUserID, uid)
	}

	// 记录结果
	sp.LogFields(log.Bool("response.ack", fack))

	// 如果错误了，记录错误
	if ferr != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(ferr)) // 不需要记录在这里的堆栈信息，因为非调用方，只是消费者
	} else if !fack {
		// 如果没错误，但是没有ack，也得标记成错误了，至于ack的值上面已经记录过了
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(fmt.Errorf("Not ack"))) // 不需要记录在这里的堆栈信息，因为非调用方，只是消费者
	}
	return
}
