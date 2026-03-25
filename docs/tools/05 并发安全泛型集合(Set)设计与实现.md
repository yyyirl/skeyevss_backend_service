# 并发安全泛型集合（`core/pkg/set`）设计与实现

## 1. 简介

`set` 包提供 **`CSet[T comparable]`**：在 Go 泛型下实现的**线程安全集合**（元素不重复）。底层使用 **`map[T]struct{}`** 作为紧凑存储（空 struct 不占值内存），对外通过 **`sync.RWMutex`** 隔离并发读写。

VSS项目应用：流名去重、会话 key 占用标记、邀请/拉流防并发击穿、设备侧状态去重等的场景。

---

## 2. 设计要点

### 2.1 为何不导出底层 map 类型

底层 **`setMap[T]` 不导出**，仅 **`CSet[T]`** 暴露 API，避免绕过锁直接改map的误用。

### 2.2 `Range`：快照遍历与死锁规避

若在 **`RWMutex` 读锁**持有期间执行用户回调 `f`，而 `f` 内部再次调用同一 `CSet` 的 **`Add` / `Remove`（写锁）**，会典型地 **读锁未释放又等写锁**，在标准 `RWMutex` 上容易 **死锁**。

因此 **`Range` 实现为**：

1. **短临界区**：在锁内仅复制当前键列表（`snapshotKeys`）；
2. **锁外枚举**：对快照逐元素调用 `f`。

### 2.3 `Values` 与快照

`Values()` 与 `snapshotKeys` 一致：返回 **当前时刻** 所有元素的一份 **新切片**；**顺序未定义**（与 `map` 遍历顺序一致）。适合一次性打印或调试，不适合依赖稳定排序（若需要排序请调用方 `slices.Sort`。

### 2.4 容量提示 `New(hint uint)`

`New` 将 `hint` 传给 `make(map, hint)`，仅为 **减少扩容次数**；**不是**集合大小上限。高并发写热点集合时可适当增大 hint。

### 2.5 `nil` 接收者

对 `(*CSet[T])(nil)` 调用 **`Add` / `Remove` / `Clear` / `Range`** 等为 **空操作**（不 panic）；**`Contains` 为 false，`IsEmpty` 为 true，`Size` 为 0，`Values` 为 nil**，与常见「防御性」用法一致。

### 2.6 批量 `Add` / `Remove`

- **`Add(elements ...T)`**：多参数等价于依次加入，单次持锁完成，减少锁次数。
- **`Remove(elements ...T)`**：批量删除。

---

## 3. API 一览

| 方法                           | 说明                |
|------------------------------|-------------------|
| `New[T](hint uint) *CSet[T]` | 初始化构造             |
| `Add(elements ...T)`         | 并入集合              |
| `Remove(elements ...T)`      | 删除                |
| `Clear()`                    | 清空                |
| `Contains(T) bool`           | 是否包含              |
| `IsEmpty() bool`             | 是否为空              |
| `Size() int`                 | 元素个数              |
| `Range(func(T) bool)`        | 快照遍历，`false` 提前结束 |
| `Values() []T`               | 快照切片              |

---

## 4. 在项目中的典型用途（VSS）

`ServiceContext` 中示例：

- **`PubStreamExistsState`**：记录已存在推流名，避免重复占流；
- **`InviteRequestState`**：同一 `streamName` 防击穿；
- **`FetchDeviceVideoState` / `TalkSipSendStatus`**：设备或会话的进行中标记。

这类场景共同特点：**高并发读 `Contains`、短路径写 `Add`/`Remove`**，`RWMutex` 较互斥锁更友好。

---

## 5. 与其它容器选型

| 需求                | 更适合                            |
|-------------------|--------------------------------|
| 键值对、按 key 取 value | `xmap` / `sync.Map` / 业务 map+锁 |
| 只关心成员与否、去重        | **`CSet`**                     |
| 读多写少、key 为 string | 可考虑 `sync.Map`                 |

---

## 6. 并发与性能提示

- 读多写少时：**`Contains` / `Size` / `IsEmpty`** 走读锁，可并行。
- **`Range`/`Values`** 会与写操作抢占锁复制快照；快照长度大时复制成本与 **O(n)** 内存分配相关，避免在超大集合上高频调用。
- 基准测试：`go test ./core/pkg/set/... -bench=. -benchmem`

---

## 7. 总结

`core/pkg/set` 用 **小 API 面 + 不导出底层 map + 快照式 `Range`**，在保持 **并发安全** 的同时，
规避 **绕过锁** 与 **`Range` 回调重入死锁** 两类常见问题；
配合 **`New`** 与 **批量 Add/Remove**，适合在信令与媒体网关中做轻量级集合状态。
