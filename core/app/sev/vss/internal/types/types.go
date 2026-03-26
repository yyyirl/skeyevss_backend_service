package types

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ghettovoice/gosip"
	"github.com/ghettovoice/gosip/sip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/use-go/onvif/xsd/onvif"

	"skeyevss/core/app/sev/vss/internal/config"
	"skeyevss/core/common/client"
	"skeyevss/core/common/stream"
	cTypes "skeyevss/core/common/types"
	"skeyevss/core/pkg/audio"
	"skeyevss/core/pkg/broadcast"
	"skeyevss/core/pkg/categories"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/functions/download"
	"skeyevss/core/pkg/response"
	"skeyevss/core/pkg/set"
	"skeyevss/core/pkg/xmap"
	"skeyevss/core/repositories/models/cascade"
	"skeyevss/core/repositories/models/channels"
	"skeyevss/core/repositories/models/devices"
	"skeyevss/core/repositories/models/dictionaries"
	mediaServers "skeyevss/core/repositories/models/media-servers"
	"skeyevss/core/repositories/models/settings"
	"skeyevss/core/repositories/redis"
	"skeyevss/core/tps"
)

type HType map[sip.RequestMethod]gosip.RequestHandler

type Request struct {
	Authorization []sip.Header
	URI           sip.Uri
	ID,
	Body,
	Source string
	TransportProtocol string
	DeviceAddr        sip.Address
	Original          sip.Request

	Caller string
}

type (
	codeType = sip.StatusCode
)

const (
	StatusBadRequest         codeType = http.StatusBadRequest
	StatusUnauthorized       codeType = http.StatusUnauthorized
	StatusForbidden          codeType = http.StatusForbidden
	StatusPreconditionFailed codeType = http.StatusPreconditionFailed
)

type XError struct {
	Message string
}

func NewErr(message string) *XError {
	return &XError{Message: fmt.Sprintf("%s%s", message, functions.Caller(2))}
}

func (e *XError) Error() string {
	return e.Message
}

var (
	DisableResponseError = errors.New("response disabled 服务端将不再响应客户端")
)

type Response struct {
	Error  *XError
	Code   codeType
	Data   string
	Ignore bool // 忽略响应

	BeforeResponse func(resp sip.Response) sip.Response
}

type StepRecordMessageSipContent struct {
	Content string
	Type    string
}

type StepRecordMessage struct {
	Message    string
	Error      error
	Done       bool
	SipContent *StepRecordMessageSipContent
}

type StepRecord struct {
	Message chan *StepRecordMessage
}

const (
	BroadcastTypeSipReceive         = "SipSevReceive"
	BroadcastTypeSipRequest         = "SipSevRequest"
	BroadcastTypeSipResponse        = "SipSevResponse"
	BroadcastTypeSipReceiveResponse = "SipSevReceiveResponse"
)

