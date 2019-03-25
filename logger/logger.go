package logger

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// Logger 应用使用的 logger 实例
var Logger = logrus.WithField("target", "haruno")

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
	conns      map[*websocket.Conn]bool
	success    int
	fails      int
	logsPath   string
	logChan    chan *Log
	logFT      string        // log时间格式
	logN       *logrus.Entry // 正常log
	logE       *logrus.Entry // 错误log
	fpN        *os.File
	fpE        *os.File
	fpLock     sync.Mutex
	wsConnLock sync.Mutex
}

// 时间格式等基本的常量
const logDateFormat = "2006-01-02"
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
func (logger *loggerService) LogFile(scope string) string {
	logspath := logger.LogsPath()
	date := time.Now().Format(logDateFormat)
	filename := fmt.Sprintf("%s.log", date)
	if len(scope) != 0 {
		filename = fmt.Sprintf("%s-%s.log", date, scope)
	}
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

func (logger *loggerService) resetLogFiles() {
	logger.logN.Logger.SetOutput(os.Stdout)
	logger.logE.Logger.SetOutput(os.Stdout)
	if logger.fpN != nil {
		logger.fpN.Close()
		logger.fpN = nil
	}
	if logger.fpE != nil {
		logger.fpE.Close()
		logger.fpE = nil
	}
}

func (logger *loggerService) setLogFiles() {
	var err error
	logfileN := logger.LogFile("")
	logfileE := logger.LogFile("error")
	if logfileN != logger.logFT {
		logger.resetLogFiles()
		logger.logFT = logfileN
		logger.fpN, err = os.OpenFile(logfileN, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			Logger.Fatal("Logger", err)
		}
		logger.fpE, err = os.OpenFile(logfileE, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			Logger.Fatal("Logger", err)
		}
		logger.logN.Logger.SetOutput(logger.fpN)
		logger.logE.Logger.SetOutput(logger.fpE)
	}
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
	// logger.fpLock.Lock()
	// defer logger.fpLock.Unlock()
	logger.setLogFiles()
	lg.Text = escapeHost(lg.Text)
	logMsg := escapeCRLF(lg.Text)
	switch lg.Type {
	case LogTypeSuccess:
		Logger.Println(logMsg)
		logger.logN.Println(lg.Text)
		logger.success++
	case LogTypeError:
		Logger.Errorln(logMsg)
		logger.logE.Println(lg.Text)
		logger.fails++
	default:
		Logger.Infoln(logMsg)
		logger.logN.Infoln(lg.Text)
	}
	logger.logChan <- lg
	if len(logger.logChan) >= maxQueueSize {
		<-logger.logChan
	}
}

// AddLog 往队列里加入一个新的log
func (logger *loggerService) AddLog(ltype int, text string) {
	lg := NewLog(ltype, text)
	logger.Add(lg)
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
		Logger.Fatal("LogsPath not set please use logger.Default.SetLogsPath func set it.")
	}
	logspath := logger.LogsPath()
	Logger.Printf("LogsPath = %s\n", logspath)
	_, err := os.Stat(logspath)
	if err != nil {
		// 不存在目录的时候创建目录
		if os.IsNotExist(err) {
			Logger.Println("LogsPath is not existed.")
			err = os.Mkdir(logspath, 0700)
			if err != nil {
				Logger.Fatal("Logger", err)
			}
			Logger.Println("LogsPath created successfully.")
		}
	}
	// 创建连接池
	logger.conns = make(map[*websocket.Conn]bool)
	// 创建log管道
	logger.logChan = make(chan *Log, maxQueueSize)
	// 创建 logrus normal 实例
	logger.logN = logrus.New().WithFields(logrus.Fields{
		"target": "haruno",
		"types":  []string{"success", "info"},
	})
	// 创建 logrus error 实例
	logger.logE = logrus.New().WithFields(logrus.Fields{
		"target": "haruno",
		"types":  []string{"error"},
	})
	logger.logN.Logger.SetFormatter(&logrus.TextFormatter{})
	logger.logE.Logger.SetFormatter(&logrus.TextFormatter{})
	logger.setLogFiles()
}
