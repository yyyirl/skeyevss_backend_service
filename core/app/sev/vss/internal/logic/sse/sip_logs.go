// @Title        sip日志
// @Description  main
// @Create       yiyiyi 2025/7/23 08:55

package sse

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/response"
)

var (
	_ types.SSEHandleSPLogic[*SipLogLogic] = (*SipLogLogic)(nil)

	SipLogsType = "sip_logs"

	VSipLogs = new(SipLogLogic)

	sipLogIsActive atomic.Bool
)

// sipLogRateLimiter 限制每秒推送到 SSE 的日志条数（收发合计），避免浏览器与信令洪峰相互拖垮。
type sipLogRateLimiter struct {
	mu        sync.Mutex
	max       int
	windowEnd time.Time
	count     int
}

func newSipLogRateLimiter(maxPerSec int) *sipLogRateLimiter {
	if maxPerSec <= 0 {
		return nil
	}

	return &sipLogRateLimiter{max: maxPerSec}
}

func (r *sipLogRateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	var now = time.Now()
	if now.After(r.windowEnd) {
		r.windowEnd = now.Add(time.Second)
		r.count = 0
	}

	if r.count >= r.max {
		return false
	}

	r.count++
	return true
}

type SipLogLogic struct {
	ctx         context.Context
	svcCtx      *types.ServiceContext
	messageChan chan *types.SSEResponse
	limiter     *sipLogRateLimiter

	closeFlag atomic.Bool

	droppedFull,
	droppedRate atomic.Int64
}

func (l *SipLogLogic) New(ctx context.Context, svcCtx *types.ServiceContext, messageChan chan *types.SSEResponse) *SipLogLogic {
	return &SipLogLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		messageChan: messageChan,
	}
}

func (l *SipLogLogic) GetType() string {
	return SipLogsType
}

func (l *SipLogLogic) DO() {
	if sipLogIsActive.Load() {
		l.messageChan <- &types.SSEResponse{
			Err: response.MakeError(response.NewHttpRespMessage().Str("其他客户端正在使用"), localization.M00274),
		}
		return
	}

	sipLogIsActive.Store(true)
	l.limiter = newSipLogRateLimiter(l.svcCtx.Config.SSE.SipLogMaxPerSecond)

	go l.do()
	go func() {
		var ticker = time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var (
					df = l.droppedFull.Swap(0)
					dr = l.droppedRate.Swap(0)
					h  = map[string]interface{}{
						"type":    "",
						"content": "heartbeat",
					}
				)
				if df > 0 {
					h["dropped_full"] = df
					h["dropped_full_hint"] = "SSE 缓冲已满，已丢弃上述条数日志，可调大 SSE.MessageChanBuffer 或 SipLogMaxPerSecond"
				}

				if dr > 0 && l.svcCtx.Config.SSE.SipLogMaxPerSecond > 0 {
					h["dropped_rate"] = dr
					h["dropped_rate_hint"] = fmt.Sprintf("超过 SipLogMaxPerSecond=%d 已限速丢弃", l.svcCtx.Config.SSE.SipLogMaxPerSecond)
				}

				var msg = &types.SSEResponse{Data: h}
				select {
				case l.messageChan <- msg:
				default:
					l.droppedFull.Add(df)
					l.droppedRate.Add(dr)
				}

			case <-l.ctx.Done():
				sipLogIsActive.Store(false)
				l.closeFlag.Store(true)
				l.messageChan <- &types.SSEResponse{
					Done: true,
				}
				l.svcCtx.Broadcast.UnregisterReceiver(types.BroadcastTypeSipRequest)
				l.svcCtx.Broadcast.UnregisterReceiver(types.BroadcastTypeSipReceive)
				return
			}
		}
	}()
}

// sendLogLine 非阻塞写入 SSE；满缓冲或超限时丢弃并累计，由心跳带上统计。
func (l *SipLogLogic) sendLogLine(typ string, content string) {
	if l.closeFlag.Load() {
		return
	}

	if l.limiter != nil && !l.limiter.allow() {
		l.droppedRate.Add(1)
		return
	}

	select {
	case l.messageChan <- &types.SSEResponse{
		Data: map[string]interface{}{
			"type":    typ,
			"content": content,
		},
	}:
	default:
		l.droppedFull.Add(1)
	}
}

func (l *SipLogLogic) do() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		var receiver = l.svcCtx.Broadcast.RegisterReceiver(types.BroadcastTypeSipRequest)
		for data := range receiver {
			if l.closeFlag.Load() {
				return
			}

			l.sendLogLine(types.BroadcastTypeSipRequest, data.(string))
		}
	}()

	go func() {
		defer wg.Done()

		var receiver = l.svcCtx.Broadcast.RegisterReceiver(types.BroadcastTypeSipReceive)
		for data := range receiver {
			if l.closeFlag.Load() {
				return
			}

			l.sendLogLine(types.BroadcastTypeSipReceive, data.(string))
		}
	}()

	wg.Wait()
}
