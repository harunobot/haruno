package logger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	RequestParamError = "请求参数错误"
	FileNotFoundError = "Log文件不存在"
	InnerServerError  = "服务器内部错误"
)

var upgrader = websocket.Upgrader{}

// WSLogHandler 广播log
func WSLogHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Service.AddLog(LogTypeError, "Logger WSLogHandler error: %s", err.Error())
		return
	}
	var welMsg = NewLog(LogTypeInfo, "Logger服务连接成功!")
	Service.wscLock.Lock()
	Service.conns[conn] = true
	Service.wscLock.Unlock()
	quit := make(chan int)
	setupPong(conn, quit)
	conn.WriteJSON(welMsg)
	for {
		if !Service.conns[conn] {
			close(quit)
			return
		}
		select {
		case <-quit:
			return
		case lg := <-Service.logChan:
			for c, ok := range Service.conns {
				if !ok {
					continue
				}
				err := c.WriteJSON(lg)
				if err != nil {
					Service.wscLock.Lock()
					Service.conns[c] = false
					Service.wscLock.Unlock()
				}
			}
		}

	}
}

// RawLogHandler 获取log文件
func RawLogHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	date := query.Get("date")
	logType := query.Get("type")
	tim, err := time.Parse(logDateFormat, date)
	if date == "" || err != nil {
		http.Error(w, RequestParamError, 404)
		return
	}
	logfileName := fmt.Sprintf("%s.log", tim.Format(logDateFormat))
	if len(logType) != 0 {
		if strings.ToLower(logType) == "error" {
			logfileName = fmt.Sprintf("%s-%s.log", tim.Format(logDateFormat), logType)
		} else {
			http.Error(w, RequestParamError, 404)
			return
		}
	}
	logfilePath := path.Join(Service.LogsPath(), logfileName)
	stat, err := os.Stat(logfilePath)
	if err != nil && os.IsNotExist(err) {
		http.Error(w, FileNotFoundError, 404)
		return
	}
	fp, err := os.OpenFile(logfilePath, os.O_RDONLY, 0600)
	if err != nil {
		Logger.Println(err)
		http.Error(w, InnerServerError, 500)
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
