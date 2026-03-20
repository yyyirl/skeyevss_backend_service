/**
 * @Author:         yi
 * @Description:    前缘节流 固定时间窗口内至多执行一次
 * @Version:        1.0.0
 * @Date:           2026/3/19
 */
package dt

import (
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

var throttleMaps = cmap.New()

type throttleEntry struct {
	mu       sync.Mutex
	lastExec time.Time
}

// Throttle 前缘节流
// 对同一 uniqueId，在任意连续 duration 内仅**第一次**调用会执行 call
// 窗口内其余调用直接丢弃
// 超过 duration 无调用后，下一次调用再次视为「窗口内首次」可执行。
func Throttle(uniqueId string, duration time.Duration, call func()) {
	if uniqueId == "" || call == nil || duration <= 0 {
		return
	}

	var entry = &throttleEntry{}
	if !throttleMaps.SetIfAbsent(uniqueId, entry) {
		v, ok := throttleMaps.Get(uniqueId)
		if !ok {
			return
		}

		data, ok := v.(*throttleEntry)
		if !ok {
			return
		}

		entry = data
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	var now = time.Now()
	if !entry.lastExec.IsZero() && now.Sub(entry.lastExec) < duration {
		return
	}

	entry.lastExec = now
	go call()
}
