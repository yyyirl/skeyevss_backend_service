package gbs_sip

import (
	"context"
	"fmt"
	"strings"

	gosip "github.com/ghettovoice/gosip/sip"
	"google.golang.org/protobuf/types/known/structpb"

	"skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/app/sev/vss/internal/pkg/sip"
	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/channels"
	"skeyevss/core/repositories/models/devices"
)

var _ types.SipReceiveHandleLogic[*CatLogLogic] = (*CatLogLogic)(nil)

type CatLogLogic struct {
	ctx    context.Context
	svcCtx *types.ServiceContext
	req    *types.Request
	tx     gosip.ServerTransaction
}

func (l *CatLogLogic) New(ctx context.Context, svcCtx *types.ServiceContext, req *types.Request, tx gosip.ServerTransaction) *CatLogLogic {
	return &CatLogLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		req:    req,
		tx:     tx,
	}
}

func (l *CatLogLogic) DO() *types.Response {
	data, err := sip.NewParser[types.SipMessageCatLog]().ToData(l.req.Original)
	if err != nil {
		return &types.Response{Error: types.NewErr(err.Error())}
	}

	var isCascade uint = 0
	for _, item := range l.req.Original.Headers() {
		if strings.ToLower(item.Name()) == sip.HeaderUserAgentKey {
			if item.Value() == sip.MakeCascadeUserAgent(l.svcCtx.Config.Name, l.svcCtx.Config.InternalIp) {
				isCascade = 1
			}
		}
	}

	// 获取设备信息
	deviceRes, err1 := response.NewRpcToHttpResp[*deviceservice.Response, *devices.Item]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Conditions: []*orm.ConditionItem{
					{Column: devices.ColumnDeviceUniqueId, Value: data.DeviceID},
				},
			})
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.Device.DeviceRow(l.ctx, data)
		},
	)
	if err1 != nil {
		return &types.Response{Error: types.NewErr(err1.Error)}
	}

	// 删除排除的通道
	if len(deviceRes.Data.ChannelFilters) > 0 {
		_, _ = response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
			func() (*deviceservice.Response, error) {
				return l.svcCtx.RpcClients.Device.ChannelDeleteWithChannelFilters(
					l.ctx,
					&deviceservice.UniqueIdsReq{
						UniqueId:  deviceRes.Data.DeviceUniqueId,
						UniqueIds: deviceRes.Data.ChannelFilters,
					},
				)
			},
		)
	}

	var (
		records  []*structpb.Struct
		onLineAt = functions.NewTimer().NowMilli()
	)
	for _, item := range data.Item {
		var (
			online   uint = 0
			parental uint = 0
		)
		if item.Status == "ON" {
			online = 1
		}

		if item.Parental != 0 {
			parental = 1
		}

		if item.ChannelID == data.DeviceID && item.Parental == 1 {
			item.Parental = 0
		}

		if len(item.ChannelID) >= 20 {
			if functions.Contains(item.ChannelID[10:13], deviceRes.Data.ChannelFilters) {
				continue
			}
		}

		var original = "{}"
		if v, err := functions.ToString(item); err == nil {
			original = v
		}

		var model = &channels.Item{
			Original: functions.StructToMap(item, "json", nil),
			Channels: &channels.Channels{
				Name:           item.Name,
				UniqueId:       item.ChannelID,
				DeviceUniqueId: data.DeviceID,
				Online:         online,
				ParentID:       item.ParentId,
				Parental:       parental,
				OnlineAt:       uint64(onLineAt),
				IsCascade:      isCascade,
				PTZType:        uint(item.Info.PTZType),
				Original:       original,
				Longitude:      item.Longitude,
				Latitude:       item.Latitude,
			},
		}

		v, err := model.ConvToModel(nil)
		if err != nil {
			return &types.Response{Error: types.NewErr(err.Error())}
		}

		record, err := structpb.NewStruct(functions.StructToMap(v, "json", nil))
		if err != nil {
			return &types.Response{Error: types.NewErr(err.Error())}
		}

		records = append(records, record)
	}

	// 设置离线
	if _, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Data: []*orm.UpdateItem{
					{Column: channels.ColumnOnline, Value: 0},
				},
				Conditions: []*orm.ConditionItem{
					{Column: channels.ColumnDeviceUniqueId, Value: data.DeviceID},
					{
						Original: &orm.ConditionOriginalItem{
							Query:  fmt.Sprintf("? - `%s` >= ?", channels.ColumnOnlineAt),
							Values: []interface{}{onLineAt, 3 * 60000},
						},
					},
				},
			})
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.Device.ChannelUpdate(l.ctx, data)
		},
	); err != nil {
		functions.LogError("rpc channel update error: ", err.Error)
		return &types.Response{Error: types.NewErr(err.Error)}
	}

	// 设置通道
	if _, err := response.NewRpcToHttpResp[*deviceservice.Response, uint64]().Parse(
		func() (*deviceservice.Response, error) {
			return l.svcCtx.RpcClients.Device.ChannelUpsert(l.ctx, &deviceservice.SliceMapReq{Data: records})
		},
	); err != nil {
		functions.LogError("rpc channel upsert error: ", err.Error)
		return &types.Response{Error: types.NewErr(err.Error)}
	}

	// 获取通道数量
	channelTotalRes, err3 := response.NewRpcToHttpResp[*deviceservice.Response, int]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Conditions: []*orm.ConditionItem{
					{Column: devices.ColumnDeviceUniqueId, Value: data.DeviceID},
				},
			})
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.Device.ChannelTotal(l.ctx, data)
		},
	)
	if err3 != nil {
		functions.LogError("sip 获取通道数量失败, err:", err3)
	}

	// 更新通道数量
	if channelTotalRes != nil {
		if _, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
			func() (*deviceservice.Response, error) {
				data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
					Conditions: []*orm.ConditionItem{
						{Column: devices.ColumnDeviceUniqueId, Value: data.DeviceID},
					},
					Data: []*orm.UpdateItem{
						{Column: devices.ColumnChannelCount, Value: channelTotalRes.Data},
					},
				})
				if err != nil {
					return nil, err
				}

				return l.svcCtx.RpcClients.Device.DeviceUpdate(l.ctx, data)
			},
		); err != nil {
			functions.LogError("sip 更新通道数量失败, err:", err)
		}
	}

	functions.LogInfo(l.req.ID, "catalog 更新成功")
	return nil
}
