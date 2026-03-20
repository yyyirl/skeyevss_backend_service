// @Title        固定栅格尾部节流
// @Description
//	每个 period 槽在槽结束时执行该槽内最后一次调用
//	时间轴从首次调用时刻 epoch 起, 按 [epoch+k*period, epoch+(k+1)*period) 划槽
//	同一槽内无论调用多少次, 仅在槽右边界执行「当前槽」的最后一次 call
//	例: period=500ms, 活动在 10.2s 内横跨槽 0..20（共 21 次执行）, 对应边界为 epoch+0.5s … epoch+10.5s,
//	其中 [epoch+10s, epoch+10.2s) 所属槽的右边界为 epoch+10.5s
// @Create       yiyiyi 2026/3/20 9:10

package dt

import (
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

// 按 uniqueId 保存每个键独立的栅格状态（epoch、当前槽待执行回调、定时器）
var throttleFixedGridMaps = cmap.New()

// 单键的固定栅格节流状态
type throttleFixedGridEntry struct {
	mu sync.Mutex

	// 该键首次调用时刻, 作为整时间轴原点; 之后所有槽边界均相对其按 period 对齐
	epoch time.Time
	// 栅格宽度; 同一 uniqueId 建议全程使用相同值, 否则中途修改会改变槽对齐方式而 epoch 不变
	period time.Duration

	// 当前正在累积的槽编号 k（elapsed/period）; 无待执行时为 -1
	pendingSlot int64
	// 当前槽内最新一次调用传入的回调; 在槽右边界或跨槽补跑时执行
	pendingCall func()

	// 指向当前槽右边界的一次性定时器; 为 nil 表示未安排或已触发/已停止
	timer *time.Timer
}

// stopTimerUnlocked 停止并丢弃定时器引用（须在持有 e.mu 时调用）
// 若 Stop 返回 false 表示已到期, 则尝试排空 C, 避免少量场景下与 AfterFunc 回调竞态遗留信号
func (e *throttleFixedGridEntry) stopTimerUnlocked() {
	if e.timer == nil {
		return
	}

	if !e.timer.Stop() {
		select {
		case <-e.timer.C:
		default:
		}
	}

	e.timer = nil
}

// scheduleSlotEndUnlocked 为槽 slot 安排在其右边界 epoch+(slot+1)*period 触发 onSlotTimerFire
// （须在持有 e.mu 时调用）会先停止上一计时器再注册新 AfterFunc; 若 deadline 已过则 d=0 立刻排期执行（已经过了原定槽边界）
func (e *throttleFixedGridEntry) scheduleSlotEndUnlocked(slot int64) {
	e.stopTimerUnlocked()
	var (
		deadline = e.epoch.Add(time.Duration(slot+1) * e.period)
		d        = time.Until(deadline)
	)
	if d < 0 {
		d = 0
	}

	e.timer = time.AfterFunc(d, e.onSlotTimerFire)
}

// flushPendingLocked 取出并清空当前待执行回调与槽位, 并停止定时器（须在持有 e.mu 时调用）
// 返回非 nil 时需由调用方决定同步或异步执行; 返回 nil 表示当时无待执行内容
func (e *throttleFixedGridEntry) flushPendingLocked() (fn func()) {
	if e.pendingCall == nil {
		return nil
	}

	fn = e.pendingCall
	e.pendingCall = nil
	e.pendingSlot = -1
	e.stopTimerUnlocked()
	return fn
}

// onSlotTimerFire 由 time.AfterFunc 在槽右边界调用: 在锁内取出 pending 并清空, 锁外异步执行回调
// 若期间已被跨槽同步 flush 或 stop, pendingCall 已为 nil, 则直接返回避免重复执行
func (e *throttleFixedGridEntry) onSlotTimerFire() {
	var fn func()
	e.mu.Lock()
	if e.pendingCall == nil {
		e.mu.Unlock()

		return
	}

	fn = e.flushPendingLocked()
	e.mu.Unlock()
	if fn != nil {
		go fn()
	}
}

// ThrottleFixedGridTrailing 固定时间栅格上的尾部节流（每格槽尾执行该格内最后一次回调）
//
// 参数 uniqueId: 逻辑隔离键, 不同键独立时间轴与epoch
// 参数 period: 槽长度, 例如 500*time.Millisecond; 必须>0
// 参数 call: 本键本次调用在当前槽内「覆盖」的前序 call; 仅在所属槽的右边界触发一次
//
// explain:
//   - 首次调用时记录 epoch=now, 槽 k = floor((now-epoch)/period)
//   - 同一槽内多次调用只更新 pendingCall, 定时器仍指向该槽右边界
//   - 定时器到期时在 onSlotTimerFire 中异步执行当时的 pendingCall
//   - 若新调用已进入更大槽号且仍有上一槽未执行, 则先取消定时器、同步补跑上一槽的最后一次, 再为当前槽重新排期
func ThrottleFixedGridTrailing(uniqueId string, period time.Duration, call func()) {
	if uniqueId == "" || call == nil || period <= 0 {
		return
	}

	var entry = &throttleFixedGridEntry{pendingSlot: -1}
	if !throttleFixedGridMaps.SetIfAbsent(uniqueId, entry) {
		v, ok := throttleFixedGridMaps.Get(uniqueId)
		if !ok {
			return
		}

		data, ok := v.(*throttleFixedGridEntry)
		if !ok {
			return
		}

		entry = data
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	var now = time.Now()
	if entry.epoch.IsZero() {
		entry.epoch = now
		entry.period = period
	}

	if entry.period != period {
		entry.period = period
	}

	var elapsed = now.Sub(entry.epoch)
	if elapsed < 0 {
		elapsed = 0
	}

	var slot = int64(elapsed / period)
	if entry.pendingCall != nil && slot > entry.pendingSlot {
		entry.stopTimerUnlocked()
		var fn = entry.flushPendingLocked()
		if fn != nil {
			go fn()
		}
	}

	entry.pendingSlot = slot
	entry.pendingCall = call
	entry.scheduleSlotEndUnlocked(slot)
}
