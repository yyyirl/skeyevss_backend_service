package gbs_proc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"skeyevss/core/app/sev/vss/internal/logic/http/gbs"
	"skeyevss/core/app/sev/vss/internal/logic/ws"
	"skeyevss/core/app/sev/vss/internal/pkg/common"
	"skeyevss/core/app/sev/vss/internal/pkg/ms"
	sip2 "skeyevss/core/app/sev/vss/internal/pkg/sip"
	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/constants"
	"skeyevss/core/pkg/dt"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/sdp"
)

var _ types.SipProcLogic = (*SendLogic)(nil)

type SendLogic struct {
	svcCtx      *types.ServiceContext
	recoverCall func(name string)
}

func NewSendLogic(svcCtx *types.ServiceContext, recoverCall func(name string)) *SendLogic {
	return &SendLogic{
		svcCtx:      svcCtx,
		recoverCall: recoverCall,
	}
}

func (l *SendLogic) DO(params *types.DOProcLogicParams) {
	l = &SendLogic{
		svcCtx:      params.SvcCtx,
		recoverCall: params.RecoverCall,
	}
	l.svcCtx.InitFetchDataState.Wait()

	defer l.recoverCall("发送处理")

	for {
		select {
		case v := <-l.svcCtx.SipSendBroadcast:
			go func(v *types.BroadcastReq) {
				if err := l.broadcast(v); err != nil {
					functions.LogError("send Broadcast failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendTalk:
			go func(v *types.GBSSipSendTalk) {
				if err := l.talk(v); err != nil {
					functions.LogError("send Talk failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendCatalog:
			go func(v *types.Request) {
				if err := l.catalog(v); err != nil {
					functions.LogError("send catalog failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendDeviceInfo:
			go func(v *types.Request) {
				if err := l.deviceInfo(v); err != nil {
					functions.LogError("send device info failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendVideoLiveInvite:
			go func(v *types.SipVideoLiveInviteMessage) {
				if err := l.VideoLiveInvite(v); err != nil {
					functions.LogError("send device invite video live failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendTalkInvite:
			go func(v *types.SipTalkInviteMessage) {
				if err := l.talkInvite(v); err != nil {
					functions.LogError("send device invite audio talk failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendBye:
			go func(v *types.SipByeMessage) {
				if err := l.bye(v); err != nil {
					functions.LogError("send device bye failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendDeviceControl:
			go func(v *types.DeviceControlReq) {
				if err := l.deviceControl(v); err != nil {
					functions.LogError("send device control failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendQueryPresetPoints:
			go func(v *types.SipSendQueryPresetPointsReq) {
				if err := l.queryPresets(v); err != nil {
					functions.LogError("send query preset points failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendSetPresetPoints:
			go func(v *types.SipSendSetPresetPointsReq) {
				if err := l.setPresets(v); err != nil {
					functions.LogError("send set preset failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendQueryVideoRecords:
			go func(v *types.QueryVideoRecordsReq) {
				if err := l.queryVideoRecords(v); err != nil {
					functions.LogError("send query video records failed err: ", err)
				}
			}(v)

		case v := <-l.svcCtx.SipSendSubscription:
			go func(v *types.SubscriptionReq) {
				if err := l.subscription(v); err != nil {
					functions.LogError("send subscription failed err: ", err)
				}
			}(v)
		}
	}
}

// 发送catalog请求
func (l *SendLogic) catalog(req *types.Request) error {
	dt.TrailingDebounce(
		req.ID,
		3*time.Second,
		func() {
			if _, err := sip2.NewGBSSender(l.svcCtx, req, req.ID).Catalog(); err != nil {
				functions.LogError("send catalog failed ID:", req.ID, " err: ", err)
				return
			}

			functions.LogInfo("send catalog success ID:", req.ID)
		},
	)

	return nil
}

// 广播
func (l *SendLogic) broadcast(data *types.BroadcastReq) error {
	resp, err := sip2.NewGBSSender(l.svcCtx, data.Req, data.DeviceUniqueId).Broadcast()
	if err != nil {
		return err
	}

	var code = resp.StatusCode()
	if code != http.StatusOK {
		return fmt.Errorf("语音对讲失败 状态码%d", code)
	}

	// 记录语音callID关系
	if v, ok := l.svcCtx.TalkSipData.Get(data.DeviceUniqueId); ok {
		if callId, ok := resp.CallID(); ok {
			v.CallID = callId.String()
		}
	}

	return nil
}

// 语音消息
func (l *SendLogic) talk(req *types.GBSSipSendTalk) error {
	res, ok := l.svcCtx.SipCatalogLoopMap.Get(req.DeviceUniqueId)
	if !ok {
		return errors.New("设备不在线")
	}

	var key = req.DeviceUniqueId
	talkSipData, ok := l.svcCtx.TalkSipData.Get(key)
	if !ok {
		if req.Stop {
			return errors.New("通信已关闭 [停止语音] " + req.StopCaller)
		}
		return errors.New("通信已关闭 [发送语音]")
	}

	// 结束对话
	if req.Stop {
		// 清理状态
		l.svcCtx.CloseWSTalkSip(key)
		// 停止消息
		if talkSipData.ACKReq != nil {
			_, err := sip2.NewGBSSender(l.svcCtx, res.Req, req.DeviceUniqueId).TalkBye(talkSipData.ACKReq)
			return err
		}

		return errors.New("talk bye ack req is nil")
	}

	// 发送消息
	return sip2.NewGBSSender(l.svcCtx, res.Req, req.DeviceUniqueId).SendWithRtpData(req, talkSipData)
}

// 发送device control请求
func (l *SendLogic) deviceControl(req *types.DeviceControlReq) error {
	// TODO 完整版请联系作者
	return nil
}

// 发送device请求
func (l *SendLogic) deviceInfo(req *types.Request) error {
	_, err := sip2.NewGBSSender(l.svcCtx, req, req.ID).DeviceInfo()
	return err
}

// bye
func (l *SendLogic) bye(req *types.SipByeMessage) error {
	if req.StreamName == "" {
		return errors.New("req.Req or req.StreamName is nil")
	}

	defer l.svcCtx.AckRequestMap.Remove(req.StreamName)

	ackReq, ok := l.svcCtx.AckRequestMap.Get(req.StreamName)
	if !ok {
		return nil
	}

	_, err := sip2.NewGBSSender(l.svcCtx, ackReq.Req, ackReq.ChannelUniqueId).Bye(ackReq.SendData)
	return err
}

// invite请求
func (l *SendLogic) inviteStep(stepInfo *types.StepRecord, content interface{}) {
	if stepInfo == nil {
		return
	}

	switch v := content.(type) {
	case string:
		stepInfo.Message <- &types.StepRecordMessage{Message: v}

	case error:
		stepInfo.Message <- &types.StepRecordMessage{Error: v}

	case types.StepRecordMessageSipContent:
		stepInfo.Message <- &types.StepRecordMessage{SipContent: &v}

	default:
		stepInfo.Message <- &types.StepRecordMessage{Done: true}
	}
}

// invite -> ack(from to tag callid) -> info(回放控制) -> notify(media status) -> bye
// 从ack开始 from to tag callid到后续流程这几个值需要保持一致
func (l *SendLogic) VideoLiveInvite(req *types.SipVideoLiveInviteMessage) error {
	if err := ms.New(context.Background(), l.svcCtx).RTPPub(req); err != nil {
		l.inviteStep(req.StepInfo, errors.New("拉流失败"))
		return err
	}

	l.inviteStep(req.StepInfo, "拉流成功")

	// invite
	inviteData, inviteRes, err := sip2.NewGBSSender(l.svcCtx, req.Req, req.ChannelUniqueId).VideoLiveInvite(req)
	if err != nil {
		l.inviteStep(req.StepInfo, errors.New("invite请求发送失败"))
		return err
	}

	if inviteRes.StatusCode() > http.StatusOK {
		l.inviteStep(req.StepInfo, fmt.Errorf("invite请求发送失败 code: %d", inviteRes.StatusCode()))
		return fmt.Errorf("video live invite指令发送失败, res: %s", inviteRes.String())
	}

	l.inviteStep(req.StepInfo, types.StepRecordMessageSipContent{
		Type:    "invite",
		Content: inviteData.String(),
	})
	l.inviteStep(req.StepInfo, "invite发送完成")

	// ack
	ackData, err := sip2.NewGBSSender(l.svcCtx, req.Req, req.ChannelUniqueId).AckReq(inviteRes)
	if err != nil {
		l.inviteStep(req.StepInfo, errors.New("ack请求发送失败"))
		return err
	}

	var ackCacheItem = &types.SendSipRequest{
		Req:             req.Req,
		SN:              sip2.NewGBSSender(l.svcCtx, req.Req, req.ChannelUniqueId).SN(req.ChannelUniqueId),
		SendData:        ackData,
		ChannelUniqueId: req.ChannelUniqueId,
	}
	l.svcCtx.AckRequestMap.Set(req.Req.ID, ackCacheItem)
	l.svcCtx.AckRequestMap.Set(req.StreamName, ackCacheItem)
	var stop = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		l.inviteStep(req.StepInfo, errors.New("invite 已停止"))
		// 发送停止请求并重新发起invite请求
		if _, err := functions.NewResty(ctx, &functions.RestyConfig{
			Mode: l.svcCtx.Config.Mode,
		}).HttpGet(
			fmt.Sprintf("http://127.0.0.1:%d%s", l.svcCtx.Config.Http.Port, gbs.StopStreamLogic.MakePath(req.StreamName, 0)),
			nil,
		); err != nil {
			functions.LogError("stop stream failed[1], err: ", err)
		}
	}

	// 解析invite
	sdpInfo, err := sdp.ParseString(inviteRes.Body())
	if err != nil {
		stop()
		return err
	}

	if len(sdpInfo.Media) <= 0 {
		stop()
		return errors.New("sdp media 解析错误 media为空")
	}

	if sdpInfo.Connection == nil {
		stop()
		return errors.New("sdp invite 解析错误 connect为空")
	}

	var filesize uint64
	for _, item := range sdpInfo.Media {
		for _, v := range item.Attributes {
			if v.Name == "filesize" {
				filesize, _ = strconv.ParseUint(v.Value, 10, 64)
			}
		}
	}

	// 向设备发送ack请求 开始推流
	if err := sip2.NewGBSSender(l.svcCtx, req.Req, req.ChannelUniqueId).SendDirect(ackData); err != nil {
		l.inviteStep(req.StepInfo, errors.New("ack发送失败"))
		return err
	}

	l.inviteStep(req.StepInfo, types.StepRecordMessageSipContent{
		Type:    "ack",
		Content: ackData.String(),
	})
	l.inviteStep(req.StepInfo, "ack发送成功")
	if err := ms.New(context.Background(), l.svcCtx).ACKRtpPub(req, sdpInfo.Media[0].Port, sdpInfo.Connection.Address, filesize); err != nil {
		stop()
		return err
	}

	l.inviteStep(req.StepInfo, "invite已完成")
	l.inviteStep(req.StepInfo, nil)
	l.svcCtx.PubStreamExistsState.Add(req.StreamName)

	return nil
}

// 发送talk invite 大华设备
func (l *SendLogic) talkInvite(req *types.SipTalkInviteMessage) error {
	var (
		usablePort = common.UsablePort(l.svcCtx)
		callback   = func(message string) error {
			// 停止对讲
			ws.BGBSSendTalkPubError(l.svcCtx, req.DeviceUniqueId, message)
			ws.RGBSTalkAudioStop(l.svcCtx, req.DeviceUniqueId)
			return errors.New(message)
		}
		sender = sip2.NewGBSSender(l.svcCtx, req.Req, req.DeviceUniqueId)
	)
	if usablePort <= 0 {
		return callback("可用端口获取失败")
	}

	// 发送 invite
	inviteReq, inviteRes, err := sender.TalkInvite(req, usablePort)
	if err != nil {
		return callback(fmt.Sprintf("talk invite指令发送失败, err: %s", err.Error()))
	}

	if inviteRes.StatusCode() > http.StatusOK {
		return callback(fmt.Sprintf("talk invite指令发送失败, res: %s", inviteRes.String()))
	}

	// 解析invite
	sdpInfo, err := sdp.ParseString(inviteRes.Body())
	if err != nil {
		return callback(fmt.Sprintf("talk invite sdp解析失败, err: %s", err.Error()))
	}

	// 发送 ack
	ackReq, err := sender.AckReq(inviteRes)
	if err != nil {
		return callback(fmt.Sprintf("talk ack指令创建失败, res: %s", err.Error()))
	}

	if err := sender.SendDirect(ackReq); err != nil {
		return callback(fmt.Sprintf("talk ack指令发送失败, res: %s", err.Error()))
	}

	// 逻辑与海康设备同步
	{
		// 设置rtp链接信息
		if err := common.SetTalkRtpConnInfo(l.svcCtx, sdpInfo, req.DeviceUniqueId, int(usablePort)); err != nil {
			return callback(err.Error())
		}

		// 创建rtp链接信息
		if v, ok := l.svcCtx.TalkSipData.Get(req.DeviceUniqueId); ok {
			if err := common.SetTalkRtpConn(l.svcCtx, inviteReq, req.DeviceUniqueId, v); err != nil {
				ws.BGBSSendTalkPubError(l.svcCtx, req.DeviceUniqueId, err.Error())
				return callback("rtp链接创建失败 err: " + err.Error())
			}
		} else {
			return callback("rtp链接创建失败")
		}
	}

	return nil
}

// 发送查询preset
func (l *SendLogic) queryPresets(req *types.SipSendQueryPresetPointsReq) error {
	res, ok := l.svcCtx.SipCatalogLoopMap.Get(req.DeviceUniqueId)
	if !ok {
		return constants.DeviceUnregistered
	}

	_, err := sip2.NewGBSSender(l.svcCtx, res.Req, req.ChannelUniqueId).QueryPresetPoints()
	return err
}

// 发送设置preset
func (l *SendLogic) setPresets(req *types.SipSendSetPresetPointsReq) error {
	var cmd = byte(types.SipPresetSet)
	switch req.Type {
	case "delete":
		cmd = byte(types.SipPresetDel)

	case "skip":
		cmd = byte(types.SipPresetCall)
	}

	res, ok := l.svcCtx.SipCatalogLoopMap.Get(req.DeviceUniqueId)
	if !ok {
		return constants.DeviceUnregistered
	}

	index, err := strconv.Atoi(req.Index)
	if err != nil {
		return err
	}

	var preset = sip2.Preset{
		CMD:   cmd,
		Point: byte(index),
	}
	_, err = sip2.NewGBSSender(l.svcCtx, res.Req, req.ChannelUniqueId).SetPresetPoints(preset.Pack())
	return err
}

// 获取录像
func (l *SendLogic) queryVideoRecords(req *types.QueryVideoRecordsReq) error {
	res, ok := l.svcCtx.SipCatalogLoopMap.Get(req.DeviceUniqueId)
	if !ok {
		return constants.DeviceUnregistered
	}

	_, err := sip2.NewGBSSender(l.svcCtx, res.Req, req.ChannelUniqueId).QueryVideoRecords(req.Day, req.SN)
	return err
}

// 发送订阅
func (l *SendLogic) subscription(req *types.SubscriptionReq) error {
	sipReq, ok := l.svcCtx.SipCatalogLoopMap.Get(req.DeviceUniqueId)
	if !ok {
		return constants.DeviceUnregistered
	}

	if req.Subscription.EmergencyCall {
		if _, err := sip2.NewGBSSender(l.svcCtx, sipReq.Req, req.DeviceUniqueId).SetGuard(); err != nil {
			return err
		}
	}

	{
		if req.Subscription.Catalog {
			if _, err := sip2.NewGBSSender(l.svcCtx, sipReq.Req, req.DeviceUniqueId).Subscription(types.SubscriptionCatalog); err != nil {
				functions.LogError("订阅消息发送失败, Catalog err:", err)
			}
		}

		if req.Subscription.EmergencyCall {
			if _, err := sip2.NewGBSSender(l.svcCtx, sipReq.Req, req.DeviceUniqueId).Subscription(types.SubscriptionAlarm); err != nil {
				functions.LogError("订阅消息发送失败, EmergencyCall err:", err)
			}
		}

		if req.Subscription.Location {
			if _, err := sip2.NewGBSSender(l.svcCtx, sipReq.Req, req.DeviceUniqueId).Subscription(types.SubscriptionMobilePosition); err != nil {
				functions.LogError("订阅消息发送失败, Location err:", err)
			}
		}

		if req.Subscription.PTZ {
			if _, err := sip2.NewGBSSender(l.svcCtx, sipReq.Req, req.DeviceUniqueId).Subscription(types.SubscriptionPTZPosition); err != nil {
				functions.LogError("订阅消息发送失败, PTZ err:", err)
			}
		}
	}

	return nil
}
