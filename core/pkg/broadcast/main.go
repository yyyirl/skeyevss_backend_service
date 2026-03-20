// @Title        事件广播
// @Description  事件广播
// @Create       yiyiyi 2025/9/8 16:50

package broadcast

import (
	"sync"
	"time"

	"skeyevss/core/pkg/functions"
)

type BroadcastManager struct {
	channels    sync.Map     // channelName -> *channelInfo
	maxCapacity int          // 每个channel最大容量
	mu          sync.RWMutex // 用于保护需要原子操作的场景
}

type channelInfo struct {
	ch        chan interface{}
	createdAt time.Time
	lastUsed  time.Time    // 最后使用时间，用于更准确的清理
	usage     int64        // 使用计数，改为int64避免溢出
	mu        sync.RWMutex // 保护单个channel的并发操作
}

// NewBroadcast 创建新的广播管理器
func NewBroadcast(maxCapacity int) *BroadcastManager {
	if maxCapacity <= 0 {
		maxCapacity = 100 // 设置默认容量
	}

	return &BroadcastManager{
		maxCapacity: maxCapacity,
	}
}

// Send 发送数据，如果channel不存在或已满，则丢弃数据
func (cm *BroadcastManager) Send(channelName string, data interface{}) bool {
	// 检查channel是否存在
	actual, exists := cm.channels.Load(channelName)
	if !exists {
		return false
	}

	var info = actual.(*channelInfo)
	// 更新最后使用时间
	info.mu.Lock()
	info.lastUsed = time.Now()
	info.usage++
	info.mu.Unlock()

	// 尝试发送数据
	select {
	case info.ch <- data:
		return true
	default:
		// channel已满，丢弃数据
		return false
	}
}

// RegisterReceiver 注册接收者，创建channel
func (cm *BroadcastManager) RegisterReceiver(channelName string) <-chan interface{} {
	info := &channelInfo{
		ch:        make(chan interface{}, cm.maxCapacity),
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		usage:     0,
	}

	actual, loaded := cm.channels.LoadOrStore(channelName, info)
	if loaded {
		// channel已存在，返回现有的
		var existingInfo = actual.(*channelInfo)
		existingInfo.mu.Lock()
		existingInfo.lastUsed = time.Now()
		existingInfo.usage++
		existingInfo.mu.Unlock()
		return existingInfo.ch
	}

	// 新创建的channel也增加使用计数
	info.mu.Lock()
	info.usage++
	info.mu.Unlock()

	return info.ch
}

// UnregisterReceiver 取消注册，清理channel
func (cm *BroadcastManager) UnregisterReceiver(channelName string) {
	actual, exists := cm.channels.Load(channelName)
	if !exists {
		return
	}

	var info = actual.(*channelInfo)
	info.mu.Lock()
	defer info.mu.Unlock()

	// 减少使用计数
	info.usage--

	// 只有当没有接收者时才真正关闭channel
	if info.usage <= 0 {
		// 从map中删除
		cm.channels.Delete(channelName)

		// 安全关闭channel
		select {
		case _, ok := <-info.ch:
			if ok {
				close(info.ch)
				// 异步清理剩余数据
				go cm.cleanupChannel(info.ch, channelName)
			}
		default:
			close(info.ch)
		}
	}
}

// UnregisterAllReceivers 取消注册指定channel的所有接收者
func (cm *BroadcastManager) UnregisterAllReceivers(channelName string) {
	actual, exists := cm.channels.Load(channelName)
	if !exists {
		return
	}

	var info = actual.(*channelInfo)
	info.mu.Lock()
	defer info.mu.Unlock()

	// 强制关闭channel，不管有多少接收者
	cm.channels.Delete(channelName)
	select {
	case _, ok := <-info.ch:
		if ok {
			close(info.ch)
			go cm.cleanupChannel(info.ch, channelName)
		}
	default:
		close(info.ch)
	}

	// 重置使用计数
	info.usage = 0
}

// cleanupChannel 清理channel中剩余的数据
func (cm *BroadcastManager) cleanupChannel(ch <-chan interface{}, channelName string) {
	var dropped = 0
	defer functions.LogInfo("Channel Dropped", channelName, dropped)

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			dropped++
		default:
			return
		}
	}

}

// GetChannelStats 获取channel的统计信息
func (cm *BroadcastManager) GetChannelStats(channelName string) (exists bool, usage int64, queueLength int, createdAt, lastUsed time.Time) {
	actual, exists := cm.channels.Load(channelName)
	if !exists {
		return false, 0, 0, time.Time{}, time.Time{}
	}

	var info = actual.(*channelInfo)
	info.mu.RLock()
	defer info.mu.RUnlock()

	return true, info.usage, len(info.ch), info.createdAt, info.lastUsed
}

// GetAllChannels 获取所有channel名称
func (cm *BroadcastManager) GetAllChannels() []string {
	var channels []string
	cm.channels.Range(func(key, value interface{}) bool {
		channels = append(channels, key.(string))
		return true
	})

	return channels
}

// StartCleanupWorker 自动清理长时间未使用的channel
func (cm *BroadcastManager) StartCleanupWorker(interval time.Duration, maxIdle time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute // 默认间隔
	}

	if maxIdle <= 0 {
		maxIdle = 30 * time.Minute // 默认最大空闲时间
	}

	go func() {
		var ticker = time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			var now = time.Now()
			cm.channels.Range(func(key, value interface{}) bool {
				var (
					channelName = key.(string)
					info        = value.(*channelInfo)
				)

				info.mu.RLock()
				var (
					idleTime = now.Sub(info.lastUsed)
					usage    = info.usage
				)
				info.mu.RUnlock()

				// 如果channel空闲超过最大空闲时间且没有接收者，则清理
				if idleTime > maxIdle && usage == 0 {
					cm.UnregisterAllReceivers(channelName)
				}
				return true
			})
		}
	}()
}

// Close 关闭所有channel并清理资源
func (cm *BroadcastManager) Close() {
	cm.channels.Range(func(key, value interface{}) bool {
		channelName := key.(string)
		cm.UnregisterAllReceivers(channelName)
		return true
	})
}

// Broadcast 向所有channel广播消息（可选功能）
func (cm *BroadcastManager) Broadcast(data interface{}) map[string]bool {
	var result = make(map[string]bool)
	cm.channels.Range(func(key, value interface{}) bool {
		channelName := key.(string)
		success := cm.Send(channelName, data)
		result[channelName] = success
		return true
	})

	return result
}

// GetChannelCount 获取当前channel数量
func (cm *BroadcastManager) GetChannelCount() int {
	var count = 0
	cm.channels.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
