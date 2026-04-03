package items

import (
	"context"
	"fmt"
	"strings"

	"github.com/use-go/onvif/device"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/structpb"

	"skeyevss/core/app/sev/backend/internal/svc"
	"skeyevss/core/app/sev/backend/internal/types"
	"skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/common/opt"
	cTypes "skeyevss/core/common/types"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/channels"
	"skeyevss/core/repositories/models/devices"
	"skeyevss/core/repositories/models/dictionaries"
	systemOperationLogs "skeyevss/core/repositories/models/system-operation-logs"
)

type CreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLogic {
	return &CreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateLogic) onvifChannels(record devices.Item, onvifReqParams map[string]interface{}, channelRecords []*structpb.Struct) ([]*structpb.Struct, *response.HttpErr) {
	// 设备通道
	var (
		deviceChannelListRes response.HttpResp[[]*cTypes.OnvifDeviceProfileItem]
		rq                   = l.svcCtx.RemoteReq(l.ctx)
	)
	if _, err := functions.NewResty(l.ctx, &functions.RestyConfig{Mode: l.svcCtx.Config.Mode, Referer: rq.Referer}).HttpPostJsonResJson(
		fmt.Sprintf("%s/api/onvif/device-profiles", rq.VssHttpUrlInternal),
		onvifReqParams,
		&deviceChannelListRes,
	); err != nil {
		return nil, response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("onvif设备通道获取失败, err: %s", err)), localization.M0010)
	}

	if deviceChannelListRes.Error != "" {
		return nil, response.MakeError(response.NewHttpRespMessage().Str(deviceChannelListRes.Error), localization.M0010)
	}

	if deviceChannelListRes.Data == nil {
		return nil, response.MakeError(response.NewHttpRespMessage().Str("onvif设备通道获取失败[1]"), localization.M0010)
	}

	// 替换uniqueId
	var now = functions.NewTimer().NowMilli()
	record.DeviceUniqueId = functions.GenerateUniqueID(8)
	for _, item := range deviceChannelListRes.Data {
		v, err := structpb.NewStruct(
			map[string]interface{}{
				channels.ColumnName:                    item.Profile,
				channels.ColumnLabel:                   item.Profile,
				channels.ColumnUniqueId:                functions.GenerateUniqueID(8),
				channels.ColumnDeviceUniqueId:          record.DeviceUniqueId,
				channels.ColumnOriginalChannelUniqueId: item.ProfileToken,
				channels.ColumnStreamUrl:               item.Url,
				channels.ColumnOnline:                  1,
				channels.ColumnOnlineAt:                now,
				// ptzType
			},
		)
		if err != nil {
			return nil, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
		}

		channelRecords = append(channelRecords, v)
	}

	return channelRecords, nil
}

