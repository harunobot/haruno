package coolq

// 文档: https://cqhttp.cc/docs/4.3/#/API?id=api-%E5%88%97%E8%A1%A8
// 大致先做这些...
const (
	// ActionSendPrivateMsg 发送私聊消息
	ActionSendPrivateMsg = "send_private_msg"
	// ActionSendGroupMsg 发送群消息
	ActionSendGroupMsg = "send_group_msg" /* ... */
	// ActionSetGroupSpecialTitle 设置群组专属头衔
	ActionSetGroupSpecialTitle = "set_group_special_title"
	// ActionSetGroupBan 群组单人禁言
	ActionSetGroupBan = "set_group_ban"
	// ActionSetGroupWholeBan 群组全员禁言
	ActionSetGroupWholeBan = "set_group_whole_ban"
	// ActionGetGroupMemberList 获取群成员列表
	ActionGetGroupMemberList = "get_group_member_list"
	// ActionGetStatus 获取插件运行状态
	ActionGetStatus = "get_status"
	// ActionSetRestartPlugin 重启 HTTP API 插件
	ActionSetRestartPlugin = "set_restart_plugin"
)
