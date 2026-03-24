# VSS系统设计文档

VSS（Video Security System）是 Skeyevss 中的**国标信令与流控核心服务**，负责 GB28181 设备接入、SIP 信令、媒体回调、WebSocket/SSE
实时通道等。本文将会讲解包含运行流程、功能介绍、开发使用与注意事项等，并配一些示例代码。

---

## 一、概述

### 1.1 定位

- **国标 GBS**: 作为平台端接收设备注册、目录、Invite/ACK/BYE、MESSAGE（心跳/目录/设备信息等）。
- **国标级联 GBC**: 作为上级或下级平台，处理级联注册、目录、Invite 等。
- **HTTP API**: 对外提供拉流/停流、设备控制、媒体查询、媒体服务回调。
- **SSE / WebSocket**: 为前端提供设备诊断、SIP 日志、语音对讲等实时数据。

### 1.2 依赖

- **etcd**: 服务发现
- **DB RPC**: 数据中心
- **Redis**: 可选
- **媒体服务（SkeyesMS）**: 通过HTTP调用

### 1.3 端口（etc/.vss.yaml，.env）

| 用途        | 默认端口  | 说明              |
|-----------|-------|-----------------|
| SIP GBS   | 11008 | 国标设备信令 TCP/UDP  |
| SIP GBC   | 11015 | 国标级联信令 TCP/UDP  |
| HTTP      | 11013 | REST API + 媒体回调 |
| SSE       | 11014 | 事件推送            |
| WebSocket | 11018 | 语音对讲、状态推送等      |

---

## 二、运行流程

### 2.1 启动顺序（main.go）

```
1. 解析 -env、-f 参数，加载 .env 与 etc/.vss.yaml
2. 设置时区、日志、pprof
3. svc.NewServiceContext(c) 创建上下文（RPC 客户端、各类 channel、map）
4. initialize.DO(svcCtx, baseConf) 初始化（Broadcast 清理、SIP 日志目录、媒体服务配置等）
5. InitFetchDataState.Wait() 配合FetchDataLogic使用，先等数据拉取完成
6. 启动4个SIP服务（阻塞 Listen）: 
   - GBS TCP、GBS UDP、GBC UDP、GBC TCP
7. 启动 SSE 服务: go server.NewSSESev(svcCtx).Start()
8. 启动 WebSocket 服务: go server.NewWSSev(svcCtx).Start()
9. 启动 HTTP 服务: go server.NewHttpSev(svcCtx).Start()
10. InitFetchDataState.Add(2)，与proc中Done配合
11. 启动后台任务链 server.NewSipProc(svcCtx).DO(...)
12. <-stop等待 SIGTERM/SIGINT，然后Shutdown四个SIP服务
```

### 2.2 后台任务链（SipProc）

`DO` 中注册的每个 `SipProcLogic` 在独立 goroutine 中执行，且内部有 `recoverCall` 防止 panic 导致进程退出:

| 顺序 | Logic                       | 作用                                                           |
|----|-----------------------------|--------------------------------------------------------------|
| 1  | FetchDataLogic              | 从DBRPC拉字典、设置、媒体服务器列表、级联、设备在线状态; ONVIF发现; 定时刷新                |
| 2  | SendLogic                   | 消费channel: Catalog、Invite、Bye、设备控制、预设位、录像、订阅、语音对讲等，发SIP到真实设备 |
| 3  | CatalogLoopLogic            | 定时向已注册设备发 Catalog 请求（间隔 Sip.CatalogInterval）                 |
| 4  | HeartbeatOfflineLogic       | 心跳超时判定设备离线                                                   |
| 5  | SetDeviceOnlineStateLogic   | 上下线结果写入队列                                                    |
| 6  | CheckDeviceOnlineStateLogic | 消费队列，通过RPC更新设备/通道在线状态                                        |
| 7  | SipLogLogic                 | 写SIP收发日志（文件/广播）                                              |
| 8  | CascadeLoopLogic            | 级联注册循环                                                       |
| 9  | SendLoopLogic               | 级联消息发送循环                                                     |

