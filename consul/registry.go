package consul

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/abmpio/abmp/pkg/log"
	"github.com/abmpio/abmp/pkg/utils/lang"
	"github.com/abmpio/configurationx"
	"github.com/abmpio/configurationx/options/consul"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

const (
	checkIDFormat     = "service:%s"
	checkUpdateOutput = "passed"
)

type Registry struct {
	err    error
	ctx    context.Context
	cancel context.CancelFunc
	client *api.Client
	opts   *Options

	mu      sync.Mutex
	watcher *Watcher
	//已经注册成功的服务id列表
	regisedIdList []string
}

// 初始化结构
func NewRegistry(opts ...Option) *Registry {
	consulOptions := configurationx.GetInstance().Consul
	endpoint := fmt.Sprintf("%s:%d", consulOptions.Host, consulOptions.Port)
	opts = append(opts, WithAddress(endpoint))
	o := NewOptions(opts...)

	config := api.DefaultConfig()
	if o.address != "" {
		config.Address = o.address
	}

	r := &Registry{
		regisedIdList: make([]string, 0),
	}
	r.opts = o
	r.ctx, r.cancel = context.WithCancel(o.ctx)
	r.client, r.err = api.NewClient(config)

	return r
}

// Register 注册服务实例
func (r *Registry) Register(info *consul.RegistrationInfo) error {
	if r.err != nil {
		return r.err
	}
	if len(info.Endpoint) <= 0 {
		return fmt.Errorf("Endpoint参数不能为空")
	}

	if len(info.Endpoint) <= 0 {
		return errors.New("必须配置好endpoint参数")
	}
	serviceAddressList, err := info.ParseServiceAddress()
	if err != nil {
		return err
	}
	httpServiceAddress, err := info.ParseServiceAddressForScheme("http")
	if err != nil {
		return err
	}
	if httpServiceAddress == nil {
		return fmt.Errorf("endpoint必须配置好http类型的值")
	}
	registration := &api.AgentServiceRegistration{
		ID:      info.ID,
		Name:    info.ServiceName,
		Meta:    info.Meta,
		Address: httpServiceAddress.Address,
		Port:    httpServiceAddress.Port,
		Tags:    info.Tags,
	}
	//设置Tagged Address
	registration.TaggedAddresses = make(map[string]api.ServiceAddress)
	for eachKey, eachAddress := range serviceAddressList {
		registration.TaggedAddresses[eachKey] = api.ServiceAddress{
			Address: eachAddress.Address,
			Port:    eachAddress.Port,
		}
	}

	if r.opts.enableHealthCheck {
		registration.Checks = append(registration.Checks, &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", r.opts.deregisterCriticalServiceAfter),
			Interval:                       fmt.Sprintf("%ds", r.opts.healthCheckInterval),
			Status:                         api.HealthPassing,
			HTTP:                           info.HealthCheckHTTP,
			TCP: lang.IfValue(len(info.HealthCheckTCP) > 0, func() string {
				return info.HealthCheckTCP
			}, ""),
			Timeout: fmt.Sprintf("%ds", r.opts.healthCheckTimeout),
		})
	}

	if r.opts.enableHeartbeatCheck {
		registration.Checks = append(registration.Checks, &api.AgentServiceCheck{
			CheckID:                        fmt.Sprintf(checkIDFormat, info.ID),
			TTL:                            fmt.Sprintf("%ds", r.opts.heartbeatCheckInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", r.opts.deregisterCriticalServiceAfter),
		})
	}

	if err := r.client.Agent().ServiceRegister(registration); err != nil {
		log.Logger.Error(fmt.Sprintf("注册服务到注册中心时出现异常,serviceName:%s,endPoint:%s", registration.Name, info.Endpoint),
			zap.Error(err))
		return err
	}
	log.Logger.Info(fmt.Sprintf("已成功注册服务到注册中心,serviceName:%s,endPoint:%s", registration.Name, info.Endpoint))

	//保存已经注册的服务
	r.regisedIdList = append(r.regisedIdList, registration.ID)
	if r.opts.enableHeartbeatCheck {
		go r.heartbeat(info.ID)
	}

	return nil
}

