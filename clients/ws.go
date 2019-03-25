package clients

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/haruno-bot/haruno/logger"
)

// WSClient 拓展的websocket客户端，可以自动重连
// 这个没有默认的客户端
type WSClient struct {
	Name      string
	OnMessage func([]byte)
	OnError   func(error)
	OnConnect func(*WSClient)
	Filter    func([]byte) bool
	headers   http.Header
	conn      *websocket.Conn
	url       string
	closed    bool
	dialer    *websocket.Dialer
	mu        sync.Mutex
}

// Dial 设置和远程服务器链接
func (c *WSClient) Dial(url string, headers http.Header) error {
	c.closed = true
	c.url = url
	c.headers = headers
	if c.Name == "" {
		c.Name = "Websocket"
	}
	if c.dialer == nil {
		c.dialer = &websocket.Dialer{
			Proxy: http.ProxyFromEnvironment,
		}
	}
	var err error
	c.conn, _, err = c.dialer.Dial(url, headers)
	if err != nil {
		return err
	}
	c.closed = false
	if c.OnConnect != nil {
		go c.OnConnect(c)
	}
	quit := make(chan int)
	go func() {
		for {
			var msg []byte
			if _, msg, err = c.conn.ReadMessage(); err != nil {
				if c.OnError != nil {
					go c.OnError(err)
				}
				close(quit)
				return
			}
			go c.onMsg(msg)
		}
	}()
	c.setupPing(quit)
	return nil
}

// Send 发送消息
func (c *WSClient) Send(msgType int, msg []byte) error {
	if c.closed {
		return errors.New("can not use closed connection")
	}
	c.mu.Lock()
	err := c.conn.WriteMessage(msgType, msg)
	c.mu.Unlock()
	if err != nil {
		c.close()
		if c.OnError != nil {
			c.OnError(err)
		}
		return err
	}
	return nil
}

// IsConnected 检查是否在连接状态
func (c *WSClient) IsConnected() bool {
	return !c.closed
}

func (c *WSClient) onMsg(msg []byte) {
	if c.Filter != nil {
		if !c.Filter(msg) {
			return
		}
	}
	if c.OnMessage != nil {
		c.OnMessage(msg)
	}
}

func (c *WSClient) close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.closed = true
	for {
		if err := c.Dial(c.url, c.headers); err == nil {
			return
		}
		logger.Logger.Println(c.Name, "has broken down, will reconnect after 5s.")
		time.Sleep(time.Second * 5)
	}
}

func (c *WSClient) setupPing(quit chan int) {
	pingTicker := time.NewTicker(time.Second * 5)
	pingMsg := []byte("")
	go func() {
		defer pingTicker.Stop()
		defer c.close()
		for {
			if c.closed {
				return
			}
			select {
			case <-quit:
				return
			case <-pingTicker.C:
				if c.Send(websocket.PingMessage, pingMsg) != nil {
					close(quit)
				}
			}
		}
	}()
}
