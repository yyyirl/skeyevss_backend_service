// @Title        functions
// @Description  main
// @Create       yiyiyi 2026/3/20 10:43

package dt

import (
	"context"
	"time"
)

// SetTimeout 在 duration 后执行 f 一次
// 返回的 cancel 可在到期前取消，取消时会停止底层计时器以避免泄漏。
func SetTimeout(duration time.Duration, f func()) context.CancelFunc {
	var (
		ctx, cancelFunc = context.WithCancel(context.Background())
	)

	go func() {
		var timer = time.NewTimer(duration)

		defer timer.Stop()

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}

			return

		case <-timer.C:
			f()
		}
	}()

	return cancelFunc
}

// SetInterval 每隔 interval 执行一次 f，直到调用返回的 cancel；首次在 interval 后执行（与常见 setInterval 一致，非立即首帧）。
func SetInterval(interval time.Duration, f func()) context.CancelFunc {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		var ticker = time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				f()
			}
		}
	}()

	return cancelFunc
}
