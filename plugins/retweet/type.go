package retweet

// TweetMsg 推特消息格式
type TweetMsg struct {
	Status   string   `json:"status"`
	Cmd      string   `json:"cmd"`
	FromID   string   `json:"fromID"`
	FromName string   `json:"fromName"`
	Avatar   string   `json:"avatar"`
	Imgs     []string `json:"imgs"`
	Text     string   `json:"msg"`
}
