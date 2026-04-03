package devices

import (
	"fmt"

	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
)

var _ orm.Model = (*Devices)(nil)

// 订阅
type Subscription struct {
	Catalog       bool `json:"catalog"`       // 目录
	EmergencyCall bool `json:"emergencyCall"` // 报警
	Location      bool `json:"location"`      // 位置
	PTZ           bool `json:"ptz"`           // PTZ
}

type Devices struct {
	ID                     uint64 `gorm:"column:id;primary_key;AUTO_INCREMENT;COMMENT:'主键'" json:"id"`
	Name                   string `gorm:"column:name;type:varchar(50);NOT NULL;comment:'设备名称'" json:"name"`
	Label                  string `gorm:"column:label;type:varchar(255);NOT NULL;DEFAULT:'';comment:'自定义标签'" json:"label"`
	AccessProtocol         uint   `gorm:"column:accessProtocol;type:tinyint(4);default:0;NOT NULL;comment:'接入协议 1 流媒体源 2 RTMP推流 3 ONVIF协议 4 GB28181协议 5 EHOME协议'" json:"accessProtocol"`
	DeviceUniqueId         string `gorm:"column:deviceUniqueId;uniqueIndex:devices_uniqueIndex;type:CHAR(70);NOT NULL;comment:'设备id'" json:"deviceUniqueId"`
	OriginalDeviceUniqueId string `gorm:"column:originalDeviceUniqueId;type:varchar(255);NOT NULL;DEFAULT:'';comment:'原始id'" json:"originalDeviceUniqueId"`
	State                  uint   `gorm:"column:state;type:tinyint(4);default:1;NOT NULL;comment:'启用状态 0 未启用 1 启用'" json:"state"`
	Online                 uint   `gorm:"column:online;type:tinyint(4);default:1;NOT NULL;comment:'在线状态 0 不在线 1 在线'" json:"online"`
	Expire                 uint64 `gorm:"column:expire;type:bigint;default:0;NOT NULL;comment:'注册有效期(到期时间) 单位/s'" json:"expire"`
	SourceType             uint   `gorm:"column:sourceType;type:tinyint(4);default:0;NOT NULL;comment:'来源 0 主动注册 1 后台添加'" json:"sourceType"`

	Address         string `gorm:"column:address;type:varchar(50);NOT NULL;comment:'设备接入地址，如：UDP://ip:port'" json:"address"`
	MediaTransMode  uint   `gorm:"column:mediaTransMode;type:tinyint(4);default:1;NOT NULL;comment:'流媒体传输模式 0-UDP 1-TCP被动 2-TCP主动；模式2只有GB接入时有效'" json:"mediaTransMode"`
	Username        string `gorm:"column:username;type:varchar(50);default:'';NOT NULL;comment:'设备登录用户名'" json:"username"`
	Password        string `gorm:"column:password;type:varchar(50);default:'';NOT NULL;comment:'设备登录密码'" json:"password"`
	StreamUrl       string `gorm:"column:streamUrl;type:varchar(255);default:'';NOT NULL;COMMENT:'输入接入码流地址，流媒体源类型接入有效'" json:"streamUrl"`
	ChannelCount    uint   `gorm:"column:channelCount;type:int(11);default:0;NOT NULL;comment:'通道数量'" json:"channelCount"`
	SmsIP           string `gorm:"column:smsIP;type:varchar(50);default:'';NOT NULL;comment:'设备推流给指定的流媒体服务器IP，为空则采用全局配置本地流媒体'" json:"smsIP"`
	ClusterServerId string `gorm:"column:clusterServerId;type:varchar(50);default:'';NOT NULL;comment:'集群服务器ID，预留'" json:"clusterServerId"`
	ManufacturerId  uint64 `gorm:"column:manufacturerId;index:devices_manufacturerId;type:int(11);default:0;NOT NULL;comment:'设备/平台厂商 字典关联id'" json:"manufacturerId"`
	ModelVersion    string `gorm:"column:modelVersion;type:varchar(150);default:'';NOT NULL;comment:'设备/平台型号以及版本号'" json:"modelVersion"`

	Subscription string `gorm:"column:subscription;type:char(4);NOT NULL;default:'0000';comment:'订阅项目 目录 报警 位置 PTZ'" json:"subscription"`

	MSIds          string `gorm:"column:msIds;type:json;default:(json_array());comment:'media server id list'" json:"msIds"`
	ChannelFilters string `gorm:"column:channelFilters;type:json;default:(json_array());comment:'通道id过滤'" json:"channelFilters"`
	DepIds         string `gorm:"column:depIds;type:json;default:(json_array());comment:'部门id集合'" json:"depIds"`
	BitstreamIndex uint   `gorm:"column:bitstreamIndex;type:tinyint(4);default:0;NOT NULL;comment:'码流索引'" json:"bitstreamIndex"`

	OfflineAt   uint64 `gorm:"column:offlineAt;type:bigint;default:0;NOT NULL;comment:'下线时间'" json:"offlineAt"`
	OnlineAt    uint64 `gorm:"column:onlineAt;type:bigint;default:0;NOT NULL;comment:'上线时间'" json:"onlineAt"`
	KeepaliveAt uint64 `gorm:"column:keepaliveAt;type:bigint;default:0;NOT NULL;comment:'心跳时间'" json:"keepaliveAt"`
	RegisterAt  uint64 `gorm:"column:registerAt;type:bigint;default:0;NOT NULL;comment:'最后一次注册时间'" json:"registerAt"`

	CreatedAt uint64 `gorm:"column:createdAt;type:bigint;default:0;NOT NULL;comment:'创建时间'" json:"createdAt"`
	UpdatedAt uint64 `gorm:"column:updatedAt;type:bigint;default:0;NOT NULL;comment:'更新时间'" json:"updatedAt"`

	// 手动输入onvif信息
	OnvifManualOperationState bool `gorm:"-" json:"onvifManualOperationState"`

	*orm.DefaultModel
}

