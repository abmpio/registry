package consul

import "context"

type Option func(o *Options)

type Options struct {
	ctx                            context.Context // context
	address                        string          // consul地址，默认127.0.0.1:8500
	enableHealthCheck              bool            // 是否启用健康检查
	healthCheckInterval            int             // 健康检查时间间隔，默认10秒
	healthCheckTimeout             int             // 健康检查超时时间，默认5秒
	enableHeartbeatCheck           bool            // 是否启用心跳检查
	enableWatcher                  bool            //是否启用watcher
	heartbeatCheckInterval         int             // 心跳检查时间间隔，默认10秒
	deregisterCriticalServiceAfter int             // 健康检测失败后自动注销服务时间，默认5秒
}

func NewOptions(opts ...Option) *Options {
	o := &Options{
		ctx:                            context.Background(),
		enableHealthCheck:              true,
		healthCheckInterval:            10,
		healthCheckTimeout:             5,
		enableHeartbeatCheck:           false,
		enableWatcher:                  false,
		heartbeatCheckInterval:         10,
		deregisterCriticalServiceAfter: 5,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithContext 设置context
func WithContext(ctx context.Context) Option {
	return func(o *Options) { o.ctx = ctx }
}

// WithAddress 设置consul地址
func WithAddress(address string) Option {
	return func(o *Options) { o.address = address }
}

// WithEnableHealthCheck 设置是否启用健康检查
func WithEnableHealthCheck(enable bool) Option {
	return func(o *Options) { o.enableHealthCheck = enable }
}

// WithHealthCheckInterval 设置健康检查时间间隔
func WithHealthCheckInterval(interval int) Option {
	return func(o *Options) { o.healthCheckInterval = interval }
}

// WithHealthCheckTimeout 设置健康检查超时时间
func WithHealthCheckTimeout(timeout int) Option {
	return func(o *Options) { o.healthCheckTimeout = timeout }
}

// WithEnableHeartbeatCheck 设置是否启用心跳检查
func WithEnableHeartbeatCheck(enable bool) Option {
	return func(o *Options) { o.enableHeartbeatCheck = enable }
}

// WithHeartbeatCheckInterval 设置心跳检查时间间隔
func WithHeartbeatCheckInterval(interval int) Option {
	return func(o *Options) { o.heartbeatCheckInterval = interval }
}

// 是否启用监控
func WithEnableWatcher(enableWatcher bool) Option {
	return func(o *Options) { o.enableWatcher = enableWatcher }
}

// WithDeregisterCriticalServiceAfter 设置健康检测失败后自动注销服务时间
func WithDeregisterCriticalServiceAfter(after int) Option {
	return func(o *Options) { o.deregisterCriticalServiceAfter = after }
}
