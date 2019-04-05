# 晴乃插件开发文档

## 插件介绍

插件接口的定义：`coolq/cqplugin.go`

```go
type pluginInterface interface {
	Name() string
	Load() error
	Filters() map[string]Filter
	Handlers() map[string]Handler
	Loaded()
}
```

也就是说一个具备插件特性的实例必须至少实现 `Name()`, `Load()`, `Filters()`, `Handlers()`, `Loaded()`方法。

下面介绍着几个方法的含义：

### 插件名称 - `Name() string`

这个方法是得到插件名称的方法，用于在插件系统区别不同的插件使用的。

> 注意：千万不要和别的插件冲突！！！一般使用 `插件名称@版本号` 作为返回值。

### 插件加载 - `Load() error`

这个函数式用来加载插件的，返回值为一个 `error` 或者 `nil`，如果出现错误，机器人则无法正常加载该插件。

### 加载结束 - `Loaded()`

这个方法是插件加载结束的钩子，为异步调用，不会阻塞主线程。

### 过滤器和处理器 - `Filters(), Handlers()`

过滤器是过滤 coolq http api 插件的事件上报的数据的。因为数据上报的数据量非常的大，需要针对自己插件想要得到的事件去处理对应的事件即可。

对于一个插件，每一个filter都应该有一个handler与之对应。反之则不需要，没有filter对应的handler会默认全部处理。

下面是插件处理上报事件数据的过程：

![处理上报事件数据的过程](https://miao.su/images/2018/09/13/c1496e9cd0a0c6874fdf1.png)

事件一次经过每一个filter，如果通过则异步调用handler。

每一个插件都可以设置多个匹配的key来对应不同的匹配结果。这个是自己根据需求设置的。

### 插件加载过程

插件加载过程：

![插件加载过程](https://miao.su/images/2018/09/13/2c6ac6.png)

## 插件开发

插件接口的所有方法都有默认的实现，被实现为`coolq.Plugin`结构。

开发是可以直接：

```go
type MyPlugin struct {
    coolq.Plugin
}
```

实现一个基本的插件，然后再去重写自己的方法。

> 注意：接口调用的方法并不是使用实例的指针，所以不会把实例内部的变量改变后的值传下去。不过可以使用包内部的变量解决。

## 全局结构

### 日志服务 - logger.Service

一个全局的logger服务，提供持久化的文本日志和websocket实时的日志两种方式。

> 并不是同步的日志，即使是websocket方式，也是要通过管道发送。

<del>日志会每隔30s清空队列，并持久化。</del>

日志最大会在管道内保存5条，即打开web端页面能看到最新的5条日志信息。


### 常用方法

#### logger.(Service.)Success(text string)

记录一条成功的信息。

#### logger.(Service.)Successf(format string, args ...interface{})

使用格式化字符串的方法记录一条成功的信息。

类似的方法还有：Info, Infof(信息), Error和Errorf(错误)。

#### logger.(Service.)Field(name string)

创建一个带field的logger，此时所有的日志都会使用下面的方式输出。

```go
fmt.Sprintf("%s: %s" field, text)
```

### <del>基本</del>底层方法：

#### logger.Service.AddLog(ltype int, text string)

加入一个新的日志记录到日志队列，`ltype` 的可选值为：

```go
// LogTypeInfo 信息类型
const LogTypeInfo = 0

// LogTypeError 错误类型
const LogTypeError = 1

// LogTypeSuccess 成功on类型
const LogTypeSuccess = 2
```

#### logger.NewLog(ltype int, text string)

返回一个日志记录的指针，这个方法存在的意义是允许对日志记录进行一定的操作。

配合加入队列的方法：

`logger.Service.Add(lg *Log)` 完成加入日志队列操作。

### 客户端结构 - clients

clients目录下包括了两种客户端结构：http和websocket。

分别对原生的功能进行了一定的扩展。

http客户端的特性：

1. 支持预设header
2. 支持cookie jar
3. 支持代理

默认的http客户端为 `clients.DefaultHTTPClient`。如果没有特殊需求，请尽量使用默认的客户端。

websocket客户端的特性：

1. 支持事件机制
2. 可以断线重连

默认websocket只能连接一个服务，所以没有默认客户端。

更多详情，请看源码：

`clients/http.go` 和 `clients/ws.go`。

### 酷Q客户端 - coolq.Client

这是个直接和 coolq http api 连接的客户端，包含了一个api连接(ws)，一个event连接(ws)，一个http连接。

用于和 http api 通信使用，全局只存在一个。

> 并没有实现所有的api，目前只会实现action下面的部分。

并不是所有的api都可以用ws实现的，部分要求响应的会使用http实现。

## 注册插件

考虑到go的plugin目前依旧不稳定，目前插件采用静态加载的方式。等稳定之后，将会切成动态加载。

需要修改 `plugins/plugins.go` 文件：

```go
package plugins

import (
    "github.com/username/myplugin"
)

// SetupPlugins 安装插件的入口
func SetupPlugins() {
    // 注册插件的实例
    coolq.PluginRegister(myplugin.Instance)
}
```

然后静态编译。

## 问题

如果开发的过程中遇到问题，请在开issue。或者email我：

macchenjl#foxmail.com (# => @)