func (d Devices) ToMap() map[string]interface{} {
	return functions.StructToMap(d, "json", nil)
}

func (d Devices) Columns() []string {
	return Columns
}

func (d Devices) UniqueKeys() []string {
	return []string{
		PrimaryId,
	}
}

func (d Devices) PrimaryKey() string {
	return PrimaryId
}

func (d Devices) TableName() string {
	return "sk-devices"
}

func (d Devices) QueryConditions(conditions []*orm.ConditionItem) []*orm.ConditionItem {
	return conditions
}

func (d Devices) SetConditions(conditions []*orm.ConditionItem) []*orm.ConditionItem {
	return conditions
}

func (d Devices) OnConflictColumns(_ []string) []string {
	return nil
}

// Correction 数据修正
func (d Devices) Correction(action orm.ActionType) interface{} {
	if d.MSIds == "" {
		d.MSIds = "[]"
	}

	if d.ChannelFilters == "" {
		d.ChannelFilters = "[]"
	}

	if d.DepIds == "" {
		d.DepIds = "[]"
	}

	if action == orm.ActionInsert {
		d.CreatedAt = uint64(functions.NewTimer().NowMilli())
	}
	d.UpdatedAt = uint64(functions.NewTimer().NowMilli())

	return d
}

// CorrectionMap map数据修正
func (d Devices) CorrectionMap(data map[string]interface{}) map[string]interface{} {
	data[ColumnUpdatedAt] = uint64(functions.NewTimer().NowMilli())

	if v, ok := data[ColumnMSIds]; ok {
		if val, ok := v.([]interface{}); ok {
			var ids []uint64
			for _, item := range val {
				id, err := functions.InterfaceToNumber[uint64](item)
				if err != nil {
					continue
				}
				ids = append(ids, id)
			}

			b, err := functions.JSONMarshal(ids)
			if err == nil {
				data[ColumnMSIds] = string(b)
			}
		}
	}

	if v, ok := data[ColumnChannelFilters]; ok {
		if val, ok := v.([]interface{}); ok {
			var ids []string
			for _, item := range val {
				id, ok := item.(string)
				if !ok {
					continue
				}
				ids = append(ids, id)
			}

			b, err := functions.JSONMarshal(ids)
			if err == nil {
				data[ColumnChannelFilters] = string(b)
			}
		}
	}

	if v, ok := data[ColumnDepIds]; ok {
		if val, ok := v.([]interface{}); ok {
			var ids []uint64
			for _, item := range val {
				id, err := functions.InterfaceToNumber[uint64](item)
				if err != nil {
					continue
				}
				ids = append(ids, id)
			}

			b, err := functions.JSONMarshal(ids)
			if err == nil {
				data[ColumnDepIds] = string(b)
			}
		}
	}

	return data
}

