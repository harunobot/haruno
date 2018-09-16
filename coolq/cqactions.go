package coolq

// 文档: https://cqhttp.cc/docs/4.4/#/API?id=api-列表
// 大致先做这些...
const (
	// ActionSendPrivateMsg 发送私聊消息
	ActionSendPrivateMsg = "send_private_msg" // DONE: websocket
	// ActionSendGroupMsg 发送群消息
	ActionSendGroupMsg = "send_group_msg" // DONE: websocket
	// ActionSetGroupKick 群组踢人
	ActionSetGroupKick = "set_group_kick" // DONE: websocket
	// ActionSetGroupBan 群组单人禁言
	ActionSetGroupBan = "set_group_ban" // DONE: websocket
	// ActionSetGroupWholeBan 群组全员禁言
	ActionSetGroupWholeBan = "set_group_whole_ban" // DONE: websocket
	// ActionGetStatus 获取插件运行状态
	ActionGetStatus = "get_status" // DONE: http
)

// CQWSMessage coolq ws基本消息类型
type CQWSMessage struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   int64       `json:"echo"`
}

// CQResponse coolq ws响应类型
type CQResponse struct {
	Status  string      `json:"status"`
	RetCode int         `json:"retcode"`
	Data    interface{} `json:"data"`
	Echo    int64       `json:"echo"`
}

// CQTypeSendGroupMsg SendGroupMsg动作的数据格式
type CQTypeSendGroupMsg struct {
	GroupID    int64  `json:"group_id"`
	Message    string `json:"message"`
	AutoEscape bool   `json:"auto_escape"`
}

// CQTypeSendPrivateMsg ActionSendPrivateMsg动作的数据格式
type CQTypeSendPrivateMsg struct {
	UserID     int64  `json:"user_id"`
	Message    string `json:"message"`
	AutoEscape bool   `json:"auto_escape"`
}

// CQTypeSetGroupKick AActionSetGroupKick动作数据格式
type CQTypeSetGroupKick struct {
	GroupID          int64 `json:"group_id"`
	UserID           int64 `json:"user_id"`
	RejectAddRequest bool  `json:"reject_add_request"`
}

// CQTypeSetGroupBan ActionSetGroupBan动作数据格式
type CQTypeSetGroupBan struct {
	GroupID  int64 `json:"group_id"`
	UserID   int64 `json:"user_id"`
	Duration int64 `json:"duration"`
}

// CQTypeSetGroupWholeBan ActionSetGroupWholeBan动作数据格式
type CQTypeSetGroupWholeBan struct {
	GroupID int64 `json:"group_id"`
	Enable  bool  `json:"enable"`
}

// CQTypeGetStatus ActionGetStatus的响应数据格式
type CQTypeGetStatus struct {
	AppInitialized bool `json:"app_initialized"`
	AppEnabled     bool `json:"app_enabled"`
	PluginsGood    bool `json:"plugins_good"`
	AppGood        bool `json:"app_good"`
	Online         bool `json:"online"`
	Good           bool `json:"good"`
}
