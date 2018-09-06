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

// LogTypeInfo 信息类型
const LogTypeInfo = 0

// LogTypeError 错误类型
const LogTypeError = 1

// LogTypeSuccess 成功on类型
const LogTypeSuccess = 2

var logTypeStr = []string{"info", "error", "success"}

// Log log消息格式(json)
type Log struct {
	Time int64  `json:"time"`
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewLog 创建一个新的Log实例
func NewLog(ltype int, text string) *Log {
	now := time.Now().Unix()
	return &Log{
		Time: now,
		Type: logTypeStr[ltype],
		Text: text,
	}
}

type loggerService struct {
	conns    map[*websocket.Conn]bool
	success  int
	fails    int
	logsPath string
	queue    []*Log
	mu       sync.Mutex
}

// 时间格式等基本的常量
const logDateLayout = "2006-01-02"
const logTimeLayout = "2006年01月02日 15:04:05"
const pingWaitTime = 5 * time.Second

// Service 单例实体
var Service loggerService

// SetLogsPath 设置log文件目录
func (logger *loggerService) SetLogsPath(p string) {
	logger.logsPath = p
}

// LogsPath 获取logs文件的绝对路径
func (logger *loggerService) LogsPath() string {
	pwd, _ := os.Getwd()
	logspath := path.Join(pwd, logger.logsPath)
	return logspath
}

// LogFile 获取当前log文件的位置
func (logger *loggerService) LogFile() string {
	logspath := logger.LogsPath()
	date := time.Now().Format(logDateLayout)
	filename := fmt.Sprintf("%s.log", date)
	return path.Join(logspath, filename)
}

// Success 获取成功计数
func (logger *loggerService) Success() int {
	return logger.success
}

// Success 获取失败计数
func (logger *loggerService) Fails() int {
	return logger.fails
}

// Add 往队列里加入一个新的log
func (logger *loggerService) Add(lg *Log) {
	switch lg.Type {
	case logTypeStr[LogTypeSuccess]:
		logger.success++
	case logTypeStr[LogTypeError]:
		logger.fails++
	}
	logger.queue = append(logger.queue, lg)
}

// AddLog 往队列里加入一个新的log
func (logger *loggerService) AddLog(ltype int, text string) {
	lg := NewLog(ltype, text)
	logger.Add(lg)
}

// pop 冲队列中取出一个log
func (logger *loggerService) pop() (int, *Log) {
	if len(logger.queue) == 0 {
		return 0, nil
	}
	lg := logger.queue[0]
	logger.queue = logger.queue[1:]
	return 1, lg
}

// writeToFile log写入文件
func (logger *loggerService) writeToFile(lg *Log) {
	logtime := time.Unix(lg.Time, 0).Format(logTimeLayout)
	logfile := logger.LogFile()
	fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("Logger", err)
	}
	logstr := fmt.Sprintf("%s [%s]: %s\n", lg.Type, logtime, lg.Text)
	defer fp.Close()
	logger.mu.Lock()
	fp.WriteString(logstr)
	logger.mu.Unlock()
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

// WSLogHandler 广播log
func WSLogHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Service.AddLog(LogTypeError, err.Error())
		return
	}
	welcome := NewLog(LogTypeSuccess, "Log websocket 服务连接成功")
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
		cnt, lg := Service.pop()
		if cnt == 0 {
			time.Sleep(time.Second)
			continue
		}
		Service.writeToFile(lg)
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

// RawLogHandler 获取log文件
func RawLogHandler(w http.ResponseWriter, r *http.Request) {
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
	logfilePath := path.Join(Service.LogsPath(), logfileName)
	fmt.Println(logfilePath)
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

// Initialize 初始化logger服务
func (logger *loggerService) Initialize() {
	if logger.logsPath == "" {
		log.Fatal("LogsPath not set please use logger.Default.SetLogsPath func set it.")
	}
	logspath := logger.LogsPath()
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
}
