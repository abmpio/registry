package consul

import (
	"fmt"
	"sync"

	"github.com/abmpio/abmp/pkg/log"
	"github.com/abmpio/configurationx"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"go.uber.org/zap"
)

type ServiceUpdateHandler func(*consulapi.ServiceEntry)

type Watcher struct {
	endpoint string
	// service变化时
	wp *watch.Plan
	//监控的service服务
	serviceUpdateHandler map[string][]ServiceUpdateHandler
	watchers             map[string]*watch.Plan
	rwMutex              sync.RWMutex

	running bool
}

func newServiceWatch(params map[string]string) (*Watcher, error) {
	consul := configurationx.GetInstance().Consul
	endpoint := fmt.Sprintf("%s:%d", consul.Host, consul.Port)

	return newWatcher("services", params, endpoint)
}

func newWatcher(watchType string, params map[string]string, consulEndpoint string) (*Watcher, error) {
	//要监控的类型,如key,service等
	var options = map[string]interface{}{
		"type": watchType,
	}
	for k, v := range params {
		options[k] = v
	}

	wp, err := watch.Parse(options)
	if err != nil {
		log.Logger.Error("在请求监控consul时无效的参数", zap.Error(err))
		return nil, err
	}
	w := &Watcher{
		endpoint:             consulEndpoint,
		wp:                   wp,
		serviceUpdateHandler: make(map[string][]ServiceUpdateHandler),
		watchers:             make(map[string]*watch.Plan),
	}
	wp.Handler = w.watchHandler
	return w, nil
}

// 用来处理服务发生改变的回调
func (w *Watcher) watchHandler(u uint64, data interface{}) {
	switch d := data.(type) {
	case *consulapi.KVPair:
		//TODO:key
	case []*consulapi.Node:
		//TODO: nodes
	case []*consulapi.HealthCheck:
		//TODO:healthcheck
	case map[string][]string:
		//services
		for serviceName := range d {
			// 如果该service已经监控了，则不再启动监控了
			// 忽略掉consul
			if _, ok := w.watchers[serviceName]; ok || serviceName == "consul" {
				continue
			}
			//检测服务是否正在被监控
			if !w.IsWatcher(serviceName) {
				continue
			}
			w.registerServiceWatcher(serviceName)
		}

		w.rwMutex.RLock()
		watches := w.watchers
		w.rwMutex.RUnlock()

		for i, svc := range watches {
			if _, ok := d[i]; !ok {
				svc.Stop()
				delete(watches, i)
			}
		}
	}
}

func (w *Watcher) Run() error {
	if w.running {
		//已经启动，则直接返回
		return nil
	}
	if err := w.wp.Run(w.endpoint); err != nil {
		log.Logger.Error("启动service监控时出现异常", zap.Error(err))
		return err
	}
	w.running = true
	return nil
}

func (w *Watcher) Stop() {
	if w.wp == nil {
		return
	}
	w.wp.Stop()
	w.running = false
}

// 注册一个服务状态发生改变时的事件
func (w *Watcher) RegistHandler(serviceName string, handler ServiceUpdateHandler) {
	if len(serviceName) <= 0 || handler == nil {
		return
	}
	handlerList, ok := w.serviceUpdateHandler[serviceName]
	if !ok {
		//还没有注册，则创建
		handlerList = make([]ServiceUpdateHandler, 0)
	}
	handlerList = append(handlerList, handler)
	w.serviceUpdateHandler[serviceName] = handlerList
}

// 检测指定的服务watcher是否正在启动
func (w *Watcher) IsWatcher(serviceName string) bool {
	_, ok := w.serviceUpdateHandler[serviceName]
	return ok
}

func (w *Watcher) notifyServiceUpdate(entry *consulapi.ServiceEntry) {
	handlerList := w.serviceUpdateHandler[entry.Service.Service]
	if len(handlerList) <= 0 {
		return
	}
	for _, eachHandler := range handlerList {
		eachHandler(entry)
	}
}

// 监控服务变化
func (w *Watcher) registerServiceWatcher(serviceName string) error {
	wp, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	})
	if err != nil {
		return err
	}
	wp.Handler = func(idx uint64, data interface{}) {
		switch serviceEntryList := data.(type) {
		case []*consulapi.ServiceEntry:
			for _, eachServiceEntry := range serviceEntryList {
				// service发生改变
				log.Logger.Info(fmt.Sprintf("service %s 已变化,status:%s", eachServiceEntry.Service.Service, eachServiceEntry.Checks.AggregatedStatus()))
				w.notifyServiceUpdate(eachServiceEntry)
			}
		}
	}
	//启动监控
	go wp.Run(w.endpoint)
	//保存
	w.rwMutex.Lock()
	w.watchers[serviceName] = wp
	w.rwMutex.Unlock()

	return nil
}

// func RegisterWatcher(watchType string, opts map[string]string) (*Watcher, error) {
// 	consul := configuration.GetInstance().Consul
// 	endpoint := fmt.Sprintf("%s:%d", consul.Host, consul.Port)
// 	//新建一个watcher
// 	watcher, err := newWatcher(watchType, opts, endpoint)
// 	if err != nil {
// 		log.Logger.Error("无法创建consul的watcher", zap.Error(err))
// 		return nil, err
// 	}
// 	defer watcher.wp.Stop()

// 	if err = watcher.wp.Run(endpoint); err != nil {
// 		log.Logger.Error("启动service监控时出现异常", zap.Error(err))
// 		return nil, err
// 	}

// 	return watcher, nil
// }

// // service监控
// func RegisterServiceWatcher(opts map[string]string) (*Watcher, error) {
// 	return RegisterWatcher("services", opts)
// }

// func (w *Watcher) services() []*ServiceRegistrationInfo {
// 	return w.serviceInstances.Load().([]*ServiceRegistrationInfo)
// }

// func (w *Watcher) update(services []*ServiceRegistrationInfo) {
// 	w.serviceInstances.Store(services)
// 	w.event <- struct{}{}
// }

// // Next 返回服务实例列表
// func (w *Watcher) Next() (services []*ServiceRegistrationInfo, err error) {
// 	select {
// 	case <-w.ctx.Done():
// 		err = w.ctx.Err()
// 	case <-w.event:
// 		if ss, ok := w.serviceInstances.Load().([]*ServiceRegistrationInfo); ok {
// 			services = append(services, ss...)
// 		}
// 	}
// 	return
// }

// // Stop 停止监听
// func (w *Watcher) Stop() error {
// 	w.cancel()
// 	close(w.event)
// 	return nil
// }
