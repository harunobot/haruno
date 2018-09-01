package coolq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/haruno-bot/haruno/clients"
	"github.com/haruno-bot/haruno/logger"
)

const timeForWait = 30

const noFilterKey = "__DO_NOT_SET_UNUSED_KEY__"

// Filter 过滤函数
type Filter func([]byte) bool

// Handler 处理函数
type Handler func([]byte)

type pluginEntry struct {
	fitlers  map[string]Filter
	handlers map[string]Handler
}

// cqclient 酷q机器人连接客户端
// 为了安全起见，暂时不允许在包外额外创建
type cqclient struct {
	mu            sync.Mutex
	apiConn       *clients.WSClient
	eventConn     *clients.WSClient
	pluginEntries map[string]pluginEntry
	echoqueue     map[int64]bool
}

func handleConnect(conn *clients.WSClient) {
	if conn.IsConnected() {
		msgText := fmt.Sprintf("酷Q机器人%s服务已成功连接！", conn.Name)
		connMsg := logger.NewLog(logger.LogTypeInfo, msgText)
		logger.Service.Add(connMsg)
	}
}

func handleError(err error) {
	msgText := err.Error()
	errMsg := logger.NewLog(logger.LogTypeError, msgText)
	logger.Service.Add(errMsg)
}

func (c *cqclient) registerAllPlugins() {
	// 先全部执行加载函数
	for _, plug := range entries {
		err := plug.Load()
		if err != nil {
			log.Fatalln(err.Error())
		}
	}
	// 注册所有的handler和filter
	for _, plug := range entries {
		pluginName := plug.Name()
		pluginFilters := plug.Filters()
		pluginHandlers := plug.Hanlders()
		hasFilter := make(map[string]bool)
		entry := pluginEntry{
			fitlers:  make(map[string]Filter),
			handlers: make(map[string]Handler),
		}
		noFilterHanlers := make([]Handler, 0)
		for key, filter := range pluginFilters {
			handler := pluginHandlers[key]
			if handler == nil {
				fmt.Printf("[WARN] 插件 %s 中存在没有使用的key: %s\n", pluginName, key)
				continue
			}
			hasFilter[key] = true
			entry.fitlers[key] = filter
			entry.handlers[key] = handler
		}
		for key, handler := range pluginHandlers {
			if !hasFilter[key] {
				noFilterHanlers = append(noFilterHanlers, handler)
			}
		}
		entry.handlers[noFilterKey] = func(data []byte) {
			for _, hanldeFunc := range noFilterHanlers {
				hanldeFunc(data)
			}
		}
		c.pluginEntries[pluginName] = entry
	}
	// 触发所有插件的onload事件
	go func() {
		for _, plug := range entries {
			plug.OnLoad()
		}
	}()
}

func (c *cqclient) Initialize() {
	c.apiConn.Name = "API"
	c.eventConn.Name = "Event"
	c.registerAllPlugins()
	// handle connect
	c.apiConn.OnConnect = handleConnect
	c.eventConn.OnConnect = handleConnect
	// handle error
	c.apiConn.OnError = handleError
	c.eventConn.OnError = handleError
	// handle message
	c.apiConn.OnMessage = func(rmsg []byte) {
		jmsg := make(map[string]interface{})
		err := json.Unmarshal(rmsg, &jmsg)
		if err != nil {
			logger.Service.AddLog(logger.LogTypeError, err.Error())
			return
		}
		echo := jmsg["echo"].(int64)
		if c.echoqueue[echo] {
			c.mu.Lock()
			delete(c.echoqueue, echo)
			c.mu.Unlock()
		}
	}
	// handle events
	c.eventConn.OnMessage = func(raw []byte) {
		for _, entry := range c.pluginEntries {
			entry.handlers[noFilterKey](raw)
			for key, filterFunc := range entry.fitlers {
				handleFunc := entry.handlers[key]
				if filterFunc(raw) {
					handleFunc(raw)
				}
			}
		}
	}

	// 定时清理echo序列
	go func() {
		now := time.Now().Unix()
		for echo, state := range c.echoqueue {
			if state && now-echo > timeForWait {
				logger.Service.AddLog(logger.LogTypeError, fmt.Sprintf("[%d]响应超时(30s).", echo))
				c.mu.Lock()
				delete(c.echoqueue, echo)
				c.mu.Unlock()
			}
		}
		time.Sleep(timeForWait)
	}()
}

// Connect 连接远程酷q api服务
// url 形如 ws://127.0.0.1:8080, wss://127.0.0.1:8080之类的url
// token 酷q机器人的access token
func (c *cqclient) Connect(url string, token string) {
	headers := make(http.Header)
	headers.Add("Authorization", fmt.Sprintf("Token %s", token))
	// 连接api服务和事件服务
	c.apiConn.Dial(fmt.Sprintf("%s/api", url), headers)
	c.eventConn.Dial(fmt.Sprintf("%s/event", url), headers)
}

// Default 唯一的酷q机器人实体
var Default = &cqclient{
	apiConn:       &clients.WSClient{},
	eventConn:     &clients.WSClient{},
	pluginEntries: make(map[string]pluginEntry),
}
