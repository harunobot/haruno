package coolq

var entries = []pluginInterface{}

// pluginInterface 插件基础接口
// 插件必须实现 Load 方法，以过滤器和处理器为参数
// 完成load会执行 Onload 钩子函数
type pluginInterface interface {
	Name() string
	Load() error
	Filters() map[string]Filter
	Handlers() map[string]Handler
	Loaded()
}

// PluginRegister 插件注册
func PluginRegister(plugins ...pluginInterface) {
	entries = append(entries, plugins...)
}

// Plugin 插件基础原型
type Plugin struct {
}

// Name 获取插件名字
func (_plugin Plugin) Name() string {
	return "__UNNAMED_PLUGIN__"
}

// Load 插件加载
func (_plugin Plugin) Load() error {
	return nil
}

// Filters 插件过滤器
func (_plugin Plugin) Filters() map[string]Filter {
	return nil
}

// Handlers 插件过滤器
func (_plugin Plugin) Handlers() map[string]Handler {
	return nil
}

// Loaded 加载完成的事件
func (_plugin Plugin) Loaded() {
}
