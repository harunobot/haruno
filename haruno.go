package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/haruno-bot/haruno/logger"

	"github.com/BurntSushi/toml"
)

// HarunoCfg 晴乃配置
type HarunoCfg struct {
	Logs string `toml:"logs"`
	Port int    `toml:"port"`
}

type config struct {
	Version string    `toml:"version"`
	Haurno  HarunoCfg `toml:"haruno"`
}

// haruno 晴乃机器人
// 机器人运行的全局属性
type haruno struct {
	startTime time.Time
	port      int
	logpath   string
	version   string
}

var bot = new(haruno)

// Initialize 从配置文件读取配置初始化
func (bot *haruno) Initialize() {
	cfg := new(config)
	_, err := toml.DecodeFile("./config.toml", cfg)
	if err != nil {
		log.Fatal("Haruno Initialize fialed", err)
	}
	bot.startTime = time.Now()
	bot.port = cfg.Haurno.Port
	bot.logpath = cfg.Haurno.Logs
	bot.version = cfg.Version
	logger.Service.SetLogsPath(bot.logpath)
}

// Status 运行状态json格式
type Status struct {
	Fails   int   `json:"fails"`
	Success int   `json:"success"`
	Start   int64 `json:"start"`
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := new(Status)
	status.Fails = logger.Service.Fails()
	status.Success = logger.Service.Success()
	status.Start = bot.startTime.UnixNano() / 1e6
	json.NewEncoder(w).Encode(status)
}

// Run 启动机器人
func (bot *haruno) Run() {
	http.HandleFunc("/logs/-/type=websocket", logger.WSLogHandler)
	http.HandleFunc("/logs/-/type=plain", logger.RawLogHandler)
	http.HandleFunc("/status", statusHandler)
	addr := fmt.Sprintf("0.0.0.0:%d", bot.port)
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
