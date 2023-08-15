package k8s

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/DoOR-Team/goutils/djson"
	"github.com/DoOR-Team/goutils/log"
)

type Watcher struct {
	namespace      string
	serviceName    string
	port           string
	clientset      *kubernetes.Clientset
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	addrs          []resolver.Address
	k8sWatchResult watch.Interface
	initFlag       bool
}

func (w *Watcher) Close() {
	w.cancel()
}

func newWatcher(namespace string, serviceName string, port string, cli *kubernetes.Clientset) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{
		namespace:   namespace,
		serviceName: serviceName,
		port:        port,
		clientset:   cli,
		ctx:         ctx,
		cancel:      cancel,
	}
	return w
}

func (w *Watcher) GetAllAddresses() []resolver.Address {

	var res []resolver.Address

	endpoint, err := w.clientset.CoreV1().Endpoints(w.namespace).Get(
		w.ctx,
		w.serviceName, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
		return res
	}
	ep := K8sEndpointInfo{}

	err = json.Unmarshal([]byte(djson.ToJsonString(endpoint)), &ep)
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			res = append(res, resolver.Address{
				Addr: addr.IP + w.port,
			})
		}
	}
	log.Info(w.serviceName, ", InitEndpointsIP:", djson.ToJsonString(res))
	return res
}

func (w *Watcher) DoWatch() {
	watchResult, err := w.clientset.CoreV1().Endpoints(w.namespace).Watch(
		w.ctx,
		metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", w.serviceName).String(),
			Watch:         true,
		})
	if err != nil {
		log.Error("Watch k8s [%s] endpoints Error:%s\n", w.serviceName, err.Error())
		// return nil
	}
	w.k8sWatchResult = watchResult
	// r.k8sReceiveChan = watchResult.ResultChan()
	return
}

func (w *Watcher) Watch() chan []resolver.Address {
	out := make(chan []resolver.Address, 10)
	w.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			w.wg.Done()
		}()

		if !w.initFlag {
			w.addrs = w.GetAllAddresses()
			w.initFlag = true
		}
		out <- w.cloneAddresses(w.addrs)

		log.Info("watching endpoint...")
		watchResult, err := w.clientset.CoreV1().Endpoints(w.namespace).Watch(
			w.ctx,
			metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", w.serviceName).String(),
				Watch:         true,
			})
		if err != nil {
			log.Errorf("Watch k8s [%s] endpoints Error:%s\n", w.serviceName, err.Error())
			return
		}
		w.k8sWatchResult = watchResult

		for {
			select {
			case event, ok := <-w.k8sWatchResult.ResultChan():
				if !ok {
					log.Println("Read Channel error:")
					w.k8sWatchResult.Stop()
					w.DoWatch()
					return
				}

				endpoint := K8sEndpointInfo{}
				if err := json.Unmarshal([]byte(djson.ToJsonString(event.Object)), &endpoint); err != nil {
					log.Error(err)
					return
				}
				var targets []resolver.Address

				for _, subset := range endpoint.Subsets {
					for _, addr := range subset.Addresses {
						targets = append(targets,
							resolver.Address{
								Addr: addr.IP + w.port,
							})
					}
				}
				log.Info("entpoint changed: ", targets)

				if len(targets) == 0 {
					log.Warn(w.serviceName, ", LookUp Ips is empty")
				} else {
					w.addrs = targets
					out <- w.cloneAddresses(w.addrs)
				}
			}
		}

	}()
	return out
}

func (w *Watcher) cloneAddresses(in []resolver.Address) []resolver.Address {
	out := make([]resolver.Address, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = in[i]
	}
	return out
}

func (w *Watcher) addAddr(addr resolver.Address) bool {
	for _, v := range w.addrs {
		if addr.Addr == v.Addr {
			return false
		}
	}
	w.addrs = append(w.addrs, addr)
	return true
}

func (w *Watcher) removeAddr(addr resolver.Address) bool {
	for i, v := range w.addrs {
		if addr.Addr == v.Addr {
			w.addrs = append(w.addrs[:i], w.addrs[i+1:]...)
			return true
		}
	}
	return false
}
