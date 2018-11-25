# Haruno(晴乃)

基于酷Q插件cqhttp提供的API开发的，具有拓展性的QQ聊天机器人。

## 安装

```
$ git clone https://github.com/haruno-bot/haruno.git
$ cd haruno
$ go build
```

## 特性

* 使用Go语言
* 能在windows, linux, mac osx等平台运行
* 支持插件
* 有完整的log系统
* 功能增强的http、websocket客户端

## 环境

1. 需要安装酷Q
2. 需要安装[CoolQ HTTP API 插件](https://cqhttp.cc/)
3. 必须开放websocket连接，http可选（不开放http可能部分”非重要”功能无法使用）

## 插件

插件示例如下：

1. [转推插件](https://github.com/haruno-bot/retweet)
2. [图灵机器人插件](https://github.com/haruno-bot/turing)

插件开发文档：

[文档](https://github.com/haruno-bot/haruno/tree/master/plugins/README.md)

## License

[The MIT License](https://github.com/haruno-bot/haruno/blob/master/LICENSE)
