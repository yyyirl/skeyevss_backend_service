// @Title        common
// @Description  main
// @Create       yiyiyi 2025/7/31 13:58

package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/app/sev/vss/internal/logic/http/gbs"
	"skeyevss/core/app/sev/vss/internal/pkg/ms"
	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/common/stream"
	ctypes "skeyevss/core/common/types"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/contextx"
	"skeyevss/core/pkg/dt"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/channels"
	"skeyevss/core/repositories/models/devices"
)

func setStreamState(ctx context.Context, c *gin.Context, svcCtx *types.ServiceContext, req types.NotifyStreamReq, state uint, path string) *types.HttpResponse {
	if req.StreamName == "" {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str("stream 不能为空"), localization.MR1004),
		}
	}

	data, err := stream.New().Parse(req.StreamName)
	if err != nil {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1004),
		}
	}

	if data.Channel == "" || data.Device == "" {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Err(err), localization.MR1004),
		}
	}

	// 回放流不支持保活
	if path == VOnSubStartLogic.Path() && data.PlayType == stream.PlayTypePlayback {
		return nil
	}

	if res, err := response.NewRpcToHttpResp[*deviceservice.Response, bool]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Conditions: []*orm.ConditionItem{
					{Column: channels.ColumnUniqueId, Value: data.Channel},
					{Column: channels.ColumnDeviceUniqueId, Value: data.Device},
				},
			})
			if err != nil {
				return nil, err
			}

			return svcCtx.RpcClients.Device.ChannelExists(ctx, data)
		},
	); err != nil {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("通道获取错误, err: %s", err.Error)), localization.MR1008),
		}
	} else {
		if !functions.ByteToBool(res.Res.Data) {
			return &types.HttpResponse{
				Err: response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("通道不存在")), localization.MR1008),
			}
		}
	}

	deviceRes, err1 := response.NewRpcToHttpResp[*deviceservice.Response, *devices.Item]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				Conditions: []*orm.ConditionItem{
					{Column: devices.ColumnDeviceUniqueId, Value: data.Device},
				},
			})
			if err != nil {
				return nil, err
			}

			return svcCtx.RpcClients.Device.DeviceRow(ctx, data)
		},
	)
	if err1 != nil {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str(fmt.Sprintf("设备获取错误, err: %s", err1.Error)), localization.MR1008),
		}
	}

	// 实时流保活
	if path == VOnSubStartLogic.Path() {
		// 获取通道信息
		dt.TrailingDebounce(
			req.StreamName,
			2*time.Second,
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// channelRes, err := response.NewRpcToHttpResp[*deviceservice.Response, *channels.Item]().Parse(
				// 	func() (*deviceservice.Response, error) {
				// 		data, err := conv.New(svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{
				// 			Conditions: []*orm.ConditionItem{
				// 				{Column: channels.ColumnUniqueId, Value: data.Channel},
				// 				{Column: channels.ColumnDeviceUniqueId, Value: data.Device},
				// 			},
				// 		})
				// 		if err != nil {
				// 			return nil, err
				// 		}
				//
				// 		return svcCtx.RpcClients.Device.ChannelRowFind(ctx, data)
				// 	},
				// )
				// if err != nil {
				// 	functions.LogError("通道获取失败, err:", err)
				// 	return
				// }

				// if deviceRes.Data.AccessProtocol == devices.AccessProtocol_4 {
				// 	// 发送invite
				// 	if res := gbs.InviteLogic.New(ctx, c, svcCtx).Invite(&gbs.InviteParams{
				// 		DeviceUniqueId: data.Device,
				// 		ChannelID:      data.Channel,
				// 		PlayType:       data.PlayType,
				// 		DeviceItem:     deviceRes.Data,
				// 		ChannelItem:    channelRes.Data,
				// 		StreamName:     req.StreamName,
				// 		OnPubStart:     true,
				// 		Caller:         "common 请求 invite path: " + path,
				// 	}); res != nil && res.Err != nil {
				// 		functions.LogError("invite发送失败, err:", res.Err.Error)
				// 	}
				// } else if deviceRes.Data.AccessProtocol == devices.AccessProtocol_1 || deviceRes.Data.AccessProtocol == devices.AccessProtocol_3 { // PULL拉流
				// 	var (
				// 		req = map[string]interface{}{
				// 			"deviceUniqueId":  data.Device,
				// 			"channelUniqueId": data.Channel,
				// 		}
				// 		url = fmt.Sprintf("http://127.0.0.1:%d/api/video/stream", svcCtx.Config.Http.Port)
				// 	)
				// 	// req["download"] = false
				// 	// req["startAt"] = 0
				// 	// req["endAt"] = 0
				// 	var streamResp response.HttpResp[ctypes.StreamResp]
				// 	if _, err := functions.NewResty(ctx, &functions.RestyConfig{Mode: svcCtx.Config.Mode}).HttpPostJsonResJson(url, req, &streamResp); err != nil {
				// 		functions.LogError("拉流保活发送失败, err:", err)
				// 	}
				// }

				var streamResp response.HttpResp[ctypes.StreamResp]
				if _, err := functions.NewResty(ctx, &functions.RestyConfig{Mode: svcCtx.Config.Mode}).HttpPostJsonResJson(
					fmt.Sprintf("http://127.0.0.1:%d/api/video/stream", svcCtx.Config.Http.Port),
					map[string]interface{}{
						"deviceUniqueId":  data.Device,
						"channelUniqueId": data.Channel,
					},
					&streamResp,
				); err != nil {
					functions.LogError("拉流保活发送失败, err:", err)
				}
			},
		)
		return nil
	}

	var (
		record []*orm.UpdateItem
		msID   = ms.New(ctx, svcCtx).VoteNodeItem(contextx.GetCtxIP(ctx))
		now    = functions.NewTimer().NowMilli()
	)
	if state == 0 { // 下线
		if deviceRes.Data.AccessProtocol == devices.AccessProtocol_2 { // RTMP推流
			record = []*orm.UpdateItem{
				{Column: channels.ColumnStreamState, Value: 0},
				{Column: channels.ColumnOnline, Value: 0},
			}
		} else {
			record = []*orm.UpdateItem{
				{Column: channels.ColumnStreamState, Value: 0},
			}
		}

		// 发送停止流请求
		go func() {
			if deviceRes.Data.AccessProtocol == devices.AccessProtocol_4 {
				// 发送停止BYE请求 停止国标推流
				if resp := gbs.StopStreamLogic.New(ctx, c, svcCtx).StopStream(req.StreamName, "0"); resp != nil && resp.Err != nil {
					functions.LogError("停止国标流失败, err:", resp.Err)
				}
			}

		}()
	} else {
		// 获取请求ip
		record = []*orm.UpdateItem{
			{Column: channels.ColumnStreamState, Value: 1},
			{Column: channels.ColumnOnline, Value: 1},
			{Column: channels.ColumnOnlineAt, Value: now},
			{Column: channels.ColumnStreamMSId, Value: msID},
		}
	}

	if len(record) <= 0 {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str("records 不能为空"), localization.MR1008),
		}
	}

	// 更新通道状态
	if _, err := response.NewRpcToHttpResp[*deviceservice.Response, string]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(svcCtx.Config.Mode).ToPBParams(
				&orm.ReqParams{
					Conditions: []*orm.ConditionItem{
						{Column: channels.ColumnUniqueId, Value: data.Channel},
						{Column: channels.ColumnDeviceUniqueId, Value: data.Device},
					},
					Data: record,
				},
			)
			if err != nil {
				return nil, err
			}

			return svcCtx.RpcClients.Device.ChannelUpdate(ctx, data)
		},
	); err != nil {
		return &types.HttpResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str(err.Error), localization.MR1004),
		}
	}

	return nil
}
