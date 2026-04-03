package devices

var (
	ColumnID                     = "id"
	ColumnName                   = "name"
	ColumnLabel                  = "label"
	ColumnAccessProtocol         = "accessProtocol"
	ColumnBitstreamIndex         = "bitstreamIndex"
	ColumnDeviceUniqueId         = "deviceUniqueId"
	ColumnOriginalDeviceUniqueId = "originalDeviceUniqueId"
	ColumnState                  = "state"
	ColumnOnline                 = "online"
	ColumnExpire                 = "expire"
	ColumnAddress                = "address"
	ColumnMediaTransMode         = "mediaTransMode"
	ColumnUsername               = "username"
	ColumnPassword               = "password"
	ColumnStreamUrl              = "streamUrl"
	ColumnChannelCount           = "channelCount"
	ColumnSmsIP                  = "smsIP"
	ColumnClusterServerId        = "clusterServerId"
	ColumnManufacturerId         = "manufacturerId"
	ColumnModelVersion           = "modelVersion"
	ColumnSubscription           = "subscription"
	ColumnSourceType             = "sourceType"
	ColumnMSIds                  = "msIds"
	ColumnChannelFilters         = "channelFilters"
	ColumnDepIds                 = "depIds"
	ColumnOfflineAt              = "offlineAt"
	ColumnOnlineAt               = "onlineAt"
	ColumnRegisterAt             = "registerAt"
	ColumnKeepaliveAt            = "keepaliveAt"
	ColumnCreatedAt              = "createdAt"
	ColumnUpdatedAt              = "updatedAt"
)

var Columns = []string{
	ColumnID,
	ColumnName,
	ColumnLabel,
	ColumnAccessProtocol,
	ColumnBitstreamIndex,
	ColumnDeviceUniqueId,
	ColumnOriginalDeviceUniqueId,
	ColumnState,
	ColumnOnline,
	ColumnExpire,
	ColumnAddress,
	ColumnMediaTransMode,
	ColumnUsername,
	ColumnPassword,
	ColumnStreamUrl,
	ColumnChannelCount,
	ColumnSmsIP,
	ColumnClusterServerId,
	ColumnManufacturerId,
	ColumnModelVersion,
	ColumnSubscription,
	ColumnSourceType,
	ColumnMSIds,
	ColumnChannelFilters,
	ColumnDepIds,
	ColumnOfflineAt,
	ColumnOnlineAt,
	ColumnKeepaliveAt,
	ColumnRegisterAt,
	ColumnCreatedAt,
	ColumnUpdatedAt,
}

const (
	PrimaryId = "id"
)

// 流媒体传输模式
const (
	MediaTransMode_0 uint = iota
	MediaTransMode_1
	MediaTransMode_2
)

var (
	MediaTransModeMaps = map[uint]string{
		MediaTransMode_0: "UDP被动",
		MediaTransMode_1: "TCP被动",
		MediaTransMode_2: "TCP主动",
	}
	MediaTransModes = []uint{
		MediaTransMode_0,
		MediaTransMode_1,
		MediaTransMode_2,
	}
)

// 接入协议
const (
	_ uint = iota
	AccessProtocol_1
	AccessProtocol_2
	AccessProtocol_3
	AccessProtocol_4
	AccessProtocol_5
)

const (
	_ uint = iota
	BitstreamIndex_1
	BitstreamIndex_2
	BitstreamIndex_3
	BitstreamIndex_4
	BitstreamIndex_5
	BitstreamIndex_6
	BitstreamIndex_7
	BitstreamIndex_8
)

var (
	AccessProtocols = map[uint]string{
		AccessProtocol_1: "流媒体源",      // backend api 创建
		AccessProtocol_2: "RTMP推流",    // backend api 创建
		AccessProtocol_3: "ONVIF协议",   // backend api 创建
		AccessProtocol_4: "GB28181协议", // vss sip gbs 创建
		AccessProtocol_5: "EHOME协议",   // vss sip gbs 创建
	}

	AccessProtocolColors = map[uint]string{
		AccessProtocol_1: "rgba(122, 218, 165, .1)",
		AccessProtocol_2: "rgba(35, 155, 167, .1)",
		AccessProtocol_3: "rgba(236, 236, 187, .1)",
		AccessProtocol_4: "rgba(225, 170, 54, .1)",
		AccessProtocol_5: "rgba(255, 242, 235, .1)",
	}

	ChannelFilters = map[string]string{
		"134": "134 - 报警输入",
		"135": "135 - 报警输出",
		"136": "136 - 语音输入",
		"137": "137 - 语音输出",
		"200": "200 - 中心信令",
		"215": "215 - 业务分组",
		"216": "216 - 虚拟组织",
	}

	BitstreamIndexes = map[uint]string{
		BitstreamIndex_1: "stream:0 - 主码流",
		BitstreamIndex_2: "stream:1 - 子码流",
		BitstreamIndex_3: "streamnumber:0 - 主码流(2022)",
		BitstreamIndex_4: "streamnumber:1 - 子码流(2022)",
		BitstreamIndex_5: "streamprofile:0 - 主码流",
		BitstreamIndex_6: "streamprofile:1 - 子码流",
		BitstreamIndex_7: "streamMode:MAIN - 主码流",
		BitstreamIndex_8: "streamMode:SUB - 子码流",
	}

	VBitstreamIndexes = map[uint]VBitstreamIndexItem{
		BitstreamIndex_1: {Key: "stream", Value: "0"},
		BitstreamIndex_2: {Key: "stream", Value: "1"},
		BitstreamIndex_3: {Key: "streamnumber", Value: "0"},
		BitstreamIndex_4: {Key: "streamnumber", Value: "1"},
		BitstreamIndex_5: {Key: "streamprofile", Value: "0"},
		BitstreamIndex_6: {Key: "streamprofile", Value: "1"},
		BitstreamIndex_7: {Key: "streamMode", Value: "MAIN"},
		BitstreamIndex_8: {Key: "streamMode", Value: "SUB"},
	}
)

type VBitstreamIndexItem struct {
	Key   string
	Value string
}
