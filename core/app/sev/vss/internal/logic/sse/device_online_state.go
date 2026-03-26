// @Title        设备诊断
// @Description  main
// @Create       yiyiyi 2025/7/23 08:55

package sse

import (
	"context"
	"time"

	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/response"
)

type SSEDeviceOnlineStatesReq struct {
	Type       string `json:"type" form:"type" path:"type" validate:"required"`
	DeviceType int64  `json:"deviceType" form:"deviceType" path:"deviceType" validate:"required"` // 1 设备 2 通道
}

var (
	_ types.SSEHandleLogic[*DeviceOnlineStateLogic, *SSEDeviceOnlineStatesReq] = (*DeviceOnlineStateLogic)(nil)

	DeviceOnlineStatesType = "device_online_state"

	VDeviceOnlineStates = new(DeviceOnlineStateLogic)
)

type DeviceOnlineStateLogic struct {
	ctx         context.Context
	svcCtx      *types.ServiceContext
	messageChan chan *types.SSEResponse
}

func (l *DeviceOnlineStateLogic) New(ctx context.Context, svcCtx *types.ServiceContext, messageChan chan *types.SSEResponse) *DeviceOnlineStateLogic {
	return &DeviceOnlineStateLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		messageChan: messageChan,
	}
}

func (l *DeviceOnlineStateLogic) GetType() string {
	return DeviceOnlineStatesType
}

func (l *DeviceOnlineStateLogic) DO(req *SSEDeviceOnlineStatesReq) {
	defer func() {
		l.messageChan <- &types.SSEResponse{
			Done: true,
		}
	}()

	l.do(req)

	for {
		select {
		case <-l.ctx.Done():
			return

		case <-time.After(5 * time.Second):
			l.do(req)
		}
	}
}

func (l *DeviceOnlineStateLogic) do(req *SSEDeviceOnlineStatesReq) {
	if l.svcCtx.DeviceOnlineState == nil {
		l.messageChan <- &types.SSEResponse{
			Err:        response.MakeError(response.NewHttpRespMessage().Str("设备在线状态获取失败, deviceInlineState 为空"), localization.M0010),
			DelayClose: true,
		}
		return
	}

	if req.DeviceType == 1 {
		l.messageChan <- &types.SSEResponse{
			Data: l.svcCtx.DeviceOnlineState.Devices,
		}
		return
	}

	l.messageChan <- &types.SSEResponse{
		Data: l.svcCtx.DeviceOnlineState.Channels,
	}
}
