package coolq

import (
	"log"
)

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

// loadAllPlugins 加载全部的插件
func loadAllPlugins() {
	// 先全部执行加载函数
	for _, plug := range entries {
		err := plug.Load()
		if err != nil {
			log.Fatalln(err.Error())
		}
	}
	Default.registerAllPlugins()
	// 触发所有插件的onload事件
	go func() {
		for _, plug := range entries {
			plug.OnLoad()
		}
	}()
}

// PluginRegister 插件注册
func PluginRegister(plug Plugin) {
	entries = append(entries, plug)
}
