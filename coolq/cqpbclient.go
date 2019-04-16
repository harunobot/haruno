package coolq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	"github.com/haruno-bot/haruno/clients"
	"github.com/haruno-bot/haruno/kcwiki_msgtransfer_protobuf"
	"github.com/haruno-bot/haruno/logger"
)

// cqpbclient 酷q机器人连接客户端
// 为了安全起见，暂时不允许在包外额外创建
type cqpbclient struct {
	mu             sync.Mutex
	token          string
	apiConn        *clients.WSClient
	eventConn      *clients.WSClient
	httpConn       *clients.HTTPClient
	apiURL         string
	pluginEntries  map[string]pluginEntry
	echoqueue      map[int64]bool
	servConn       *clients.WSClient
	servToken      string
	pluginCallback map[string]func([]byte)
}

// RegisterAllPlugins 注册所有的插件
func (c *cqpbclient) RegisterAllPlugins() {
	// 1. 先全部执行加载函数
	loaded := make([]pbPluginInterface, 0)
	for _, plug := range pbentries {
		err := plug.Load()
		if err != nil {
			logger.Errorf("Plugin %s can't be loaded, reason:\n %s", plug.Name(), err.Error())
			continue
		}
		loaded = append(loaded, plug)
	}
	// 2. 注册所有的handler和filter
	c.mu.Lock()
	for _, plug := range loaded {
		pluginName := plug.Name()
		pluginModuleName := plug.Module()
		pluginFilters := plug.Filters()
		pluginHandlers := plug.Handlers()
		hasFilter := make(map[string]bool)
		entry := pluginEntry{
			keys:     make([]string, 0),
			fitlers:  make(map[string]Filter),
			handlers: make(map[string]Handler),
		}
		noFilterHanlers := make([]Handler, 0)
		// 对应filter的key寻找相应的handler， 没有的话则给出警告
		for key, filter := range pluginFilters {
			handler := pluginHandlers[key]
			if handler == nil {
				logger.Logger.Warnf("插件 %s 中存在没有使用的key: %s\n", pluginName, key)
				continue
			}
			hasFilter[key] = true
			entry.keys = append(entry.keys, key)
			entry.fitlers[key] = filter
			entry.handlers[key] = handler
		}
		for key, handler := range pluginHandlers {
			if !hasFilter[key] {
				noFilterHanlers = append(noFilterHanlers, handler)
			}
		}
		// 最后注册无key的handler
		entry.handlers[noFilterKey] = func(event *CQEvent) {
			for _, hanldeFunc := range noFilterHanlers {
				hanldeFunc(event)
			}
		}
		c.pluginEntries[pluginName] = entry
		c.pluginCallback[pluginModuleName] = plug.Callback
	}
	c.mu.Unlock()
	// 3. 触发所有插件的onload事件
	for _, plug := range loaded {
		go plug.Loaded()
	}
}

