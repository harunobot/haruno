package plugins

import (
	"github.com/haruno-bot/haruno/coolq"
	"github.com/haruno-bot/haruno/plugins/turing"
)

// SetupPlugins 安装插件的入口
func SetupPlugins() {
	coolq.PluginRegister(turing.Instance)
}
