package bmap

import (
	"bytes"
	"encoding/base64"
	"sync"
	"time"
)

type (
	Item struct {
		Data      *bytes.Buffer
		CreatedAt int64
		Bytes,
		// 转换后的数据
		ConvBytes []byte
	}

	BufferManager struct {
		items map[string]*Item
		mu    sync.RWMutex
	}
)

func NewBufferManager() *BufferManager {
	return &BufferManager{
		items: make(map[string]*Item),
	}
}

func (bm *BufferManager) Add(key, base64Data string) error {
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return err
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	item, exists := bm.items[key]
	if !exists {
		item = &Item{
			Data:      &bytes.Buffer{},
			Bytes:     nil,
			CreatedAt: time.Now().UnixMilli(),
		}
		bm.items[key] = item
	}

	// 清空缓存字节，下次获取时重新生成
	item.Bytes = nil
	_, err = item.Data.Write(data)
	return err
}

func (bm *BufferManager) Set(key string, data *Item) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.items[key] = data
}

func (bm *BufferManager) Get(key string) *Item {
	bm.mu.RLock()
	item, exists := bm.items[key]
	if !exists || item.Data.Len() == 0 {
		bm.mu.RUnlock()
		return nil
	}

	// 已经生成过 Bytes 副本，直接返回
	if item.Bytes != nil {
		bm.mu.RUnlock()
		return item
	}
	bm.mu.RUnlock()

	// 需要生成 Bytes 副本，使用写锁保证并发安全
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 可能在获取写锁期间已有其他协程生成了 Bytes，这里再检查一次
	item, exists = bm.items[key]
	if !exists || item.Data.Len() == 0 {
		return nil
	}

	if item.Bytes == nil {
		var data = item.Data.Bytes()
		item.Bytes = make([]byte, len(data))
		copy(item.Bytes, data)
	}
	return item
}

func (bm *BufferManager) GetBufferSize(key string) int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	item, exists := bm.items[key]
	if !exists {
		return 0
	}

	return item.Data.Len()
}

func (bm *BufferManager) Remove(key string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.items, key)
}

func (bm *BufferManager) Reset(key string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	item, exists := bm.items[key]
	if exists {
		item.Data.Reset()
		item.Bytes = nil
	}
}

func (bm *BufferManager) All() []string {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var keys = make([]string, 0, len(bm.items))
	for key := range bm.items {
		keys = append(keys, key)
	}
	return keys
}

func (bm *BufferManager) Len() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	return len(bm.items)
}

func (bm *BufferManager) Size() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var totalBytes int
	for _, item := range bm.items {
		totalBytes += item.Data.Len()
	}

	return totalBytes
}

func (bm *BufferManager) Range(callback func(key string, item *Item)) {
	bm.mu.RLock()
	// 先拷贝一份快照，避免在回调中再次调用 BufferManager 导致死锁
	var snapshot = make(map[string]*Item, len(bm.items))
	for key, item := range bm.items {
		snapshot[key] = item
	}
	bm.mu.RUnlock()

	for key, item := range snapshot {
		callback(key, item)
	}
}

// 清空所有过期的缓冲区（超过指定毫秒数）
func (bm *BufferManager) Cleanup(maxAgeMillis int64) int {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	var (
		now     = time.Now().UnixMilli()
		removed = 0
	)
	for key, item := range bm.items {
		if now-item.CreatedAt > maxAgeMillis {
			delete(bm.items, key)
			removed++
		}
	}
	return removed
}

func (bm *BufferManager) Exists(key string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	_, exists := bm.items[key]
	return exists
}
