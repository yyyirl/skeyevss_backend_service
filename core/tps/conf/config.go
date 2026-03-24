// @Title        config
// @Description  main
// @Create       yiyiyi 2025/9/9 14:58

package conf

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"

	"skeyevss/core/tps"
)

type BackendApiConf struct {
	rest.RestConf

	ContainerizedState bool `json:",optional"`
	UseSipPrintLog     bool
	Version,
	ProductName,
	ActivateCodePath string
	WebsocketPort uint
	WebsocketHost,
	TMapKey,
	Domain,
	WSProxy string

	InitializeSetPassword bool
	MinioApiTarget,
	OSEnvironment,
	SevVolumesDir string

	Auth     tps.YamlAuth
	SavePath tps.YamlSavePath

	// Redis tps.YamlRedis
	Email tps.YamlEmail

	RedisHost string

	SevBase  tps.YamlSevBaseConfig
	SevRes   tps.YamlSevRes `json:"SevRes"`
	DBGrpc   zrpc.RpcClientConf
	Accounts tps.YamlAccounts

	Sip tps.YamlSip

	InternalIP,
	ExternalIP,
	VssHttpTarget,
	VssSseTarget,
	LogPath,
	SaveFileDir,
	SaveVideoDir string
	VssSseTargetFrontend       string `json:",optional"`
	VssHttpTargetFrontend      string `json:",optional"`
	WebProxyFileTargetFrontend string `json:",optional"`
	SipLogPath                 string `json:",optional"`

	RpcInterceptor tps.YamlRpcInterceptorConf

	EnvFile    string             `json:",optional"`
	ConfigPath tps.YamlConfigPath `json:"ConfigPath"`

	PProfPort    uint
	PProfFileDir string

	PProf              tps.YamlPProf
	GenUniqueId        tps.YamlGenUniqueId
	UseShowcaseAccount bool `json:",optional"`
}

type CronConfig struct {
	ContainerizedState bool `json:",optional"`
	Mode,
	Name string
	Log            logx.LogConf
	SevBase        tps.YamlSevBaseConfig `json:"SevBase"`
	Redis          tps.YamlRedis
	DBGrpc         zrpc.RpcClientConf
	UseSipPrintLog bool
	VssHttpTarget  string

	Http struct {
		VssHttpPort int
	}

	RpcInterceptor tps.YamlRpcInterceptorConf
	PProfPort      uint

	Sip tps.YamlSip

	InternalIP,
	ExternalIP,
	PProfFileDir string
}

type DBSevConf struct {
	tps.YamlFoundation

	InternalIp,
	ExternalIp string
	ContainerizedState bool   `json:",optional"`
	ActivateCodePath   string `json:",optional"`
	zrpc.RpcServerConf

	RedisHost            string
	SaveVideoSnapshotDir string

	Sip                tps.YamlSip
	CRedis             tps.YamlRedis
	Accounts           tps.YamlAccounts
	Databases          tps.YamlDatabases
	PProfPort          uint
	PProfFileDir       string
	UseShowcaseAccount bool `json:",optional"`
}

type VssSevConfig struct {
	ContainerizedState bool   `json:",optional"`
	Host               string `json:",default=0.0.0.0"`
	Port               int
	Timeout            int64
	Version,
	ProductName string
	MaxBytes int64 `json:",default=1048576"`
	UseSipLogToFile,
	UseSipPrintLog bool
	Mode string `json:",default=pro,options=dev|test|rt|pre|pro"`
	Name,
	InternalIp,
	ExternalIp string
	SaveVideoSnapshotDir,
	Domain string

	RpcInterceptor tps.YamlRpcInterceptorConf

	Log logx.LogConf

	SipLogPath string `json:",optional"`

	Sip   tps.YamlSip
	Onvif tps.YamlOnvif

	Http struct {
		Port int
	}
	SSE struct {
		Port int
		// MessageChanBuffer 每个 SSE 连接 messageChan 缓冲长度，缓解 SIP 日志等洪峰 0 表示使用服务端默认（256）
		MessageChanBuffer int `json:",optional"`
		// SipLogMaxPerSecond 单路 sip_logs 连接每秒最多推送的日志条数（收发合计），0 表示不限制仍受缓冲满丢弃策略影响
		SipLogMaxPerSecond int `json:",optional"`
	}

	XAuth tps.YamlAuth
	WS    struct {
		Port int
		ReadBufferMaxSize,
		WriteBufferMaxSize,
		WaitTimeOut,
		HeartbeatTimer int
		ClearTalkSipInterval,
		MaxLifetime,
		AuthorizationLifetime int64
		ReqTimeout uint64
	}
	DBGrpc              zrpc.RpcClientConf
	SevBase             tps.YamlSevBaseConfig
	StreamPlayProxyPath tps.YamlStreamPlayProxyPath
	Redis               tps.YamlRedis
	FFMpeg              tps.YamlFFMpeg

	PProfPort    uint
	PProfFileDir string

	SavePath tps.YamlSavePath
}

type WebConfig struct {
	ContainerizedState bool `json:",optional"`
	InternalIP,
	ExternalIP,
	Mode,
	Name string

	Log logx.LogConf

	SevBase      tps.YamlSevBaseConfig `json:"SevBase"`
	PProfPort    uint
	PProfFileDir string
}