**注意**: SIP 服务器要 `InitFetchDataState.Wait()` 后才 `Listen`，确保字典、设置等已拉取，避免程序运行时拿不到配置。

### 2.3 数据流简图

```
设备/NVR ──SIP(TCP/UDP)──► VSS (GBS/GBC)
                              │
                              ├─► SendLogic 等消费 channel 发 SIP
                              ├─► HTTP 回调 (媒体服务) ──► notify/* ──► 更新通道状态、保活等
                              ├─► HTTP API (Backend/前端) ──► invite/stop/deviceControl/...
                              ├─► SSE ──► 设备诊断、SIP 日志、状态
                              └─► WebSocket ──► 语音对讲、状态推送

VSS ──RPC──► DB (配置/设备/通道)
VSS ──HTTP──► 媒体服务 (start_rtp_pub, ack_rtp_pub, 停止流等)
```

---

## 三、目录结构

```
core/app/sev/vss/
├── main.go                   # 项目入口: flag、env、config、SIP/SSE/WS/HTTP、SipProc
├── internal/
│   ├── config/               # 配置（使用 tps.VssSevConfig）
│   ├── svc/                  # ServiceContext注入服务上下文，构造（RPC、Redis、各类 channel/map）
│   ├── types/                # 请求/响应/SIP 相关类型、ServiceContext 定义
│   ├── handler/
│   │   ├── http/             # Gin 路由注册、泛型 handler（newHandler/newHandlerWithParams）
│   │   ├── sse/              # SSE 路由（type=xxx 分发）
│   │   ├── ws/               # WebSocket 路由、广播
│   │   ├── gbs_sip/          # GBS SIP 方法注册（REGISTER/INVITE/ACK/BYE/MESSAGE）
│   │   └── gbc_sip/          # GBC SIP 方法注册
│   ├── logic/
│   │   ├── proc/             # 后台任务: FetchDataLogic
│   │   ├── gbs_proc/         # GBS 相关: Send、CatalogLoop、Heartbeat、OnlineState、SipLog
│   │   ├── gbc_proc/         # GBC: CascadeLoop、SendLoop
│   │   ├── gbs_sip/          # GBS SIP 处理: Register、Invite、ACK、Bye、Keepalive、Catalog、...
│   │   ├── gbc_sip/          # GBC SIP 处理，消息转发，可以理解为真实设备（中转）
│   │   ├── http/
│   │   │   ├── base/         # 状态、设备控制、预设位、录像、WS Token
│   │   │   ├── video/        # 流播放/停止、流信息
│   │   │   ├── gbs/          # Catalog、Invite、StopStream、回放控制、订阅
│   │   │   ├── gbc/          # 级联 Catalog
│   │   │   ├── ms/           # 媒体服务: 流组、录像、配置、Reload
│   │   │   ├── notify/       # 媒体回调: on_pub_start/stop、on_sub_start/stop、on_rtmp_connect 等
│   │   │   └── onvif/        # ONVIF 发现、设备信息、Profile
│   │   ├── sse/              # 设备/通道诊断、文件下载、服务状态、SIP 日志等
│   │   └── ws/               # 语音对讲: 发送/停止、SIP 注册、通道注册、广播
│   ├── server/               # SIP/HTTP/SSE/WS 服务封装
│   ├── interceptor/          # HTTP 超时、Header 等
│   └── pkg/                  # SIP 解析/发送、媒体服务 HTTP 封装、ONVIF、端口分配等
```

---

## 四、功能介绍

### 4.1 SIP（GBS）

