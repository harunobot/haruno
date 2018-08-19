package logger

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// Log log消息格式(json)
type Log struct {
	Time int64  `json:"time"`
	Type string `json:"type"`
	Text string `json:"text"`
}

// Logger 全局的log服务
// LogsPath 本地log文件目录
type logger struct {
	conns    map[*websocket.Conn]bool
	LogsPath string
	queue    []Log
	mu       sync.Mutex
}

// 时间格式等基本的常量
const logDateLayout = "2006-01-02"
const logTimeLayout = "2006年01月02日 15:04:05"
const pingWaitTime = 5 * time.Second

// Service 单例实体
var Service logger

// SetLogsPath 设置log文件目录
func (lger *logger) SetLogsPath(p string) {
	lger.LogsPath = p
}

// GetLogsPath 获取logs文件的绝对路径
func (lger *logger) GetLogsPath() string {
	pwd, _ := os.Getwd()
	logspath := path.Join(pwd, lger.LogsPath)
	return logspath
}

// GetLogFile 获取当前log文件的位置
func (lger *logger) GetLogFile() string {
	logspath := lger.GetLogsPath()
	date := time.Now().Format(logDateLayout)
	filename := fmt.Sprintf("%s.log", date)
	return path.Join(logspath, filename)
}

// PushLog 往队列里加入一个新的log
func (lger *logger) PushLog(lg *Log) {
	lger.queue = append(lger.queue, *lg)
}

// PopLog 冲队列中取出一个log
func (lger *logger) PopLog() (int, *Log) {
	if len(lger.queue) == 0 {
		return 0, nil
	}
	lg := &lger.queue[0]
	lger.queue = lger.queue[1:]
	return 1, lg
}

// WriteToFile log写入文件
func (lger *logger) WriteToFile(lg *Log) {
	logtime := time.Unix(lg.Time, 0).Format(logTimeLayout)
	logfile := lger.GetLogFile()
	fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("Logger", err)
	}
	logstr := fmt.Sprintf("%s [%s]: %s\n", lg.Type, logtime, lg.Text)
	defer fp.Close()
	lger.mu.Lock()
	fp.WriteString(logstr)
	lger.mu.Unlock()
}

func setupPong(conn *websocket.Conn, lock *sync.Mutex) {
	pingTicker := time.NewTicker(pingWaitTime)
	defer func() {
		pingTicker.Stop()
		lock.Lock()
		delete(Service.conns, conn)
		lock.Unlock()
		conn.Close()
	}()
	pongMsg := []byte("")
	for {
		if Service.conns[conn] != true {
			return
		}
		select {
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(pingWaitTime))
			if err := conn.WriteMessage(websocket.PingMessage, pongMsg); err != nil {
				return
			}
		}
	}
}

const (
	// RequestParamError 请求log的参数错误
	RequestParamError = "请求参数错误"
	// FileNotFoundError log文件不存在
	FileNotFoundError = "Log文件不存在"
	// InnerServerError 内部错误
	InnerServerError = "服务器内部错误"
)

// wsLogHandler 广播log
func wsLogHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log := Log{Type: "error", Time: time.Now().Unix(), Text: err.Error()}
		Service.PushLog(&log)
		return
	}
	welcome := &Log{
		Type: "success",
		Time: time.Now().Unix(),
		Text: "Log websocket 服务连接成功",
	}
	var lock sync.Mutex
	lock.Lock()
	Service.conns[conn] = true
	lock.Unlock()
	conn.WriteJSON(welcome)
	go setupPong(conn, &lock)
	for {
		if !Service.conns[conn] {
			return
		}
		cnt, lg := Service.PopLog()
		if cnt == 0 {
			time.Sleep(time.Second)
			continue
		}
		Service.WriteToFile(lg)
		for c, ok := range Service.conns {
			if !ok {
				continue
			}
			err := c.WriteJSON(lg)
			if err != nil {
				Service.conns[c] = false
			}
		}
	}
}

func rawLogHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	date := query.Get("date")
	if date == "" {
		w.WriteHeader(404)
		w.Write([]byte(RequestParamError))
		return
	}
	tim, err := time.Parse(logDateLayout, date)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(RequestParamError))
		return
	}
	logfileName := fmt.Sprintf("%s.log", tim.Format(logDateLayout))
	logfilePath := path.Join(Service.GetLogsPath(), logfileName)
	stat, err := os.Stat(logfilePath)
	if err != nil && os.IsNotExist(err) {
		// 不存在目录的时候创建目录
		w.WriteHeader(404)
		w.Write([]byte(FileNotFoundError))
		return
	}
	fp, err := os.OpenFile(logfilePath, os.O_RDONLY, 0600)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		w.Write([]byte(InnerServerError))
		return
	}
	defer fp.Close()
	w.Header().Add("Content-Type", "text/plain")
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

// Listen 监听一个端口 提供websocket和http服务
func (lger *logger) Listen(port int) {
	if lger.LogsPath == "" {
		log.Fatal("LogsPath not set please use logger.Default.SetLogsPath func set it.")
	}
	logspath := lger.GetLogsPath()
	log.Printf("LogsPath = %s\n", logspath)
	_, err := os.Stat(logspath)
	if err != nil {
		// 不存在目录的时候创建目录
		if os.IsNotExist(err) {
			log.Println("LogsPath is not existed.")
			err = os.Mkdir(logspath, 0700)
			if err != nil {
				log.Fatal("Logger", err)
			}
			log.Println("LogsPath created successfully.")
		}
	}
	// 创建连接池
	Service.conns = make(map[*websocket.Conn]bool)
	http.HandleFunc("/log/-/type=websocket", wsLogHandler)
	http.HandleFunc("/log/-/type=plain", rawLogHandler)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("Logger server works on http://%s.\n", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Logger listen fialed", err)
	}
}