type (
	SipCatalogLoopReq struct {
		Req    *Request
		Online bool
		Now    int64
	}

	SipVideoLiveInviteMessage struct {
		StreamPort     uint
		MediaTransMode string
		MediaServerUrl string

		MediaServerIP   string
		MediaServerPort uint

		StreamName string
		// play playback
		PlayType        stream.PlayType
		ChannelUniqueId string
		DeviceUniqueId  string

		TransportProtocol *devices.TransportProtocol

		StartAt  string
		EndAt    string
		Download bool
		Speed    float64

		MediaProtocolMode uint
		Req               *Request

		Data interface{}

		StepInfo *StepRecord

		Caller string
	}

	SipTalkInviteMessage struct {
		ChannelUniqueId string        `json:"channelUniqueId"`
		DeviceUniqueId  string        `json:"deviceUniqueId"`
		Req             *Request      `json:"req"`
		DeviceRow       *devices.Item `json:"deviceRow"`
	}

	SipByeMessage struct {
		Data       *stream.Item
		StreamName string
	}

	SipHeartbeatLoopReq struct {
		ID string
		Now,
		RegisterExpireAt int64
	}

	DCOnlineReq struct {
		DeviceUniqueId,
		ChannelUniqueId string
		CId    uint64
		Online bool
	}

	SipLogItem struct {
		Content string
		Type    string
	}

	// PlaybackControlItem struct {
	// 	CellID sip.CallID
	// 	AckReq *SendSipRequest
	// }

	CascadeRegisterItem struct {
		Last,
		Current *cascade.Item
	}

	GBCRegisterChanItem struct {
		Caller string
		Item   *cascade.Item
	}

	GBSSipSendTalk struct {
		Data           []byte
		DeviceUniqueId string
		Stop           bool
		StopCaller     string
	}

	ServiceContext struct {
		Config      config.Config
		RpcClients  *client.GRPCClients
		RedisClient *redis.Client

		// websocket
		WSClientCache     *WSClientsCache
		WSProc            *WSProc
		WSTalkUsageStatus *xmap.XMap[string, string]

		// 对讲前请求sip完成状态 Broadcast -> invite -> ack
		TalkSipData *xmap.XMap[string, *audio.TalkSessionItem]
		// 对讲前请求sip状态
		TalkSipSendStatus *set.CSet[string]

		SipSendCatalog, // 发送catalog
		SipSendDeviceInfo chan *Request // 发送deviceConfig
		SipSendVideoLiveInvite   chan *SipVideoLiveInviteMessage   // 发送invite 视频播放
		SipSendTalkInvite        chan *SipTalkInviteMessage        // 发送invite 拾音
		SipSendBye               chan *SipByeMessage               // 发送bye
		SipSendDeviceControl     chan *DeviceControlReq            // 发送device control
		SipSendQueryPresetPoints chan *SipSendQueryPresetPointsReq // 发送preset point查询
		SipSendSetPresetPoints   chan *SipSendSetPresetPointsReq   // 发送设置preset point
		SipSendQueryVideoRecords chan *QueryVideoRecordsReq        // 发送录像获取
		SipSendSubscription      chan *SubscriptionReq             // 发送订阅
		SipSendBroadcast         chan *BroadcastReq                // 语音对讲广播b
		SipSendTalk              chan *GBSSipSendTalk              // 发送语音对讲

		Broadcast *broadcast.BroadcastManager
		// sip日志 -----------------------------------------------------------------
		SipLog chan *SipLogItem // sip日志
		// sip日志 -----------------------------------------------------------------

		// catalog 任务 -----------------------------------------------------------------
		SipCatalogLoop    chan *SipCatalogLoopReq                // 上线创建定时器 下线停止定时器
		SipCatalogLoopMap *xmap.XMap[string, *SipCatalogLoopReq] // catalog请求(节流器) SipCatalogLoop -> SipCatalogLoopMap
		// catalog 任务 -----------------------------------------------------------------

		// 心跳检测 任务 -----------------------------------------------------------------
		SipHeartbeatLoop    chan *SipHeartbeatLoopReq                // 心跳检测
		SipHeartbeatLoopMap *xmap.XMap[string, *SipHeartbeatLoopReq] // 心跳检测 节流器 SipHeartbeatLoop -> SipHeartbeatLoopMap
		// 心跳检测 任务 -----------------------------------------------------------------

		// 流状态 -----------------------------------------------------------------

		// 当前做为上级,下级(设备)给当前推流 接收国标推流流是否存在[streamName](国标) 对应 GBCInviteReqMaps // 当前作为下级(设备),给上级推流 接收国标推流流是否存在[streamName](国标)
		InviteRequestState *set.CSet[string] // invite请求限制防止并发击穿信令[streamName]

		// 流是否存在
		PubStreamExistsState *set.CSet[string]
		InviteRequestLock    sync.Mutex

		// 流状态 -----------------------------------------------------------------

		// 记录sn [ uniqueId: sn ]
		SipGBSSNMap *xmap.XMap[string, uint32]
		SipGBCSNMap *xmap.XMap[string, uint32]
		// [ streamName: Request ] 访问invite时创建 bye和stop_stream删除
		AckRequestMap *xmap.XMap[string, *SendSipRequest]
		// 设备录像获取标识 如果指定设备/通道正在获取视频信息需要等待其余客户端完成
		FetchDeviceVideoState *set.CSet[string]

		// 更新设备上线下线状态队列 --------------------------------------------------------
		SetDeviceOnline            chan *DCOnlineReq
		DeviceOnlineStateUpdateMap *xmap.XMap[string, *DCOnlineReq] // 设置设备在线状态(节流器) SetDeviceOnline -> DeviceOnlineStateUpdateMap
		// 更新设备上线下线状态队列 --------------------------------------------------------

		// 预置位数据存储
		SipMessagePresetPointsMap *xmap.XMap[string, *SipMessageQueryPresetPointsResp]
		// 设备视频回放 key = 设备id + 通道id + sn
		SipMessageVideoRecordMap *xmap.XMap[string, *SipMessageVideoRecords]

		// 字典数据信息
		DictionaryMap map[string]*categories.Item[int, *dictionaries.Item]
		// 设置
		Setting *settings.Item
		// onvif 设备探测
		OnvifDiscoverDevices []*cTypes.OnvifDeviceItem
		// media server记录
		MediaServerRecords []*mediaServers.Item
		// 设备级联
		CascadeRecords           []*cascade.Item
		CascadeRegister          *xmap.XMap[uint64, *CascadeRegisterItem]
		CascadeKeepaliveCounter  *xmap.XMap[uint64, uint64]
		CascadeRegisterExecuting *xmap.XMap[string, bool]

		// 所有 设备/通道 在线状态
		DeviceOnlineState *cTypes.DeviceOnlineStateResp

		// 初始化数据加载完成状态
		InitFetchDataState sync.WaitGroup

		GBSUDPSev,
		GBSTCPSev,
		GBCUDPSev,
		GBCTCPSev *gosip.Server

		GBCRegisterChan  chan *GBCRegisterChanItem
		GBCKeepaliveChan chan *cascade.Item

		// GBCInviteThrottleSet *set.CSet[string] // gbc invite请求限流
		GBCInviteReqMaps *xmap.XMap[string, *SipGBCInviteReqItem] // [key: cascadeChannelUniqueId]
		// GBC获取设备录像标志
		GBCRecordInfoSendMaps *xmap.XMap[string, *GBCRecordInfoItem] // [channelUniqueId: data]

		// 文件下载
		DownloadManager *download.DownloadManager
	}
)

func (l *ServiceContext) CloseWSTalkSip(key string) {
	v, ok := l.TalkSipData.Get(key)
	if ok && v.RTPSession != nil {
		v.RTPSession.Stop()
	}

	l.TalkSipData.Remove(key)
}

// 接收消息
type SipReceiveHandleLogic[T any] interface {
	DO() *Response
	New(ctx context.Context, svcCtx *ServiceContext, req *Request, tx sip.ServerTransaction) T
}

// interval
type DOProcLogicParams struct {
	SvcCtx      *ServiceContext
	RecoverCall func(name string)
}

type SipProcLogic interface {
	DO(params *DOProcLogicParams)
}

type (
	HttpResponse struct {
		Data interface{}
		Err  *response.HttpErr
	}

	HttpHandleLogicBase[Logic any] interface {
		Path() string
		New(ctx context.Context, c *gin.Context, svcCtx *ServiceContext) Logic
	}

	HttpRHandleLogic[Logic, Req any] interface {
		HttpHandleLogicBase[Logic]
		DO(req Req) *HttpResponse
	}

	HttpEHandleLogic[Logic any] interface {
		HttpHandleLogicBase[Logic]
		DO() *HttpResponse
	}
)