- **REGISTER**: 设备注册，回复 200 OK; 触发目录订阅、心跳定时。
- **INVITE**: 设备侧发起（如对讲）; 服务端也可主动 Invite（直播/回放）通过 SendLogic 执行 `SipSendVideoLiveInvite`。
- **ACK**: 对Invite的确认，携带SDP，之后设备开始推流。
- **BYE**: 结束会话。
- **MESSAGE**: 根据 CmdType 分发为
  Keepalive、Catalog、DeviceInfo、ConfigDownload、PresetQuery、RecordInfo、Alarm、MediaStatus、Broadcast 等。

### 4.2 SIP（GBC 级联）

- 独立端口，对Register/Invite/ACK/BYE/MESSAGE 处理，实际上和真实设备处理信令一直。
- 级联注册、保活、目录、Invite 等由 gbc_proc 与 gbc_sip 配合完成。

### 4.3 HTTP API 分组

- **base**: 状态、设备控制、预设位、录像查询、WS Token。
- **video**: 获取播放地址、停止播放、流信息。
- **ms**: 媒体服务流组、按流名查录像、配置、Reload。
- **gbs**: Catalog、Invite（直播/回放）、StopStream、回放控制、订阅。
- **gbc**: 级联 Catalog。
- **onvif**: 发现、设备信息、Profile。
- **notify**: 媒体服务回调。

### 4.4 媒体服务回调（notify）

媒体服务按配置调用 VSS 的 POST 接口，VSS 更新通道状态、做保活或停流:

| 路径                                                   | 含义              |
|------------------------------------------------------|-----------------|
| /api/notify/on-pub-start                             | 设备向MS开始推流       |
| /api/notify/on-pub-stop                              | 推流停止            |
| /api/notify/on-push-start / on-push-stop             | 作为下级推给上级        |
| /api/notify/on-reply-pull-start / on-reply-pull-stop | 拉流开始/停止         |
| /api/notify/on-rtmp-connect                          | 有RTMP推流连接建立的事件通知   |
| /api/notify/on-sub-start / on-sub-stop               | 播放开始/停止（做实时流保活） |

实现: 解析 `streamName` → 校验通道存在 → 更新DB中通道stream_state/online 等，流还会在on_pub_stop时发BYE停流。

### 4.5 SSE

- 连接: `GET /events?type=xxx`。
- `type` 取值: 设备诊断、通道诊断、文件下载、服务状态、设备在线状态、SIP 日志等（详细见 `handler/sse/routers.go`，也是通过这个文件注册）。
- 通过 `messageChan` 向客户端推事件。

### 4.6 WebSocket

- 连接: `/`，需携带合法 token（通过 backendApi`/api/base/ws-token`获取）。
- 消息体带 `type` 路由到对应 handler（见 `handler/ws/register.go`），如:
    - 语音对讲: 发送音频、停止、SIP 注册、通道注册;
    - 广播: 对讲 SIP 状态、使用状态等。
    - 后续功能增加处理对应`register.go`

---

## 五、配置说明

### 5.1 配置文件

- 路径: `etc/.vss.yaml`，内容由环境变量占位（如 `${SKEYEVSS_VSS_HTTP_PORT}`），需配合 .env 使用。

### 5.2 关键配置项（core/tps/conf 中 VssSevConfig）

| 配置                          | 说明                 |
|-----------------------------|--------------------|
| Host / Port                 | SIP 监听地址与端口（GBS）   |
| Sip.CascadeSipPort          | 级联 SIP 端口          |
| Http.Port                   | HTTP API 端口        |
| SSE.Port / WS.Port          | SSE / WebSocket 端口 |
| Sip.CatalogInterval         | 定时 Catalog 间隔（秒）   |
| Sip.LifetimeTimeoutInterval | 心跳超时（秒）            |
| DBGrpc                      | 通过 Etcd 发现 DB RPC  |
| Onvif                       | ONVIF 多播地址、端口、发现超时 |

---

## 六、开发使用

### 6.1 本地运行

