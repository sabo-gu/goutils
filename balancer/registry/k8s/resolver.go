package k8s

import (
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"
	"k8s.io/client-go/kubernetes"

	"github.com/DoOR-Team/goutils/log"
)

// type etcdResolver struct {
// 	scheme        string
// 	etcdConfig    etcd_cli.Config
// 	etcdWatchPath string
// 	watcher       *Watcher
// 	cc            resolver.ClientConn
// 	wg            sync.WaitGroup
// }

// Resolver is an implementation of a DNS SRV resolver for a domain.
type k8sResolver struct {
	scheme      string
	clientset   *kubernetes.Clientset
	serviceName string
	ttl         time.Duration

	watcher *Watcher

	port string

	namespace string

	cc  resolver.ClientConn
	ctx context.Context
	wg  sync.WaitGroup
}

func (r *k8sResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.cc = cc
	r.watcher = newWatcher(r.namespace, r.serviceName, r.port, r.clientset)
	r.start()
	return r, nil
}

func (r *k8sResolver) Scheme() string {
	return r.scheme
}

func (r *k8sResolver) start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		out := r.watcher.Watch()
		for addr := range out {
			r.cc.UpdateState(resolver.State{Addresses: addr})
		}
	}()
}

func (r *k8sResolver) ResolveNow(o resolver.ResolveNowOptions) {
}

func (r *k8sResolver) Close() {
	r.watcher.Close()
	r.wg.Wait()
}

func NewK8sResolver(schema, port string, ns string, sn string, dummyTtl time.Duration) *k8sResolver {
	clientset := InitK8sClient()
	if clientset == nil {
		log.Errorf("Namespace:%s,serviceName:%s,InitK8sClient failed.......\n", ns, sn)
		return nil
	}
	ctx := context.Background()
	return &k8sResolver{
		scheme:      schema,
		clientset:   clientset,
		serviceName: sn,
		ttl:         dummyTtl,
		port:        port,
		// k8sReceiveChan: watchResult.ResultChan(),
		namespace: ns,
		ctx:       ctx,
	}
}

// func RegisterResolver(scheme string, etcdConfig etcd_cli.Config, registryDir, srvName, srvVersion string) {
func RegisterResolver(schema, port string, namespace string, serviceName string, dummyTtl time.Duration) {

	// 尝试新版，若返回nil，尝试旧版
	log.Println("NewResolver start...")
	result := NewK8sResolver(schema, port, namespace, serviceName, dummyTtl)
	if result != nil {
		log.Infof("### Namespace:%s,serviceName:%s, Init K8sSolver Success ...", namespace, serviceName)
	} else {
		log.Errorf("### Namespace:%s,serviceName:%s, Init K8sSolver Failed ...", namespace, serviceName)
	}

	// etcdHost := viper.GetString("etcd_host")
	// if !strings.HasPrefix(etcdHost, "http") {
	// 	panic("etcd_host 格式不正确：" + etcdHost)
	// }

	// addr := "/registry/services/endpoints/" + namespace + "/" + serviceName

	// result = srv.NewGoResolver(port, etcdHost, addr, dummyTtl)

	resolver.Register(result)
}
