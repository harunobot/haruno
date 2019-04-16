package plugins

import (
	"github.com/haruno-bot/enshuhelper"
	"github.com/haruno-bot/haruno/coolq"
	"github.com/haruno-bot/retweet"
)

// SetupPbPlugins 安装插件的入口
func SetupPbPlugins() {
	coolq.PbPluginRegister(retweet.Instance, enshuhelper.Instance)
}