```bash
# 确保 etcd、DB RPC、媒体服务已启动，.env 已配置
go run main.go -env .env.local -f etc/.vss.yaml
```

### 6.2 新增 HTTP 接口（无请求体）

1. 在`internal/logic/http/xxx`下新增Logic，需要实现`types.HttpEHandleLogic`:

```
// 实现 HttpEHandleLogic[*XxxLogic]
var (
    _ types.HttpEHandleLogic[*XxxLogic] = (*XxxLogic)(nil)
    XxxLogic = new(xxxLogic)
)

type xxxLogic struct {
    ctx    context.Context
    c      *gin.Context
    svcCtx *types.ServiceContext
}

func (l *xxxLogic) New(ctx context.Context, c *gin.Context, svcCtx *types.ServiceContext) *xxxLogic {
    return &xxxLogic{ctx: ctx, c: c, svcCtx: svcCtx}
}

func (l *xxxLogic) Path() string {
    return "/xxx/path"
}

func (l *xxxLogic) DO() *types.HttpResponse {
    // 使用 l.ctx, l.c, l.svcCtx 处理，返回 types.HttpResponse
    return &types.HttpResponse{Data: result}
}
```

2. 在 `internal/handler/http/routers.go` 中注册:

```
router.GET(xxxLogic.Path(), newHandler(svcCtx, xxxLogic))
```

### 6.3 新增HTTP接口（带请求体）

1. 定义请求类型（如 `types.XxxReq`），Logic 实现 `types.HttpRHandleLogic[Logic, Req]`:

```
var _ types.HttpRHandleLogic[*XxxLogic, types.XxxReq] = (*XxxLogic)(nil)

func (l *xxxLogic) DO(req types.XxxReq) *types.HttpResponse {
    // 使用 req 与 l.svcCtx
    return &types.HttpResponse{Data: result}
}
```

2. 注册:

```
router.POST(xxxLogic.Path(), newHandlerWithParams[types.XxxReq](svcCtx, xxxLogic))
```

### 6.4 新增媒体回调（notify）

1. 在 `internal/logic/http/notify` 下新增 Logic，实现 `HttpRHandleLogic`，`Path()` 返回如 `/notify/on-xxx`。
2. 在 `routers.go` 中注册POST/GET:

```
router.POST(notify.VOnXxxLogic.Path(), newHandlerWithParams[types.NotifyStreamReq](svcCtx, notify.VOnXxxLogic))
```

3. 若逻辑与on_pub_start/on_pub_stop类似（仅更新流状态），可复用`setStreamState`（见`notify/common.go`）。

### 6.5 新增后台任务（SipProcLogic）

1. 在 `internal/logic/gbs_proc` 或新建包中实现:

```
var _ types.SipProcLogic = (*XxxProcLogic)(nil)

type XxxProcLogic struct {
    svcCtx      *types.ServiceContext
    recoverCall func (name string)
}

func (l *XxxProcLogic) DO(params *types.DOProcLogicParams) {
    l.svcCtx = params.SvcCtx
    l.recoverCall = params.RecoverCall
    defer l.recoverCall("XxxProcLogic")
    // 循环 select channel 或 ticker...
}
```

2. 在 `main.go` 的 `server.NewSipProc(svcCtx).DO(...)` 中追加:

```
new(xxxpkg.XxxProcLogic),
```

### 6.6 发送 SIP 请求（通过channel）

业务侧不直接发SIP，而是向`svcCtx`的channel投递，由SendLogic统一发送，例如:

```
// Catalog
l.svcCtx.SipSendCatalog <- &types.Request{ID: deviceId, Req: req, ...}

// Invite（通常由HTTP invite逻辑组织好，再SipVideoLiveInviteMessage后投递）
l.svcCtx.SipSendVideoLiveInvite <- &types.SipVideoLiveInviteMessage{...}

// BYE
l.svcCtx.SipSendBye <- &types.SipByeMessage{StreamName: streamName}
```

