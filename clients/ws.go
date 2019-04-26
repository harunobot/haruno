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
	rquit     chan int
	wquit     chan int
	dialer    *websocket.Dialer
	mmu       sync.Mutex
	cmu       sync.Mutex
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
	if c.conn, _, err = c.dialer.Dial(url, headers); err != nil {
		return err
	}
	c.closed = false
	c.rquit = make(chan int)
	c.wquit = make(chan int)
	if c.OnConnect != nil {
		go c.OnConnect(c)
	}
	go func() {
		for {
			var msg []byte
			if _, msg, err = c.conn.ReadMessage(); err != nil {
				if c.OnError != nil {
					go c.OnError(err)
				}
				close(c.rquit)
				return
			}
			if c.Filter != nil {
				if !c.Filter(msg) {
					continue
				}
			}
			if c.OnMessage != nil {
				go c.OnMessage(msg)
			}
		}
	}()
	go c.setupPing()
	return nil
}

// Send 发送消息
func (c *WSClient) Send(msgType int, msg []byte) error {
	if c.closed {
		return errors.New("can not use closed connection")
	}
	c.mmu.Lock()
	defer c.mmu.Unlock()
	err := c.conn.WriteMessage(msgType, msg)
	if err != nil {
		close(c.wquit)
		if c.OnError != nil {
			go c.OnError(err)
		}
		return err
	}
	return nil
}

// IsConnected 检查是否在连接状态
func (c *WSClient) IsConnected() bool {
	return !c.closed
}

func (c *WSClient) close() {
	c.cmu.Lock()
	defer c.cmu.Unlock()
	if c.closed {
		return
	}
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

func (c *WSClient) setupPing() {
	ticker := time.NewTicker(time.Second * 5)
	pingMsg := []byte("")
	defer ticker.Stop()
	defer c.close()
	for {
		select {
		case <-c.rquit:
			return
		case <-c.wquit:
			return
		case <-ticker.C:
			if c.Send(websocket.PingMessage, pingMsg) != nil {
				return
			}
		}
	}
}