type (
	SSERequestType interface {
	}

	SSEResponse struct {
		Data       interface{}
		Err        *response.HttpErr
		Done       bool
		DelayClose bool
	}

	SSEHandleLogic[Logic any, Req SSERequestType] interface {
		New(ctx context.Context, svcCtx *ServiceContext, messageChan chan *SSEResponse) Logic
		DO(req Req)
		GetType() string
	}

	SSEHandleSPLogic[Logic any] interface {
		New(ctx context.Context, svcCtx *ServiceContext, messageChan chan *SSEResponse) Logic
		DO()
		GetType() string
	}
)

type (
	DeviceDiagnosesItem struct {
		Line  string      `json:"line,optional,omitempty"`
		Title string      `json:"title,optional,omitempty"`
		Value interface{} `json:"value,optional,omitempty"`
		Color string      `json:"color,optional,omitempty"`
	}

	DeviceDiagnosesResp struct {
		Title   string                 `json:"title"`
		Records []*DeviceDiagnosesItem `json:"records,optional,omitempty"`
		Line    string                 `json:"line,optional,omitempty"`
		Done    bool                   `json:"done,optional,omitempty"`
	}
)

// message 请求参数 -----------------------------------------------------------------------------------------------

const (
	MessageCMDTypeCatalog        = "Catalog"
	MessageCMDTypeDeviceInfo     = "DeviceInfo"
	MessageCMDTypePresetQuery    = "PresetQuery"
	MessageCMDTypeKeepalive      = "Keepalive"
	MessageCMDTypeDeviceControl  = "DeviceControl"
	MessageCMDTypeAlarm          = "Alarm"
	MessageCMDTypeMobilePosition = "MobilePosition"
	MessageCMDTypeBroadcast      = "Broadcast"
	MessageCMDTypeBroadcastStop  = "BroadcastStop"
	MessageCMDTypeRecordInfo     = "RecordInfo"
	MessageCMDTypeMediaStatus    = "MediaStatus"
	MessageCMDTypeDeviceConfig   = "DeviceConfig"
	MessageCMDTypeConfigDownload = "ConfigDownload"
	MessageGuardCmdTypeSetGuard  = "SetGuard"

	SubscriptionCatalog        = "Catalog"
	SubscriptionAlarm          = "Alarm"
	SubscriptionMobilePosition = "MobilePosition"
	SubscriptionPTZPosition    = "PTZPosition"
)

const SipPresetMax = 255

type IMessageReceive interface {
	GetCmdType() string
}

type MessageReceiveBase struct {
	CmdType string `xml:"CmdType"`
	SN      int    `xml:"SN"`
}

type MessageInfo struct {
	// PTZType 摄像机类型：1-球机，2-半球，3-固定枪机，4-遥控枪机 （是设备）
	PTZType       int    `xml:"PTZType" json:"ptztype"`
	Resolution    string `xml:"Resolution" json:"resolution"` // 23041296/4/2<
	DownloadSpeed string `xml:"DownloadSpeed" json:"downloadSpeed"`
}

func (b MessageReceiveBase) GetCmdType() string {
	return strings.ToLower(b.CmdType)
}

type (
	SipChannel struct {
		// ChannelID 通道唯一编码ID，国标接入则为通道国标ID
		ChannelID string `xml:"DeviceID" json:"channelid"`
		// // DeviceID 设备编号
		// DeviceID string `xml:"-" json:"deviceid"`
		// Memo 备注（用来标示通道信息）
		MeMo string `json:"memo"`
		// Name 通道名称（设备/系统名称）
		Name string `xml:"Name" json:"name"`
		// Manufacturer 设备厂商
		Manufacturer string `xml:"Manufacturer" json:"manufacturer"`
		// Model 设备型号
		Model string `xml:"Model" json:"model"`
		// Owner 设备归属
		Owner string `xml:"Owner" json:"owner"`
		// CivilCode 行政区域编码，比如 四川：510000
		CivilCode string `xml:"CivilCode" json:"civilcode"`
		// Address 设备安装的ip地址
		Address string `xml:"Address" json:"address"`
		// Parental 设备是否有子设备,有表示是组织架构或者目录，没有表示是设备通道（1：有，0：没有）
		Parental int `xml:"Parental" json:"parental"`
		// 父目录Id
		ParentId string `xml:"ParentID" json:"parentID"`
		// SafetyWay 信令安装模式，缺省为0 （0：不采用，2：S/MIME签名方式，3：S/MIME加密签名同时采用方式，4：数字摘要方式）
		SafetyWay int `xml:"SafetyWay" json:"safetyway"`
		// RegisterWay 注册方式，缺省为1 （1：符合IETF RFC3261标准的认证注册模式，2：基于口令的双向认证注册模式，3：基于数字证书的双向认证注册模式）
		RegisterWay int `xml:"RegisterWay" json:"registerway"`
		// Secrecy 保密属性，缺省为0 （0：不涉密，1：涉密）
		Secrecy int `xml:"Secrecy" json:"secrecy"`
		// Status 设备在线状态  on：在线  off：离线
		Status string `xml:"Status" json:"status"`

		// 新增参数 [Dingshuai 2025/06/25]
		Info      MessageInfo `xml:"Info" json:"info"`
		Longitude float64     `xml:"Longitude" json:"longitude"`
		Latitude  float64     `xml:"Latitude" json:"latitude"`

		// Active 最后活跃时间
		Active int64  `json:"active"`
		URIStr string ` json:"uri"`

		// 视频编码格式
		VF string ` json:"vf"`
		// 视频高
		Height int `json:"height"`
		// 视频宽
		Width int `json:"width"`
		// 视频FPS
		FPS int `json:"fps"`
		//  pull 媒体服务器主动拉流，push 监控设备主动推流
		StreamType string `json:"streamtype"`
		// streamtype=pull时，拉流地址
		URL string `json:"url"`

		addr *sip.Address
	}

	SipBasicParam struct {
		Name              string `xml:"Name"`              // 设备名称
		Expiration        int    `xml:"Expiration"`        // 注册过期时间
		HeartBeatInterval int    `xml:"HeartBeatInterval"` // 心跳间隔时间
		HeartBeatCount    int    `xml:"HeartBeatCount"`    // 心跳超时次数
	}

	SipSnapShot struct {
		SnapNum   int    `xml:"SnapNum"`   // 连拍张数(必选)，最多10张，当手动抓拍时，取值为1
		Interval  int    `xml:"Interval"`  // 单张抓拍间隔时间，单位：秒(必选)，取值范围:最短1秒
		UploadURL string `xml:"UploadURL"` // 抓拍图像上传路径(必选)
		SessionID string `xml:"SessionID"` // 会话ID，由平台生成，用于关联抓拍的图像与平台请求(必选)
	}
)