// UseCache 数据库缓存
func (d Devices) UseCache() *orm.UseCacheAdvanced {
	return &orm.UseCacheAdvanced{
		Create:       true,
		Query:        true,
		Delete:       true,
		Update:       true,
		UpdateDelete: true,
		Row:          true,
		Raw:          true,

		CacheKeyPrefix: d.TableName(),
		// Driver:         new(orm.CacheRedisDriver),
		Driver: new(orm.CacheMemoryDriver),
		Expire: 60,
	}
}

// ConvToItem 数据转换
func (d Devices) ConvToItem() (*Item, error) {
	var msIds []uint64
	if d.MSIds == "" {
		msIds = []uint64{}
	} else {
		if err := functions.ConvStringToType(d.MSIds, &msIds); err != nil {
			return nil, err
		}
	}

	var depIds []uint64
	if d.DepIds == "" {
		depIds = []uint64{}
	} else {
		if err := functions.ConvStringToType(d.DepIds, &depIds); err != nil {
			return nil, err
		}
	}

	var channelFilters []string
	if d.ChannelFilters == "" {
		channelFilters = []string{}
	} else {
		if err := functions.ConvStringToType(d.ChannelFilters, &channelFilters); err != nil {
			return nil, err
		}
	}

	var useDBCache = false
	if d.DefaultModel != nil {
		useDBCache = d.DefaultModel.UseDBCache
	}

	return &Item{
		Devices:        &d,
		MSIds:          msIds,
		DepIds:         depIds,
		ChannelFilters: channelFilters,
		Sub:            d.ConvSubscription(d.Subscription),
		UseDBCache:     useDBCache,
	}, nil
}

func (d Devices) ConvSubscription(data string) Subscription {
	var sub Subscription
	if data != "" && len(data) >= 4 {
		sub.Catalog = data[0] == '1'
		sub.EmergencyCall = data[1] == '1'
		sub.Location = data[2] == '1'
		sub.PTZ = data[3] == '1'
	}

	return sub
}

func (d Devices) Conv(data interface{}) error {
	b, err := functions.JSONMarshal(d)
	if err != nil {
		return err
	}

	return functions.JSONUnmarshal(b, data)
}

type XListItem struct {
	ID             uint64 `gorm:"column:id;primary_key;AUTO_INCREMENT;COMMENT:'主键'" json:"id"`
	Name           string `gorm:"column:name;type:varchar(50);NOT NULL;comment:'设备名称'" json:"name"`
	Label          string `gorm:"column:label;type:varchar(255);NOT NULL;DEFAULT:'';comment:'自定义标签'" json:"label"`
	DeviceUniqueId string `gorm:"column:deviceUniqueId;uniqueIndex:devices_uniqueIndex;type:CHAR(70);NOT NULL;comment:'设备id'" json:"deviceUniqueId"`
	AccessProtocol uint   `gorm:"column:accessProtocol;type:tinyint(4);default:0;NOT NULL;comment:'接入协议 1 流媒体源 2 RTMP推流 3 ONVIF协议 4 GB28181协议 5 EHOME协议'" json:"accessProtocol"`
}

