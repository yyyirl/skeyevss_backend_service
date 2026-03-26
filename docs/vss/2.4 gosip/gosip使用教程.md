## GoSIP 使用教程：从 Server 到收发 SIP 请求

本教程面向希望在 Go 业务里集成 SIP 能力的开发者，基于 `github.com/ghettovoice/gosip` 的现有代码结构，给出一套“最小可运行”的使用路径：

1. 创建 `gosip.Server`
2. `Listen` 监听 UDP/TCP
3. `OnRequest` 注册服务端处理回调
4. 使用 `sip.RequestBuilder` 构建客户端请求（例如 `OPTIONS`）
5. 调用 `RequestWithContext` 发起请求并等待最终响应

协议标准参考：RFC 3261（[IETF](https://tools.ietf.org/html/rfc3261)）

**项目地址** [https://github.com/openskeye/go-vss](https://github.com/openskeye/go-vss)

---

## 1. 最小可运行示例：监听 `OPTIONS` 并做回包

下面示例做两件事：

- 服务端监听 `127.0.0.1:5060`，注册 `OPTIONS` 的 handler
- 客户端从同一进程发起一个 `OPTIONS` 请求到服务端，打印返回结果

> 说明：为了让示例更聚焦，示例只实现 `OPTIONS`。事务状态机、重传、解析、ACK 等底层能力由 gosip 自动处理。

```go
package main

import (
	"context"
	"fmt"
	"time"

	gosip "github.com/ghettovoice/gosip"
	gosiplog "github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
)

func main() {
	var (
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	)
	defer cancel()

	var logger = gosiplog.NewDefaultLogrusLogger()

	var srv = gosip.NewServer(
		gosip.ServerConfig{
			Host:     "127.0.0.1",
			UserAgent: "GoSIP-Tutorial",
		},
		nil,
		nil,
		logger,
	)

	var listenErr = srv.Listen("udp", "127.0.0.1:5060")
	if listenErr != nil {
		panic(listenErr)
	}

	var optionsErr = srv.OnRequest(sip.OPTIONS, func(req sip.Request, tx sip.ServerTransaction) {
		var res = sip.NewResponseFromRequest("", req, 200, "OK", "")
		_ = tx.Respond(res)
	})
	if optionsErr != nil {
		panic(optionsErr)
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()

	// -------------------- 客户端请求 --------------------
	var port sip.Port = 5060

	var (
		recipient = &sip.SipUri{
			FHost: "127.0.0.1",
			FPort: &port,
		}
	)

	var (
		fromTag = sip.String{Str: "from-tag-1"}
		toTag   = sip.String{Str: "to-tag-1"}
	)

	var (
		from = &sip.Address{
			Uri: &sip.SipUri{
				FHost: "127.0.0.1",
				FPort: &port,
			},
			Params: sip.NewParams().
				Add("tag", fromTag),
		}
		to = &sip.Address{
			Uri: &sip.SipUri{
				FHost: "127.0.0.1",
				FPort: &port,
			},
			Params: sip.NewParams().
				Add("tag", toTag),
		}
	)

	var (
		viaHop = &sip.ViaHop{
			ProtocolName:    "SIP",
			ProtocolVersion: "2.0",
			Transport:       "UDP",
			Host:            "127.0.0.1",
			Params: sip.NewParams().
				Add("branch", sip.String{Str: sip.GenerateBranch()}),
		}
	)

	var (
		builder = sip.NewRequestBuilder().
			SetTransport("UDP").
			SetHost("127.0.0.1").
			SetMethod(sip.OPTIONS).
			SetRecipient(recipient).
			SetFrom(from).
			SetTo(to)
	)

	builder = builder.AddVia(viaHop)

	var (
		req, buildErr = builder.Build()
	)
	if buildErr != nil {
		panic(buildErr)
	}

	var (
		resp, reqErr = srv.RequestWithContext(ctx, req)
	)
	if reqErr != nil {
		panic(reqErr)
	}

	fmt.Printf("OPTIONS 返回：%d %s\n", resp.StatusCode(), resp.Reason())

	// 给 goroutine 收尾一点时间
	time.Sleep(200 * time.Millisecond)
}
```

---

## 2. Server 的关键 API 怎么用

### 2.1 创建 Server

核心构造：

- `gosip.NewServer(config, tpFactory, txFactory, logger)`

常用字段：

- `ServerConfig.Host`：本机对外地址/域名（示例用 `127.0.0.1`）
- `ServerConfig.UserAgent`：server 侧会在发送时自动补齐 `User-Agent`

如果你不需要自定义传输/事务工厂：

- `tpFactory` / `txFactory` 传 `nil` 即可使用默认实现

### 2.2 监听端口：`Listen`

- `srv.Listen(network, listenAddr, options...)`

示例里使用：

- `network = "udp"`
- `listenAddr = "127.0.0.1:5060"`

对于 `tls/tcp/ws/wss`，你可以在后续扩展 ListenOption（例如 TLSConfig）。

### 2.3 注册服务端回调：`OnRequest`

- `srv.OnRequest(method, handler)`

handler 签名：

- `func(req sip.Request, tx sip.ServerTransaction)`

在 handler 内一般做：

- 构造 `sip.NewResponseFromRequest(...)`
- 调用 `tx.Respond(res)` 回包

### 2.4 优雅关闭：`Shutdown`

- `srv.Shutdown()` 会停止 transaction layer、transport layer，并等待 handler 完成。

建议在业务退出时统一调用，避免资源泄露或端口占用。

---

## 3. 客户端怎么发请求：RequestBuilder + RequestWithContext

### 3.1 构建请求：`sip.RequestBuilder`

你需要至少设置：

- `SetMethod(sip.XXX)`
- `SetRecipient(sip.Uri)`
- `SetFrom(*sip.Address)`
- `SetTo(*sip.Address)`

并且要添加：

- `AddVia(*sip.ViaHop)`：transport 在发送时要求存在 Via Header，用于确定 sent-by 等信息

如果你使用 `sip.GenerateBranch()` 给 branch 赋值，事务匹配会更符合 RFC 3261 的路径。

### 3.2 发起请求并等待最终响应：`RequestWithContext`

- `resp, err := srv.RequestWithContext(ctx, req, options...)`

返回的 `resp` 是最终响应：

- success（2xx）才会返回
- provisional（1xx）不会作为最终返回值，而是在事务内部被累积为 `Previous()`（你也可以用 ResponseHandler 逐条处理）

示例里：

- 直接打印 `resp.StatusCode()` 与 `resp.Reason()`