const (
	SipPresetSet  = 0x81
	SipPresetCall = 0x82
	SipPresetDel  = 0x83
)

type (
	XMLNotify[T any] struct {
		XMLName xml.Name `xml:"Notify"`
		Content T        `xml:",inline"`
	}

	XMLResponse[T any] struct {
		XMLName xml.Name `xml:"Response"`
		Content T        `xml:",inline"`
	}
)

type (
	SipMessageKeepalive struct {
		MessageReceiveBase
		DeviceID string `xml:"DeviceID"`
		Status   string `xml:"Status"`
		Info     string `xml:"Info"`
	}

	SipMessageGBCKeepalive struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Notify"`
		DeviceID string   `xml:"DeviceID"`
		Status   string   `xml:"Status"`
		Info     string   `xml:"Info"`
	}

	SipMessageDeviceList struct {
		Num  string        `xml:"Num,attr"` // 作为属性
		Item []*SipChannel `xml:"Item"`
	}

	SipMessageCatLog struct {
		MessageReceiveBase
		XMLName  xml.Name      `xml:"Response"`
		CmdType  string        `xml:"CmdType"`
		SN       int           `xml:"SN"`
		DeviceID string        `xml:"DeviceID"`
		SumNum   int           `xml:"SumNum"`
		Item     []*SipChannel `xml:"DeviceList>Item"`
	}

	SipMessageGBCCatLog struct {
		MessageReceiveBase
		XMLName    xml.Name             `xml:"Response"`
		CmdType    string               `xml:"CmdType"`
		SN         int                  `xml:"SN"`
		DeviceID   string               `xml:"DeviceID"`
		SumNum     int                  `xml:"SumNum"`
		DeviceList SipMessageDeviceList `xml:"DeviceList"`
	}

	SipMessageBroadcastInfo struct {
		XMLName xml.Name `xml:"Info"`
		Reason  string   `xml:"Reason"`
	}

	SipMessageBroadcast struct {
		MessageReceiveBase
		XMLName  xml.Name                 `xml:"Response"`
		CmdType  string                   `xml:"CmdType"`
		SN       int                      `xml:"SN"`
		DeviceID string                   `xml:"DeviceID"`
		Result   string                   `xml:"Result"`
		Info     *SipMessageBroadcastInfo `xml:"Info"`
	}

	SipMessageGBCCatLogReq struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       int      `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipGBCMessagePresetQuery struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       int      `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageAlarmTypeParam struct {
		XMLName   xml.Name `xml:"AlarmTypeParam"`
		EventType uint     `xml:"EventType"` // 报警类型扩展参数。在人侵检测报警时可携带(EventType)事件类型(/EventType〉,事件类型取值:1-进入区域;2-离开区域
	}
	SipMessageAlarmInfo struct {
		XMLName xml.Name `xml:"Info"`
		// 报警类型
		// 		报警方式为2时,不携带AlarmType为默认的报警设备报警,
		// 			1 => 1-视频丢失报警
		// 			2 => 2-设备防拆报警
		// 			3 => 3-存储欖形闲备磁盘满报警
		// 			4 => 4-设备高温报警
		// 			5 => 5-设备低温报警
		// 		报警方式为5时
		// 			6 => 1-人工视频报警
		// 			7 => 2-运动目标检测报警
		// 			8 => 3-遗留物检测报警
		// 			9 => 4-物体移除检测报警
		// 			10 => 5-绊线检测报警
		// 			11 => 6-人侵检测报警
		// 			12 => 7-逆行检测报警
		// 			13 => 8-徘徊检测报警
		// 			14 => 9-流量统计报警
		// 			15 => 10-密度检测报警
		// 			16 => 11-视频异常检测报警
		// 			17 => 12-快速移动报警
		// 		报警方式为6时
		// 			18 => 1-存储设备磁盘故障报警
		// 			19 => 2-存储设备风扇故障报警。
		AlarmType      int                       `xml:"AlarmType"`
		AlarmTypeParam *SipMessageAlarmTypeParam `xml:"AlarmTypeParam"`
	}

	SipMessageAlarm struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Notify"`
		CmdType  string   `xml:"CmdType"`
		SN       int      `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`

		AlarmMethod      uint   `xml:"AlarmMethod"`      // 报警方式(必选),取值1为电话报警,2为设备报警,3为短信报警,4为GPS报警,5视频报警,6为设备故障报警,7其他报警
		AlarmPriority    uint   `xml:"AlarmPriority"`    // 报警级别 1为一级警情,2为二级警情,3为三级警情,4为四级警情
		AlarmTime        string `xml:"AlarmTime"`        // 2025-09-13T10:01:22
		AlarmDescription string `xml:"AlarmDescription"` // 报警描述
		Longitude        string `xml:"Longitude"`        // 经度
		Latitude         string `xml:"Latitude"`         // 维度

		Info *SipMessageAlarmInfo `xml:"Info"`
	}

	SipMessageMediaStatus struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Notify"`
		CmdType  string   `xml:"CmdType"`
		SN       int      `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`

		NotifyType uint `xml:"NotifyType"`
	}

	SipMessageVideoRecordItem struct {
		DeviceID   string `json:"deviceID"`
		Name       string `json:"name"`
		FilePath   string `json:"filePath"`
		FileSize   string `json:"fileSize"`
		Address    string `json:"address"`
		StartTime  string `json:"startTime"`
		EndTime    string `json:"endTime"`
		Secrecy    string `json:"secrecy"`    // 保密属性: 0-不涉密, 1-涉密
		Type       string `json:"type"`       // 录像产生类型: time, alarm, manual, all
		RecorderID string `json:"recorderID"` // 录像触发者ID
		UniqueId   string `json:"uniqueId,optional,omitempty"`
	}

	SipVideoRecordItem struct {
		*SipMessageVideoRecordItem
		ChannelID string `json:"channelID"`
	}

	SipMessageVideoRecordsResp struct {
		MessageReceiveBase
		SN         int64                        `xml:"SN"`
		XMLName    xml.Name                     `xml:"Response"`
		DeviceID   string                       `xml:"DeviceID"`
		Name       string                       `xml:"Name"`
		SumNum     int                          `xml:"SumNum"`
		RecordList []*SipMessageVideoRecordItem `xml:"RecordList>Item"`
	}

	SipGBCMessageVideoRecordsResp struct {
		XMLName    xml.Name                     `xml:"Response"`
		CmdType    string                       `xml:"CmdType"`
		SN         int64                        `xml:"SN"`
		DeviceID   string                       `xml:"DeviceID"`
		Name       string                       `xml:"Name"`
		SumNum     int                          `xml:"SumNum"`
		RecordList []*SipMessageVideoRecordItem `xml:"RecordList>Item"`
	}

	SipMessageVideoRecords struct {
		List  []*SipVideoRecordItem
		Total int
	}

	SipMessagePresetItem struct {
		PresetID   string
		PresetName string
	}

	SipGBCInviteReqItem struct {
		To     *sip.ToHeader
		From   *sip.FromHeader
		Callid *sip.CallID
		CSeq   *sip.CSeq
		StreamName,
		SessionId string
		Req *Request
	}

	GBCRecordInfoItem struct {
		Channel *channels.Item
		Device  *devices.Item
	}

	SipMessageQueryPresetPoints struct {
		MessageReceiveBase
		XMLName  xml.Name                `xml:"Response"`
		CmdType  string                  `xml:"CmdType"`
		SN       int                     `xml:"SN"`
		DeviceID string                  `xml:"DeviceID"`
		SumNum   int                     `xml:"SumNum"`
		Item     []*SipMessagePresetItem `xml:"PresetList>Item"`
	}

	SipMessageQueryPresetPointsResp struct {
		Count   uint
		Records []*SipMessagePresetItem
	}

	SipSendQueryPresetPointsReq struct {
		DeviceUniqueId  string `json:"deviceUniqueId"`
		ChannelUniqueId string `json:"channelUniqueId"`
	}

	SipSendSetPresetPointsReq struct {
		DeviceUniqueId  string `json:"deviceUniqueId"`
		ChannelUniqueId string `json:"channelUniqueId"`
		Type            string `json:"type"`
		Index           string `json:"index"`
		Title           string `json:"title"`
	}

	SipMessageDeviceInfo struct {
		MessageReceiveBase
		DeviceID     string `xml:"DeviceID"`     // 目标设备的编码(必选)
		DeviceName   string `xml:"DeviceName"`   // 目标设备的名称(可选
		Manufacturer string `xml:"Manufacturer"` // 设备生产商(可选)
		Model        string `xml:"Model"`        // 设备型号(可选)
		Firmware     string `xml:"Firmware"`     // 设备固件版本(可选)
		Channel      uint   `xml:"Channel"`      // 通道数量
		Result       string `xml:"Result"`       // 査询结果(必选)

		// DeviceType string `xml:"DeviceType"`
		// MaxCamera  int    `xml:"MaxCamera"`
		// MaxAlarm   int    `xml:"MaxAlarm"`
	}

	SipMessageGBCDeviceInfo struct {
		MessageReceiveBase
		XMLName      xml.Name `xml:"Response"`
		DeviceID     string   `xml:"DeviceID"`     // 目标设备的编码(必选)
		DeviceName   string   `xml:"DeviceName"`   // 目标设备的名称(可选
		Manufacturer string   `xml:"Manufacturer"` // 设备生产商(可选)
		Model        string   `xml:"Model"`        // 设备型号(可选)
		Firmware     string   `xml:"Firmware"`     // 设备固件版本(可选)
		Channel      uint     `xml:"Channel"`      // 通道数量
		Result       string   `xml:"Result"`       // 査询结果(必选)

		// DeviceType string `xml:"DeviceType"`
		// MaxCamera  int    `xml:"MaxCamera"`
		// MaxAlarm   int    `xml:"MaxAlarm"`
	}

	SipMessageGBSDeviceInfo struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageGBSCatalog struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageGBSPtz struct {
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		PTZCmd   string   `xml:"PTZCmd"`
	}

	SipMessageGBSPresetPoints struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageGBSRecordInfo struct {
		XMLName   xml.Name `xml:"Query"`
		CmdType   string   `xml:"CmdType"`
		SN        uint32   `xml:"SN"`
		DeviceID  string   `xml:"DeviceID"`
		StartTime string   `xml:"StartTime"`
		EndTime   string   `xml:"EndTime"`
		Type      string   `xml:"Type"` // 默认值： all  可选：time、alarm、manual、all
	}

	SipMessageGBSGuard struct {
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		GuardCmd string   `xml:"GuardCmd"`
	}

	SipMessageGBSBroadcast struct {
		XMLName  xml.Name `xml:"Notify"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		SourceID string   `xml:"SourceID"`
		TargetID string   `xml:"TargetID"`
	}

	SipMessageGBSSubscriptionCatalog struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageGBSSubscriptionAlarm struct {
		XMLName            xml.Name `xml:"Query"`
		CmdType            string   `xml:"CmdType"`
		SN                 uint32   `xml:"SN"`
		DeviceID           string   `xml:"DeviceID"`
		StartAlarmPriority int      `xml:"StartAlarmPriority"`
		EndAlarmPriority   int      `xml:"EndAlarmPriority"`
		AlarmMethod        int      `xml:"AlarmMethod"`
	}

	SipMessageGBSSubscriptionLocation struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint32   `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		Interval int      `xml:"Interval"`
	}

	SipMessageConfigDownload struct {
		MessageReceiveBase
		XMLName    xml.Name       `xml:"Response"`
		DeviceID   string         `xml:"DeviceID"`
		Result     string         `xml:"Result"`
		BasicParam *SipBasicParam `xml:"BasicParam"`
		// VideoParamOpt       *VideoParamOpt       `xml:"VideoParamOpt"`
		// SVACEncodeConfig    *SVACEncodeConfig    `xml:"SVACEncodeConfig"`
		// SVACDecodeConfig    *SVACDecodeConfig    `xml:"SVACDecodeConfig"`
		// VideoParamAttribute *VideoParamAttribute `xml:"VideoParamAttribute"`
		// VideoRecordPlan     *VideoRecordPlan     `xml:"VideoRecordPlan"`
		// VideoAlarmRecord    *VideoAlarmRecord    `xml:"VideoAlarmRecord"`
		// PictureMask         *PictureMask         `xml:"PictureMask"`
		// FrameMirror         *FrameMirror         `xml:"FrameMirror"`
		// AlarmReport         *AlarmReport         `xml:"AlarmReport"`
		// OSDConfig           *OSDConfig           `xml:"OSDConfig"`
		SnapShot *SipSnapShot `xml:"SnapShot"`
	}

	SipMessageSendReq struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}

	SipMessageQueryRecordInfo struct {
		XMLName   xml.Name `xml:"Query"`
		CmdType   string   `xml:"CmdType"`
		SN        uint32   `xml:"SN"`
		DeviceID  string   `xml:"DeviceID"`
		StartTime string   `xml:"StartTime"`
		EndTime   string   `xml:"EndTime"`
		Type      string   `xml:"Type"` // 默认值： all  可选：time、alarm、manual、all
	}

	SipGBCMessageQueryRecordInfo struct {
		MessageReceiveBase
		XMLName   xml.Name `xml:"Query"`
		CmdType   string   `xml:"CmdType"`
		SN        uint     `xml:"SN"`
		DeviceID  string   `xml:"DeviceID"`
		StartTime string   `xml:"StartTime"`
		EndTime   string   `xml:"EndTime"`
		Type      string   `xml:"Type"` // 默认值： all  可选：time、alarm、manual、all
	}

	SipMessageGuard struct {
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		GuardCmd string   `xml:"GuardCmd"`
	}

	SipMessagePtz struct {
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		PTZCmd   string   `xml:"PTZCmd"`
	}

	SipGBCMessagePtz struct {
		MessageReceiveBase
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		PTZCmd   string   `xml:"PTZCmd"`
		GuardCmd string   `xml:"GuardCmd"`
	}

	SipSubscriptionAlarm struct {
		XMLName            xml.Name `xml:"Query"`
		CmdType            string   `xml:"CmdType"`
		SN                 uint     `xml:"SN"`
		DeviceID           string   `xml:"DeviceID"`
		StartAlarmPriority int      `xml:"StartAlarmPriority"`
		EndAlarmPriority   int      `xml:"EndAlarmPriority"`
		AlarmMethod        int      `xml:"AlarmMethod"`
	}

	SipSubscriptionLocation struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		Interval int      `xml:"Interval"`
	}

	SipSubscriptionPTZ struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}
	SipSubscriptionCatalog struct {
		XMLName  xml.Name `xml:"Query"`
		CmdType  string   `xml:"CmdType"`
		SN       uint     `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
	}
)

// message 请求参数 -----------------------------------------------------------------------------------------------

// message 发送参数 -----------------------------------------------------------------------------------------------
type SendSipRequest struct {
	// 向设备发送的数据
	SendData sip.Request
	// 设备请求数据
	Req *Request
	SN  uint32

	ChannelUniqueId string
}

// message 发送参数 -----------------------------------------------------------------------------------------------

// onvif -----------------------------------------------------------------------------------------------
type (
	OnvifDeviceInfoReq struct {
		IP       string `json:"ip"`
		Port     uint   `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	OnvifWSDiscoveryResponse struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			ProbeMatches struct {
				Matches []struct {
					EndpointReference struct {
						Address string `xml:"Address"`
					} `xml:"EndpointReference"`
					Types  string `xml:"Types"`
					Scopes string `xml:"Scopes"`
					XAddrs string `xml:"XAddrs"`
				} `xml:"ProbeMatch"`
			} `xml:"ProbeMatches"`
		} `xml:"Body"`
	}

	OnvifPresetPointsEnvelopeResp struct {
		Header struct{}
		Body   struct {
			GetPresetsResponse struct {
				Preset []onvif.PTZPreset
			}
		}
	}

	OnvifPresetPointNodesResp struct {
		Header struct{}
		Body   struct {
			GetNodesResponse struct {
				PTZNode []onvif.PTZNode
			}
		}
	}
)

