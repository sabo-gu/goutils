package balancer

import "testing"

func Test_NewRPCClient(t *testing.T) {
	_ = NewRPCClient("laike.daily.svc.cluster.local:9988")
}