// 注销所有已经注销的服务
func (r *Registry) DeregisterAll() {
	if len(r.regisedIdList) <= 0 {
		return
	}
	for _, eachId := range r.regisedIdList {
		err := r.Deregister(eachId)
		if err != nil {
			log.Logger.Error(fmt.Sprintf("在注销consul的服务时出现异常,serviceId:%s", eachId), zap.Error(err))
			continue
		}
		log.Logger.Info(fmt.Sprintf("已成功注销consul的服务,serviceId:%s", eachId))
	}
}

// Deregister 注销服务实例
func (r *Registry) Deregister(id string) error {
	if len(id) <= 0 {
		return nil
	}
	r.cancel()
	return r.client.Agent().ServiceDeregister(id)
}

// Services 获取服务实例列表
func (r *Registry) Services(ctx context.Context, serviceName string) ([]*consul.RegistrationInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	services, _, err := r.services(ctx, serviceName, 0, true)
	return services, err
}

// watch一个service
func (r *Registry) WatchService(serviceName string, handler ServiceUpdateHandler) {
	if r.watcher == nil {
		//用来监控服务
		watcher, _ := newServiceWatch(nil)
		r.watcher = watcher
	}
	if r.watcher == nil {
		return
	}

	r.watcher.RegistHandler(serviceName, handler)
	//启动监控
	go r.watcher.Run()
}

func (r *Registry) services(ctx context.Context, serviceName string, waitIndex uint64, passingOnly bool) (
	[]*consul.RegistrationInfo, uint64, error) {
	opts := &api.QueryOptions{
		WaitIndex: waitIndex,
		WaitTime:  60 * time.Second,
	}
	opts.WithContext(ctx)

	entries, meta, err := r.client.Health().Service(serviceName, "", passingOnly, opts)
	if err != nil {
		return nil, 0, err
	}

	services := make([]*consul.RegistrationInfo, 0, len(entries))
	for _, entry := range entries {

		var endpointList []string
		for scheme, addr := range entry.Service.TaggedAddresses {
			if scheme == "lan_ipv4" || scheme == "wan_ipv4" || scheme == "lan_ipv6" || scheme == "wan_ipv6" {
				continue
			}
			endpointList = append(endpointList, (&url.URL{
				Scheme: scheme,
				Host:   net.JoinHostPort(addr.Address, strconv.Itoa(addr.Port)),
			}).String())
		}
		if len(endpointList) <= 0 {
			continue
		}

		//TODO:
		currentRegistrationInfo := &consul.RegistrationInfo{
			ID:          entry.Service.ID,
			ServiceName: entry.Service.Service,
			Meta:        entry.Service.Meta,
			Endpoint:    endpointList,
			Tags:        entry.Service.Tags,
		}
		currentRegistrationInfo.Product = entry.Service.Meta[MetaName_Product]
		services = append(services, currentRegistrationInfo)
	}

	return services, meta.LastIndex, nil
}

// 心跳
func (r *Registry) heartbeat(insID string) {
	time.Sleep(time.Second)

	checkID := fmt.Sprintf(checkIDFormat, insID)

	err := r.client.Agent().UpdateTTL(checkID, checkUpdateOutput, api.HealthPassing)
	if err != nil {
		log.Errorf("update heartbeat ttl failed: %v", err)
	}

	ticker := time.NewTicker(time.Duration(r.opts.heartbeatCheckInterval) * time.Second / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err = r.client.Agent().UpdateTTL(checkID, checkUpdateOutput, api.HealthPassing); err != nil {
				log.Errorf("update heartbeat ttl failed: %v", err)
			}
		case <-r.ctx.Done():
			return
		}
	}
}

// func (r *Registry) normialize(i consul.RegistrationInfo) error {
// 	if len(i.Endpoint) <= 0 {
// 		return fmt.Errorf("无效的endpoint参数值,值不能为空")
// 	}
// 	for _, eachEndPoint := range i.Endpoint {
// 		scheme, serviceAddress, err := r.getServiceAddress(eachEndPoint)
// 		if err != nil {
// 			return fmt.Errorf("无效的endpoint配置值:%s", eachEndPoint)
// 		}
// 		if len(i.ID) <= 0 {
// 			if strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https") {
// 				i.ID = strings.Join([]string{serviceAddress.Address, strconv.Itoa(serviceAddress.Port)}, "_")
// 			}
// 		}
// 	}
// 	return nil
// }
