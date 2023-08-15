package grpc_http

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/DoOR-Team/goutils/alert"
	"github.com/DoOR-Team/goutils/derror"
	"github.com/DoOR-Team/goutils/trace"
	"github.com/DoOR-Team/goutils/tracing"
)

const TRACE_ID string = "trace_id"
const USER_ID string = "user_id"
const PREV_APP string = "prev_app"
const PREV_METHOD string = "prev_method"
const USER_INFO = "user_info"

var init_lock = sync.Mutex{}

func init() {
	init_lock.Lock()
	defer init_lock.Unlock()

	serverInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		var method string

		defer func() {
			if perr := recover(); perr != nil {
				alert.AlertDingMsg(fmt.Sprintf("%s %s %s 异常", viper.GetString("k8s_appname"), "grpc", method),
					fmt.Sprintf("%#v %#v", err, string(trace.PanicTrace(10))),
					alert.AutoFire(),
				)
				log.Println(string(trace.PanicTrace(10)))
				err = derror.NewNoTipsError(fmt.Sprintf("%#v %#v", err, perr))
				err = errors.Wrap(err, "panic")

				if tracing.Enable {
					tracing.SignSerious(true)
				}
			}
		}()

		tips := strings.Split(info.FullMethod, "/")
		method = tips[len(tips)-1]
		out, err = handler(ctx, req)
		return
	}

	ServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
		si := serverInterceptor
		if tracing.Enable {
			si = tracing.NewGRPCUnaryServerInterceptor(si, tracing.TracingMaxBodyLogSize(5*1024*1024))
		}
		return si(ctx, req, info, handler)
	}

	clientInterceptor = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		var traceID string
		defer func() {
			if perr := recover(); perr != nil {
				alert.AlertDingMsg("grpc 调用异常",
					fmt.Sprintf("%s %#v %#v %s", method, err, perr, traceID),
					alert.AutoFire(),
				)
				err = derror.NewNoTipsError(fmt.Sprintf("%s %#v %#v", method, err, perr))
				err = errors.Wrap(err, "panic")

				if tracing.Enable {
					tracing.SignSerious(true)
				}
			}
		}()

		traceID = tracing.GetTraceID()
		userID := tracing.GetUserID()
		ctx = SetGrpcOutContext(ctx, TRACE_ID, traceID)
		ctx = SetGrpcOutContext(ctx, USER_ID, userID)
		ctx = SetGrpcOutContext(ctx, PREV_APP, tracing.GetPrevApp())
		ctx = SetGrpcOutContext(ctx, PREV_METHOD, tracing.GetPrevMethod())
		ctx, _ = context.WithTimeout(ctx, time.Second*100)
		err = invoker(ctx, method, req, reply, cc, opts...)
		// 透传WrapError支持
		err = derror.WrapWithGrpcStatus(status.Convert(err))
		return
	}

	DeliverUserInfoServerInterceptorFactory = func(userInfoKey string, userInfo interface{}) grpc.UnaryServerInterceptor {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (out interface{}, err error) {
			userInfoJson := GetGrpcInContext(ctx, userInfoKey)
			jsonErr := json.Unmarshal([]byte(userInfoJson), &userInfo)
			if jsonErr != nil {
				log.Println("DeliverUserInfoServerInterceptorFactory json.Unmarshal", derror.Wrap(err))
			} else {
				ctx = context.WithValue(ctx, userInfoKey, userInfo)
			}
			return handler(ctx, req)
		}
	}

	DeliverUserInfoClientInterceptorFactory = func(userInfoKey string, userInfo interface{}) grpc.UnaryClientInterceptor {
		return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
			value := ctx.Value(userInfoKey)
			if value != nil {
				userInfoJson, _ := json.Marshal(value)
				ctx = SetGrpcOutContext(ctx, USER_INFO, string(userInfoJson))
			}
			err = invoker(ctx, method, req, reply, cc, opts...)
			return
		}
	}

	ClientInterceptor = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		ci := clientInterceptor
		if tracing.Enable {
			ci = tracing.NewGRPCUnaryClientInterceptor(ci, tracing.TracingMaxBodyLogSize(5*1024*1024))
		}
		return ci(ctx, method, req, reply, cc, invoker, opts...)
	}
}