func (l *CreateLogic) Create(req *types.RecordReq) (interface{}, *response.HttpErr) {
	// 日志记录
	opt.NewSystemOperationLogs(l.svcCtx.RpcClients).Make(l.ctx, systemOperationLogs.Types[systemOperationLogs.TypeDepartmentCreate], req)

	if len(req.Record) <= 0 {
		return 0, response.MakeError(response.NewHttpRespMessage().Str("record 不能为空"), localization.MR1004)
	}

	var record devices.Item
	if err := functions.ConvInterface(req.Record, &record); err != nil {
		return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
	}

	if record.Devices == nil {
		return 0, response.MakeError(response.NewHttpRespMessage().Str("记录解析错误"), localization.MR1002)
	}

	if record.DeviceUniqueId == "" {
		record.DeviceUniqueId = functions.GenerateUniqueID(8)
	}

	// 通道列表
	var (
		channelRecords []*structpb.Struct
		now            = functions.NewTimer().NowMilli()
	)
	// 规则校验
	switch record.AccessProtocol {
	case devices.AccessProtocol_1: // 流媒体源
		if record.StreamUrl == "" {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("输入接入码流地址不能为空"), localization.MR1004)
		}

		if !functions.Contains(record.MediaTransMode, devices.MediaTransModes) {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("流媒体传输模式值错误"), localization.MR1004)
		}

		// if record.Username == "" {
		// 	return 0, response.MakeError(response.NewHttpRespMessage().Str("用户名不能为空"), localization.MR1004)
		// }
		//
		// if record.Password == "" {
		// 	return 0, response.MakeError(response.NewHttpRespMessage().Str("密码不能为空"), localization.MR1004)
		// }

		// streamUrl
		urlRes, err := functions.ExtractBaseURL(record.StreamUrl)
		if err != nil {
			return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
		}

		record.Address = fmt.Sprintf("%s://%s", urlRes.Scheme, urlRes.Host)
		record.Online = 1
		record.OnlineAt = uint64(now)
		var name = fmt.Sprintf("%s-通道-1", record.Name)
		v, err := structpb.NewStruct(
			map[string]interface{}{
				channels.ColumnName:           name,
				channels.ColumnLabel:          name,
				channels.ColumnOnline:         1,
				channels.ColumnOnlineAt:       now,
				channels.ColumnUniqueId:       functions.GenerateUniqueID(8),
				channels.ColumnDeviceUniqueId: record.DeviceUniqueId,
				channels.ColumnStreamUrl:      record.StreamUrl,
			},
		)
		if err != nil {
			return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
		}
		channelRecords = append(channelRecords, v)

	case devices.AccessProtocol_2: // RTMP推流
		if record.StreamUrl == "" {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("输入接入码流地址不能为空"), localization.MR1004)
		}

		if len(record.MSIds) <= 0 || record.MSIds[0] == 0 {
			// 默认服务
			record.StreamUrl = fmt.Sprintf("rtmp://%s/rlive/%s", l.svcCtx.MSVoteNode(nil).Address, record.DeviceUniqueId)
		}

		record.Online = 1
		record.OnlineAt = uint64(now)
		var name = fmt.Sprintf("%s-通道-1", record.Name)
		v, err := structpb.NewStruct(
			map[string]interface{}{
				channels.ColumnName:           name,
				channels.ColumnLabel:          name,
				channels.ColumnUniqueId:       functions.GenerateUniqueID(8),
				channels.ColumnDeviceUniqueId: record.DeviceUniqueId,
				channels.ColumnOnline:         1,
				channels.ColumnOnlineAt:       now,
			},
		)
		if err != nil {
			return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
		}
		channelRecords = append(channelRecords, v)

	case devices.AccessProtocol_3: // ONVIF协议
		if record.Username == "" {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("用户名不能为空"), localization.MR1004)
		}

		if record.Password == "" {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("密码不能为空"), localization.MR1004)
		}

		record.Online = 1
		record.OnlineAt = uint64(now)
		// 获取onvif设备
		var (
			deviceListRes response.HttpResp[[]*cTypes.OnvifDeviceItem]
			deviceItem    *cTypes.OnvifDeviceItem
			rq            = l.svcCtx.RemoteReq(l.ctx)
		)
		if _, err := functions.NewResty(l.ctx, &functions.RestyConfig{Mode: l.svcCtx.Config.Mode, Referer: rq.Referer}).HttpGetResJson(
			fmt.Sprintf(
				"%s/api/onvif/discover",
				rq.VssHttpUrlInternal,
			),
			nil,
			&deviceListRes,
		); err != nil {
			return 0, response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("onvif discover设备获取失败, err: %s", err)), localization.M0010)
		}

		if deviceListRes.Error != "" {
			return 0, response.MakeError(response.NewHttpRespMessage().Str(deviceListRes.Error), localization.M0010)
		}

		if deviceListRes.Data == nil {
			return 0, response.MakeError(response.NewHttpRespMessage().Str("onvif discover设备获取失败[1]"), localization.M0010)
		}

	Loop:
		for _, item := range deviceListRes.Data {
			if item.UUID == record.DeviceUniqueId {
				deviceItem = item
				break Loop
			}

			if record.OnvifManualOperationState && len(item.ServiceURLs) > 0 {
				if item.ServiceURLs[0] == record.Address {
					deviceItem = item
					break Loop
				}
			}
		}

		if deviceItem != nil {
			record.OriginalDeviceUniqueId = deviceItem.OriginalUid
			if deviceItem.Address == "" {
				return 0, response.MakeError(response.NewHttpRespMessage().Str("设备信息获取失败 address为空"), localization.MR1004)
			}

			if record.Address == "" {
				if len(deviceItem.ServiceURLs) <= 0 {
					return 0, response.MakeError(response.NewHttpRespMessage().Str("设备接入地址不能为空"), localization.MR1004)
				}
				record.Address = deviceItem.ServiceURLs[0]
			}

			if record.Name == "" {
				record.Name = deviceItem.Name
			}

			addrRes, err := functions.ExtractBaseURL(record.Address)
			if err != nil {
				return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.M0026)
			}

			var onvifReqParams = map[string]interface{}{
				"ip":       addrRes.IP,
				"port":     addrRes.Port,
				"username": record.Username,
				"password": record.Password,
			}
			// 获取设备信息
			var (
				deviceInfoRes response.HttpResp[*device.GetDeviceInformationResponse]
				rq            = l.svcCtx.RemoteReq(l.ctx)
			)
			if _, err := functions.NewResty(l.ctx, &functions.RestyConfig{Mode: l.svcCtx.Config.Mode, Referer: rq.Referer}).HttpPostJsonResJson(
				fmt.Sprintf("%s/api/onvif/device-info", rq.VssHttpUrlInternal),
				onvifReqParams,
				&deviceInfoRes,
			); err != nil {
				return 0, response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("onvif设备获取失败[1], err: %s", err)), localization.M0010)
			}

			if deviceInfoRes.Error != "" {
				return 0, response.MakeError(response.NewHttpRespMessage().Str(deviceInfoRes.Error), localization.M0010)
			}

			if deviceInfoRes.Data == nil {
				return 0, response.MakeError(response.NewHttpRespMessage().Str("onvif设备获取失败[2]"), localization.M0010)
			}

			// 平台厂商
			if deviceInfoRes.Data.Manufacturer == "" {
			Loop1:
				for _, item := range l.svcCtx.Dictionaries() {
					if item.UniqueId == dictionaries.UniqueIdDeviceManufacturer_20 {
						record.ManufacturerId = item.ID
						break Loop1
					}
				}
			} else {
			Loop2:
				for _, item := range l.svcCtx.Dictionaries() {
					var multiValue = append(strings.Split(item.MultiValue, "\n"), item.Name)
					for _, v := range multiValue {
						if strings.ToLower(deviceInfoRes.Data.Manufacturer) == strings.ToLower(v) {
							record.ManufacturerId = item.ID
							break Loop2
						}
					}
				}
			}

			// 设备/平台型号
			if record.ModelVersion == "" {
				record.ModelVersion = deviceInfoRes.Data.Model
			}

			var err3 *response.HttpErr
			channelRecords, err3 = l.onvifChannels(record, onvifReqParams, channelRecords)
			if err3 != nil {
				return 0, err3
			}
		} else {
			if record.OnvifManualOperationState {
				if record.Address == "" {
					return 0, response.MakeError(response.NewHttpRespMessage().Str("address 不能为空"), localization.M0001)
				}

				addrRes, err := functions.ExtractBaseURL(record.Address)
				if err != nil {
					return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.M0026)
				}

				if record.Name == "" {
					record.Name = fmt.Sprintf("onvif设备")
				}

				var err3 *response.HttpErr
				channelRecords, err3 = l.onvifChannels(
					record,
					map[string]interface{}{
						"ip":       addrRes.IP,
						"port":     addrRes.Port,
						"username": record.Username,
						"password": record.Password,
					},
					channelRecords,
				)
				if err3 != nil {
					return 0, err3
				}
			} else {
				return 0, response.MakeError(response.NewHttpRespMessage().Str("设备不存在"), localization.M0010)
			}
		}

	default:
		return 0, response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("不允许创建的类型 AccessProtocol: %d", record.AccessProtocol)), localization.MR1004)
	}

	if record.Name == "" {
		return 0, response.MakeError(response.NewHttpRespMessage().Str("设备名称不能为空"), localization.MR1004)
	}

	if record.Label == "" {
		record.Label = record.Name
	}

	record.MSIds = functions.ArrUnique(record.MSIds)
	if record.ManufacturerId <= 0 {
		for _, item := range l.svcCtx.Dictionaries() {
			if item.UniqueId == dictionaries.UniqueIdDeviceManufacturer_20 {
				record.ManufacturerId = item.ID
				break
			}
		}
	}

	var data map[string]interface{}
	if err := functions.ConvInterface(record, &data); err != nil {
		return 0, response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1002)
	}

	res, err := response.NewRpcToHttpResp[*deviceservice.Response, uint64]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := structpb.NewStruct(data)
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.Device.DeviceCreate(l.ctx, &deviceservice.MapReq{Data: data})
		},
	)
	if err != nil {
		return 0, err
	}

	id, err1 := functions.ConvBytes[uint64](res.Res.Data)
	if err1 != nil {
		return 0, response.MakeError(response.NewHttpRespMessage().Err(err1), localization.MR1002)
	}

	if len(channelRecords) > 0 {
		// 创建通道
		_, err := response.NewRpcToHttpResp[*deviceservice.Response, uint64]().Parse(
			func() (*deviceservice.Response, error) {
				return l.svcCtx.RpcClients.Device.ChannelUpsert(l.ctx, &deviceservice.SliceMapReq{Data: channelRecords})
			},
		)
		if err != nil {
			// 删除 设备
			if _, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
				func() (*deviceservice.Response, error) {
					data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
						Conditions: []*orm.ConditionItem{
							{Column: devices.ColumnID, Value: id},
						},
					})
					if err != nil {
						return nil, err
					}

					return l.svcCtx.RpcClients.Device.DeviceDelete(l.ctx, data)
				},
			); err != nil {
				functions.LogcError(l.ctx, "创建设备 自动生成通道失败后删除设备失败, err:", err)
			}

			return 0, err
		}

		// 更新通道数量
		if _, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
			func() (*deviceservice.Response, error) {
				data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
					Conditions: []*orm.ConditionItem{
						{Column: devices.ColumnID, Value: id},
					},
					Data: []*orm.UpdateItem{
						{Column: devices.ColumnChannelCount, Value: len(channelRecords)},
					},
				})
				if err != nil {
					return nil, err
				}

				return l.svcCtx.RpcClients.Device.DeviceUpdate(l.ctx, data)
			},
		); err != nil {
			functions.LogcError(l.ctx, "创建设备 更新通道数量失败, err:", err)
		}
	}

	return id, nil
}
