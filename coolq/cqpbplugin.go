package coolq

var pbentries = []pbPluginInterface{}

// pbPluginInterface 插件基础接口
// 插件必须实现 Load 方法，以过滤器和处理器为参数
// 完成load会执行 Onload 钩子函数
type pbPluginInterface interface {
	pluginInterface
	Module() string
	Callback(data []byte)
}

// PbPluginRegister 插件注册
func PbPluginRegister(plugins ...pbPluginInterface) {
	pbentries = append(pbentries, plugins...)
}

// PbPlugin 插件基础原型
type PbPlugin struct {
}

// Name 获取插件名字
func (_plugin PbPlugin) Name() string {
	return "__UNNAMED_PLUGIN__"
}

// Load 插件加载
func (_plugin PbPlugin) Load() error {
	return nil
}

// Filters 插件过滤器
func (_plugin PbPlugin) Filters() map[string]Filter {
	return nil
}

// Handlers 插件过滤器
func (_plugin PbPlugin) Handlers() map[string]Handler {
	return nil
}

// Loaded 加载完成的事件
func (_plugin PbPlugin) Loaded() {
}

// Module 获取插件归属名字
func (_plugin PbPlugin) Module() string {
	return "__UNNAMED_PLUGIN__"
}

// Callback 推送回调的事件
func (_plugin PbPlugin) Callback(data []byte) {
}
