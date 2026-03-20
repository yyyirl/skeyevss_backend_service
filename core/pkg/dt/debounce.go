/**
 * @Author:         yi
 * @Description:    防抖
 * @Version:        1.0.0
 * @Date:           2025/2/11 17:00
 */
package dt

import (
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

var debounceMaps = cmap.New()

// Debounce 每次调用都会将执行时刻推迟到「当前时刻 + interval」；仅在持续 interval 无新调用时，由后台 ticker 触发一次 call。
func Debounce(uniqueId string, interval time.Duration, call func()) {
	if uniqueId == "" || call == nil || interval <= 0 {
		return
	}

	debounceMaps.Set(uniqueId, &debounceType{
		Call:     call,
		ExecTime: time.Now().UnixMilli() + interval.Milliseconds(),
	})
}

func debounceRunner() {
	var ticker = time.NewTicker(time.Millisecond * 10)
	for t := range ticker.C {
		for uniqueId, item := range debounceMaps.Items() {
			current, ok := item.(*debounceType)
			if !ok {
				continue
			}

			if t.UnixMilli() >= current.ExecTime {
				go current.Call()

				debounceMaps.Remove(uniqueId)
			}
		}
	}
}