// onvif -----------------------------------------------------------------------------------------------

// notify -----------------------------------------------------------------------------------------------
type (
	NotifyStreamReq struct {
		ServerId      string `json:"server_id,optional"`
		Protocol      string `json:"protocol,optional"`
		Url           string `json:"url,optional"`
		AppName       string `json:"app_name,optional"`
		StreamName    string `json:"stream_name,optional"`
		UrlParam      string `json:"url_param,optional"`
		SessionId     string `json:"session_id,optional"`
		RemotetAddr   string `json:"remotet_addr,optional"`
		HasInSession  bool   `json:"has_in_session,optional"`
		HasOutSession bool   `json:"has_out_session,optional"`
	}

	NotifyRtmpConnectReq struct {
		ServerId   string `json:"server_id,optional"`
		AppName    string `json:"app_name,omitempty,optional"`
		StreamName string `json:"stream_name,omitempty,optional"`
		SessionId  string `json:"session_id,optional"`
		RemoteAddr string `json:"remote_addr,optional"`
		FlashVer   string `json:"flashVer,optional"`
		TcUrl      string `json:"tcUrl,optional"`
	}
)

// notify -----------------------------------------------------------------------------------------------

// gbc -------------------------------------------------------------------------------------------------

type (
	SipGBCCatalogReq struct {
		CascadeID uint64 `json:"cascadeId,optional"`

		DepartmentUniqueId    string `json:"departmentUniqueId,optional"`
		OldDepartmentUniqueId string `json:"oldDepartmentUniqueId,optional"`

		ChannelUniqueId    string `json:"channelUniqueId,optional"`
		OldChannelUniqueId string `json:"oldChannelUniqueId,optional"`
	}
)

