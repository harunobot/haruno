package coolq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/haruno-bot/haruno/clients"
	"github.com/haruno-bot/haruno/logger"
)

const timeForWait = 30

const noFilterKey = "__NEVER_SET_UNUSED_KEY__"

// Filter 过滤函数
type Filter func(*CQEvent) bool

// Handler 处理函数
type Handler func(*CQEvent)

type pluginEntry struct {
	fitlers  map[string]Filter
	handlers map[string]Handler
}

// cqclient 酷q机器人连接客户端
// 为了安全起见，暂时不允许在包外额外创建
type cqclient struct {
	mu            sync.Mutex
	token         string
	apiConn       *clients.WSClient
	eventConn     *clients.WSClient
	httpConn      *clients.HTTPClient
	apiURL        string
	pluginEntries map[string]pluginEntry
	echoqueue     map[int64]bool
}

func handleConnect(conn *clients.WSClient) {
	if conn.IsConnected() {
		msgText := fmt.Sprintf("%s服务已成功连接！", conn.Name)
		logger.Service.AddLog(logger.LogTypeInfo, msgText)
	}
}

func handleError(err error) {
	msgText := err.Error()
	errMsg := logger.NewLog(logger.LogTypeError, msgText)
	logger.Service.Add(errMsg)
}

func (c *cqclient) registerAllPlugins() {
	// 先全部执行加载函数
	loaded := make([]pluginInterface, 0)
	for _, plug := range entries {
		err := plug.Load()
		if err != nil {
			errMsg := fmt.Sprintf("Plugin %s can't be loaded, reason:\n %s", plug.Name(), err.Error())
			logger.Service.AddLog(logger.LogTypeError, errMsg)
			continue
		}
		loaded = append(loaded, plug)
	}
	// 注册所有的handler和filter
	for _, plug := range loaded {
		pluginName := plug.Name()
		pluginFilters := plug.Filters()
		pluginHandlers := plug.Handlers()
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
		entry.handlers[noFilterKey] = func(event *CQEvent) {
			for _, hanldeFunc := range noFilterHanlers {
				hanldeFunc(event)
			}
		}
		c.pluginEntries[pluginName] = entry
	}
	// 触发所有插件的onload事件
	for _, plug := range loaded {
		go plug.Loaded()
	}
}

// Initialize 初始化客户端
// token 酷q机器人的access token
func (c *cqclient) Initialize(token string) {
	c.token = token
	c.httpConn = clients.NewHTTPClient("")
	c.httpConn.Header.Set("Authorization", fmt.Sprintf("Token %s", c.token))

	c.apiConn.Name = "酷Q机器人Api"
	c.eventConn.Name = "酷Q机器人Event"
	c.registerAllPlugins()
	// handle connect
	c.apiConn.OnConnect = handleConnect
	c.eventConn.OnConnect = handleConnect
	// handle error
	c.apiConn.OnError = handleError
	c.eventConn.OnError = handleError
	// handle message
	c.apiConn.OnMessage = func(raw []byte) {
		msg := new(CQResponse)
		err := json.Unmarshal(raw, msg)
		if err != nil {
			logger.Service.AddLog(logger.LogTypeError, err.Error())
			return
		}
		echo := msg.Echo
		if c.echoqueue[echo] {
			c.mu.Lock()
			delete(c.echoqueue, echo)
			c.mu.Unlock()
		}
	}
	// handle events
	c.eventConn.OnMessage = func(raw []byte) {
		event := new(CQEvent)
		err := json.Unmarshal(raw, event)
		if err != nil {
			errMsg := err.Error()
			logger.Service.AddLog(logger.LogTypeError, errMsg)
			return
		}
		for _, entry := range c.pluginEntries {
			go entry.handlers[noFilterKey](event)
			for key, filterFunc := range entry.fitlers {
				handleFunc := entry.handlers[key]
				go func(filterFunc *Filter) {
					if (*filterFunc)(event) {
						handleFunc(event)
					}
				}(&filterFunc)
			}
		}
	}

	// 定时清理echo序列
	go func() {
		ticker := time.NewTicker(timeForWait * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().Unix()
				for echo, state := range c.echoqueue {
					if state && now-echo > timeForWait {
						logger.Service.AddLog(logger.LogTypeError, fmt.Sprintf("Echo = %d 响应超时(30s).", echo))
						c.mu.Lock()
						delete(c.echoqueue, echo)
						c.mu.Unlock()
					}
				}
			}
		}
	}()
}

