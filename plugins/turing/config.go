package turing

// Config 插件总的入口
type Config struct {
	Turing Cfg `toml:"turing"`
}

// Cfg 插件配置信息
type Cfg struct {
	Name      string  `toml:"name"`
	Token     string  `toml:"token"`
	Version   string  `toml:"version"`
	GroupNums []int64 `toml:"groupNums"`
}
