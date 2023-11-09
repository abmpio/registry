package consul

const (
	// 用来描述实例的产品的meta key
	MetaName_Product = "product"
	// 当注册到consul时，所有基于abmp的框架的都会在meta中加入此属性，以指定注册到consul中的这个服务是中台的实例
	MetaName_IsHostInABMP = "isHostInABMP"
	// 用来描述实例的描述信息的meta key
	MetaName_Description = "description"
	// 用来描述实例的启动时间的meta key
	MetaName_StartTime = "startTime"
	// 用来描述实例的应用版本号的meta key
	MetaName_AppVersion = "appVersion"
	// 用来描述实例的中台框架版本号的meta key
	MetaName_AppFrameworkVersion = "appFrameworkVersion"
	// 用来描述实例的运行环境的版本号的meta key
	MetaName_HostEnvironment = "hostEnvironment"
	// 用来描述实例所提供的web api的http地址的meta key
	MetaName_Http = "http"
	/// 用来描述实例所提供的web api的用于consul健康检查的地址
	MetaName_Healthcheck = "healthcheck"
)