---

## 七、注意事项

### 7.1 启动顺序

- 必须先启动**etcd**和**DBRPC**，再启动VSS，否则RPC连接失败会导致FetchData失败、InitFetchDataState不Done，内置服务将无法正常启动。
- **媒体服务** 在Vss启动时需要更新检测配置并启动

### 7.2 端口与防火墙

- SIP端口（GBS/GBC）需对设备/上级平台开放;RTP端口范围（Invite时由媒体服务分配）也要开放（需要和.env配置文件一致）。
- 若VSS与媒体服务不在同一机，媒体服务回调的VSS地址要填对（内网或外网），否则将会无法播放、状态无法正常更新。

### 7.3 并发与 channel

- `SipSendVideoLiveInvite`、`SipSendBye` 等channel有缓冲，避免慢消费时阻塞调用方;但不要无限制投递，防止内存与goroutine暴涨。
- `AckRequestMap`、`PubStreamExistsState`、`SipCatalogLoopMap`等由多goroutine访问，实现为`xmap`/`set`等并发安全结构。

### 7.4 流名称与锁❗❗❗❗

- **同一个`streamName`的Invite流程会加锁（见invite逻辑），避免重复Invite或状态错乱**。
- **停流时要同时:发BYE、删`AckRequestMap`、删`PubStreamExistsState`，并通知媒体服务关流**。

### 7.5 配置与环境变量

- `.vss.yaml`中大量`${SKEYEVSS_*}`，部署前确认.env中已导出且无误，否则端口为空会启动失败或监听异常。
- 修改SIP端口、HTTP端口、媒体回调地址后，需同时检查媒体服务配置里的回调URL与VSS实际地址一致。

### 7.6 日志与排查

- SIP 收发可开启 `UseSipPrintLog` 或写文件（`SipLogPath`），便于排查信令问题。
- 使用pprof时注意`PProfPort`、`PProfFileDir`配置，避免与其它服务冲突。

## 八、系统设计图

