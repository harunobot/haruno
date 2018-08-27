package coolq

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/haruno-bot/haruno/clients"
	"github.com/haruno-bot/haruno/logger"
)

const timeForWait = 30

// Filter 过滤函数
type Filter func([]byte) bool

// Handler 处理函数
type Handler func([]byte)

// cqclient 酷q机器人连接客户端
// 为了安全起见，暂时不允许在包外额外创建
type cqclient struct {
	mu        sync.Mutex
	apiConn   *clients.WSClient
	eventConn *clients.WSClient
	filters   map[string]Filter
	handlers  map[string]Handler
	echoqueue map[int64]bool
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

func (c *cqclient) Initialize() {
	c.filters = make(map[string]Filter)
	c.handlers = make(map[string]Handler)
	c.apiConn.Name = "API"
	c.eventConn.Name = "Event"
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
	c.eventConn.OnMessage = func(rmsg []byte) {
		for handlerName, handle := range c.handlers {
			filter := c.filters[handlerName]
			ok := true
			if filter != nil && !filter(rmsg) {
				ok = false
			}
			if ok {
				handle(rmsg)
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
	apiConn:   &clients.WSClient{},
	eventConn: &clients.WSClient{},
	filters:   make(map[string]Filter),
	handlers:  make(map[string]Handler),
}
