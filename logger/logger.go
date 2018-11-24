package logger

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LogTypeInfo 信息类型
const LogTypeInfo = 0

// LogTypeError 错误类型
const LogTypeError = 1

// LogTypeSuccess 成功类型
const LogTypeSuccess = 2

// maxQueueSize 队列最大大小
// == 用户首次通过websocket链接能看到的最大的日志数量
const maxQueueSize = 5

var logTypeStr = []string{"info", "error", "success"}

// Log log消息格式(json)
type Log struct {
	Time int64  `json:"time"`
	Type int    `json:"type"`
	Text string `json:"text"`
}

// NewLog 创建一个新的Log实例
func NewLog(ltype int, text string) *Log {
	now := time.Now().Unix()
	return &Log{
		Time: now,
		Type: ltype,
		Text: text,
	}
}

type loggerService struct {
	conns        map[*websocket.Conn]bool
	success      int
	fails        int
	logsPath     string
	queue        []*Log
	logChan      chan int64
	logLock      sync.Mutex
	logWriteLock sync.Mutex
	wsConnLock   sync.Mutex
}

// 时间格式等基本的常量
const logDateFormat = "2006-01-02"
const logTimeFormat = "2006-01-02 15:04:05 -0700"
const pongWaitTime = 5 * time.Second

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
	date := time.Now().Format(logDateFormat)
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

func escapeCRLF(s string) string {
	cr, _ := regexp.Compile(`\r`)
	lf, _ := regexp.Compile(`\n`)
	s = cr.ReplaceAllString(s, "\\r")
	s = lf.ReplaceAllString(s, "\\n")
	return s
}

func escapeHost(s string) string {
	host, _ := regexp.Compile(`(\d+)\.\d+\.\d+\.(\d+)(?:\:(\d+))?`)
	s = host.ReplaceAllString(s, "$1.*.*.$2:$3")
	return s
}

// Add 往队列里加入一个新的log
func (logger *loggerService) Add(lg *Log) {
	lg.Text = escapeHost(lg.Text)
	msg := escapeCRLF(lg.Text)
	log.Println(msg)
	logger.logLock.Lock()
	switch lg.Type {
	case LogTypeSuccess:
		logger.success++
	case LogTypeError:
		logger.fails++
	}
	logger.queue = append(logger.queue, lg)
	logger.writeToFile(lg)
	logger.logChan <- lg.Time
	if len(logger.queue) >= maxQueueSize {
		<-logger.logChan
		logger.queue = logger.queue[1:]
	}
	logger.logLock.Unlock()
}

// AddLog 往队列里加入一个新的log
func (logger *loggerService) AddLog(ltype int, text string) {
	lg := NewLog(ltype, text)
	logger.Add(lg)
}

// pop 从队列中取出一个log
func (logger *loggerService) pop() (bool, *Log) {
	if len(logger.queue) == 0 {
		return false, nil
	}
	lg := logger.queue[0]
	logger.logLock.Lock()
	logger.queue = logger.queue[1:]
	logger.logLock.Unlock()
	return true, lg
}

// writeToFile log写入文件
func (logger *loggerService) writeToFile(lg *Log) {
	logtime := time.Unix(lg.Time, 0).Format(logTimeFormat)
	logfile := logger.LogFile()
	fp, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("Logger", err)
	}
	logstr := fmt.Sprintf("%s - - %s - - %s\n", logtime, logTypeStr[lg.Type], lg.Text)
	defer fp.Close()
	logger.logWriteLock.Lock()
	fp.WriteString(logstr)
	logger.logWriteLock.Unlock()
}

func delConn(conn *websocket.Conn) {
	Service.wsConnLock.Lock()
	delete(Service.conns, conn)
	Service.wsConnLock.Unlock()
}

func setupPong(conn *websocket.Conn, quit chan int) {
	pongTicker := time.NewTicker(pongWaitTime)
	pongMsg := []byte("")
	go func() {
		defer pongTicker.Stop()
		defer conn.Close()
		defer delConn(conn)
		for {
			if Service.conns[conn] != true {
				close(quit)
			}
			select {
			case <-quit:
				return
			case <-pongTicker.C:
				conn.SetWriteDeadline(time.Now().Add(pongWaitTime))
				if err := conn.WriteMessage(websocket.PongMessage, pongMsg); err != nil {
					close(quit)
				}
			}
		}
	}()
}

// Initialize 初始化logger服务
func (logger *loggerService) Initialize() {
	// 建立日志目录
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
	logger.conns = make(map[*websocket.Conn]bool)
	// 创建log管道
	logger.logChan = make(chan int64, maxQueueSize)
}