// Initialize 初始化客户端
// token 酷q机器人的access token
func (c *cqpbclient) Initialize(token, servToken string) {
	c.token = token
	c.servToken = servToken
	c.httpConn = clients.NewHTTPClient()
	c.httpConn.Header.Set("Authorization", fmt.Sprintf("Token %s", c.token))

	c.apiConn.Name = "酷Q机器人Api"
	c.eventConn.Name = "酷Q机器人Event"
	c.servConn.Name = "信息推送"
	// 注册连接事件回调
	c.apiConn.OnConnect = handleConnect
	c.eventConn.OnConnect = handleConnect
	c.servConn.OnConnect = handleConnect
	// 注册错误事件回调
	c.apiConn.OnError = func(err error) {
		logger.Field("cqpbclient api conn error").Error(err.Error())
	}
	c.eventConn.OnError = func(err error) {
		logger.Field("cqpbclient event conn error").Error(err.Error())
	}
	c.servConn.OnError = func(err error) {
		logger.Field("cqpbclient server socket conn error").Error(err.Error())
	}
	// 注册消息事件回调
	c.apiConn.OnMessage = func(raw []byte) {
		msg := new(CQResponse)
		err := json.Unmarshal(raw, msg)
		if err != nil {
			logger.Errorf("API conn on message erorr: %s", err.Error())
			return
		}
		// echo队列 - 确定发送消息是否超时
		echo := msg.Echo
		if c.echoqueue[echo] {
			c.mu.Lock()
			delete(c.echoqueue, echo)
			c.mu.Unlock()
		}
	}
	// 注册上报事件回调
	c.eventConn.OnMessage = func(raw []byte) {
		event := new(CQEvent)
		err := json.Unmarshal(raw, event)
		if err != nil {
			logger.Errorf("Event conn on message erorr: %s", err.Error())
			return
		}
		for name, entry := range c.pluginEntries {
			// 先异步处理没有key的回调
			go entry.handlers[noFilterKey](event)
			// 一次异步执行所有的 filter 和 handler 对
			for _, key := range entry.keys {
				go func(key string, name string) {
					if c.pluginEntries[name].fitlers[key](event) {
						c.pluginEntries[name].handlers[key](event)
					}
				}(key, name)
			}
		}
	}
	// 注册服务器推送事件回调
	c.servConn.OnMessage = func(raw []byte) {
		wsWrapper := new(kcwiki_msgtransfer_protobuf.Websocket)
		err := proto.Unmarshal(raw, wsWrapper)
		if err != nil {
			logger.Service.Field("Server Socket").Errorf("%s", err.Error())
			return
		}
		switch wsWrapper.GetProtoType() {
		case kcwiki_msgtransfer_protobuf.Websocket_SYSTEM:
			wsSystem := new(websocketSystem)
			err := json.Unmarshal(wsWrapper.GetProtoPayload(), wsSystem)
			if err != nil {
				logger.Service.Field("Server Socket").Errorf("%s", err.Error())
				return
			}
			logger.Service.Field("Server Socket").Successf("%s - %s", wsSystem.MsgType, wsSystem.Data)
		case kcwiki_msgtransfer_protobuf.Websocket_NON_SYSTEM:
			module := wsWrapper.GetProtoModule()
			if wsWrapper.GetProtoCode() != kcwiki_msgtransfer_protobuf.Websocket_SUCCESS {
				logger.Service.Field(module).Infof("%s", string(wsWrapper.GetProtoPayload()))
			}
			if c.pluginCallback[module] == nil {
				logger.Service.Field(module).Error("could not find module")
				return
			}
			c.pluginCallback[module](wsWrapper.GetProtoPayload())
		}
	}

	// 定时清理echo队列 (30s)
	go func() {
		ticker := time.NewTicker(timeForWait * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().Unix()
				for echo, state := range c.echoqueue {
					// 对于超过30s未响应的给出提示
					if state && now-echo > timeForWait {
						logger.Errorf("Echo = %d 响应超时(30s).", echo)
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
func (c *cqpbclient) Connect(wsURL, httpURL, servWsURL string) {
	headers := make(http.Header)
	headers.Add("Authorization", fmt.Sprintf("Token %s", c.token))
	// 连接api服务和事件服务
	c.apiConn.Dial(fmt.Sprintf("%s/api", wsURL), headers)
	c.eventConn.Dial(fmt.Sprintf("%s/event", wsURL), headers)
	c.servConn.Dial(servWsURL, http.Header{
		"x-access-token": []string{c.servToken},
	})
	c.apiURL = httpURL
}

// IsAPIOk api服务是否可用
func (c *cqpbclient) IsAPIOk() bool {
	return c.apiConn.IsConnected()
}

// IsEventOk event服务是否可用
func (c *cqpbclient) IsEventOk() bool {
	return c.eventConn.IsConnected()
}

// ServerSendNonSystemJSON 发送server json格式的数据
func (c *cqpbclient) ServerSendNonSystemJSON(module string, data interface{}) {
	if !c.IsAPIOk() {
		return
	}
	msg, _ := json.Marshal(data)
	wsWrapper := new(kcwiki_msgtransfer_protobuf.Websocket)
	wsWrapper.ProtoCode = kcwiki_msgtransfer_protobuf.Websocket_SUCCESS
	wsWrapper.ProtoType = kcwiki_msgtransfer_protobuf.Websocket_NON_SYSTEM
	wsWrapper.ProtoModule = module
	wsWrapper.ProtoPayload = msg
	payload, err := proto.Marshal(wsWrapper)
	if err != nil {
		logger.Service.Field(module).Errorf("%s", err.Error())
		return
	}
	c.servConn.Send(websocket.BinaryMessage, payload)
}

// ServerSendSystemJSON 发送server json格式的数据
func (c *cqpbclient) ServerSendSystemJSON(rstype ResultType, data websocketSystem) {
	if !c.IsAPIOk() {
		return
	}
	msg, _ := json.Marshal(data)
	wsWrapper := new(kcwiki_msgtransfer_protobuf.Websocket)
	switch rstype {
	case SUCCESS:
		wsWrapper.ProtoCode = kcwiki_msgtransfer_protobuf.Websocket_SUCCESS
	case FAIL:
		wsWrapper.ProtoCode = kcwiki_msgtransfer_protobuf.Websocket_FAIL
	case ERROR:
		wsWrapper.ProtoCode = kcwiki_msgtransfer_protobuf.Websocket_ERROR
	}
	wsWrapper.ProtoType = kcwiki_msgtransfer_protobuf.Websocket_SYSTEM
	wsWrapper.ProtoPayload = msg
	payload, err := proto.Marshal(wsWrapper)
	if err != nil {
		logger.Service.Field("Server Socket").Errorf("%s", err.Error())
		return
	}
	c.servConn.Send(websocket.BinaryMessage, payload)
}

// APISendJSON 发送api json格式的数据
func (c *cqpbclient) APISendJSON(data interface{}) {
	if !c.IsAPIOk() {
		return
	}
	msg, _ := json.Marshal(data)
	err := c.apiConn.Send(websocket.TextMessage, msg)
	if err != nil {
		log.Fatal(err)
		logger.Service.Field("Server Socket").Errorf("%s", err.Error())
	}
}

// SendGroupMsg 发送群消息
// websocket 接口
func (c *cqpbclient) SendGroupMsg(groupID int64, message string) {
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
func (c *cqpbclient) SendPrivateMsg(userID int64, message string) {
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
func (c *cqpbclient) SetGroupKick(groupID, userID int64, reject bool) {
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
func (c *cqpbclient) SetGroupBan(groupID, userID int64, duration int64) {
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
func (c *cqpbclient) SetGroupWholeBan(groupID int64, enable bool) {
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

func (c *cqpbclient) getAPIURL(api string) string {
	return fmt.Sprintf("%s/%s", c.apiURL, api)
}

// GetStatus 获取插件运行状态
// http 接口
func (c *cqpbclient) GetStatus() *CQTypeGetStatus {
	if c.apiURL == "" {
		warnHTTPApiURLNotSet()
		return nil
	}
	url := c.getAPIURL(ActionGetStatus)
	res, err := c.httpConn.Get(url)
	if err != nil {
		logger.Errorf("cqpbclient http method getStatus error: %s", err.Error())
		return nil
	}
	defer res.Body.Close()
	response := new(CQResponse)
	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		logger.Errorf("cqpbclient http method getStatus error: %s", err.Error())
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

// PbClient 唯一的酷q机器人实体
var PbClient = &cqpbclient{
	apiConn:        new(clients.WSClient),
	eventConn:      new(clients.WSClient),
	servConn:       new(clients.WSClient),
	pluginEntries:  make(map[string]pluginEntry),
	pluginCallback: make(map[string]func([]byte)),
}
