package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/haruno-bot/haruno/coolq"

	"github.com/haruno-bot/haruno/plugins"

	"github.com/BurntSushi/toml"
	"github.com/haruno-bot/haruno/logger"
)

type config struct {
	Version    string `toml:"version"`
	LogsPath   string `toml:"logsPath"`
	ServerPort int    `toml:"serverPort"`
	CQURL      string `toml:"cqURL"`
	CQToken    string `toml:"cqToken"`
}

// haruno 晴乃机器人
// 机器人运行的全局属性
type haruno struct {
	startTime int64
	port      int
	logpath   string
	version   string
	cqURL     string
	cqToken   string
}

var bot = new(haruno)

func (bot *haruno) loadConfig() {
	cfg := new(config)
	_, err := toml.DecodeFile("./config.toml", cfg)
	if err != nil {
		log.Fatal("Haruno Initialize fialed", err)
	}
	bot.startTime = time.Now().UnixNano() / 1e6
	bot.port = cfg.ServerPort
	bot.logpath = cfg.LogsPath
	bot.version = cfg.Version
	bot.cqURL = cfg.CQURL
	bot.cqToken = cfg.CQToken
}

// Initialize 从配置文件读取配置初始化
func (bot *haruno) Initialize() {
	bot.loadConfig()
	plugins.SetupPlugins()
	logger.Service.SetLogsPath(bot.logpath)
	logger.Service.Initialize()
	coolq.Default.Initialize()
	coolq.Default.Connect(bot.cqURL, bot.cqToken)
}

// Status 运行状态json格式
type Status struct {
	Go      int    `json:"go"`
	Version string `json:"version"`
	Success int    `json:"success"`
	Fails   int    `json:"fails"`
	Start   int64  `json:"start"`
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := new(Status)
	status.Fails = logger.Service.Fails()
	status.Success = logger.Service.Success()
	status.Start = bot.startTime
	status.Version = bot.version
	w.Header().Set("Content-Type", "application/json")
	status.Go = runtime.NumGoroutine()
	json.NewEncoder(w).Encode(status)
}

// Run 启动机器人
func (bot *haruno) Run() {
	page := http.FileServer(http.Dir("./server"))
	http.Handle("/", page)

	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/logs/-/type=websocket", logger.WSLogHandler)
	http.HandleFunc("/logs/-/type=plain", logger.RawLogHandler)

	addr := fmt.Sprintf("127.0.0.1:%d", bot.port)
	log.Printf("Haruno server works on http://%s.\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Haruno listen fialed", err)
	}
}

func main() {
	bot.Initialize()
	bot.Run()
}
