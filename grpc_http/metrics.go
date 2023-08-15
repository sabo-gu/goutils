package grpc_http

//
// import (
// 	"time"
//
// 	"github.com/go-kit/kit/endpoint"
// 	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
// 	stdprometheus "github.com/prometheus/client_golang/prometheus"
// 	"golang.org/x/net/context"
// )
//
// var fieldKeys = []string{"method", "sucess"}
// var counter = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
// 	Namespace: "api",
// 	Subsystem: "member_service",
// 	Name:      "request_count",
// 	Help:      "Number of requests received.",
// }, fieldKeys)
// var gauge = kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
// 	Namespace: "api",
// 	Subsystem: "member_service",
// 	Name:      "request_latency_microseconds",
// 	Help:      "Total duration of requests in microseconds.",
// }, fieldKeys)
//
// func MetricsMiddleware(method string) endpoint.Middleware {
// 	// logger = log.NewLogfmtLogger(log.NewSyncWriter(logFile))
// 	return func(next endpoint.Endpoint) endpoint.Endpoint {
// 		return func(ctx context.Context, request interface{}) (resp interface{}, err error) {
// 			defer func(begin time.Time) {
// 				if err == nil {
// 					counter.With("method", method).With("sucess", "1").Add(1)
// 					gauge.With("method", method).With("sucess", "1").Observe(time.Since(begin).Seconds())
// 				} else {
// 					counter.With("method", method).With("sucess", "0").Add(1)
// 					gauge.With("method", method).With("sucess", "0").Observe(time.Since(begin).Seconds())
// 				}
// 			}(time.Now())
// 			resp, err = next(ctx, request)
// 			return resp, err
// 		}
// 	}
// }