// gbc -------------------------------------------------------------------------------------------------

// base -------------------------------------------------------------------------------------------------

type (
	VideoStreamReq struct {
		DeviceUniqueId  string  `json:"deviceUniqueId,optional"`
		ChannelUniqueId string  `json:"channelUniqueId,optional"`
		StartAt         int64   `json:"startAt,optional"`
		EndAt           int64   `json:"endAt,optional"`
		Download        bool    `json:"download,optional"`
		Speed           float64 `json:"speed,optional"`
		Https           bool    `json:"https,optional"`
	}

	DCReq struct {
		ChannelUniqueId string `json:"channelUniqueId,optional"`
		DeviceUniqueId  string `json:"deviceUniqueId,optional"`
		MsID            uint64 `json:"msID,optional"`
	}

	MsQueryRecordByNameReq struct {
		StreamNames     []string `json:"streamNames"`
		RecordType      uint     `json:"recordType"`
		ChannelUniqueId string   `json:"channelUniqueId,optional"`
		DeviceUniqueId  string   `json:"deviceUniqueId,optional"`
	}

	MsReloadReq struct {
		IP     string                 `json:"ip"`
		Port   int                    `json:"port"`
		Reboot bool                   `json:"reboot,optional"`
		Delay  int                    `json:"delay,optional"`
		Config map[string]interface{} `json:"config"`
	}
	MsGetConfigReq struct {
		IP   string `json:"ip"`
		Port int    `json:"port"`
	}

	VideoPlaybackControlReq struct {
		StreamName string  `json:"streamName"`
		Speed      float64 `json:"speed,optional"`
	}

	SubscriptionReq struct {
		DeviceUniqueId string               `json:"deviceUniqueId"`
		Subscription   devices.Subscription `json:"subscription"`
	}

	BroadcastReq struct {
		ChannelUniqueId string   `json:"channelUniqueId"`
		DeviceUniqueId  string   `json:"deviceUniqueId"`
		Req             *Request `json:"req"`
	}

	VideoStreamStopReq struct {
		StreamName  string   `json:"streamName,optional"`
		StreamNames []string `json:"streamNames,optional"`
		ID          uint64   `json:"id"` // media server id
	}

	DeviceControlReq struct {
		Horizontal int `json:"horizontal,optional"` // 水平移动 正数向左+1 负数向右-1
		Vertical   int `json:"vertical,optional"`   // 垂直移动 正数+1向上 负数-1向下
		Minifier   int `json:"minifier,optional"`   // 变倍 拉近拉远
		Zoom       int `json:"zoom,optional"`       // 变焦
		Speed      int `json:"speed,optional"`      // 速度
		Diaphragm  int `json:"diaphragm,optional"`  // 光圈

		Stop bool `json:"stop,optional"` // 停止

		DeviceUniqueId  string `json:"deviceUniqueId"`  // 设备id
		ChannelUniqueId string `json:"channelUniqueId"` // 通道id
	}

	PresetPointSetReq struct {
		DeviceUniqueId  string `json:"deviceUniqueId"`  // 设备id
		ChannelUniqueId string `json:"channelUniqueId"` // 通道id
		Title           string `json:"title"`           // 标题
		Index           string `json:"index"`           // 索引
		Type            string `json:"type"`            // add reset delete skip
	}

	PresetPointsReq struct {
		DeviceUniqueId  string `json:"deviceUniqueId"`  // 设备id
		ChannelUniqueId string `json:"channelUniqueId"` // 通道id
	}

	PresetPointItem struct {
		Name  string `json:"name"`
		Index string `json:"index"`
	}

	GetPresetPointResp struct {
		Records []*PresetPointItem `json:"records"`
		Count   uint               `json:"count"`
	}

	QueryVideoRecordsReq struct {
		ChannelUniqueId string `json:"channelUniqueId"`
		DeviceUniqueId  string `json:"deviceUniqueId"`
		Day             int64  `json:"day"`
		Page            uint64 `json:"page"`
		Limit           uint64 `json:"limit"`

		SN int64 `json:"SN,optional,omitempty"`
	}

	WSTokenReq struct {
		ID uint64 `json:"id"`
	}
)

