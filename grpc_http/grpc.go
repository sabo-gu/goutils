package grpc_http

import (
	"net"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/waitgroup"
)

type RegisterServerHandler func(s *grpc.Server)

var ServerInterceptor grpc.UnaryServerInterceptor
var ClientInterceptor grpc.UnaryClientInterceptor
var DeliverUserInfoServerInterceptorFactory func(userInfoKey string, userInfo interface{}) grpc.UnaryServerInterceptor
var DeliverUserInfoClientInterceptorFactory func(userInfoKey string, userInfo interface{}) grpc.UnaryClientInterceptor
var serverInterceptor grpc.UnaryServerInterceptor
var clientInterceptor grpc.UnaryClientInterceptor

func WrapServeGRPC(register RegisterServerHandler) error {
	return WrapServeGRPCWithUserInfo(register, "", nil)
}

func WrapServeGRPCWithUserInfo(register RegisterServerHandler, userInfoKey string, userInfo interface{}) error {
	log.Notice("GRPC 服务开始启动...")
	defer log.Info("GRPC 服务开始启动成功。")
	lis, err := net.Listen("tcp", viper.GetString("grpc_address"))
	if err != nil {
		log.Panicf("监听地址 [%s] 失败：%s。",
			viper.GetString("address"), err.Error())
	}

	//s := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{}))
	opts := []grpc.ServerOption{grpc.UnaryInterceptor(ServerInterceptor)}
	if userInfoKey != "" && userInfo != nil {
		opts = append(opts, grpc.UnaryInterceptor(DeliverUserInfoServerInterceptorFactory(userInfoKey, userInfo)))
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(ServerInterceptor))
	register(s)

	err = waitgroup.AddModAndWrapServer("GRPC_Server", &Srv{
		ServeFunc: func() error {
			log.Infof("监听服务地址 [%s] 成功。", viper.GetString("grpc_address"))
			return s.Serve(lis)
		},
		CloseFunc: func() error {
			s.GracefulStop()
			return nil
		},
	})
	if err != nil {
		panic(err)
	}

	return err
}
