package grpc_http

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

func SetGrpcOutContext(ctx context.Context, k string, v string) context.Context {
	c, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{k: v}))
		return ctx
	}
	cp := c.Copy()
	cp[k] = []string{v}
	ctx = metadata.NewOutgoingContext(ctx, cp)
	return ctx
}

func GetGrpcInContext(ctx context.Context, k string) string {
	c, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := c[k]
	if vals == nil {
		return ""
	}
	return vals[0]
}