// base -------------------------------------------------------------------------------------------------

// websocket -------------------------------------------------------------------------------------------------
type (
	WSMessageReceiveItem struct {
		Client *WSClient
		// 长连接接收到的消息内容
		Content *WSReceiveMessage
	}

	WSResponseMessageItem struct {
		Client    *WSClient
		Content   *WSResponseMessage
		AlterCall func()
	}

	BroadcastMessageItem struct {
		Type,
		Caller string
		Data interface{}
	}

	BroadcastMessageTalkSipState struct {
		Key          string `json:"key"`
		State        uint   `json:"state"`        // 1 占用 2 等待 3 sip交互完成
		FailedReason string `json:"failedReason"` // 错误信息
	}

	BroadcastMessageTalkUsageStatus struct {
		Key      string `json:"key"`
		State    uint   `json:"state"`    // 0 解除状态 1 正在占用
		UniqueId string `json:"uniqueId"` // 语音id
	}

	WSCloseChanItem struct {
		Error  *tps.XError
		Client *WSClient
	}

	WSProc struct {
		ReceiveMessageChan  chan *WSMessageReceiveItem
		ResponseMessageChan chan *WSResponseMessageItem
		BroadcastChan       chan *BroadcastMessageItem // 广播数据
		CloseChan           chan *WSCloseChanItem
	}

	WSClient struct {
		WebsocketConn *websocket.Conn // websocket connect

		Token,
		Userid, // 用户ID 未登录时设置一个 uniqueId
		ClientId, // 客户端唯一标识
		ConnType string // 链接类型
		ActiveTime, // 活跃时间 单位/s
		ConnTime int64 // 链接时间

		// 响应其他客户端
		ResponseTo func(message *WSResponseMessage, userid uint64) error

		// 关闭链接
		// CloseChan chan *tps.XError
		// 链接是否已被关闭
		IsClosed bool
		// 关闭链接执行 只执行一次
		CloseChanSignal sync.Once

		// token验证是否成功
		Validate bool

		// 客户端连接城后后正在进行语音对讲的key deviceUniqueId-channelUniqueId
		SipTalkActivateKey string
	}

	// 读取消息
	WSReceiveMessage struct {
		// 长连接接收到的消息内容
		Content []byte
		// 消息类型
		MessageType int
	}

	WSResponse struct {
		// resp
		Type string      `json:"type"`
		Data interface{} `json:"data"`

		Message string      `json:"msg"`
		Errors  *tps.XError `json:"errors"`
	}

	// 响应消息
	WSResponseMessage struct {
		MessageType int    `json:"messageType"`
		Lan         string `json:"lan"`
		// Receiver    int64  `json:"receiver"`
		ConnType string `json:"connType"`

		*WSResponse
	}

	// 接收到的消息
	WSRequestContent struct {
		MessageType int `json:"-"`

		Lan  string      `json:"lan"`
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	WSHandlerCallParams struct {
		Ctx          context.Context
		Client       *WSClient
		Req          *WSRequestContent
		RequestParse func(req *WSRequestContent, data interface{}) *WSResponse
	}
)

func (r *WSResponseMessage) ToRespMap() map[string]interface{} {
	var data = make(map[string]interface{})
	// if r.WSResponse != nil && r.Code != 0 {
	// 	data["code"] = r.Code
	// }
	if r.WSResponse != nil && r.Type != "" {
		data["type"] = r.Type
	}
	// if r.Receiver != 0 {
	// 	data["receiver"] = r.Receiver
	// }
	if r.ConnType != "" {
		data["connType"] = r.ConnType
	}
	if r.WSResponse != nil && r.Message != "" {
		data["msg"] = r.Message
	}
	if r.WSResponse != nil && r.Data != nil {
		data["data"] = r.Data
	}
	if r.WSResponse != nil && r.Type != "" {
		data["type"] = r.Type
	}

	if r.WSResponse != nil && r.Errors != nil {
		data["errors"] = r.Errors.Error()
	}

	return data
}

type WSClientsCache struct {
	maps sync.Map
}

func NewWSClientsCache() *WSClientsCache {
	return new(WSClientsCache)
}

func (c *WSClientsCache) Add(clientId string, client *WSClient) {
	c.maps.Store(clientId, client)
}

func (c *WSClientsCache) Row(clientId string) *WSClient {
	data, ok := c.maps.Load(clientId)
	if !ok {
		return nil
	}

	value, ok := data.(*WSClient)
	if !ok {
		return nil
	}

	return value
}

func (c *WSClientsCache) List(clientIds []string) map[string]*WSClient {
	var clients = make(map[string]*WSClient)
	for _, item := range clientIds {
		var value = c.Row(item)
		if clients == nil {
			continue
		}

		clients[item] = value
	}

	return clients
}

func (c *WSClientsCache) Range(call func(client *WSClient)) {
	c.maps.Range(func(_, value any) bool {
		if v, ok := value.(*WSClient); ok {
			call(v)
		}

		return true
	})
}

func (c *WSClientsCache) Delete(client *WSClient) {
	c.maps.Delete(client.ClientId)
}

func (c *WSClientsCache) Len() int {
	var length = 0
	c.maps.Range(
		func(_, value any) bool {
			length++
			return true
		},
	)

	return length
}

type WSGBSTalkAudioSendReq struct {
	UniqueId        string `json:"uniqueId"`
	Stream          string `json:"stream"`
	ChannelUniqueId string `json:"channelUniqueId"`
	DeviceUniqueId  string `json:"deviceUniqueId"`
}

type WSGBSTalkAudioStopReq struct {
	ChannelUniqueId string `json:"channelUniqueId"`
	DeviceUniqueId  string `json:"deviceUniqueId"`
}

type WSGBSTalkSip struct {
	UniqueId        string `json:"uniqueId"`
	ChannelUniqueId string `json:"channelUniqueId"`
	DeviceUniqueId  string `json:"deviceUniqueId"`
}

type WSGBSTalkSipPub struct {
	Unset           bool   `json:"unset,optional"`
	ChannelUniqueId string `json:"channelUniqueId"`
	DeviceUniqueId  string `json:"deviceUniqueId"`
}

type WSGBSTalkChannelRegister struct {
	Offline         bool   `json:"offline,optional"`
	ChannelUniqueId string `json:"channelUniqueId"`
	DeviceUniqueId  string `json:"deviceUniqueId"`
}

// websocket -------------------------------------------------------------------------------------------------
