package coolq

// 文档: https://cqhttp.cc/docs/4.4/#/API?id=api-列表
// 大致先做这些...
const (
	// ActionSendPrivateMsg 发送私聊消息
	ActionSendPrivateMsg = "/send_private_msg"
	// ActionSendGroupMsg 发送群消息
	ActionSendGroupMsg = "/send_group_msg" /* ... */
	// ActionSetGroupSpecialTitle 设置群组专属头衔
	ActionSetGroupSpecialTitle = "/set_group_special_title"
	// ActionSetGroupBan 群组单人禁言
	ActionSetGroupBan = "/set_group_ban"
	// ActionSetGroupWholeBan 群组全员禁言
	ActionSetGroupWholeBan = "/set_group_whole_ban"
	// ActionGetGroupMemberList 获取群成员列表
	ActionGetGroupMemberList = "/get_group_member_list"
	// ActionGetStatus 获取插件运行状态
	ActionGetStatus = "/get_status"
	// ActionSetRestartPlugin 重启 HTTP API 插件
	ActionSetRestartPlugin = "/set_restart_plugin"
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

// CQTypeSendGroupMsg SendGroupMsg动作的数据类型
type CQTypeSendGroupMsg struct {
	GroupID    int64  `json:"group_id"`
	Message    string `json:"message"`
	AutoEscape bool   `json:"auto_escape"`
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
