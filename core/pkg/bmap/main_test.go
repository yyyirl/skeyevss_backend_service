// @Title        bmap
// @Description  缓冲区管理器测试用例
// @Create       yiyiyi

package bmap

import (
	"bytes"
	"encoding/base64"
	"sync"
	"testing"
	"time"
)

func TestAddAndGet(t *testing.T) {
	bm := NewBufferManager()

	var raw = []byte("hello world")
	var encoded = base64.StdEncoding.EncodeToString(raw)

	if err := bm.Add("k1", encoded); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}

	item := bm.Get("k1")
	if item == nil {
		t.Fatal("Get 应返回非 nil 的 Item")
	}

	if got := item.Data.Len(); got != len(raw) {
		t.Fatalf("缓冲区长度不匹配，期望 %d，实际 %d", len(raw), got)
	}

	// Bytes 懒加载副本应与原始数据一致
	if item.Bytes == nil {
		t.Fatal("懒加载后 Bytes 应已初始化")
	}
	if !bytes.Equal(item.Bytes, raw) {
		t.Fatalf("Bytes 内容不匹配，期望 %q，实际 %q", string(raw), string(item.Bytes))
	}

	// 再次 Get 不应重新分配 Bytes（地址保持不变）
	item2 := bm.Get("k1")
	if item2 == nil {
		t.Fatal("第二次 Get 应返回非 nil 的 Item")
	}
	if &item2.Bytes[0] != &item.Bytes[0] {
		t.Fatal("两次 Get 应复用同一 Bytes 切片，不应重新分配")
	}
}

func TestSetAndGet(t *testing.T) {
	bm := NewBufferManager()

	var buf = bytes.NewBufferString("data")
	var item = &Item{
		Data:      buf,
		CreatedAt: time.Now().UnixMilli(),
	}
	bm.Set("k2", item)

	got := bm.Get("k2")
	if got == nil {
		t.Fatal("Get 应返回非 nil 的 Item")
	}
	if got.Data.String() != "data" {
		t.Fatalf("Data 内容不匹配，期望 %q，实际 %q", "data", got.Data.String())
	}
}

func TestReset(t *testing.T) {
	bm := NewBufferManager()

	var raw = []byte("hello")
	var encoded = base64.StdEncoding.EncodeToString(raw)
	if err := bm.Add("k", encoded); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if bm.GetBufferSize("k") == 0 {
		t.Fatal("Add 后该 key 的缓冲区大小应大于 0")
	}

	bm.Reset("k")
	if bm.GetBufferSize("k") != 0 {
		t.Fatalf("Reset 后缓冲区大小应为 0，实际 %d", bm.GetBufferSize("k"))
	}

	// Reset 不会删除 key
	if !bm.Exists("k") {
		t.Fatal("Reset 后 key 仍应存在")
	}
}

func TestRemove(t *testing.T) {
	bm := NewBufferManager()

	bm.Set("k", &Item{Data: &bytes.Buffer{}})
	if !bm.Exists("k") {
		t.Fatal("Set 后 key 应存在")
	}

	bm.Remove("k")
	if bm.Exists("k") {
		t.Fatal("Remove 后 key 应被删除")
	}
}

func TestAllLenSize(t *testing.T) {
	bm := NewBufferManager()

	bm.Set("a", &Item{Data: bytes.NewBufferString("123")})
	bm.Set("b", &Item{Data: bytes.NewBufferString("4567")})

	if bm.Len() != 2 {
		t.Fatalf("条目数不匹配，期望 2，实际 %d", bm.Len())
	}

	var keys = bm.All()
	if len(keys) != 2 {
		t.Fatalf("All() 返回的 key 数量不匹配，期望 2，实际 %d", len(keys))
	}

	if bm.Size() != 3+4 {
		t.Fatalf("总字节数不匹配，期望 %d，实际 %d", 3+4, bm.Size())
	}
}

func TestCleanup(t *testing.T) {
	bm := NewBufferManager()

	var oldItem = &Item{
		Data:      bytes.NewBufferString("old"),
		CreatedAt: time.Now().Add(-2 * time.Second).UnixMilli(),
	}
	var newItem = &Item{
		Data:      bytes.NewBufferString("new"),
		CreatedAt: time.Now().UnixMilli(),
	}

	bm.Set("old", oldItem)
	bm.Set("new", newItem)

	// 500ms 过期，只应清理 old
	var removed = bm.Cleanup(500)
	if removed != 1 {
		t.Fatalf("Cleanup 应删除 1 个过期项，实际删除 %d", removed)
	}

	if bm.Exists("old") {
		t.Fatal("过期项 old 应被 Cleanup 删除")
	}
	if !bm.Exists("new") {
		t.Fatal("未过期项 new 应保留")
	}
}

func TestRangeSnapshotAndConcurrencySafety(t *testing.T) {
	bm := NewBufferManager()

	bm.Set("k1", &Item{Data: bytes.NewBufferString("1")})
	bm.Set("k2", &Item{Data: bytes.NewBufferString("2")})

	var (
		mu    sync.Mutex
		count int
	)

	// 在 Range 回调中再次操作 BufferManager，验证不会死锁
	bm.Range(func(key string, item *Item) {
		mu.Lock()
		defer mu.Unlock()
		count++

		_ = bm.Exists(key)
		_ = bm.GetBufferSize(key)
	})

	if count != 2 {
		t.Fatalf("Range 应遍历 2 个条目，实际 %d", count)
	}
}
