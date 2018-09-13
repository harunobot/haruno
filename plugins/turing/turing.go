package turing

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/haruno-bot/haruno/logger"

	"github.com/haruno-bot/haruno/clients"

	"github.com/BurntSushi/toml"

	"github.com/haruno-bot/haruno/coolq"
)

var groupNums = make(map[int64]bool)
var token string
var client = clients.NewHTTPClient("")
var name string
var version string

// 没有问题的回答
var unReply = coolq.NewTextSection("我听不清，你在说什么呀？")

// Turing 结合图灵机器人api的插件
type Turing struct {
	coolq.Plugin
}

// Name 插件名字+版本号
func (_plugin Turing) Name() string {
	return fmt.Sprintf("%s@%s", name, version)
}

func (_plugin *Turing) loadConfig() error {
	cfg := new(Config)
	toml.DecodeFile("cofig.toml", cfg)
	_, err := toml.DecodeFile("config.toml", cfg)
	if err != nil {
		return err
	}
	pcfg := cfg.Turing
	name = pcfg.Name
	version = pcfg.Version
	token = pcfg.Token
	for _, groupID := range pcfg.GroupNums {
		groupNums[groupID] = true
	}
	return nil
}

// Filters 过滤酷Q上报事件用，利于提升插件性能
func (_plugin Turing) Filters() map[string]coolq.Filter {
	filters := make(map[string]coolq.Filter)
	filters["turing"] = func(event *coolq.CQEvent) bool {
		if event.PostType != "message" ||
			event.MessageType != "group" ||
			event.SubType != "normal" {
			return false
		}
		if !groupNums[event.GroupID] {
			return false
		}
		msg := new(coolq.Message)
		err := coolq.Unmarshal([]byte(event.Message), msg)
		if err != nil {
			fmt.Println(err)
		}
		for _, section := range *msg {
			if section.Type == "at" {
				qqNum, _ := strconv.ParseInt(section.Data["qq"], 10, 64)
				if qqNum == event.SelfID {
					return true
				}
			}
		}
		return false
	}
	return filters
}

// Handlers 处理酷Q上报事件用
func (_plugin Turing) Handlers() map[string]coolq.Handler {
	handlers := make(map[string]coolq.Handler)
	handlers["turing"] = func(event *coolq.CQEvent) {
		msg := new(coolq.Message)
		err := coolq.Unmarshal([]byte(event.Message), msg)
		if err != nil {
			return
		}
		var question string
		for _, section := range *msg {
			if section.Type == "text" {
				question = strings.TrimSpace(section.Data["text"])
				if len(question) > 0 {
					break
				}
			}
		}
		reply := coolq.NewMessage()
		replto := coolq.NewSection("at", map[string]string{
			"qq": fmt.Sprintf("%d", event.UserID),
		})
		reply = coolq.AddSection(reply, replto)

		if len(question) > 0 {
			qsURL := fmt.Sprintf("http://www.tuling123.com/openapi/api?key=%s&info=%s&userid=%d", token, url.QueryEscape(question), event.UserID)
			res, err := client.Get(qsURL)
			if err != nil {
				errMsg := err.Error()
				fmt.Println(errMsg)
				logger.Service.AddLog(logger.LogTypeError, errMsg)
				return
			}
			defer res.Body.Close()
			ans := new(Answer)
			err = json.NewDecoder(res.Body).Decode(ans)
			if err != nil {
				errMsg := err.Error()
				fmt.Println(errMsg)
				logger.Service.AddLog(logger.LogTypeError, errMsg)
				return
			}
			var text string
			if ans.Code == 100000 {
				text = ans.Text
			} else {
				text = "？"
			}
			reply = coolq.AddSection(reply, coolq.NewTextSection(text))
			replyMsg := string(coolq.Marshal(reply))
			coolq.Client.SendGroupMsg(event.GroupID, replyMsg)
			logMsg := fmt.Sprintf("向酷Q发送：%s", replyMsg)
			log.Println(logMsg)
			logger.Service.AddLog(logger.LogTypeSuccess, logMsg)
		} else {
			reply = coolq.AddSection(reply, unReply)
			replyMsg := string(coolq.Marshal(reply))
			coolq.Client.SendGroupMsg(event.GroupID, replyMsg)
		}
	}
	return handlers
}

// Load 加载插件
func (_plugin Turing) Load() error {
	return _plugin.loadConfig()
}

// Loaded 加载完成
func (_plugin Turing) Loaded() {
	logMsg := fmt.Sprintf("%s已成功加载", _plugin.Name())
	log.Println(logMsg)
	logger.Service.AddLog(logger.LogTypeInfo, logMsg)
}

// Instance 实体
var Instance = Turing{}
