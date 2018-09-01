package coolq

var entries = []Plugin{}

// Plugin 插件基础接口
// 插件必须实现 Load 方法，以过滤器和处理器为参数
// 完成load会执行 Onload 钩子函数
type Plugin interface {
	Name() string
	Load() error
	Filters() map[string]Filter
	Hanlders() map[string]Handler
	OnLoad()
}

// PluginRegister 插件注册
func PluginRegister(plug Plugin) {
	entries = append(entries, plug)
}
