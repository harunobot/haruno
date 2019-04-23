package plugins

import (
	"plugin"

	"github.com/haruno-bot/haruno/logger"

	"github.com/haruno-bot/haruno/coolq"
)

// SetupPlugins 安装插件的入口
func SetupPlugins(tplugins map[string]string) {
	for pPath, pName := range tplugins {
		p, err := plugin.Open(pPath)
		if err != nil {
			logger.Logger.Fatalln(err)
		}
		_plugin, err := p.Lookup(pName)
		if err != nil {
			logger.Logger.Fatalln(err)
		}
		coolq.PluginRegister(_plugin.(coolq.PluginInterface))
	}
}
