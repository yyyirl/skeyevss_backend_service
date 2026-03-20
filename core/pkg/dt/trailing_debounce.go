/**
 * @Author:         yi
 * @Description:    尾部防抖,连续触发时仅在最后一次触发后的静默期结束时执行
 * @Version:        1.0.0
 * @Date:           2025/2/11 17:11
 */

package dt

import (
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

var trailingDebounceMaps = cmap.New()

func TrailingDebounce(uniqueId string, duration time.Duration, call func()) {
	if uniqueId == "" || call == nil || duration <= 0 {
		return
	}

	// 取消任务
	if val, ok := trailingDebounceMaps.Get(uniqueId); ok {
		if item, ok := val.(*throttledType); ok && item.Cancel != nil {
			item.Cancel()
		}
	}

	var entry = &throttledType{Call: call}
	entry.Cancel = SetTimeout(duration, func() {
		current, ok := trailingDebounceMaps.Get(uniqueId)
		if !ok {
			return
		}

		item, ok := current.(*throttledType)
		if !ok || item != entry {
			// 已被新的一次 TrailingDebounce 替换，忽略过期定时器
			return
		}

		go item.Call()
		trailingDebounceMaps.Remove(uniqueId)
	})

	trailingDebounceMaps.Set(uniqueId, entry)
}