```mermaid
graph TB
  classDef config fill: #e1f5fe, stroke: #01579b, stroke-width: 2px, font-size: 30px
  classDef main fill: #fff3e0, stroke: #e65100, stroke-width: 2px, font-size: 30px
  classDef gbs fill: #f3e5f5, stroke: #4a148c, stroke-width: 2px, font-size: 30px
  classDef gbc fill: #ffebee, stroke: #b71c1c, stroke-width: 2px, font-size: 30px
  classDef sse fill: #e0f7fa, stroke: #006064, stroke-width: 2px, font-size: 30px
  classDef ws fill: #fff9c4, stroke: #f57f17, stroke-width: 2px, font-size: 30px
  classDef http fill: #f1d6fa, stroke: #6a1b9a, stroke-width: 2px, font-size: 30px
  classDef proc fill: #ede7f6, stroke: #311b92, stroke-width: 2px, font-size: 30px
  classDef device fill: #e0f2f1, stroke: #004d40, stroke-width: 2px, font-size: 30px

  subgraph Top [<b>系统配置与初始化</b>]
    direction TB
    C1[环境变量配置<br/>.env.dev/.env.test/.env.prod]
    C2[vss.yml配置中心]
    C1 --> C2
    M1[main函数启动]
    M2[加载环境变量 .env.$env]
    M3[解析配置 vss.yml]
    M4[初始化pprof性能分析]
    M5[初始化ServiceContext]
    C2 --> M1
    M1 --> M2 --> M3 --> M4 --> M5
  end

  subgraph Middle [<b>启动服务</b>]
    direction TB

    subgraph SIP_Row [<b>SIP信令服务</b>]
      direction TB
      GBS_Box[GBS服务器 - 上级信令]
      GBS_TCP[GBS TCP异步]
      GBS_UDP[GBS UDP异步]
      GBS_Addr[生成地址启动服务]
      GBS_Route[注册路由 handlers]
      GBS_Box --> GBS_TCP
      GBS_Box --> GBS_UDP
      GBS_TCP --> GBS_Addr
      GBS_UDP --> GBS_Addr
      GBS_Addr --> GBS_Route

      subgraph GBS_Proc [GBS请求处理流程]
        GBS_Parse[解析请求] --> GBS_Timeout[检测超时]
        GBS_Timeout --> GBS_Call[调用Logic]
        GBS_Call --> GBS_Judge{判断Code}
        GBS_Judge -->|Success| GBS_Succ[返回success]
        GBS_Judge -->|Error| GBS_Err[返回error]
      end
      GBS_Route --> GBS_Proc
      GBC_Box[GBC服务器 - 国标级联]
      GBC_TCP[GBC TCP异步]
      GBC_UDP[GBC UDP异步]
      GBC_Addr[生成地址启动服务]
      GBC_Route[注册路由 handlers]
      GBC_Box --> GBC_TCP
      GBC_Box --> GBC_UDP
      GBC_TCP --> GBC_Addr
      GBC_UDP --> GBC_Addr
      GBC_Addr --> GBC_Route

      subgraph GBC_Proc [GBC级联转发流程]
        GBC_Front[前端A请求] --> GBC_Reg[注册到级联平台]
        GBC_Reg --> GBC_Handle[作为设备处理请求]
        GBC_Handle --> GBC_Send[转发到真实设备C]
        GBC_Send --> GBC_Resp[设备响应返回]
        GBC_Resp --> GBC_Forward[转发到上级GBS]
      end
      GBC_Route --> GBC_Proc
    end

    subgraph Other_Row [<b>其他服务</b>]
      direction TB
      SSE_Box[SSE服务器 - 事件推送]
      SSE_Addr[生成地址 Host:Port]
      SSE_Endpoint[events 端点]
      SSE_Handler[SSE Handler流程]
      SSE_Box --> SSE_Addr --> SSE_Endpoint --> SSE_Handler

      subgraph SSE_Detail [SSE处理]
        SSE_Header[设置SSE Headers] --> SSE_Chan[创建messageChan]
        SSE_Chan --> SSE_Router[注册Router]
        SSE_Router --> SSE_Loop[遍历messageChan]
        SSE_Loop --> SSE_Write[写入Response并Flush]
        SSE_Loop --> SSE_Delay{判断DelayClose}
        SSE_Delay -->|true| SSE_CloseDelay[延时2s关闭]
        SSE_Delay -->|false| SSE_CloseNow[立即关闭]
      end
      SSE_Handler --> SSE_Detail
      WS_Box[WebSocket服务器]
      WS_Addr[生成地址]
      WS_Endpoint[ws 端点]
      WS_Timer[定时器 链接检测]
      WS_Upgrade[WebSocket Upgrade]
      WS_Box --> WS_Addr --> WS_Endpoint --> WS_Timer --> WS_Upgrade

      subgraph WS_Detail [WebSocket处理流程]
        direction TB
        WS_Read[reader协程解析消息] --> WS_Chan1[message processor channel]
        WS_Chan1 --> WS_Dispatch[dispatcher分发]
        WS_Dispatch --> WS_Logic[调用对应Logic]
        WS_Logic --> WS_Chan2[response message channel]
        WS_Chan2 --> WS_Write[返回给客户端]
        WS_State[closed state channel] --> WS_Close[关闭链接释放资源]
        WS_Heartbeat[心跳检测] --> WS_Active[更新ActiveTime]
      end
      WS_Upgrade --> WS_Detail
      HTTP_Box[HTTP API服务器 - Gin框架]
      HTTP_Addr[生成地址]
      HTTP_Group[api 路由组]
      HTTP_Middle[中间件层]
      HTTP_Routes[路由注册]
      HTTP_Box --> HTTP_Addr --> HTTP_Group --> HTTP_Middle --> HTTP_Routes

      subgraph HTTP_MW [HTTP中间件]
        HTTP_CORS[CORS跨域配置]
        HTTP_Header[HttpHeader拦截器]
        HTTP_Timeout[Timeout超时控制]
      end
      HTTP_Middle --> HTTP_MW

      subgraph HTTP_RouteDetail [API路由组]
        HTTP_Basic[基础路由]
        HTTP_Video[视频路由]
        HTTP_GBS[GBS路由]
        HTTP_GBC[GBC路由]
        HTTP_Onvif[Onvif路由]
        HTTP_Notify[Notify通知]
      end
      HTTP_Routes --> HTTP_RouteDetail
    end
  end

  subgraph Bottom [<b>任务注册层 - server.NewSipProc</b>]
    direction TB
    T1[SIP处理器]
    T2[数据获取 FetchDataLogic]
    T3[GBS任务组<br>]
    T4[GBC任务组<br>]
    T1 --> T2
    T1 --> T3
    T1 --> T4

    subgraph GBS_Tasks [GBS任务组]
      GBS_Send[请求发送 SendLogic]
      GBS_Catalog[定时Catalog CatalogLoopLogic]
      GBS_Heartbeat[心跳检测 HeartbeatOfflineLogic]
      GBS_State[更新设备状态 SetDeviceOnlineStateLogic]
      GBS_Check[检测在线 CheckDeviceOnlineStateLogic]
      GBS_Log[SIP日志 SipLogLogic]
      GBS_Send --> GBS_Catalog --> GBS_Heartbeat --> GBS_State --> GBS_Check --> GBS_Log
    end

    subgraph GBC_Tasks [GBC任务组]
      GBC_RegLoop[设备注册 CascadeLoopLogic]
      GBC_SendLoop[消息发送 SendLoopLogic]
      GBC_RegLoop --> GBC_SendLoop
    end

    T3 --> GBS_Send
    T4 --> GBC_RegLoop
  end

  subgraph External [<b>外部实体</b>]
    Device[终端设备<br/>摄像头/NVR]
    WS_Client[WebSocket客户端<br/>前端浏览器/App]
    SSE_Client[SSE客户端<br/>事件监听端]
    API_Client[API调用方<br/>第三方系统]
  end

  M5 ==> GBS_Box
  M5 ==> GBC_Box
  M5 ==> SSE_Box
  M5 ==> WS_Box
  M5 ==> HTTP_Box
  M5 ==> T1
  GBC_Forward ==> GBS_Proc
  GBC_Send ==> Device
  Device ==> GBC_Resp
  WS_Detail ==> WS_Client
  WS_Client ==> WS_Read
  SSE_Detail ==> SSE_Client
  HTTP_RouteDetail ==> API_Client
  API_Client ==> HTTP_RouteDetail
class C1,C2 config
class M1,M2,M3,M4,M5 main
class GBS_Box,GBS_TCP,GBS_UDP,GBS_Addr,GBS_Route,GBS_Proc gbs
class GBC_Box,GBC_TCP,GBC_UDP,GBC_Addr,GBC_Route,GBC_Proc gbc
class SSE_Box,SSE_Addr,SSE_Endpoint,SSE_Handler,SSE_Detail sse
class WS_Box,WS_Addr,WS_Endpoint,WS_Timer,WS_Upgrade,WS_Detail ws
class HTTP_Box,HTTP_Addr,HTTP_Group,HTTP_Middle,HTTP_Routes,HTTP_MW,HTTP_RouteDetail http
class T1,T2,T3,T4,GBS_Tasks,GBC_Tasks,GBS_Send,GBS_Catalog,GBS_Heartbeat,GBS_State,GBS_Check,GBS_Log,GBC_RegLoop,GBC_SendLoop proc
class Device,WS_Client,SSE_Client,API_Client device
```