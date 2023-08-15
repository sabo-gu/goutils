package balancer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/DoOR-Team/goutils/balancer/balancer_policy"
	"github.com/DoOR-Team/goutils/balancer/registry/k8s"
	"github.com/DoOR-Team/goutils/grpc_http"
	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/waitgroup"
)

func NewRPCClientWithUserInfo(ctx context.Context, address string, userInfoKey string, userInfo interface{}) *grpc.ClientConn {
	useBalance := false

	var env, serviceName string
	attrs := strings.Split(address, ".")
	if len(attrs) >= 2 {
		env = attrs[1]
		serviceName = attrs[0]
		if env == "daily" || env == "production" {

			useBalance = true
		}
	}

	log.Println("address :", address)
	port := address[strings.Index(address, ":"):]
	// rr := grpc.RoundRobin(grpcsrvlb.New(srv.NewGoResolver(port, etcdHost, addr, 2*time.Second)))
	// rr := grpc.RoundRobin(grpcsrvlb.New(NewResolver(port, env, serviceName, 2*time.Second)))

	var opt []grpc.DialOption

	if useBalance {
		log.Info("使用load balancer 初始化：", address)
		k8s.RegisterResolver("k8s", port, env, serviceName, time.Second*5)
		opt = append(opt, grpc.WithBalancerName(balancer_policy.RoundRobin))
		address = "k8s:///" + address
		log.Info("address changing to", address)
	}
	opt = append(opt, grpc.WithInsecure())
	//	opt = append(opt, grpc.WithDefaultCallOptions(grpc.FailFast(false)))
	// opt = append(opt, grpc.WithWaitForHandshake())

	opt = append(opt, grpc.WithUnaryInterceptor(grpc_http.ClientInterceptor))
	if userInfoKey != "" && userInfo != nil {
		opt = append(opt, grpc.WithUnaryInterceptor(grpc_http.DeliverUserInfoClientInterceptorFactory(userInfoKey, userInfo)))
	}
	/*	opt = append(opt, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                50 * time.Millisecond,
			Timeout:             1 * time.Millisecond,
			PermitWithoutStream: true,
		}))
	*/
	conn, err := grpc.DialContext(ctx, address, opt...)
	if err != nil {
		// log.Panic(err)
		log.Fatal(err)
	}
	log.Info("初始化 grpc Dial success:", conn, err)
	_ = waitgroup.AddModAndWrapServer(fmt.Sprintf("GRPC_Client(%s)", address), &waitgroup.Cli{
		CloseFunc: func() error {
			return conn.Close()
		},
	})

	return conn
}

func NewRPCClient(address string) *grpc.ClientConn {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*100)
	return NewRPCClientWithUserInfo(ctx, address, "", nil)
}
