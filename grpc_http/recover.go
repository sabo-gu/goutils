package grpc_http

import (
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/DoOR-Team/goutils/alert"
	"github.com/DoOR-Team/goutils/derror"
	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/trace"
	"github.com/DoOR-Team/goutils/tracing"
)

func RecoverMiddleware(method string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (out interface{}, err error) {
			defer func() {
				if perr := recover(); perr != nil {
					alert.AlertDingMsg(fmt.Sprintf("%s %s %s 异常", viper.GetString("k8s_appname"), "http", method),
						fmt.Sprintf("%#v %#v", err, string(trace.PanicTrace(10))),
						alert.AutoFire(),
					)
					log.Error(string(trace.PanicTrace(10)))
					err = derror.NewNoTipsError(fmt.Sprintf("%#v %#v", err, perr))
					err = errors.Wrap(err, "panic")

					if tracing.Enable {
						tracing.LogError(err) // 因为并没有直接传递原error对象到外部，所以我们在这里记录下
						tracing.SignSerious(true)
					}
				}
			}()

			return next(ctx, request)
		}
	}
}