func NewXList() *XListItem {
	return new(XListItem)
}

func (cs *XListItem) columns() string {
	return fmt.Sprintf(
		"`%s`, `%s`, `%s`, `%s`, `%s`",
		ColumnID,
		ColumnName,
		ColumnLabel,
		ColumnDeviceUniqueId,
		ColumnAccessProtocol,
	)
}

type OnlineStateListItem struct {
	ID             uint64 `gorm:"column:id;primary_key;AUTO_INCREMENT;COMMENT:'主键'" json:"id"`
	DeviceUniqueId string `gorm:"column:deviceUniqueId;uniqueIndex:devices_uniqueIndex;type:CHAR(70);NOT NULL;comment:'设备id'" json:"deviceUniqueId"`
	Online         uint   `gorm:"column:online;type:tinyint(4);default:1;NOT NULL;comment:'在线状态 0 不在线 1 在线'" json:"online"`
}

func NewOnlineStateList() *OnlineStateListItem {
	return new(OnlineStateListItem)
}

func (cs *OnlineStateListItem) columns() string {
	return fmt.Sprintf(
		"`%s`, `%s`, `%s`",
		ColumnID,
		ColumnDeviceUniqueId,
		ColumnOnline,
	)
}

type SimpleItem struct {
	ID             uint64 `gorm:"column:id;primary_key;AUTO_INCREMENT;COMMENT:'主键'" json:"id"`
	DeviceUniqueId string `gorm:"column:deviceUniqueId;uniqueIndex:devices_uniqueIndex;type:CHAR(70);NOT NULL;comment:'设备id'" json:"deviceUniqueId"`
	AccessProtocol uint   `gorm:"column:accessProtocol;type:tinyint(4);default:0;NOT NULL;comment:'接入协议 1 流媒体源 2 RTMP推流 3 ONVIF协议 4 GB28181协议 5 EHOME协议'" json:"accessProtocol"`
	StreamUrl      string `gorm:"column:streamUrl;type:varchar(255);default:'';NOT NULL;COMMENT:'输入接入码流地址，流媒体源类型接入有效'" json:"streamUrl"`
}

func NewSList() *SimpleItem {
	return new(SimpleItem)
}

func (cs *SimpleItem) columns() string {
	return fmt.Sprintf(
		"`%s`, `%s`, `%s`, `%s`",
		ColumnID,
		ColumnDeviceUniqueId,
		ColumnAccessProtocol,
		ColumnStreamUrl,
	)
}

type MSimpleItem struct {
	DeviceUniqueId string `gorm:"column:deviceUniqueId;uniqueIndex:devices_uniqueIndex;type:CHAR(70);NOT NULL;comment:'设备id'" json:"deviceUniqueId"`
	MSIds          string `gorm:"column:msIds;type:json;default:(json_array());comment:'media server id list'" json:"msIds"`
}

func NewMSList() *MSimpleItem {
	return new(MSimpleItem)
}

func (cs *MSimpleItem) columns() string {
	return fmt.Sprintf(
		"`%s`, `%s`",
		ColumnDeviceUniqueId,
		ColumnMSIds,
	)
}

type AccessProtocolGroup struct {
	Cnt            uint `gorm:"column:cnt" json:"cnt"`
	AccessProtocol uint `gorm:"column:accessProtocol;type:tinyint(4);default:0;NOT NULL;comment:'接入协议 1 流媒体源 2 RTMP推流 3 ONVIF协议 4 GB28181协议 5 EHOME协议'" json:"accessProtocol"`
}

func NewAccessProtocol() *AccessProtocolGroup {
	return new(AccessProtocolGroup)
}

func (cs *AccessProtocolGroup) columns() string {
	return fmt.Sprintf("IFNULL(COUNT(*), 0) cnt, %s", ColumnAccessProtocol)
}
