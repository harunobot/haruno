package logger

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// RequestParamError 请求log的参数错误
	RequestParamError = []byte("请求参数错误")
	// FileNotFoundError log文件不存在
	FileNotFoundError = []byte("Log文件不存在")
	// InnerServerError 内部错误
	InnerServerError = []byte("服务器内部错误")
)

var upgrader = websocket.Upgrader{}

// WSLogHandler 广播log
func WSLogHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Service.AddLog(LogTypeError, err.Error())
		return
	}
	welcome := NewLog(LogTypeInfo, "Logger服务连接成功!")
	Service.wsConnLock.Lock()
	Service.conns[conn] = true
	Service.wsConnLock.Unlock()
	quit := make(chan int)
	setupPong(conn, quit)
	conn.WriteJSON(welcome)
	for {
		if !Service.conns[conn] {
			close(quit)
			return
		}
		select {
		case <-quit:
			return
		case <-Service.logChan:
			ok, lg := Service.pop()
			if !ok {
				break
			}
			for c, ok := range Service.conns {
				if !ok {
					continue
				}
				err := c.WriteJSON(lg)
				if err != nil {
					Service.wsConnLock.Lock()
					Service.conns[c] = false
					Service.wsConnLock.Unlock()
				}
			}
		}

	}
}

// RawLogHandler 获取log文件
func RawLogHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	date := query.Get("date")
	if date == "" {
		w.WriteHeader(404)
		w.Write(RequestParamError)
		return
	}
	tim, err := time.Parse(logDateFormat, date)
	if err != nil {
		w.WriteHeader(404)
		w.Write(RequestParamError)
		return
	}

	logfileName := fmt.Sprintf("%s.log", tim.Format(logDateFormat))
	logfilePath := path.Join(Service.LogsPath(), logfileName)
	stat, err := os.Stat(logfilePath)
	if err != nil && os.IsNotExist(err) {
		w.WriteHeader(404)
		w.Write(FileNotFoundError)
		return
	}
	fp, err := os.OpenFile(logfilePath, os.O_RDONLY, 0600)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write(InnerServerError)
		return
	}
	defer fp.Close()
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", stat.Size()))
	buff := make([]byte, 100)
	for {
		cnt, err := fp.Read(buff)
		w.Write(buff[:cnt])
		if err == io.EOF {
			return
		}
	}
}
