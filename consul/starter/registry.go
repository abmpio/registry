package starter

import (
	"strconv"
	"time"

	"github.com/abmpio/abmp/pkg/log"
	"github.com/abmpio/app"
	"github.com/abmpio/app/host"
	"github.com/abmpio/configurationx"
	"github.com/abmpio/configurationx/options/consul"
	registryConsul "github.com/abmpio/registry/consul"
	"go.uber.org/zap"
)

var (
	_registry *registryConsul.Registry
)

func init() {
	//注册并在最后执行，因为有些服务需要端口启动后才能注册，否则一注册consul就会做健康检查
	//这样容易导致被consul注销
	app.RegisterOneStartupAction(registryStartupAction).SetName("consul.registry").SetLast()
	app.RegisterOneShutdown(registryShutdown)
}

func registryStartupAction() app.IStartupAction {
	return app.NewStartupAction(func() {
		if app.HostApplication.SystemConfig().App.IsRunInCli {
			return
		}
		_registry = app.Context.GetInstance(new(registryConsul.Registry)).(*registryConsul.Registry)
		if _registry == nil {
			log.Logger.Warn("无法注册服务到注册中心中,应用没有import github.com/abmpio/registry/consul包")
			return
		}
		consul := configurationx.GetInstance().Consul
		if !consul.Registration.Enabled {
			log.Logger.Warn("consul.registration.enabled参数为false,已禁用注册服务到注册中心中")
			return
		}
		log.Logger.Info("准备注册服务到注册中心中...")
		for {
			setupServiceRegistryInfo(consul.Registration)
			err := _registry.Register(consul.Registration)
			if err == nil {
				log.Logger.Info("已成功注册所有服务到注册中心")
				break
			}
			log.Logger.Error("将服务注册到consul注册中心时出现异常, 3秒后将重试...", zap.Error(err))
			time.Sleep(3 * time.Second)
		}
	})
}

// 关闭时注销已经注册的服务
func registryShutdown() app.IShutdownAction {
	return app.NewShutdownAction(func() {
		if app.HostApplication.SystemConfig().App.IsRunInCli {
			return
		}
		if _registry != nil {
			_registry.DeregisterAll()
		}
	})
}

func setupServiceRegistryInfo(registryInfo *consul.RegistrationInfo) {
	if registryInfo.Meta == nil {
		registryInfo.Meta = make(map[string]string)
	}
	//构建meta
	setServiceMeta(registryInfo.Meta)
}

func setServiceMeta(meta map[string]string) {
	appVer := host.GetHostEnvironment().GetEnvString(host.ENV_AppVersion)
	if len(appVer) > 0 {
		meta[registryConsul.MetaName_AppVersion] = appVer
	}
	frameworkVer := host.GetHostEnvironment().GetEnvString(host.ENV_FrameworkVersion)
	if len(frameworkVer) > 0 {
		meta[registryConsul.MetaName_AppFrameworkVersion] = frameworkVer
	}
	hostEnv := host.GetHostEnvironment().GetEnvString(host.ENV_HostEnvironment)
	if len(hostEnv) > 0 {
		meta[registryConsul.MetaName_HostEnvironment] = hostEnv
	}
	pro := host.GetHostEnvironment().GetEnvString(host.ENV_Product)
	if len(pro) > 0 {
		meta[registryConsul.MetaName_Product] = pro
	}
	desc := host.GetHostEnvironment().GetEnvString(host.ENV_Description)
	if len(desc) > 0 {
		meta[registryConsul.MetaName_Description] = desc
	}
	startTime, ok := host.GetHostEnvironment().GetEnv(host.ENV_StartTime).(time.Time)
	if ok {
		meta[registryConsul.MetaName_StartTime] = startTime.Format(time.RFC3339)
	}
	hostInABMP, ok := host.GetHostEnvironment().GetEnv(host.ENV_IsHostInABMP).(bool)
	if ok {
		meta[registryConsul.MetaName_IsHostInABMP] = strconv.FormatBool(hostInABMP)
	}
	http := host.GetHostEnvironment().GetEnvString(host.ENV_HTTP)
	if len(http) > 0 {
		meta[registryConsul.MetaName_Http] = http
	}
	healthcheck := host.GetHostEnvironment().GetEnvString(host.ENV_Healthcheck)
	if len(healthcheck) > 0 {
		meta[registryConsul.MetaName_Healthcheck] = healthcheck
	}
}