// Connect 连接远程酷q api服务
// wsURL 形如 ws://127.0.0.1:8080, wss://127.0.0.1:8080之类的url 用于建立ws连接
// httpURL 形如 http://127.0.0.1:8080之类的url 用户建立http”连接“
func (c *cqclient) Connect(wsURL, httpURL string) {
	headers := make(http.Header)
	headers.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	// 连接api服务和事件服务
	c.apiConn.Dial(fmt.Sprintf("%s/api", wsURL), headers)
	c.eventConn.Dial(fmt.Sprintf("%s/event", wsURL), headers)
	c.apiURL = httpURL
}

// IsAPIOk api服务是否可用
func (c *cqclient) IsAPIOk() bool {
	return c.apiConn.IsConnected()
}

// IsEventOk event服务是否可用
func (c *cqclient) IsEventOk() bool {
	return c.eventConn.IsConnected()
}

// APISendJSON 发送api json格式的数据
func (c *cqclient) APISendJSON(data interface{}) {
	if !c.IsAPIOk() {
		return
	}
	msg, _ := json.Marshal(data)
	c.apiConn.Send(websocket.TextMessage, msg)
}

// SendGroupMsg 发送群消息
// websocket 接口
func (c *cqclient) SendGroupMsg(groupID int64, message string) {
	payload := &CQWSMessage{
		Action: ActionSendGroupMsg,
		Params: CQTypeSendGroupMsg{
			GroupID: groupID,
			Message: message,
		},
		Echo: time.Now().Unix(),
	}
	c.APISendJSON(payload)
}

// SendPrivateMsg 发送私聊消息
// websocket 接口
func (c *cqclient) SendPrivateMsg(userID int64, message string) {
	payload := &CQWSMessage{
		Action: ActionSendPrivateMsg,
		Params: CQTypeSendPrivateMsg{
			UserID:  userID,
			Message: message,
		},
		Echo: time.Now().Unix(),
	}
	c.APISendJSON(payload)
}

// SetGroupKick 群组踢人
// reject 是否拒绝加群申请
// websocket 接口
func (c *cqclient) SetGroupKick(groupID, userID int64, reject bool) {
	payload := &CQWSMessage{
		Action: ActionSetGroupKick,
		Params: CQTypeSetGroupKick{
			GroupID:          groupID,
			UserID:           userID,
			RejectAddRequest: reject,
		},
		Echo: time.Now().Unix(),
	}
	c.APISendJSON(payload)
}

// SetGroupBan 群组单人禁言
// duration 禁言时长，单位秒，0 表示取消禁言
// websocket 接口
func (c *cqclient) SetGroupBan(groupID, userID int64, duration int64) {
	payload := &CQWSMessage{
		Action: ActionSetGroupBan,
		Params: CQTypeSetGroupBan{
			GroupID:  groupID,
			UserID:   userID,
			Duration: duration,
		},
		Echo: time.Now().Unix(),
	}
	c.APISendJSON(payload)
}

// SetGroupWholeBan 群组全员禁言
// enable 是否禁言
// websocket 接口
func (c *cqclient) SetGroupWholeBan(groupID int64, enable bool) {
	payload := &CQWSMessage{
		Action: ActionSetGroupWholeBan,
		Params: CQTypeSetGroupWholeBan{
			GroupID: groupID,
			Enable:  enable,
		},
		Echo: time.Now().Unix(),
	}
	c.APISendJSON(payload)
}

func warnHTTPApiURLNotSet() {
	log.Println("[WARNING] Try to request a http api url, but no http api url was set.")
}

func (c *cqclient) getAPIURL(api string) string {
	return fmt.Sprintf("%s/%s", c.apiURL, api)
}

// GetStatus 获取插件运行状态
// http 接口
func (c *cqclient) GetStatus() *CQTypeGetStatus {
	if c.apiURL == "" {
		warnHTTPApiURLNotSet()
		return nil
	}
	url := c.getAPIURL(ActionGetStatus)
	res, err := c.httpConn.Get(url)
	if err != nil {
		errMsg := err.Error()
		logger.Service.AddLog(logger.LogTypeError, errMsg)
		return nil
	}
	defer res.Body.Close()
	response := new(CQResponse)
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		errMsg := err.Error()
		logger.Service.AddLog(logger.LogTypeError, errMsg)
		return nil
	}
	if response.RetCode != 0 {
		return nil
	}
	data := response.Data.(map[string]interface{})
	status := new(CQTypeGetStatus)
	status.AppInitialized = data["app_initialized"].(bool)
	status.AppEnabled = data["app_enabled"].(bool)
	status.PluginsGood = data["plugins_good"].(bool)
	status.AppGood = data["app_good"].(bool)
	status.Online = data["online"].(bool)
	status.Good = data["good"].(bool)
	return status
}

// Client 唯一的酷q机器人实体
var Client = &cqclient{
	apiConn:       new(clients.WSClient),
	eventConn:     new(clients.WSClient),
	pluginEntries: make(map[string]pluginEntry),
}
