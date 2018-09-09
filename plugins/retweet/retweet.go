package retweet

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/haruno-bot/haruno/logger"

	"github.com/BurntSushi/toml"

	"github.com/haruno-bot/haruno/clients"
	"github.com/haruno-bot/haruno/coolq"
)

// Retweet 转推插件
type Retweet struct {
	coolq.Plugin
	name      string
	url       string
	version   string
	broadcast map[string][]int64
	conn      *clients.WSClient
}

// Name 插件名称
func (_plugin Retweet) Name() string {
	return _plugin.name
}

func removeRepeatedString(arr []string) []string {
	m := make(map[string]bool)
	n := make([]string, 0)
	for _, val := range arr {
		if m[val] {
			continue
		}
		n = append(n, val)
		m[val] = true
	}
	return n
}

func removeRepeatedInteger(arr []int64) []int64 {
	m := make(map[int64]bool)
	n := make([]int64, 0)
	for _, val := range arr {
		if m[val] {
			continue
		}
		n = append(n, val)
		m[val] = true
	}
	return n
}

func (_plugin *Retweet) loadConfig() error {
	cfg := new(Config)
	_, err := toml.DecodeFile("config.toml", cfg)
	if err != nil {
		return err
	}
	pcfg := cfg.Retweet
	_plugin.name = pcfg.Name
	_plugin.url = pcfg.URL
	_plugin.version = pcfg.Version
	_plugin.broadcast = make(map[string][]int64)
	// 确定广播组
	for _, broadcast := range pcfg.Broadcast {
		accounts := make([]string, 0)
		toGroupNums := removeRepeatedInteger(broadcast.GroupNums)
		if broadcast.Account != "" {
			accounts = append(accounts, broadcast.Account)
		}
		for _, account := range broadcast.Accounts {
			accounts = append(accounts, account)
		}
		accounts = removeRepeatedString(accounts)
		for _, account := range accounts {
			if _plugin.broadcast[account] == nil {
				_plugin.broadcast[account] = make([]int64, 0)
			}
			_plugin.broadcast[account] = append(_plugin.broadcast[account], toGroupNums...)
			_plugin.broadcast[account] = removeRepeatedInteger(_plugin.broadcast[account])
		}
	}
	return nil
}

// Load 插件加载
func (_plugin Retweet) Load() error {
	err := _plugin.loadConfig()
	if err != nil {
		return err
	}
	_plugin.conn = &clients.WSClient{
		Name: "Retweet Plugin",
		OnConnect: func(conn *clients.WSClient) {
			msg := fmt.Sprintf("%s has been connected to the twitter api server.", conn.Name)
			logger.Service.AddLog(logger.LogTypeInfo, msg)
			log.Println(msg)
		},
		OnMessage: func(raw []byte) {
			msg := new(TweetMsg)
			err := json.Unmarshal(raw, msg)
			if err != nil {
				logger.Service.AddLog(logger.LogTypeError, err.Error())
				return
			}
			groupNums := _plugin.broadcast[msg.FromID]
			switch msg.Status {
			case "1": // 推文
				for _, groupID := range groupNums {
					coolq.Default.SendGroupMsg(groupID, msg.Text)
				}
			case "2": // 头像
				cqMsg := make(coolq.Message, 0)
				name := msg.FromName
				section := coolq.NewTextSection(fmt.Sprintf("%s 更新了头像\n", name))
				cqMsg = append(cqMsg, section)
				avatar := msg.Avatar
				section = coolq.NewImageSection(avatar)
				cqMsg = append(cqMsg, section)
				data := coolq.Marshal(cqMsg)
				for _, groupID := range groupNums {
					coolq.Default.SendGroupMsg(groupID, string(data))
				}
			}
		},
		OnError: func(err error) {
			msg := err.Error()
			logger.Service.AddLog(logger.LogTypeError, msg)
			log.Println(msg)
		},
	}
	err = _plugin.conn.Dial(_plugin.url, nil)
	if err != nil {
		return err
	}
	return nil
}

// Instance 转推插件实体
var Instance = Retweet{}
