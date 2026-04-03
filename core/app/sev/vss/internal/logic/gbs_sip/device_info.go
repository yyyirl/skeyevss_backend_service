package gbs_sip

import (
	"context"
	"fmt"
	"strings"

	gosip "github.com/ghettovoice/gosip/sip"

	"skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/app/sev/vss/internal/pkg/sip"
	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/devices"
	"skeyevss/core/repositories/models/dictionaries"
)

var _ types.SipReceiveHandleLogic[*DeviceInfoLogic] = (*DeviceInfoLogic)(nil)

type DeviceInfoLogic struct {
	ctx    context.Context
	svcCtx *types.ServiceContext
	req    *types.Request
	tx     gosip.ServerTransaction
}

func (l *DeviceInfoLogic) New(ctx context.Context, svcCtx *types.ServiceContext, req *types.Request, tx gosip.ServerTransaction) *DeviceInfoLogic {
	return &DeviceInfoLogic{
		svcCtx: svcCtx,
		ctx:    ctx,
		req:    req,
		tx:     tx,
	}
}

func (l *DeviceInfoLogic) DO() *types.Response {
	data, err := sip.NewParser[types.SipMessageDeviceInfo]().ToData(l.req.Original)
	if err != nil {
		return &types.Response{Error: types.NewErr(err.Error())}
	}

	var manufacturerId uint64 = 0
	dictionaryList, ok := l.svcCtx.DictionaryMap[dictionaries.UniqueIdDeviceManufacturer]
	if ok {
		for _, item := range dictionaryList.Children {
			if data.Manufacturer == strings.TrimSpace(item.Name) {
				manufacturerId = item.Raw.ID
				break
			}

			var multiValues = item.Raw.GetMultiValue()
			if len(multiValues) > 0 {
				if functions.Contains(data.Manufacturer, multiValues) {
					manufacturerId = item.Raw.ID
					break
				}
			}
		}
	}

	// 更新设备信息
	if _, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Conditions: []*orm.ConditionItem{
					{
						Column: devices.ColumnDeviceUniqueId,
						Value:  data.DeviceID,
					},
				},
				Data: []*orm.UpdateItem{
					{
						Column: devices.ColumnName,
						Value:  data.DeviceName,
					},
					{
						Column: devices.ColumnManufacturerId,
						Value:  manufacturerId,
					},
					{
						Column: devices.ColumnModelVersion,
						Value:  fmt.Sprintf("%s %s", data.Model, data.Firmware),
					},
					{
						Column: devices.ColumnChannelCount,
						Value:  data.Channel,
					},
				},
			})
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.Device.DeviceUpdate(l.ctx, data)
		},
	); err != nil {
		functions.LogError("更新设备信息失败, err: ", err.Error)
	}

	return nil
}
